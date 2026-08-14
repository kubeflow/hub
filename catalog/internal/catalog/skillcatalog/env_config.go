package skillcatalog

import (
	"os"
	"strconv"
	"time"

	"github.com/golang/glog"
)

// Environment variables that override the skill catalog's compiled-in limits.
// An unset, empty, or unparseable value leaves the corresponding default in
// place, so the defaults hold unless an operator explicitly opts in.
const (
	// EnvMaxInFlightClones overrides the global cap on concurrent git clones
	// across all sources (default 50).
	EnvMaxInFlightClones = "SKILL_CATALOG_MAX_INFLIGHT_CLONES"
	// EnvMaxResolveWorkers overrides the per-source resolve worker-pool size
	// (default 10).
	EnvMaxResolveWorkers = "SKILL_CATALOG_MAX_RESOLVE_WORKERS"
	// EnvMaxRefsPerRepo overrides the maximum refs a single repository may list
	// (default 10).
	EnvMaxRefsPerRepo = "SKILL_CATALOG_MAX_REFS_PER_REPO"
	// EnvCloneTimeout overrides the per-clone timeout, as a Go duration string
	// such as "90s" or "3m" (default 2m).
	EnvCloneTimeout = "SKILL_CATALOG_CLONE_TIMEOUT"
	// EnvMaxRepoSizeBytes overrides the per-clone maximum checked-out size in
	// bytes (default 524288000, i.e. 500 MiB).
	EnvMaxRepoSizeBytes = "SKILL_CATALOG_MAX_REPO_SIZE_BYTES"
	// EnvGitCredentialsDir overrides the directory where the git-credentials Secret
	// is mounted; each file in it is a token named by a repository's credentialRef
	// (default /etc/skill-catalog/git-credentials).
	EnvGitCredentialsDir = "SKILL_CATALOG_GIT_CREDENTIALS_DIR"

	// EnvMarketplaceName overrides the `name` of the rendered Claude Code plugin
	// marketplace (kebab-case; users install with name@<this>). Default
	// "kubeflow-skill-catalog".
	EnvMarketplaceName = "SKILL_CATALOG_MARKETPLACE_NAME"
	// EnvMarketplaceOwner overrides the marketplace owner/maintainer name shown in
	// the rendered marketplace. Default "Kubeflow".
	EnvMarketplaceOwner = "SKILL_CATALOG_MARKETPLACE_OWNER"
	// EnvContentMirror sets the external base URL that marketplace plugin clone
	// URLs are rewritten onto (canonical https://<host>/{org}/{repo}.git ->
	// {base}/{org}/{repo}.git). Unset means canonical upstream URLs are served.
	EnvContentMirror = "SKILL_CONTENT_MIRROR"
	// EnvContentMirrorInternal sets the base URL used instead of EnvContentMirror
	// when a marketplace request carries ?audience=cluster (in-cluster consumers).
	// Unset means the external mirror (or canonical URL) is used for all audiences.
	EnvContentMirrorInternal = "SKILL_CONTENT_MIRROR_INTERNAL"
)

// Defaults for the rendered Claude Code plugin marketplace.
const (
	defaultMarketplaceName  = "kubeflow-skill-catalog"
	defaultMarketplaceOwner = "Kubeflow"
)

// MarketplaceConfigFromEnv builds the marketplace rendering config from the
// environment, falling back to the compiled-in defaults for name and owner and to
// canonical (un-rewritten) clone URLs when no content mirror is configured.
func MarketplaceConfigFromEnv() MarketplaceConfig {
	name := os.Getenv(EnvMarketplaceName)
	if name == "" {
		name = defaultMarketplaceName
	}
	owner := os.Getenv(EnvMarketplaceOwner)
	if owner == "" {
		owner = defaultMarketplaceOwner
	}
	return MarketplaceConfig{
		Name:           name,
		Owner:          owner,
		ExternalMirror: os.Getenv(EnvContentMirror),
		InternalMirror: os.Getenv(EnvContentMirrorInternal),
	}
}

// CredentialsDirFromEnv returns the git-credentials mount directory, overridden by
// EnvGitCredentialsDir when set to a non-empty value; otherwise the default.
func CredentialsDirFromEnv() string {
	if dir := os.Getenv(EnvGitCredentialsDir); dir != "" {
		return dir
	}
	return defaultGitCredentialsDir
}

// SyncLimitsFromEnv returns the sync fan-out limits, with each field overridden
// by its environment variable when set to a valid positive value; otherwise the
// compiled-in default is used.
func SyncLimitsFromEnv() SyncLimits {
	return SyncLimits{
		MaxInFlightClones: envInt64(EnvMaxInFlightClones, defaultMaxInFlightClones),
		MaxResolveWorkers: envInt(EnvMaxResolveWorkers, defaultMaxResolveWorkers),
		MaxRefsPerRepo:    envInt(EnvMaxRefsPerRepo, defaultMaxRefsPerRepo),
	}
}

// ResolveLimitsFromEnv returns the per-clone resolver limits, with each field
// overridden by its environment variable when set to a valid positive value;
// otherwise the compiled-in default is used.
func ResolveLimitsFromEnv() ResolveLimits {
	return ResolveLimits{
		CloneTimeout:     envDuration(EnvCloneTimeout, defaultCloneTimeout),
		MaxRepoSizeBytes: envInt64(EnvMaxRepoSizeBytes, defaultMaxRepoSize),
	}
}

// envInt64 parses a positive int64 from the named environment variable, falling
// back to def (and logging) when unset, empty, or invalid.
func envInt64(name string, def int64) int64 {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return def
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		glog.Warningf("skill catalog: ignoring invalid %s=%q (want a positive integer); using default %d", name, raw, def)
		return def
	}
	glog.Infof("skill catalog: %s=%d overrides default %d", name, v, def)
	return v
}

// envInt is envInt64 for an int-typed limit.
func envInt(name string, def int) int {
	return int(envInt64(name, int64(def)))
}

// envDuration parses a positive Go duration from the named environment variable,
// falling back to def (and logging) when unset, empty, or invalid.
func envDuration(name string, def time.Duration) time.Duration {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return def
	}
	v, err := time.ParseDuration(raw)
	if err != nil || v <= 0 {
		glog.Warningf("skill catalog: ignoring invalid %s=%q (want a positive Go duration like \"90s\"); using default %s", name, raw, def)
		return def
	}
	glog.Infof("skill catalog: %s=%s overrides default %s", name, v, def)
	return v
}
