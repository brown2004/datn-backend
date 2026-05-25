package domain

import "time"

type RequestRegisterOTPInput struct {
	PhoneNumber string
}

type VerifyRegisterOTPInput struct {
	PhoneNumber string
	OTP         string
}

type CompleteRegisterInput struct {
	FullName string
	Password string
}

type LoginInput struct {
	PhoneNumber string
	Password    string
}

type RefreshInput struct {
	RefreshToken string
}

type LogoutInput struct {
	RefreshToken string
}

type AuthSession struct {
	User   *User
	Tokens AuthTokens
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    time.Time
	ExpiresIn    int64
}

type RegisterOTPChallenge struct {
	ExpiresAt time.Time
	DevOTP    *string
}

type RegisterOTPVerification struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
	ExpiresIn   int64
}
