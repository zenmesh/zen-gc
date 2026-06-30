# kind — Validation Evidence

## Environment

| Field | Value |
|-------|-------|
| **kind version** | v0.20.0 (go1.20.4, linux/amd64) |
| **Node image** | `kindest/node:v1.36.1` |
| **Kubernetes** | v1.36.1 |
| **Container runtime** | containerd://2.3.1 (Debian GNU/Linux 13 (trixie), kernel 6.17.0-35) |

## Validation Date

2026-06-29

## Procedure

1. Created kind cluster `zen-gc-k8s136` with `kindest/node:v1.36.1` (single control-plane node).
2. Built controller image locally (statically linked, scratch base) and loaded into kind via `ctr images import`.
3. Applied CRDs (`garbagecollectionpolicies.gc.ops.zen-mesh.io`).
4. Created namespace `gc-system`, RBAC, and deployment (2 replicas, leader election enabled, self-signed webhook cert).
5. Waited for controller Ready.
6. Created validation namespace `gc-validation`.
7. Created `GarbageCollectionPolicy` with `dryRun: true`, targeting ConfigMaps with 30s TTL.
8. Confirmed policy became Active with ResourcesMatched > 0.
9. Confirmed controller logs showed no errors or panics.
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
| Resources deleted (dry-run) | 0 (correct — dry-run) |
| Controller logs — errors | 0 |
| Controller crash loops | 0 (leader replica) |
| Cleanup | PASS |

## Controller Logs (leader)

```
I0630 01:23:06.523550       1 leaderelection.go:272] "Successfully acquired lease"
I0630 01:23:06.523720       1 election.go:196] "Started leading"
{"msg":"Webhook server starting with TLS"}
{"msg":"Starting GC controller manager","operation":"start"}
{"msg":"Starting Controller","controller":"garbagecollectionpolicy"}
{"msg":"Starting workers","worker count":1}
```

## Limitations

- Only one control-plane node tested (single-node kind cluster).
- Webhook uses self-signed certs; TLS verification not tested.
- Dry-run only; actual deletion not validated.
- kind v0.20.0 is older than the node image; manual `ctr images import` was required (kind `load docker-image` failed due to containerd snapshotter incompatibility).

## Evidence Files

- `kind.json` — structured evidence data.
