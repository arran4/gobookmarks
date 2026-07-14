package skill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// SkillSource represents a source from which a skill can be fetched.
type SkillSource interface {
	// Fetch retrieves the skill and writes its contents to destDir.
	// It returns the resolved metadata for the fetched skill.
	Fetch(ctx context.Context, destDir string) (SkillMetadata, error)
}

// LocalSource represents a skill located on the local filesystem.
type LocalSource struct {
	Path string
	Name string
}

func (s *LocalSource) Fetch(ctx context.Context, destDir string) (SkillMetadata, error) {
	absPath, err := filepath.Abs(s.Path)
	if err != nil {
		return SkillMetadata{}, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return SkillMetadata{}, err
	}

	if !info.IsDir() {
		return SkillMetadata{}, fmt.Errorf("local source must be a directory: %s", absPath)
	}

	skillMdPath := filepath.Join(absPath, "SKILL.md")
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return SkillMetadata{}, fmt.Errorf("SKILL.md not found in %s", absPath)
	}

	// Calculate a simple digest based on SKILL.md for local changes tracking
	b, err := os.ReadFile(skillMdPath)
	if err != nil {
		return SkillMetadata{}, err
	}
	hash := sha256.Sum256(b)
	digest := hex.EncodeToString(hash[:])

	err = CopyDir(absPath, destDir)
	if err != nil {
		return SkillMetadata{}, err
	}

	return SkillMetadata{
		Name:     s.Name,
		Source:   "local",
		Original: absPath,
		Revision: digest,
	}, nil
}

// GitSource represents a skill located in a git repository.
type GitSource struct {
	RepositoryURL string
	PathInRepo    string
	Revision      string // branch, tag, or commit
	Name          string
}

func (s *GitSource) Fetch(ctx context.Context, destDir string) (SkillMetadata, error) {
	// Clone to a temporary directory first
	tempDir, err := os.MkdirTemp("", "gobookmarks-skill-*")
	if err != nil {
		return SkillMetadata{}, err
	}
	defer os.RemoveAll(tempDir)

	cloneOpts := &git.CloneOptions{
		URL:      s.RepositoryURL,
		Progress: nil, // Add progress if needed
		Depth:    1,
	}

	if s.Revision != "" {
		cloneOpts.ReferenceName = plumbing.ReferenceName(s.Revision)
		if !strings.HasPrefix(s.Revision, "refs/") {
			// A rough heuristic; in real life we might want to check branches/tags
			cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(s.Revision)
		}
	}

	r, err := git.PlainCloneContext(ctx, tempDir, false, cloneOpts)
	if err != nil {
		return SkillMetadata{}, fmt.Errorf("failed to clone repository: %w", err)
	}

	ref, err := r.Head()
	if err != nil {
		return SkillMetadata{}, err
	}
	commitHash := ref.Hash().String()

	sourcePath := tempDir
	if s.PathInRepo != "" {
		sourcePath = filepath.Join(tempDir, s.PathInRepo)
		// Ensure path in repo doesn't escape the temp dir
		cleanSourcePath := filepath.Clean(sourcePath)
		if !strings.HasPrefix(cleanSourcePath, tempDir) {
			return SkillMetadata{}, fmt.Errorf("invalid path in repository: %s", s.PathInRepo)
		}
	}

	skillMdPath := filepath.Join(sourcePath, "SKILL.md")
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return SkillMetadata{}, fmt.Errorf("SKILL.md not found in %s", s.PathInRepo)
	}

	err = CopyDir(sourcePath, destDir)
	if err != nil {
		return SkillMetadata{}, err
	}

	return SkillMetadata{
		Name:     s.Name,
		Source:   "git",
		Original: s.RepositoryURL,
		Path:     s.PathInRepo,
		Revision: commitHash,
	}, nil
}

// copyDir recursively copies a directory tree, attempting to preserve permissions.
// Warning: This does not resolve symlinks safely against directory traversal in its simple form.
// For a production installer, we should be much more careful.
func CopyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// Skip copying the .git directory
		if relPath == ".git" || strings.HasPrefix(relPath, ".git/") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// Safe symlink handling is complex. For now, we skip symlinks or return an error.
			return fmt.Errorf("symlinks in skills are not supported for security reasons: %s", path)
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// ParseSource parses a source string and returns an appropriate SkillSource.
func ParseSource(sourceStr string, name string) (SkillSource, error) {
	// Very basic parser.
	// If it starts with . or /, treat it as a local path.
	if strings.HasPrefix(sourceStr, ".") || strings.HasPrefix(sourceStr, "/") {
		return &LocalSource{Path: sourceStr, Name: name}, nil
	}

	// If it looks like owner/repo, treat it as github
	parts := strings.Split(sourceStr, "/")
	if len(parts) == 2 && !strings.Contains(sourceStr, "://") {
		return &GitSource{
			RepositoryURL: fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1]),
			Name:          name,
		}, nil
	}

	// Treat as a direct Git URL if it has scheme
	if strings.HasPrefix(sourceStr, "http://") || strings.HasPrefix(sourceStr, "https://") || strings.HasPrefix(sourceStr, "git@") {
		return &GitSource{
			RepositoryURL: sourceStr,
			Name:          name,
		}, nil
	}

	return nil, fmt.Errorf("unrecognized source format: %s", sourceStr)
}
