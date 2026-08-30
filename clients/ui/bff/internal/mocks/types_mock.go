package mocks

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/kubeflow/hub/pkg/openapi-v1"
)

func GenerateMockRegisteredModelList() openapi.RegisteredModelList {
	var models []openapi.RegisteredModel
	for i := 0; i < 2; i++ {
		model := GenerateMockRegisteredModel()
		models = append(models, model)
	}

	return openapi.RegisteredModelList{
		NextPageToken: gofakeit.UUID(),
		PageSize:      int32(gofakeit.Number(1, 20)),
		Size:          int32(len(models)),
		Items:         models,
	}
}

func GenerateMockRegisteredModel() openapi.RegisteredModel {
	model := openapi.RegisteredModel{
		CustomProperties: map[string]openapi.MetadataValue{
			"example_key": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue:  gofakeit.Sentence(3),
					MetadataType: "string",
				},
			},
		},
		Description:              stringToPointer(gofakeit.Sentence(5)),
		ExternalId:               stringToPointer(gofakeit.UUID()),
		Name:                     gofakeit.Name(),
		Id:                       stringToPointer(gofakeit.UUID()),
		CreateTimeSinceEpoch:     randomEpochTime(),
		LastUpdateTimeSinceEpoch: randomEpochTime(),
		Owner:                    stringToPointer(gofakeit.Name()),
		State:                    stateToPointer(openapi.RegisteredModelState(gofakeit.RandomString([]string{string(openapi.REGISTEREDMODELSTATE_LIVE), string(openapi.REGISTEREDMODELSTATE_ARCHIVED)}))),
	}
	return model
}

func GenerateMockModelVersion() openapi.ModelVersion {
	model := openapi.ModelVersion{
		CustomProperties: map[string]openapi.MetadataValue{
			"example_key": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue:  gofakeit.Sentence(3),
					MetadataType: "string",
				},
			},
		},
		Description:              stringToPointer(gofakeit.Sentence(5)),
		ExternalId:               stringToPointer(gofakeit.UUID()),
		Name:                     gofakeit.Name(),
		Id:                       stringToPointer(gofakeit.UUID()),
		CreateTimeSinceEpoch:     randomEpochTime(),
		LastUpdateTimeSinceEpoch: randomEpochTime(),
		Author:                   stringToPointer(gofakeit.Name()),
		State:                    stateToPointer(openapi.ModelVersionState(gofakeit.RandomString([]string{string(openapi.MODELVERSIONSTATE_LIVE), string(openapi.MODELVERSIONSTATE_ARCHIVED)}))),
	}
	return model
}

func GenerateMockModelVersionList() openapi.ModelVersionList {
	var versions []openapi.ModelVersion

	for i := 0; i < 2; i++ {
		version := GenerateMockModelVersion()
		versions = append(versions, version)
	}

	return openapi.ModelVersionList{
		NextPageToken: gofakeit.UUID(),
		PageSize:      int32(gofakeit.Number(1, 20)),
		Size:          int32(len(versions)),
		Items:         versions,
	}
}

func GenerateMockModelArtifact() openapi.ModelArtifact {
	artifact := openapi.ModelArtifact{
		ArtifactType: stringToPointer("model-artifact"),
		CustomProperties: map[string]openapi.MetadataValue{
			"example_key": {
				MetadataStringValue: &openapi.MetadataStringValue{
					StringValue:  gofakeit.Sentence(3),
					MetadataType: "string",
				},
			},
		},
		Description:              stringToPointer(gofakeit.Sentence(5)),
		ExternalId:               stringToPointer(gofakeit.UUID()),
		Uri:                      stringToPointer(gofakeit.URL()),
		State:                    randomArtifactState(),
		Name:                     stringToPointer(gofakeit.Name()),
		Id:                       stringToPointer(gofakeit.UUID()),
		CreateTimeSinceEpoch:     randomEpochTime(),
		LastUpdateTimeSinceEpoch: randomEpochTime(),
		ModelFormatName:          stringToPointer(gofakeit.Name()),
		StorageKey:               stringToPointer(gofakeit.Word()),
		StoragePath:              stringToPointer("/" + gofakeit.Word() + "/" + gofakeit.Word()),
		ModelFormatVersion:       stringToPointer(gofakeit.AppVersion()),
		ServiceAccountName:       stringToPointer(gofakeit.Username()),
	}
	return artifact
}

func GenerateMockModelArtifactList() openapi.ModelArtifactList {
	var artifacts []openapi.ModelArtifact

	for i := 0; i < 2; i++ {
		artifact := GenerateMockModelArtifact()
		artifacts = append(artifacts, artifact)
	}

	return openapi.ModelArtifactList{
		NextPageToken: gofakeit.UUID(),
		PageSize:      int32(gofakeit.Number(1, 20)),
		Size:          int32(len(artifacts)),
		Items:         artifacts,
	}
}

func GenerateMockPageValues() url.Values {
	pageValues := url.Values{}

	pageValues.Add("pageSize", strconv.Itoa(gofakeit.Number(1, 100)))
	pageValues.Add("orderBy", gofakeit.RandomString([]string{"CREATE_TIME", "LAST_UPDATE_TIME", "ID"}))
	pageValues.Add("sortOrder", gofakeit.RandomString([]string{"ASC", "DESC"}))
	pageValues.Add("nextPageToken", gofakeit.UUID())

	return pageValues
}

func randomEpochTime() *string {
	return stringToPointer(fmt.Sprintf("%d", gofakeit.Date().UnixMilli()))
}

func randomArtifactState() *openapi.ArtifactState {
	return stateToPointer(openapi.ArtifactState(gofakeit.RandomString([]string{
		string(openapi.ARTIFACTSTATE_LIVE),
		string(openapi.ARTIFACTSTATE_DELETED),
		string(openapi.ARTIFACTSTATE_ABANDONED),
		string(openapi.ARTIFACTSTATE_MARKED_FOR_DELETION),
		string(openapi.ARTIFACTSTATE_PENDING),
		string(openapi.ARTIFACTSTATE_REFERENCE),
		string(openapi.ARTIFACTSTATE_UNKNOWN),
	})))
}

func stateToPointer[T any](s T) *T {
	return &s
}

func stringToPointer(s string) *string {
	return &s
}

func boolToPointer(b bool) *bool {
	return &b
}
