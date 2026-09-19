package gateway

import (
	"encoding/json"
	"net/http"
	"walras-exchange/internal/db"
)

type PricingHandler struct {
	Repo *db.Repository
}

func NewPricingHandler(repo *db.Repository) *PricingHandler {
	return &PricingHandler{Repo: repo}
}

type ModelPricing struct {
	ModelID           string  `json:"model_id"`
	Name              string  `json:"name"`
	InputPricePer1M   float64 `json:"input_price_per_1m_usd"`
	OutputPricePer1M  float64 `json:"output_price_per_1m_usd"`
	ProprietaryPrice  float64 `json:"proprietary_price_per_1m_usd"`
	SavingsPercentage int     `json:"savings_percentage"`
	VRAMRecommended   string  `json:"vram_recommended"`
	Description       string  `json:"description"`
}

type TakeRateStructure struct {
	SellerTakeRatePercent float64 `json:"seller_take_rate_percent"`
	BuyerSurchargePercent float64 `json:"buyer_surcharge_percent"`
	NetPlatformSpread     float64 `json:"net_platform_spread_percent"`
	Description           string  `json:"description"`
}

type PricingResponse struct {
	Status             string            `json:"status"`
	FreeTrialGrantUSD  float64           `json:"free_trial_grant_usd"`
	TakeRate           TakeRateStructure `json:"take_rate"`
	Models             []ModelPricing    `json:"models"`
	HardwareEarnings   map[string]string `json:"hardware_monthly_earnings_estimate"`
}

func (h *PricingHandler) HandleGetPricing(w http.ResponseWriter, r *http.Request) {
	resp := PricingResponse{
		Status:            "active",
		FreeTrialGrantUSD: 50.00,
		TakeRate: TakeRateStructure{
			SellerTakeRatePercent: 1.0,
			BuyerSurchargePercent: 3.0,
			NetPlatformSpread:     4.0,
			Description:           "1% deducted from seller gross earnings; 3% added to buyer compute for SLA & sub-50ns routing. Net 4% clearinghouse spread.",
		},
		Models: []ModelPricing{
			{
				ModelID:           "deepseek-r1",
				Name:              "DeepSeek-R1 (Decentralized Cluster)",
				InputPricePer1M:   0.55,
				OutputPricePer1M:  2.19,
				ProprietaryPrice:  15.00,
				SavingsPercentage: 86,
				VRAMRecommended:   "24GB - 80GB",
				Description:       "Open-weights frontier reasoning model comparable to OpenAI o1 at a fraction of the cost.",
			},
			{
				ModelID:           "llama-3.3-70b",
				Name:              "Llama-3.3-70B-Instruct",
				InputPricePer1M:   0.35,
				OutputPricePer1M:  0.40,
				ProprietaryPrice:  3.50,
				SavingsPercentage: 89,
				VRAMRecommended:   "48GB - 80GB",
				Description:       "High-throughput general instruction model running across distributed H100 and A100 nodes.",
			},
			{
				ModelID:           "qwen-2.5-coder",
				Name:              "Qwen-2.5-Coder-32B",
				InputPricePer1M:   0.18,
				OutputPricePer1M:  0.22,
				ProprietaryPrice:  2.00,
				SavingsPercentage: 90,
				VRAMRecommended:   "24GB (RTX 4090 / Mac Studio)",
				Description:       "State-of-the-art coding and agentic tool invocation model with ultra-fast latency.",
			},
		},
		HardwareEarnings: map[string]string{
			"NVIDIA RTX 4090 (24GB)":     "$160 - $220 / month",
			"NVIDIA RTX 3090 (24GB)":     "$110 - $150 / month",
			"NVIDIA H100 SXM5 (80GB)":    "$1,200 - $1,650 / month",
			"Apple M2/M3 Ultra (128GB)": "$85 - $130 / month",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
