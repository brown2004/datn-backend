package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const upsertQuery = `
		INSERT INTO mobile_devices (user_id, fcm_token, platform)
		VALUES ($1, $2, $3)
		ON CONFLICT (fcm_token)
		DO UPDATE SET user_id = EXCLUDED.user_id,
					  platform = EXCLUDED.platform
		RETURNING id, user_id, fcm_token, platform
	`

	savedDevice, err := scanMobileDevice(tx.QueryRowContext(ctx, upsertQuery, device.UserID, device.FCMToken, device.Platform))
	if err != nil {
		return nil, err
	}
	if savedDevice == nil {
		return nil, fmt.Errorf("mobile device upsert returned no row")
	}

	const cleanupQuery = `
		DELETE FROM mobile_devices
		WHERE user_id = $1
		  AND platform = $2
		  AND fcm_token <> $3
	`
	if _, err := tx.ExecContext(ctx, cleanupQuery, savedDevice.UserID, savedDevice.Platform, savedDevice.FCMToken); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return savedDevice, nil
}

func (r *MobileDeviceRepository) DeleteByFCMToken(ctx context.Context, fcmToken string) error {
	const query = `
		DELETE FROM mobile_devices
		WHERE fcm_token = $1
	`

	_, err := r.db.ExecContext(ctx, query, fcmToken)
	return err
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
