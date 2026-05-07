package notification

import (
	"context"

	"datn-backend/internal/domain"
)

type Sender interface {
	SendAlert(ctx context.Context, alert domain.Alert) error
}
