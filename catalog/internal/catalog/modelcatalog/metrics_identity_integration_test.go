package modelcatalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/models"
	modelservice "github.com/kubeflow/hub/catalog/internal/catalog/modelcatalog/service"
	"github.com/kubeflow/hub/catalog/internal/db/service"
	"github.com/kubeflow/hub/catalog/internal/testhelpers"
	"github.com/kubeflow/hub/internal/platform/datastore"
	dbmodels "github.com/kubeflow/hub/internal/platform/db/entity"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests exercise ingestion and database uniqueness together: using raw
// producer IDs either rolls back the batch or conflates different metrics types.
func TestMetricsIdentityIngestion(t *testing.T) {
	for _, backend := range []struct {
		name  string
		setup func(*testing.T, *datastore.Spec) (*gorm.DB, func())
	}{
		{"postgres", testutils.SetupPostgresWithMigrations},
		{"mysql", func(t *testing.T, spec *datastore.Spec) (*gorm.DB, func()) {
			require.NoError(t, testutils.SetupSharedMySQL())
			t.Cleanup(testutils.CleanupSharedMySQL)
			return testutils.SetupMySQLWithMigrations(t, spec)
		}},
	} {
		t.Run(backend.name, func(t *testing.T) {
			db, cleanup := backend.setup(t, testhelpers.MustDatastoreSpec(t))
			t.Cleanup(cleanup)
			modelTypeID := testhelpers.GetCatalogModelTypeIDForDBTest(t, db)
			metricsTypeID := testhelpers.GetCatalogMetricsArtifactTypeIDForDBTest(t, db)
			modelRepo := modelservice.NewCatalogModelRepository(db, modelTypeID)
			metricsRepo := modelservice.NewCatalogMetricsArtifactRepository(db, metricsTypeID)
			artifactRepo := service.NewCatalogArtifactRepository(db, map[string]int32{
				service.CatalogMetricsArtifactTypeName: metricsTypeID,
			})
			createModel := func(t *testing.T, source string) models.CatalogModel {
				t.Helper()
				model, err := modelRepo.Save(&models.CatalogModelImpl{
					Attributes: &models.CatalogModelAttributes{Name: new("vendor/shared-model")},
					Properties: &[]dbmodels.Properties{{Name: "source_id", StringValue: &source}},
				})
				require.NoError(t, err)
				return model
			}
			load := func(t *testing.T, dir string, model models.CatalogModel) int {
				t.Helper()
				count, err := processModelDirectory(dir, modelRepo, metricsRepo, modelTypeID, metricsTypeID, *model.GetAttributes().Name)
				require.NoError(t, err)
				return count
			}
			list := func(t *testing.T, model models.CatalogModel) []models.CatalogMetricsArtifact {
				t.Helper()
				artifacts, err := metricsRepo.List(models.CatalogMetricsArtifactListOptions{ParentResourceID: model.GetID()})
				require.NoError(t, err)
				return artifacts.Items
			}

			t.Run("shared files and duplicate records", func(t *testing.T) {
				first := createModel(t, "identity-first")
				second := createModel(t, "identity-second")
				ids := []string{"shared", "Case", "case", "a:b/c", strings.Repeat("long-id", 100)}
				dir := writeMetricsIdentityFiles(t, ids, true)
				wantCount := 1 + 3*len(ids)
				require.Equal(t, wantCount, load(t, dir, first))
				require.Equal(t, wantCount, load(t, dir, second))
				secondBefore := list(t, second)
				storedIDs := map[string]bool{}
				for _, model := range []models.CatalogModel{first, second} {
					artifacts := list(t, model)
					require.Len(t, artifacts, wantCount)
					var attributions int64
					require.NoError(t, db.Model(&schema.Attribution{}).Where("context_id = ?", *model.GetID()).Count(&attributions).Error)
					assert.EqualValues(t, wantCount, attributions)
					for _, artifact := range artifacts {
						attrs := artifact.GetAttributes()
						require.NotNil(t, attrs.ExternalID)
						require.NotNil(t, attrs.Name)
						assert.LessOrEqual(t, len(*attrs.Name), 255)
						assert.LessOrEqual(t, len(*attrs.ExternalID), 255)
						assert.False(t, storedIDs[*attrs.ExternalID], "identity reused across models or metrics types")
						storedIDs[*attrs.ExternalID] = true
						filtered, err := metricsRepo.List(models.CatalogMetricsArtifactListOptions{
							ParentResourceID: model.GetID(), ExternalID: attrs.ExternalID,
						})
						require.NoError(t, err)
						require.Len(t, filtered.Items, 1)
						assert.Equal(t, artifact.GetID(), filtered.Items[0].GetID())
						if attrs.MetricsType == models.MetricsTypeAccuracy {
							assert.Equal(t, fmt.Sprintf("accuracy-metrics-model-%d", *model.GetID()), *attrs.ExternalID)
							continue
						}
						mapped, err := mapToMetricsArtifact(artifact, string(attrs.MetricsType))
						require.NoError(t, err)
						apiArtifact := mapped.CatalogMetricsArtifact
						assert.Equal(t, attrs.ExternalID, apiArtifact.ExternalId)
						assert.Equal(t, attrs.Name, apiArtifact.Name)
						assert.Equal(t, "1700000000000", *apiArtifact.CreateTimeSinceEpoch)
						assert.Equal(t, "1700000001000", *apiArtifact.LastUpdateTimeSinceEpoch)
						props := map[string]dbmodels.Properties{}
						for _, prop := range *artifact.GetCustomProperties() {
							props[prop.Name] = prop
							_, found := apiArtifact.CustomProperties[prop.Name]
							assert.True(t, found, "API lost custom property %s", prop.Name)
						}
						require.NotNil(t, props["id"].StringValue)
						assert.Contains(t, ids, *props["id"].StringValue)
						require.NotNil(t, apiArtifact.CustomProperties["id"].MetadataStringValue)
						assert.Equal(t, *props["id"].StringValue, apiArtifact.CustomProperties["id"].MetadataStringValue.StringValue)
						require.NotNil(t, props["result"].DoubleValue)
						assert.Equal(t, 0.25, *props["result"].DoubleValue, "first duplicate should win")
					}
					assert.Zero(t, load(t, dir, model))
					assert.Equal(t, artifacts, list(t, model), "repeat ingestion must preserve artifacts")
				}
				if backend.name == "postgres" {
					t.Run("reload and source removal", func(t *testing.T) {
						// Catalog lifecycle methods currently use PostgreSQL-specific SQL.
						require.NoError(t, artifactRepo.DeleteByParentID(service.CatalogMetricsArtifactTypeName, *first.GetID()))
						assert.Equal(t, wantCount, load(t, dir, first))
						assert.Equal(t, secondBefore, list(t, second))
						require.NoError(t, modelRepo.DeleteBySource("identity-first"))
						assert.Empty(t, list(t, first))
						assert.Equal(t, secondBefore, list(t, second))
					})
				}
			})

			t.Run("producer ID resembling a scoped identity", func(t *testing.T) {
				model := createModel(t, "identity-shaped-producer-id")
				assert.Equal(t, 4, load(t, writeMetricsIdentityFiles(t, []string{"initial"}, false), model))
				var producerID string
				for _, artifact := range list(t, model) {
					if artifact.GetAttributes().MetricsType == models.MetricsTypeEvaluation {
						producerID = *artifact.GetAttributes().ExternalID
					}
				}
				require.NotEmpty(t, producerID)
				dir := writeMetricsIdentityFiles(t, []string{"initial", producerID}, false)
				assert.Equal(t, 3, load(t, dir, model), "a scoped identity is not a legacy producer ID")
				assert.Zero(t, load(t, dir, model))
				require.Len(t, list(t, model), 7)
			})

			for _, legacyType := range []models.MetricsType{models.MetricsTypeEvaluation, models.MetricsTypePerformance, models.MetricsTypeSecurityMetrics} {
				t.Run("legacy "+string(legacyType), func(t *testing.T) {
					first := createModel(t, "legacy-first-"+string(legacyType))
					second := createModel(t, "legacy-second-"+string(legacyType))
					producerID := "legacy-" + string(legacyType)
					legacy, err := metricsRepo.Save(&models.CatalogMetricsArtifactImpl{
						Attributes: &models.CatalogMetricsArtifactAttributes{
							Name: new("old-" + producerID), ExternalID: &producerID, MetricsType: legacyType,
						},
					}, first.GetID())
					require.NoError(t, err)
					dir := writeMetricsIdentityFiles(t, []string{producerID}, false)
					// Only the matching kind is skipped for the legacy owner.
					assert.Equal(t, 3, load(t, dir, first))
					assert.Equal(t, 4, load(t, dir, second))
					assert.Zero(t, load(t, dir, first))
					require.Len(t, list(t, first), 4)
					preserved, err := metricsRepo.GetByID(*legacy.GetID())
					require.NoError(t, err)
					assert.Equal(t, producerID, *preserved.GetAttributes().ExternalID)
					if backend.name == "postgres" {
						secondBefore := list(t, second)
						require.NoError(t, artifactRepo.DeleteByParentID(service.CatalogMetricsArtifactTypeName, *first.GetID()))
						assert.Equal(t, 4, load(t, dir, first))
						for _, artifact := range list(t, first) {
							assert.NotEqual(t, producerID, *artifact.GetAttributes().ExternalID)
						}
						assert.Equal(t, secondBefore, list(t, second))
					}
				})
			}

			t.Run("legacy ID occupying the scoped namespace", func(t *testing.T) {
				model := createModel(t, "legacy-scoped-namespace")
				// Use the prospective identity as a legacy producer ID. It must
				// cause a reported database conflict, not silently hide a result.
				legacyID := metricsArtifactIdentity(*model.GetID(), models.MetricsTypeEvaluation, "new-result")
				_, err := metricsRepo.Save(&models.CatalogMetricsArtifactImpl{
					Attributes: &models.CatalogMetricsArtifactAttributes{
						Name: new("evaluation-" + legacyID), ExternalID: &legacyID, MetricsType: models.MetricsTypeEvaluation,
					},
				}, model.GetID())
				require.NoError(t, err)
				dir := writeMetricsIdentityFiles(t, []string{"new-result"}, false)
				_, err = processModelDirectory(dir, modelRepo, metricsRepo, modelTypeID, metricsTypeID, *model.GetAttributes().Name)
				require.ErrorContains(t, err, "failed to batch save artifacts")
				require.Len(t, list(t, model), 1, "conflicting batch must be rolled back")
				if backend.name == "postgres" {
					require.NoError(t, artifactRepo.DeleteByParentID(service.CatalogMetricsArtifactTypeName, *model.GetID()))
					assert.Equal(t, 7, load(t, writeMetricsIdentityFiles(t, []string{"new-result", legacyID}, false), model))
				}
			})
		})
	}
}

func writeMetricsIdentityFiles(t *testing.T, ids []string, duplicates bool) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.json"), []byte(`{"id":"vendor/shared-model"}`), 0600))
	for _, filename := range []string{"evaluations.ndjson", "performance.ndjson", "security-evaluations.ndjson"} {
		var contents strings.Builder
		for _, id := range ids {
			scores := []float64{0.25}
			if duplicates {
				scores = append(scores, 0.75)
			}
			for _, score := range scores {
				data, err := json.Marshal(map[string]any{
					"id": id, "model_id": "vendor/shared-model", "run_id": "run-1",
					"evaluation": "Language evaluation", "category": "general-llm", "benchmark": "mmlu",
					"description": "Language understanding", "result": score, "result_metric": "accuracy",
					"created_at": int64(1700000000000), "updated_at": int64(1700000001000),
				})
				require.NoError(t, err)
				contents.Write(data)
				contents.WriteByte('\n')
			}
		}
		require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(contents.String()), 0600))
	}
	return dir
}
