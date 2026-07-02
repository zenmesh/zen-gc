# zen-gc — Claims and Maturity

## What zen-gc does

zen-gc is a Kubernetes controller that provides **declarative garbage collection** for Kubernetes resources matched by `GarbageCollectionPolicy` CRDs. It:

- Watches `GarbageCollectionPolicy` CRDs and reconciles matching resources
- Supports **four TTL modes**: fixed (`secondsAfterCreation`), field-based dynamic (`fieldPath`), mapped (`fieldPath` + `mappings`), relative (`relativeTo` + `secondsAfter`)
- Supports **label selectors, field selectors, and conditions** (phase, labels, annotations, fields)
- Provides **rate limiting** (per-policy token bucket) and **dry-run** mode
- Emits **Prometheus metrics**, Kubernetes events, and structured logs
- Runs **leader election** (2+ replicas; multi-node HA runtime not validated)
- Runs **non-root** with **restricted** Pod Security Standards
- Compiles with **Go 1.26** using `controller-runtime v0.19`

### Validated deletion behavior (real, non-dry-run)

The following matrix has been validated with real deletion confirmed via controller logs:

| TTL Mode | Pod | ReplicaSet | ConfigMap | Secret | Job |
|----------|:---:|:----------:|:---------:|:-----:|:---:|
| Fixed (`secondsAfterCreation`) | ✅ kind, k3d | ✅ kind, k3d | ✅ kind | ✅ kind | ✅ kind |
| Field-based dynamic (`fieldPath`: int64) | ✅ kind, k3d | — | — | — | — |
| Mapped (`fieldPath` + `mappings`) | ✅ kind, k3d | ✅ kind, k3d | ✅ kind | ✅ kind | ✅ kind |
| Relative (`relativeTo` + `secondsAfter`) | ✅ kind, k3d | — | — | — | — |

See `docs/evidence/kubernetes/v1.36/kind.md` and `docs/evidence/kubernetes/v1.36/k3d.md` for full matrix detail.

**Not validated**: Cloud K8s, OpenShift, Rancher, older Kubernetes versions, multi-node HA, webhook admission, performance under load.

## What zen-gc does NOT do

- Does **not** implement Kubernetes-native TTL (that is upstream's job — see [TTL-after-finished](https://kubernetes.io/docs/concepts/workloads/controllers/ttlafterfinished/))
- Does **not** enforce admission policy (use OPA/Gatekeeper/Kyverno)
- Does **not** manage secrets or credentials
- Does **not** provide a SaaS control plane
- Does **not** provide multi-tenant isolation (runs as a single controller)
- Does **not** provide cross-cluster garbage collection
- Does **not** replace `kubectl delete` for ad-hoc cleanup

## Current maturity

| Attribute | Status | Evidence |
|-----------|--------|----------|
| API version | `v1alpha1` (may change) | CRD manifests |
| Semantic version | `0.0.1-alpha` | git tags, Helm chart |
| Unit tests | ✅ 65%+ coverage gate | CI (`make coverage`) |
| Integration tests | ✅ Run in CI | `test/integration/` |
| E2E tests | ⚠️ Optional (`make e2e-kind`) | Not run in CI |
| Race detection | ✅ `go test -race` | CI test job |
| Vulnerability scan | ✅ `govulncheck` + `gosec` | CI security job |
| Container image | ✅ Multi-arch (amd64, arm64) | CI build job |
| Leader election safety (kind) | ✅ Validated (2+3 replicas) | `docs/evidence/kubernetes/v1.36/leader-election.md` |
| SLSA/provenance | ❌ Not published | — |
| SBOM | ❌ Not generated | — |
| Fuzz testing | ❌ Not implemented | — |
| Penetration test | ❌ Not performed | — |
| Cloud K8s E2E | ❌ Not evidenced | — |

## Supported environments

zen-gc is built with `client-go v0.35` targeting Kubernetes 1.31.x API. The CI test suite runs on `ubuntu-latest` GitHub runners with no cloud Kubernetes cluster. Compatibility with specific Kubernetes distributions or cloud providers is **not** actively evidenced.

**Evidenced:**
- Go build + unit + integration tests pass
- Validated on **kind** with Kubernetes v1.36.1 (`docs/evidence/kubernetes/v1.36/kind.md`)
- Validated on **k3d (K3s)** with Kubernetes v1.36.2+k3s1 (`docs/evidence/kubernetes/v1.36/k3d.md`)
- Validated on **kubeadm** with Kubernetes v1.36.2 (Debian 13, containerd 2.2.5) (`docs/evidence/kubernetes/v1.36/kubeadm.md`)
- Validated on **kubeadm** with Kubernetes v1.34.9 (Debian 13, containerd 2.2.5) (`docs/evidence/kubernetes/v1.34/kubeadm.md`)

**Not evidenced:** Behavior on EKS, GKE, AKS, OpenShift, Rancher, or any Kubernetes version other than the specific validated environments above.

## Security claims

- ✅ Non-root container execution (security context enforced)
- ✅ Restricted Pod Security Standards compatible
- ✅ Read-only root filesystem by default
- ✅ Regular vulnerability scanning via `govulncheck` and `gosec`
- ✅ Dependency updates via Dependabot (Go modules, Docker, GitHub Actions)
- ⚠️ Audit logging at application level (no runtime audit integration)
- ❌ No formal security audit or penetration test

## Non-goals

- zen-gc is **not** an enterprise product — it is a community OSS project
- zen-gc is **not** production-live or commercially launched
- zen-gc is **not** a managed/hosted service
- zen-gc is **not** a replacement for Kubernetes garbage collection
- zen-gc is **not** tested against every Kubernetes version or distribution

## Experimental/planned areas

- `v1beta1` CRD API — planned after API stabilization
- Controller-runtime-based webhook validation — partially implemented
- Artifact Hub helm chart — published for 0.0.1-alpha

## Evidence policy

All claims in this file are either:

1. **Backed by evidence** in `docs/evidence/` with specific test or CI results, or
2. **Explicitly marked as missing** (❌), or
3. **Explicitly marked as planned** (🔜)

Do not infer unlisted claims. If evidence does not exist, assume the claim is unsupported.
