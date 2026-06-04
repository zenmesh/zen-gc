# zen-gc

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![CI](https://github.com/zen-mesh/zen-gc/workflows/CI/badge.svg)](https://github.com/zen-mesh/zen-gc/actions)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.26+-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)

Kubernetes controller for **declarative garbage collection**: define **`GarbageCollectionPolicy`** objects (TTL, selectors, dry-run, rate limits) and let zen-gc delete matching resources safely.

**zen-gc** is open source (Apache-2.0) from **[Zen Mesh Inc.](https://zen-mesh.io)**. The commercial **[Zen Mesh](https://zen-mesh.io)** offering is separate; this repository is meant for community use, self-managed clusters, and contributions.

**Security:** report vulnerabilities to **[security@zen-mesh.io](mailto:security@zen-mesh.io)** or via [GitHub Security Advisories](https://github.com/zen-mesh/zen-gc/security) — see [SECURITY.md](SECURITY.md).

Builds require **Go 1.26+** (see `go.mod`). Published container images are built with Go **1.26**.

## Quick start

### With kubectl

From a clone of this repo (paths match the tree layout):

```bash
kubectl apply -f deploy/crds/gc.kube-zen.io_garbagecollectionpolicies.yaml
kubectl apply -f deploy/manifests/namespace.yaml
kubectl apply -f deploy/manifests/rbac.yaml
# Point deploy/manifests/deployment.yaml at your image (build locally or use a registry tag).
kubectl apply -f deploy/manifests/deployment.yaml
```

The validating admission webhook requires **TLS** in the cluster before you rely on admission enforcement. **Recommended path:** cert-manager + Secret `gc-controller-webhook-cert` + `deploy/webhook/validating-webhook.yaml`. Full steps: **[docs/WEBHOOK_TLS.md](docs/WEBHOOK_TLS.md)** (includes the manual-CA flow used by `scripts/comprehensive_e2e.sh` on kind).

Apply an example policy:

```bash
kubectl apply -f examples/temp-configmap-cleanup.yaml
```

**Remote raw manifests** (replace branch/tag as needed):

```bash
kubectl apply -f https://raw.githubusercontent.com/zen-mesh/zen-gc/main/deploy/crds/gc.kube-zen.io_garbagecollectionpolicies.yaml
kubectl apply -f https://raw.githubusercontent.com/zen-mesh/zen-gc/main/deploy/manifests/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/zen-mesh/zen-gc/main/deploy/manifests/rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/zen-mesh/zen-gc/main/deploy/manifests/deployment.yaml
kubectl apply -f https://raw.githubusercontent.com/zen-mesh/zen-gc/main/examples/temp-configmap-cleanup.yaml
```

### With Helm

```bash
helm repo add zen-gc https://zen-mesh.github.io/zen-gc
helm repo update
helm install gc-controller zen-gc/gc-controller --version 0.0.1-alpha --namespace gc-system --create-namespace
kubectl apply -f https://raw.githubusercontent.com/zen-mesh/zen-gc/main/examples/temp-configmap-cleanup.yaml
```

### Minimal policy example

```yaml
apiVersion: gc.ops.zen-mesh.io/v1alpha1
kind: GarbageCollectionPolicy
metadata:
  name: cleanup-temp-configmaps
spec:
  targetResource:
    apiVersion: v1
    kind: ConfigMap
    labelSelector:
      matchLabels:
        temporary: "true"
  ttl:
    secondsAfterCreation: 3600
  behavior:
    maxDeletionsPerSecond: 10
```

## Why zen-gc?

Kubernetes only provides built-in TTL support for **Jobs**. For ConfigMaps, Secrets, Pods, CRDs, and everything else, you either write bespoke controllers or operate without automation. zen-gc fills that gap with plain YAML policies.

## Powerful TTL options

zen-gc supports **fixed**, **field-based**, **mapped**, and **relative** TTL modes — see [User Guide](docs/USER_GUIDE.md) for details.

### Fixed TTL

```yaml
ttl:
  secondsAfterCreation: 604800
```

### Field-based TTL

```yaml
ttl:
  fieldPath: "spec.ttlSeconds"
```

### Mapped TTL

```yaml
ttl:
  fieldPath: "spec.severity"
  mappings:
    CRITICAL: 1814400
    HIGH: 1209600
    MEDIUM: 604800
    LOW: 259200
  default: 604800
```

### Relative TTL

```yaml
ttl:
  relativeTo: "status.lastProcessedAt"
  secondsAfter: 86400
```

## Key benefits

- Works with any Kubernetes resource type the API server exposes
- Policies as CRDs — no custom binaries per workload
- Rate limiting, metrics, events, leader election for HA
- Dry-run mode to validate behavior before destructive deletes
- Observable via Prometheus and Kubernetes events

## Use cases

- Clean up completed Jobs and temporary CI/CD ConfigMaps/Secrets
- Remove evicted Pods, orphaned ReplicaSets, released PVCs
- Namespace-scoped or cluster-scoped policies with selectors

## Documentation

- **[Webhook TLS (production)](docs/WEBHOOK_TLS.md)**: cert-manager vs manual CA — matches how we test on kind
- **[Linting](docs/LINTING.md)**: golangci-lint scope and known debt
- **[KEP-style design](docs/KEP_GENERIC_GARBAGE_COLLECTION.md)**: background and API notes
- **[API Reference](docs/API_REFERENCE.md)**
- **[User Guide](docs/USER_GUIDE.md)**
- **[Operator Guide](docs/OPERATOR_GUIDE.md)**
- **[Metrics](docs/METRICS.md)**
- **[Security (operations)](docs/SECURITY.md)** and [SECURITY.md](SECURITY.md) (disclosure policy)
- **[Secret / TLS storage](docs/SECRET_MANAGEMENT.md)**
- **[Version compatibility](docs/VERSION_COMPATIBILITY.md)**
- **[Architecture](docs/ARCHITECTURE.md)**
- **[Examples](examples/)**
- **[Contributing](CONTRIBUTING.md)**
- **[Governance](GOVERNANCE.md)** · **[Maintainers](MAINTAINERS.md)** · **[Releasing](RELEASING.md)** · **[Adopters](ADOPTERS.md)**

## Features

- Generic resource support (including CRDs)
- Four TTL modes: fixed, field-based, mapped, relative
- Label / field selectors and conditions
- Rate limiting and dry-run
- Prometheus metrics, Kubernetes events, HA via leader election
- Test suite with CI and optional kind e2e (`make e2e-kind`)

## Status

zen-gc is actively maintained as OSS. Feedback and contributions are welcome.

## References

- [Kubernetes TTL-after-finished](https://kubernetes.io/docs/concepts/workloads/controllers/ttlafterfinished/)
- [Kubernetes Enhancement Proposals](https://github.com/kubernetes/enhancements)

## Links

- **Zen Mesh Inc.**: [zen-mesh.io](https://zen-mesh.io)
- **Helm (Artifact Hub)**: [gc-controller chart](https://artifacthub.io/packages/helm/zengc/gc-controller)
- **Explainer video**: [YouTube](https://www.youtube.com/watch?v=P8afhcgjWVQ&list=PL1AGc_sKXJBdInu0yffTJxN828oaCuqwx)
