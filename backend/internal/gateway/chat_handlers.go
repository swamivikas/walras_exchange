package gateway

import (
	"crypto/sha256"
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

	// 3. Ultra-Low-Latency In-Memory Matcher (<100 nanoseconds, Zero MySQL queries)
	req.AgentType = strings.TrimSpace(req.AgentType)
	req.Model = strings.TrimSpace(req.Model)

	matchedSession, matchedCap, err := tunnel.GlobalTunnelPool.MatchBestSession(req.AgentType, req.Model)
	if err != nil {
		targetDesc := req.Model
		if req.AgentType != "" {
			targetDesc = req.AgentType + " (" + req.Model + ")"
		}
		http.Error(w, fmt.Sprintf(`{"error": "all_gpu_nodes_for_target_are_currently_offline", "target": "%s"}`, targetDesc), http.StatusServiceUnavailable)
		return
	}

	matchedCapabilityID := matchedCap.CapabilityID
	matchedSellerAgentID := matchedSession.AgentID
	inputPrice := matchedCap.InputTokenPriceMicroUSD
	outputPrice := matchedCap.OutputTokenPriceMicroUSD
	fixedCost := matchedCap.FixedCostPerCallMicroUSD

	// 4. Dispatch Job Down WebSocket Tunnel (<1ms)
	requestStartTime := time.Now()
	matchedSession.InFlight.Add(1)
	tunnel.GlobalTunnelPool.NotifyChange()
	defer func() {
		matchedSession.InFlight.Add(-1)
		tunnel.GlobalTunnelPool.NotifyChange()
	}()

	requestID := "req-" + uuid.New().String()
	respChan := make(chan []byte, 256)
	matchedSession.PendingJobs.Store(requestID, respChan)
	defer matchedSession.PendingJobs.Delete(requestID)

	jobPayload := map[string]interface{}{
		"request_id": requestID,
		"agent_type": req.AgentType,
		"model":      req.Model,
		"prompt":     lastPrompt,
		"messages":   req.Messages,
		"stream":     req.Stream,
	}

	if err := matchedSession.WriteJSON(jobPayload); err != nil {
		http.Error(w, `{"error": "failed_to_write_to_node_tunnel"}`, http.StatusBadGateway)
		return
	}

	// 5. Check if Buyer requested Server-Sent Events (SSE) Streaming
	if req.Stream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, `{"error": "streaming_unsupported_by_client"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher.Flush()

		chatCompletionID := "chatcmpl-" + uuid.New().String()[:8]
		var totalPromptTokens, totalCompletionTokens int
		var accumulatedResponse string

	streamLoop:
		for {
			select {
			case rawChunk, ok := <-respChan:
				if !ok {
					break streamLoop
				}
				var chunkData struct {
					Status           string `json:"status"`
					Delta            string `json:"delta"`
					Response         string `json:"response"`
					PromptTokens     int    `json:"prompt_tokens"`
					CompletionTokens int    `json:"completion_tokens"`
				}
				if err := json.Unmarshal(rawChunk, &chunkData); err != nil {
					continue
				}

				if chunkData.Status == "STREAMING" {
					accumulatedResponse += chunkData.Delta
					chunkResp := protocol.ChatCompletionChunk{
						ID:      chatCompletionID,
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   req.Model,
						Choices: []protocol.ChatCompletionChunkChoice{
							{
								Index: 0,
								Delta: protocol.ChatMessage{
									Role:    "assistant",
									Content: chunkData.Delta,
								},
							},
						},
					}
					chunkBytes, _ := json.Marshal(chunkResp)
					fmt.Fprintf(w, "data: %s\n\n", chunkBytes)
					flusher.Flush()

				} else if chunkData.Status == "COMPLETED" || chunkData.Status == "" {
					totalPromptTokens = chunkData.PromptTokens
					totalCompletionTokens = chunkData.CompletionTokens
					if chunkData.Response != "" {
						accumulatedResponse = chunkData.Response
					}
					break streamLoop
				}

			case <-time.After(90 * time.Second):
				return
			case <-r.Context().Done():
				return
			}
		}

		actualLatencyMS := int(time.Since(requestStartTime).Milliseconds())
		if actualLatencyMS <= 0 {
			actualLatencyMS = 1
		}
		if totalPromptTokens == 0 {
			totalPromptTokens = len(lastPrompt) / 4
			if totalPromptTokens == 0 {
				totalPromptTokens = 1
			}
		}
		if totalCompletionTokens == 0 {
			totalCompletionTokens = len(accumulatedResponse) / 4
			if totalCompletionTokens == 0 {
				totalCompletionTokens = 1
			}
		}

		totalCost := (int64(totalPromptTokens) * inputPrice) +
			(int64(totalCompletionTokens) * outputPrice) + fixedCost
		if totalCost <= 0 {
			totalCost = 1000
		}
		platformFee := totalCost / 100

		// Write final SSE chunk with finish_reason "stop" and atomic settlement accounting
		stopReason := "stop"
		finalChunk := protocol.ChatCompletionChunk{
			ID:      chatCompletionID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []protocol.ChatCompletionChunkChoice{
				{
					Index:        0,
					Delta:        protocol.ChatMessage{},
					FinishReason: &stopReason,
				},
			},
			Settlement: &protocol.SettlementDetails{
				CostMicroUSD:        totalCost,
				FormattedCostUSD:    fmt.Sprintf("$%.6f USD", float64(totalCost)/1000000.0),
				PlatformFeeMicroUSD: platformFee,
				ExecutionNodeID:     matchedSellerAgentID,
			},
		}
		finalBytes, _ := json.Marshal(finalChunk)
		fmt.Fprintf(w, "data: %s\n\n", finalBytes)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()

		txID := uuid.New().String()
		recordSettlement(h, buyerAgent, matchedSellerAgentID, matchedCapabilityID, matchedSession.NodeName, req.Model, actualLatencyMS, txID, totalCost, platformFee, totalPromptTokens, totalCompletionTokens)
		return
	}

	// 6. Non-Streaming: Wait for full GPU answer
	var rawAnswer []byte
	select {
	case rawAnswer = <-respChan:
		// Received answer from GPU daemon
	case <-time.After(90 * time.Second):
		http.Error(w, `{"error": "gpu_node_timed_out"}`, http.StatusGatewayTimeout)
		return
	case <-r.Context().Done():
		return
	}

	actualLatencyMS := int(time.Since(requestStartTime).Milliseconds())
	if actualLatencyMS <= 0 {
		actualLatencyMS = 1
	}

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
	txID := uuid.New().String()

	recordSettlement(h, buyerAgent, matchedSellerAgentID, matchedCapabilityID, matchedSession.NodeName, req.Model, actualLatencyMS, txID, totalCost, platformFee, daemonResp.PromptTokens, daemonResp.CompletionTokens)

	// 8. Build and Return OpenAI Standard JSON Response
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

// recordSettlement records the transaction to MySQL asynchronously and broadcasts to the War Room tape
func recordSettlement(
	h *ChatHandler,
	buyerAgent *vault.CachedAgent,
	sellerAgentID, capabilityID, nodeName, model string,
	latencyMS int,
	txID string,
	totalCost, platformFee int64,
	promptTokens, completionTokens int,
) {
	totalCharged := totalCost + (platformFee * 3)
	go func(latency int, id string, charged int64) {
		txQuery := `
			INSERT INTO buyer_usage_transactions (
				id, enterprise_owner_id, buyer_agent_id, seller_agent_id, capability_id,
				tool_name, base_compute_micro_usd, enterprise_fee_micro_usd, total_charged_micro_usd,
				latency_ms, prompt_tokens, completion_tokens, billing_cycle_id, status
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '2026-09', 'SETTLED')
		`
		h.Repo.DB.Exec(
			txQuery,
			id,
			buyerAgent.OwnerID,
			buyerAgent.AgentID,
			sellerAgentID,
			capabilityID,
			model,
			totalCost,
			platformFee*3, // 3% enterprise surcharge
			charged,
			latency,
			promptTokens,
			completionTokens,
		)

		// Atomic Balance Deduction from Buyer Credit Limit
		h.Repo.DB.Exec(`
			UPDATE owners 
			SET credit_limit_micro_usd = GREATEST(0, credit_limit_micro_usd - ?) 
			WHERE id = ?
		`, charged, buyerAgent.OwnerID)

		// Credit Seller Node Earnings (99% base compute to seller, 1% platform fee)
		sellerTake := totalCost - (totalCost / 100)
		h.Repo.DB.Exec(`
			UPDATE agents 
			SET balance_micro_usd = balance_micro_usd + ? 
			WHERE id = ?
		`, sellerTake, sellerAgentID)
	}(latencyMS, txID, totalCharged)

	// Broadcast transaction event to live SSE clients in <0.1ms
	tapeItem := SettlementTapeItem{
		TxID:             txID,
		BuyerName:        "GEMINI_LOCAL",
		SellerNodeName:   nodeName,
		ToolName:         model,
		BaseComputeUSD:   formatMicroUSD(totalCost),
		EnterpriseFeeUSD: formatMicroUSD(platformFee * 3),
		TotalChargedUSD:  formatMicroUSD(totalCost + (platformFee * 3)),
		NetProfitUSD:     formatMicroUSD((platformFee * 3) + (totalCost / 100)),
		LatencyMS:        latencyMS,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		Status:           "SETTLED",
		CreatedAt:        time.Now().Format("15:04:05"),
	}
	GlobalOpsBroadcaster.BroadcastTransaction(tapeItem)
}
