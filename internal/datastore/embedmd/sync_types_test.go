package embedmd

import (
	"context"
	"testing"

	"github.com/kubeflow/hub/internal/db/service"
	"github.com/kubeflow/hub/internal/platform/datastore"
	"github.com/kubeflow/hub/internal/platform/db/mysql"
	"github.com/kubeflow/hub/internal/platform/db/schema"
	"github.com/kubeflow/hub/internal/platform/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cont_mysql "github.com/testcontainers/testcontainers-go/modules/mysql"
)

// TestSyncTypesDeterministic verifies that syncTypes produces the same type IDs
// on a fresh database regardless of map iteration order. This is critical for
// the emptyDir recovery path: RunMigrations recreates rows from scratch and the
// plugins must end up with the same IDs they had before the data loss.
func TestSyncTypesDeterministic(t *testing.T) {
	ctx := context.Background()

	mysqlContainer, err := cont_mysql.Run(ctx, "mysql:8.3",
		cont_mysql.WithDatabase("test"),
		cont_mysql.WithUsername("root"),
		cont_mysql.WithPassword("root"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = mysqlContainer.Terminate(ctx) })

	dsn, err := mysqlContainer.ConnectionString(ctx)
	require.NoError(t, err)

	dbConnector := mysql.NewMySQLDBConnector(dsn, &tls.TLSConfig{})
	conn, err := dbConnector.Connect()
	require.NoError(t, err)

	// Create just the Type and TypeProperty tables without seeding any mlmd.* rows.
	// This isolates syncTypes from the migration seed data, letting us test that
	// two calls on an empty table produce identical auto-increment assignments.
	require.NoError(t, conn.AutoMigrate(&schema.Type{}, &schema.TypeProperty{}))

	// A spec with multiple types across all three categories mirrors real usage.
	spec := datastore.NewSpec().
		AddArtifact("kf.ArtifactA", datastore.NewSpecType(service.NewModelArtifactRepository).AddString("uri")).
		AddArtifact("kf.ArtifactB", datastore.NewSpecType(service.NewDocArtifactRepository).AddString("uri")).
		AddContext("kf.ContextA", datastore.NewSpecType(service.NewRegisteredModelRepository)).
		AddContext("kf.ContextB", datastore.NewSpecType(service.NewModelVersionRepository)).
		AddExecution("kf.ExecutionA", datastore.NewSpecType(service.NewModelVersionRepository))

	svc := &EmbedMDService{dbConnector: dbConnector}

	// First syncTypes run.
	require.NoError(t, svc.syncTypes(conn, spec))
	rs1, err := newRepoSet(conn, spec)
	require.NoError(t, err)
	typeMap1 := rs1.TypeMap()

	// Simulate data loss: TRUNCATE resets MySQL auto-increment to 1,
	// reproducing the exact state of a freshly created database.
	require.NoError(t, conn.Exec("TRUNCATE TABLE TypeProperty").Error)
	require.NoError(t, conn.Exec("TRUNCATE TABLE Type").Error)

	// Second syncTypes run on the fresh empty tables.
	require.NoError(t, svc.syncTypes(conn, spec))
	rs2, err := newRepoSet(conn, spec)
	require.NoError(t, err)
	typeMap2 := rs2.TypeMap()

	assert.Equal(t, typeMap1, typeMap2, "type IDs must be identical after DB recreation")
}
