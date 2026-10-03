#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
unset KF_MR_TEST_ACCESS_KEY_ID KF_MR_TEST_SECRET_ACCESS_KEY KF_MR_TEST_S3_ENDPOINT KF_MR_TEST_BUCKET_NAME

kubectl delete namespace seaweedfs --ignore-not-found --wait=true --timeout=300s
rm -f "$SCRIPT_DIR/manifests/seaweedfs/.env"
