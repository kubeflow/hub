# Model Registry Python Client

[![Python](https://img.shields.io/badge/python%20-3.9%7C3.10%7C3.11%7C3.12-blue)](https://github.com/kubeflow/hub)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](../../../LICENSE)
[![Read the Docs](https://img.shields.io/readthedocs/model-registry)](https://model-registry.readthedocs.io/en/latest/)
[![Tutorial Website](https://img.shields.io/badge/Website-green?style=plastic&label=Tutorial&labelColor=blue)](https://www.kubeflow.org/docs/components/model-registry/getting-started/)

This is the Python SDK for the **Model Registry** component of [Kubeflow Hub](https://github.com/kubeflow/hub). It provides a high-level interface for interacting with a model registry server.

> **Alpha**
>
> This Kubeflow component has **alpha** status with limited support.
> See the [Kubeflow versioning policies](https://www.kubeflow.org/docs/started/support/#application-status).
> The Kubeflow team is interested in your [feedback](https://github.com/kubeflow/hub) about the usability of the feature.

## Installation

In your Python environment, you can install the latest version of the Model Registry Python client with:

```
pip install model-registry
```

### Installing extras

Some capabilities of this Model Registry Python client, such as [importing model from Hugging Face](#importing-from-hugging-face-hub),
require additional dependencies.

By [installing an extra variant](https://packaging.python.org/en/latest/tutorials/installing-packages/#installing-extras) of this package
the additional dependencies will be managed for you automatically, for instance with:

```
pip install 'model-registry[hf]'
```

This step is not required if you already installed the additional dependencies already, for instance with:

```
pip install huggingface-hub
```
#### Extras that can be installed
```
pip install 'model-registry[hf]'
```
```
pip install 'model-registry[s3]'
```
```
pip install 'model-registry[olot]'
```
```
pip install 'model-registry[signing]'
```

## Basic usage

### Connecting to MR

You can connect to a secure Model Registry using the default constructor (recommended):

```py
from model_registry import ModelRegistry

registry = ModelRegistry("https://server-address", author="Ada Lovelace")  # Defaults to a secure connection via port 443
```

Or you can set the `is_secure` flag to `False` to connect **without** TLS (not recommended):

```py
registry = ModelRegistry("http://server-address", 8080, author="Ada Lovelace", is_secure=False)  # insecure port set to 8080
```

### Registering models

To register your first model, you can use the `register_model` method:

```py

model = registry.register_model(
    "my-model",  # model name
    "https://storage-place.my-company.com",  # model URI
    version="2.0.0",
    version_description="lorem ipsum",
    model_format_name="onnx",
    model_format_version="1",
    storage_key="my-data-connection",
    storage_path="path/to/model",
    metadata={
        # can be one of the following types
        "int_key": 1,
        "bool_key": False,
        "float_key": 3.14,
        "str_key": "str_value",
    }
)

model = registry.get_registered_model("my-model")
print(model)

version = registry.get_model_version("my-model", "2.0.0")
print(version)

artifact = registry.get_model_artifact("my-model", "2.0.0")
print(artifact)
```

You can also update your models:

```py
# change is not reflected on pushed model version
version.description = "Updated model version"

# you can update it using
registry.update(version)
```

### Importing from S3

When registering models stored on S3-compatible object storage, you should use `utils.s3_uri_from` to build an
unambiguous URI for your artifact.

```py
from model_registry import utils

model = registry.register_model(
    "my-model",  # model name
    uri=utils.s3_uri_from("path/to/model", "my-bucket"),
    version="2.0.0",
    version_description="lorem ipsum",
    model_format_name="onnx",
    model_format_version="1",
    storage_key="my-data-connection",
    metadata={
        # can be one of the following types
        "int_key": 1,
        "bool_key": False,
        "float_key": 3.14,
        "str_key": "str_value",
    }
)
```

### Importing from Hugging Face Hub

To import models from Hugging Face Hub, start by installing the `huggingface-hub` package, either directly or as an
extra (available as `model-registry[hf]`). Reference section "[installing extras](#installing-extras)" above for
more information.

Models can be imported with

```py
hf_model = registry.register_hf_model(
    "hf-namespace/hf-model",  # HF repo
    "relative/path/to/model/file.onnx",
    version="1.2.3",
    model_name="my-model",
    version_description="lorem ipsum",
    model_format_name="onnx",
    model_format_version="1",
)
```

There are caveats to be noted when using this method:

- It's only possible to import a single model file per Hugging Face Hub repo right now.

### Listing models

To list models you can use
```py
for model in registry.get_registered_models():
    ... # your logic using `model` loop variable here

# and versions associated with a model
for version in registry.get_model_versions("my-model"):
    ... # your logic using `version` loop variable here
```

<!-- see https://github.com/kubeflow/hub/issues/358 until fixed, the below is just easier not to mention in the doc.

You can also use `order_by_creation_time`, `order_by_update_time`, or `order_by_id` to change the sorting order

```py
latest_updates = registry.get_model_versions("my-model").order_by_update_time().descending()
for version in latest_updates:
    ...
```

By default, all queries will be `ascending`, but this method is also available for explicitness. -->

> Advanced usage note: You can also set the `page_size()` that you want the Pager to use when invoking the Model Registry backend.
> When using it as an iterator, it will automatically manage pages for you.

#### Implementation notes

The pager will manage pages for you in order to prevent infinite looping.
Currently, the Model Registry backend treats model lists as a circular buffer, and **will not end iteration** for you.

### Uploading local models to external storage and registering them

To both upload and register a model, use the convenience method `upload_artifact_and_register_model`.

This method supports both s3-based storage (via [boto3](https://github.com/boto/boto3)) as well as OCI-based image registries (via [olot](https://github.com/containers/olot), using either of the CLI tools [skopeo](https://github.com/containers/skopeo) or [oras](https://github.com/oras-project/oras))

In order to utilize this method you must instantiate an `upload_params` object which contains the necessary locations and credentials needed to perform the upload to that storage provider.

#### S3 based external storage

Common S3 env vars will be automatically read, such ass the access_key_id, etc. It can also be provided explicitly in the `S3Params` object if desired.

```python
from model_registry.utils import S3Params

s3_upload_params = S3Params(
    bucket_name="my-bucket",
    s3_prefix="models/my_fraud_model",
)

registered_model = registry.upload_artifact_and_register_model(
    name="hello_world_model",
    model_files_path="/home/user-01/models/model_training_01",
    # If the model consists of a single file, such as a .onnx file, you can specify that as well
    # model_files_path="/home/user-01/models/model_training_01.onnx"
    author="Mr. Trainer",
    version="0.0.1",
    upload_params=s3_upload_params
)
```

#### OCI-registry based storage
First, you must ensure you are logged in the to appropriate OCI registry using
`skopeo login`, `podman login`, or you can authenticate via the `OCIParams` by either:

1. Reading a pull secret JSON from an environment variable. Default is `.dockerconfigjson`.

    ```python
    OCIParams(
        ...,
        oci_auth_env_var="DOCKER_CONFIG_JSON"
    )
    ```

2. Passing in username and password directly.

    ```python
    OCIParams(
        ...,
        oci_username="user",
        oci_password="userpass"
    )
    ```

Full example:

```python
from model_registry.utils import OCIParams

oci_upload_params = OCIParams(
    base_image="busybox",
    oci_ref="registry.example.com/acme_org/hello_world_model:0.0.1"
)

registered_model = registry.upload_artifact_and_register_model(
    name="hello_world_model",
    model_files_path="/home/user-01/models/model_training_01",
    # If the model consists of a single file, such as a .onnx file, you can specify that as well
    # model_files_path="/home/user-01/models/model_training_01.onnx"
    author="Mr. Trainer",
    version="0.0.1",
    upload_params=oci_upload_params
)
```

Additionally, OCI-based storage supports multiple CLI clients to perform the upload. However, one of these clients must be available in the hosts `$PATH`. **Ensure your host has either [skopeo](https://github.com/containers/skopeo) or [oras](https://github.com/oras-project/oras) installed and available.**

By default, `skopeo` is used to perform the OCI image download/upload.

If you prefer to use `oras` instead, you can specify it like so:

```python
oci_upload_params = OCIParams(
    base_image="busybox",
    oci_ref="registry.example.com/acme_org/hello_world_model:0.0.1",
    backend="oras"
)
```

Additionally, if neither of these CLI clients are sufficient for you, you can provide a `custom_oci_backend` in the `OCIParams` and specify the appropriate methods

```python
def is_available():
    pass
def pull():
    pass
def push():
    pass

custom_oci_backend = {
    "is_available": is_available,
    "pull": pull,
    "push": push,
}

oci_upload_params = OCIParams(
    base_image="busybox",
    oci_ref="registry.example.com/acme_org/hello_world_model:0.0.1",
    custom_oci_backend=custom_oci_backend,
)
```

### Signing and Verifying Models

The Model Registry Python client supports signing and verifying both model artifacts and container images using [Sigstore](https://www.sigstore.dev/).

To use signing features, install the signing extra:

```
pip install 'model-registry[signing]'
```

#### Quickstart

Set the following environment variables to configure Sigstore:

| Environment Variable | Description |
|---|---|
| `SIGSTORE_TUF_URL` | TUF server URL |
| `SIGSTORE_FULCIO_URL` | Fulcio server URL |
| `SIGSTORE_REKOR_URL` | Rekor server URL |
| `SIGSTORE_TSA_URL` | TSA server URL |

When running on Kubernetes, the identity token is automatically read from the service account token at `/var/run/secrets/kubernetes.io/serviceaccount/token`. Other values such as `oidc_issuer`, `client_id`, and `certificate_identity` are extracted from the token when not explicitly provided.

With the environment configured, signing and verifying is straightforward:

```python
from model_registry.signing import Signer

signer = Signer()

# Sign and verify a model directory
signer.sign_model("/path/to/model")
signer.verify_model("/path/to/model")

# Sign and verify a container image (requires digest reference)
signer.sign_image("quay.io/user/image@sha256:abc123...")
signer.verify_image("quay.io/user/image@sha256:abc123...")
```

#### Explicit Configuration

Instead of environment variables, you can pass configuration directly:

```python
signer = Signer(
    tuf_url="https://tuf.example.com",
    fulcio_url="https://fulcio.example.com",
    rekor_url="https://rekor.example.com",
    identity_token_path="/path/to/token.jwt",
    certificate_identity="user@example.com",
    oidc_issuer="https://accounts.example.com",
)
```

#### Model Signing Options

By default, the signature is written to `model.sig` inside the model directory. You can customize the output path and exclude files from signing:

```python
signer.sign_model(
    "/path/to/model",
    signature_path="/custom/path/model.sig",
    ignore_paths=[".cache", "__pycache__"],
)
```

#### Pre-downloading Trust Metadata

Trust metadata is downloaded automatically when needed. If you want to pre-cache the trust config or ensure ahead of time that this step won't fail, call `initialize()` explicitly:

```python
signer.initialize()
```

#### Logging

You can control the log level for signing operations:

```python
import logging

signer = Signer(log_level=logging.DEBUG)
# Or change it later
signer.set_log_level(logging.WARNING)
```

### Running ModelRegistry on Ray or Uvloop
When running `ModelRegistry` on a platform that sets a custom event loop that cannot be nested, an error will occur.

To solve this, you can specify a custom `async_runner` when initializing the client, one that is compatible with your environment.

`async_runner` is a function or a method that takes in a coroutine.


Example of an async runner compatible with Ray or Uvloop can be found [here](tests/extras/async_task_runner.py) in `tests/extras`.

Example usage:
```py
atr = AsyncTaskRunner()
registry = ModelRegistry("http://server-address", 8080, author="Ada Lovelace", async_runner=atr.run)
```

See also the [test case](tests/test_client.py#L854) in `test_custom_async_runner_with_ray`.

Please keep in mind, the `AsyncTaskRunner` used here for testing does not ship within the library so you will need to copy it into your code directly or import from elsewhere.

## Development

### Using the Makefile

The `Makefile` contains most common development tasks

To install dependencies:

```bash
make
```

Then you can run tests:

```bash
make test test-e2e
```

Then you can run fuzz tests:

```bash
make test-fuzz
```

### Using Nox

Common tasks, such as building documentation and running tests, can be executed using [`nox`](https://github.com/wntrblm/nox) sessions.

Use `nox -l` to list sessions and execute them using `nox -s [session]`.

### Testing requirements

To run the e2e tests you will need [kind](https://kind.sigs.k8s.io/) to be installed. This is necessary as the e2e test suite will manage a Model Registry deployment and an MLMD deployment to ensure a clean MR target on each run.

### Running Locally on Mac M1 or M2 (arm64 architecture)

Check out our [recommendations on setting up your docker engine](https://github.com/kubeflow/hub/blob/main/CONTRIBUTING.md#docker-engine) on an ARM processor.


### Troubleshooting

- On running `make test test-e2e` if you see a similar problem `unknown flag: --load`, install [buildx](https://formulae.brew.sh/formula/docker-buildx). You will then need to add `cliPluginsExtraDirs` to ~/.docker/config.json, like so:
```
"cliPluginsExtraDirs": [
      "/opt/homebrew/lib/docker/cli-plugins"  # depending on your system config, brew should give you the proper one
  ]
```
Before running make, ensure docker is running (`docker ps -a`). If it's not, assuming you're using `colima` on macos, run
`colima start`.

- On running `make test-e2e` you might see an error similar to
```
102.7 /workspace/bin/golangci-lint run cmd/... internal/... ./pkg/...  --timeout 3m
124.6 make: *** [Makefile:243: lint] Killed
------
Dockerfile:60
--------------------
  58 |
  59 |     # prepare the build in a separate layer
  60 | >>> RUN make clean build/prepare
  61 |     # compile separately to optimize multi-platform builds
  62 |     RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} make build/compile
--------------------
ERROR: failed to solve: process "/bin/sh -c make clean build/prepare" did not complete successfully: exit code: 2
make[1]: *** [image/build] Error 1
make: *** [deploy-latest-mr] Error 2
```
To solve it, you can try launching `colima` with these settings:
```
colima start --cpu 6 --memory 16 --profile docker --arch aarch64 --vm-type=vz --vz-rosetta
```

<!-- github-only -->
