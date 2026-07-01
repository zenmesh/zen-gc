# Kubernetes v1.34 Validation Evidence

This directory contains validation evidence for zen-gc on Kubernetes 1.34
environments. **Scope: CRD/API compatibility only** — runtime controller
behavior is not claimed for v1.34.

## Summary

| Method | K8s Version | Status | Scope |
|--------|-------------|--------|-------|
| [kubeadm](kubeadm.md) | v1.34.9 | PASS | CRD/API compatibility |
| kind | — | ❌ Not tested | — |
| k3d/K3s | — | ❌ Not tested | — |

## What was validated

- CRD registration and install idempotency
- API resource discovery (short names, group, version)
- CRUD lifecycle: create, list, read, delete
- Schema validation (type enforcement)
- All spec fields: targetResource, ttl, conditions, behavior
- Dry-run behavior configuration

## What was NOT validated on v1.34

- Runtime controller reconciliation
- kind or k3d/K3s
- EKS, GKE, AKS, OpenShift, Rancher
- Webhook admission
- Non-dry-run (actual deletion) behavior
- Performance under load

For runtime-controller evidence, see the [v1.36 reports](../v1.36/).

## Repo

- **Repo**: `zenmesh/zen-gc`
- **Commit**: `ee5fea2`
