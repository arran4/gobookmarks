package gobookmarks

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

func OpenDB(c *Configuration) (*sql.DB, error) {
	if c.DBConnectionProvider == "" {
		return nil, NewSystemError("Database error", fmt.Errorf("db provider not configured"))
	}

	db, err := sql.Open(c.DBConnectionProvider, c.DBConnectionString)
	if err != nil {
		return nil, NewSystemError("Database error", fmt.Errorf("failed to open db: %w", err))
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, NewSystemError("Database error", err)
	}

	if err := ensureSQLSchema(db, c.DBConnectionProvider); err != nil {
		db.Close()
		return nil, NewSystemError("Database error", fmt.Errorf("failed to ensure schema: %w", err))
	}
	return db, nil
}

func ensureSQLSchema(db *sql.DB, provider string) error {
	switch strings.ToLower(provider) {
	case "mysql":
	case "sqlite3":
	default:
		return fmt.Errorf("unsupported connection provider, current supported: mysql, sqlite3; you used %s", provider)
	}

	schemaFile := "sql/schema." + strings.ToLower(provider) + ".sql"
	sqlSchema, err := sqlSchemas.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to find sql schema %s: %w", schemaFile, err)
	}

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS meta (version INTEGER)"); err != nil {
		return fmt.Errorf("failed to create meta table: %v", err)
	}

	var ver int
	row := db.QueryRow("SELECT version FROM meta LIMIT 1")
	switch err := row.Scan(&ver); {
	case err == sql.ErrNoRows:
		if _, err := db.Exec(string(sqlSchema)); err != nil {
			return fmt.Errorf("failed to import schema: %w", err)
		}
		if _, err := db.Exec("INSERT INTO meta(version) VALUES(?)", sqlSchemaVersion); err != nil {
			return fmt.Errorf("failed to set schema version: %w", err)
		}
		ver = sqlSchemaVersion
	case err != nil:
		return fmt.Errorf("failed to query schema version: %w", err)
	}

	if ver != sqlSchemaVersion {
		return fmt.Errorf("unsupported schema version %d", ver)
	}
	return nil
}
