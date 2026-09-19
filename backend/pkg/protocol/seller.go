package protocol

// AddSellerAgentRequest represents the payload to register a new GPU node / tool
type AddSellerAgentRequest struct {
	// Node Hardware Profile
	NodeName       string `json:"node_name"`       // e.g. "Austin-RTX4090-Rig-1"
	HardwareType   string `json:"hardware_type"`   // "LOCAL_GPU", "APPLE_SILICON", "CLOUD_VM"
	VRAMGigabytes  int    `json:"vram_gb"`         // e.g. 24
	MaxConcurrency int    `json:"max_concurrency"` // e.g. 2

	// Standardized Dropdown Agent Type
	AgentType string `json:"agent_type"` // "MATH_SOLVER", "OCR_DOCUMENT_PARSER", "GENERAL_LLM_INFERENCE", etc.

	// Capability / Tool Manifest
	ToolName        string `json:"tool_name"`        // e.g. "deepseek-r1"
	Description     string `json:"description"`      // "Local DeepSeek R1 32B model"
	InputJSONSchema string `json:"json_schema"`       // Parameter JSON Schema

	// Unified Pricing (microUSD)
	InputTokenPriceMicroUSD  int64 `json:"input_token_price_micro_usd"`  // Price per 1M input tokens
	OutputTokenPriceMicroUSD int64 `json:"output_token_price_micro_usd"` // Price per 1M output tokens
	FixedCostPerCallMicroUSD int64 `json:"fixed_cost_per_call_micro_usd"` // Fixed execution fee (for non-LLMs)
}

// AddSellerAgentResponse is returned upon successful registration
type AddSellerAgentResponse struct {
	OwnerID        string `json:"owner_id"`
	AgentID        string `json:"agent_id"`
	CapabilityID   string `json:"capability_id"`
	ToolName       string `json:"tool_name"`
	AgentType      string `json:"agent_type"`
	ApprovalStatus string `json:"approval_status"` // "PENDING_REVIEW"
	
	// Dedicated Node API Key for the WebSocket Tunnel Daemon
	NodeAPIKey     string `json:"node_api_key"`
	NodeKeyPrefix  string `json:"node_key_prefix"`

	Message string `json:"message"`
}
