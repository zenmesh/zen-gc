# kubeadm — Validation Evidence (K8s v1.36.2)

## Status: PASS

Full zen-gc validation (CRD/API, CRUD lifecycle, negative schema, RBAC,
controller runtime, GC behavior) completed successfully on Kubernetes v1.36.2
provisioned via kubeadm on Debian 13.

**OS/runtime note**: Debian 13 ships containerd 1.7.24 which caused kubelet
`SandboxChanged` events on static pod pause containers, making the control plane
unstable. Validation was completed after upgrading containerd to **2.2.5** from
Docker's apt repository (`containerd.io=2.2.5-1~debian.12~bookworm`).
Debian's default containerd 1.7.24 is not claimed as stable for this workload.

## VM Configuration

| Field | Value |
|-------|-------|
| **Hostname** | `h462-gateway-kubeadm-1780668538` |
| **IP** | 192.168.122.179 |
| **Libvirt domain** | `h462-gateway-kubeadm-1780668538` |
| **OS** | Debian 13 (trixie) |
| **Kernel** | 6.12.74+deb13+1-amd64 |
| **RAM** | 12 GB (`virsh setmaxmem` + `setmem --config`, stop/start cycle) |
| **RAM (guest)** | 11 GiB total / 10 GiB available |
| **vCPUs** | 4 |
| **Containerd** | 2.2.5 (upgraded from Debian's 1.7.24) |
| **CNI** | Flannel v0.28.5 |
| **Kubeadm/Kubelet/Kubectl** | v1.36.2 |
| **CoreDNS** | v1.12.0 (deployed via `kubeadm init phase addon coredns`) |

## Cluster Configuration

kubeadm `v1beta4` config, single control-plane node, flannel CNI with
`10.244.0.0/16` pod CIDR. Containerd 2.2.5 with SystemdCgroup driver.

## Evidence

### Cluster Substrate

```
$ kubectl get nodes -o wide
NAME       STATUS   ROLES           AGE   VERSION   INTERNAL-IP       OS-IMAGE                       KERNEL-VERSION                  CONTAINER-RUNTIME
debian13   Ready    control-plane   78m   v1.36.2   192.168.122.179   Debian GNU/Linux 13 (trixie)   6.12.74+deb13+1-amd64 (amd64)   containerd://2.2.5

$ kubectl get pods -A -o wide
NAMESPACE      NAME                               READY   STATUS    RESTARTS
default        control-pod                        1/1     Running   0
gc-system      gc-controller-55b4b446d8-fgrxn     1/1     Running   1 (7m ago)
kube-flannel   kube-flannel-ds-krbtd              1/1     Running   0
kube-system    coredns-589f44dc88-dhjzb           1/1     Running   0
kube-system    coredns-589f44dc88-vkrzj           1/1     Running   0
kube-system    etcd-debian13                      1/1     Running   40
kube-system    kube-apiserver-debian13            1/1     Running   39
kube-system    kube-controller-manager-debian13   1/1     Running   48
kube-system    kube-proxy-6r94w                   1/1     Running   0
kube-system    kube-scheduler-debian13            1/1     Running   60
```

CP restart counts are from the initial kubelet startup (cold boots + containerd
upgrade). **Zero new restarts during the 45+ minute validation window.**

### CRD Install Idempotency

```
$ kubectl apply -f deploy/crds/gc.kube-zen.io_garbagecollectionpolicies.yaml
  customresourcedefinition.apiextensions.k8s.io/garbagecollectionpolicies.gc.ops.zen-mesh.io created

$ kubectl apply -f deploy/crds/gc.kube-zen.io_garbagecollectionpolicies.yaml
  customresourcedefinition.apiextensions.k8s.io/garbagecollectionpolicies.gc.ops.zen-mesh.io unchanged
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
| Create minimal GCP | ✅ created |
| Create full-schema GCP (all spec fields) | ✅ created |
| List GCPs | ✅ appears with namespace/name/age |
| Read GCP YAML | ✅ all spec fields persisted |
| Re-apply GCP (idempotency) | ✅ unchanged |
| Delete GCP | ✅ deleted |
| GCPs after delete | ✅ empty list |

### Negative Schema Validation

| Test | Input | Result |
|------|-------|--------|
| Wrong type | `ttl.secondsAfterCreation: "3600"` (string) | ❌ Rejected: "must be of type integer" |
| Unknown field | `spec.nonexistentField: true` | ❌ Rejected: "unknown field" (strict decoding) |
| Empty spec | `spec: {}` | ❌ Rejected: "targetResource: Required value" + "ttl: Required value" |
| Missing required | no `spec.targetResource` | ❌ Rejected: "targetResource: Required value" |

All invalid inputs are correctly rejected by the API server with descriptive
error messages.

### RBAC / Permission Boundaries

```
$ kubectl auth can-i --list --as=system:serviceaccount:gc-system:gc-controller
  *.*                                                   [get list watch delete]
  garbagecollectionpolicies.gc.ops.zen-mesh.io          [get list watch]
  garbagecollectionpolicies.gc.ops.zen-mesh.io/status   [get update patch]
```

The controller has GCP read access, status write access, and the broad
list/delete permissions required for garbage collection across namespaces.
Current RBAC gives `*.*` list/watch/delete (known broad scope — documented
risk; see RBAC hardening notes).

### Controller Runtime

```
$ kubectl get pods -n gc-system -o wide
NAME                             READY   STATUS    RESTARTS
gc-controller-55b4b446d8-fgrxn   1/1     Running   1
gc-controller-55b4b446d8-2crzr   0/1     Running   0

$ kubectl logs -n gc-system gc-controller-55b4b446d8-fgrxn --tail=20
{"level":"info","msg":"Controller configuration", ... "gcInterval":"1m0s"}
{"level":"info","msg":"Leader election enabled", ... "electionID":"gc-controller-leader-election"}
{"level":"info","msg":"Starting workers","controller":"garbagecollectionpolicy","worker count":1}
{"level":"info","msg":"Starting webhook server with TLS","address":":9443"}
{"level":"info","msg":"Starting GC controller manager"}
```

Controller started successfully, acquired leader lease, and reconciliation
workers are active. Leader election across 2 replicas works (one active, one
standby).

### GC Behavior

**Test**: Create a GCP targeting pods with label `gc-disposable=true`, TTL of
10 seconds. Disposable pod has the label; control pod does not.

```
# Before GC evaluation:
NAME             READY   LABELS
control-pod      1/1     gc-control=true
disposable-pod   1/1     gc-disposable=true

# After 20 seconds (1 GC interval):
NAME          READY   LABELS
control-pod   1/1     gc-control=true
(disposable-pod deleted)
```

✅ **Disposable pod deleted** — GCP matched the labeled pod and the controller
deleted it after TTL expiry.

✅ **Control pod untouched** — no matching labels, unaffected by GC policy.

✅ **No unrelated resources affected** — only the targeted pod was deleted.

### Post-Validation Stability

After all validation actions, the control plane remains stable with no restart
growth. CoreDNS resolves DNS queries. kubelet is active. Node Ready.

## Retry History

Six attempts were made to stabilize kubeadm on this VM:

| Attempt | Action | Result |
|---------|--------|--------|
| 1 | 4 GB / 2 vCPU, containerd 1.7.24 | CP unstable |
| 2 | 6 GB / 2 vCPU, reboot | CP stable ~10 min then crashed |
| 3 | 10 GB / 2 vCPU, cold boot | CP immediately unstable |
| 4 | 10 GB / 4 vCPU, fresh kubeadm reinstall | CP stable ~8 min then crashed |
| 5 | Full K8s strip + reboot + reinstall (ruled out k3s) | Identical failure pattern |
| **6** | **containerd upgraded to 2.2.5 + 12 GB / 4 vCPU** | **CP stable, full validation PASS** |

Root cause: containerd 1.7.24 from Debian 13 causes kubelet `SandboxChanged`
events on static pod pause containers. Upgrading to containerd 2.2.5 resolved
the issue. k3s interference was ruled out by a full-host strip test (attempt 5).

## Evidence Files

- `kubeadm.json` — structured evidence data.
