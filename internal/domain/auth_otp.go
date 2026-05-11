package domain

import "time"

const OTPPurposeRegister = "register"

type AuthOTP struct {
	ID           string
	PhoneNumber  string
	Purpose      string
	OTPHash      string
	ExpiresAt    time.Time
	AttemptCount int
	MaxAttempts  int
	UsedAt       *time.Time
	CreatedAt    time.Time
}

func (otp AuthOTP) IsUsed() bool {
	return otp.UsedAt != nil
}
