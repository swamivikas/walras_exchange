package db

import (
	"database/sql"
	"errors"
	"fmt"

	"walras-exchange/internal/auth"
	"walras-exchange/internal/vault"
	"walras-exchange/pkg/protocol"

	"github.com/google/uuid"
)

// AddSellerAgent verifies owner role and atomically provisions a seller agent & capability
func (r *Repository) AddSellerAgent(ownerID string, req *protocol.AddSellerAgentRequest) (*protocol.AddSellerAgentResponse, error) {
	// 1. Verify Owner exists and check user_type permission
	var userType string
	err := r.DB.QueryRow("SELECT user_type FROM owners WHERE id = ?", ownerID).Scan(&userType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("owner_not_found")
		}
		return nil, fmt.Errorf("failed to query owner: %w", err)
	}

	// 2. Enforce Role Permission: Only SELLER_OWNER or HYBRID can add seller agents
	if userType == "ENTERPRISE_BUYER" {
		return nil, errors.New("permission_denied: enterprise buyers cannot register seller agents, please upgrade to SELLER_OWNER or HYBRID")
	}

	// 3. Generate UUIDs and Dedicated Node API Key
	agentID := uuid.New().String()
	capabilityID := uuid.New().String()

	generatedKey, err := auth.GenerateAPIKey(ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate node api key: %w", err)
	}

	// 4. Default JSON schema if not provided
	if req.InputJSONSchema == "" {
		req.InputJSONSchema = `{"type": "object", "properties": {}}`
	}

	// 5. Start Atomic Database Transaction
	tx, err := r.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 6. Insert into `agents` table with max_concurrency and vram_gb
	agentQuery := `
		INSERT INTO agents (
			id, owner_id, name, agent_type, max_concurrency, vram_gb, agent_role,
			api_key_hash, api_key_prefix, balance_micro_usd,
			status, approval_status
		) VALUES (?, ?, ?, ?, ?, ?, 'SELLER', ?, ?, 0, 'OFFLINE', 'PENDING_REVIEW')
	`
	if req.MaxConcurrency <= 0 {
		req.MaxConcurrency = 1
	}
	_, err = tx.Exec(
		agentQuery, 
		agentID, 
		ownerID, 
		req.NodeName, 
		req.AgentType, 
		req.MaxConcurrency, 
		req.VRAMGigabytes, 
		generatedKey.KeyHash, 
		generatedKey.KeyPrefix,
	)

	// 7. Insert into `capabilities` table with unified pricing
	capQuery := `
		INSERT INTO capabilities (
			id, agent_id, tool_name, description, json_schema,
			input_token_price_micro_usd, output_token_price_micro_usd, fixed_cost_per_call_micro_usd,
			avg_latency_ms, reputation_score, is_active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 5.00, 1)
	`
	_, err = tx.Exec(
		capQuery, 
		capabilityID, 
		agentID, 
		req.ToolName, 
		req.Description, 
		req.InputJSONSchema, 
		req.InputTokenPriceMicroUSD, 
		req.OutputTokenPriceMicroUSD, 
		req.FixedCostPerCallMicroUSD,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert capability: %w", err)
	}

	// 8. Commit Transaction to MySQL
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 9. Hydrate Go Hot RAM
	cachedAgent := &vault.CachedAgent{
		AgentID:        agentID,
		OwnerID:        ownerID,
		Role:           "SELLER",
		AgentType:      req.AgentType,
		Status:         "OFFLINE",
		ApprovalStatus: "PENDING_REVIEW",
		CreditLimit:    0,
	}
	vault.GlobalVault.RegisterAgentInRAM(generatedKey.KeyHash, cachedAgent)

	// 10. Return Response
	return &protocol.AddSellerAgentResponse{
		OwnerID:        ownerID,
		AgentID:        agentID,
		CapabilityID:   capabilityID,
		ToolName:       req.ToolName,
		AgentType:      req.AgentType,
		ApprovalStatus: "PENDING_REVIEW",
		NodeAPIKey:     generatedKey.RawKey,
		NodeKeyPrefix:  generatedKey.KeyPrefix,
		Message:        "Seller agent and tool capability registered successfully. Status is PENDING_REVIEW (OFFLINE until daemon connects).",
	}, nil
}
