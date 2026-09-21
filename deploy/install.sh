#!/usr/bin/env bash
# Canonical zen-gc install (S025-GC §2).
# Required env:
#   GC_IMAGE             immutable image ref (refuses :latest)
#   WEBHOOK_NS           namespace (default gc-system)
set -euo pipefail
cd "$(dirname "$0")/.."
NS="${WEBHOOK_NAMESPACE:-gc-system}"
fail(){ echo "FAIL: $*" >&2; exit 1; }
command -v kubectl >/dev/null || fail "kubectl required"
command -v openssl >/dev/null || fail "openssl required (webhook cert generation)"
[ -n "${GC_IMAGE:-}" ] || fail "GC_IMAGE required (immutable tag)"
case "$GC_IMAGE" in *:latest|*:latest@*) fail "refusing :latest";; *:*) ;; *) fail "GC_IMAGE must carry explicit tag";; esac
if grep -rq "REPLACE_WITH\|SET_BY_INSTALLER" deploy/manifests/deployment.yaml; then fail "deployment.yaml still contains placeholders"; fi
kubectl create ns "$NS" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deploy/crds/
kubectl apply -n "$NS" -f deploy/manifests/rbac.yaml
bash deploy/gen-webhook-cert.sh
kubectl apply -n "$NS" -f deploy/manifests/service.yaml
GC_IMAGE="$GC_IMAGE" envsubst < deploy/manifests/deployment.yaml | kubectl apply -n "$NS" -f -
echo "installed zen-gc; verify: kubectl -n $NS rollout status deployment/gc-controller"
