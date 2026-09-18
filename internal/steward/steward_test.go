// Copyright 2026 Zen Mesh. All rights reserved.

package steward

import "testing"

// TestClassifyPRProtectedPathFlag is a regression gate for the M058
// evidence-integrity defect: the protected-path flag was computed inside
// the deny-path loop and then clobbered back to false before the
// classification was returned, so persisted evidence always recorded
// protected_touched=false even when a protected path was modified.
func TestClassifyPRProtectedPathFlag(t *testing.T) {
	policy := DefaultAutoMergePolicy()
	pr := PRInfo{
		Number:    23,
		Title:     "deps(deps): bump go.uber.org/zap from 1.27.1 to 1.28.0",
		Author:    []byte(`{"login":"dependabot[bot]"}`),
		Additions: 4,
		Deletions: 4,
	}
	rc := ClassifyPR(&pr, &policy, []string{"go.mod", "go.sum"}, pr.Title)

	if !rc.ProtectedTouched {
		t.Fatalf("protected_touched=false for go.mod change: reasons=%v", rc.Reasons)
	}
	if rc.AutoMergeEligible {
		t.Fatalf("auto_merge_eligible=true for protected path change: verdict=%s", rc.Verdict)
	}
	if rc.Verdict != "ESCALATE" {
		t.Fatalf("verdict=%s, want ESCALATE", rc.Verdict)
	}
	if rc.Level != "HIGH" {
		t.Fatalf("level=%s, want HIGH", rc.Level)
	}
	if pr.AuthorClass != "DEPENDENCY_BOT" {
		t.Fatalf("author_class=%s, want DEPENDENCY_BOT", pr.AuthorClass)
	}
}

// TestClassifyPROrdinaryPatch pins the allow side of the path layer: a
// small, non-protected dependency-bot patch stays auto-merge eligible.
func TestClassifyPROrdinaryPatch(t *testing.T) {
	policy := DefaultAutoMergePolicy()
	pr := PRInfo{
		Number:    23,
		Title:     "deps(deps): bump go.uber.org/zap from 1.27.1 to 1.28.0",
		Author:    []byte(`{"login":"dependabot[bot]"}`),
		Additions: 4,
		Deletions: 4,
	}
	rc := ClassifyPR(&pr, &policy, []string{"vendor/example.go"}, pr.Title)

	if rc.ProtectedTouched {
		t.Fatalf("protected_touched=true for ordinary patch: reasons=%v", rc.Reasons)
	}
	if !rc.AutoMergeEligible {
		t.Fatalf("auto_merge_eligible=false for ordinary patch: verdict=%s reasons=%v", rc.Verdict, rc.Reasons)
	}
	if rc.Verdict != "AUTO_MERGE_ELIGIBLE" {
		t.Fatalf("verdict=%s, want AUTO_MERGE_ELIGIBLE", rc.Verdict)
	}
}

// TestSemanticLayerOverridesPathLayer documents why the two layers both
// exist: the path layer alone considers a one-line Dockerfile toolchain
// bump auto-merge eligible (Dockerfile is not in DenyPaths), but the
// semantic layer classifies it GO_TOOLCHAIN_CHANGE and DENIES authority.
// This is the PR #27 class of defect (M057 retrospective).
func TestSemanticLayerOverridesPathLayer(t *testing.T) {
	policy := DefaultAutoMergePolicy()
	pr := PRInfo{
		Number:    27,
		Title:     "docker(deps): bump golang from 1.26-alpine to 1.27-alpine",
		Author:    []byte(`{"login":"dependabot[bot]"}`),
		Additions: 1,
		Deletions: 1,
	}
	rc := ClassifyPR(&pr, &policy, []string{"Dockerfile"}, pr.Title)

	if !rc.AutoMergeEligible {
		t.Fatalf("precondition: path layer should mark this eligible, got verdict=%s reasons=%v", rc.Verdict, rc.Reasons)
	}
	sem, reason := classifySemantic(pr.ResolveAuthor(), pr.Title, []string{"Dockerfile"})
	if sem != ClassGoToolchain {
		t.Fatalf("semantic class=%s, want %s (%s)", sem, ClassGoToolchain, reason)
	}
	decision := EvaluateAutoMerge(pr.ResolveAuthor(), sem, reason)
	if decision.Allowed {
		t.Fatal("authority granted for GO_TOOLCHAIN_CHANGE; semantic layer must deny")
	}
}
