package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"walras-exchange/internal/db"
	"walras-exchange/internal/tunnel"
)

type OpsBroadcaster struct {
	clients map[chan []byte]bool
	mu      sync.RWMutex
}

var GlobalOpsBroadcaster = &OpsBroadcaster{
	clients: make(map[chan []byte]bool),
}

func init() {
	tunnel.OnPoolStateChanged = func() {
		GlobalOpsBroadcaster.BroadcastTelemetry()
	}
}

func (b *OpsBroadcaster) Subscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[ch] = true
}

func (b *OpsBroadcaster) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
	close(ch)
}

func (b *OpsBroadcaster) Broadcast(eventName string, data interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.clients) == 0 {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, payload))
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (b *OpsBroadcaster) BroadcastTelemetry() {
	telemetry := tunnel.GlobalTunnelPool.GetTelemetry()
	b.Broadcast("telemetry", telemetry)
}

func (b *OpsBroadcaster) BroadcastTransaction(item SettlementTapeItem) {
	b.Broadcast("transaction", item)
}

type OpsHandler struct {
	Repo *db.Repository
}

func NewOpsHandler(repo *db.Repository) *OpsHandler {
	return &OpsHandler{Repo: repo}
}

// HandleTelemetry serves GET /v1/ops/telemetry (<100 microseconds from Go Hot RAM)
func (h *OpsHandler) HandleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	telemetry := tunnel.GlobalTunnelPool.GetTelemetry()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(telemetry)
}

type SettlementTapeItem struct {
	TxID             string `json:"tx_id"`
	BuyerName        string `json:"buyer_name"`
	SellerNodeName   string `json:"seller_node_name"`
	ToolName         string `json:"tool_name"`
	BaseComputeUSD   string `json:"base_compute_usd"`
	EnterpriseFeeUSD string `json:"enterprise_fee_usd"`
	TotalChargedUSD  string `json:"total_charged_usd"`
	NetProfitUSD     string `json:"net_profit_usd"`
	LatencyMS        int    `json:"latency_ms"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
}

// HandleSettlementTape serves GET /v1/ops/settlement-tape (Last 30 cleared transactions)
func (h *OpsHandler) HandleSettlementTape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	query := `
		SELECT t.id, COALESCE(o.name, 'Enterprise Buyer'), COALESCE(a.name, 'GPU-Worker-Node'),
		       t.tool_name, t.base_compute_micro_usd, t.enterprise_fee_micro_usd, t.total_charged_micro_usd,
		       t.latency_ms, t.prompt_tokens, t.completion_tokens, t.status, t.created_at
		FROM buyer_usage_transactions t
		LEFT JOIN owners o ON t.enterprise_owner_id = o.id
		LEFT JOIN agents a ON t.seller_agent_id = a.id
		ORDER BY t.created_at DESC
		LIMIT 30
	`

	rows, err := h.Repo.DB.Query(query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tape []SettlementTapeItem
	for rows.Next() {
		var item SettlementTapeItem
		var base, fee, total int64
		var createdAt time.Time

		if err := rows.Scan(
			&item.TxID,
			&item.BuyerName,
			&item.SellerNodeName,
			&item.ToolName,
			&base,
			&fee,
			&total,
			&item.LatencyMS,
			&item.PromptTokens,
			&item.CompletionTokens,
			&item.Status,
			&createdAt,
		); err == nil {
			item.BaseComputeUSD = formatMicroUSD(base)
			item.EnterpriseFeeUSD = formatMicroUSD(fee)
			item.TotalChargedUSD = formatMicroUSD(total)
			// Programmatic 4% Net spread = 1% seller take + 3% buyer fee
			sellerTake := base / 100
			netProfit := fee + sellerTake
			item.NetProfitUSD = formatMicroUSD(netProfit)
			item.TotalTokens = item.PromptTokens + item.CompletionTokens
			item.CreatedAt = createdAt.Format("15:04:05")
			tape = append(tape, item)
		}
	}

	if tape == nil {
		tape = []SettlementTapeItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_cleared_records": len(tape),
		"transactions":          tape,
	})
}

// HandleKillswitch serves POST /v1/ops/killswitch
func (h *OpsHandler) HandleKillswitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AgentID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_agent_id"})
		return
	}

	err := tunnel.GlobalTunnelPool.DisconnectNode(req.AgentID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "quarantined",
		"agent_id": req.AgentID,
		"message":  "Node disconnected and isolated from active tunnel pool.",
	})
}

func formatMicroUSD(micro int64) string {
	dollars := float64(micro) / 1000000.0
	return fmt.Sprintf("$%.4f", dollars)
}

// HandleStream serves GET /v1/ops/stream (Real-Time Server-Sent Events SSE <1ms)
func (h *OpsHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error": "streaming_unsupported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	// 1. Send initial telemetry snapshot immediately
	initialTelemetry := tunnel.GlobalTunnelPool.GetTelemetry()
	initData, _ := json.Marshal(initialTelemetry)
	fmt.Fprintf(w, "event: telemetry\ndata: %s\n\n", initData)
	flusher.Flush()

	// 2. Register to live event channel
	ch := make(chan []byte, 32)
	GlobalOpsBroadcaster.Subscribe(ch)
	defer GlobalOpsBroadcaster.Unsubscribe(ch)

	// Heartbeat tick every 5 seconds to keep connection alive
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write(msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
