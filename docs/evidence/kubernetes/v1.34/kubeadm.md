# kubeadm — Validation Evidence (K8s v1.34.9)

## Status: PASS

zen-gc CRD (`GarbageCollectionPolicy`) has been validated against Kubernetes
v1.34.9 provisioned via kubeadm on a Debian 13 VM.

**Scope**: CRD/API compatibility only. kind and k3d/K3s were **not tested**
on v1.34. Full controller-runtime proof is not claimed for v1.34.

## VM Configuration

| Field | Value |
|-------|-------|
| **Hostname** | `h462-gateway-kubeadm-1780668538` |
| **IP** | 192.168.122.179 |
| **Libvirt domain** | `h462-gateway-kubeadm-1780668538` |
| **OS** | Debian 13 (trixie) |
| **Kernel** | 6.12.74+deb13+1-amd64 |
| **RAM** | 4 GB |
| **vCPUs** | 2 |
| **Containerd** | 1.7.24 (Debian repos) |
| **CNI** | Flannel v0.28.5 |
| **Kubeadm/Kubelet/Kubectl** | v1.34.9 |

## Evidence

### kubeadm Version
```
kubeadm version: v1.34.9
  GitCommit:"ad7c7374b74c04d07ea041d367ecb1a526bdf758"
  GoVersion:"go1.25.11"
```

### CRD Registration
```
$ kubectl get crds garbagecollectionpolicies.gc.ops.zen-mesh.io
NAME                                           CREATED AT
garbagecollectionpolicies.gc.ops.zen-mesh.io   2026-07-01T11:40:34Z
```

### API Resource Discovery
```
$ kubectl api-resources --api-group=gc.ops.zen-mesh.io
NAME                        SHORTNAMES     APIVERSION                    NAMESPACED   KIND
garbagecollectionpolicies   gcp,gcpolicy   gc.ops.zen-mesh.io/v1alpha1   true         GarbageCollectionPolicy
```

### CRUD Lifecycle
| Operation | Result |
|-----------|--------|
| Create minimal GCP | ✅ |
| Create full-schema GCP (all spec fields) | ✅ |
| List GCPs | ✅ |
| Read GCP YAML | ✅ |
| Delete GCP | ✅ |
| Schema validation (wrong types) | ✅ |

## Known Issues

### Control-Plane Stability (v1.34)
The v1.34.9 cluster exhibited periodic crash-loops of control-plane components
every 1–5 minutes. Root cause: containerd sandbox_image mismatch — kubeadm
expects `pause:3.10.1`, Debian containerd defaults to `pause:3.8`. Fixed by
setting `sandbox_image = "registry.k8s.io/pause:3.10.1"` in
`/etc/containerd/config.toml` and performing a nuclear reset. The fix restores
stability temporarily but backoff accumulation eventually triggers re-cycling.

## What Was NOT Validated on v1.34

- ❌ Runtime controller behavior (pods Pending due to instability)
- ❌ kind or k3d/K3s on v1.34
- ❌ Webhook admission
- ❌ Non-dry-run (actual deletion) behavior

For runtime-controller evidence, see the [v1.36 kind](/v1.36/kind.md) and
[k3d](/v1.36/k3d.md) reports.
