#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="seaweedfs"
rm -f "$SCRIPT_DIR/manifests/seaweedfs/.env"

if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
    kubectl create namespace "$NAMESPACE"
fi

kubectl apply -f "$SCRIPT_DIR/manifests/seaweedfs/deployment.yaml" -n "$NAMESPACE"
if ! kubectl rollout status deployment/seaweedfs -n "$NAMESPACE" --timeout=3m; then
    kubectl describe deployment/seaweedfs -n "$NAMESPACE"
    kubectl logs deployment/seaweedfs -n "$NAMESPACE" || true
    exit 1
fi

# Mini starts the S3 gateway before it creates S3_BUCKET. Check the filer so
# tests cannot start against a healthy gateway with a missing default bucket.
bucket_ready=false
for _ in $(seq 1 30); do
    if bucket_list=$(printf 's3.bucket.list\n' | kubectl exec -i -n "$NAMESPACE" deployment/seaweedfs -- weed shell -master=localhost:9333 -filer=localhost:8888 2>/dev/null) && \
        grep -Eq '^[[:space:]]+default[[:space:]]+size:' <<< "$bucket_list"; then
        bucket_ready=true
        break
    fi
    sleep 1
done
if [[ "$bucket_ready" != true ]]; then
    echo "Default SeaweedFS test bucket is unavailable" >&2
    kubectl logs deployment/seaweedfs -n "$NAMESPACE" --tail=100 || true
    exit 1
fi

access_key=$(kubectl get secret seaweedfs-secret -n "$NAMESPACE" -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 --decode)
secret_key=$(kubectl get secret seaweedfs-secret -n "$NAMESPACE" -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 --decode)
if [[ -z "$access_key" || -z "$secret_key" ]]; then
    echo "Failed to retrieve SeaweedFS test credentials" >&2
    exit 1
fi

cat > "$SCRIPT_DIR/manifests/seaweedfs/.env" <<EOF
KF_MR_TEST_S3_ENDPOINT=http://localhost:9000
KF_MR_TEST_BUCKET_NAME=default
KF_MR_TEST_ACCESS_KEY_ID=$access_key
KF_MR_TEST_SECRET_ACCESS_KEY=$secret_key
EOF
