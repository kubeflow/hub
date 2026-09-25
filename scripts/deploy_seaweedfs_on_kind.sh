#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SEAWEEDFS_NAMESPACE="seaweedfs"
rm -f "$SCRIPT_DIR/manifests/seaweedfs/.env"

# modularity to allow re-use this script against a remote k8s cluster
if [[ -n "${LOCAL:-}" ]]; then
    CLUSTER_NAME="${CLUSTER_NAME:-kind}"
    echo 'Creating local Kind cluster and loading image'
    if kind get clusters | grep -Fxq "$CLUSTER_NAME"; then
        echo 'Cluster already exists, skipping creation'
        kubectl config use-context "kind-$CLUSTER_NAME"
    else
        kind create cluster -n "$CLUSTER_NAME"
    fi
    if [[ -n "${IMG:-}" ]]; then
        kind load docker-image -n "$CLUSTER_NAME" "$IMG"
        echo 'Image loaded into kind cluster - use this command to port forward the mr service:'
        # echo "kubectl port-forward -n $MR_NAMESPACE service/model-registry-service 8080:8080 &"
    fi
fi

echo 'Deploying SeaweedFS S3 storage to Kind cluster'
if kubectl get namespace "$SEAWEEDFS_NAMESPACE" >/dev/null 2>&1; then
    echo 'Namespace already exists, skipping creation'
else
    kubectl create namespace "$SEAWEEDFS_NAMESPACE"
fi

kubectl apply -f "$SCRIPT_DIR/manifests/seaweedfs/deployment.yaml" -n "$SEAWEEDFS_NAMESPACE"
if ! kubectl rollout status deployment/seaweedfs -n "$SEAWEEDFS_NAMESPACE" --timeout=3m; then
     echo "SeaweedFS deployment took more than 3 minutes."
     kubectl describe deployment/seaweedfs -n "$SEAWEEDFS_NAMESPACE"
     kubectl logs deployment/seaweedfs -n "$SEAWEEDFS_NAMESPACE" --tail=100 || true
     exit 1
fi

KF_MR_TEST_ACCESS_KEY_ID=$(kubectl get secret seaweedfs-secret -n "$SEAWEEDFS_NAMESPACE" -o jsonpath="{.data.ACCESS_KEY_ID}" | base64 --decode)
KF_MR_TEST_SECRET_ACCESS_KEY=$(kubectl get secret seaweedfs-secret -n "$SEAWEEDFS_NAMESPACE" -o jsonpath="{.data.SECRET_KEY}" | base64 --decode)

if [[ -z "$KF_MR_TEST_ACCESS_KEY_ID" || -z "$KF_MR_TEST_SECRET_ACCESS_KEY" ]]; then
    echo "Error: Failed to retrieve SeaweedFS credentials. Exiting."
    exit 1
fi

cat > "$SCRIPT_DIR/manifests/seaweedfs/.env" <<EOF
KF_MR_TEST_S3_ENDPOINT=http://localhost:8333
KF_MR_TEST_BUCKET_NAME=default
KF_MR_TEST_ACCESS_KEY_ID=$KF_MR_TEST_ACCESS_KEY_ID
KF_MR_TEST_SECRET_ACCESS_KEY=$KF_MR_TEST_SECRET_ACCESS_KEY
EOF
