# kubeadm — Validation Evidence

## Status: BLOCKED

kubeadm validation on Kubernetes v1.36.x was not completed.

## GAPI VM Identified

| Field | Value |
|-------|-------|
| **Hostname** | `h462-gateway-kubeadm-1780668538` |
| **IP** | 192.168.122.179 |
| **Libvirt domain** | `h462-gateway-kubeadm-1780668538` |
| **OS** | Debian 13 (trixie) |
| **RAM** | 4 GB (6 GB max) |
| **vCPUs** | 2 |
| **Disk** | 11 GB qcow2 |

## Why Blocked

The GAPI VM exists and is running, but its installed Kubernetes version is **v1.32.13**,
not v1.36.x. Upgrading from 1.32 to 1.36 is a multi-step process (1.32 → 1.33 → 1.34 →
1.35 → 1.36) that exceeds the scope of this validation pass.

### VM Access

The VM is accessible via the libvirt QEMU guest agent (`virsh qemu-agent-command`).
SSH password authentication was not enabled on the VM; `virsh set-user-password` was
used to set a root password but SSH `PasswordAuthentication` remained disabled.

### Confirmed

- VM is disposable (dedicated to kubeadm work, no production workload).
- kubelet version: v1.32.13
- kubeadm version: v1.32.13
- Running kubeadm cluster with flannel CNI.

## Prerequisites for Re-testing

1. Install kubeadm/kubelet/kubectl packages for Kubernetes v1.36.x on the VM
   (e.g., `1.36.2-1.1` from the Kubernetes apt repository).
2. Upgrade or reinitialize the cluster at 1.36.x.
3. Run the common validation scenario (same as kind and k3d).

## Evidence Files

- `kubeadm.json` — structured evidence data.
