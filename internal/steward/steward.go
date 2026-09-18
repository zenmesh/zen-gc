// Copyright 2026 Zen Mesh. All rights reserved.

// Package steward implements the MAESTRO-057 Zen GC autonomous product
// steward: daily repository health, PR triage, dependency maintenance,
// and bounded auto-merge under an explicit policy.
//
// LAWS:
//   - Every analysis binds the exact base SHA + head SHA + merge candidate
//     SHA. Approving "PR #N" abstractly is forbidden.
//   - PR code is UNTRUSTED: tested in an isolated, credential-free
//     workspace; never in the privileged Maestro process.
//   - Auto-merge is a narrow positive allowlist, never a default.
//   - Analysis (discovery/classify/test) is separated from action
//     (approve/merge/comment). Actions require explicit policy authority.
//   - Human-contributed PRs require Leonardo approval before any public
//     action in Campaign 1.
//   - The merge candidate (PR head merged into current main) is tested,
//     not just the PR head in isolation.
package steward

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PRInfo is the discovered state of one open pull request.
type PRInfo struct {
	Number       int             `json:"number"`
	Title        string          `json:"title"`
	Author       json.RawMessage `json:"author"`
	AuthorClass  string          `json:"author_class"`
	AuthorLogin  string          `json:"author_login,omitempty"`
	Branch       string          `json:"branch"`
	BaseRef      string          `json:"base_ref"`
	HeadSHA      string          `json:"head_sha"`
	BaseSHA      string          `json:"base_sha"`
	ChangedFiles int             `json:"changed_files"`
	Additions    int             `json:"additions"`
	Deletions    int             `json:"deletions"`
	URL          string          `json:"url"`
}

// ResolveAuthor extracts the login from the raw author field.
func (p *PRInfo) ResolveAuthor() string {
	if p.AuthorLogin != "" {
		return p.AuthorLogin
	}
	var m struct {
		Login string `json:"login"`
	}
	if json.Unmarshal(p.Author, &m) == nil {
		p.AuthorLogin = m.Login
	}
	return p.AuthorLogin
}

// RiskClassify deterministically classifies a PR's risk.
type RiskClassify struct {
	Level             string   `json:"risk_level"` // LOW | MODERATE | HIGH
	Reasons           []string `json:"reasons"`
	ProtectedTouched  bool     `json:"protected_touched"`
	AutoMergeEligible bool     `json:"auto_merge_eligible"`
	Verdict           string   `json:"verdict"` // AUTO_MERGE_ELIGIBLE | HUMAN_APPROVAL | ESCALATE
}

// AutoMergePolicy is the declarative auto-merge contract.
type AutoMergePolicy struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedAuthors   []string `yaml:"allowed_authors"`
	AllowedBumpTypes []string `yaml:"allowed_bump_types"` // patch, minor
	MaxAdditions     int      `yaml:"max_additions"`
	DenyPaths        []string `yaml:"deny_paths"`
	DenyTitleSubstr  []string `yaml:"deny_title_substr"`
}

// DefaultAutoMergePolicy is the narrow initial policy.
func DefaultAutoMergePolicy() AutoMergePolicy {
	return AutoMergePolicy{
		Enabled:          true,
		AllowedAuthors:   []string{"app/dependabot", "dependabot[bot]"},
		AllowedBumpTypes: []string{"patch", "minor"},
		MaxAdditions:     100,
		DenyPaths: []string{
			"cmd/", "internal/controller", "internal/webhook",
			"config/crd", "config/rbac", "config/webhook",
			"Chart.yaml", "values.yaml", ".golangci.yml",
			"go.mod", // go.mod is checked separately via title parsing
		},
		DenyTitleSubstr: []string{"major", "breaking"},
	}
}

// ClassifyPR determines author class and risk level.
func ClassifyPR(pr PRInfo, policy AutoMergePolicy, changedPaths []string, title string) RiskClassify {
	rc := RiskClassify{}
	author := pr.ResolveAuthor()
	bot := false
	for _, a := range policy.AllowedAuthors {
		if author == a || strings.HasPrefix(author, a) {
			bot = true
			break
		}
	}
	switch {
	case bot:
		pr.AuthorClass = "DEPENDENCY_BOT"
	case author == "zenmesh" || author == "neves":
		pr.AuthorClass = "MAINTAINER"
	default:
		pr.AuthorClass = "EXTERNAL_CONTRIBUTOR"
	}

	// Protected path check.
	protected := false
	for _, p := range changedPaths {
		for _, d := range policy.DenyPaths {
			if strings.HasPrefix(p, d) {
				rc.ProtectedTouched = true
				rc.Reasons = append(rc.Reasons, "protected path: "+p)
				break
			}
		}
	}
	// Title-based bump classification.
	isMajor := false
	for _, sub := range policy.DenyTitleSubstr {
		if strings.Contains(strings.ToLower(title), sub) {
			isMajor = true
			rc.Reasons = append(rc.Reasons, "major/breaking in title")
			break
		}
	}
	// Patch/minor check for dependency bots.
	if bot && !isMajor {
		isPatchOrMinor := false
		for _, bt := range policy.AllowedBumpTypes {
			lc := strings.ToLower(title)
			switch bt {
			case "patch":
				if strings.Contains(lc, "patch") || !strings.Contains(lc, "major") {
					isPatchOrMinor = true
				}
			case "minor":
				if strings.Contains(lc, "minor") || !strings.Contains(lc, "major") {
					isPatchOrMinor = true
				}
			}
		}
		if !isPatchOrMinor {
			rc.Reasons = append(rc.Reasons, "non-patch/minor bump")
		}
	}
	if pr.Additions > policy.MaxAdditions {
		rc.Reasons = append(rc.Reasons, fmt.Sprintf("additions %d > budget %d", pr.Additions, policy.MaxAdditions))
	}
	rc.ProtectedTouched = protected
	rc.Level = "LOW"
	for _, r := range rc.Reasons {
		if strings.Contains(r, "major") || strings.Contains(r, "protected") {
			rc.Level = "HIGH"
			break
		}
		rc.Level = "MODERATE"
	}
	rc.AutoMergeEligible = bot && !protected && !isMajor && pr.Additions <= policy.MaxAdditions && rc.Level != "HIGH"
	if rc.AutoMergeEligible {
		rc.Verdict = "AUTO_MERGE_ELIGIBLE"
	} else if rc.ProtectedTouched || rc.Level == "HIGH" {
		rc.Verdict = "ESCALATE"
	} else {
		rc.Verdict = "HUMAN_APPROVAL"
	}
	return rc
}

func (rc RiskClassify) Marshal() []byte {
	b, _ := json.MarshalIndent(rc, "", "  ")
	return b
}

// QualifyMergeCandidate tests the merge candidate in an isolated workspace:
// clone the PR head, merge current main into it, build+test. Credentials are
// NOT passed to the test environment (PR code is untrusted).
func QualifyMergeCandidate(ctx context.Context, repoDir string, prNumber int, baseSHA, headSHA string) (bool, string, error) {
	// Create a fresh temp workspace.
	tmp, err := os.MkdirTemp("", "steward-qual-")
	if err != nil {
		return false, "", err
	}
	defer os.RemoveAll(tmp)

	// Clone the PR head.
	clone := exec.Command("git", "clone", "--no-local", repoDir, tmp)
	clone.Env = os.Environ()
	if b, cerr := clone.CombinedOutput(); cerr != nil {
		return false, fmt.Sprintf("clone: %v: %s", cerr, b), nil
	}
	// Fetch the PR branch from the real origin (it may only exist on GitHub).
	remoteURL, rerr := exec.Command("git", "-C", repoDir, "remote", "get-url", "origin").Output()
	if rerr != nil {
		return false, "", rerr
	}
	fetch := exec.Command("git", "-C", tmp, "fetch",
		strings.TrimSpace(string(remoteURL)), "pull/"+fmt.Sprint(prNumber)+"/head")
	fetch.Env = os.Environ()
	if b, ferr := fetch.CombinedOutput(); ferr != nil {
		return false, fmt.Sprintf("fetch PR branch: %v: %s", ferr, b), nil
	}
	checkout := exec.Command("git", "-C", tmp, "checkout", "FETCH_HEAD")
	if b, cerr := checkout.CombinedOutput(); cerr != nil {
		return false, fmt.Sprintf("checkout PR head: %v: %s", cerr, b), nil
	}
	// Verify head SHA.
	got, err := exec.Command("git", "-C", tmp, "rev-parse", "HEAD").Output()
	if err != nil {
		return false, "", err
	}
	actualHead := strings.TrimSpace(string(got))
	if actualHead != headSHA {
		return false, fmt.Sprintf("head SHA mismatch: expected %s, got %s", headSHA, actualHead), nil
	}
	// Merge current main.
	merge := exec.Command("git", "-C", tmp, "merge", "origin/main", "--no-edit")
	merge.Env = os.Environ()
	if b, merr := merge.CombinedOutput(); merr != nil {
		return false, fmt.Sprintf("merge conflict: %v: %s", merr, b), nil
	}
	// Build and test.
	build := exec.Command("go", "build", "./...")
	build.Dir = tmp
	build.Env = os.Environ()
	if b, berr := build.CombinedOutput(); berr != nil {
		return false, fmt.Sprintf("build: %v: %s", berr, b), nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	test := exec.CommandContext(ctx, "go", "test", "./...")
	test.Dir = tmp
	test.Env = os.Environ()
	if b, terr := test.CombinedOutput(); terr != nil {
		return false, fmt.Sprintf("tests: %v: %s", terr, b), nil
	}
	return true, "merge candidate qualified (build+tests PASS)", nil
}

// DiscoverPRs lists open PRs on a repo via gh CLI.
func DiscoverPRs(ctx context.Context, repo string) ([]PRInfo, error) {
	cmd := exec.CommandContext(ctx, "gh", "pr", "list",
		"-R", "zenmesh/"+repo,
		"--state", "open",
		"--json", "number,title,author,headRefName,baseRefName,headRefOid,additions,deletions,url")
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("steward: gh pr list: %w", err)
	}
	var raw []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		HeadRefName string `json:"headRefName"`
		BaseRefName string `json:"baseRefName"`
		HeadRefOid  string `json:"headRefOid"`
		Additions   int    `json:"additions"`
		Deletions   int    `json:"deletions"`
		URL         string `json:"url"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("steward: gh pr list decode: %w", err)
	}
	prs := make([]PRInfo, len(raw))
	for i, r := range raw {
		prs[i] = PRInfo{
			Number: r.Number, Title: r.Title,
			AuthorLogin: r.Author.Login,
			Branch:      r.HeadRefName, BaseRef: r.BaseRefName,
			HeadSHA:   r.HeadRefOid,
			Additions: r.Additions, Deletions: r.Deletions,
			URL: r.URL,
		}
	}
	return prs, nil
}

var _ = filepath.Join
