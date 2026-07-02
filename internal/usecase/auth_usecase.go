package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
	"datn-backend/internal/token"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrPhoneAlreadyExists  = errors.New("phone number already exists")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidOTP          = errors.New("invalid otp")
	ErrOTPExpired          = errors.New("otp expired")
	ErrOTPUsed             = errors.New("otp already used")
	ErrOTPTooManyAttempts  = errors.New("otp too many attempts")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
)

const (
	registerOTPTTL         = 5 * time.Minute
	registerOTPMaxAttempts = 5
)

var phoneNumberPattern = regexp.MustCompile(`^\+?[0-9]{9,15}$`)

type AuthUseCase struct {
	users           repo.UserRepository
	otps            repo.AuthOTPRepository
	refreshTokens   repo.RefreshTokenRepository
	registrations   repo.RegistrationRepository
	tokens          *token.Service
	refreshTokenTTL time.Duration
}

type AuthOptions struct {
	TokenService    *token.Service
	RefreshTokenTTL time.Duration
}

func NewAuthUseCase(users repo.UserRepository, otps repo.AuthOTPRepository, refreshTokens repo.RefreshTokenRepository, registrations repo.RegistrationRepository, options AuthOptions) *AuthUseCase {
	refreshTokenTTL := options.RefreshTokenTTL
	if refreshTokenTTL == 0 {
		refreshTokenTTL = 30 * 24 * time.Hour
	}

	return &AuthUseCase{
		users:           users,
		otps:            otps,
		refreshTokens:   refreshTokens,
		registrations:   registrations,
		tokens:          options.TokenService,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (uc *AuthUseCase) RequestRegisterOTP(ctx context.Context, input domain.RequestRegisterOTPInput) (*domain.RegisterOTPChallenge, error) {
	phoneNumber, err := normalizePhoneNumber(input.PhoneNumber)
	if err != nil {
		return nil, err
	}
	existing, err := uc.users.FindByPhoneNumber(ctx, phoneNumber)
	if err != nil && !errors.Is(err, repo.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrPhoneAlreadyExists
	}

	otpCode, err := generateOTPCode()
	if err != nil {
		return nil, err
	}

	otpHash, err := bcrypt.GenerateFromPassword([]byte(otpCode), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(registerOTPTTL)
	otp, err := uc.otps.Create(ctx, domain.AuthOTP{
		PhoneNumber:  phoneNumber,
		Purpose:      domain.OTPPurposeRegister,
		OTPHash:      string(otpHash),
		ExpiresAt:    expiresAt,
		AttemptCount: 0,
		MaxAttempts:  registerOTPMaxAttempts,
	})
	if err != nil {
		return nil, err
	}

	output := &domain.RegisterOTPChallenge{
		ExpiresAt: otp.ExpiresAt,
		DevOTP:    &otpCode,
	}

	return output, nil
}

func (uc *AuthUseCase) VerifyRegisterOTP(ctx context.Context, input domain.VerifyRegisterOTPInput) (*domain.RegisterOTPVerification, error) {
	phoneNumber, err := normalizePhoneNumber(input.PhoneNumber)
	if err != nil {
		return nil, err
	}

	otpCode := strings.TrimSpace(input.OTP)

	if !isValidOTPCode(otpCode) {
		return nil, ErrInvalidInput
	}

	existing, err := uc.users.FindByPhoneNumber(ctx, phoneNumber)
	if err != nil && !errors.Is(err, repo.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrPhoneAlreadyExists
	}

	otp, err := uc.otps.FindLatest(ctx, phoneNumber, domain.OTPPurposeRegister)
	if err != nil {
		return nil, err
	}
	if otp.IsUsed() {
		return nil, ErrOTPUsed
	}
	if time.Now().UTC().After(otp.ExpiresAt) {
		return nil, ErrOTPExpired
	}
	if otp.AttemptCount >= otp.MaxAttempts {
		return nil, ErrOTPTooManyAttempts
	}
	if err := bcrypt.CompareHashAndPassword([]byte(otp.OTPHash), []byte(otpCode)); err != nil {
		if incrementErr := uc.otps.IncrementAttempt(ctx, otp.ID); incrementErr != nil {
			return nil, incrementErr
		}
		return nil, ErrInvalidOTP
	}

	registerToken, err := uc.tokens.IssueRegisterToken(phoneNumber)
	if err != nil {
		return nil, err
	}

	if err := uc.otps.MarkUsed(ctx, otp.ID); err != nil {
		return nil, err
	}

	return &domain.RegisterOTPVerification{
		AccessToken: registerToken.Token,
		TokenType:   "Bearer",
		ExpiresAt:   registerToken.ExpiresAt,
		ExpiresIn:   int64(time.Until(registerToken.ExpiresAt).Seconds()),
	}, nil
}

func (uc *AuthUseCase) CompleteRegister(ctx context.Context, phoneNumber string, input domain.CompleteRegisterInput) (*domain.AuthSession, error) {
	phoneNumber, err := normalizePhoneNumber(phoneNumber)
	if err != nil {
		return nil, err
	}

	fullName := strings.TrimSpace(input.FullName)
	if fullName == "" || len(input.Password) < 8 {
		return nil, ErrInvalidInput
	}

	existing, err := uc.users.FindByPhoneNumber(ctx, phoneNumber)
	if err != nil && !errors.Is(err, repo.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrPhoneAlreadyExists
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := uc.registrations.CompleteRegister(ctx, domain.User{
		PhoneNumber:  phoneNumber,
		FullName:     fullName,
		PasswordHash: string(passwordHash),
	}, domain.RefreshToken{
		TokenHash: token.HashRefreshToken(refreshToken),
		ExpiresAt: time.Now().UTC().Add(uc.refreshTokenTTL),
	})
	if errors.Is(err, repo.ErrUserAlreadyExists) {
		return nil, ErrPhoneAlreadyExists
	}
	if err != nil {
		return nil, err
	}

	accessToken, err := uc.tokens.IssueAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &domain.AuthSession{
		User: user,
		Tokens: domain.AuthTokens{
			AccessToken:  accessToken.Token,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresAt:    accessToken.ExpiresAt,
			ExpiresIn:    int64(time.Until(accessToken.ExpiresAt).Seconds()),
		},
	}, nil
}

func (uc *AuthUseCase) Login(ctx context.Context, input domain.LoginInput) (*domain.AuthSession, error) {
	phoneNumber, err := normalizePhoneNumber(input.PhoneNumber)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.Password) == "" {
		return nil, ErrInvalidInput
	}

	user, err := uc.users.FindByPhoneNumber(ctx, phoneNumber)
	if errors.Is(err, repo.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return uc.issueSession(ctx, user)
}

func (uc *AuthUseCase) Refresh(ctx context.Context, input domain.RefreshInput) (*domain.AuthTokens, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return nil, ErrInvalidInput
	}

	tokenHash := token.HashRefreshToken(refreshToken)
	stored, err := uc.refreshTokens.FindByHash(ctx, tokenHash)
	if errors.Is(err, repo.ErrRefreshTokenNotFound) {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}
	if stored.RevokedAt != nil {
		return nil, ErrRefreshTokenRevoked
	}
	if time.Now().UTC().After(stored.ExpiresAt) {
		return nil, ErrRefreshTokenExpired
	}
	if _, err := uc.users.FindByID(ctx, stored.UserID); err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	accessToken, err := uc.tokens.IssueAccessToken(stored.UserID)
	if err != nil {
		return nil, err
	}

	nextRefreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	_, err = uc.refreshTokens.Replace(ctx, tokenHash, domain.RefreshToken{
		UserID:    stored.UserID,
		TokenHash: token.HashRefreshToken(nextRefreshToken),
		ExpiresAt: time.Now().UTC().Add(uc.refreshTokenTTL),
	})
	if errors.Is(err, repo.ErrRefreshTokenNotFound) {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}

	return &domain.AuthTokens{
		AccessToken:  accessToken.Token,
		RefreshToken: nextRefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    accessToken.ExpiresAt,
		ExpiresIn:    int64(time.Until(accessToken.ExpiresAt).Seconds()),
	}, nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, input domain.LogoutInput) error {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return ErrInvalidInput
	}

	err := uc.refreshTokens.RevokeByHash(ctx, token.HashRefreshToken(refreshToken))
	if errors.Is(err, repo.ErrRefreshTokenNotFound) {
		return ErrInvalidRefreshToken
	}

	return err
}

func (uc *AuthUseCase) LogoutAll(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalidInput
	}

	return uc.refreshTokens.RevokeAllByUserID(ctx, userID)
}

func (uc *AuthUseCase) LinkEmail(ctx context.Context, userID string, email string) (*domain.User, error) {
	userID = strings.TrimSpace(userID)
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}

	if userID == "" || email == "" {
		return nil, ErrInvalidInput
	}

	existing, err := uc.users.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, repo.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil && existing.ID != userID {
		return nil, ErrEmailAlreadyExists
	}

	return uc.users.UpdateEmail(ctx, userID, email)
}

func (uc *AuthUseCase) issueSession(ctx context.Context, user *domain.User) (*domain.AuthSession, error) {
	accessToken, err := uc.tokens.IssueAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := token.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	if _, err := uc.refreshTokens.Create(ctx, domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: token.HashRefreshToken(refreshToken),
		ExpiresAt: time.Now().UTC().Add(uc.refreshTokenTTL),
	}); err != nil {
		return nil, err
	}

	return &domain.AuthSession{
		User: user,
		Tokens: domain.AuthTokens{
			AccessToken:  accessToken.Token,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresAt:    accessToken.ExpiresAt,
			ExpiresIn:    int64(time.Until(accessToken.ExpiresAt).Seconds()),
		},
	}, nil
}

func normalizePhoneNumber(phoneNumber string) (string, error) {
	phoneNumber = strings.TrimSpace(phoneNumber)
	phoneNumber = strings.NewReplacer(" ", "", "-", "", ".", "", "(", "", ")", "").Replace(phoneNumber)

	if !phoneNumberPattern.MatchString(phoneNumber) {
		return "", ErrInvalidInput
	}

	return phoneNumber, nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", ErrInvalidInput
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address == "" {
		return "", ErrInvalidInput
	}

	return parsed.Address, nil
}

func generateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func isValidOTPCode(otp string) bool {
	if len(otp) != 6 {
		return false
	}

	for _, ch := range otp {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return true
}
