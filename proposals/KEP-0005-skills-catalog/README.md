# KEP-0005: Skill Catalog Plugin

## Summary

Add a `skill` catalog plugin that indexes AI agent skills - directories containing a `SKILL.md` per the [Agent Skills specification](https://agentskills.io/specification) - serving them through the standard catalog API, a catalog UI alongside the model/MCP/agent catalogs, and a plugin-marketplace endpoint so compatible agents install skills with one `marketplace add`.

Skill sources are git repositories, listed in a YAML source file (which can be mounted from a ConfigMap) along with their catalog metadata. Each time the catalog syncs, the plugin reads the repositories directly and rebuilds its index from scratch. The repositories themselves are never stored.

## Motivation

Agent skills have an open specification, growing public repositories, and installation tooling (`npx skills add`, Claude Code marketplaces). Organizations need what Hub already provides for models, MCP servers, and agents: a curated, filterable catalog with provenance and a governed acquisition path. Skills extend the catalog family.

### Goals

- `skill` plugin; API base `/api/skill_catalog/v1alpha1`.
- A `git-skills-plugin` source type - a dedicated type for git-hosted skills, alongside the existing `yaml` and `hf` types - listing repositories whose SKILL.md files are parsed per the specification at sync time.
- Versioned entries: a repository can list multiple refs (tags, releases, branches, or commits). Each ref becomes its own set of catalog entries, with the version shown in the UI. A branch is surfaced as `latest` (not the raw branch name), and every entry pins to the commit it resolved to, so installs are reproducible.
- Custom metadata (trust tier, provider, category, labels) assigned in the source file; per-skill overrides; include/exclude filtering.
- Catalog UI: gallery, detail with rendered SKILL.md, filters, install instructions; an admin settings page that reuses the existing model/MCP catalog-settings mechanism to manage sources (add/edit/delete user-managed sources with include/exclude preview before save; shipped defaults read-only) alongside sync status and manual sync.
- `marketplace.json` endpoint, with optional URL rewriting for deployments fronting repos with an internal git mirror.

### Non-Goals

- Delivering skills into agent pods. That consumes this API and needs no changes to the plugin.
- Mirroring or air-gapped distribution. That is a deployment-packaging concern; reading from a mirror instead of GitHub is just a config change.
- Security scanning of skill content. Deferred.
- Authoring or editing skills. The catalog is read-only and one-way; the source repositories are authoritative.
- Usage analytics. Deferred.
- Unified search across plugins. That is a Hub-wide concern.

## Proposal

### 1. New `skill` plugin

A standard Hub catalog plugin, following the existing plugin architecture.

### 2. Skill sources

Registered like any other catalog source, as `type: git-skills-plugin`. A source lists git repositories and their catalog metadata. The repository list can live in the source config **inline** (used by UI-added sources - written to the user-managed ConfigMap, editable like an `hf` source) or **via a file** (`yamlCatalogPath`, used by shipped defaults that bundle many repos). The loader accepts both.

```yaml
# inline form (UI-added source in the user-managed ConfigMap)
- name: Community Skills
  id: community-skills
  type: git-skills-plugin
  enabled: true
  trustTier: communityContributed
  repositories:
    - url: https://github.com/example/skills.git
      refs: [main, v1.0]          # optional; tags/releases/branches/commits -
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

Each time the catalog syncs, the plugin reads each listed repository at each listed ref (making a temporary, lightweight copy used only for parsing, then throwing it away), finds and parses `SKILL.md` files per the specification, and rebuilds the index. A skill's `version` is the ref (a branch is surfaced as `latest`, not the branch name), and the exact commit it resolved to is recorded as `resolvedCommit`. Skills, refs, repositories, or sources that have been removed are cleaned up. Pulling from private git repositories with securely configured credentials will be supported.

Source management reuses the same mechanism as the model and MCP catalogs: read-only defaults (shipped in a ConfigMap) merged with a user-managed ConfigMap that the settings UI writes via the BFF, with per-source auth stored as Secrets. Adding a skills git source through the UI works like adding a HuggingFace source for models. Editing the ConfigMap directly (kubectl/GitOps) works too; the file-watcher picks up either path.

### 3. Custom metadata

Trust tier, provider, category, and labels come from the source file and are applied to its skills at sync time - they are never read from the repository's content. `trustTier` is simply a label - one of `platformProvided`, `partnerVerified`, `organizationApproved`, or `communityContributed` (downstream products may show their own display names). It appears as a badge and can be filtered on, but carries no ordering or special meaning.

### 4. Skill identity

A skill's permanent identity is `(repository, path)` - the upstream repository URL plus the skill's directory - with `version` telling entries apart. The index is a temporary cache, so its internal IDs are not stable. Using the repository and path as the identity keeps external references stable across index rebuilds, and even across deployments that read the same skill through different URLs.

### 5. API surface

```
GET  /skills    # name, q, source, sourceLabel, filterQuery, paging
GET  /skills/{id}
GET  /skills/filter_options
GET  /claude/marketplace.json
POST .../sources/preview  # existing endpoint, assetType: skills
```

The `Skill` resource carries the SKILL.md fields (including the body as `readme`), the identity (`repository`, `path`, `version`, `resolvedCommit`), and the catalog metadata (tier, provider, category, labels, and the paths of supporting files). The contents of supporting files are not stored - the catalog links to them in the repository instead.

### 6. Marketplace endpoint

`GET .../claude/marketplace.json` serves the marketplace manifest in the [Claude Code plugin marketplace schema](https://code.claude.com/docs/en/plugin-marketplaces). The `claude/` segment is a deliberate per-consumer-format namespace: this path is Claude Code's schema, and another agent's format would live at a sibling path under the same API version - the manifest is a rendered view of the one skill index, not a separate data source.

```
/plugin marketplace add https://hub.example.com/api/skill_catalog/v1alpha1
/plugin install deploy@example-skill-catalog
```

Every listed ref of a skill is exposed as its own plugin entry carrying its version (a branch appears as `latest`), so all versions are discoverable, and each entry's git source is pinned to the commit it resolved to (`resolvedCommit`) so `/plugin install` is reproducible even for `latest`. Source URLs default to the repositories' own URLs. An optional `skill_content_mirror: {internalBaseUrl, externalBaseUrl}` setting rewrites them for deployments that put an internal mirror in front of the repositories (using the external base by default, and the internal base for `?audience=cluster`). A note for user docs: `npx skills add` reads git repositories directly, never this endpoint.

### 7. UI

Follows the MCP catalog UI, built on the shared catalog components, with a BFF (backend-for-frontend) proxy handler. It includes gallery cards (name, provider, tier badge, version label, description, category/license chips, labels); a detail page (rendered readme, metadata sidebar, supporting files linked to the repository, and install instructions with copy-paste marketplace/npx/manual commands); filters driven by `filter_options`; and an admin settings page reusing the existing model/MCP catalog-settings pattern (list sources with status, add/edit/delete user-managed sources with a preview before saving, manual sync; shipped defaults read-only).

## Risks and Mitigations

Source URLs are treated as trusted: managing sources requires admin access (settings page) or configuration access (ConfigMap/GitOps edits), the same trust boundary as the other catalogs. The operational risk is resource exhaustion, not untrusted input.

- **Hub fetches remote content at sync** → only temporary, lightweight copies are made (kept isolated and cleaned up afterward); content is only parsed, never executed; private repositories authenticate through Kubernetes Secrets. Concrete limits bound the blast radius: a per-clone timeout, a maximum repository size, a maximum number of refs per repository, and a global cap on in-flight clones so a large source set (many repositories × many refs) cannot saturate the pod.
- **Sync storms from rapid config changes** → hot-reload is debounced, so a burst of ConfigMap edits (UI or GitOps) coalesces into a single sync rather than a fetch storm.
- **A source in trouble** → shown as a degraded source status; a content-scanning hook is planned for the future.

## Alternatives

- **Pre-parse skill metadata into the source file.** Rejected: it would duplicate the parser outside Hub and freeze the metadata at the moment the file is generated. By listing repository locations in the file instead, a single parser refreshes the metadata from the repositories on every sync.
- **Store supporting-file contents in the database.** Rejected: the content is always available in the source repository.
- **Semantic versioning.** Rejected: it isn't part of the SKILL.md spec. The refs (tags, releases, commits) chosen in the source file are explicit and honest.
