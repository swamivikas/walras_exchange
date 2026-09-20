package gateway

import (
	"encoding/json"
	"net/http"
)

// AgentTypesResponse represents the ultra-clean discovery manifest
type AgentTypesResponse struct {
	AgentTypes []string `json:"agent_types"`
}

// HandleAgentTypes serves GET /v1/agent-types
func HandleAgentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	types := []string{
		"GENERAL_LLM_INFERENCE",
		"MATH_SOLVER",
		"CODE_GENERATOR",
		"OCR_DOCUMENT_PARSER",
		"NLP_TRANSLATOR",
		"WEB_SCRAPER",
		"CUSTOM_MICROSERVICE",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(AgentTypesResponse{
		AgentTypes: types,
	})
}
