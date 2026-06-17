package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"datn-backend/internal/domain"
)

type AlertRepository struct {
	db *sql.DB
}

func NewAlertRepository(db *sql.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) Save(ctx context.Context, alert *domain.Alert) error {
	if alert.CreatedAt.IsZero() {
		const query = `
			INSERT INTO alerts (pc_agent_id, alert_type)
			VALUES ($1, $2)
			RETURNING id, triggered_at
		`
		return r.db.QueryRowContext(ctx, query, alert.AgentID, alert.Type).Scan(&alert.ID, &alert.CreatedAt)
	}

	const query = `
		INSERT INTO alerts (pc_agent_id, alert_type, triggered_at)
		VALUES ($1, $2, $3)
		RETURNING id, triggered_at
	`
	return r.db.QueryRowContext(ctx, query, alert.AgentID, alert.Type, alert.CreatedAt).Scan(&alert.ID, &alert.CreatedAt)
}

func (r *AlertRepository) FindByUserID(ctx context.Context, userID string) ([]domain.Alert, error) {
	const query = `
		SELECT alerts.id, alerts.pc_agent_id, pc_agents.device_name, pc_agents.user_id, alerts.alert_type, alerts.triggered_at
		FROM alerts
		INNER JOIN pc_agents ON pc_agents.id = alerts.pc_agent_id
		WHERE pc_agents.user_id = $1
		ORDER BY alerts.triggered_at DESC
		LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []domain.Alert
	for rows.Next() {
		alert, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, *alert)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return alerts, nil
}

func (r *AlertRepository) FindRecentByAgentAndType(ctx context.Context, agentID string, alertType string, since time.Time) (*domain.Alert, error) {
	const query = `
		SELECT alerts.id, alerts.pc_agent_id, pc_agents.device_name, pc_agents.user_id, alerts.alert_type, alerts.triggered_at
		FROM alerts
		INNER JOIN pc_agents ON pc_agents.id = alerts.pc_agent_id
		WHERE alerts.pc_agent_id = $1
		  AND alerts.alert_type = $2
		  AND alerts.triggered_at >= $3
		ORDER BY alerts.triggered_at DESC
		LIMIT 1
	`

	return scanAlert(r.db.QueryRowContext(ctx, query, agentID, alertType, since))
}

type alertRow interface {
	Scan(dest ...any) error
}

func scanAlert(row alertRow) (*domain.Alert, error) {
	var alert domain.Alert
	if err := row.Scan(
		&alert.ID,
		&alert.AgentID,
		&alert.AgentName,
		&alert.UserID,
		&alert.Type,
		&alert.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	alert.Message = alertMessage(alert.Type)
	return &alert, nil
}

func alertMessage(alertType string) string {
	switch alertType {
	case domain.AlertTypeMotionDetected:
		return "Phát hiện rung lắc hoặc di chuyển bất thường."
	case domain.AlertTypeUSBRemoved:
		return "Thiết bị bảo vệ đã bị ngắt kết nối."
	default:
		return "Phát hiện cảnh báo mới từ thiết bị."
	}
}
