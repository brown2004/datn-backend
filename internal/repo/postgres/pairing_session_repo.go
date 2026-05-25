package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
)

type PairingSessionRepository struct {
	db *sql.DB
}

func NewPairingSessionRepository(db *sql.DB) *PairingSessionRepository {
	return &PairingSessionRepository{db: db}
}

func (r *PairingSessionRepository) Create(ctx context.Context, session domain.PairingSession) (*domain.PairingSession, error) {
	const query = `
		INSERT INTO pairing_sessions (device_code, requested_pc_agent_id, device_name, os_type, status, expires_at)
		VALUES ($1, gen_random_uuid(), $2, $3, $4, $5)
		RETURNING id, device_code, requested_pc_agent_id, device_name, os_type, status, expires_at,
			confirmed_by_user_id, pc_agent_id, confirmed_at, created_at
	`

	created, err := scanPairingSession(r.db.QueryRowContext(
		ctx,
		query,
		session.DeviceCode,
		session.DeviceName,
		session.OSType,
		session.Status,
		session.ExpiresAt,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, repo.ErrPairingSessionCodeExists
		}
		return nil, err
	}

	return created, nil
}

func (r *PairingSessionRepository) FindByDeviceCode(ctx context.Context, deviceCode string) (*domain.PairingSession, error) {
	const query = `
		SELECT id, device_code, requested_pc_agent_id, device_name, os_type, status, expires_at,
			confirmed_by_user_id, pc_agent_id, confirmed_at, created_at
		FROM pairing_sessions
		WHERE device_code = $1
	`

	return scanPairingSession(r.db.QueryRowContext(ctx, query, deviceCode))
}

func (r *PairingSessionRepository) FindByIDAndDeviceCode(ctx context.Context, id string, deviceCode string) (*domain.PairingSession, error) {
	const query = `
		SELECT id, device_code, requested_pc_agent_id, device_name, os_type, status, expires_at,
			confirmed_by_user_id, pc_agent_id, confirmed_at, created_at
		FROM pairing_sessions
		WHERE id = $1 AND device_code = $2
	`

	return scanPairingSession(r.db.QueryRowContext(ctx, query, id, deviceCode))
}

func (r *PairingSessionRepository) ExistsByDeviceCode(ctx context.Context, deviceCode string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM pairing_sessions
			WHERE device_code = $1
		)
	`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, deviceCode).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *PairingSessionRepository) Expire(ctx context.Context, id string) (*domain.PairingSession, error) {
	const query = `
		UPDATE pairing_sessions
		SET status = $2
		WHERE id = $1 AND status = $3
		RETURNING id, device_code, requested_pc_agent_id, device_name, os_type, status, expires_at,
			confirmed_by_user_id, pc_agent_id, confirmed_at, created_at
	`

	return scanPairingSession(r.db.QueryRowContext(ctx, query, id, domain.PairingStatusExpired, domain.PairingStatusPending))
}

func (r *PairingSessionRepository) Confirm(ctx context.Context, deviceCode string, userID string, now time.Time) (*domain.PCAgent, *domain.PairingSession, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	session, err := scanPairingSession(tx.QueryRowContext(
		ctx,
		`
			SELECT id, device_code, requested_pc_agent_id, device_name, os_type, status, expires_at,
				confirmed_by_user_id, pc_agent_id, confirmed_at, created_at
			FROM pairing_sessions
			WHERE device_code = $1
			FOR UPDATE
		`,
		deviceCode,
	))
	if err != nil {
		return nil, nil, err
	}
	if session.Status != domain.PairingStatusPending {
		return nil, session, repo.ErrPairingSessionNotPending
	}
	if now.After(session.ExpiresAt) {
		expired, expireErr := scanPairingSession(tx.QueryRowContext(
			ctx,
			`
				UPDATE pairing_sessions
				SET status = $2
				WHERE id = $1
				RETURNING id, device_code, requested_pc_agent_id, device_name, os_type, status, expires_at,
					confirmed_by_user_id, pc_agent_id, confirmed_at, created_at
			`,
			session.ID,
			domain.PairingStatusExpired,
		))
		if expireErr != nil {
			return nil, nil, expireErr
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, nil, commitErr
		}

		return nil, expired, repo.ErrPairingSessionExpired
	}
	if session.RequestedPCAgentID == nil || *session.RequestedPCAgentID == "" {
		return nil, nil, errors.New("pairing session missing pc agent id")
	}

	agent, err := scanPCAgent(tx.QueryRowContext(
		ctx,
		`
			INSERT INTO pc_agents (id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status)
			VALUES ($1, $2, $3, $4, NULL, $5, $6)
			RETURNING id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status, last_seen_at, created_at
		`,
		*session.RequestedPCAgentID,
		userID,
		session.DeviceName,
		session.OSType,
		domain.AgentStatusOffline,
		domain.ProtectionStatusDisabled,
	))
	if err != nil {
		return nil, nil, err
	}

	confirmed, err := scanPairingSession(tx.QueryRowContext(
		ctx,
		`
			UPDATE pairing_sessions
			SET status = $2,
				confirmed_by_user_id = $3,
				pc_agent_id = $4,
				confirmed_at = $5
			WHERE id = $1
			RETURNING id, device_code, requested_pc_agent_id, device_name, os_type, status, expires_at,
				confirmed_by_user_id, pc_agent_id, confirmed_at, created_at
		`,
		session.ID,
		domain.PairingStatusConfirmed,
		userID,
		agent.ID,
		now,
	))
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	return agent, confirmed, nil
}

type pairingSessionRow interface {
	Scan(dest ...any) error
}

func scanPairingSession(row pairingSessionRow) (*domain.PairingSession, error) {
	var session domain.PairingSession
	var requestedPCAgentID sql.NullString
	var confirmedByUserID sql.NullString
	var pcAgentID sql.NullString
	var confirmedAt sql.NullTime

	if err := row.Scan(
		&session.ID,
		&session.DeviceCode,
		&requestedPCAgentID,
		&session.DeviceName,
		&session.OSType,
		&session.Status,
		&session.ExpiresAt,
		&confirmedByUserID,
		&pcAgentID,
		&confirmedAt,
		&session.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrPairingSessionNotFound
		}
		return nil, err
	}

	if confirmedByUserID.Valid {
		session.ConfirmedByUserID = &confirmedByUserID.String
	}
	if requestedPCAgentID.Valid {
		session.RequestedPCAgentID = &requestedPCAgentID.String
	}
	if pcAgentID.Valid {
		session.PCAgentID = &pcAgentID.String
	}
	if confirmedAt.Valid {
		session.ConfirmedAt = &confirmedAt.Time
	}

	return &session, nil
}
