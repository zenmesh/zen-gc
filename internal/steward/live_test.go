// Copyright 2026 Zen Mesh. All rights reserved.

package steward

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func execCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// TestStewardLivePRs is the M057 live qualification (env-gated).
//
// M058: the live flow is RECOMMENDATION-mode by default — it discovers,
// classifies (path layer + semantic authority layer), qualifies merge
// candidates in isolation, and persists evidence, but never mutates Git
// or GitHub. Merging additionally requires ZEN_GC_AUTO_MERGE_LIVE=1
// (auto-merge remains disabled in M058).
func TestStewardLivePRs(t *testing.T) {
	if os.Getenv("ZEN_PP_LIVE") != "1" {
		t.Skip("M057 live steward: set ZEN_PP_LIVE=1")
	}
	autoMergeLive := os.Getenv("ZEN_GC_AUTO_MERGE_LIVE") == "1"
	prs, err := DiscoverPRs(context.Background(), "zen-gc")
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultAutoMergePolicy()
	var results []map[string]any
	for _, pr := range prs {
		paths := changedPathsFor(t, pr.Number)
		rc := ClassifyPR(&pr, &policy, paths, pr.Title)
		sem, semReason := classifySemantic(pr.ResolveAuthor(), pr.Title, paths)
		decision := EvaluateAutoMerge(pr.ResolveAuthor(), sem, semReason)
		row := map[string]any{
			"pr": pr.Number, "author": pr.Author, "author_class": pr.AuthorClass,
			"title": pr.Title, "risk": rc.Level, "verdict": rc.Verdict,
			"auto_merge_eligible": rc.AutoMergeEligible, "reasons": rc.Reasons,
			"semantic_class": sem, "semantic_reason": semReason,
			"semantic_authority_allowed": decision.Allowed,
			"mode":                       "recommendation",
		}
		t.Logf("PR #%d (%s): risk=%s verdict=%s semantic=%s authority_allowed=%v reasons=%v",
			pr.Number, pr.AuthorClass, rc.Level, rc.Verdict, sem, decision.Allowed, rc.Reasons)

		mergeAuthorized := rc.AutoMergeEligible && decision.Allowed
		if !mergeAuthorized {
			row["decision"] = "RECOMMENDATION_ONLY"
			results = append(results, row)
			continue
		}
		pass, detail, qerr := QualifyMergeCandidate(context.Background(),
			"/home/neves/zenmesh/zen-gc", pr.Number, pr.BaseSHA, pr.HeadSHA)
		row["qualified"] = pass
		row["qualification"] = detail
		if qerr != nil {
			row["qualification_error"] = qerr.Error()
			t.Logf("PR #%d qualification error: %v", pr.Number, qerr)
			results = append(results, row)
			continue
		}
		if !pass {
			t.Logf("PR #%d qualification FAILED: %s", pr.Number, detail)
			results = append(results, row)
			continue
		}
		if !autoMergeLive {
			row["decision"] = "RECOMMEND_MERGE_BLOCKED_BY_MODE"
			t.Logf("PR #%d qualified; auto-merge disabled (recommendation mode)", pr.Number)
			results = append(results, row)
			continue
		}
		merge := execCommand("gh", "pr", "merge", fmt.Sprint(pr.Number),
			"-R", "zenmesh/zen-gc", "--squash", "--delete-branch")
		if b, merr := merge.CombinedOutput(); merr != nil {
			row["merge_error"] = merr.Error()
			row["merge_output"] = string(b)
			t.Logf("PR #%d merge failed: %v: %s", pr.Number, merr, b)
		} else {
			row["merged"] = true
			t.Logf("PR #%d MERGED", pr.Number)
		}
		results = append(results, row)
	}
	b, _ := json.MarshalIndent(results, "", "  ")
	evidence := filepath.Join(evidenceDir(t, os.Getenv("ZEN_GC_STEWARD_EVIDENCE")), "steward-pr-census.json")
	// The destination is operator-supplied env in a manually-run live
	// qualification, sanitized by evidenceDir (absolute, cleaned, existing
	// directory). Not an untrusted source.
	if err := os.WriteFile(evidence, b, 0o600); err != nil { // #nosec G703
		t.Fatal(err)
	}
	t.Logf("evidence: %s", evidence)
}

// evidenceDir returns the operator-provided evidence directory when it is
// a safe absolute path that already exists; otherwise the test's own
// temp directory is used.
func evidenceDir(t *testing.T, env string) string {
	t.Helper()
	clean := filepath.Clean(env)
	if filepath.IsAbs(env) && clean != "." && !strings.HasPrefix(clean, "..") {
		if info, err := os.Stat(clean); err == nil && info.IsDir() {
			return clean
		}
	}
	return t.TempDir()
}

func changedPathsFor(t *testing.T, prNum int) []string {
	// Raw per-line output: an array filter ([.files[].path]) returns a
	// JSON array string that defeats prefix matching downstream (the
	// M058 discovery defect — protected-path detection never fired).
	out, err := exec.Command("gh", "pr", "view", fmt.Sprint(prNum),
		"-R", "zenmesh/zen-gc", "--json", "files", "-q", ".files[].path").Output()
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}
