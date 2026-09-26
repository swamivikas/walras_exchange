package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "walras:walras@tcp(127.0.0.1:3306)/walras_exchange")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	keys := []struct {
		name string
		key  string
	}{
		{"LLaVA OCR", "walras_live_org_99ea14fd_78b530b885cb52d2f12ad4264d44f5e9"},
		{"Code Gen", "walras_live_org_99ea14fd_0abd7e64c36d2d7278acd83ceb413e40"},
	}

	for _, k := range keys {
		hasher := sha256.New()
		hasher.Write([]byte(k.key))
		keyHash := hex.EncodeToString(hasher.Sum(nil))

		var id, name, role, aType, status string
		err = db.QueryRow("SELECT id, name, agent_role, agent_type, status FROM agents WHERE api_key_hash = ?", keyHash).Scan(&id, &name, &role, &aType, &status)
		if err != nil {
			fmt.Printf("❌ %s (%s...): NOT found in DB (%v)\n", k.name, k.key[:20], err)
		} else {
			fmt.Printf("✅ %s: Found in DB! ID: %s | Name: %s | Role: %s | Type: %s | Status: %s\n", k.name, id[:8], name, role, aType, status)
		}
	}
}
