# kubeadm — Validation Evidence (K8s v1.34.9)

## Status: PASS (Full Runtime + GC)

zen-gc CRD (`GarbageCollectionPolicy`) has been fully validated against Kubernetes
v1.34.9 provisioned via kubeadm on a Debian 13 VM, including controller runtime
reconciliation and GC deletion behavior.

**Previous v1.34 evidence** was limited to CRD/API compatibility only due to
containerd 1.7.24 CP instability. The containerd upgrade to **2.2.5** resolves
the instability and enables full runtime validation.

## VM Configuration

| Field | Value |
|-------|-------|
| **Hostname** | `h462-gateway-kubeadm-1780668538` |
| **IP** | 192.168.122.179 |
| **Libvirt domain** | `h462-gateway-kubeadm-1780668538` |
| **OS** | Debian 13 (trixie) |
| **Kernel** | 6.12.74+deb13+1-amd64 |
| **RAM** | 12 GB |
| **vCPUs** | 4 |
| **Containerd** | 2.2.5 (upgraded from Debian's 1.7.24 via Docker apt repo) |
| **CNI** | Flannel v0.28.5 |
| **Kubeadm/Kubelet/Kubectl** | v1.34.9 |
| **Controller image** | Static CGO_ENABLED=0 build, scratch base |

## Validation Results

### CRD Registration & API Discovery
```
$ kubectl get crds garbagecollectionpolicies.gc.ops.zen-mesh.io
NAME                                           CREATED AT
garbagecollectionpolicies.gc.ops.zen-mesh.io   2026-07-01T18:04:40Z

$ kubectl api-resources --api-group=gc.ops.zen-mesh.io
NAME                        SHORTNAMES     APIVERSION                    NAMESPACED   KIND
garbagecollectionpolicies   gcp,gcpolicy   gc.ops.zen-mesh.io/v1alpha1   true         GarbageCollectionPolicy
```

### CRUD Lifecycle
| Operation | Result |
|-----------|--------|
| Create minimal GCP | ✅ |
| Create full-schema GCP | ✅ |
| List GCPs | ✅ |
| Read GCP YAML | ✅ |
| Re-apply CRD (idempotent) | ✅ |
| Delete GCP | ✅ |

### Negative Schema Validation
| Test | Result |
|------|--------|
| Wrong type (string for integer) | ✅ Rejected |
| Unknown field | ✅ Rejected (strict decoding) |
| Empty spec | ✅ Rejected (required fields) |
| Missing required `targetResource` | ✅ Rejected |

### Controller Deployment
```
$ kubectl get deployment gc-controller
NAME            READY   UP-TO-DATE   AVAILABLE   AGE
gc-controller   2/2     2            2           3m

$ kubectl logs gc-controller-... | tail
... "Starting workers" worker count=1
... "Deleted resource disposable-pod (reason: ttl_expired)"
... "Evaluated policy: matched=1, deleted=1, pending=0"
```

### GC Behavior
- **GCP**: `disposable-pod-cleanup` — matches pods with label `gc-disposable=true`, TTL 10s
- **Disposable pod** (`gc-disposable=true`): detected and **deleted** within 1 GC interval
- **Control pod** (`gc-control=true`): **not matched** — remains running
- **After GC**: second cycle confirmed `matched=0, deleted=0, pending=0`

### Control-Plane Stability
All CP components running with **0 restarts** (frozen since init), 9+ min uptime.

```
NAMESPACE      NAME                               READY   STATUS    RESTARTS   AGE
kube-system    etcd-debian13                      1/1     Running   0          9m
kube-system    kube-apiserver-debian13            1/1     Running   0          9m
kube-system    kube-controller-manager-debian13   1/1     Running   0          9m
kube-system    kube-scheduler-debian13            1/1     Running   0          9m
kube-system    coredns-66bc5c9577-9bs8p           1/1     Running   0          9m
```

## Known Issues
- Events RBAC missing — controller cannot create/patch events. GC operations complete successfully regardless.

## Limitations
- Single-node control-plane only (no HA)
- Debian 13 with containerd 2.2.5 (not Debian's default containerd 1.7.24)
- flannel CNI only
- Webhook running in insecure mode (no TLS certs)
- No cloud Kubernetes testing
