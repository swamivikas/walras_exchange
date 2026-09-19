package protocol

// SignupRequest defines the incoming payload for user registration
type SignupRequest struct {
	Name     string `json:"name"`      // e.g. "Acme Corp" or "John Doe"
	Email    string `json:"email"`     // e.g. "dev@acmecorp.com"
	Password string `json:"password"`  // Plaintext password (will be hashed with bcrypt)
	UserType string `json:"user_type"` // "ENTERPRISE_BUYER", "SELLER_OWNER", or "HYBRID"
}

// DefaultAgentInfo contains details of the auto-generated Buyer Agent
type DefaultAgentInfo struct {
	AgentID        string `json:"agent_id"`
	Name           string `json:"name"`
	Role           string `json:"role"`           // "BUYER"
	AgentType      string `json:"agent_type"`     // "GENERAL_LLM_INFERENCE"
	Status         string `json:"status"`         // "ACTIVE"
	ApprovalStatus string `json:"approval_status"`// "APPROVED"
	TrialBalance   string `json:"trial_balance"`  // "$50.00 USD"
	APIKey         string `json:"api_key"`        // Shown ONCE to the user!
	APIKeyPrefix   string `json:"api_key_prefix"` // e.g. "walras_live_org_9b1...489"
}

// SignupResponse is returned to the client upon successful registration
type SignupResponse struct {
	OwnerID      string           `json:"owner_id"`
	Name         string           `json:"name"`
	Email        string           `json:"email"`
	UserType     string           `json:"user_type"`
	DefaultAgent DefaultAgentInfo `json:"default_agent"`
	Message      string           `json:"message"`
}

// LoginRequest defines incoming payload for user authentication
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserAgentItem contains complete information about an agent owned by the user
type UserAgentItem struct {
	AgentID                  string  `json:"agent_id"`
	Name                     string  `json:"name"`
	Role                     string  `json:"role"`
	AgentType                string  `json:"agent_type"`
	Status                   string  `json:"status"`
	ApprovalStatus           string  `json:"approval_status"`
	BalanceMicroUSD          int64   `json:"balance_micro_usd"`
	APIKeyPrefix             string  `json:"api_key_prefix"`
	Model                    string  `json:"model,omitempty"`
	Description              string  `json:"description,omitempty"`
	VRAMGB                   int     `json:"vram_gb,omitempty"`
	MaxConcurrency           int     `json:"max_concurrency,omitempty"`
	InputTokenPriceMicroUSD  int64   `json:"input_token_price_micro_usd,omitempty"`
	OutputTokenPriceMicroUSD int64   `json:"output_token_price_micro_usd,omitempty"`
	FixedCostPerCallMicroUSD int64   `json:"fixed_cost_per_call_micro_usd,omitempty"`
	ReputationScore          float64 `json:"reputation_score,omitempty"`
}

// LoginResponse contains owner details and their deployed agents
type LoginResponse struct {
	OwnerID             string          `json:"owner_id"`
	Name                string          `json:"name"`
	Email               string          `json:"email"`
	UserType            string          `json:"user_type"`
	CreditLimitMicroUSD int64           `json:"credit_limit_micro_usd"`
	BalanceUSD          string          `json:"balance_usd"`
	PlanType            string          `json:"plan_type"`
	Agents              []UserAgentItem `json:"agents"`
}
