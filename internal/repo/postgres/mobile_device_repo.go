package postgres

import (
	"context"
	"database/sql"
	"errors"

	"datn-backend/internal/domain"
)

type MobileDeviceRepository struct {
	db *sql.DB
}

func NewMobileDeviceRepository(db *sql.DB) *MobileDeviceRepository {
	return &MobileDeviceRepository{db: db}
}

func (r *MobileDeviceRepository) FindByUserID(ctx context.Context, userID string) ([]domain.MobileDevice, error) {
	const query = `
		SELECT id, user_id, fcm_token, platform
		FROM mobile_devices
		WHERE user_id = $1
		ORDER BY platform ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []domain.MobileDevice
	for rows.Next() {
		device, err := scanMobileDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *device)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func (r *MobileDeviceRepository) Upsert(ctx context.Context, device domain.MobileDevice) (*domain.MobileDevice, error) {
	const query = `
		INSERT INTO mobile_devices (user_id, fcm_token, platform)
		VALUES ($1, $2, $3)
		ON CONFLICT (fcm_token)
		DO UPDATE SET user_id = EXCLUDED.user_id,
					  platform = EXCLUDED.platform
		RETURNING id, user_id, fcm_token, platform
	`

	return scanMobileDevice(r.db.QueryRowContext(ctx, query, device.UserID, device.FCMToken, device.Platform))
}

func (r *MobileDeviceRepository) Save(ctx context.Context, device *domain.MobileDevice) error {
	_, err := r.Upsert(ctx, *device)
	return err
}

type mobileDeviceRow interface {
	Scan(dest ...any) error
}

func scanMobileDevice(row mobileDeviceRow) (*domain.MobileDevice, error) {
	var device domain.MobileDevice

	if err := row.Scan(
		&device.ID,
		&device.UserID,
		&device.FCMToken,
		&device.Platform,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &device, nil
}
