//go:build jsonprovider

package gobookmarks

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// JSONProvider implements Provider using a JSON file on disk.
type JSONProvider struct{}

func init() { RegisterProvider(JSONProvider{}) }

func (JSONProvider) Name() string { return "json" }

func (JSONProvider) DefaultServer() string { return "" }

func (JSONProvider) Config(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return nil
}

func (JSONProvider) CurrentUser(ctx context.Context, token *oauth2.Token) (*User, error) {
	return &User{Login: "local"}, nil
}

func (JSONProvider) GetTags(ctx context.Context, user string, token *oauth2.Token) ([]*Tag, error) {
	path := filepath.Join(userDir(user), "tags.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	res := make([]*Tag, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			continue
		}
		res = append(res, &Tag{Name: l})
	}
	return res, nil
}

func (JSONProvider) GetBranches(ctx context.Context, user string, token *oauth2.Token) ([]*Branch, error) {
	path := filepath.Join(userDir(user), "branches.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Branch{{Name: "main"}}, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return []*Branch{{Name: "main"}}, nil
	}
	res := make([]*Branch, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			continue
		}
		res = append(res, &Branch{Name: l})
	}
	return res, nil
}

func (JSONProvider) GetCommits(ctx context.Context, user string, token *oauth2.Token) ([]*Commit, error) {
	path := commitTarPath(user, "main")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	var commits []*Commit
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		commits = append(commits, &Commit{
			SHA:            strings.TrimSuffix(hdr.Name, filepath.Ext(hdr.Name)),
			Message:        hdr.Name,
			CommitterName:  "local",
			CommitterEmail: "local@example.com",
			CommitterDate:  hdr.ModTime,
		})
	}
	return commits, nil
}

func (JSONProvider) GetBookmarks(ctx context.Context, user, ref string, token *oauth2.Token) (string, string, error) {
	if ref == "" {
		ref = "main"
	}
	p := filepath.Join(userDir(user), ref+".txt")
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", ErrRepoNotFound
		}
		return "", "", err
	}
	sum := sha256.Sum256(data)
	return string(data), hex.EncodeToString(sum[:]), nil
}

func (JSONProvider) UpdateBookmarks(ctx context.Context, user string, token *oauth2.Token, sourceRef, branch, text, expectSHA string) error {
	if branch == "" {
		branch = "main"
	}
	_, sha, err := JSONProvider{}.GetBookmarks(ctx, user, branch, token)
	if err != nil && !errors.Is(err, ErrRepoNotFound) {
		return err
	}
	if err == nil && sha != expectSHA {
		return errors.New("sha mismatch")
	}
	if err := os.WriteFile(filepath.Join(userDir(user), branch+".txt"), []byte(text), 0600); err != nil {
		return err
	}
	return appendCommit(user, branch, []byte(text))
}

func (JSONProvider) CreateBookmarks(ctx context.Context, user string, token *oauth2.Token, branch, text string) error {
	if branch == "" {
		branch = "main"
	}
	dir := userDir(user)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, branch+".txt"), []byte(text), 0600); err != nil {
		return err
	}
	branchesPath := filepath.Join(dir, "branches.txt")
	var branches []string
	if data, err := os.ReadFile(branchesPath); err == nil {
		branches = strings.Split(strings.TrimSpace(string(data)), "\n")
	}
	found := false
	for i, b := range branches {
		if b == branch {
			branches[i] = branch
			found = true
		}
	}
	if !found {
		branches = append(branches, branch)
	}
	if err := os.WriteFile(branchesPath, []byte(strings.Join(branches, "\n")+"\n"), 0600); err != nil {
		return err
	}
	tagsPath := filepath.Join(dir, "tags.txt")
	if data, err := os.ReadFile(tagsPath); err == nil {
		for _, t := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if t == branch {
				return fmt.Errorf("branch %s already exists as tag", branch)
			}
		}
	}
	if _, err := os.Stat(tagsPath); os.IsNotExist(err) {
		if err := os.WriteFile(tagsPath, []byte(""), 0600); err != nil {
			return err
		}
	}
	return appendCommit(user, branch, []byte(text))
}

func (JSONProvider) CreateRepo(ctx context.Context, user string, token *oauth2.Token, name string) error {
	return JSONProvider{}.CreateBookmarks(ctx, user, token, "main", "")
}

func userDir(user string) string {
	uh := sha256.Sum256([]byte(user))
	return filepath.Join(JSONDBPath, hex.EncodeToString(uh[:]))
}

func userPasswordPath(user string) string { return filepath.Join(userDir(user), ".password") }

func commitTarPath(user, branch string) string { return filepath.Join(userDir(user), branch+".tar") }

func appendCommit(user, branch string, data []byte) error {
	path := commitTarPath(user, branch)
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// copy existing tar entries if any
	if old, err := os.Open(path); err == nil {
		tr := tar.NewReader(old)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				old.Close()
				return err
			}
			if err := tw.WriteHeader(hdr); err != nil {
				old.Close()
				return err
			}
			if _, err := io.Copy(tw, tr); err != nil {
				old.Close()
				return err
			}
		}
		old.Close()
	}

	name := time.Now().UTC().Format("20060102150405") + ".txt"
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), ModTime: time.Now()}); err != nil {
		return err
	}
	if _, err := tw.Write(data); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0600)
}
