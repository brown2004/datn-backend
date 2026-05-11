package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type Service struct {
	secret    []byte
	issuer    string
	audience  string
	accessTTL time.Duration
}

type AccessToken struct {
	Token     string
	ExpiresAt time.Time
}

type AccessClaims struct {
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

func NewService(secret string, issuer string, audience string, accessTTL time.Duration) *Service {
	return &Service{
		secret:    []byte(secret),
		issuer:    issuer,
		audience:  audience,
		accessTTL: accessTTL,
	}
}

func (s *Service) IssueAccessToken(userID string) (*AccessToken, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.accessTTL)

	claims := AccessClaims{
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return nil, err
	}

	return &AccessToken{
		Token:     signed,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) VerifyAccessToken(rawToken string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	parser := jwt.NewParser(
		jwt.WithAudience(s.audience),
		jwt.WithIssuer(s.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	parsed, err := parser.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil || !parsed.Valid || claims.TokenType != "access" || claims.Subject == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashRefreshToken(refreshToken string) string {
	hash := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(hash[:])
}
