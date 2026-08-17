package skillcatalog

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/kubeflow/hub/catalog/internal/catalog/basecatalog"
)

const (
	defaultCloneTimeout   = 2 * time.Minute
	defaultMaxRepoSize    = int64(500 << 20) // 500 MiB
	skillManifestFileName = "SKILL.md"
)

// defaultAllowedProtocols are the git transports permitted by default. Notably
// this excludes `ext::` (arbitrary command execution) and `file` (local paths).
var defaultAllowedProtocols = []string{"https", "http", "git"}

var scpLikeRe = regexp.MustCompile(`^[\w.-]+@[\w.-]+:.*`)

// fullSHA matches a full-length git object name (sha-1 or sha-256), which is
// already an immutable commit and resolves to itself.
var fullSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$|^[0-9a-fA-F]{64}$`)

// ResolvedSkill is a skill discovered in a repository at a specific ref, ready to
// be indexed. It pairs the parsed SKILL.md with its canonical identity.
type ResolvedSkill struct {
	Repository      string
	Path            string // skill directory relative to the repo root (slash-separated)
	Version         string // the pinned ref (a tag or commit SHA); branches/HEAD are refused
	ResolvedCommit  string // the commit SHA the ref resolved to
	Skill           *ParsedSkill
	SupportingFiles []string // sibling file paths relative to the repo root
}

// ResolveLimits bound a single clone to protect the pod from abusive sources.
type ResolveLimits struct {
	CloneTimeout     time.Duration // 0 → defaultCloneTimeout
	MaxRepoSizeBytes int64         // 0 → defaultMaxRepoSize
}

// Credentials authenticate to a private repository. Values are passed to git via
// GIT_ASKPASS (never on the command line or in the URL) to keep them out of the
// process table and logs.
type Credentials struct {
	Username string
	Token    string
}

// RepoResolver resolves repository entries into skills by shallow-cloning each
// repo at a ref into a temporary directory (parse-only; the clone is discarded).
type RepoResolver struct {
	limits           ResolveLimits
	allowedProtocols map[string]bool
	baseEnv          []string
}

// Option configures a RepoResolver.
type Option func(*RepoResolver)

// WithAllowedProtocols overrides the permitted git transports. Primarily for
// tests, which need the local `file` transport.
func WithAllowedProtocols(protocols ...string) Option {
	return func(r *RepoResolver) {
		r.allowedProtocols = make(map[string]bool, len(protocols))
		for _, p := range protocols {
			r.allowedProtocols[strings.ToLower(p)] = true
		}
	}
}

// NewRepoResolver creates a resolver with the given limits and options.
func NewRepoResolver(limits ResolveLimits, opts ...Option) *RepoResolver {
	if limits.CloneTimeout <= 0 {
		limits.CloneTimeout = defaultCloneTimeout
	}
	if limits.MaxRepoSizeBytes <= 0 {
		limits.MaxRepoSizeBytes = defaultMaxRepoSize
	}
	r := &RepoResolver{limits: limits, allowedProtocols: map[string]bool{}}
	for _, p := range defaultAllowedProtocols {
		r.allowedProtocols[p] = true
	}
	for _, opt := range opts {
		opt(r)
	}
	r.baseEnv = r.computeBaseEnv()
	return r
}

// computeBaseEnv builds the constant portion of the git environment: prompts
// disabled, transport allowlist applied, and system config suppressed. The
// result is precomputed once so concurrent clones only copy a slice.
func (r *RepoResolver) computeBaseEnv() []string {
	allowed := make([]string, 0, len(r.allowedProtocols))
	for p := range r.allowedProtocols {
		allowed = append(allowed, p)
	}
	return append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ALLOW_PROTOCOL="+strings.Join(allowed, ":"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
	)
}

// Resolve shallow-clones repo.URL at ref, scans its scanPaths for SKILL.md files,
// applies the include/exclude filters, parses each, and returns the discovered
// skills. The temporary clone is always removed, even on error. Repo content is
// never executed.
func (r *RepoResolver) Resolve(ctx context.Context, repo SkillRepository, ref string, creds *Credentials) ([]ResolvedSkill, error) {
	if repo.URL == "" {
		return nil, fmt.Errorf("repository url is required")
	}
	if ref == "" {
		return nil, fmt.Errorf("ref is required")
	}
	// A url or ref beginning with '-' could be misparsed by git as an option
	// (e.g. --upload-pack=...) since both are passed as positional arguments.
	if strings.HasPrefix(repo.URL, "-") {
		return nil, fmt.Errorf("repository url %q must not begin with '-'", repo.URL)
	}
	if strings.HasPrefix(ref, "-") {
		return nil, fmt.Errorf("ref %q must not begin with '-'", ref)
	}
	if err := r.checkProtocol(repo.URL); err != nil {
		return nil, err
	}

	filter, err := basecatalog.NewNameFilter("includedSkills", repo.IncludedSkills, "excludedSkills", repo.ExcludedSkills)
	if err != nil {
		return nil, fmt.Errorf("invalid include/exclude patterns: %w", err)
	}

	workDir, err := os.MkdirTemp("", "skill-resolve-")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	askpassDir, env, err := r.gitEnv(creds)
	if err != nil {
		return nil, err
	}
	if askpassDir != "" {
		defer func() { _ = os.RemoveAll(askpassDir) }()
	}

	ctx, cancel := context.WithTimeout(ctx, r.limits.CloneTimeout)
	defer cancel()

	// Skills must be pinned to an immutable ref for reproducibility: a branch or
	// HEAD (moving pointers) is refused.
	if err := r.checkImmutableRef(ctx, workDir, repo.URL, ref, env); err != nil {
		return nil, err
	}

	if err := r.fetchRef(ctx, workDir, repo.URL, ref, env); err != nil {
		return nil, err
	}
	if err := r.enforceSizeLimit(workDir); err != nil {
		return nil, err
	}

	commit, err := runGitCmd(ctx, workDir, env, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("resolving commit: %w", err)
	}

	// The version is the pinned ref verbatim (a tag or commit SHA).
	skills, err := r.scan(workDir, repo, ref, strings.TrimSpace(commit), filter)
	if err != nil {
		return nil, err
	}
	return skills, nil
}

// RemoteCommit returns the commit that ref currently resolves to on the remote,
// without cloning — a cheap ls-remote. A full commit-SHA ref resolves to itself;
// a tag is peeled to its commit. This lets the loader skip re-cloning a
// (repo, ref) whose commit has not changed since it was last indexed.
func (r *RepoResolver) RemoteCommit(ctx context.Context, repo SkillRepository, ref string, creds *Credentials) (string, error) {
	if repo.URL == "" || ref == "" {
		return "", fmt.Errorf("repository url and ref are required")
	}
	if strings.HasPrefix(repo.URL, "-") || strings.HasPrefix(ref, "-") {
		return "", fmt.Errorf("repository url and ref must not begin with '-'")
	}
	if err := r.checkProtocol(repo.URL); err != nil {
		return "", err
	}
	if fullSHA.MatchString(ref) {
		return strings.ToLower(ref), nil
	}

	askpassDir, env, err := r.gitEnv(creds)
	if err != nil {
		return "", err
	}
	if askpassDir != "" {
		defer func() { _ = os.RemoveAll(askpassDir) }()
	}

	ctx, cancel := context.WithTimeout(ctx, r.limits.CloneTimeout)
	defer cancel()

	// Use an isolated temp dir as the working directory so git does not pick up
	// .git/config from any ancestor repository that happens to contain os.TempDir().
	lsRemoteDir, err := os.MkdirTemp("", "skill-lsremote-")
	if err != nil {
		return "", fmt.Errorf("creating temp dir for ls-remote: %w", err)
	}
	defer func() { _ = os.RemoveAll(lsRemoteDir) }()

	// Pass the exact ref and a trailing-glob: a narrow pattern omits the peeled
	// (`^{}`) line for an annotated tag, while the glob captures it (along with
	// sibling refs, which peelLsRemote filters out by exact name).
	out, err := runGitCmd(ctx, lsRemoteDir, env, "ls-remote", repo.URL, ref, ref+"*")
	if err != nil {
		return "", fmt.Errorf("ls-remote %s at %q: %w", repo.URL, ref, err)
	}
	return peelLsRemote(out, ref), nil
}

// peelLsRemote returns the commit SHA that ref resolves to, from `git ls-remote`
// output that may include sibling refs and a peeled (`^{}`) line for an annotated
// tag. It considers only lines whose ref name matches ref exactly, preferring the
// peeled line so an annotated tag yields its underlying commit rather than the tag
// object (matching what Resolve records). Returns "" when nothing matched.
func peelLsRemote(out, ref string) string {
	var direct, peeled string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sha, name := fields[0], fields[1]
		core := strings.TrimSuffix(name, "^{}")
		if core != ref && !strings.HasSuffix(core, "/"+ref) {
			continue // a sibling ref matched by the glob (e.g. v2.0-rc for v2.0)
		}
		if strings.HasSuffix(name, "^{}") {
			peeled = sha
		} else if direct == "" {
			direct = sha
		}
	}
	if peeled != "" {
		return peeled
	}
	return direct
}

// checkProtocol rejects repository URLs whose git transport is not allowed,
// blocking dangerous transports such as `ext::` up front.
func (r *RepoResolver) checkProtocol(rawURL string) error {
	t := transportOf(rawURL)
	if !r.allowedProtocols[t] {
		return fmt.Errorf("repository url %q uses disallowed transport %q", rawURL, t)
	}
	return nil
}

func transportOf(rawURL string) string {
	if i := strings.Index(rawURL, "://"); i >= 0 {
		return strings.ToLower(rawURL[:i])
	}
	// git remote-helper syntax: transport::address (e.g. ext::sh -c ...).
	if i := strings.Index(rawURL, "::"); i >= 0 {
		return strings.ToLower(rawURL[:i])
	}
	if scpLikeRe.MatchString(rawURL) {
		return "ssh"
	}
	return "file"
}

// fetchRef initializes a repo and shallow-fetches exactly the requested ref,
// which works uniformly for branches, tags, and (server permitting) commit SHAs.
func (r *RepoResolver) fetchRef(ctx context.Context, dir, url, ref string, env []string) error {
	if _, err := runGitCmd(ctx, dir, env, "init", "-q"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	if _, err := runGitCmd(ctx, dir, env, "remote", "add", "origin", url); err != nil {
		return fmt.Errorf("git remote add: %w", err)
	}
	if _, err := runGitCmd(ctx, dir, env, "fetch", "--depth", "1", "--quiet", "origin", ref); err != nil {
		return fmt.Errorf("fetching %s at %q: %w", url, ref, err)
	}
	if _, err := runGitCmd(ctx, dir, env, "checkout", "--quiet", "--detach", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("checking out %q: %w", ref, err)
	}
	return nil
}

// checkImmutableRef returns an error unless ref is an immutable ref (a tag or
// commit SHA). HEAD and branches are moving pointers and are refused. It fails
// closed: if the branch probe itself fails, the ref is rejected rather than
// assumed non-branch, so a branch cannot slip through on a transient error.
func (r *RepoResolver) checkImmutableRef(ctx context.Context, dir, url, ref string, env []string) error {
	if ref == "HEAD" || ref == "" {
		return fmt.Errorf("ref %q is a branch or HEAD; skills must be pinned to a tag or commit SHA", ref)
	}
	out, err := runGitCmd(ctx, dir, env, "ls-remote", "--heads", url, ref)
	if err != nil {
		return fmt.Errorf("unable to verify ref %q is immutable: %w", ref, err)
	}
	if strings.TrimSpace(out) != "" {
		return fmt.Errorf("ref %q is a branch; skills must be pinned to a tag or commit SHA", ref)
	}
	return nil
}

// enforceSizeLimit fails if the checked-out working tree exceeds the size limit.
// This is a post-fetch backstop: it rejects an oversized repository (and the temp
// clone is then removed) rather than bounding the download itself. Bounding the
// download in-flight is deliberately not done via --filter=blob:limit, which many
// git servers silently ignore; runaway fetches are instead bounded by CloneTimeout.
func (r *RepoResolver) enforceSizeLimit(dir string) error {
	var total int64
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if total > r.limits.MaxRepoSizeBytes {
			return fmt.Errorf("repository exceeds the maximum size of %d bytes", r.limits.MaxRepoSizeBytes)
		}
		return nil
	})
	return err
}

// skillFilterName returns the name a skill's include/exclude filter matches
// against: the skill's directory name, or — for a repo-root SKILL.md (relDir ".")
// — its frontmatter name, since a root skill has no directory name to match on.
// relDir is slash-separated. This is the single definition of the filter-match
// rule, shared by the resolver's scan and the source previewer so the two cannot
// drift apart.
func skillFilterName(relDir, frontmatterName string) string {
	if relDir == "." || relDir == "" {
		return frontmatterName
	}
	return path.Base(relDir)
}

// scan walks the repo's scanPaths for SKILL.md files, parses each, and builds the
// resolved skills. Directories named .git and node_modules are skipped.
// All non-SKILL.md files are accumulated in a single pass and used to derive
// supporting files without a second directory walk per skill.
func (r *RepoResolver) scan(root string, repo SkillRepository, version, commit string, filter *basecatalog.NameFilter) ([]ResolvedSkill, error) {
	scanRoots := repo.ScanPaths
	if len(scanRoots) == 0 {
		scanRoots = []string{"."}
	}

	var skills []ResolvedSkill
	seen := make(map[string]bool)
	// dirFiles accumulates all non-SKILL.md files encountered during the primary
	// walk, keyed by their containing directory (absolute), so supporting files
	// can be derived without a second per-skill WalkDir. Using a set (map to
	// struct{}) prevents duplicates when scanPaths overlap.
	dirFiles := make(map[string]map[string]struct{})

	for _, sp := range scanRoots {
		start := filepath.Join(root, filepath.Clean("/"+sp))
		walkErr := filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil // a configured scanPath that doesn't exist is not fatal
				}
				return err
			}
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == "node_modules" {
					return fs.SkipDir
				}
				return nil
			}

			if d.Name() != skillManifestFileName {
				// Accumulate for later supporting-file lookup.
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				dir := filepath.Dir(path)
				if dirFiles[dir] == nil {
					dirFiles[dir] = make(map[string]struct{})
				}
				dirFiles[dir][filepath.ToSlash(rel)] = struct{}{}
				return nil
			}

			// Only read a regular SKILL.md; a symlinked SKILL.md could point outside
			// the clone (e.g. at a host file), so it is skipped rather than followed.
			if !d.Type().IsRegular() {
				glog.Warningf("skipping non-regular SKILL.md at %q in %s", path, repo.URL)
				return nil
			}

			skillDir := filepath.Dir(path)
			relDir, err := filepath.Rel(root, skillDir)
			if err != nil {
				return err
			}
			relDir = filepath.ToSlash(relDir)
			if seen[relDir] {
				return nil
			}
			seen[relDir] = true

			// dirName is the skill's directory name, used for the name-mismatch warning
			// inside ParseSkillMD and for skip log messages. A repo-root SKILL.md
			// (relDir ".") has no directory name. A nested skill can be pre-filtered
			// here from its directory name; a root skill has nothing to match against
			// until its frontmatter name is parsed, so it is filtered below instead.
			// filepath.Base (not path.Base) because the walk's `path` variable shadows
			// the path package here; relDir is slash-separated and the resolver runs on
			// Linux, so the two are equivalent.
			dirName := ""
			if relDir != "." {
				dirName = filepath.Base(relDir)
				if !filter.Allows(skillFilterName(relDir, "")) {
					return nil
				}
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			parsed, perr := ParseSkillMD(content, dirName)
			if perr != nil {
				// Lenient: an unparseable skill is skipped and logged, per the spec
				// ("missing description or unparseable YAML → skip and log"); it is
				// not fatal to the rest of the repo.
				glog.Warningf("skipping skill %q (%s) in %s: %v", dirName, relDir, repo.URL, perr)
				return nil
			}

			// A root-level skill is filtered here, once its frontmatter name — the
			// only name it can be matched on — has been parsed.
			if relDir == "." && !filter.Allows(skillFilterName(relDir, parsed.Name)) {
				return nil
			}

			skills = append(skills, ResolvedSkill{
				Repository:     repo.URL,
				Path:           relDir,
				Version:        version,
				ResolvedCommit: commit,
				Skill:          parsed,
				// SupportingFiles populated below after all scanPaths are walked.
			})
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("scanning %q: %w", sp, walkErr)
		}
	}

	// Assign each accumulated file to its nearest enclosing skill directory, so a
	// nested skill's files are attributed to it rather than also to an ancestor
	// skill. This uses the already-collected dirFiles map, avoiding a second walk.
	skillDirToIdx := make(map[string]int, len(skills))
	for i := range skills {
		abs := filepath.Join(root, filepath.FromSlash(skills[i].Path))
		skillDirToIdx[abs] = i
	}
	for dir, fileSet := range dirFiles {
		if idx, ok := nearestSkillDir(dir, root, skillDirToIdx); ok {
			for f := range fileSet {
				skills[idx].SupportingFiles = append(skills[idx].SupportingFiles, f)
			}
		}
	}

	// The loop above ranges over maps, whose iteration order Go randomizes, so
	// sort each skill's files. Otherwise identical repository content yields a
	// differently-ordered JSON array on every sync, churning the stored property
	// and reshuffling the API response for no reason.
	for i := range skills {
		slices.Sort(skills[i].SupportingFiles)
	}

	return skills, nil
}

// nearestSkillDir walks up from dir toward root, returning the index of the first
// directory that is itself a skill directory — the file's owning skill. A file with
// no skill-directory ancestor at or below root belongs to no skill.
func nearestSkillDir(dir, root string, skillDirToIdx map[string]int) (int, bool) {
	for {
		if idx, ok := skillDirToIdx[dir]; ok {
			return idx, true
		}
		if dir == root {
			return 0, false
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return 0, false
		}
		dir = parent
	}
}

// gitEnv builds the environment for git invocations: prompts disabled, the
// transport allowlist applied, and — if credentials are given — a temporary
// GIT_ASKPASS helper so the token never appears in argv or the URL. The caller
// must remove the returned askpass directory.
func (r *RepoResolver) gitEnv(creds *Credentials) (askpassDir string, env []string, err error) {
	env = append([]string(nil), r.baseEnv...)
	if creds == nil || creds.Token == "" {
		return "", env, nil
	}

	askpassDir, err = os.MkdirTemp("", "skill-askpass-")
	if err != nil {
		return "", nil, fmt.Errorf("creating askpass dir: %w", err)
	}
	script := "#!/bin/sh\ncase \"$1\" in\nUsername*) printf '%s' \"$GIT_SKILL_USER\" ;;\n*) printf '%s' \"$GIT_SKILL_TOKEN\" ;;\nesac\n"
	askpassPath := filepath.Join(askpassDir, "askpass.sh")
	if err := os.WriteFile(askpassPath, []byte(script), 0o700); err != nil {
		_ = os.RemoveAll(askpassDir)
		return "", nil, fmt.Errorf("writing askpass helper: %w", err)
	}
	user := creds.Username
	if user == "" {
		user = "x-access-token"
	}
	env = append(env,
		"GIT_ASKPASS="+askpassPath,
		"GIT_SKILL_USER="+user,
		"GIT_SKILL_TOKEN="+creds.Token,
	)
	return askpassDir, env, nil
}

// runGitCmd runs a git subcommand in dir with the given environment and no shell,
// returning combined output. git content is data only; it is never executed.
func runGitCmd(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
