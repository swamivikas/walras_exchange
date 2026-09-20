package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"walras-exchange/internal/auth"
	"walras-exchange/internal/vault"
	"walras-exchange/pkg/protocol"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type Repository struct {
	DB *sql.DB
}

// NewRepository creates and verifies a MySQL connection
func NewRepository(dsn string) (*Repository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection Pool Settings
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{DB: db}, nil
}

// CreateUserAndDefaultAgent atomically provisions an owner and a default buyer agent
func (r *Repository) CreateUserAndDefaultAgent(req *protocol.SignupRequest) (*protocol.SignupResponse, error) {
	// 1. Hash the password with bcrypt
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("crypto error: %w", err)
	}

	// 2. Generate Owner UUID
	ownerID := uuid.New().String()

	// 4. Start Atomic Database Transaction
	tx, err := r.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe rollback if tx fails before Commit

	// 5. Query 1: Insert into `owners` table (With $50 Trial & 30-Day Expiry)
	trialExpiresAt := time.Now().Add(30 * 24 * time.Hour)
	ownerQuery := `
		INSERT INTO owners (
			id, name, email, password_hash, user_type, 
			credit_limit_micro_usd, plan_type, trial_expires_at
		) VALUES (?, ?, ?, ?, ?, ?, 'FREE_TRIAL', ?)
	`
	defaultCreditLimit := int64(50000000) // $50.00 in microUSD
	_, err = tx.Exec(ownerQuery, ownerID, req.Name, req.Email, passwordHash, req.UserType, defaultCreditLimit, trialExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert owner: %w", err)
	}

	// 6. Query 2: Provision EXACTLY 1 Initial Agent strictly matching user's chosen role
	agentID := uuid.New().String()
	apiKey, err := auth.GenerateAPIKey(ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key: %w", err)
	}

	// 6. Query 2: Provision EXACTLY 1 Default Buyer Agent for EVERY signup
	agentName := "Enterprise-Buyer-Client"
	agentRole := "BUYER"
	vramGB := 0

	agentQuery := `
		INSERT INTO agents (
			id, owner_id, name, agent_type, max_concurrency, vram_gb, agent_role, 
			api_key_hash, api_key_prefix, balance_micro_usd, 
			status, approval_status
		) VALUES (?, ?, ?, 'GENERAL_LLM_INFERENCE', 4, ?, ?, ?, ?, 0, 'ACTIVE', 'APPROVED')
	`
	_, err = tx.Exec(agentQuery, agentID, ownerID, agentName, vramGB, agentRole, apiKey.KeyHash, apiKey.KeyPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to insert initial buyer agent: %w", err)
	}

	// 7. Commit Transaction to MySQL
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 8. Hydrate Buyer Agent in Go Hot RAM (<50ns Live Execution Ready)
	cachedAgent := &vault.CachedAgent{
		AgentID:        agentID,
		OwnerID:        ownerID,
		Role:           "BUYER",
		AgentType:      "GENERAL_LLM_INFERENCE",
		Status:         "ACTIVE",
		ApprovalStatus: "APPROVED",
		CreditLimit:    defaultCreditLimit,
	}
	vault.GlobalVault.RegisterAgentInRAM(apiKey.KeyHash, cachedAgent)

	// 9. Build and Return Response
	return &protocol.SignupResponse{
		OwnerID:  ownerID,
		Name:     req.Name,
		Email:    req.Email,
		UserType: req.UserType,
		DefaultAgent: protocol.DefaultAgentInfo{
			AgentID:        agentID,
			Name:           agentName,
			Role:           agentRole,
			AgentType:      "GENERAL_LLM_INFERENCE",
			Status:         "ACTIVE",
			ApprovalStatus: "APPROVED",
			TrialBalance:   "$50.00 USD",
			APIKey:         apiKey.RawKey, // Shown only ONCE
			APIKeyPrefix:   apiKey.KeyPrefix,
		},
		Message: "User account and initial agent node initialized successfully.",
	}, nil
}

// AuthenticateUser verifies user credentials and returns account data + registered agents
func (r *Repository) AuthenticateUser(email, password string) (*protocol.LoginResponse, error) {
	var ownerID, name, dbEmail, passwordHash, userType, planType string
	var creditLimitMicroUSD int64

	query := `
		SELECT id, name, email, password_hash, user_type, credit_limit_micro_usd, plan_type 
		FROM owners 
		WHERE email = ? 
		LIMIT 1
	`
	err := r.DB.QueryRow(query, email).Scan(
		&ownerID, &name, &dbEmail, &passwordHash, &userType, &creditLimitMicroUSD, &planType,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid_email_or_password")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Verify bcrypt password
	if !auth.CheckPassword(password, passwordHash) {
		return nil, fmt.Errorf("invalid_email_or_password")
	}

	// Fetch all agents owned by this user
	agents, err := r.GetUserAgents(ownerID)
	if err != nil {
		return nil, err
	}

	balanceUSD := fmt.Sprintf("$%.2f USD", float64(creditLimitMicroUSD)/1000000.0)

	return &protocol.LoginResponse{
		OwnerID:             ownerID,
		Name:                name,
		Email:               dbEmail,
		UserType:            userType,
		CreditLimitMicroUSD: creditLimitMicroUSD,
		BalanceUSD:          balanceUSD,
		PlanType:            planType,
		Agents:              agents,
	}, nil
}

// GetUserAgents fetches all agents owned by an owner ID
func (r *Repository) GetUserAgents(ownerID string) ([]protocol.UserAgentItem, error) {
	agentQuery := `
		SELECT a.id, a.name, a.agent_role, a.agent_type, a.status, a.approval_status, a.balance_micro_usd, COALESCE(a.api_key_prefix, ''),
		       COALESCE(c.tool_name, ''), COALESCE(a.vram_gb, 0), COALESCE(a.max_concurrency, 1),
		       COALESCE(c.description, ''), COALESCE(c.input_token_price_micro_usd, 0), COALESCE(c.output_token_price_micro_usd, 0),
		       COALESCE(c.fixed_cost_per_call_micro_usd, 0), COALESCE(c.reputation_score, 5.00)
		FROM agents a
		LEFT JOIN capabilities c ON a.id = c.agent_id
		WHERE a.owner_id = ?
		ORDER BY a.created_at DESC
	`
	rows, err := r.DB.Query(agentQuery, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user agents: %w", err)
	}
	defer rows.Close()

	var agents []protocol.UserAgentItem
	for rows.Next() {
		var item protocol.UserAgentItem
		if err := rows.Scan(
			&item.AgentID,
			&item.Name,
			&item.Role,
			&item.AgentType,
			&item.Status,
			&item.ApprovalStatus,
			&item.BalanceMicroUSD,
			&item.APIKeyPrefix,
			&item.Model,
			&item.VRAMGB,
			&item.MaxConcurrency,
			&item.Description,
			&item.InputTokenPriceMicroUSD,
			&item.OutputTokenPriceMicroUSD,
			&item.FixedCostPerCallMicroUSD,
			&item.ReputationScore,
		); err != nil {
			log.Printf("⚠️ [DB] Error scanning agent row: %v", err)
		} else {
			agents = append(agents, item)
		}
	}

	if agents == nil {
		agents = []protocol.UserAgentItem{}
	}

	return agents, nil
}
