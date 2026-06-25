package gitapi

import (
	"os"
	"testing"
)

func TestNewGitConfig(t *testing.T) {
	// Set up environment variables
	os.Setenv("GIT_REPO_URL", "https://github.com/test/repo.git")
	os.Setenv("GIT_DEPLOY_TOKEN", "ghp_test_token")
	defer func() {
		os.Unsetenv("GIT_REPO_URL")
		os.Unsetenv("GIT_DEPLOY_TOKEN")
	}()

	cfg, err := NewGitConfig()
	if err != nil {
		t.Fatalf("NewGitConfig() error = %v", err)
	}

	if cfg.RepoURL != "https://github.com/test/repo.git" {
		t.Errorf("RepoURL = %v, want %v", cfg.RepoURL, "https://github.com/test/repo.git")
	}

	if cfg.Token != "ghp_test_token" {
		t.Errorf("Token = %v, want %v", cfg.Token, "ghp_test_token")
	}

	// Test defaults
	if cfg.Branch != "main" {
		t.Errorf("Branch = %v, want %v", cfg.Branch, "main")
	}

	if cfg.BackupDir != "outline-backup" {
		t.Errorf("BackupDir = %v, want %v", cfg.BackupDir, "outline-backup")
	}

	if cfg.CommitMsg != "Outline Wiki backup {timestamp}" {
		t.Errorf("CommitMsg = %v, want %v", cfg.CommitMsg, "Outline Wiki backup {timestamp}")
	}

	if cfg.AuthorName != "outline-backup" {
		t.Errorf("AuthorName = %v, want %v", cfg.AuthorName, "outline-backup")
	}

	if cfg.AuthorEmail != "backup@outlinewiki.local" {
		t.Errorf("AuthorEmail = %v, want %v", cfg.AuthorEmail, "backup@outlinewiki.local")
	}
}

func TestNewGitConfigCustomValues(t *testing.T) {
	// Set up environment variables with custom values
	os.Setenv("GIT_REPO_URL", "https://gitlab.com/test/repo.git")
	os.Setenv("GIT_DEPLOY_TOKEN", "glpat_test_token")
	os.Setenv("GIT_BRANCH", "backups")
	os.Setenv("GIT_BACKUP_DIR", "my-backups")
	os.Setenv("GIT_COMMIT_MESSAGE", "Custom backup at {timestamp}")
	os.Setenv("GIT_AUTHOR_NAME", "My Backup Bot")
	os.Setenv("GIT_AUTHOR_EMAIL", "bot@example.com")
	defer func() {
		os.Unsetenv("GIT_REPO_URL")
		os.Unsetenv("GIT_DEPLOY_TOKEN")
		os.Unsetenv("GIT_BRANCH")
		os.Unsetenv("GIT_BACKUP_DIR")
		os.Unsetenv("GIT_COMMIT_MESSAGE")
		os.Unsetenv("GIT_AUTHOR_NAME")
		os.Unsetenv("GIT_AUTHOR_EMAIL")
	}()

	cfg, err := NewGitConfig()
	if err != nil {
		t.Fatalf("NewGitConfig() error = %v", err)
	}

	if cfg.Branch != "backups" {
		t.Errorf("Branch = %v, want %v", cfg.Branch, "backups")
	}

	if cfg.BackupDir != "my-backups" {
		t.Errorf("BackupDir = %v, want %v", cfg.BackupDir, "my-backups")
	}

	if cfg.CommitMsg != "Custom backup at {timestamp}" {
		t.Errorf("CommitMsg = %v, want %v", cfg.CommitMsg, "Custom backup at {timestamp}")
	}

	if cfg.AuthorName != "My Backup Bot" {
		t.Errorf("AuthorName = %v, want %v", cfg.AuthorName, "My Backup Bot")
	}

	if cfg.AuthorEmail != "bot@example.com" {
		t.Errorf("AuthorEmail = %v, want %v", cfg.AuthorEmail, "bot@example.com")
	}
}

func TestNewGitConfigMissingRequired(t *testing.T) {
	// Test missing GIT_REPO_URL
	os.Unsetenv("GIT_REPO_URL")
	os.Setenv("GIT_DEPLOY_TOKEN", "token")
	defer os.Unsetenv("GIT_DEPLOY_TOKEN")

	_, err := NewGitConfig()
	if err == nil {
		t.Fatal("NewGitConfig() should error when GIT_REPO_URL is missing")
	}

	// Test missing GIT_DEPLOY_TOKEN
	os.Setenv("GIT_REPO_URL", "https://github.com/test/repo.git")
	os.Unsetenv("GIT_DEPLOY_TOKEN")
	defer os.Unsetenv("GIT_REPO_URL")

	_, err = NewGitConfig()
	if err == nil {
		t.Fatal("NewGitConfig() should error when GIT_DEPLOY_TOKEN is missing")
	}
}

func TestClearDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-clear-dir-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files and subdirectories
	testFile := tmpDir + "/file1.txt"
	testDir := tmpDir + "/subdir"
	testSubFile := testDir + "/file2.txt"

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	if err := os.WriteFile(testSubFile, []byte("test2"), 0644); err != nil {
		t.Fatalf("Failed to create test subfile: %v", err)
	}

	// Clear the directory
	if err := clearDirectory(tmpDir); err != nil {
		t.Fatalf("clearDirectory() error = %v", err)
	}

	// Check that all files are removed
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read dir: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestClearDirectoryNotExists(t *testing.T) {
	// Try to clear a non-existent directory - should not error
	err := clearDirectory("/nonexistent/path/12345")
	if err != nil {
		t.Errorf("clearDirectory() should not error for non-existent dir, got: %v", err)
	}
}
