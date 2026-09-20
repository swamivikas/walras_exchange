package vault

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// CachedAgent represents an active agent living in Go Hot RAM (<50ns access)
type CachedAgent struct {
	AgentID        string
	OwnerID        string
	NodeName       string
	Role           string       // "BUYER", "SELLER", "HYBRID"
	AgentType      string       // "MATH_SOLVER", "OCR_DOCUMENT_PARSER", "GENERAL_LLM_INFERENCE", etc.
	Status         string       // "ACTIVE", "SUSPENDED"
	ApprovalStatus string       // "PENDING_REVIEW", "APPROVED", "REJECTED"
	
	CreditLimit    int64        // In microUSD (for buyers)
	CurrentUsage   atomic.Int64 // Sub-50ns thread-safe usage counter
	Balance        atomic.Int64 // Accrued seller earnings
	
	// Concurrency Controls
	MaxConcurrency      int32        // e.g. 2 parallel requests
	ActiveRequestsCount atomic.Int32 // Current in-flight requests

	LastActiveAt time.Time
}

// CanAcceptJob checks if the node has spare concurrency (<50ns)
func (a *CachedAgent) CanAcceptJob() bool {
	return a.ActiveRequestsCount.Load() < a.MaxConcurrency
}

// AcquireSlot reserves an execution slot
func (a *CachedAgent) AcquireSlot() bool {
	for {
		curr := a.ActiveRequestsCount.Load()
		if curr >= a.MaxConcurrency {
			return false
		}
		if a.ActiveRequestsCount.CompareAndSwap(curr, curr+1) {
			return true
		}
	}
}

// ReleaseSlot frees the execution slot when job completes
func (a *CachedAgent) ReleaseSlot() {
	a.ActiveRequestsCount.Add(-1)
}

// Vault holds the global in-memory maps
type Vault struct {
	authCache  sync.Map // apiKeyHash -> *CachedAgent
	agentCache sync.Map // agentID -> *CachedAgent
}

var GlobalVault = &Vault{}

func (v *Vault) RegisterAgentInRAM(apiKeyHash string, agent *CachedAgent) {
	agent.LastActiveAt = time.Now()
	v.authCache.Store(apiKeyHash, agent)
	v.agentCache.Store(agent.AgentID, agent)
}

func (v *Vault) GetAgentByAPIKeyHash(apiKeyHash string) (*CachedAgent, error) {
	val, ok := v.authCache.Load(apiKeyHash)
	if !ok {
		return nil, errors.New("agent_not_found_in_ram")
	}
	agent := val.(*CachedAgent)
	if agent.Status != "ACTIVE" {
		return nil, errors.New("agent_is_not_active")
	}
	return agent, nil
}

func (v *Vault) GetAgentByID(agentID string) (*CachedAgent, error) {
	val, ok := v.agentCache.Load(agentID)
	if !ok {
		return nil, errors.New("agent_not_found_in_ram")
	}
	return val.(*CachedAgent), nil
}
