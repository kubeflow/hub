# KEP-0005: Skill Catalog Plugin

## Summary

Add a `skill` catalog plugin that indexes AI agent skills - directories containing a `SKILL.md` per the [Agent Skills specification](https://agentskills.io/specification) - serving them through the standard catalog API, a catalog UI alongside the model/MCP/agent catalogs, and a plugin-marketplace endpoint so compatible agents install skills with one `marketplace add`.

Skill sources are git repositories, listed in a YAML source file (ConfigMap-mountable) together with their catalog metadata. At sync, the plugin reads the repos directly and rebuilds the ephemeral database index; repos are never stored.

## Motivation

Agent skills have an open specification, growing public repositories, and installation tooling (`npx skills add`, Claude Code marketplaces). Organizations need what Hub already provides for models, MCP servers, and agents: a curated, filterable catalog with provenance and a governed acquisition path. Skills extend the catalog family.

### Goals

- `skill` plugin; API base `/api/skill_catalog/v1alpha1`.
- A `git-skills-plugin` source type - named per the existing convention of naming source types for what they read (`yaml`, `hf`) - configured via a YAML file listing repositories; SKILL.md parsed per the specification at sync.
- Versioned entries: repo entries may list refs (tags, releases, branches, commits); each ref yields its own catalog entries, with the version shown in the UI.
- Custom metadata (trust tier, provider, category, labels) assigned in the source file; per-skill overrides; include/exclude filtering.
- Catalog UI: gallery, detail with rendered SKILL.md, filters, install instructions; an admin settings page that manages sources (add/modify/delete source files and repo entries, with include/exclude preview before save) alongside sync status and manual sync.
- `marketplace.json` endpoint, with optional URL rewriting for deployments fronting repos with an internal git mirror.

### Non-Goals

- Skill delivery into agent pods - consumes this API; needs no plugin changes.
- Mirroring / air-gapped distribution - deployment packaging; reading a mirror instead of GitHub is a config change.
- Security scanning of skill content - deferred.
- Skill authoring/editing - read-only, one-way, source-authoritative.
- Usage analytics - deferred.
- Cross-plugin unified search - Hub-wide concern.

## Proposal

### 1. New `skill` plugin

A standard Hub catalog plugin, following the existing plugin architecture.

### 2. Skill sources

Registered like any other catalog source, as `type: git-skills-plugin`; the source's configured file lists git repositories and their catalog metadata:

```yaml
source: Community Skills
trustTier: communityContributed
repositories:
  - url: https://github.com/example/skills.git
    refs: [main, v1.0]            # optional; tags/releases/branches/commits -
                                  #   each ref yields its own catalog entries.
                                  #   Also: scanPaths, authSecretName,
                                  #   canonicalUrl (when url is a mirror)
    provider: Example Org
    category: DevOps
    labels: [community]
    includedSkills: ["*"]
    excludedSkills: ["*-draft"]
    skillOverrides: [{name: deploy, category: SRE}]
```

At sync, the plugin reads each listed repository at each listed ref directly (a temporary shallow clone used only for parsing, then discarded), finds and parses `SKILL.md` files per the specification, and rebuilds the index. A skill's `version` is the ref, with the resolved commit SHA recorded. Removed skills, refs, repos, or sources are cleaned up. Capability to pull from private git repositories with safe credential configuration will be supported.

ConfigMap-backed source file picked up by hot-reload; immutably mounted files render read-only in the UI.

### 3. Custom metadata

Trust tier, provider, category, and labels come from the source file and are applied to its skills at sync - never read from repo content. `trustTier` is simply a label - one of `platformProvided`, `partnerVerified`, `organizationApproved`, `communityContributed` (downstream products may map display names) - shown as a badge and filterable, with no ordering or special semantics.

### 4. Skill identity

Permanent identity is `(repository, path)` - the upstream repo URL plus the skill's directory - with `version` distinguishing entries. The index is an ephemeral cache; internal IDs are not durable. This keeps external references stable across rebuilds and across deployments reading the same skill through different URLs.

### 5. API surface

```
GET  /skills    # name, q, source, sourceLabel, filterQuery, paging
GET  /skills/{id}
GET  /skills/filter_options
GET  /claude/marketplace.json
POST .../sources/preview  # existing endpoint, assetType: skills
```

The `Skill` resource carries the SKILL.md fields (including the body as `readme`), the identity (`repository`, `path`, `version`, `resolvedCommit`), and the catalog metadata (tier, provider, category, labels, supporting-file paths). Supporting-file contents are not stored - they are linked to the repo.

### 6. Marketplace endpoint

`GET .../marketplace.json` conforms to the [Claude Code plugin marketplace schema](https://code.claude.com/docs/en/plugin-marketplaces):

```
/plugin marketplace add https://hub.example.com/api/skill_catalog/v1alpha1
/plugin install deploy@example-skill-catalog
```

Source URLs default to the repos' own URLs; optional `skill_content_mirror: {internalBaseUrl, externalBaseUrl}` rewrites them for deployments fronting repos with an internal mirror (external base by default, internal for `?audience=cluster`). When a skill has entries at multiple refs, the repo's first-listed ref is exposed. User-docs note: `npx skills add` consumes git repos directly, never this endpoint.

### 7. UI

Follows the MCP catalog UI on the shared catalog components, with a BFF proxy handler. Gallery cards (name, provider, tier badge, version label, description, category/license chips, labels); detail page (rendered readme, metadata sidebar, supporting files linked to the repo, install instructions with copy-paste marketplace/npx/manual commands); filters from `filter_options`; read-only admin settings page (source status, include/exclude preview, manual sync).

## Risks and Mitigations

- **Hub fetches remote content at sync** → temporary shallow clones only (isolated, cleaned up), timeouts and size limits, parse-only (repo content is never executed), Secret-based auth for private repos.
- **Malicious or oversized configured repos** → source management requires configuration access, limits, degraded-source status surfacing, future scan hook.

## Alternatives

- **Pre-parse skill metadata into the source file.** Rejected: it duplicates the parser outside Hub and fixes metadata at file-generation time; with repo locations in the file, one parser refreshes from the repos at every sync.
- **Store supporting-file contents in the DB.** Rejected: content is always reachable in the source repo.
- **Semantic versioning.** Rejected: not in the SKILL.md spec; refs (tags/releases/commits) chosen in the source file are explicit and honest.
