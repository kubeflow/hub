# Kubeflow Hub

> ⚠️  NOTE: UNSTABLE API  ⚠️
> This library is provided as a convenience for Kubeflow Hub developers.
> If you are not actively involved in the development of Hub, please prefer the [REST API](https://editor.swagger.io/?url=https://raw.githubusercontent.com/kubeflow/hub/main/api/openapi/model-registry.yaml).

## Getting Started

Model Registry is a high level Go library client for recording and retrieving metadata associated with ML developer and data scientist workflows that provides a [high-level model metadata registry API](../pkg/api/api.go).

You can use Model Registry to manage [convenient type definitions](../pkg/openapi/) for models, model versions, artifacts, serving environments, inference services, and serve models.

### Prerequisites

* Go >= 1.26

Install it using:

```bash
go get github.com/kubeflow/hub
```

Assuming that an Model Registry server is running at `localhost:8080`, you can connect with it using:

```go
import (
  "net/http"
  "github.com/kubeflow/hub/pkg/openapi"
)

cfg := &openapi.Configuration{
  HTTPClient: http.DefaultClient,
  Servers: openapi.ServerConfigurations{
    {
      URL: "http://localhost:8080",
    },
  },
}

client := openapi.NewAPIClient(cfg)
```

Once the connection is established, you can create a `ModelRegistryServiceAPI` service:

```go
service := client.ModelRegistryServiceAPI
```

### Example usage

After setting up your Model Registry Service, you can try some of the following:

#### Creating models

Create a `RegisteredModel`

```go
modelName := "MODEL_NAME"
modelDescription := "MODEL_DESCRIPTION"

// register a new model
registeredModel, _, err := service.CreateRegisteredModel(ctx).RegisteredModelCreate(openapi.RegisteredModelCreate{
  Name: modelName,
  Description: &modelDescription,
}).Execute()
if err != nil {
  return nil, fmt.Errorf("error registering model: %w", err)
}
```

Create a `ModelVersion` for the registered model

```go
versionName := "VERSION_NAME"
versionDescription := "VERSION_DESCRIPTION"
versionScore := 0.83

// register model version
modelVersion, _, err := service.CreateModelVersion(ctx).ModelVersionCreate(openapi.ModelVersionCreate{
  Name:              versionName,
  Description:       &versionDescription,
  RegisteredModelId: *registeredModel.Id,
  CustomProperties: &map[string]openapi.MetadataValue{
    "score": {
      MetadataDoubleValue: &openapi.MetadataDoubleValue{
        DoubleValue: &versionScore,
      },
    },
  },
}).Execute()
if err != nil {
  return nil, fmt.Errorf("unable to create model version: %w", err)
}
```

Create a `ModelArtifact` for the version

```go
artifactName := "ARTIFACT_NAME"
artifactDescription := "ARTIFACT_DESCRIPTION"
artifactUri := "ARTIFACT_URI"
artifactType := "model-artifact"

// register model artifact
modelArtifact, _, err = service.UpsertModelVersionArtifact(ctx, *modelVersion.Id).Artifact(openapi.Artifact{
  ModelArtifact: &openapi.ModelArtifact{
    Name:        &artifactName,
    Description: &artifactDescription,
    Uri:         &artifactUri,
    ArtifactType: &artifactType,
  },
}).Execute()
if err != nil {
  return nil, fmt.Errorf("unable to create model artifact: %w", err)
}
```

#### Updating models

The Model Registry Service provides `Upsert*` methods for all supported models, meaning that you can use them to
in**sert** a new model, or **up**date it:

```go
artifactName := "ARTIFACT_NAME"
artifactDescription := "ARTIFACT_DESCRIPTION"
artifactUri := "ARTIFACT_URI"
artifactType := "model-artifact"

// register model artifact
modelArtifact, _, err = service.UpsertModelVersionArtifact(ctx, *modelVersion.Id).Artifact(openapi.Artifact{
  ModelArtifact: &openapi.ModelArtifact{
    Name:        &artifactName,
    Description: &artifactDescription,
    Uri:         &artifactUri,
    ArtifactType: &artifactType,
  },
}).Execute()
if err != nil {
  return nil, fmt.Errorf("unable to create model artifact: %w", err)
}

// update model artifact
newDescription := "update it!"

modelArtifact, _, err = service.UpsertModelVersionArtifact(ctx, *modelVersion.Id).Artifact(openapi.Artifact{
  ModelArtifact: &openapi.ModelArtifact{
    Name:        &artifactName,
    Description: &newDescription,
    Uri:         &artifactUri,
    ArtifactType: &artifactType,
  },
}).Execute()
if err != nil {
  return nil, fmt.Errorf("unable to update model artifact: %w", err)
}
```


#### Querying models

Get a `RegisteredModel` by name:

```go
modelName := "MODEL_NAME"
registeredModel, _, err := service.FindRegisteredModel(ctx).Name(modelName).Execute()
if err != nil {
  return nil, fmt.Errorf("unable to find registered model: %w", err)
}
```

Get all `ModelVersion` associated to a specific registered model

```go
allVersions, _, err := service.FindModelVersion(ctx).ParentResourceId(*regModelFound.Id).Execute()
if err != nil {
  return nil, fmt.Errorf("unable to find model version: %w", err)
}
```

## API documentation

Check out the [Model Registry Service interface](../pkg/api/api.go) and the [core layer implementation](../pkg/core/) for additional details.
