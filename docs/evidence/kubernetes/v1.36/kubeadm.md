# kubeadm — Validation Evidence (K8s v1.36.2)

## Status: PARTIAL

zen-gc CRD (`GarbageCollectionPolicy`) has been validated against Kubernetes
v1.36.2 provisioned via kubeadm on a Debian 13 VM.

**Scope**: CRD/API compatibility, CRUD lifecycle, schema validation, RBAC
boundaries, and install idempotency (confirmed PASS). Runtime controller
reconciliation is **not proven** on this VM due to control-plane instability
(see Known Issues).

## VM Configuration

| Field | Value |
|-------|-------|
| **Hostname** | `h462-gateway-kubeadm-1780668538` |
| **IP** | 192.168.122.179 |
| **Libvirt domain** | `h462-gateway-kubeadm-1780668538` |
| **OS** | Debian 13 (trixie) |
| **Kernel** | 6.12.74+deb13+1-amd64 |
| **RAM** | 10 GB (increased from 4→6→10 GB; cold boot to set max memory) |
| **vCPUs** | 2 |
| **Containerd** | 1.7.24 (Debian repos) |
| **CNI** | Flannel v0.28.5 |
| **Kubeadm/Kubelet/Kubectl** | v1.36.2 |

## Cluster Configuration

kubeadm `v1beta4` config, single control-plane node, flannel CNI with
`10.244.0.0/16` pod CIDR. Containerd sandbox_image set to `pause:3.10.1`
before init.

## Evidence

### CRD Install Idempotency
```
$ kubectl apply -f deploy/crds/gc.kube-zen.io_garbagecollectionpolicies.yaml
  customresourcedefinition.apiextensions.k8s.io/garbagecollectionpolicies.gc.ops.zen-mesh.io created

$ kubectl apply -f deploy/crds/gc.kube-zen.io_garbagecollectionpolicies.yaml
  customresourcedefinition.apiextensions.k8s.io/garbagecollectionpolicies.gc.ops.zen-mesh.io unchanged
```
Re-applying the CRD is safe — results in `unchanged`, no errors.

### API Resource Discovery
```
$ kubectl api-resources --api-group=gc.ops.zen-mesh.io
NAME                        SHORTNAMES     APIVERSION                    NAMESPACED   KIND
garbagecollectionpolicies   gcp,gcpolicy   gc.ops.zen-mesh.io/v1alpha1   true         GarbageCollectionPolicy
```

### CRUD Lifecycle
| Operation | Result |
|-----------|--------|
| Create minimal GCP | ✅ created |
| Create full-schema GCP (all spec fields) | ✅ created |
| List GCPs | ✅ appears with namespace/name/age |
| Read GCP YAML | ✅ all spec fields persisted |
| Delete GCP | ✅ deleted |
| GCPs after delete | ✅ empty list |

### Negative Schema Validation
| Test | Input | Result |
|------|-------|--------|
| Wrong type | `ttl.secondsAfterCreation: "3600"` (string) | ❌ Rejected: "must be of type integer" |
| Unknown field | `spec.nonexistentField: true` | ❌ Rejected: "unknown field" |
| Empty spec | `spec: {}` | ❌ Rejected: "targetResource: Required value" + "ttl: Required value" |
| Missing required | no `spec.targetResource` | ❌ Rejected: "targetResource: Required value" |
| Wrong array type | `conditions.phase: Succeeded` (string) | ❌ Rejected: "must be of type array" |

All invalid inputs are correctly rejected by the API server with descriptive
error messages — not by client-side tooling.

### RBAC / Permission Boundaries

The gc-controller service account permissions (from `deploy/manifests/rbac.yaml`):
```
$ kubectl auth can-i --list --as=system:serviceaccount:gc-system:gc-controller
  garbagecollectionpolicies.gc.ops.zen-mesh.io          → [get list watch]
  garbagecollectionpolicies.gc.ops.zen-mesh.io/status   → [get update patch]

$ kubectl auth can-i list pods -n gc-system --as=...
  yes

$ kubectl auth can-i delete pods --all-namespaces --as=...
  yes
```

The controller has read access to GCP resources, write access to GCP status,
and the ability to list/delete pods in its target namespace (required for
garbage collection).

## Known Issues

### Control-Plane Instability (Retried)
The kubelet continuously cycles `kube-controller-manager` and
`kube-scheduler` through CrashLoopBackOff every 1–5 minutes, despite:

- **VM RAM increased** from 4 GB → 6 GB (live via `virsh setmem`, confirmed
  inside VM as 5.8 GiB total / 3.9 GiB available)
- **VM rebooted** (uptime 23 min at post-reboot check)
- **Nuclear reset** (`crictl rm -a && crictl rmp -a && systemctl restart kubelet`)
- **Correct sandbox_image** (`pause:3.10.1`) set from the start

**Root cause** (from container logs):
- controller-manager: `Error retrieving lease lock` — connection refused to API server
- scheduler: `Failed to renew lease` — context deadline exceeded when dialing
  `https://192.168.122.179:6443`
- Both exit with code 1 (Error), not 0 (SIGTERM)
- No OOM events; 3.9 GiB available memory

The failure cascade is: API server becomes briefly unreachable → CM and scheduler
lose leader election leases → exit with error → kubelet restarts → CrashLoopBackOff
accumulates. The root trigger (why the API server becomes unreachable) remains
unresolved and appears to be specific to this kernel/containerd/kubeadm combination
on Debian 13 (kernel `6.12.74+deb13+1-amd64` + containerd `1.7.24`).

A third attempt with **10 GB RAM** (cold boot, `virsh setmaxmem` to 10 GiB,
confirmed 9.7 GiB total / 7.7 GiB available inside VM) showed the identical
failure pattern: API server logs confirm etcd connection timeout
(`dial tcp 127.0.0.1:2379: operation was canceled`), followed by CM and scheduler
lease loss. The host also runs a `k3s server` process (6% MEM), which may
contribute to resource contention or port conflict. Root cause is not memory
pressure — appears to be related to etcd/API server interaction under 2 vCPUs
or kernel 6.12.74.

A fourth attempt added **2 more vCPUs** (4 total) + **fresh kubeadm reinstall**
(`kubeadm reset`, state cleanup, `kubeadm init` with v1.36.2). The fresh cluster
was stable for ~8 minutes (GCP CRD installed, test pods scheduled, controller
binary uploaded). It then degraded into the identical failure pattern: API server
etcd connection timeout, CM/scheduler lease loss, CrashLoopBackOff. No OOM at
any point (7.7 GB available, 4 vCPUs idle).

**Root cause is definitively NOT memory or CPU starvation** — the failure occurs
regardless of resource allocation (4 GB/2 vCPU → 6 GB/2 vCPU → 10 GB/4 vCPU).
Likely causes (in order of suspicion):
1. **k3s server interference** — a `k3s server` process runs alongside kubeadm,
   may contend for etcd ports or API server resources
2. **containerd 1.7.24 + kernel 6.12.74 incompatibility** — this specific
   combination on Debian 13 may have issues with pause container lifecycle
3. **kubeadm v1.36.2 static pod behavior** — static pod manifests for
   control-plane may interact poorly with kubelet reconciliation on this kernel

The RAM and vCPU increases alone did not resolve the instability. The control-plane
components run as containers in the same pause sandbox and are subject to
kubelet reconciliation that continues to cycle them.

This prevents:
- GC controller pod scheduling (requires functional scheduler + CMS)
- Runtime reconciliation verification on this VM

### Where Runtime IS Verified
Runtime controller behavior was verified on **kind** (v1.36.1) and **k3d**
(K3s v1.36.2+k3s1) — see `kind.md` and `k3d.md` in this directory.

## Evidence Files

- `kubeadm.json` — structured evidence data.
