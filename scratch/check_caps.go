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
		SELECT a.id, a.name, a.agent_type, COALESCE(c.tool_name, 'NO_CAP')
		FROM agents a
		LEFT JOIN capabilities c ON a.id = c.agent_id
		WHERE a.status = 'ACTIVE'
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Agents and their registered Capabilities in DB:")
	for rows.Next() {
		var id, name, aType, toolName string
		rows.Scan(&id, &name, &aType, &toolName)
		fmt.Printf("• %s (%s) -> Type: %s | Tool: %s\n", id[:8], name, aType, toolName)
	}
}
