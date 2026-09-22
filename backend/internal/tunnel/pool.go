package tunnel

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// NodeCapability represents an in-memory rate card and tool capability registered to a seller node
type NodeCapability struct {
	CapabilityID             string  `json:"capability_id"`
	AgentID                  string  `json:"agent_id"`
	ToolName                 string  `json:"tool_name"`
	InputTokenPriceMicroUSD  int64   `json:"input_token_price_micro_usd"`
	OutputTokenPriceMicroUSD int64   `json:"output_token_price_micro_usd"`
	FixedCostPerCallMicroUSD int64   `json:"fixed_cost_per_call_micro_usd"`
	ReputationScore          float64 `json:"reputation_score"`
	IsActive                 bool    `json:"is_active"`
}

// TunnelSession represents a live, persistent WebSocket stream from a seller node
type TunnelSession struct {
	AgentID        string
	OwnerID        string
	NodeName       string
	AgentType      string
	Model          string
	MaxConcurrency int32
	InFlight       atomic.Int32
	Conn           *websocket.Conn
	ConnectedAt    time.Time
	mu             sync.Mutex       // Protects concurrent writes down the same socket
	PendingJobs    sync.Map         // requestID -> chan []byte (for response multiplexing)
	Capabilities   []NodeCapability // Pinned in RAM for sub-microsecond matching
}

// WriteJSON sends a thread-safe payload down the open WebSocket stream
func (s *TunnelSession) WriteJSON(v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteJSON(v)
}

type NodeTelemetryItem struct {
	AgentID        string `json:"agent_id"`
	OwnerID        string `json:"owner_id"`
	NodeName       string `json:"node_name"`
	AgentType      string `json:"agent_type"`
	Model          string `json:"model"`
	MaxConcurrency int32  `json:"max_concurrency"`
	InFlight       int32  `json:"in_flight"`
	AvailableSlots int32  `json:"available_slots"`
	ConnectedAt    string `json:"connected_at"`
	UptimeSeconds  int64  `json:"uptime_seconds"`
	Status         string `json:"status"`
}

type DomainPoolStats struct {
	DomainName     string  `json:"domain_name"`
	OnlineNodes    int     `json:"online_nodes"`
	TotalCapacity  int32   `json:"total_capacity"`
	MaxCapacity    int32   `json:"max_capacity"`
	ActiveInFlight int32   `json:"active_in_flight"`
	InFlight       int32   `json:"in_flight"`
	AvailableSlots int32   `json:"available_slots"`
	UtilizationPct float64 `json:"utilization_pct"`
}

type PoolTelemetryResponse struct {
	Timestamp            int64                      `json:"timestamp"`
	TotalOnlineNodes     int                        `json:"total_online_nodes"`
	TotalNetworkCapacity int32                      `json:"total_network_capacity"`
	TotalInFlightJobs    int32                      `json:"total_in_flight_jobs"`
	GlobalUtilizationPct float64                    `json:"global_utilization_pct"`
	DomainPools          map[string]DomainPoolStats `json:"domain_pools"`
	ActiveNodes          []NodeTelemetryItem        `json:"active_nodes"`
	Nodes                []NodeTelemetryItem        `json:"nodes"`
}

// TunnelPool manages all active hardware connections globally
type TunnelPool struct {
	sessions sync.Map // agentID (string) -> *TunnelSession
}

// GlobalTunnelPool is the singleton tunnel registry
var GlobalTunnelPool = &TunnelPool{}

// OnPoolStateChanged is an optional callback invoked whenever nodes or slots change
var OnPoolStateChanged func()

// NotifyChange immediately invokes the real-time event hook
func (p *TunnelPool) NotifyChange() {
	if OnPoolStateChanged != nil {
		go OnPoolStateChanged()
	}
}

// RegisterSession adds a newly connected hardware node to the live pool
func (p *TunnelPool) RegisterSession(session *TunnelSession) {
	p.sessions.Store(session.AgentID, session)
	log.Printf("🔌 [Tunnel Pool] Node Connected & Live: AgentID=%s (Total Live Nodes: %d)", session.AgentID, p.Count())
	p.NotifyChange()
}

// UnregisterSession removes a disconnected node
func (p *TunnelPool) UnregisterSession(agentID string) {
	p.sessions.Delete(agentID)
	log.Printf("🔌 [Tunnel Pool] Node Disconnected: AgentID=%s", agentID)
	p.NotifyChange()
}

// DisconnectNode terminates the socket and unregisters the node (Killswitch)
func (p *TunnelPool) DisconnectNode(agentID string) error {
	val, ok := p.sessions.Load(agentID)
	if !ok {
		return errors.New("node_not_found_in_live_pool")
	}
	session := val.(*TunnelSession)
	session.Conn.Close()
	p.sessions.Delete(agentID)
	log.Printf("🛑 [Killswitch] Quarantined & Disconnected Node: AgentID=%s", agentID)
	p.NotifyChange()
	return nil
}

// GetSession retrieves an active WebSocket stream by agentID
func (p *TunnelPool) GetSession(agentID string) (*TunnelSession, error) {
	val, ok := p.sessions.Load(agentID)
	if !ok {
		return nil, errors.New("provider_node_is_offline_or_not_connected")
	}
	return val.(*TunnelSession), nil
}

// Count returns the number of currently connected hardware nodes
func (p *TunnelPool) Count() int {
	count := 0
	p.sessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetTelemetry aggregates in-memory live pool statistics in <100 microseconds
func (p *TunnelPool) GetTelemetry() PoolTelemetryResponse {
	domains := []string{
		"MATH_SOLVER",
		"CODE_GENERATOR",
		"OCR_DOCUMENT_PARSER",
		"GENERAL_LLM_INFERENCE",
		"NLP_TRANSLATOR",
		"WEB_SCRAPER",
		"CUSTOM_MICROSERVICE",
	}

	domainMap := make(map[string]DomainPoolStats)
	for _, d := range domains {
		domainMap[d] = DomainPoolStats{
			DomainName:     d,
			OnlineNodes:    0,
			TotalCapacity:  0,
			ActiveInFlight: 0,
			AvailableSlots: 0,
			UtilizationPct: 0.0,
		}
	}

	var activeNodes []NodeTelemetryItem
	var totalCapacity int32 = 0
	var totalInFlight int32 = 0

	p.sessions.Range(func(key, value interface{}) bool {
		session := value.(*TunnelSession)
		inFlight := session.InFlight.Load()
		capLimit := session.MaxConcurrency
		if capLimit <= 0 {
			capLimit = 1
		}
		avail := capLimit - inFlight
		if avail < 0 {
			avail = 0
		}

		totalCapacity += capLimit
		totalInFlight += inFlight

		uptime := int64(time.Since(session.ConnectedAt).Seconds())
		status := "IDLE"
		if inFlight >= capLimit {
			status = "HIGH_LOAD"
		} else if inFlight > 0 {
			status = "BUSY"
		}

		nodeName := session.NodeName
		if nodeName == "" {
			nodeName = fmt.Sprintf("Node-%s", session.AgentID[:8])
		}

		item := NodeTelemetryItem{
			AgentID:        session.AgentID,
			OwnerID:        session.OwnerID,
			NodeName:       nodeName,
			AgentType:      session.AgentType,
			Model:          session.Model,
			MaxConcurrency: capLimit,
			InFlight:       inFlight,
			AvailableSlots: avail,
			ConnectedAt:    session.ConnectedAt.Format("15:04:05"),
			UptimeSeconds:  uptime,
			Status:         status,
		}
		activeNodes = append(activeNodes, item)

		// Aggregate into domain
		dType := session.AgentType
		if dType == "" {
			dType = "GENERAL_LLM_INFERENCE"
		}
		stats, exists := domainMap[dType]
		if !exists {
			stats = DomainPoolStats{DomainName: dType}
		}
		stats.OnlineNodes++
		stats.TotalCapacity += capLimit
		stats.MaxCapacity = stats.TotalCapacity
		stats.ActiveInFlight += inFlight
		stats.InFlight = stats.ActiveInFlight
		stats.AvailableSlots += avail
		if stats.TotalCapacity > 0 {
			stats.UtilizationPct = float64(stats.ActiveInFlight) / float64(stats.TotalCapacity) * 100.0
		}
		domainMap[dType] = stats

		return true
	})

	var globalUtil float64 = 0.0
	if totalCapacity > 0 {
		globalUtil = (float64(totalInFlight) / float64(totalCapacity)) * 100.0
	}

	return PoolTelemetryResponse{
		Timestamp:            time.Now().Unix(),
		TotalOnlineNodes:     len(activeNodes),
		TotalNetworkCapacity: totalCapacity,
		TotalInFlightJobs:    totalInFlight,
		GlobalUtilizationPct: globalUtil,
		DomainPools:          domainMap,
		ActiveNodes:          activeNodes,
		Nodes:                activeNodes,
	}
}

// MatchBestSession finds the highest-rated, lowest-load online node directly in Hot RAM (<100 nanoseconds).
// It completely replaces disk/SQL queries on the prompt routing path.
func (p *TunnelPool) MatchBestSession(agentType string, model string) (*TunnelSession, *NodeCapability, error) {
	agentType = strings.TrimSpace(agentType)
	model = strings.TrimSpace(model)

	var bestSession *TunnelSession
	var bestCap *NodeCapability
	var bestScore float64 = -1e9

	p.sessions.Range(func(key, value interface{}) bool {
		session := value.(*TunnelSession)

		// 1. Filter by Domain Pool / AgentType if specified
		if agentType != "" && !strings.EqualFold(session.AgentType, agentType) {
			return true // continue scan
		}

		inFlight := session.InFlight.Load()
		capLimit := session.MaxConcurrency
		if capLimit <= 0 {
			capLimit = 1
		}
		availableSlots := capLimit - inFlight

		// 2. Scan in-memory capabilities for model match
		for i := range session.Capabilities {
			capItem := &session.Capabilities[i]
			if !capItem.IsActive {
				continue
			}

			// Tool / Model check
			if model != "" && !strings.EqualFold(capItem.ToolName, model) && !strings.EqualFold(session.Model, model) {
				continue
			}

			// Score: Weighted by reputation score + available concurrency headroom
			score := capItem.ReputationScore * 100.0
			if availableSlots > 0 {
				score += float64(availableSlots) * 20.0
			} else {
				score -= 500.0 // penalize saturated node
			}

			if score > bestScore {
				bestScore = score
				bestSession = session
				bestCap = capItem
			}
		}

		// 3. Fallback: If node connected without explicit DB capabilities row, synthesize fallback rate card
		if len(session.Capabilities) == 0 && (model == "" || strings.EqualFold(session.Model, model)) {
			fallbackCap := &NodeCapability{
				CapabilityID:             "cap-fallback-" + session.AgentID,
				AgentID:                  session.AgentID,
				ToolName:                 session.Model,
				InputTokenPriceMicroUSD:  2000,
				OutputTokenPriceMicroUSD: 4000,
				FixedCostPerCallMicroUSD: 0,
				ReputationScore:          5.00,
				IsActive:                 true,
			}
			score := 500.0 + float64(availableSlots)*20.0
			if score > bestScore {
				bestScore = score
				bestSession = session
				bestCap = fallbackCap
			}
		}

		return true
	})

	if bestSession == nil || bestCap == nil {
		return nil, nil, errors.New("all_gpu_nodes_for_target_are_currently_offline")
	}

	return bestSession, bestCap, nil
}

