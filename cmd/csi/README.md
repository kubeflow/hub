# Model Registry Custom Storage Initializer

This is a Model Registry specific implementation of a KServe Custom Storage Initializer (CSI).
More details on what `Custom Storage Initializer` is can be found in the [KServe doc](https://kserve.github.io/website/docs/model-serving/storage/storage-containers).

## Implementation

The Model Registry CSI is a simple Go executable that basically takes two positional arguments:
1. __Source URI__: identifies the `storageUri` set in the `InferenceService`, this must be a model-registry custom URI, i.e., `model-registry://...`
2. __Destination Path__: the location where the model should be stored, e.g., `/mnt/models`

The core logic of this CSI is pretty simple and it consists of three main steps:
1. Parse the custom URI in order to extract `registered model name` and `model version`
2. Query the model registry in order to retrieve the original model location (e.g., `http`, `s3`, `gcs` and so on)
3. Use `github.com/kserve/kserve/pkg/agent/storage` pkg to actually download the model from well-known protocols.

### Workflow

The below sequence diagram should highlight the workflow when this CSI is injected into the KServe pod deployment.

```mermaid
sequenceDiagram
    actor U as User
    participant MR as Model Registry
    participant KC as KServe Controller
    participant MD as Model Deployment (Pod)
    participant MRSI as Model Registry Storage Initializer
    U->>+MR: Register ML Model
    MR-->>-U: Indexed Model
    U->>U: Create InferenceService CR
    Note right of U: The InferenceService should<br/>point to the model registry<br/>indexed model, e.g.,:<br/> model-registry://<model-registry-url>/<model>/<version>
    KC->>KC: React to InferenceService creation
    KC->>+MD: Create Model Deployment
    MD->>+MRSI: Initialization (Download Model)
    MRSI->>MRSI: Parse URI
    MRSI->>+MR: Fetch Model Metadata
    MR-->>-MRSI: Model Metadata
    Note over MR,MRSI: The main information that is fetched is the artifact URI which specifies the real model location, e.g.,: https://.. or s3://...
    MRSI->>MRSI: Download Model
    Note right of MRSI: The storage initializer will use<br/> the KServe default providers<br/> to download the model<br/> based on the artifact URI
    MRSI-->>-MD: Downloaded Model
    MD->>-MD: Deploy Model
```


## Get Started

Please look at [Get Started](./GET_STARTED.md) guide for a very simple quickstart that showcases how this custom storage initializer can be used for ML models serving operations.

## Development

### Build the executable

You can just run:
```bash
make build
```

Which wil create the executable under `bin/mr-storage-initializer`.

### Run the executable

You can run `main.go` (without building the executable) by running:
```bash
./bin/mr-storage-initializer "model-registry://model-registry-url/model/version" "./"
```

or directly running the `main.go` skipping the previous step:
```bash
make SOURCE_URI=model-registry://model-registry-url/model/version DEST_PATH=./ run
```

> [!NOTE]
> `model-registry-url` is optional, if not provided the value of `MODEL_REGISTRY_BASE_URL` env variable will be used.

> [!NOTE]
> A Model Registry service should be up and running at `localhost:8080`.

### Build container image

Using a fixed version of the model-registry library:
```bash
make docker-build
```

Or, using the local model-registry module:
```bash
make docker-build-dev
```

By default the container image name is `quay.io/${USER}/model-registry-storage-initializer:latest` but it can be overridden providing the `IMG` env variable, e.g., `make IMG=abc/ORG/NAME:TAG docker-build`.

### Push container image

Issue the following command:
```bash
make [IMG=..] docker-push
```
