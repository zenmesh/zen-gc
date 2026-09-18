// Copyright 2026 Zen Mesh. All rights reserved.

// Semantic dependency-risk classifier (M057 governance correction):
// risk evaluation must combine author class + changed paths + DEPENDENCY
// IDENTITY + SEMANTIC CLASS + manifest type + version delta. Path/size
// checks alone are insufficient (a one-line Dockerfile toolchain bump is
// a protected GO_TOOLCHAIN_CHANGE + CONTAINER_BASE_CHANGE even if
// Dependabot authored it and tests pass).
package steward

import (
	"fmt"
	"strings"
)

// SemanticClass is the protected dependency semantic classification.
type SemanticClass string

const (
	ClassOrdinaryPatch     SemanticClass = "ORDINARY_PATCH"
	ClassOrdinaryMinor     SemanticClass = "ORDINARY_MINOR"
	ClassGoToolchain       SemanticClass = "GO_TOOLCHAIN_CHANGE"
	ClassContainerBase     SemanticClass = "CONTAINER_BASE_CHANGE"
	ClassK8sCompat         SemanticClass = "KUBERNETES_COMPATIBILITY"
	ClassCRDSchema         SemanticClass = "CRD_SCHEMA"
	ClassControllerRuntime SemanticClass = "CONTROLLER_RUNTIME"
	ClassSecurity          SemanticClass = "SECURITY_SENSITIVE"
	ClassSigning           SemanticClass = "SIGNING_PROVENANCE"
	ClassBuildInfra        SemanticClass = "BUILD_RELEASE_INFRA"
	ClassHumanContrib      SemanticClass = "HUMAN_CONTRIBUTION"
	ClassUnknown           SemanticClass = "UNKNOWN"
)

// classifySemantic determines the protected semantic class from the PR's
// identity and content. Unknown => HUMAN_APPROVAL_REQUIRED (fail closed).
func classifySemantic(author, title string, changedPaths []string) (SemanticClass, string) {
	lc := strings.ToLower(title)

	// Container base image change (Dockerfile).
	for _, p := range changedPaths {
		lcp := strings.ToLower(p)
		if strings.Contains(lcp, "dockerfile") {
			if strings.Contains(lc, "golang") || strings.Contains(lc, "go ") {
				return ClassGoToolchain, "Dockerfile Go toolchain/base image change"
			}
			return ClassContainerBase, "Dockerfile container base image change"
		}
	}

	// Go toolchain directive: title patterns like "bump go from 1.26 to 1.27"
	// or "bump golang from X to Y" in go.mod. Also match explicit "toolchain".
	if strings.Contains(lc, "toolchain") || strings.Contains(lc, "go 1.") ||
		strings.Contains(lc, "bump go from") || strings.Contains(lc, "bump golang from") {
		return ClassGoToolchain, "Go toolchain directive bump"
	}

	// Major version bump: parse "from X to Y" and compare major components.
	if strings.Contains(lc, "major") {
		return ClassUnknown, "major version bump (protected by default)"
	}
	for _, pattern := range []string{"from ", "to "} {
		_ = pattern
	}
	if i := strings.Index(lc, " from "); i >= 0 {
		fromPart := lc[i+6:]
		if j := strings.Index(fromPart, " to "); j >= 0 {
			from := fromPart[:j]
			to := strings.TrimSpace(fromPart[j+4:])
			fromMajor := semMajor(from)
			toMajor := semMajor(to)
			if fromMajor != "" && toMajor != "" && fromMajor != toMajor {
				return ClassUnknown, fmt.Sprintf("major version bump %s -> %s", fromMajor, toMajor)
			}
		}
	}

	// Kubernetes compatibility dependencies.
	for _, k8sDep := range []string{"k8s.io/api", "k8s.io/apimachinery", "k8s.io/client-go", "k8s.io/apiextensions"} {
		if strings.Contains(lc, k8sDep) {
			return ClassK8sCompat, "Kubernetes compatibility dependency"
		}
	}

	// controller-runtime (K8s reconciliation semantics).
	if strings.Contains(lc, "controller-runtime") {
		return ClassControllerRuntime, "controller-runtime reconciliation dependency"
	}

	// CRD/schema dependencies.
	for _, crdDep := range []string{"apiextensions", "crd", "schema"} {
		if strings.Contains(lc, crdDep) {
			return ClassCRDSchema, "CRD/schema dependency"
		}
	}

	// Security-sensitive libraries.
	for _, secDep := range []string{"crypto", "tls", "security", "jwt", "oauth"} {
		if strings.Contains(lc, secDep) {
			return ClassSecurity, "security-sensitive dependency"
		}
	}

	// Signing/provenance/supply-chain.
	for _, s := range []string{"sigstore", "cosign", "provenance", "sbom", "syft"} {
		if strings.Contains(lc, s) {
			return ClassSigning, "signing/provenance dependency"
		}
	}

	// Build/release infrastructure.
	for _, b := range []string{".github/workflows", "Makefile", "Dockerfile", "release"} {
		for _, p := range changedPaths {
			if strings.Contains(strings.ToLower(p), strings.ToLower(b)) {
				return ClassBuildInfra, "build/release infrastructure change"
			}
		}
	}

	// Human contribution: always human approval.
	if !strings.Contains(strings.ToLower(author), "dependabot") &&
		!strings.Contains(strings.ToLower(author), "renovate") {
		return ClassHumanContrib, "human contributor PR"
	}

	// Default for known dependency bots: classify by bump type.
	if strings.Contains(lc, "major") {
		return ClassUnknown, "major version bump (protected by default)"
	}
	return ClassOrdinaryPatch, "ordinary dependency update"
}

// AutoMergeDecision is the final authority decision (semantic class
// overrides all other signals — tests green does NOT grant authority).
type AutoMergeDecision struct {
	Allowed  bool
	Semantic SemanticClass
	Reason   string
}

// EvaluateAutoMerge is the authoritative decision: the semantic class
// DENIES auto-merge for protected classes even if the bot is trusted,
// the diff is small, and all tests are green.
func EvaluateAutoMerge(author string, sem SemanticClass, reason string) AutoMergeDecision {
	switch sem {
	case ClassOrdinaryPatch:
		return AutoMergeDecision{Allowed: true, Semantic: sem, Reason: reason}
	case ClassOrdinaryMinor:
		return AutoMergeDecision{Allowed: true, Semantic: sem, Reason: reason}
	case ClassGoToolchain, ClassContainerBase, ClassK8sCompat,
		ClassCRDSchema, ClassControllerRuntime, ClassSecurity,
		ClassSigning, ClassBuildInfra, ClassHumanContrib, ClassUnknown:
		return AutoMergeDecision{Allowed: false, Semantic: sem, Reason: "protected semantic class: " + string(sem)}
	default:
		return AutoMergeDecision{Allowed: false, Semantic: sem, Reason: "unknown semantic class: " + reason}
	}
}

// semMajor extracts the major version component from a version string.
func semMajor(v string) string {
	parts := strings.SplitN(strings.TrimPrefix(v, "v"), ".", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
