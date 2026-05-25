package token

import (
	"errors"
	"testing"
	"time"
)

func TestServiceIssueAndVerifyAccessToken(t *testing.T) {
	service := NewService("test-secret", "datn-backend", "datn-api", 15*time.Minute)

	accessToken, err := service.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	claims, err := service.VerifyAccessToken(accessToken.Token)
	if err != nil {
		t.Fatalf("VerifyAccessToken returned error: %v", err)
	}

	if claims.Subject != "user-123" {
		t.Fatalf("expected subject user-123, got %q", claims.Subject)
	}
	if claims.TokenType != "access" {
		t.Fatalf("expected token type access, got %q", claims.TokenType)
	}
}

func TestServiceIssueAndVerifyRegisterToken(t *testing.T) {
	service := NewService("test-secret", "datn-backend", "datn-api", 15*time.Minute)

	registerToken, err := service.IssueRegisterToken("0962143076")
	if err != nil {
		t.Fatalf("IssueRegisterToken returned error: %v", err)
	}

	claims, err := service.VerifyRegisterToken(registerToken.Token)
	if err != nil {
		t.Fatalf("VerifyRegisterToken returned error: %v", err)
	}

	if claims.Subject != "0962143076" {
		t.Fatalf("expected subject 0962143076, got %q", claims.Subject)
	}
	if claims.TokenType != TokenTypeRegister {
		t.Fatalf("expected token type register, got %q", claims.TokenType)
	}
}

func TestServiceVerifyAccessTokenRejectsRegisterToken(t *testing.T) {
	service := NewService("test-secret", "datn-backend", "datn-api", 15*time.Minute)

	registerToken, err := service.IssueRegisterToken("0962143076")
	if err != nil {
		t.Fatalf("IssueRegisterToken returned error: %v", err)
	}

	if _, err := service.VerifyAccessToken(registerToken.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestServiceVerifyAccessTokenRejectsWrongAudience(t *testing.T) {
	issuer := NewService("test-secret", "datn-backend", "datn-api", 15*time.Minute)
	verifier := NewService("test-secret", "datn-backend", "other-api", 15*time.Minute)

	accessToken, err := issuer.IssueAccessToken("user-123")
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	if _, err := verifier.VerifyAccessToken(accessToken.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestHashRefreshTokenIsStable(t *testing.T) {
	first := HashRefreshToken("refresh-token")
	second := HashRefreshToken("refresh-token")

	if first == "" {
		t.Fatal("expected non-empty hash")
	}
	if first != second {
		t.Fatal("expected stable refresh token hash")
	}
	if first == "refresh-token" {
		t.Fatal("refresh token hash must not equal raw token")
	}
}
