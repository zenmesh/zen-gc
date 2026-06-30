# k3d (K3s) — Validation Evidence

## Environment

| Field | Value |
|-------|-------|
| **k3d version** | v5.8.3 |
| **K3s image** | `rancher/k3s:v1.36.2-k3s1` |
| **Kubernetes** | v1.36.2+k3s1 |
| **Container runtime** | containerd://2.3.2-k3s2 |
| **Go (K3s)** | go1.26.4 |
| **OS image** | K3s v1.36.2+k3s1 |

## Validation Date

2026-06-29

## Procedure

1. Created k3d cluster `zen-gc-k8s136` with `rancher/k3s:v1.36.2-k3s1` (single server node, no agents).
2. Imported controller image via `k3d image import`.
3. Applied CRDs, namespace, RBAC, deployment (2 replicas, leader election, self-signed webhook cert), and service.
4. Waited for controller Ready (leader replica).
5. Created validation namespace `gc-validation`.
6. Created test ConfigMap.
7. Created `GarbageCollectionPolicy` with `dryRun: true`, targeting ConfigMaps, TTL 30s.
8. Confirmed policy Active with ResourcesMatched > 0.
9. Confirmed controller logs showed no errors.
10. Cleaned up: deleted policy, namespace.

## Results

| Check | Result |
|-------|--------|
| Cluster created | PASS |
| CRDs installed | PASS |
| Controller deployed | PASS |
| Pod Ready (leader) | PASS |
| Policy created | PASS |
| Policy status Active | PASS |
| Resources matched | PASS (2 ConfigMaps) |
| Resources deleted (dry-run) | 0 (correct) |
| Controller logs — errors | 0 |
| Controller crash loops | 0 (leader replica) |
| Image import | PASS (`k3d image import`) |
| Cleanup | PASS |

## Controller Logs (leader)

```
{"msg":"Controller configuration","gcInterval":"1m0s","maxDeletionsPerSecond":10,"batchSize":50,"maxConcurrentEvaluations":5}
{"msg":"Leader election enabled","electionID":"gc-controller-leader-election","namespace":"gc-system"}
"Attempting to acquire leader lease..."
"Successfully acquired lease"
"Started leading"
{"msg":"Webhook server starting with TLS","address":":9443"}
{"msg":"Starting GC controller manager","operation":"start"}
{"msg":"Starting Controller","controller":"garbagecollectionpolicy"}
{"msg":"Starting workers","worker count":1}
```

## Limitations

- Single-server k3d cluster only (no agent nodes).
- Webhook uses self-signed certs; TLS verification not tested.
- Dry-run only; actual deletion not validated.
- K3s bundles its own components (flannel, CoreDNS, metrics-server, local-path-provisioner); behavior may differ on non-K3s distributions.

## Evidence Files

- `k3d.json` — structured evidence data.
