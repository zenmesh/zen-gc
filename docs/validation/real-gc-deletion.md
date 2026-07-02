# Real Deletion Validation Runbook

Validate that the zen-gc controller performs **real (non-dry-run) deletion**
of matched resources across all 4 canonical TTL modes and supported GVRs.

## Prerequisites

- **kind** v0.20+ (for kind-based validation)
- **k3d** v5.8+ (for k3d-based validation)
- **Docker** (for building images)
- **Go** 1.26+ (for building the controller binary)
- **kubectl**
- **openssl** (for webhook TLS generation, even though webhook is disabled)

## Quick Start

```bash
# kind — validates with all 4 TTL modes × Pod + ReplicaSet (8 scenarios)
./scripts/validation/validate-real-gc-deletion.sh kind /tmp/zen-gc-kind

# k3d
./scripts/validation/validate-real-gc-deletion.sh k3d /tmp/zen-gc-k3d

# kubeadm (run on cluster node or with remote kubeconfig)
./scripts/validation/validate-real-gc-deletion.sh kubeadm /tmp/zen-gc-kubeadm
```

## What It Tests

| # | TTL Mode | Resource Kind | Validates |
|---|----------|---------------|-----------|
| 1 | Fixed (`SecondsAfterCreation`) | Pod | Deletion after N seconds |
| 2 | Dynamic (`FieldPath` as int64) | Pod | Reads TTL from annotation |
| 3 | Mapped (`FieldPath` + Mappings) | Pod | Maps severity label to TTL |
| 4 | Relative (`RelativeTo` + `SecondsAfter`) | Pod | Deletion after timestamp |
| 5-8 | Same 4 TTL modes | ReplicaSet | Same invariants |

Each scenario creates:
1. **Matching resource** (correct labels) — must be deleted
2. **Control resource** (wrong labels, same namespace) — must survive
3. **Other-namespace control** (correct labels, different NS) — must survive

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KEEP_CLUSTER` | (unset) | Set to `1` to leave cluster running |
| `GC_INTERVAL` | `20s` | Controller GC evaluation interval |
| `TTL_SHORT` | `15` | TTL in seconds for validation |
| `TAG` | `zenmesh/zen-gc-controller:validate` | Controller image tag |

## Interpreting Results

- `Resources matched ≥ 1` — controller found and matched resources
- `Resources deleted ≥ 1` — controller actually deleted them
- Matching resource `Not Found` — real deletion confirmed
- Control resources still exist — namespace/label scoping correct
- Other-namespace resource still exists — namespace isolation correct

## Known Issues

### Status counter reset (cosmetic)
The `resourcesMatched`/`resourcesDeleted` status counters may reset mid-cycle for
some resource kinds (notably ReplicaSet and dynamic-field Pod evaluations). The
actual deletion still occurs and is verified independently via controller logs
and `kubectl get`. See Bug #3 in the validation report.

### Fixed bugs
Three bugs were identified and fixed during this validation cycle:
1. **Evaluation service singleton** — `getOrCreateEvaluationService` cached a single
   service per controller process, keyed by the first policy's GVR. All subsequent
   policies with different targets received wrong `matched=0` results. Fixed by
   keying the cache by target resource (apiVersion/kind/namespace).
2. **Relative TTL never deletes** — `ErrRelativeTTLExpired` was treated as "no TTL"
   in `shouldDelete()`, preventing deletion of already-expired resources. Fixed by
   checking for `errors.Is(err, sdkttl.ErrRelativeTTLExpired)` and returning
   `ReasonTTLExpired`.
3. **Field path dot splitting** — `parseFieldPath` split on all dots, breaking
   annotation keys containing dots (e.g., `gc.ops.zen-mesh.io/ttl-seconds`). Fixed
   with backslash-escaped dot support (`\.`).

## Architecture Notes

- Controller uses **dynamic informers** (`dynamicinformer.DynamicSharedInformerFactory`),
  supporting any GVR at runtime without code generation.
- GC interval default is 1m in production; validation uses 20s for fast feedback.
- Webhook is disabled during validation to avoid TLS certificate dependencies.
- Leader election is enabled but single-replica for simplicity.
