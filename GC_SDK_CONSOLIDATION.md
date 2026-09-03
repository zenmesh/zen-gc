# GC SDK-First Consolidation (tranche 010)

zen-gc previously carried local copies of two zen-sdk/pkg/gc primitives:
internal/backoff and internal/ratelimiter. SDK-first review found the
implementations functionally identical (only the copyright header differed).

Change (tranche 010): imports swapped to zen-sdk/pkg/gc/{backoff,ratelimiter};
local packages deleted. The SDK is pinned to the same pseudo-version used by
zen-fabric/zen-identity (v0.7.3-ac2c4c2.0.20260830235011-72143d2032f3).

Full matrix: SDK_REUSE_MATRIX.md. Guard: do not re-create local packages
duplicating zen-sdk/pkg/gc/* primitives.
