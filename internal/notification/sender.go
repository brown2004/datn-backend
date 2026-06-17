package notification

import (
	"context"

	"datn-backend/internal/domain"
)

type Sender interface {
	SendAlert(ctx context.Context, alert domain.Alert, devices []domain.MobileDevice) (SendReport, error)
}

type SendReport struct {
	Targeted int
	Sent     int
	Failed   []SendFailure
}

type SendFailure struct {
	Token        string
	Err          error
	InvalidToken bool
}
