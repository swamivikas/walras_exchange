package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"walras-exchange/internal/db"
	"walras-exchange/internal/gateway"
)

// withCORS wraps an http.Handler with standard cross-origin headers
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Println("=================================================================")
	log.Println("🚀 WALRAS EXCHANGE: High-Performance Compute & MCP Clearinghouse")
	log.Println("=================================================================")

	// 1. MySQL Connection String (from our docker-compose.yml)
	dsn := "walras:walras@tcp(127.0.0.1:3306)/walras_exchange?parseTime=true"

	log.Println("Connecting to MySQL Database...")
	repo, err := db.NewRepository(dsn)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer repo.DB.Close()
	log.Println("✅ Database Connected & Connection Pool Ready!")

	// 2. Initialize Handlers
	authHandler := gateway.NewAuthHandler(repo)
	sellerHandler := gateway.NewSellerHandler(repo)
	tunnelHandler := gateway.NewTunnelHandler(repo)
	chatHandler := gateway.NewChatHandler(repo)
	pricingHandler := gateway.NewPricingHandler(repo)

	// 3. Register HTTP Routes using Go 1.22 Mux
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/auth/signup", authHandler.HandleSignup)
	mux.HandleFunc("POST /v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("GET /v1/user/agents", authHandler.HandleGetUserAgents)
	mux.HandleFunc("POST /v1/seller/agents", sellerHandler.HandleAddSellerAgent)
	mux.HandleFunc("GET /v1/provider/tunnel", tunnelHandler.HandleProviderTunnel)
	mux.HandleFunc("POST /v1/chat/completions", chatHandler.HandleChatCompletions)
	mux.HandleFunc("GET /v1/agent-types", gateway.HandleAgentTypes)
	mux.HandleFunc("GET /v1/pricing", pricingHandler.HandleGetPricing)
	
	// Health Check Route
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "engine": "walras-gateway-v1"}`))
	})

	// 4. Serve the Frontend directly from ./frontend (or ../frontend)
	frontendDir := "./frontend"
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		if _, err := os.Stat("../frontend"); err == nil {
			frontendDir = "../frontend"
		}
	}

	// Ensure assets/images directory and copy assets if present in artifacts
	syncAssets(frontendDir)

	fs := http.FileServer(http.Dir(frontendDir))
	mux.Handle("GET /", fs)

	// 5. Dedicated Internal Network Operations War Room (NOC) on Port :8090
	opsHandler := gateway.NewOpsHandler(repo)
	opsMux := http.NewServeMux()
	opsMux.HandleFunc("GET /v1/ops/telemetry", opsHandler.HandleTelemetry)
	opsMux.HandleFunc("GET /v1/ops/settlement-tape", opsHandler.HandleSettlementTape)
	opsMux.HandleFunc("GET /v1/ops/stream", opsHandler.HandleStream)
	opsMux.HandleFunc("POST /v1/ops/killswitch", opsHandler.HandleKillswitch)
	opsMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "service": "walras-warroom-ops", "port": 8090}`))
	})

	warroomDir := "./warroom"
	if _, err := os.Stat(warroomDir); os.IsNotExist(err) {
		if _, err := os.Stat("../warroom"); err == nil {
			warroomDir = "../warroom"
		}
	}
	opsMux.Handle("GET /", http.FileServer(http.Dir(warroomDir)))

	opsServer := &http.Server{
		Addr:         ":8090",
		Handler:      withCORS(opsMux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // Zero timeout required for indefinite SSE persistent streaming
	}

	go func() {
		log.Println("🚨 [INTERNAL WAR ROOM] Dedicated NOC Command Center live on http://localhost:8090")
		if err := opsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Internal War Room Server error: %v", err)
		}
	}()

	// 6. Start Public Gateway Server on :8080 with CORS enabled
	server := &http.Server{
		Addr:         ":8080",
		Handler:      withCORS(mux),
		ReadTimeout:  0, // Zero timeout required for long-lived WebSocket tunnels and LLM inference
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("⚡ Gateway Engine listening on http://localhost:8080")
	log.Println("🖥️  Public Client Dashboard live on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Server error: %v", err)
	}
}

func syncAssets(frontendDir string) {
	imagesDir := frontendDir + "/assets/images"
	os.MkdirAll(imagesDir, 0755)

	artifactMap := map[string]string{
		"C:/Users/swami/.gemini/antigravity-ide/brain/757737a3-f14b-4b2e-bfa8-cdafb30a4dbf/hero_compute_cluster_1789818662725.jpg": imagesDir + "/hero_compute_cluster.jpg",
		"C:/Users/swami/.gemini/antigravity-ide/brain/757737a3-f14b-4b2e-bfa8-cdafb30a4dbf/global_edge_mesh_1789818699308.jpg":     imagesDir + "/global_edge_mesh.jpg",
		"C:/Users/swami/.gemini/antigravity-ide/brain/757737a3-f14b-4b2e-bfa8-cdafb30a4dbf/atomic_settlement_chip_1789818769613.jpg": imagesDir + "/atomic_settlement_chip.jpg",
	}

	for src, dst := range artifactMap {
		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err == nil {
				os.WriteFile(dst, data, 0644)
			}
		}
	}
}

