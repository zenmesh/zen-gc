# zen-gc — Kubernetes Compatibility

This document summarizes what is **evidenced** about zen-gc's Kubernetes compatibility. Claims here are grounded in the repo's actual dependencies, tests, and CI configuration — not assumed.

## Compilation compatibility

zen-gc is built with:

- `client-go v0.35` — targets Kubernetes 1.31.x API surface
- `controller-runtime v0.19` — uses envtest for integration tests
- `apiextensions-apiserver v0.31` (indirect) for CRD support
- CRD API version: `apiextensions.k8s.io/v1`

Build requirement: **Go 1.26+** (see `go.mod`).

## Tested configurations

| Configuration | CI | Notes |
|---------------|----|-------|
| Unit tests (envtest) | ✅ `go test ./...` | Uses controller-runtime envtest (fake API server) |
| Integration tests | ✅ `test/integration/` | Runs against envtest-backed API |
| kind cluster (local) | ⚠️ `make e2e-kind` | Available but not run in CI |

## Not evidenced

The following are **not** actively tested or evidenced:

- Cloud Kubernetes (EKS, GKE, AKS)
- Kubernetes distributions (OpenShift, Rancher, k0s, etc.)
- Specific Kubernetes versions beyond compilation compatibility
- Network plugins (Calico, Cilium, Flannel)
- Storage providers (CSI drivers)
- ARM64 Kubernetes clusters
- Windows Kubernetes nodes

## Version support policy

zen-gc follows `client-go` compatibility: compiled against the latest stable Kubernetes API. Older API versions may work but are not actively tested. Breaking changes from upstream Kubernetes may affect behavior — report issues via GitHub.

## CRD API version

| CRD Version | Status | Notes |
|-------------|--------|-------|
| `v1alpha1` | ✅ Current | May change; not stable API |
| `v1beta1` | 🔜 Planned | No timeline |
| `v1` | 🔜 Planned | No timeline |

## See also

- `go.mod` — exact dependency versions
- `.github/workflows/ci.yml` — CI test matrix
- `docs/VERSION_COMPATIBILITY.md` — additional version migration documentation
