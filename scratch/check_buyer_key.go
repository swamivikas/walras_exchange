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

	key := "walras_live_org_5987bfec_e2ddbec4b678f307a6de07e39f0278d3"
	hasher := sha256.New()
	hasher.Write([]byte(key))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	var id, name, role, status string
	err = db.QueryRow("SELECT id, name, agent_role, status FROM agents WHERE api_key_hash = ?", keyHash).Scan(&id, &name, &role, &status)
	if err != nil {
		fmt.Printf("Buyer key not found (%v). Looking for any active BUYER or HYBRID agents:\n", err)
		rows, err2 := db.Query("SELECT id, name, agent_role, api_key_prefix, status FROM agents WHERE agent_role IN ('BUYER', 'HYBRID') LIMIT 5")
		if err2 == nil {
			defer rows.Close()
			for rows.Next() {
				var bID, bName, bRole, bPrefix, bStatus string
				rows.Scan(&bID, &bName, &bRole, &bPrefix, &bStatus)
				fmt.Printf(" - ID: %s | Name: %s | Role: %s | Prefix: %s | Status: %s\n", bID[:8], bName, bRole, bPrefix, bStatus)
			}
		}
	} else {
		fmt.Printf("Found buyer key! ID: %s, Name: %s, Role: %s, Status: %s\n", id, name, role, status)
	}
}
