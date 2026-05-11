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

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrPhoneAlreadyExists = errors.New("phone number already exists")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidOTP         = errors.New("invalid otp")
	ErrOTPExpired         = errors.New("otp expired")
	ErrOTPUsed            = errors.New("otp already used")
	ErrOTPTooManyAttempts = errors.New("otp too many attempts")
)

const (
	registerOTPTTL         = 5 * time.Minute
	registerOTPMaxAttempts = 5
)

var phoneNumberPattern = regexp.MustCompile(`^\+?[0-9]{9,15}$`)

type AuthUseCase struct {
	users        repo.UserRepository
	otps         repo.AuthOTPRepository
	exposeDevOTP bool
}

type RequestRegisterOTPInput struct {
	PhoneNumber string
}

type RequestRegisterOTPOutput struct {
	ExpiresAt time.Time
	DevOTP    *string
}

type VerifyRegisterInput struct {
	PhoneNumber string
	OTP         string
	FullName    string
	Password    string
}

type AuthOptions struct {
	ExposeDevOTP bool
}

func NewAuthUseCase(users repo.UserRepository, otps repo.AuthOTPRepository, options AuthOptions) *AuthUseCase {
	return &AuthUseCase{
		users:        users,
		otps:         otps,
		exposeDevOTP: options.ExposeDevOTP,
	}
}

func (uc *AuthUseCase) RequestRegisterOTP(ctx context.Context, input RequestRegisterOTPInput) (*RequestRegisterOTPOutput, error) {
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

	output := &RequestRegisterOTPOutput{
		ExpiresAt: otp.ExpiresAt,
	}
	if uc.exposeDevOTP {
		output.DevOTP = &otpCode
	}

	return output, nil
}

func (uc *AuthUseCase) VerifyRegister(ctx context.Context, input VerifyRegisterInput) (*domain.User, error) {
	phoneNumber, err := normalizePhoneNumber(input.PhoneNumber)
	if err != nil {
		return nil, err
	}

	otpCode := strings.TrimSpace(input.OTP)
	fullName := strings.TrimSpace(input.FullName)

	if !isValidOTPCode(otpCode) || fullName == "" || len(input.Password) < 8 {
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

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := uc.users.Create(ctx, domain.User{
		PhoneNumber:  phoneNumber,
		FullName:     fullName,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		return nil, err
	}

	if err := uc.otps.MarkUsed(ctx, otp.ID); err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *AuthUseCase) LinkEmail(ctx context.Context, userID string, email string) (*domain.User, error) {
	userID = strings.TrimSpace(userID)
	email = strings.TrimSpace(strings.ToLower(email))

	if userID == "" || email == "" {
		return nil, ErrInvalidInput
	}
	if _, err := mail.ParseAddress(email); err != nil {
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

func normalizePhoneNumber(phoneNumber string) (string, error) {
	phoneNumber = strings.TrimSpace(phoneNumber)
	phoneNumber = strings.NewReplacer(" ", "", "-", "", ".", "", "(", "", ")").Replace(phoneNumber)

	if !phoneNumberPattern.MatchString(phoneNumber) {
		return "", ErrInvalidInput
	}

	return phoneNumber, nil
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
