package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/kubeflow/hub/internal/platform/db/postgres"
	_tls "github.com/kubeflow/hub/internal/platform/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	cont_postgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

// Type represents the Type table structure
type Type struct {
	ID          int64  `gorm:"primaryKey"`
	Name        string `gorm:"column:name"`
	Version     string `gorm:"column:version"`
	ExternalID  string `gorm:"column:external_id"`
	Description string `gorm:"column:description"`
}

func (Type) TableName() string {
	return "Type"
}

// Package-level shared database instance
var (
	sharedDB          *gorm.DB
	postgresContainer *cont_postgres.PostgresContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Create Postgres container once for all tests
	container, err := cont_postgres.Run(
		ctx,
		"postgres:16",
		cont_postgres.WithUsername("postgres"),
		cont_postgres.WithPassword("testpass"),
		cont_postgres.WithDatabase("testdb"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		panic("Failed to start Postgres container: " + err.Error())
	}
	postgresContainer = container

	defer func() {
		if sharedDB != nil {
			if sqlDB, err := sharedDB.DB(); err == nil {
				sqlDB.Close() //nolint:errcheck
			}
		}

		if postgresContainer != nil {
			testcontainers.TerminateContainer(postgresContainer) //nolint:errcheck
		}
	}()

	// Connect to the database
	dbConnector := postgres.NewPostgresDBConnector(postgresContainer.MustConnectionString(ctx), &_tls.TLSConfig{})
	sharedDB, err = dbConnector.Connect()
	if err != nil {
		panic("Failed to connect to database: " + err.Error())
	}

	// Run all tests
	code := m.Run()

	os.Exit(code)
}

// cleanupTestData truncates tables and drops indexes to provide clean state between tests
func cleanupTestData(t *testing.T, db *gorm.DB) {
	// First, drop any indexes that might conflict with migrations
	dropIndexes := []string{
		"idx_artifact_uri",
		"idx_artifact_create_time_since_epoch",
		"idx_artifact_last_update_time_since_epoch",
		"idx_artifact_external_id",
		"idx_context_name",
		"idx_context_create_time_since_epoch",
		"idx_context_last_update_time_since_epoch",
		"idx_context_external_id",
		"idx_execution_name",
		"idx_execution_create_time_since_epoch",
		"idx_execution_last_update_time_since_epoch",
		"idx_execution_external_id",
		"idx_event_artifact_id",
		"idx_event_execution_id",
		"idx_event_milliseconds_since_epoch",
		"idx_parentcontext_parent_id",
		"idx_parentcontext_child_id",
		"idx_association_context_id",
		"idx_association_execution_id",
		"idx_attribution_context_id",
		"idx_attribution_artifact_id",
		"idx_artifactproperty_artifact_id",
		"idx_artifactproperty_name",
		"idx_contextproperty_context_id",
		"idx_contextproperty_name",
		"idx_executionproperty_execution_id",
		"idx_executionproperty_name",
		"idx_typeproperty_type_id",
		"idx_typeproperty_name",
	}

	for _, index := range dropIndexes {
		err := db.Exec("DROP INDEX IF EXISTS " + index).Error
		if err != nil {
			// Log but don't fail - index might not exist
			t.Logf("Could not drop index %s: %v", index, err)
		}
	}

	// List of tables to clean up (in dependency order - dependent tables first)
	tables := []string{
		"ArtifactProperty",
		"ContextProperty",
		"ExecutionProperty",
		"TypeProperty",
		"ParentContext",
		"Association",
		"Attribution",
		"Event",
		"Artifact",
		"Execution",
		"Context",
		"Type",
		"MLMDEnv",
		"schema_migrations",
	}

	// Disable triggers and foreign key constraints temporarily (PostgreSQL-specific)
	err := db.Exec("SET session_replication_role = replica").Error
	require.NoError(t, err)

	// Truncate all tables
	for _, table := range tables {
		err := db.Exec("TRUNCATE TABLE IF EXISTS \"" + table + "\" CASCADE").Error
		if err != nil {
			// Log but don't fail - table might not exist
			t.Logf("Could not truncate table %s: %v", table, err)
		}
	}

	// Re-enable triggers and foreign key constraints (PostgreSQL-specific)
	err = db.Exec("SET session_replication_role = DEFAULT").Error
	require.NoError(t, err)
}

func TestMigrations(t *testing.T) {
	cleanupTestData(t, sharedDB)

	// Create migrator
	migrator, err := postgres.NewPostgresMigrator(sharedDB)
	require.NoError(t, err)

	// Run migrations
	err = migrator.Migrate()
	require.NoError(t, err)

	// Verify MLMDEnv table
	var schemaVersion int
	err = sharedDB.Raw("SELECT schema_version FROM \"MLMDEnv\" LIMIT 1").Scan(&schemaVersion).Error
	require.NoError(t, err)
	assert.Equal(t, 10, schemaVersion)

	// Verify Type table has expected entries
	var count int64
	err = sharedDB.Model(&Type{}).Count(&count).Error
	require.NoError(t, err)
	assert.Greater(t, count, int64(0))
}

// TestTypeNameUniqueMigrationDedupesExistingRows verifies migration 000026
// (uq_type_name) collapses pre-existing duplicate Type rows -- the state a
// database could be in after the TypeRepository.Save race this fix closes --
// before adding the UNIQUE(name) constraint, and repoints a real Artifact
// row that referenced the duplicate being removed.
func TestTypeNameUniqueMigrationDedupesExistingRows(t *testing.T) {
	cleanupTestData(t, sharedDB)

	migrator, err := postgres.NewPostgresMigrator(sharedDB)
	require.NoError(t, err)

	// Bring the shared database to a known state (fully migrated), then step
	// back exactly one migration -- 000026_type_name_unique -- so the
	// UNIQUE(name) constraint does not yet exist and duplicate rows can be
	// inserted directly. Stepping relative to "fully migrated" rather than
	// to an absolute version keeps this independent of how many migrations
	// precede it and of whatever state other tests in this file left behind.
	require.NoError(t, migrator.Migrate())
	downOne := -1
	require.NoError(t, migrator.Down(&downOne))

	// Two duplicate Type rows for the same name, as the pre-fix race could
	// produce: the lower id is the one later migrations/the application
	// expect to survive.
	require.NoError(t, sharedDB.Exec(
		`INSERT INTO "Type" (name, type_kind) VALUES ('test.DedupType', 1)`).Error)
	var survivorID int64
	require.NoError(t, sharedDB.Raw(
		`SELECT id FROM "Type" WHERE name = 'test.DedupType'`).Scan(&survivorID).Error)
	require.NoError(t, sharedDB.Exec(
		`INSERT INTO "Type" (name, type_kind) VALUES ('test.DedupType', 1)`).Error)
	var dupID int64
	require.NoError(t, sharedDB.Raw(
		`SELECT id FROM "Type" WHERE name = 'test.DedupType' AND id != ?`, survivorID).Scan(&dupID).Error)

	// A real Artifact row referencing the duplicate must be repointed to the
	// survivor rather than left dangling once the duplicate is removed.
	require.NoError(t, sharedDB.Exec(
		`INSERT INTO "Artifact" (type_id, name, uri) VALUES (?, 'test-artifact', 'file:///dedup-test')`, dupID).Error)

	// Apply 000026.
	require.NoError(t, migrator.Up(nil))

	var count int64
	require.NoError(t, sharedDB.Raw(`SELECT COUNT(*) FROM "Type" WHERE name = 'test.DedupType'`).Scan(&count).Error)
	assert.Equal(t, int64(1), count, "duplicate Type rows must be collapsed to one survivor")

	var remainingID int64
	require.NoError(t, sharedDB.Raw(`SELECT id FROM "Type" WHERE name = 'test.DedupType'`).Scan(&remainingID).Error)
	assert.Equal(t, survivorID, remainingID, "the lowest id must be the surviving row")

	var artifactTypeID int64
	require.NoError(t, sharedDB.Raw(`SELECT type_id FROM "Artifact" WHERE name = 'test-artifact'`).Scan(&artifactTypeID).Error)
	assert.Equal(t, survivorID, artifactTypeID, "the Artifact row must be repointed to the surviving Type")

	// The constraint itself must now reject a fresh duplicate.
	err = sharedDB.Exec(`INSERT INTO "Type" (name, type_kind) VALUES ('test.DedupType', 1)`).Error
	assert.Error(t, err)
}

func TestDownMigrations(t *testing.T) {
	cleanupTestData(t, sharedDB)

	migrator, err := postgres.NewPostgresMigrator(sharedDB)
	require.NoError(t, err)

	// Run migrations
	err = migrator.Migrate()
	require.NoError(t, err)

	// Down migrations
	err = migrator.Down(nil)
	require.NoError(t, err)

	// Verify tables don't exist (except schema_migrations)
	var count int64
	err = sharedDB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name != 'schema_migrations'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
