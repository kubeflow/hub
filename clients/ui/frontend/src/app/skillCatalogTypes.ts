import { APIOptions } from 'mod-arch-core';
import type {
  CatalogFilterOptionsList,
  PreviewCatalogSourceQueryParams,
} from '~/app/modelCatalogTypes';
import { PaginationParams } from '~/app/shared/types/catalogTypes';

export type Skill = {
  id: string;
  name: string;
  sourceId?: string;
  displayName?: string;
  description?: string;
  readme?: string;
  tags?: string[];
  repositoryUrl?: string;
  labels?: string[];
  provider?: string;
  category?: string;
  trustTier?: string;
  repository?: string;
  path?: string;
  version?: string;
  resolvedCommit?: string;
  license?: string;
  /**
   * SKILL.md body length. Surfaced as a quality signal: an agent loads the whole body
   * into its context, so an over-long skill costs context budget. Not filterable.
   */
  bodyLineCount?: number;
  author?: string;
  compatibility?: string;
  allowedTools?: string[];
  supportingFiles?: string[];
};

export type SkillList = PaginationParams & { items?: Skill[] };

export type SkillListParams = {
  sourceLabel?: string;
  pageSize?: number | string;
  nextPageToken?: string;
  q?: string;
  filterQuery?: string;
};

export enum SkillCatalogSourceType {
  GIT_SKILLS = 'git-skills-plugin',
}

/**
 * A per-skill exception to a repository's custom metadata. The repository's own
 * category/labels apply to every skill it yields; an override replaces them for the
 * one skill it names.
 *
 * There is no form control for these — the settings UI edits repository-level
 * metadata only. They are carried through so that saving a source authored by
 * GitOps does not delete them, since the BFF rebuilds the repository entry from
 * whatever the UI sends.
 */
export type SkillOverride = {
  name: string;
  category?: string;
  labels?: string[];
};

export type SkillRepository = {
  url: string;
  canonicalUrl?: string;
  refs?: string[];
  scanPaths?: string[];
  includedSkills?: string[];
  excludedSkills?: string[];
  /** Read-only in the UI; round-tripped verbatim so a UI save preserves them. */
  skillOverrides?: SkillOverride[];
  /**
   * Write-only. The git token entered in the admin form. The BFF stores it in the
   * shared git-credentials Secret and never echoes it back, so this is always
   * undefined on reads.
   */
  authToken?: string;
  /**
   * Read-only. Key in the git-credentials Secret mounted into the catalog pod,
   * set by the BFF when a token is supplied. Round-tripped on edit so that saving
   * a source without re-entering the token keeps its existing credential.
   */
  credentialRef?: string;
  provider?: string;
  category?: string;
  trustTier?: string;
  /**
   * Stamped onto every skill this repository yields, so they appear on skill cards
   * and in the catalog filter options. Distinct from
   * SkillCatalogSourceConfig.labels, which only group the source itself.
   */
  labels?: string[];
};

export type SkillCatalogSourceConfig = {
  id: string;
  name: string;
  type: SkillCatalogSourceType;
  enabled?: boolean;
  isDefault?: boolean;
  repositories?: SkillRepository[];
  trustTier?: string;
  provider?: string;
  category?: string;
  labels?: string[];
};

export type SkillCatalogSourceConfigPayload =
  | SkillCatalogSourceConfig
  | Pick<SkillCatalogSourceConfig, 'enabled' | 'repositories'>;

export type SkillCatalogSourceConfigList = {
  catalogs: SkillCatalogSourceConfig[];
};

export type GetSkillCatalogSourceConfigs = (
  opts: APIOptions,
) => Promise<SkillCatalogSourceConfigList>;
export type CreateSkillCatalogSourceConfig = (
  opts: APIOptions,
  data: SkillCatalogSourceConfigPayload,
) => Promise<SkillCatalogSourceConfig>;
export type GetSkillCatalogSourceConfig = (
  opts: APIOptions,
  sourceId: string,
) => Promise<SkillCatalogSourceConfig>;
export type UpdateSkillCatalogSourceConfig = (
  opts: APIOptions,
  sourceId: string,
  data: Partial<SkillCatalogSourceConfigPayload>,
) => Promise<SkillCatalogSourceConfig>;
export type DeleteSkillCatalogSourceConfig = (opts: APIOptions, sourceId: string) => Promise<void>;

export type SkillCatalogSourcePreviewRequest = {
  type: SkillCatalogSourceType;
  properties?: Record<string, unknown>;
};

export type SkillCatalogSourcePreviewAsset = {
  name: string;
  included: boolean;
};

export type SkillCatalogSourcePreviewSummary = {
  totalAssets: number;
  includedAssets: number;
  excludedAssets: number;
};

export type SkillCatalogSourcePreviewResult = {
  items: SkillCatalogSourcePreviewAsset[];
  summary: SkillCatalogSourcePreviewSummary;
  nextPageToken: string;
  pageSize: number;
  size: number;
};

export type PreviewSkillCatalogSource = (
  opts: APIOptions,
  data: SkillCatalogSourcePreviewRequest,
  queryParams?: PreviewCatalogSourceQueryParams,
) => Promise<SkillCatalogSourcePreviewResult>;

export type SkillCatalogSettingsAPIs = {
  getSkillCatalogSourceConfigs: GetSkillCatalogSourceConfigs;
  createSkillCatalogSourceConfig: CreateSkillCatalogSourceConfig;
  getSkillCatalogSourceConfig: GetSkillCatalogSourceConfig;
  updateSkillCatalogSourceConfig: UpdateSkillCatalogSourceConfig;
  deleteSkillCatalogSourceConfig: DeleteSkillCatalogSourceConfig;
  previewSkillCatalogSource: PreviewSkillCatalogSource;
};

export type GetSkillList = (opts: APIOptions, listParams?: SkillListParams) => Promise<SkillList>;

export type GetSkillFilterOptionList = (opts: APIOptions) => Promise<CatalogFilterOptionsList>;

export type GetSkill = (opts: APIOptions, skillId: string) => Promise<Skill>;

export type SkillMarketplacePluginSource = {
  url: string;
  path: string;
  ref?: string;
  sha?: string;
};

export type SkillMarketplacePlugin = {
  name: string;
  source: SkillMarketplacePluginSource;
};

export type SkillMarketplace = {
  name: string;
  plugins?: SkillMarketplacePlugin[];
};

/**
 * The marketplace manifest plus the URL to hand to `/plugin marketplace add`.
 *
 * The URL is not this BFF endpoint's: that route is namespace-scoped and
 * authenticated, which an agent cannot satisfy. By default it is the catalog
 * service's in-cluster address, reachable by agents running in the cluster;
 * `external` is true only when an operator has configured a routable URL.
 */
export type SkillMarketplaceResult = {
  marketplace: SkillMarketplace;
  marketplaceUrl: string;
  external: boolean;
};

export type GetSkillMarketplace = (opts: APIOptions) => Promise<SkillMarketplaceResult>;

export type SkillCatalogSpecificAPIs = {
  getSkillList: GetSkillList;
  getSkillFilterOptionList: GetSkillFilterOptionList;
  getSkill: GetSkill;
  getSkillMarketplace: GetSkillMarketplace;
};
