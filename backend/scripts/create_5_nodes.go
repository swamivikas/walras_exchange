package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/go-sql-driver/mysql"
	"walras-exchange/internal/auth"
)

func main() {
	db, err := sql.Open("mysql", "walras:walras@tcp(127.0.0.1:3306)/walras_exchange")
	if err != nil {
		log.Fatalf("MySQL connect failed: %v", err)
	}
	defer db.Close()

	// Find the seller owner
	var ownerID string
	err = db.QueryRow("SELECT id FROM owners WHERE id LIKE '29c249b6%' LIMIT 1").Scan(&ownerID)
	if err != nil {
		// Fallback to any seller owner
		err = db.QueryRow("SELECT id FROM owners WHERE user_type IN ('SELLER_OWNER', 'HYBRID') LIMIT 1").Scan(&ownerID)
		if err != nil {
			log.Fatalf("Failed to find seller owner: %v", err)
		}
	}

	fmt.Printf("Using Owner ID: %s\n\n", ownerID)

	for i := 1; i <= 5; i++ {
		nodeName := fmt.Sprintf("colab-gpu-node-%d", i)
		agentID := uuid.New().String()
		capID := uuid.New().String()

		apiKey, err := auth.GenerateAPIKey(ownerID)
		if err != nil {
			log.Fatalf("Failed to generate API key: %v", err)
		}

		hasher := sha256.New()
		hasher.Write([]byte(apiKey))
		keyHash := hex.EncodeToString(hasher.Sum(nil))
		keyPrefix := apiKey[:20]

		// Insert Agent
		_, err = db.Exec(`
			INSERT INTO agents (
				id, owner_id, name, agent_type, max_concurrency, vram_gb, agent_role,
				api_key_hash, api_key_prefix, balance_micro_usd, status, approval_status
			) VALUES (?, ?, ?, 'MATH_SOLVER', 4, 16, 'SELLER', ?, ?, 0, 'ACTIVE', 'APPROVED')
		`, agentID, ownerID, nodeName, keyHash, keyPrefix)
		if err != nil {
			log.Fatalf("Insert agent %d failed: %v", i, err)
		}

		// Insert Capability
		_, err = db.Exec(`
			INSERT INTO capabilities (
				id, agent_id, tool_name, description, input_token_price_micro_usd,
				output_token_price_micro_usd, fixed_cost_per_call_micro_usd, reputation_score, is_active
			) VALUES (?, ?, 'deepseek-r1:1.5b', 'High-throughput GPU calculus & math reasoning', 550, 2190, 0, 5.00, 1)
		`, capID, agentID)
		if err != nil {
			log.Fatalf("Insert capability %d failed: %v", i, err)
		}

		fmt.Printf("✅ NODE %d CREATED:\n", i)
		fmt.Printf("   Name     : %s\n", nodeName)
		fmt.Printf("   Agent ID : %s\n", agentID)
		fmt.Printf("   API Key  : %s\n\n", apiKey)
	}
}
