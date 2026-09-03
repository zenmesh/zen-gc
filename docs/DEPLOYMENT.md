# zen-gc Deployment

## SDK reuse model
- Backoff/rate limiting: zen-sdk/pkg/gc/{backoff,ratelimiter} (consolidated 010)
- TTL evaluation: zen-gc internal (superset of SDK; escaped dots + relative TTL)
- Selectors/field paths: SDK pkg/gc/{selector,fieldpath} for unstructured access
- Leader election: controller-runtime manager (SDK leader package inspected; NOT_APPLICABLE — controller-runtime is the semantic fit)
- Health/logging/metrics: zen-gc internal packages (product-scoped; no local generic framework)

## Profiles
One binary, one chart, two deployment profiles (placement, not ownership):

| Profile | Target kinds | RBAC scope | Use |
|---|---|---|---|
| control | control-plane transient resources | namespace-scoped Role | control-plane namespaces |
| data | data-plane transient resources | namespace-scoped Role | data-plane namespaces |

## Safety
- Default deny: unknown GVK is never GC-able
- Explicit allowlist per profile
- Protection annotation: gc.ops.zen-mesh.io/protected: "true" -> never deleted
- Dry-run: full selection/planning without mutation
- Bounded batch: max deletions per cycle configurable
- Finalizers respected: BLOCKED_BY_FINALIZER recorded, never stripped
- Native K8s GC preferred where sufficient (TTLAfterFinished, ownerReferences)

## Authority boundary
zen-gc does NOT delete: Identity credentials, Trust evidence, Lock custody,
Fabric Plane state, customer payloads, database rows. Resource GC only.
