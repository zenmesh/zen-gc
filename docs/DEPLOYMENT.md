# zen-gc Deployment

zen-gc is a standalone Kubernetes garbage-collection controller. It is
installed and configured entirely through generic Kubernetes mechanisms:
the `GarbageCollectionPolicy` CRD, RBAC, and Helm values. It has no
knowledge of any specific application architecture placed on top of it.

## Install

### 1. CRDs

```bash
kubectl apply -f deploy/crds/gc.ops.zen-mesh.io_garbagecollectionpolicies.yaml
```

### 2. Controller

Helm:

```bash
helm upgrade --install gc-controller deploy/helm/zen-gc \
  --namespace gc-system --create-namespace
```

or the install script:

```bash
./install.sh --method kubectl   # or --method helm
```

## Configuration

| Value | Purpose |
|---|---|
| `image.repository/tag/digest` | Controller image |
| `rbac.targetKinds` | Explicit allowlist of resource classes the controller may collect (default deny; no wildcards) |
| `rbac.namespaceScope` | `true` = Role in the release namespace; `false` = ClusterRole (cluster-wide) |
| `leaderElection.enabled` | controller-runtime leader election |
| `networkPolicy.enabled` | Egress-restricted NetworkPolicy (Kubernetes API + DNS only) |

Scope the controller by granting only the namespaces and resource classes
it should manage. For namespace-scoped deployments, install one release per
namespace with `rbac.namespaceScope: true` and an explicit
`rbac.targetKinds` allowlist.

## Safety

- Default deny: an unknown GVK is never GC-able
- Explicit resource-class allowlist via `rbac.targetKinds`
- Protection annotation: `gc.ops.zen-mesh.io/protected: "true"` means never deleted
- Dry-run: full selection/planning without mutation
- Bounded batch: max deletions per cycle configurable
- Finalizers respected: blocked deletions are recorded, never stripped
- Native Kubernetes GC is preferred where sufficient (TTL-after-finished,
  ownerReferences)

## Authority boundary

zen-gc deletes only the Kubernetes resources its policies name within its
RBAC grant. It never touches anything else: application databases, secrets
outside its grant, or resources in namespaces it cannot reach.
