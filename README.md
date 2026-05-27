# Kubeflow Hub

![build checks status](https://github.com/kubeflow/hub/actions/workflows/build.yml/badge.svg?branch=main)
[![codecov](https://codecov.io/github/kubeflow/hub/graph/badge.svg?token=61URLQA3VS)](https://codecov.io/github/kubeflow/hub)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubeflow%2Fmodel-registry.svg?type=shield&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubeflow%2Fmodel-registry?ref=badge_shield&issueType=license)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/9937/badge)](https://www.bestpractices.dev/projects/9937)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/kubeflow/hub/badge)](https://scorecard.dev/viewer/?uri=github.com/kubeflow/hub)

Kubeflow Hub is an umbrella project including the **Model Registry**, which provides a central repository for model developers to store and manage models, versions, and artifact metadata; and the **[Catalog](catalog/README.md)**, which enables federated discovery across external catalogs.

The project was formerly known as Model Registry.

## Red Hat's Pledge
- Red Hat drives the project's development through Open Source principles, ensuring transparency, sustainability, and community ownership.
- Red Hat values the Kubeflow community and commits to providing a minimum of 12 months' notice before ending project maintenance after the initial release.

> **Alpha**
>
> This Kubeflow component has alpha status with limited support. See the [Kubeflow versioning policies](https://www.kubeflow.org/docs/started/support/#application-status). The Kubeflow team is interested in your [feedback](https://github.com/kubeflow/hub) about the usability of the feature.

## Documentation links:

1. Introduction
 - [What is Kubeflow Model Registry](https://www.kubeflow.org/docs/components/model-registry/overview/)
 - [Blog KF 1.9 introducing Model Registry](https://blog.kubeflow.org/kubeflow-1.9-release/#model-registry)
 - [Blog KF 1.10 introducing UI for Model Registry, CSI, and other features](https://blog.kubeflow.org/kubeflow-1.10-release/#model-registry)
2. Installation
 - [installing Model Registry standalone](https://www.kubeflow.org/docs/components/model-registry/installation/#standalone-installation)
 - [installing Model Registry with Kubeflow manifests](https://github.com/kubeflow/manifests/tree/master/applications/model-registry/upstream#readme)
 - [installing Model Registry using ODH Operator](https://github.com/opendatahub-io/model-registry-operator/tree/main/docs#readme)
3. Concepts
 - [Logical Model](./docs/logical_model.md)
4. Python client
 - [installing and using the Model Registry Python client](https://model-registry.readthedocs.io/en/latest/)
5. Tutorials
 - [end-to-end tutorial](https://www.kubeflow.org/docs/components/model-registry/getting-started/)
 - [demonstration video](https://www.youtube.com/watch?v=JVxUTkAKsMU)
6. [FAQs](#faq)
7. Development
 - [introduction to local build and development](#pre-requisites)
 - [contributing](./CONTRIBUTING.md)
 - [Kubeflow community and the Model Registry working group](https://www.kubeflow.org/docs/about/community/)
 - REST API
   - [OpenAPI definition](https://editor.swagger.io/?url=https://raw.githubusercontent.com/kubeflow/hub/main/api/openapi/model-registry.yaml)
   - [playground](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/kubeflow/hub/main/api/openapi/model-registry.yaml)
 - [license scanning](https://github.com/kubeflow/hub/issues/323)
 - [monitoring image quality](https://github.com/kubeflow/hub/issues/327)
8. [UI](clients/ui/README.md)

## Pre-requisites:
- go >= 1.26
- protoc v24.3 - [Protocol Buffers v24.3 Release](https://github.com/protocolbuffers/protobuf/releases/tag/v24.3)
- npm >= 10.2.0 - [Installing Node.js and npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)
- Java >= 11.0
- python 3.9

## OpenAPI Proxy Server

The Hub backend server follows a contract-first approach, where the contract is identified by [model-registry.yaml](api/openapi/model-registry.yaml) OpenAPI specification.

You can also easily display the latest OpenAPI contract for model-registry in a Swagger-like editor directly from the documentation site, [here](https://www.kubeflow.org/docs/components/model-registry/reference/rest-api/#swagger-ui).
### Starting the OpenAPI Proxy Server
Run the following command to start the OpenAPI proxy server from source:

```shell
make run/proxy
```
The backend service implements the OpenAPI defined in [model-registry.yaml](api/openapi/model-registry.yaml) to create a Model Registry specific REST API.

### Model registry logical model

For a high-level documentation of the Model Registry _logical model_, please check [this guide](./docs/logical_model.md).

## Model Registry Core

The model registry core is the layer which implements the core/business logic by interacting with the underlying datastore internal service.
It provides a model registry domain-specific [api](pkg/api/api.go) that is in charge of proxying all, appropriately transformed, requests to the datastore internal service.

### Model registry library

For more background on Model Registry Go core library and instructions on using it, please check [getting started guide](./docs/mr_go_library.md).

## Development

### Database Schema Changes

When making changes to the database schema, you need to regenerate the GORM structs. This is done using the `gen/gorm` target:

```bash
make gen/gorm
```

This target will:
1. Start a temporary database
2. Run migrations
3. Generate GORM structs based on the schema
4. Clean up the temporary database

> **NOTE:** The target requires Docker to be running.

### Building
Run the following command to build the server binary:

```shell
make build
```

The generated binary uses `spf13` cmdline args. More information on using the server can be obtained by running the command:

```shell
./model-registry --help
```

Run the following command to clean the server binary, generated models and etc.:

```shell
make clean
```

### Testing

Run the following command to trigger all tests:

```shell
make test
```

or, to see the statement coverage:

```shell
make test-cover
```

### Docker Image
#### Building the docker image
The following command builds a docker image for the server with the tag `model-registry`:

```shell
docker build -t model-registry .
```

Note that the first build will be longer as it downloads the build tool dependencies.
Subsequent builds will re-use the cached tools layer.

#### Running the proxy server

The following command starts the proxy server:

```shell
docker run -d -p <hostname>:<port>:8080 --user <uid>:<gid> --name server model-registry proxy -n 0.0.0.0
```

Where, `<uid>`, `<gid>`, and `<host-path>` are the same as in the migrate command above.
And `<hostname>` and `<port>` are the local ip and port to use to expose the container's default `8080` listening port.
The server listens on `localhost` by default, hence the `-n 0.0.0.0` option allows the server port to be exposed.

#### Running Kubeflow Hub

> **NOTE:** Docker Compose or Podman Compose must be installed in your environment.

There are two `docker-compose` files that make the startup easier:

- `docker-compose.yaml` - Uses pre-built images from registry
- `docker-compose-local.yaml` - Builds Hub from source

Both files support MySQL and PostgreSQL databases using profiles.

#### Using Makefile targets (recommended)

The easiest way to run the services is using the provided Makefile targets:

```shell
# Start with MySQL (using pre-built images)
make compose/up

# Start with PostgreSQL (using pre-built images)
make compose/up/postgres

# Start with MySQL (builds from source)
make compose/local/up

# Start with PostgreSQL (builds from source)
make compose/local/up/postgres

# Stop services
make compose/down  # or compose/local/down

# Clean up all volumes and networks
make compose/clean
```

#### Manual docker-compose usage

Alternatively, you can run the compose files directly:

```shell
# Using pre-built images with MySQL
docker-compose --profile mysql up

# Using pre-built images with PostgreSQL
DB_TYPE=postgres docker-compose --profile postgres up

# Building from source with PostgreSQL
DB_TYPE=postgres docker-compose -f docker-compose-local.yaml --profile postgres up
```

The Makefile automatically detects whether to use `docker-compose`, `podman-compose`, or `docker compose` based on what's available on your system.

### Testing architecture

The following diagram illustrates testing strategy for the several components in the Hub project:

![](/docs/Model%20Registry%20Testing%20areas.drawio.png)

Go layers components are tested with Unit Tests written in Go, as well as Integration Tests leveraging Testcontainers.
This allows to verify the expected "Core layer" of logical data mapping developed and implemented in Go, matches technical expectations.

Python client is also tested with Unit Tests and Integration Tests written in Python.

End-to-end testing is developed with KinD and Pytest; this higher-lever layer of testing is used to demonstrate *User Stories* from high level perspective.

## Related Components

### Model Catalog Service
- [Model Catalog Service](catalog/README.md) - Federated model discovery across external catalogs

### Kubernetes Components
- [Controller](cmd/controller/README.md) - Kubernetes controller for model registry CRDs
- [CSI Driver](cmd/csi/README.md) - Container Storage Interface for model artifacts

### Client Components
- [UI Backend for Frontend (BFF)](clients/ui/bff/README.md) - Go-based BFF service for the React UI
- [UI Frontend](clients/ui/frontend/README.md) - React-based frontend application

### Job Components
- [Async Upload Job](jobs/async-upload/README.md) - Background job for handling asynchronous model uploads

### Development & Deployment
- [Development Environment](devenv/README.md) - Local development setup and tools
- [Kubernetes Manifests](manifests/kustomize/README.md) - Kustomize-based Kubernetes deployment manifests

## FAQ

### How do I delete metadata resources using the Model Registry API?

MR utilizes a common `ARCHIVED` status for all types.
To delete something, simply update its status.

### How do I check reasons and re-run for FOSSA failures against a PR on this repo?

Follow the link from the GitHub PR page "View details" link of the specific FOSSA action which is failing. This will take you to a FOSSA page for that specific check for that PR. Address the failures listed in FOSSA as most appropriate; then "🔁 Run policy scan" button from that same page.
Please note: re-running the scan from the _main_ FOSSA page of the project/repo does not achieve the same result.
You will need an account for the [CNCF FOSSA instance](https://contribute.cncf.io/resources/services/hosted-tools/) to access all related FOSSA pages.

### How do I get an account for the CNCF FOSSA instance?

You need to provide your email, name, and mention you're a project Approver (refer [OWNERS file](OWNERS)) to the [CNCF Service Desk](http://servicedesk.cncf.io/), requesting access to the [CNCF FOSSA instance](https://contribute.cncf.io/resources/services/hosted-tools/).
Please note: you do not need to create an account on fossa.com on your own; if you already have an account, you'll need to ask migration to the CNCF Service Desk.
If you cannot access the CNCF Service Desk, ask one of your fellow Approvers for help!

## Tips
### Pull image rate limiting

Occasionally you may encounter an 'ImagePullBackOff' error when deploying the Hub project's manifests. See example below for the `model-registry-db` container.

```
Failed to pull image “mysql:8.3.0”: rpc error: code = Unknown desc = fetching target platform image selected from image index: reading manifest sha256:f9097d95a4ba5451fff79f4110ea6d750ac17ca08840f1190a73320b84ca4c62 in docker.io/library/mysql: toomanyrequests: You have reached your pull rate limit. You may increase the limit by authenticating and upgrading: https://www.docker.com/increase-rate-limit
```

This error is triggered by the rate limits from docker.io; in this example specifically about the image `mysql:8.3.0` (the expanded reference is `docker.io/library/mysql:8.3.0`). To mitigate this error you could [authenticate using image pull secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/) for _local development_; or replace the image used with alternative mirrored images, for instance with the following example:
```
manifests/kustomize/overlays/db/model-registry-db-deployment.yaml file.

spec.template.spec.containers.image: public.ecr.aws/docker/library/mysql:8.3.0
```
