package main

import (
	"database/sql"
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

	rows, err := db.Query(`
		SELECT id, owner_id, name, agent_role, agent_type, api_key_prefix, status, approval_status
		FROM agents
		WHERE agent_role IN ('SELLER', 'HYBRID')
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Recent Seller Agents in DB:")
	for rows.Next() {
		var id, ownerID, name, role, aType, keyPrefix, status, appStatus string
		rows.Scan(&id, &ownerID, &name, &role, &aType, &keyPrefix, &status, &appStatus)
		fmt.Printf("ID: %s | Name: %s | Owner: %s | Role: %s | Type: %s | KeyPrefix: %s | Status: %s/%s\n",
			id[:8], name, ownerID[:8], role, aType, keyPrefix, status, appStatus)
	}
}
