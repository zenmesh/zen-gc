# Kubernetes v1.36 Validation Evidence

This directory contains validation evidence for zen-gc on Kubernetes 1.36 environments.
Each subdirectory or file represents a specific cluster provisioning method.

## Summary

| Method | K8s Version | Status | Scope |
|--------|-------------|--------|-------|
| [kind](kind.md) | v1.36.1 | PASS | CRD + runtime |
| [k3d (K3s)](k3d.md) | v1.36.2+k3s1 | PASS | CRD + runtime |
| [kubeadm](kubeadm.md) | v1.36.2 | PASS | CRD/API + negative + RBAC + controller runtime + GC behavior (containerd 2.2.5 upgraded from Debian's 1.7.24) |

## Scope

Validated environments are **not** a guarantee of correctness across all
Kubernetes 1.36 distributions. Operators should validate in their own
environment.

### What was validated

- Controller startup and leader election
- CRD registration (`GarbageCollectionPolicy`)
- Policy creation with dry-run behavior
- Resource matching and status reporting
- No crash loops, panics, or API errors

### What was NOT validated

- EKS, GKE, AKS, OpenShift, Rancher, or any managed Kubernetes distribution
- All CNI plugins
- All storage providers
- Webhook admission (certificate setup is environment-dependent)
- Non-dry-run (actual deletion) behavior
- Performance under load
- Network policies or restrictions

## Repo

- **Repo**: `zenmesh/zen-gc`
- **Commit**: `451a95c`
- **Controller image**: `zenmesh/zen-gc-controller:v0.0.1-alpha-4be11fe` (build from commit, statically linked, scratch base)
- **Go**: 1.26.4
- **client-go**: v0.35.0
- **controller-runtime**: v0.19.0
