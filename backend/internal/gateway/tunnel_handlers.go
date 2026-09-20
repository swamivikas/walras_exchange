package gateway

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"walras-exchange/internal/db"
	"walras-exchange/internal/tunnel"
	"walras-exchange/internal/vault"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 64,
	WriteBufferSize: 1024 * 64,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type TunnelHandler struct {
	Repo *db.Repository
}

func NewTunnelHandler(repo *db.Repository) *TunnelHandler {
	return &TunnelHandler{Repo: repo}
}

func (h *TunnelHandler) HandleProviderTunnel(w http.ResponseWriter, r *http.Request) {
	// 1. Extract API Key
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || token == authHeader {
		token = r.URL.Query().Get("token")
	}

	if token == "" {
		http.Error(w, "missing_token", http.StatusUnauthorized)
		return
	}

	// 2. Hash Token with SHA-256
	hasher := sha256.New()
	hasher.Write([]byte(token))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	// 3. Fast In-Memory Check or Cold DB Hydration
	agent, err := vault.GlobalVault.GetAgentByAPIKeyHash(keyHash)
	if err != nil {
		var dbAgent vault.CachedAgent
		query := `
			SELECT id, owner_id, name, agent_role, agent_type, max_concurrency, status, approval_status 
			FROM agents 
			WHERE api_key_hash = ? AND status = 'ACTIVE' 
			LIMIT 1
		`
		err = h.Repo.DB.QueryRow(query, keyHash).Scan(
			&dbAgent.AgentID,
			&dbAgent.OwnerID,
			&dbAgent.NodeName,
			&dbAgent.Role,
			&dbAgent.AgentType,
			&dbAgent.MaxConcurrency,
			&dbAgent.Status,
			&dbAgent.ApprovalStatus,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "invalid_or_unauthorized_token", http.StatusUnauthorized)
			} else {
				http.Error(w, "database_error", http.StatusInternalServerError)
			}
			return
		}
		vault.GlobalVault.RegisterAgentInRAM(keyHash, &dbAgent)
		agent = &dbAgent
	}

	if agent.Role != "SELLER" && agent.Role != "HYBRID" {
		http.Error(w, "only_seller_nodes_can_connect_tunnel", http.StatusForbidden)
		return
	}

	// 4. Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade websocket: %v", err)
		return
	}

	session := &tunnel.TunnelSession{
		AgentID:        agent.AgentID,
		OwnerID:        agent.OwnerID,
		NodeName:       agent.NodeName,
		AgentType:      agent.AgentType,
		MaxConcurrency: agent.MaxConcurrency,
		Conn:           conn,
		ConnectedAt:    time.Now(),
	}
	tunnel.GlobalTunnelPool.RegisterSession(session)
	defer tunnel.GlobalTunnelPool.UnregisterSession(agent.AgentID)
	defer conn.Close()

	// 5. Listen for incoming responses from Python daemon & route them
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			log.Printf("🔌 Connection closed by provider %s: %v", agent.AgentID, err)
			break
		}

		// Decode request_id from daemon response and route to waiting HTTP handler
		var respMap map[string]interface{}
		if err := json.Unmarshal(payload, &respMap); err == nil {
			if reqID, ok := respMap["request_id"].(string); ok {
				if chVal, found := session.PendingJobs.Load(reqID); found {
					ch := chVal.(chan []byte)
					ch <- payload
					session.PendingJobs.Delete(reqID)
				}
			}
		}
	}
}
