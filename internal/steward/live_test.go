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
func TestStewardLivePRs(t *testing.T) {
	if os.Getenv("ZEN_PP_LIVE") != "1" {
		t.Skip("M057 live steward: set ZEN_PP_LIVE=1")
	}
	prs, err := DiscoverPRs(context.Background(), "zen-gc")
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultAutoMergePolicy()
	var results []map[string]any
	for _, pr := range prs {
		paths := changedPathsFor(t, pr.Number)
		rc := ClassifyPR(&pr, &policy, paths, pr.Title)
		row := map[string]any{
			"pr": pr.Number, "author": pr.Author, "author_class": pr.AuthorClass,
			"title": pr.Title, "risk": rc.Level, "verdict": rc.Verdict,
			"auto_merge_eligible": rc.AutoMergeEligible, "reasons": rc.Reasons,
		}
		t.Logf("PR #%d (%s): risk=%s verdict=%s reasons=%v", pr.Number, pr.AuthorClass, rc.Level, rc.Verdict, rc.Reasons)

		if rc.AutoMergeEligible {
			pass, detail, qerr := QualifyMergeCandidate(context.Background(),
				"/home/neves/zenmesh/zen-gc", pr.Number, pr.BaseSHA, pr.HeadSHA)
			row["qualified"] = pass
			row["qualification"] = detail
			if qerr != nil {
				row["qualification_error"] = qerr.Error()
				t.Logf("PR #%d qualification error: %v", pr.Number, qerr)
				continue
			}
			if pass {
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
			} else {
				t.Logf("PR #%d qualification FAILED: %s", pr.Number, detail)
			}
		}
		results = append(results, row)
	}
	b, _ := json.MarshalIndent(results, "", "  ")
	evidence := filepath.Join(t.TempDir(), "m057-steward-prs.json")
	if err := os.WriteFile(evidence, b, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("evidence: %s", evidence)
}

func changedPathsFor(t *testing.T, prNum int) []string {
	out, err := exec.Command("gh", "pr", "view", fmt.Sprint(prNum),
		"-R", "zenmesh/zen-gc", "--json", "files", "-q", "[.files[].path]").Output()
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}
