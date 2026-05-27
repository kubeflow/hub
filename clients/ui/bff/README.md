# Kubeflow Hub UI BFF

The Kubeflow Hub UI BFF is the _backend for frontend_ (BFF) used by the Kubeflow Hub UI.

## Pre-requisites:

### Dependencies

- Go >= 1.26

### Running model registry

To be operational, our BFF needs the Model Registry backend running.

> **NOTE:** Docker compose must be installed in your environment.

There are two `docker-compose` files located at the [root](https://github.com/kubeflow/hub) of Model Registry repository that make the startup of both model registry easier by simply running:

```shell
docker compose -f docker-compose[-local].yaml up
```

The main difference between the two docker compose files is that `-local` one build the model registry from source, the other one, instead, download the `latest` pushed [quay.io](https://quay.io/repository/opendatahub/model-registry?tab=tags) image.

## Development

Run the following command to build the BFF:

```shell
make build
```

After building it, you can run our app with:

```shell
make run
```

If you want to use a different port, mock kubernetes client or model registry client - useful for front-end development, you can run:

```shell
make run PORT=8000 MOCK_K8S_CLIENT=true MOCK_MR_CLIENT=true
```

If you want to change the log level on deployment, add the LOG_LEVEL argument when running, supported levels are: ERROR, WARN, INFO, DEBUG. The default level is INFO.

```shell
# Run with debug logging
make run LOG_LEVEL=DEBUG
```

## Running the linter locally

The BFF directory uses golangci-lint to combine multiple linters for a more comprehensive linting process. To install and run simply use:

```shell
cd clients/ui/bff
make lint
```

For more information on configuring golangci-lint see the [documentation](https://golangci-lint.run/).

## Building and Deploying

Run the following command to build the BFF:

```shell
make build
```

The BFF binary will be inside `bin` directory

You can also build BFF docker image with:

```shell
make docker-build
```

## Getting started

### Endpoints

See the [OpenAPI specification](../api/openapi/mod-arch.yaml) for a complete list of endpoints.

### Sample local calls

You will need to inject your requests with a `kubeflow-userid` header and namespace for authorization purposes.

When running the service with the mocked Kubernetes client (MOCK_K8S_CLIENT=true), the user `user@example.com` is preconfigured with the necessary RBAC permissions to perform these actions.

```
# GET /v1/healthcheck
curl -i "localhost:4000/healthcheck"
```

```
# GET /v1/user
curl -i -H "kubeflow-userid: user@example.com" "localhost:4000/api/v1/user"
curl -i -H "Authorization: Bearer $TOKEN" "localhost:4000/api/v1/user"
```

```
# GET /v1/namespaces (only works when DEV_MODE=true)
curl -i -H "kubeflow-userid: user@example.com" "localhost:4000/api/v1/namespaces"
curl -i -H "Authorization: Bearer $TOKEN" "localhost:4000/api/v1/namespaces"
```

```
# GET /v1/model_registry
curl -i -H "kubeflow-userid: user@example.com" "localhost:4000/api/v1/model_registry?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "localhost:4000/api/v1/model_registry?namespace=kubeflow"
```

```
# GET /v1/model_registry using groups permissions
curl -i \
  -H "kubeflow-userid: non-user@example.com" \
  -H "kubeflow-groups: dora-namespace-group ,group2,group3" \
  "http://localhost:4000/api/v1/model_registry?namespace=dora-namespace"
```

```
# GET /v1/model_registry/{model_registry_id}/registered_models
curl -i -H "kubeflow-userid: user@example.com" "localhost:4000/api/v1/model_registry/model-registry/registered_models?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "localhost:4000/api/v1/model_registry/model-registry-service/registered_models?namespace=kubeflow-user-example-com""
```

```
# GET /v1/model_registry/{model_registry_id}/registered_models using group permissions
curl -i \
  -H "kubeflow-userid: non-user@example.com" \
  -H "kubeflow-groups: dora-namespace-group ,dora-service-group,group3" \
  "http://localhost:4000/api/v1/model_registry/model-registry-dora/registered_models?namespace=dora-namespace"
```

```
#POST /v1/model_registry/{model_registry_id}/registered_models
curl -i -H "kubeflow-userid: user@example.com" -X POST "http://localhost:4000/api/v1/model_registry/model-registry/registered_models?namespace=kubeflow" \
     -H "Content-Type: application/json" \
     -d '{ "data": {
  "customProperties": {
    "my-label9": {
      "metadataType": "MetadataStringValue",
      "string_value": "val"
    }
  },
  "description": "bella description",
  "externalId": "9927",
  "name": "bella",
  "owner": "eder",
  "state": "LIVE"
}}'
```

```
# GET /v1/model_registry/{model_registry_id}/registered_models/{registered_model_id}
curl -i -H "kubeflow-userid: user@example.com" "localhost:4000/api/v1/model_registry/model-registry/registered_models/1?namespace=kubeflow"
```

```
# PATCH /v1/model_registry/{model_registry_id}/registered_models/{registered_model_id}
curl -i -H "kubeflow-userid: user@example.com" -X PATCH "http://localhost:4000/api/v1/model_registry/model-registry/registered_models/1?namespace=kubeflow" \
-H "Content-Type: application/json" \
-d '{ "data": {
  "description": "New description"
}}'
```

```
# GET /api/v1/model_registry/{model_registry_id}/model_versions
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/model_versions?namespace=kubeflow"
```

```
# GET /api/v1/model_registry/{model_registry_id}/model_versions/{model_version_id}
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/model_versions/1?namespace=kubeflow"
```

```
# POST /api/v1/model_registry/{model_registry_id}/model_versions
curl -i -H "kubeflow-userid: user@example.com" -X POST "http://localhost:4000/api/v1/model_registry/model-registry/model_versions?namespace=kubeflow" \
     -H "Content-Type: application/json" \
     -d '{ "data": {
  "customProperties": {
    "my-label9": {
      "metadataType": "MetadataStringValue",
      "string_value": "val"
    }
  },
  "description": "Version description",
  "externalId": "9927",
  "name": "ModelVersion One",
  "state": "LIVE",
  "author": "alex",
  "registeredModelId": "1"
}}'
```

```
# PATCH /api/v1/model_registry/{model_registry_id}/model_versions/{model_version_id}
curl -i -H "kubeflow-userid: user@example.com" -X PATCH "http://localhost:4000/api/v1/model_registry/model-registry/model_versions/1?namespace=kubeflow" \
     -H "Content-Type: application/json" \
-d '{ "data": {
  "description": "New description 2"
}}'
```

```
# GET /v1/model_registry/{model_registry_id}/registered_models/{registered_model_id}/versions
curl -i -H "kubeflow-userid: user@example.com" "localhost:4000/api/v1/model_registry/model-registry/registered_models/1/versions?namespace=kubeflow"
```

```
# POST /v1/model_registry/{model_registry_id}/registered_models/{registered_model_id}/versions
curl -i -H "kubeflow-userid: user@example.com" -X POST "http://localhost:4000/api/v1/model_registry/model-registry/registered_models/1/versions?namespace=kubeflow" \
     -H "Content-Type: application/json" \
     -d '{ "data": {
  "customProperties": {
    "my-label9": {
      "metadataType": "MetadataStringValue",
      "string_value": "val"
    }
  },
  "description": "Description",
  "externalId": "9928",
  "name": "ModelVersion One",
  "state": "LIVE",
  "author": "alex",
  "registeredModelId": "1"
}}'
```

```
# GET /api/v1/model_registry/{model_registry_id}/model_versions/{model_version_id}/artifacts
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/model_versions/1/artifacts?namespace=kubeflow"
```

```
# POST /api/v1/model_registry/{model_registry_id}/model_versions/{model_version_id}/artifacts
curl -i -H "kubeflow-userid: user@example.com" -X POST "http://localhost:4000/api/v1/model_registry/model-registry/model_versions/1/artifacts?namespace=kubeflow" \
     -H "Content-Type: application/json" \
     -d '{ "data": {
  "customProperties": {
    "my-label9": {
      "metadataType": "MetadataStringValue",
      "string_value": "val"
    }
  },
  "description": "New description",
  "externalId": "9927",
  "name": "ModelArtifact One",
  "state": "LIVE",
  "artifactType": "TYPE_ONE"
}}'
```

```
# GET /api/v1/model_registry/{model_registry_id}/artifacts
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/artifacts?namespace=kubeflow"

```

```
# GET /api/v1/model_registry/{model_registry_id}/artifacts/{artifact_id}
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/artifacts/{artifact_id}?namespace=kubeflow"

```

```
# POST /api/v1/model_registry/{model_registry_id}/artifacts
curl -i \
  -H "kubeflow-userid: user@example.com" \
  -X POST "http://localhost:4000/api/v1/model_registry/model-registry/artifacts?namespace=kubeflow" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "artifactType": "model-artifact",
      "name": "dora-classifier-v2",
      "description": "MNIST digit classification model trained on TensorFlow",
      "uri": "gs://my-models/mnist-classifier/v2",
      "externalId": "model-12345678",
      "modelFormatName": "tensorflow",
      "modelFormatVersion": "2.9.0",
      "storageKey": "models/mnist/1.0.0",
      "storagePath": "/models/mnist/1.0.0/model.savedmodel"
    }
  }'
```

```
# PATCH /api/v1/model_registry/{model_registry_id}/artifacts/{artifact_id}
curl -i -H "kubeflow-userid: user@example.com" -X PATCH "http://localhost:4000/api/v1/model_registry/model-registry/artifacts/1?namespace=kubeflow" \
     -H "Content-Type: application/json" \
-d '{ "data": {
  "artifactType": "model-artifact",
  "description": "New description 2"
}}'
```

```
# GET /api/v1/model_catalog/models (source parameter is required)
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_catalog/models?namespace=kubeflow&source=sample-source"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_catalog/models?namespace=kubeflow&source=sample-source"
```

```
# GET /api/v1/model_catalog/sources
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_catalog/sources?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_catalog/sources?namespace=kubeflow"
```

```
# GET /api/v1/model_catalog/models/filter_options
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_catalog/models/filter_options?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_catalog/models/filter_options?namespace=kubeflow"
```

```
# GET /api/v1/model_catalog/sources with name filter
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_catalog/sources?namespace=kubeflow&name=sample-source"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_catalog/sources?namespace=kubeflow&name=sample-source"
```

```
# GET /api/v1/model_catalog/sources/{source_id}/models/{model_name}
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_catalog/sources/sample-source/models/model-name?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_catalog/sources/sample-source/models/model-name?namespace=kubeflow"
```

```
# GET /api/v1/model_catalog/sources/{source_id}/artifacts/{model_name}
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_catalog/sources/sample-source/artifacts/model-name?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_catalog/sources/sample-source/artifacts/model-name?namespace=kubeflow"
```

```
# GET /api/v1/model_catalog/labels
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_catalog/labels?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_catalog/labels?namespace=kubeflow"
```

```
# GET /api/v1/settings/model_catalog/source_configs
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/settings/model_catalog/source_configs?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/settings/model_catalog/source_configs?namespace=kubeflow"
```

```
# GET /api/v1/settings/model_catalog/source_configs/{sourceId}
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/settings/model_catalog/source_configs/test-catalog?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/settings/model_catalog/source_configs/test-catalog?namespace=kubeflow"
```

```
# POST /api/v1/settings/model_catalog/source_configs
curl -i \
-H "kubeflow-userid: user@example.com" \
-X POST "http://localhost:4000/api/v1/settings/model_catalog/source_configs?namespace=kubeflow" \
-H "Content-Type: application/json" \
-d '{
  "data":{
    "id":"test-catalog",
    "name":"Test Catalog",
    "type":"yaml",
    "yaml": "yamlContent:\n - name: test-model"
    }
  }'
curl -i -H "Authorization: Bearer $TOKEN" \
-X POST "http://localhost:4000/api/v1/settings/model_catalog/source_configs?namespace=kubeflow" \
-H "Content-Type: application/json" \
-d '{
  "data":{
    "id":"test-catalog",
    "name":"Test Catalog",
    "type":"yaml",
      "yaml": "yamlContent:\n - name: test-model"
    }
  }'
```

```
# PATCH /api/v1/settings/model_catalog/source_configs/{sourceId}
curl -i \
-H "kubeflow-userid: user@example.com" \
-X PATCH "http://localhost:4000/api/v1/settings/model_catalog/source_configs/test-catalog?namespace=kubeflow" \
-H "Content-Type: application/json" \
-d '{
  "data":{
    "id":"test-catalog",
    "name":"Updated Catalog",
    "type":"yaml"
    }
  }'
curl -i -H "Authorization: Bearer $TOKEN" \
-X PATCH "http://localhost:4000/api/v1/settings/model_catalog/source_configs/test-catalog?namespace=kubeflow" \
-H "Content-Type: application/json" \
-d '{
  "data":{
    "id":"test-catalog",
    "name":"Updated Catalog",
    "type":"yaml"
    }
  }'
```

```
# DELETE /api/v1/settings/model_catalog/source_configs/{sourceId}
curl -i -H "kubeflow-userid: user@example.com" -X DELETE "http://localhost:4000/api/v1/settings/model_catalog/source_configs/test-catalog?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" -X DELETE "http://localhost:4000/api/v1/settings/model_catalog/source_configs/test-catalog?namespace=kubeflow"
```

```
# GET api/v1/model_registry/model-registry/model_transfer_jobs
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs?namespace=kubeflow"
```

```
# GET api/v1/model_registry/model-registry/model_transfer_jobs/{job_name}
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs/test-job?namespace=kubeflow&jobNamespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs/test-job?namespace=kubeflow&jobNamespace=kubeflow"
```

```
# GET api/v1/model_registry/model-registry/model_transfer_jobs/{job_name}/events
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs/test-job/events?namespace=kubeflow&jobNamespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs/test-job/events?namespace=kubeflow&jobNamespace=kubeflow"
```

```
# POST /api/v1/model_registry/model-registry/model_transfer_jobs
curl -i \
  -H "kubeflow-userid: user@example.com" \
  -H "Content-Type: application/json" \
  -X POST "http://localhost:4000/api/v1/model_registry/model-registry-service/model_transfer_jobs?namespace=kubeflow" \
  -d '{
  "data": {
    "name": "my-test-job",
    "source": {
      "type": "s3",
      "bucket": "my-bucket",
      "key": "models/path",
      "awsAccessKeyId": "key",
      "awsSecretAccessKey": "secret"
    },
    "destination": {
      "type": "oci",
      "uri": "quay.io/myorg/model:v1",
      "registry": "quay.io",
      "username": "user",
      "password": "pass"
    },
    "uploadIntent": "create_model",
    "registeredModelName": "My Model",
    "modelVersionName": "v1.0.0"
  }
}'

curl -i -H "Authorization: Bearer $TOKEN" \
-X POST "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs?namespace=kubeflow" \
-H "Content-Type: application/json" \
-d '{
  "data": {
    "name": "my-test-job",
    "source": {
      "type": "s3",
      "bucket": "my-bucket",
      "key": "models/path",
      "awsAccessKeyId": "key",
      "awsSecretAccessKey": "secret"
    },
    "destination": {
      "type": "oci",
      "uri": "quay.io/myorg/model:v1",
      "registry": "quay.io",
      "username": "user",
      "password": "pass"
    },
    "uploadIntent": "create_model",
    "registeredModelName": "My Model",
    "modelVersionName": "v1.0.0"
  }
}'
```

```
# PATCH api/v1/model_registry/model-registry/model_transfer_jobs/{job_name}
curl -i \
  -H "kubeflow-userid: user@example.com" \
  -H "Content-Type: application/json" \
  -X PATCH "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs/my-job?namespace=kubeflow" \
  -d '{"data": {"name": "my-job-2", "namespace": "default", "jobDisplayName": "test-job"}}'

curl -i -H "Authorization: Bearer $TOKEN" \
-X PATCH "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs/transfer-job-002?namespace=bella-namespace" \
-H "Content-Type: application/json" \
-d '{"data": {"name": "my-job",  "namespace": "default", "jobDisplayName": "test-job"}}'

```

```
# DELETE api/v1/model_registry/model-registry/model_transfer_jobs/{job_name}
curl -i -H "kubeflow-userid: user@example.com" -X DELETE "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs/transfer-job-001?namespace=kubeflow&jobNamespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" -X DELETE "http://localhost:4000/api/v1/model_registry/model-registry/model_transfer_jobs/transfer-job-001?namespace=kubeflow&jobNamespace=kubeflow"
```

```
# GET /api/v1/mcp_catalog/mcp_servers_filter_options
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/mcp_catalog/mcp_servers_filter_options?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/mcp_catalog/mcp_servers_filter_options?namespace=kubeflow"
```

```
# GET /api/v1/mcp_catalog/mcp_servers
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/mcp_catalog/mcp_servers?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/mcp_catalog/mcp_servers?namespace=kubeflow"
```

```
# GET /api/v1/mcp_catalog/mcp_servers/{server_id}
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/mcp_catalog/mcp_servers/server-1?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/mcp_catalog/mcp_servers/server-1?namespace=kubeflow"
```

```
# GET /api/v1/mcp_catalog/mcp_servers/{server_id}/tools
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/mcp_catalog/mcp_servers/server-1/tools?namespace=kubeflow"
curl -i -H "Authorization: Bearer $TOKEN" "http://localhost:4000/api/v1/mcp_catalog/mcp_servers/server-1/tools?namespace=kubeflow"
```

### Pagination

The following query parameters are supported by "Get All" style endpoints to control pagination.

| Parameter Name | Description                                                                                               |
| -------------- | --------------------------------------------------------------------------------------------------------- |
| pageSize       | Number of entities in each page                                                                           |
| orderBy        | Specifies the order by criteria for listing entities. Available values: CREATE_TIME, LAST_UPDATE_TIME, ID |
| sortOrder      | Specifies the sort order for listing entities. Available values: ASC, DESC. Default: ASC                  |
| nextPageToken  | Token to use to retrieve next page of results.                                                            |

### Sample local calls

```
# Get with a page size of 5 getting a specific page.
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/registered_models?pageSize=5&nextPageToken=CAEQARoCCAE"
```

```
# Get with a page size of 5, order by last update time in descending order.
curl -i -H "kubeflow-userid: user@example.com" "http://localhost:4000/api/v1/model_registry/model-registry/registered_models?pageSize=5&orderBy=LAST_UPDATE_TIME&sortOrder=DESC"
```

### FAQ

#### 1. How do we filter model registry services from other Kubernetes services?

We filter Model Registry services by using the Kubernetes label `component: model-registry. This label helps distinguish Model Registry services from other services in the cluster.

For example, in our service manifest, the `component label is defined as follows:

```yaml
# ...
labels:
  # ...
  component: model-registry
#...
```

You can view the complete Model Registry service manifest [here](https://github.com/kubeflow/hub/blob/main/manifests/kustomize/base/model-registry-service.yaml#L10).

#### 2. What is the structure of the mock Kubernetes environment?

The mock Kubernetes environment is activated when the environment variable `MOCK_K8S_CLIENT` is set to `true`. It is based on `env-test` and is designed to simulate a realistic Kubernetes setup for testing. The mock has the following characteristics:

- **Namespaces**:
  - `kubeflow`
  - `dora-namespace`
  - `bella-namespace`

- **Users**:
  - `user@example.com` (has `cluster-admin` privileges)
  - `doraNonAdmin@example.com` (restricted to the `dora-namespace`)
  - `bellaNonAdmin@example.com` (restricted to the `bella-namespace`)

- **Groups**:
  - `dora-service-group` (has access to `model-registry-dora` inside `dora-namespace`)
  - `dora-namespace-group` (has access to the `dora-namespace`)
- **Services (Model Registries)**:
  - `model-registry`: resides in the `kubeflow` namespace with the label `component: model-registry`.
  - `model-registry-one`: resides in the `kubeflow` namespace with the label `component: model-registry`.
  - `non-model-registry`: resides in the `kubeflow` namespace _without_ the label `component: model-registry`.
  - `model-registry-dora`: resides in the `dora-namespace` namespace with the label `component: model-registry`.

#### 3. Which Authentication methods are supported?

The BFF supports two authentication modes, selectable via the --auth-method flag or AUTH_METHOD environment variable (default: internal):

- `internal`: Uses the credentials of the running backend.
  - If running inside the cluster, it uses the pod’s service account.
  - If running locally (e.g. for development), it uses the current user's active kubeconfig context.
  - In this mode, user identity is passed via the kubeflow-userid and optionally kubeflow-groups headers.
  - This is the default mode and works well with mock clients and local testing.
- `user_token`: Uses a user-provided Bearer token for authentication.
  - The token must be passed in the `Authorization` header using the Bearer schema (e.g., `Authorization: Bearer <token>`).
  - This method works with OIDC-authenticated flows and frontend proxies that preserve standard Bearer tokens.

#### 4. How BFF authorization works?

Authorization is performed using Kubernetes access reviews, validating whether the user (or their groups) can perform certain actions.
There are two review mechanisms depending on the authentication mode:

- Internal mode (auth-method=internal):
  Uses SubjectAccessReview (SAR) to check whether the impersonated user (from kubeflow-userid and kubeflow-groups headers) has the required permissions.
- User token mode (auth-method=user_token): Uses SelfSubjectAccessReview (SSAR), leveraging the Bearer token provided in the `Authorization` header to check the current user's permissions directly.

##### Authorization logic

- Access to Model Registry List (/v1/model_registry):
  - Checks for get and list on services in the target namespace.
  - If the user (or groups, in internal mode) has permission, access is granted.

- Access to Specific Model Registry Endpoints (/v1/model_registry/{model_registry_id}/...):
  - Checks for get on the specific service (identified by model_registry_id) in the namespace.
  - If authorized, access is granted.

##### Overriding Token Header and Prefix

By default, the BFF expects the token to be passed in the standard Authorization header with a Bearer prefix:

```shell
Authorization: Bearer <your-token>
```

If you're integrating with a proxy or tool that uses a custom header (e.g., X-Forwarded-Access-Token without a prefix), you can override this behavior using environment variables or Makefile arguments.

```shell
make run AUTH_METHOD=user_token AUTH_TOKEN_HEADER=X-Forwarded-Access-Token AUTH_TOKEN_PREFIX=""
```

This will configure the BFF to extract the raw token from the following header:

```shell
X-Forwarded-Access-Token: <your-token>
```

#### 5. How do I allow CORS requests from other origins

When serving the UI directly from the BFF there is no need for any CORS headers to be served, by default they are turned off for security reasons.

If you need to enable CORS for any reasons you can add origins to the allow-list in several ways:

##### Via the make command

Add the following parameter to your command: `ALLOWED_ORIGINS` this takes a comma separated list of origins to permit serving to, alternatively you can specify the value `*` to allow all origins, **Note this is not recommended in production deployments as it poses a security risk**

Examples:

```shell
# Allow only the origin http://example.com:8081
make run ALLOWED_ORIGINS="http://example.com:8081"

# Allow the origins http://example.com and http://very-nice.com
make run ALLOWED_ORIGINS="http://example.com,http://very-nice.com"

# Allow all origins
make run ALLOWED_ORIGINS="*"

# Explicitly disable CORS (default behaviour)
make run ALLOWED_ORIGINS=""
```

#### Via environment variable

Setting CORS via environment variable follows the same rules as using the Makefile, simply set the environment variable `ALLOWED_ORIGINS` with the same value as above.

#### Via the command line arguments

Setting CORS via command line arguments follows the same rules as using the Makefile. Simply add the `--allowed-origins=` flag to your command.

Examples:

```shell
./bff --allowed-origins="http://my-domain.com,http://my-other-domain.com"
```

#### 6. How do I disable TLS verification for local Kubeflow installations?

For local Kubeflow installations with self-signed certificates, you may need to disable TLS certificate verification.

**Kubernetes deployment:**

```yaml
env:
  - name: INSECURE_SKIP_VERIFY
    value: "true"
```

**Local development:**

```shell
./bin/bff --insecure-skip-verify
# or
export INSECURE_SKIP_VERIFY=true
```

> **Warning:** Only use in development. Keep TLS verification enabled in production.
