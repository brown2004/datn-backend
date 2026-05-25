package usecase

import (
	"errors"
	"testing"
)

func TestNormalizePhoneNumber(t *testing.T) {
	phoneNumber, err := normalizePhoneNumber(" 096-214.3076 ")
	if err != nil {
		t.Fatalf("normalizePhoneNumber returned error: %v", err)
	}

	if phoneNumber != "0962143076" {
		t.Fatalf("expected normalized phone number 0962143076, got %q", phoneNumber)
	}
}

func TestNormalizePhoneNumberRejectsInvalidInput(t *testing.T) {
	if _, err := normalizePhoneNumber("abc"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestNormalizeEmail(t *testing.T) {
	email, err := normalizeEmail(" User@Example.COM ")
	if err != nil {
		t.Fatalf("normalizeEmail returned error: %v", err)
	}

	if email != "user@example.com" {
		t.Fatalf("expected user@example.com, got %q", email)
	}
}

func TestNormalizeEmailStoresAddressOnly(t *testing.T) {
	email, err := normalizeEmail("User <User@Example.COM>")
	if err != nil {
		t.Fatalf("normalizeEmail returned error: %v", err)
	}

	if email != "user@example.com" {
		t.Fatalf("expected user@example.com, got %q", email)
	}
}

func TestNormalizeEmailRejectsInvalidInput(t *testing.T) {
	if _, err := normalizeEmail("not-an-email"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
