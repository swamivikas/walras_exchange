package tunnel

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// TunnelSession represents a live, persistent WebSocket stream from a seller node
type TunnelSession struct {
	AgentID       string
	OwnerID       string
	NodeName      string
	Conn          *websocket.Conn
	ConnectedAt   time.Time
	mu            sync.Mutex // Protects concurrent writes down the same socket
	PendingJobs   sync.Map   // requestID -> chan []byte (for response multiplexing)
}

// WriteJSON sends a thread-safe payload down the open WebSocket stream
func (s *TunnelSession) WriteJSON(v interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteJSON(v)
}

// TunnelPool manages all active hardware connections globally
type TunnelPool struct {
	sessions sync.Map // agentID (string) -> *TunnelSession
}

// GlobalTunnelPool is the singleton tunnel registry
var GlobalTunnelPool = &TunnelPool{}

// RegisterSession adds a newly connected hardware node to the live pool
func (p *TunnelPool) RegisterSession(session *TunnelSession) {
	p.sessions.Store(session.AgentID, session)
	log.Printf("🔌 [Tunnel Pool] Node Connected & Live: AgentID=%s (Total Live Nodes: %d)", session.AgentID, p.Count())
}

// UnregisterSession removes a disconnected node
func (p *TunnelPool) UnregisterSession(agentID string) {
	p.sessions.Delete(agentID)
	log.Printf("🔌 [Tunnel Pool] Node Disconnected: AgentID=%s", agentID)
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
