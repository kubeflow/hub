#!/usr/bin/env bash
# Verify that kustomize build output for all overlays sets namespace: kubeflow
# and does not produce resources in the default namespace.
#
# Usage: ./check-namespace.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUSTOMIZE_DIR="${SCRIPT_DIR}"

OVERLAYS=(
  "overlays/postgres"
  "options/istio"
  "options/ui/overlays/istio"
  "options/catalog/overlays/demo"
)

EXIT_CODE=0

for overlay in "${OVERLAYS[@]}"; do
  overlay_path="${KUSTOMIZE_DIR}/${overlay}"

  if [ ! -d "${overlay_path}" ]; then
    echo "SKIP: ${overlay} (directory not found)"
    continue
  fi

  echo "Checking ${overlay}..."

  output=$(kustomize build "${overlay_path}" 2>&1) || {
    echo "FAIL: kustomize build failed for ${overlay}"
    EXIT_CODE=1
    continue
  }

  # Check for resources landing in the default namespace
  default_ns_resources=$(echo "${output}" | grep -c '^\s*namespace: default$' || true)
  if [ "${default_ns_resources}" -gt 0 ]; then
    echo "FAIL: ${overlay} produces ${default_ns_resources} resource(s) with namespace: default"
    EXIT_CODE=1
  fi

  # Verify namespace: kubeflow is present in the kustomization
  if ! grep -q 'namespace: kubeflow' "${overlay_path}/kustomization.yaml"; then
    echo "FAIL: ${overlay}/kustomization.yaml is missing 'namespace: kubeflow'"
    EXIT_CODE=1
  else
    echo "PASS: ${overlay}"
  fi
done

if [ "${EXIT_CODE}" -eq 0 ]; then
  echo ""
  echo "All overlays correctly set namespace: kubeflow"
else
  echo ""
  echo "ERROR: namespace check failed — see above"
fi

exit "${EXIT_CODE}"
