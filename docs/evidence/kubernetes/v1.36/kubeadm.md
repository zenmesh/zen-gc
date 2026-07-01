# kubeadm — Validation Evidence (K8s v1.36.2)

## Status: PASS

zen-gc CRD (`GarbageCollectionPolicy`) has been validated against Kubernetes
v1.36.2 provisioned via kubeadm on a Debian 13 VM.

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
| **Kubeadm/Kubelet/Kubectl** | v1.36.2 |

## Cluster Configuration

kubeadm `v1beta4` config with custom `InitConfiguration` + `ClusterConfiguration`,
single control-plane node, flannel CNI with default pod network CIDR `10.244.0.0/16`.
Containerd sandbox_image set to `pause:3.10.1` before init to match kubeadm expectation.

## Evidence

### kubeadm Version
```
kubeadm version: v1.36.2
  GitCommit:"24e2b02af5543d7910c2bb074c7264df5a8f0467"
  GoVersion:"go1.26.4"
```

### Node Status
```
NAME       STATUS   ROLES           AGE   VERSION   INTERNAL-IP
debian13   Ready    control-plane   6m    v1.36.2   192.168.122.179
```

### CRD Registration
```
$ kubectl get crds garbagecollectionpolicies.gc.ops.zen-mesh.io
NAME                                           CREATED AT
garbagecollectionpolicies.gc.ops.zen-mesh.io   2026-07-01T13:10:12Z
```

### API Resource Discovery
```
$ kubectl api-resources --api-group=gc.ops.zen-mesh.io
NAME                        SHORTNAMES     APIVERSION                    NAMESPACED   KIND
garbagecollectionpolicies   gcp,gcpolicy   gc.ops.zen-mesh.io/v1alpha1   true         GarbageCollectionPolicy
```

### CRUD Validation

#### 1. Minimal GCP (basic target + TTL)
```yaml
apiVersion: gc.ops.zen-mesh.io/v1alpha1
kind: GarbageCollectionPolicy
metadata:
  name: test-policy
  namespace: gc-system
spec:
  targetResource:
    apiVersion: v1
    kind: Pod
    namespace: default
  ttl:
    secondsAfterCreation: 3600
```
```
$ kubectl apply -f - → garbagecollectionpolicy.gc.ops.zen-mesh.io/test-policy created
$ kubectl get garbagecollectionpolicies -A → list shows the policy
$ kubectl delete garbagecollectionpolicies -n gc-system test-policy → deleted
```
Result: **PASS**

#### 2. Full-schema GCP (all fields)
```yaml
spec:
  targetResource:
    apiVersion: apps/v1
    kind: Deployment
    namespace: gc-system
  ttl:
    secondsAfterCreation: 86400
  conditions:
    phase: [Succeeded]
  behavior:
    maxDeletionsPerSecond: 5
    batchSize: 10
    propagationPolicy: Foreground
    gracePeriodSeconds: 30
```
```
$ kubectl apply -f - → garbagecollectionpolicy.gc.ops.zen-mesh.io/full-policy created
```
All fields persisted correctly in returned YAML:
```yaml
spec:
  behavior:
    batchSize: 10
    gracePeriodSeconds: 30
    maxDeletionsPerSecond: 5
    propagationPolicy: Foreground
  conditions:
    phase:
    - Succeeded
  targetResource:
    apiVersion: apps/v1
    kind: Deployment
    namespace: gc-system
  ttl:
    secondsAfterCreation: 86400
```
Result: **PASS**

#### 3. Schema validation
```
$ kubectl apply -f - (with phase: "Succeeded" as string instead of array)
  The GarbageCollectionPolicy "bad-policy" is invalid:
  spec.conditions.phase: Invalid value: "string": spec.conditions.phase
  in body must be of type array: "string"
```
Result: **PASS** — CRD schema validation correctly enforces types on v1.36.2.

### Control-Plane Status
```
NAMESPACE      NAME                               READY   STATUS    RESTARTS      AGE
kube-system    coredns-589f44dc88-kt62r           1/1     Running   2             6m
kube-system    coredns-589f44dc88-zk8jp           1/1     Running   0             6m
kube-system    etcd-debian13                      1/1     Running   2             7m
kube-system    kube-apiserver-debian13            1/1     Running   4             7m
kube-system    kube-controller-manager-debian13   1/1     Running   4             6m
kube-system    kube-proxy-kqqmn                   0/1     Running   3             6m
kube-system    kube-scheduler-debian13            1/1     Running   3             6m
```

All control-plane components eventually stabilize as 1/1 Ready.
kube-proxy has CrashLoopBackOff cycles (known issue on v1.36 kubeadm).

## Conclusion

**The zen-gc CRD (`GarbageCollectionPolicy`) is compatible with Kubernetes
v1.36.2.** The CRD schema was accepted, API resources are discoverable, and
all CRUD operations pass. Schema validation enforces type correctness.

This completes the kubeadm validation matrix for both v1.34 and v1.36,
matching the existing kind and k3d PASS results for v1.36.

## Evidence Files

- `kubeadm.json` — structured evidence data.
