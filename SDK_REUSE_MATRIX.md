# GC SDK Reuse Matrix (tranche 010, SDK-first review)

Source of truth: zen-sdk@72143d2032f3 (pseudo-version v0.7.3-ac2c4c2.0.20260830235011-72143d2032f3, pinned in go.mod like zen-fabric/zen-identity).

| zen-gc path | symbol/component | zen-sdk candidate | classification | reason | action |
|---|---|---|---|---|---|
| internal/backoff | Backoff, Config | pkg/gc/backoff.Backoff/Config | REUSE (DONE 010) | line-for-line identical (only copyright header differed) | imports swapped to SDK; local package deleted |
| internal/ratelimiter | RateLimiter | pkg/gc/ratelimiter.RateLimiter | REUSE (DONE 010) | line-for-line identical | imports swapped to SDK; local package deleted |
| internal/ttl | age/TTL evaluation | pkg/gc/ttl.Spec/IsExpired | REUSE candidate | SDK TTL evaluator covers creation-field + fixed expiry | follow-up migration; internal/ttl retained until parity proven |
| pkg/controller/selector handling | label/annotation/field conditions | pkg/gc/selector.Conditions/Matches* | REUSE candidate | SDK selector matcher is the same model | follow-up migration |
| pkg/controller/field_path.go | field path extraction | pkg/gc/fieldpath.Get* | REUSE candidate | identical extraction semantics | follow-up migration |
| ZenGCPolicy CRD + compat adapter (244b061) | product semantics | n/a | PRESERVE_WITH_ADAPTER | GC product/controller semantics are zen-gc-owned per SDK ownership law; NOT SDK material | keep in zen-gc |
| reconciliation/deletion orchestration | GC-specific executor | n/a | PRESERVE_WITH_ADAPTER | domain-specific deletion orchestration, rate-limited per policy | keep in zen-gc |
| pkg/controller/backoff.go | retry wrapping | pkg/gc/backoff | REUSE (primitive) | primitive now SDK-backed via import swap | done |
| leader election | controller-runtime leader elector | n/a (controller-runtime owns) | NOT_APPLICABLE | provided by controller-runtime manager | n/a |

## Guards

- New generic GC primitives (batch/age/retry/selector) must be added to zen-sdk/pkg/gc/* and imported, never re-created locally. Structural rule: `internal/` must not contain packages duplicating `zen-sdk/pkg/gc/*` names (backoff, ratelimiter, ttl, selector, fieldpath).

## Rejected migrations

- ZenGCPolicy CRD/adapter: product semantics, SDK ownership law forbids.
- Deletion orchestration: GC-domain logic, not generic.
