package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

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
		INSERT INTO pc_agents (user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status, last_seen_at, created_at
	`

	return scanPCAgent(r.db.QueryRowContext(
		ctx,
		query,
		agent.UserID,
		agent.DeviceName,
		agent.OSType,
		agent.AgentSecretHash,
		agent.Status,
		agent.ProtectionStatus,
	))
}

func (r *PCAgentRepository) FindByID(ctx context.Context, id string) (*domain.PCAgent, error) {
	const query = `
		SELECT id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status, last_seen_at, created_at
		FROM pc_agents
		WHERE id = $1
	`

	return scanPCAgent(r.db.QueryRowContext(ctx, query, id))
}

func (r *PCAgentRepository) FindByUserID(ctx context.Context, userID string) ([]domain.PCAgent, error) {
	const query = `
		SELECT id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status, last_seen_at, created_at
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

func (r *PCAgentRepository) SetAgentSecretHashIfEmpty(ctx context.Context, id string, secretHash string) (*domain.PCAgent, bool, error) {
	const query = `
		UPDATE pc_agents
		SET agent_secret_hash = $2
		WHERE id = $1 AND agent_secret_hash IS NULL
		RETURNING id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status, last_seen_at, created_at
	`

	agent, err := scanPCAgent(r.db.QueryRowContext(ctx, query, id, secretHash))
	if err == nil {
		return agent, true, nil
	}
	if !errors.Is(err, repo.ErrPCAgentNotFound) {
		return nil, false, err
	}

	agent, err = r.FindByID(ctx, id)
	if err != nil {
		return nil, false, err
	}

	return agent, false, nil
}

func (r *PCAgentRepository) MarkVerified(ctx context.Context, id string, verifiedAt time.Time) (*domain.PCAgent, error) {
	const query = `
		UPDATE pc_agents
		SET agent_status = $2,
			last_seen_at = $3
		WHERE id = $1
		RETURNING id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status, last_seen_at, created_at
	`

	return scanPCAgent(r.db.QueryRowContext(ctx, query, id, domain.AgentStatusOnline, verifiedAt))
}

func (r *PCAgentRepository) UpdateProtectionStatusByID(ctx context.Context, id string, protectionStatus string) (*domain.PCAgent, error) {
	const query = `
		UPDATE pc_agents
		SET protection_status = $2
		WHERE id = $1
		RETURNING id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status, last_seen_at, created_at
	`

	return scanPCAgent(r.db.QueryRowContext(ctx, query, id, protectionStatus))
}

func (r *PCAgentRepository) UpdateProtectionStatusByIDAndUserID(ctx context.Context, id string, userID string, protectionStatus string) (*domain.PCAgent, error) {
	const query = `
		UPDATE pc_agents
		SET protection_status = $3
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, device_name, os_type, agent_secret_hash, agent_status, protection_status, last_seen_at, created_at
	`

	return scanPCAgent(r.db.QueryRowContext(ctx, query, id, userID, protectionStatus))
}

func (r *PCAgentRepository) DeleteByIDAndUserID(ctx context.Context, id string, userID string) error {
	const query = `
		DELETE FROM pc_agents
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, id, userID)
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
	var userID sql.NullString
	var agentSecretHash sql.NullString
	var lastSeenAt sql.NullTime

	if err := row.Scan(
		&agent.ID,
		&userID,
		&agent.DeviceName,
		&agent.OSType,
		&agentSecretHash,
		&agent.Status,
		&agent.ProtectionStatus,
		&lastSeenAt,
		&agent.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrPCAgentNotFound
		}
		return nil, err
	}

	if userID.Valid {
		agent.UserID = userID.String
	}
	if agentSecretHash.Valid {
		agent.AgentSecretHash = &agentSecretHash.String
	}
	if lastSeenAt.Valid {
		agent.LastSeenAt = &lastSeenAt.Time
	}

	return &agent, nil
}
