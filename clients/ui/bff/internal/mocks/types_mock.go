package mocks

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/brianvoe/gofakeit/v7"
	openapiv1 "github.com/kubeflow/hub/pkg/openapi-v1"
)

func GenerateMockRegisteredModelList() openapiv1.RegisteredModelList {
	var models []openapiv1.RegisteredModel
	for i := 0; i < 2; i++ {
		model := GenerateMockRegisteredModel()
		models = append(models, model)
	}

	return openapiv1.RegisteredModelList{
		NextPageToken: gofakeit.UUID(),
		PageSize:      int32(gofakeit.Number(1, 20)),
		Size:          int32(len(models)),
		Items:         models,
	}
}

func GenerateMockRegisteredModel() openapiv1.RegisteredModel {
	model := openapiv1.RegisteredModel{
		CustomProperties: map[string]openapiv1.MetadataValue{
			"example_key": {
				MetadataStringValue: &openapiv1.MetadataStringValue{
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
		State:                    stateToPointer(openapiv1.RegisteredModelState(gofakeit.RandomString([]string{string(openapiv1.REGISTEREDMODELSTATE_LIVE), string(openapiv1.REGISTEREDMODELSTATE_ARCHIVED)}))),
	}
	return model
}

func GenerateMockModelVersion() openapiv1.ModelVersion {
	model := openapiv1.ModelVersion{
		CustomProperties: map[string]openapiv1.MetadataValue{
			"example_key": {
				MetadataStringValue: &openapiv1.MetadataStringValue{
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
		State:                    stateToPointer(openapiv1.ModelVersionState(gofakeit.RandomString([]string{string(openapiv1.MODELVERSIONSTATE_LIVE), string(openapiv1.MODELVERSIONSTATE_ARCHIVED)}))),
	}
	return model
}

func GenerateMockModelVersionList() openapiv1.ModelVersionList {
	var versions []openapiv1.ModelVersion

	for i := 0; i < 2; i++ {
		version := GenerateMockModelVersion()
		versions = append(versions, version)
	}

	return openapiv1.ModelVersionList{
		NextPageToken: gofakeit.UUID(),
		PageSize:      int32(gofakeit.Number(1, 20)),
		Size:          int32(len(versions)),
		Items:         versions,
	}
}

func GenerateMockModelArtifact() openapiv1.ModelArtifact {
	artifact := openapiv1.ModelArtifact{
		ArtifactType: stringToPointer("model-artifact"),
		CustomProperties: map[string]openapiv1.MetadataValue{
			"example_key": {
				MetadataStringValue: &openapiv1.MetadataStringValue{
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

func GenerateMockModelArtifactList() openapiv1.ModelArtifactList {
	var artifacts []openapiv1.ModelArtifact

	for i := 0; i < 2; i++ {
		artifact := GenerateMockModelArtifact()
		artifacts = append(artifacts, artifact)
	}

	return openapiv1.ModelArtifactList{
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

func randomArtifactState() *openapiv1.ArtifactState {
	return stateToPointer(openapiv1.ArtifactState(gofakeit.RandomString([]string{
		string(openapiv1.ARTIFACTSTATE_LIVE),
		string(openapiv1.ARTIFACTSTATE_DELETED),
		string(openapiv1.ARTIFACTSTATE_ABANDONED),
		string(openapiv1.ARTIFACTSTATE_MARKED_FOR_DELETION),
		string(openapiv1.ARTIFACTSTATE_PENDING),
		string(openapiv1.ARTIFACTSTATE_REFERENCE),
		string(openapiv1.ARTIFACTSTATE_UNKNOWN),
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
