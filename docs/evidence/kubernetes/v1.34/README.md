# Kubernetes v1.34 Validation Evidence

This directory contains validation evidence for zen-gc on Kubernetes 1.34
environments. Each file represents a specific cluster provisioning method.

## Summary

| Method | K8s Version | Status |
|--------|-------------|--------|
| [kubeadm](kubeadm.md) | v1.34.9 | PASS |

## Scope

Validated environments are **not** a guarantee of correctness across all
Kubernetes 1.34 distributions. Operators should validate in their own
environment.

### What was validated

- CRD registration (`GarbageCollectionPolicy`)
- API resource discovery (short names, group, version)
- CRUD lifecycle: create, list, read, delete
- Schema validation (type enforcement)
- All spec fields: targetResource, ttl, conditions, behavior
- Dry-run behavior configuration

### What was NOT validated

- EKS, GKE, AKS, OpenShift, Rancher, or any managed Kubernetes distribution
- All CNI plugins (only flannel v0.28.5 tested)
- All storage providers
- Webhook admission (certificate setup is environment-dependent)
- Non-dry-run (actual deletion) behavior
- Performance under load
- Network policies or restrictions
- HA/multi-node control plane
- Operator/controller runtime behavior (pods Pending due to scheduler instability)

## Repo

- **Repo**: `zenmesh/zen-gc`
- **Commit**: `a632052`
