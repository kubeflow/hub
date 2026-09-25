# Kind Dev Environment

Local Kubernetes development environment for the model-registry UI using Kind and Tilt.

## Prerequisites

- A Docker-compatible runtime (e.g. Docker Desktop, Colima, Podman with `podman-docker`)
- [Kind](https://kind.sigs.k8s.io/) installed
- kubectl installed
- Go >= 1.26
- Node.js >= 22.0.0
- Tilt v0.33.22+ (auto-downloaded by `make tilt-up` if not present)
- `yq` and `kustomize` (required for running the Catalog service)

## Architecture

The dev environment runs three concurrent processes:

| Component | Directory | Port | Purpose |
|-----------|-----------|------|---------|
| Infrastructure | `devenv/` | 10350 (Tilt) | Kind cluster + Tilt for deploying model-registry resources |
| BFF | `clients/ui/bff/` | 4000 | Go backend-for-frontend server |
| Frontend | `clients/ui/frontend/` | 9000 | React dev server (proxies to BFF) |

## Quick Start

### 1. Start Infrastructure

```bash
# Start your Docker runtime (example with Colima on macOS):
colima start

# Create the Kind cluster (if it doesn't exist):
kind get clusters | grep -q '^model-registry$' || kind create cluster --name model-registry
kubectl config use-context kind-model-registry

# Start Tilt (auto-downloads if not installed):
cd devenv && make tilt-up
```

### 2. Start BFF

```bash
cd clients/ui/bff
go run ./cmd --port=4000 --dev-mode \
  --dev-mode-model-registry-port=8080 \
  --dev-mode-catalog-port=8082 \
  --deployment-mode=standalone
```

Add `--mock-k8s-client` if you don't need a real cluster for basic UI testing.

### 3. Start Frontend

```bash
cd clients/ui/frontend
DEPLOYMENT_MODE=standalone STYLE_THEME=patternfly npm run start:dev
```

### 4. RBAC Setup (real K8s only)

When using the real K8s client, the BFF's namespace registry access check uses SubjectAccessReview with only the `User` field (no groups), so group-based RoleBindings don't take effect. Create a ClusterRoleBinding that directly binds each namespace's default ServiceAccount:

```bash
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: model-registry-all-sa-service-access
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: model-registry-ui-services-reader
subjects:
- kind: ServiceAccount
  name: default
  namespace: default
- kind: ServiceAccount
  name: default
  namespace: kubeflow
- kind: ServiceAccount
  name: default
  namespace: seaweedfs
- kind: ServiceAccount
  name: default
  namespace: kube-system
- kind: ServiceAccount
  name: default
  namespace: kube-public
- kind: ServiceAccount
  name: default
  namespace: kube-node-lease
- kind: ServiceAccount
  name: default
  namespace: local-path-storage
EOF
```

Without this, the UI shows "The selected namespace does not have access to this model registry" for every namespace.

## Optional Components

### Model Catalog (with performance data)

```bash
./scripts/deploy_catalog_demo_on_kind.sh
kubectl port-forward -n model-catalog svc/model-catalog-server 8082:8080
```

The BFF connects to this on `--dev-mode-catalog-port=8082`. See [deploy_catalog_demo_on_kind.sh](../scripts/deploy_catalog_demo_on_kind.sh) for details.

### SeaweedFS (S3 storage for transfer jobs)

```bash
./scripts/deploy_seaweedfs_on_kind.sh
```

| Detail | Value |
|--------|-------|
| Internal endpoint | `http://seaweedfs.seaweedfs.svc.cluster.local:8333` |
| Bucket | `default` |
| Credentials | `seaweedadmin` / `seaweedadmin` (test only) |
| K8s Secret | `seaweedfs-secret` (namespace: `seaweedfs`) |

To upload test data with the AWS CLI, forward port 9002 in a separate terminal (port 9000 is used by the frontend):

```bash
kubectl port-forward -n seaweedfs svc/seaweedfs 9002:8333
```

Then upload from another terminal:

```bash
echo "sample model content" | \
  AWS_ACCESS_KEY_ID=seaweedadmin AWS_SECRET_ACCESS_KEY=seaweedadmin \
  aws --endpoint-url http://localhost:9002 s3 cp - s3://default/models/sample-model/model.txt
```

### OCI Model Transfer Jobs

Test S3-to-OCI model transfer jobs end-to-end. Requires SeaweedFS (above) and a destination OCI registry (e.g. quay.io).

No local ARM64 image build is needed — upstream, midstream, and downstream each have their own async-upload images.

**Form values when creating a transfer job via UI:**

| Field | Value | Notes |
|-------|-------|-------|
| Source type | `s3` | |
| S3 endpoint | `http://seaweedfs.seaweedfs:8333` | Internal cluster DNS |
| S3 bucket | `default` | |
| S3 key | `models/sample-model/` | Directory prefix, **not** full file path |
| S3 access key | `seaweedadmin` | |
| S3 secret key | `seaweedadmin` | |
| Destination type | `oci` | |
| Destination URI | `quay.io/yourorg/yourrepo:tag` | OCI ref format, **no** `https://` |
| Destination registry | `quay.io` | |

## Teardown

```bash
# Stop everything and delete cluster:
./scripts/dev_teardown.sh

# Or keep the cluster for quick restart:
./scripts/dev_teardown.sh --keep-cluster
```

Override default ports with environment variables:

```bash
FRONTEND_PORT=9001 BFF_PORT=4001 ./scripts/dev_teardown.sh
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| Tilt refuses to start (production context) | `kubectl config use-context kind-model-registry` |
| Port conflict (4000, 9000, or 10350) | `lsof -ti:PORT \| xargs kill -9` |
| Frontend proxy `ECONNREFUSED` | BFF not ready yet; wait for it to start |
| `ImagePullBackOff` on async-upload job | Verify the correct image is configured for your environment (upstream/midstream/downstream each have their own) |
| "namespace does not have access to this model registry" | Apply the RBAC ClusterRoleBinding (see above). The BFF's SAR uses `User` only, not groups. |
| SeaweedFS service unavailable | Verify the SeaweedFS deployment in the `seaweedfs` namespace. |
| SeaweedFS bucket missing after pod restart | SeaweedFS has no PV in this test setup. Check the pod logs, then re-run: `./scripts/deploy_seaweedfs_on_kind.sh` |
| `envtest` port lock error in Go tests | `rm -f ~/Library/Caches/kubebuilder-envtest/port-*` |
| Transfer job S3 download fails with EBUSY | Use directory prefix as source key, not full file path |
| Transfer job OCI push "invalid reference" | Use OCI ref format `quay.io/org/repo:tag`, not web URL |
