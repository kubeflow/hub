package skillcatalog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSyncLimitsFromEnv_DefaultsWhenUnset(t *testing.T) {
	got := SyncLimitsFromEnv()
	assert.Equal(t, int64(defaultMaxInFlightClones), got.MaxInFlightClones)
	assert.Equal(t, defaultMaxResolveWorkers, got.MaxResolveWorkers)
	assert.Equal(t, defaultMaxRefsPerRepo, got.MaxRefsPerRepo)
}

func TestSyncLimitsFromEnv_ValidOverrides(t *testing.T) {
	t.Setenv(EnvMaxInFlightClones, "100")
	t.Setenv(EnvMaxResolveWorkers, "20")
	t.Setenv(EnvMaxRefsPerRepo, "5")

	got := SyncLimitsFromEnv()
	assert.Equal(t, int64(100), got.MaxInFlightClones)
	assert.Equal(t, 20, got.MaxResolveWorkers)
	assert.Equal(t, 5, got.MaxRefsPerRepo)
}

func TestSyncLimitsFromEnv_InvalidAndNonPositiveKeepDefaults(t *testing.T) {
	t.Setenv(EnvMaxInFlightClones, "not-a-number")
	t.Setenv(EnvMaxResolveWorkers, "0")
	t.Setenv(EnvMaxRefsPerRepo, "-3")

	got := SyncLimitsFromEnv()
	assert.Equal(t, int64(defaultMaxInFlightClones), got.MaxInFlightClones)
	assert.Equal(t, defaultMaxResolveWorkers, got.MaxResolveWorkers)
	assert.Equal(t, defaultMaxRefsPerRepo, got.MaxRefsPerRepo)
}

func TestResolveLimitsFromEnv_DefaultsWhenUnset(t *testing.T) {
	got := ResolveLimitsFromEnv()
	assert.Equal(t, defaultCloneTimeout, got.CloneTimeout)
	assert.Equal(t, defaultMaxRepoSize, got.MaxRepoSizeBytes)
}

func TestResolveLimitsFromEnv_ValidOverrides(t *testing.T) {
	t.Setenv(EnvCloneTimeout, "30s")
	t.Setenv(EnvMaxRepoSizeBytes, "1048576")

	got := ResolveLimitsFromEnv()
	assert.Equal(t, 30*time.Second, got.CloneTimeout)
	assert.Equal(t, int64(1048576), got.MaxRepoSizeBytes)
}

func TestResolveLimitsFromEnv_InvalidKeepsDefaults(t *testing.T) {
	t.Setenv(EnvCloneTimeout, "garbage")
	t.Setenv(EnvMaxRepoSizeBytes, "-1")

	got := ResolveLimitsFromEnv()
	assert.Equal(t, defaultCloneTimeout, got.CloneTimeout)
	assert.Equal(t, defaultMaxRepoSize, got.MaxRepoSizeBytes)
}

func TestCredentialsDirFromEnv(t *testing.T) {
	assert.Equal(t, defaultGitCredentialsDir, CredentialsDirFromEnv(), "default when unset")

	t.Setenv(EnvGitCredentialsDir, "/custom/creds")
	assert.Equal(t, "/custom/creds", CredentialsDirFromEnv())
}
