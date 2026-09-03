# GC Policy API Compatibility (ZenGCPolicy)

## Kinds

| API | Kind | Group/Version | Status |
|---|---|---|---|
| Legacy | GarbageCollectionPolicy | gc.ops.zen-mesh.io/v1alpha1 | SERVED (execution projection target) |
| Zen-named | ZenGCPolicy | gc.ops.zen-mesh.io/v1alpha1 | SERVED (preferred public surface) |

## Migration model

The ZenGCPolicy adapter controller mirrors ZenGCPolicy objects 1:1 onto legacy
GarbageCollectionPolicy objects (same namespace, same name, annotated
`gc.ops.zen-mesh.io/projected-from: zengcpolicy`). The legacy kind remains the
execution projection target: exactly ONE reconciler executes each logical
policy, so duplicate execution is impossible by construction.

## Coexistence rules

- legacy only: works (unchanged; user-owned objects without the projection
  annotation are never touched by the adapter).
- Zen only: works (adapter projects onto legacy; legacy reconciler executes).
- both, distinct policies: both work.
- both, same logical policy: NOT two executions. The same namespace/name pair
  is one logical policy identity; the Zen object owns the projection.

## Migration recipe

1. Create a ZenGCPolicy with the same namespace/name and spec as your legacy
   object.
2. The adapter mirrors it onto the legacy object (or asserts an existing one).
3. Delete your legacy object when you are ready — the Zen object remains the
   source. Deleting the ZEN object removes the projection too.

## Deprecation policy

The legacy kind is not removed in this change. No removal date is promised.
A future release may mark the legacy kind deprecated in-schema; removal would
be a separate, announced change.
