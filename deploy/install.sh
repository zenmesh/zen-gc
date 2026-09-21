#!/usr/bin/env bash
# Canonical zen-gc install (S025-GC §2; SUPPORT2-032R §4 packaging closure).
# Required env:
#   GC_IMAGE             immutable image ref (refuses :latest)
#   WEBHOOK_NAMESPACE    namespace (default gc-system)
#
# SUPPORT2-032R fixes over the S025-GC §2 revision:
#   - deployment image is actually substituted (manifest now carries
#     ${GC_IMAGE}; the former bare IMAGE_PLACEHOLDER survived envsubst and
#     deployed an unpullable ref)
#   - the generated webhook CA bundle is wired into the admission configs
#     (explicit caBundle; TLS admission path installs with the component)
#   - every namespaced manifest (deployment, service, rbac, webhook
#     service/clientConfig) is templated on WEBHOOK_NAMESPACE
#   - rendered output is guarded: no unsubstituted placeholder can be
#     applied to the cluster
set -euo pipefail
cd "$(dirname "$0")/.."
NS="${WEBHOOK_NAMESPACE:-gc-system}"
export WEBHOOK_NAMESPACE="$NS"
fail(){ echo "FAIL: $*" >&2; exit 1; }
command -v kubectl >/dev/null || fail "kubectl required"
command -v openssl >/dev/null || fail "openssl required (webhook cert generation)"
command -v envsubst >/dev/null || fail "envsubst required (manifest rendering)"
[ -n "${GC_IMAGE:-}" ] || fail "GC_IMAGE required (immutable tag)"
case "$GC_IMAGE" in *:latest|*:latest@*) fail "refusing :latest";; *:*) ;; *) fail "GC_IMAGE must carry explicit tag"; esac
if grep -rq "REPLACE_WITH\|SET_BY_INSTALLER" deploy/manifests/deployment.yaml; then fail "deployment.yaml still contains placeholders"; fi

kubectl create ns "$NS" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deploy/crds/
kubectl apply -n "$NS" -f deploy/manifests/rbac.yaml

bash deploy/gen-webhook-cert.sh

# The generated serving cert doubles as the admission CA bundle (base64 PEM
# is exactly what the secret's tls.crt data field holds).
WEBHOOK_CA_BUNDLE="$(kubectl -n "$NS" get secret gc-controller-webhook-cert -o jsonpath='{.data.tls\.crt}')"
export WEBHOOK_CA_BUNDLE

kubectl apply -n "$NS" -f deploy/manifests/service.yaml

TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
envsubst < deploy/manifests/deployment.yaml > "$TMP/deployment.yaml"
envsubst < deploy/webhook/mutating-webhook.yaml > "$TMP/mutating-webhook.yaml"
envsubst < deploy/webhook/validating-webhook.yaml > "$TMP/validating-webhook.yaml"
if grep -rI '\${' "$TMP"; then fail "unsubstituted placeholder in rendered manifests"; fi

kubectl apply -n "$NS" -f "$TMP/deployment.yaml"
kubectl apply -f "$TMP/mutating-webhook.yaml" -f "$TMP/validating-webhook.yaml"

echo "installed zen-gc; verify: kubectl -n $NS rollout status deployment/gc-controller"
