package main

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

// schema used by the SQL provider
const schema = `CREATE TABLE IF NOT EXISTS bookmarks (
    user TEXT PRIMARY KEY,
    list BLOB
);
CREATE TABLE IF NOT EXISTS passwords (
    user TEXT PRIMARY KEY,
    hash BLOB
);
CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user TEXT,
    sha TEXT,
    message TEXT,
    text BLOB,
    date TIMESTAMP
);
CREATE TABLE IF NOT EXISTS branches (
    user TEXT,
    name TEXT,
    sha TEXT,
    PRIMARY KEY(user, name)
);
CREATE TABLE IF NOT EXISTS tags (
    user TEXT,
    name TEXT,
    sha TEXT,
    PRIMARY KEY(user, name)
);
CREATE TABLE IF NOT EXISTS meta (
    version INTEGER
);`

const schemaVersion = 1

func ensureSchema(db *sql.DB) error {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS meta (version INTEGER)"); err != nil {
		return err
	}
	var ver int
	err := db.QueryRow("SELECT version FROM meta LIMIT 1").Scan(&ver)
	if err == sql.ErrNoRows {
		if _, err := db.Exec(schema); err != nil {
			return err
		}
		_, err = db.Exec("INSERT INTO meta(version) VALUES(?)", schemaVersion)
		return err
	}
	if err != nil {
		return err
	}
	if ver != schemaVersion {
		return fmt.Errorf("unsupported schema version %d", ver)
	}
	return nil
}

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
	if err := ensureSchema(db); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	// migrate users -> passwords
	rows, err := db.Query("SELECT username, passwd FROM users")
	if err != nil {
		log.Fatalf("query users: %v", err)
	}
	for rows.Next() {
		var user string
		var pass sql.NullString
		if err := rows.Scan(&user, &pass); err != nil {
			log.Fatalf("scan user: %v", err)
		}
		if _, err := db.Exec("REPLACE INTO passwords(user, hash) VALUES(?, ?)", user, pass.String); err != nil {
			log.Fatalf("insert password: %v", err)
		}
	}
	rows.Close()

	// migrate bookmarks
	rows, err = db.Query("SELECT b.list, u.username FROM bookmarks b JOIN users u ON b.users_idusers=u.idusers")
	if err != nil {
		log.Fatalf("query bookmarks: %v", err)
	}
	now := time.Now()
	for rows.Next() {
		var list []byte
		var user string
		if err := rows.Scan(&list, &user); err != nil {
			log.Fatalf("scan bookmark: %v", err)
		}
		if _, err := db.Exec("REPLACE INTO bookmarks(user, list) VALUES(?, ?)", user, list); err != nil {
			log.Fatalf("insert bookmarks: %v", err)
		}
		sum := sha1.Sum([]byte(now.String() + string(list)))
		sha := hex.EncodeToString(sum[:])
		if _, err := db.Exec("INSERT INTO history(user, sha, message, text, date) VALUES(?,?,?,?,?)", user, sha, "import", list, now); err != nil {
			log.Fatalf("insert history: %v", err)
		}
		if _, err := db.Exec("REPLACE INTO branches(user, name, sha) VALUES(?, 'main', ?)", user, sha); err != nil {
			log.Fatalf("insert branch: %v", err)
		}
	}
	rows.Close()

	log.Println("migration complete")
}
