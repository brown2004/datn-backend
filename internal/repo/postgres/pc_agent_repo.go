package postgres

import (
	"context"
	"database/sql"
	"errors"

	"datn-backend/internal/domain"
	"datn-backend/internal/repo"
)

type PCAgentRepository struct {
	db *sql.DB
}

func NewPCAgentRepository(db *sql.DB) *PCAgentRepository {
	return &PCAgentRepository{db: db}
}

func (r *PCAgentRepository) Create(ctx context.Context, agent domain.PCAgent) (*domain.PCAgent, error) {
	const query = `
		INSERT INTO pc_agents (user_id, device_name, device_code, os_type, agent_status, protection_status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, device_name, device_code, os_type, agent_status, protection_status, last_seen_at
	`

	created, err := scanPCAgent(r.db.QueryRowContext(
		ctx,
		query,
		agent.UserID,
		agent.DeviceName,
		agent.DeviceCode,
		agent.OSType,
		agent.Status,
		agent.ProtectionStatus,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, repo.ErrPCAgentAlreadyLinked
		}
		return nil, err
	}

	return created, nil
}

func (r *PCAgentRepository) FindByID(ctx context.Context, id string) (*domain.PCAgent, error) {
	const query = `
		SELECT id, user_id, device_name, device_code, os_type, agent_status, protection_status, last_seen_at
		FROM pc_agents
		WHERE id = $1
	`

	return scanPCAgent(r.db.QueryRowContext(ctx, query, id))
}

func (r *PCAgentRepository) FindByUserID(ctx context.Context, userID string) ([]domain.PCAgent, error) {
	const query = `
		SELECT id, user_id, device_name, device_code, os_type, agent_status, protection_status, last_seen_at
		FROM pc_agents
		WHERE user_id = $1
		ORDER BY device_name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []domain.PCAgent
	for rows.Next() {
		agent, err := scanPCAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, *agent)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return agents, nil
}

func (r *PCAgentRepository) Save(ctx context.Context, agent *domain.PCAgent) error {
	const query = `
		UPDATE pc_agents
		SET device_name = $2,
			agent_status = $3,
			protection_status = $4,
			last_seen_at = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, agent.ID, agent.DeviceName, agent.Status, agent.ProtectionStatus, agent.LastSeenAt)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repo.ErrPCAgentNotFound
	}

	return nil
}

type pcAgentRow interface {
	Scan(dest ...any) error
}

func scanPCAgent(row pcAgentRow) (*domain.PCAgent, error) {
	var agent domain.PCAgent
	var lastSeenAt sql.NullTime

	if err := row.Scan(
		&agent.ID,
		&agent.UserID,
		&agent.DeviceName,
		&agent.DeviceCode,
		&agent.OSType,
		&agent.Status,
		&agent.ProtectionStatus,
		&lastSeenAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrPCAgentNotFound
		}
		return nil, err
	}

	if lastSeenAt.Valid {
		agent.LastSeenAt = &lastSeenAt.Time
	}

	return &agent, nil
}
