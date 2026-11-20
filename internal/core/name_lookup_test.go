package core_test

import (
	"testing"

	"github.com/kubeflow/hub/pkg/api"
	"github.com/kubeflow/hub/pkg/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertNameMatchedLiterally checks that "_" and "%" in a looked-up name are not
// treated as SQL LIKE wildcards: "a_b" must not find "a-b", and "a%" must not find
// anything that merely starts with "a".
func assertNameMatchedLiterally(t *testing.T, prefix string, create func(name string) string, find func(name string) (string, error)) {
	t.Helper()

	create(prefix + "-v2")

	_, err := find(prefix + "_v2")
	require.ErrorIs(t, err, api.ErrNotFound, "%s_v2 must not match %s-v2", prefix, prefix)
	_, err = find(prefix + "%")
	require.ErrorIs(t, err, api.ErrNotFound, "%s%% must not match %s-v2", prefix, prefix)

	id := create(prefix + "_v2")
	found, err := find(prefix + "_v2")
	require.NoError(t, err)
	assert.Equal(t, id, found)
}

func TestGetByParamsMatchesNameLiterally(t *testing.T) {
	service, cleanup := SetupModelRegistryService(t)
	defer cleanup()

	rm, err := service.UpsertRegisteredModel(&openapi.RegisteredModel{Name: "literal-parent-model"})
	require.NoError(t, err)
	mv, err := service.UpsertModelVersion(&openapi.ModelVersion{Name: "literal-parent-version"}, rm.Id)
	require.NoError(t, err)
	experiment, err := service.UpsertExperiment(&openapi.Experiment{Name: "literal-parent-experiment"})
	require.NoError(t, err)
	env, err := service.UpsertServingEnvironment(&openapi.ServingEnvironment{Name: "literal-parent-env"})
	require.NoError(t, err)

	t.Run("registered model", func(t *testing.T) {
		assertNameMatchedLiterally(t, "iris",
			func(name string) string {
				created, err := service.UpsertRegisteredModel(&openapi.RegisteredModel{Name: name})
				require.NoError(t, err)
				return *created.Id
			},
			func(name string) (string, error) {
				found, err := service.GetRegisteredModelByParams(&name, nil)
				if err != nil {
					return "", err
				}
				return *found.Id, nil
			})
	})

	t.Run("model version", func(t *testing.T) {
		assertNameMatchedLiterally(t, "version",
			func(name string) string {
				created, err := service.UpsertModelVersion(&openapi.ModelVersion{Name: name}, rm.Id)
				require.NoError(t, err)
				return *created.Id
			},
			func(name string) (string, error) {
				found, err := service.GetModelVersionByParams(&name, rm.Id, nil)
				if err != nil {
					return "", err
				}
				return *found.Id, nil
			})
	})

	t.Run("model artifact", func(t *testing.T) {
		assertNameMatchedLiterally(t, "weights",
			func(name string) string {
				created, err := service.UpsertModelVersionArtifact(&openapi.Artifact{
					ModelArtifact: &openapi.ModelArtifact{Name: &name, Uri: new("s3://bucket/" + name)},
				}, *mv.Id)
				require.NoError(t, err)
				return *created.ModelArtifact.Id
			},
			func(name string) (string, error) {
				found, err := service.GetModelArtifactByParams(&name, mv.Id, nil)
				if err != nil {
					return "", err
				}
				return *found.Id, nil
			})
	})

	t.Run("artifact", func(t *testing.T) {
		assertNameMatchedLiterally(t, "readme",
			func(name string) string {
				created, err := service.UpsertModelVersionArtifact(&openapi.Artifact{
					DocArtifact: &openapi.DocArtifact{Name: &name, Uri: new("s3://bucket/" + name)},
				}, *mv.Id)
				require.NoError(t, err)
				return *created.DocArtifact.Id
			},
			func(name string) (string, error) {
				found, err := service.GetArtifactByParams(&name, mv.Id, nil)
				if err != nil {
					return "", err
				}
				return *found.DocArtifact.Id, nil
			})
	})

	t.Run("experiment", func(t *testing.T) {
		assertNameMatchedLiterally(t, "tuning",
			func(name string) string {
				created, err := service.UpsertExperiment(&openapi.Experiment{Name: name})
				require.NoError(t, err)
				return *created.Id
			},
			func(name string) (string, error) {
				found, err := service.GetExperimentByParams(&name, nil)
				if err != nil {
					return "", err
				}
				return *found.Id, nil
			})
	})

	t.Run("experiment run", func(t *testing.T) {
		assertNameMatchedLiterally(t, "run",
			func(name string) string {
				created, err := service.UpsertExperimentRun(&openapi.ExperimentRun{Name: &name}, experiment.Id)
				require.NoError(t, err)
				return *created.Id
			},
			func(name string) (string, error) {
				found, err := service.GetExperimentRunByParams(&name, experiment.Id, nil)
				if err != nil {
					return "", err
				}
				return *found.Id, nil
			})
	})

	t.Run("serving environment", func(t *testing.T) {
		assertNameMatchedLiterally(t, "staging",
			func(name string) string {
				created, err := service.UpsertServingEnvironment(&openapi.ServingEnvironment{Name: name})
				require.NoError(t, err)
				return *created.Id
			},
			func(name string) (string, error) {
				found, err := service.GetServingEnvironmentByParams(&name, nil)
				if err != nil {
					return "", err
				}
				return *found.Id, nil
			})
	})

	t.Run("inference service", func(t *testing.T) {
		assertNameMatchedLiterally(t, "predictor",
			func(name string) string {
				created, err := service.UpsertInferenceService(&openapi.InferenceService{
					Name:                 &name,
					ServingEnvironmentId: *env.Id,
					RegisteredModelId:    *rm.Id,
				})
				require.NoError(t, err)
				return *created.Id
			},
			func(name string) (string, error) {
				found, err := service.GetInferenceServiceByParams(&name, env.Id, nil)
				if err != nil {
					return "", err
				}
				return *found.Id, nil
			})
	})
}
