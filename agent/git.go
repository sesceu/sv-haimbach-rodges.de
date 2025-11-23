package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitCreateBranch creates a new branch for the post.
func GitCreateBranch(title string) (string, error) {
	slug := slugify(title)
	branchName := fmt.Sprintf("post/%s-%d", slug, time.Now().Unix())

	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = ".." // Run in repo root
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git checkout -b failed: %s, %v", string(out), err)
	}
	return branchName, nil
}

// GitAddAndCommit adds files and commits them.
func GitAddAndCommit(files []string, message string) error {
	// Add files
	for _, f := range files {
		cmd := exec.Command("git", "add", f)
		cmd.Dir = ".."
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git add %s failed: %s, %v", f, string(out), err)
		}
	}

	// Commit
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = ".."
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s, %v", string(out), err)
	}
	return nil
}

// GitDiff returns the diff of the last commit.
func GitDiff() (string, error) {
	cmd := exec.Command("git", "show", "--stat", "--patch", "HEAD")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show failed: %s, %v", string(out), err)
	}
	return string(out), nil
}

// GitPush pushes the branch to origin.
func GitPush(branchName string) error {
	cmd := exec.Command("git", "push", "-u", "origin", branchName)
	cmd.Dir = ".."
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s, %v", string(out), err)
	}
	return nil
}

// GitHubCreatePR creates a PR using gh CLI.
func GitHubCreatePR(title, body string) (string, error) {
	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return "", fmt.Errorf("gh cli not found")
	}

	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr create failed: %s, %v", string(out), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "ä", "ae")
	s = strings.ReplaceAll(s, "ö", "oe")
	s = strings.ReplaceAll(s, "ü", "ue")
	s = strings.ReplaceAll(s, "ß", "ss")
	// Remove special chars
	reg := func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}
	return strings.Map(reg, s)
}
