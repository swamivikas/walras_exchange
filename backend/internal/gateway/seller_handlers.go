package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"walras-exchange/internal/db"
	"walras-exchange/pkg/protocol"
)

type SellerHandler struct {
	Repo *db.Repository
}

func NewSellerHandler(repo *db.Repository) *SellerHandler {
	return &SellerHandler{Repo: repo}
}

// HandleAddSellerAgent processes POST /v1/seller/agents?owner_id=...
func (h *SellerHandler) HandleAddSellerAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 1. Get owner_id from Query Param or Header
	ownerID := r.URL.Query().Get("owner_id")
	if ownerID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing_owner_id_query_param"})
		return
	}

	// 2. Decode incoming JSON payload
	var req protocol.AddSellerAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json_payload"})
		return
	}

	// 3. Validation: Must have a node name, tool name, and at least one pricing metric >= 0
	req.NodeName = strings.TrimSpace(req.NodeName)
	req.ToolName = strings.TrimSpace(req.ToolName)
	totalPricing := req.InputTokenPriceMicroUSD + req.OutputTokenPriceMicroUSD + req.FixedCostPerCallMicroUSD
	if req.NodeName == "" || req.ToolName == "" || totalPricing <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "validation_failed",
			"details": "node_name, tool_name, and pricing parameters are required",
		})
		return
	}

	// 4. Provision Seller Agent & Capability via Database Repository
	resp, err := h.Repo.AddSellerAgent(ownerID, &req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(err.Error(), "permission_denied") {
			w.WriteHeader(http.StatusForbidden)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 5. Return 201 Created Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}
