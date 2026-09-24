#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
kubectl delete namespace seaweedfs --ignore-not-found --wait=true --timeout=300s
rm -f "$SCRIPT_DIR/manifests/seaweedfs/.env"
