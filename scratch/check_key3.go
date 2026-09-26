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

	key3 := "walras_live_org_99ea14fd_986a30eb5c55542b10f187919e72ec43"
	hasher := sha256.New()
	hasher.Write([]byte(key3))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	var id, name, role, status string
	err = db.QueryRow("SELECT id, name, agent_role, status FROM agents WHERE api_key_hash = ?", keyHash).Scan(&id, &name, &role, &status)
	if err != nil {
		fmt.Printf("Lookup failed for hash %s: %v\n", keyHash[:16], err)

		// Let's check agent f143f82d
		var dbHash string
		db.QueryRow("SELECT id, name, api_key_hash FROM agents WHERE id LIKE 'f143f82d%'").Scan(&id, &name, &dbHash)
		fmt.Printf("DB agent %s has hash: %s\nComputed hash:       %s\n", name, dbHash[:16], keyHash[:16])
	} else {
		fmt.Printf("Found agent in DB! ID: %s, Name: %s, Role: %s, Status: %s\n", id, name, role, status)
	}
}
