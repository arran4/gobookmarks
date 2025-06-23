package main

import (
	"database/sql"
	_ "embed"
	"flag"
	"log"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed legacy_migrate.sql
var migrateSQL string

func main() {
	provider := flag.String("provider", "", "SQL driver name")
	conn := flag.String("conn", "", "connection string")
	flag.Parse()
	if *provider == "" || *conn == "" {
		flag.Usage()
		return
	}

	db, err := sql.Open(*provider, *conn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	if _, err := db.Exec(migrateSQL); err != nil {
		log.Fatalf("apply migration: %v", err)
	}

	log.Println("migration complete")
}
