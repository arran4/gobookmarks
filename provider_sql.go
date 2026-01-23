package gobookmarks

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"embed"
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

//go:embed sql/schema*.sql
var sqlSchemas embed.FS

func init() {
	RegisterProvider(SQLProvider{})
}

func (SQLProvider) Name() string                                                     { return "sql" }
func (SQLProvider) DefaultServer() string                                            { return "" }
func (SQLProvider) Config(clientID, clientSecret, redirectURL string) *oauth2.Config { return nil }
func (SQLProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return nil, errors.New("not implemented")
}

func (p SQLProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	db, err := OpenDB()
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

func (p SQLProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	db, err := OpenDB()
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

func (p SQLProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token, ref string, page, perPage int) ([]*Commit, error) {
	db, err := OpenDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT sha, message, date FROM history WHERE user=? ORDER BY id DESC"
	args := []any{user}
	if perPage > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, perPage, (page-1)*perPage)
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %v", err)
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

func (p SQLProvider) AdjacentCommits(ctx context.Context, user string, token *oauth2.Token, ref, sha string) (string, string, error) {
	db, err := OpenDB()
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	var id int
	err = db.QueryRowContext(ctx, "SELECT id FROM history WHERE user=? AND sha=?", user, sha).Scan(&id)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	var prev, next sql.NullString
	err = db.QueryRowContext(ctx, "SELECT sha FROM history WHERE user=? AND id < ? ORDER BY id DESC LIMIT 1", user, id).Scan(&prev)
	if err != nil && err != sql.ErrNoRows {
		return "", "", err
	}
	err = db.QueryRowContext(ctx, "SELECT sha FROM history WHERE user=? AND id > ? ORDER BY id ASC LIMIT 1", user, id).Scan(&next)
	if err != nil && err != sql.ErrNoRows {
		return "", "", err
	}
	return prev.String, next.String, nil
}

func (p SQLProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	db, err := OpenDB()
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

	var sha, text string
	switch {
	case ref == "main" || !strings.Contains(ref, "/"):
		err = db.QueryRowContext(ctx, "SELECT sha FROM branches WHERE user=? AND name=?", user, ref).Scan(&sha)
		if err == sql.ErrNoRows {
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

func (p SQLProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	if branch == "" {
		branch = "main"
	}
	db, err := OpenDB()
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
	newSha := hex.EncodeToString(sum[:])

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO history(user, sha, message, text, date) VALUES(?,?,?,?,?)",
		user, newSha, "update", text, time.Now(),
	); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE bookmarks SET list=? WHERE user=?", text, user,
	); err != nil {
		tx.Rollback()
		return err
	}

	// dialect-specific insert/update for branches
	switch strings.ToLower(AppConfig.DBConnectionProvider) {
	case "mysql":
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO branches(user, name, sha)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE sha = VALUES(sha)
		`, user, branch, newSha); err != nil {
			tx.Rollback()
			return err
		}
	case "sqlite3":
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO branches(user, name, sha)
			VALUES (?, ?, ?)
			ON CONFLICT(user, name) DO UPDATE SET sha = excluded.sha
		`, user, branch, newSha); err != nil {
			tx.Rollback()
			return err
		}
	default:
		tx.Rollback()
		return errors.New("unsupported connection provider")
	}

	return tx.Commit()
}

func (p SQLProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	if branch == "" {
		branch = "main"
	}
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// ensure a bookmarks row exists
	switch strings.ToLower(AppConfig.DBConnectionProvider) {
	case "mysql":
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO bookmarks(user, list) VALUES(?, '') ON DUPLICATE KEY UPDATE list=list",
			user,
		); err != nil {
			tx.Rollback()
			return err
		}
	case "sqlite3":
		if _, err := tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO bookmarks(user, list) VALUES(?, '')",
			user,
		); err != nil {
			tx.Rollback()
			return err
		}
	default:
		tx.Rollback()
		return errors.New("unsupported connection provider")
	}

	sum := sha1.Sum([]byte(time.Now().String() + text))
	newSha := hex.EncodeToString(sum[:])

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO history(user, sha, message, text, date) VALUES(?,?,?,?,?)",
		user, newSha, "create", text, time.Now(),
	); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx,
		"UPDATE bookmarks SET list=? WHERE user=?", text, user,
	); err != nil {
		tx.Rollback()
		return err
	}

	// ensure a branch pointer
	switch strings.ToLower(AppConfig.DBConnectionProvider) {
	case "mysql":
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO branches(user, name, sha)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE sha=VALUES(sha)
		`, user, branch, newSha); err != nil {
			tx.Rollback()
			return err
		}
	case "sqlite3":
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO branches(user, name, sha)
			VALUES (?, ?, ?)
			ON CONFLICT(user, name) DO UPDATE SET sha = excluded.sha
		`, user, branch, newSha); err != nil {
			tx.Rollback()
			return err
		}
	default:
		tx.Rollback()
		return errors.New("unsupported connection provider")
	}

	return tx.Commit()
}

func (p SQLProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	switch strings.ToLower(AppConfig.DBConnectionProvider) {
	case "mysql":
		// bookmarks row
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO bookmarks(user, list) VALUES(?, '') ON DUPLICATE KEY UPDATE list=list",
			user,
		); err != nil {
			tx.Rollback()
			return err
		}
		// default branch
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO branches(user, name, sha) VALUES(?, 'main', '') ON DUPLICATE KEY UPDATE sha=sha",
			user,
		); err != nil {
			tx.Rollback()
			return err
		}
	case "sqlite3":
		if _, err := tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO bookmarks(user, list) VALUES(?, '')",
			user,
		); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO branches(user, name, sha) VALUES(?, 'main', '')",
			user,
		); err != nil {
			tx.Rollback()
			return err
		}
	default:
		tx.Rollback()
		return errors.New("unsupported connection provider")
	}

	return tx.Commit()
}

func (p SQLProvider) RepoExists(ctx context.Context, user string, token *oauth2.Token, name string) (bool, error) {
	db, err := OpenDB()
	if err != nil {
		return false, err
	}
	defer db.Close()

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(1) FROM bookmarks WHERE user=?", user).Scan(&count)
	return count > 0, err
}

func (p SQLProvider) CreateUser(ctx context.Context, user, password string) error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM passwords WHERE user=?", user).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "INSERT INTO passwords(user, hash) VALUES(?, ?)", user, hash)
	return err
}

func (p SQLProvider) SetPassword(ctx context.Context, user, password string) error {
	db, err := OpenDB()
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
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (p SQLProvider) CheckPassword(ctx context.Context, user, password string) (bool, error) {
	db, err := OpenDB()
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
