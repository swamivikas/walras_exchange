package gateway

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"walras-exchange/internal/db"
	"walras-exchange/internal/tunnel"
	"walras-exchange/internal/vault"
	"walras-exchange/pkg/protocol"

	"github.com/google/uuid"
)

type ChatHandler struct {
	Repo *db.Repository
}

func NewChatHandler(repo *db.Repository) *ChatHandler {
	return &ChatHandler{Repo: repo}
}

// HandleChatCompletions processes POST /v1/chat/completions (OpenAI Compatible)
func (h *ChatHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 1. Authenticate Buyer API Key in Go Hot RAM (<50ns)
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || token == authHeader {
		http.Error(w, `{"error": "missing_or_invalid_bearer_token"}`, http.StatusUnauthorized)
		return
	}

	hasher := sha256.New()
	hasher.Write([]byte(token))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	buyerAgent, err := vault.GlobalVault.GetAgentByAPIKeyHash(keyHash)
	if err != nil {
		// Cold DB Hydration for Buyer
		var dbAgent vault.CachedAgent
		query := `
			SELECT id, owner_id, agent_role, agent_type, status, approval_status 
			FROM agents 
			WHERE api_key_hash = ? AND status = 'ACTIVE' 
			LIMIT 1
		`
		err = h.Repo.DB.QueryRow(query, keyHash).Scan(
			&dbAgent.AgentID,
			&dbAgent.OwnerID,
			&dbAgent.Role,
			&dbAgent.AgentType,
			&dbAgent.Status,
			&dbAgent.ApprovalStatus,
		)
		if err != nil {
			http.Error(w, `{"error": "invalid_api_key"}`, http.StatusUnauthorized)
			return
		}
		vault.GlobalVault.RegisterAgentInRAM(keyHash, &dbAgent)
		buyerAgent = &dbAgent
	}

	// 2. Decode Chat Completion Request
	var req protocol.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid_json_payload"}`, http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, `{"error": "messages_cannot_be_empty"}`, http.StatusBadRequest)
		return
	}

	lastPrompt := req.Messages[len(req.Messages)-1].Content

	// 3. Matcher: Support optional AgentType domain targeting + Model matching
	var capQuery string
	var rows *sql.Rows

	req.AgentType = strings.TrimSpace(req.AgentType)
	req.Model = strings.TrimSpace(req.Model)

	if req.AgentType != "" {
		capQuery = `
			SELECT c.id, c.agent_id, c.input_token_price_micro_usd, c.output_token_price_micro_usd, c.fixed_cost_per_call_micro_usd 
			FROM capabilities c
			JOIN agents a ON c.agent_id = a.id
			WHERE a.agent_type = ?
			  AND (c.tool_name = ? OR ? = '')
			  AND c.is_active = 1 
			  AND a.status = 'ACTIVE' 
			  AND a.approval_status = 'APPROVED'
			ORDER BY c.reputation_score DESC
		`
		rows, err = h.Repo.DB.Query(capQuery, req.AgentType, req.Model, req.Model)
	} else {
		capQuery = `
			SELECT c.id, c.agent_id, c.input_token_price_micro_usd, c.output_token_price_micro_usd, c.fixed_cost_per_call_micro_usd 
			FROM capabilities c
			JOIN agents a ON c.agent_id = a.id
			WHERE c.tool_name = ? 
			  AND c.is_active = 1 
			  AND a.status = 'ACTIVE' 
			  AND a.approval_status = 'APPROVED'
			ORDER BY c.reputation_score DESC
		`
		rows, err = h.Repo.DB.Query(capQuery, req.Model)
	}

	if err != nil {
		http.Error(w, `{"error": "database_error_querying_capabilities"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var matchedSession *tunnel.TunnelSession
	var matchedCapabilityID, matchedSellerAgentID string
	var inputPrice, outputPrice, fixedCost int64
	foundOnlineNode := false

	for rows.Next() {
		var capID, agentID string
		var inPrice, outPrice, fCost int64
		if err := rows.Scan(&capID, &agentID, &inPrice, &outPrice, &fCost); err == nil {
			// Check if this candidate is online in our TunnelPool
			if session, err := tunnel.GlobalTunnelPool.GetSession(agentID); err == nil {
				matchedSession = session
				matchedCapabilityID = capID
				matchedSellerAgentID = agentID
				inputPrice = inPrice
				outputPrice = outPrice
				fixedCost = fCost
				foundOnlineNode = true
				break // Found the best online GPU node in this pool!
			}
		}
	}

	if !foundOnlineNode {
		targetDesc := req.Model
		if req.AgentType != "" {
			targetDesc = req.AgentType + " (" + req.Model + ")"
		}
		http.Error(w, fmt.Sprintf(`{"error": "all_gpu_nodes_for_target_are_currently_offline", "target": "%s"}`, targetDesc), http.StatusServiceUnavailable)
		return
	}

	// 4. Dispatch Job Down WebSocket Tunnel (<1ms)
	matchedSession.InFlight.Add(1)
	defer matchedSession.InFlight.Add(-1)

	requestID := "req-" + uuid.New().String()
	respChan := make(chan []byte, 1)
	matchedSession.PendingJobs.Store(requestID, respChan)

	jobPayload := map[string]interface{}{
		"request_id": requestID,
		"agent_type": req.AgentType,
		"model":      req.Model,
		"prompt":     lastPrompt,
		"messages":   req.Messages,
	}

	if err := matchedSession.WriteJSON(jobPayload); err != nil {
		matchedSession.PendingJobs.Delete(requestID)
		http.Error(w, `{"error": "failed_to_write_to_node_tunnel"}`, http.StatusBadGateway)
		return
	}

	// 5. Wait for GPU answer (with 15s timeout)
	var rawAnswer []byte
	select {
	case rawAnswer = <-respChan:
		// Received answer from GPU daemon!
	case <-time.After(15 * time.Second):
		matchedSession.PendingJobs.Delete(requestID)
		http.Error(w, `{"error": "gpu_node_timed_out"}`, http.StatusGatewayTimeout)
		return
	}

	// 6. Parse response from daemon
	var daemonResp struct {
		Response         string `json:"response"`
		PromptTokens     int    `json:"prompt_tokens"`
		CompletionTokens int    `json:"completion_tokens"`
	}
	json.Unmarshal(rawAnswer, &daemonResp)
	if daemonResp.PromptTokens == 0 {
		daemonResp.PromptTokens = len(lastPrompt) / 4
		if daemonResp.PromptTokens == 0 {
			daemonResp.PromptTokens = 1
		}
	}
	if daemonResp.CompletionTokens == 0 {
		daemonResp.CompletionTokens = len(daemonResp.Response) / 4
		if daemonResp.CompletionTokens == 0 {
			daemonResp.CompletionTokens = 1
		}
	}

	// 7. Atomic In-Memory Settlement (<50ns)
	totalCost := (int64(daemonResp.PromptTokens) * inputPrice) +
		(int64(daemonResp.CompletionTokens) * outputPrice) + fixedCost
	if totalCost <= 0 {
		totalCost = 1000 // Minimum $0.001
	}

	platformFee := totalCost / 100 // 1% platform fee

	// 8. Record in `buyer_usage_transactions` (Async DB insert)
	go func() {
		txQuery := `
			INSERT INTO buyer_usage_transactions (
				id, enterprise_owner_id, buyer_agent_id, seller_agent_id, capability_id,
				tool_name, base_compute_micro_usd, enterprise_fee_micro_usd, total_charged_micro_usd,
				latency_ms, prompt_tokens, completion_tokens, billing_cycle_id, status
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 300, ?, ?, '2026-09', 'SETTLED')
		`
		h.Repo.DB.Exec(
			txQuery,
			uuid.New().String(),
			buyerAgent.OwnerID,
			buyerAgent.AgentID,
			matchedSellerAgentID,
			matchedCapabilityID,
			req.Model,
			totalCost,
			platformFee*3, // 3% enterprise surcharge
			totalCost+(platformFee*3),
			daemonResp.PromptTokens,
			daemonResp.CompletionTokens,
		)
	}()

	// 9. Build and Return OpenAI Standard JSON Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(protocol.ChatCompletionResponse{
		ID:      "chatcmpl-" + uuid.New().String()[:8],
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []protocol.ChatChoice{
			{
				Index: 0,
				Message: protocol.ChatMessage{
					Role:    "assistant",
					Content: daemonResp.Response,
				},
				FinishReason: "stop",
			},
		},
		Usage: protocol.TokenUsage{
			PromptTokens:     daemonResp.PromptTokens,
			CompletionTokens: daemonResp.CompletionTokens,
			TotalTokens:      daemonResp.PromptTokens + daemonResp.CompletionTokens,
		},
		Settlement: protocol.SettlementDetails{
			CostMicroUSD:        totalCost,
			FormattedCostUSD:    fmt.Sprintf("$%.6f USD", float64(totalCost)/1000000.0),
			PlatformFeeMicroUSD: platformFee,
			ExecutionNodeID:     matchedSellerAgentID,
		},
	})
}
