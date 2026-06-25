package gitapi

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitConfig holds Git repository configuration
type GitConfig struct {
	RepoURL     string
	Token       string
	Branch      string
	BackupDir   string
	CommitMsg   string
	AuthorName  string
	AuthorEmail string
}

// NewGitConfig creates a new Git configuration from environment variables
func NewGitConfig() (*GitConfig, error) {
	repoURL := os.Getenv("GIT_REPO_URL")
	if repoURL == "" {
		return nil, fmt.Errorf("GIT_REPO_URL environment variable is not set")
	}

	token := os.Getenv("GIT_DEPLOY_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GIT_DEPLOY_TOKEN environment variable is not set")
	}

	branch := os.Getenv("GIT_BRANCH")
	if branch == "" {
		branch = "main"
	}

	backupDir := os.Getenv("GIT_BACKUP_DIR")
	if backupDir == "" {
		backupDir = "outline-backup"
	}

	commitMsg := os.Getenv("GIT_COMMIT_MESSAGE")
	if commitMsg == "" {
		commitMsg = "Outline Wiki backup {timestamp}"
	}

	authorName := os.Getenv("GIT_AUTHOR_NAME")
	if authorName == "" {
		authorName = "outline-backup"
	}

	authorEmail := os.Getenv("GIT_AUTHOR_EMAIL")
	if authorEmail == "" {
		authorEmail = "backup@outlinewiki.local"
	}

	return &GitConfig{
		RepoURL:     repoURL,
		Token:       token,
		Branch:      branch,
		BackupDir:   backupDir,
		CommitMsg:   commitMsg,
		AuthorName:  authorName,
		AuthorEmail: authorEmail,
	}, nil
}

// UploadToGit extracts the zip backup and commits to Git repository
func UploadToGit(zipPath string) error {
	cfg, err := NewGitConfig()
	if err != nil {
		return fmt.Errorf("failed to create git config: %w", err)
	}

	// Create temp directory for git operations
	tempDir, err := os.MkdirTemp("", "outline-git-backup-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository
	repo, err := cloneRepository(tempDir, cfg)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Clear existing backup directory
	backupPath := filepath.Join(tempDir, cfg.BackupDir)
	if err := clearDirectory(backupPath); err != nil {
		return fmt.Errorf("failed to clear backup directory: %w", err)
	}

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Extract zip to backup directory
	if err := extractZip(zipPath, backupPath); err != nil {
		return fmt.Errorf("failed to extract zip: %w", err)
	}

	// Stage, commit, and push
	return commitAndPush(repo, cfg, backupPath)
}

// cloneRepository clones the git repository
func cloneRepository(targetDir string, cfg *GitConfig) (*git.Repository, error) {
	auth := &http.BasicAuth{
		Username: "token",
		Password: cfg.Token,
	}

	// Try with branch first
	referenceName := plumbing.NewBranchReferenceName(cfg.Branch)
	repo, err := git.PlainClone(targetDir, false, &git.CloneOptions{
		URL:           cfg.RepoURL,
		Auth:          auth,
		Depth:         1,
		ReferenceName: referenceName,
		SingleBranch:  true,
		Progress:      os.Stdout,
	})
	if err != nil {
		// Check if it's a branch not found error
		if isBranchNotFound(err) {
			// Try without specifying branch (will use default branch)
			repo, err = git.PlainClone(targetDir, false, &git.CloneOptions{
				URL:      cfg.RepoURL,
				Auth:     auth,
				Depth:    1,
				Progress: os.Stdout,
			})
		}
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository from %s: %w", cfg.RepoURL, err)
		}
	}
	return repo, nil
}

// isBranchNotFound checks if the error is due to branch not found
func isBranchNotFound(err error) bool {
	return strings.Contains(err.Error(), "branch not found") ||
		strings.Contains(err.Error(), "reference not found") ||
		strings.Contains(err.Error(), "not found")
}

// extractZip extracts a zip file to the target directory
func extractZip(zipPath string, targetDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Skip directories - they'll be created automatically
		if f.FileInfo().IsDir() {
			continue
		}

		destPath := filepath.Join(targetDir, f.Name)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", f.Name, err)
		}

		// Extract file
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip %s: %w", f.Name, err)
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file %s: %w", destPath, err)
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}
	}
	return nil
}

// commitAndPush stages, commits, and pushes the changes
func commitAndPush(repo *git.Repository, cfg *GitConfig, backupDir string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add the entire backup directory (this handles additions, modifications, and deletions)
	if _, err := worktree.Add(cfg.BackupDir); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	// Generate commit message with timestamp
	timestamp := time.Now().Format("2006-01-02 15:04:05 MST")
	commitMsg := strings.ReplaceAll(cfg.CommitMsg, "{timestamp}", timestamp)

	// Create commit
	commitHash, err := worktree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  cfg.AuthorName,
			Email: cfg.AuthorEmail,
			When:  time.Now(),
		},
		All: true, // This ensures all changes including deletions are committed
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Printf("Created commit: %s", commitHash.String()[:7])

	// Push
	auth := &http.BasicAuth{
		Username: "token",
		Password: cfg.Token,
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push: %w", err)
	}

	log.Println("Successfully pushed to Git repository")
	return nil
}

// clearDirectory removes all files and subdirectories from a directory
func clearDirectory(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	return nil
}
