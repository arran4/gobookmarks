package gobookmarks

import (
	"context"
	"crypto/sha1"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

type SQLProvider struct{}

const sqlSchemaVersion = 1

//go:embed sql/schema.sql
var sqlSchema string

func init() { RegisterProvider(SQLProvider{}) }

func (SQLProvider) Name() string                                                     { return "sql" }
func (SQLProvider) DefaultServer() string                                            { return "" }
func (SQLProvider) Config(clientID, clientSecret, redirectURL string) *oauth2.Config { return nil }
func (SQLProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return nil, errors.New("not implemented")
}

func openDB() (*sql.DB, error) {
	if DBConnectionProvider == "" {
		return nil, errors.New("db provider not configured")
	}
	db, err := sql.Open(DBConnectionProvider, DBConnectionString)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if err := ensureSQLSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func ensureSQLSchema(db *sql.DB) error {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS meta (version INTEGER)"); err != nil {
		return err
	}
	var ver int
	err := db.QueryRow("SELECT version FROM meta LIMIT 1").Scan(&ver)
	if err == sql.ErrNoRows {
		if _, err := db.Exec(sqlSchema); err != nil {
			return err
		}
		_, err = db.Exec("INSERT INTO meta(version) VALUES(?)", sqlSchemaVersion)
		return err
	}
	if err != nil {
		return err
	}
	if ver != sqlSchemaVersion {
		return fmt.Errorf("unsupported schema version %d", ver)
	}
	return nil
}

func (SQLProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, "SELECT name FROM tags WHERE user=? ORDER BY name", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []*Tag
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		tags = append(tags, &Tag{Name: n})
	}
	return tags, rows.Err()
}

func (SQLProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, "SELECT name FROM branches WHERE user=? ORDER BY name", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var branches []*Branch
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		branches = append(branches, &Branch{Name: n})
	}
	if len(branches) == 0 {
		branches = append(branches, &Branch{Name: "main"})
	}
	return branches, rows.Err()
}

func (SQLProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, "SELECT sha, message, date FROM history WHERE user=? ORDER BY id DESC", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var commits []*Commit
	for rows.Next() {
		var sha, msg string
		var t time.Time
		if err := rows.Scan(&sha, &msg, &t); err != nil {
			return nil, err
		}
		commits = append(commits, &Commit{
			SHA:            sha,
			Message:        msg,
			CommitterName:  "gobookmarks",
			CommitterEmail: "gobookmarks@arran.net.au",
			CommitterDate:  t,
		})
	}
	return commits, rows.Err()
}

func (SQLProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	db, err := openDB()
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	if ref == "" {
		ref = "refs/heads/main"
	}

	if strings.HasPrefix(ref, "refs/heads/") {
		ref = strings.TrimPrefix(ref, "refs/heads/")
	} else if strings.HasPrefix(ref, "heads/") {
		ref = strings.TrimPrefix(ref, "heads/")
	}

	var sha string
	var text string

	switch {
	case ref == "main" || strings.Contains(ref, "/") == false:
		// treat as branch
		err = db.QueryRowContext(ctx, "SELECT sha FROM branches WHERE user=? AND name=?", user, ref).Scan(&sha)
		if err == sql.ErrNoRows {
			// fall back to latest
			err = db.QueryRowContext(ctx, "SELECT list FROM bookmarks WHERE user=?", user).Scan(&text)
			if err == sql.ErrNoRows {
				return "", "", nil
			}
			return text, "", err
		} else if err != nil {
			return "", "", err
		}
	default:
		sha = ref
	}

	if sha != "" {
		err = db.QueryRowContext(ctx, "SELECT text FROM history WHERE user=? AND sha=?", user, sha).Scan(&text)
		if err == sql.ErrNoRows {
			return "", "", nil
		}
		if err != nil {
			return "", "", err
		}
	}

	return text, sha, nil
}

func (SQLProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	if branch == "" {
		branch = "main"
	}
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var curSha sql.NullString
	err = tx.QueryRowContext(ctx, "SELECT sha FROM branches WHERE user=? AND name=?", user, branch).Scan(&curSha)
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}
	if expectSHA != "" && curSha.Valid && curSha.String != expectSHA {
		tx.Rollback()
		return errors.New("sha mismatch")
	}

	sum := sha1.Sum([]byte(time.Now().String() + text))
	sha := hex.EncodeToString(sum[:])
	if _, err := tx.ExecContext(ctx, "INSERT INTO history(user, sha, message, text, date) VALUES(?,?,?,?,?)", user, sha, "update", text, time.Now()); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE bookmarks SET list=? WHERE user=?", text, user); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO branches(user, name, sha) VALUES(?,?,?) ON CONFLICT(user,name) DO UPDATE SET sha=excluded.sha", user, branch, sha); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (SQLProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	if branch == "" {
		branch = "main"
	}
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO bookmarks(user, list) VALUES(?, '')", user); err != nil {
		tx.Rollback()
		return err
	}
	sum := sha1.Sum([]byte(time.Now().String() + text))
	sha := hex.EncodeToString(sum[:])
	if _, err := tx.ExecContext(ctx, "INSERT INTO history(user, sha, message, text, date) VALUES(?,?,?,?,?)", user, sha, "create", text, time.Now()); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE bookmarks SET list=? WHERE user=?", text, user); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO branches(user, name, sha) VALUES(?,?,?) ON CONFLICT(user,name) DO UPDATE SET sha=excluded.sha", user, branch, sha); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (SQLProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO bookmarks(user, list) VALUES(?, '')", user); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO branches(user, name, sha) VALUES(?, 'main', '')", user); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (SQLProvider) RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error) {
	db, err := openDB()
	if err != nil {
		return false, err
	}
	defer db.Close()
	var c int
	err = db.QueryRowContext(ctx, "SELECT COUNT(1) FROM bookmarks WHERE user=?", user).Scan(&c)
	return c > 0, err
}

func (SQLProvider) CreateUser(ctx context.Context, user, password string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	var c int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM passwords WHERE user=?", user).Scan(&c); err != nil {
		return err
	}
	if c > 0 {
		return ErrUserExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "INSERT INTO passwords(user, hash) VALUES(?, ?)", user, hash)
	return err
}

func (SQLProvider) SetPassword(ctx context.Context, user, password string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := db.ExecContext(ctx, "UPDATE passwords SET hash=? WHERE user=?", hash, user)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (SQLProvider) CheckPassword(ctx context.Context, user, password string) (bool, error) {
	db, err := openDB()
	if err != nil {
		return false, err
	}
	defer db.Close()
	var hash []byte
	err = db.QueryRowContext(ctx, "SELECT hash FROM passwords WHERE user=?", user).Scan(&hash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil, nil
}
