import { APIOptions, handleRestFailures, isModArchResponse, restGET } from 'mod-arch-core';
import {
  Skill,
  SkillList,
  SkillListParams,
  SkillMarketplace,
  SkillMarketplaceResult,
} from '~/app/skillCatalogTypes';
import { CatalogFilterOptionsList } from '~/app/modelCatalogTypes';

export const getSkillList =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, listParams?: SkillListParams): Promise<SkillList> => {
    const pageSize = listParams?.pageSize !== undefined ? String(listParams.pageSize) : undefined;
    const allParams = {
      ...queryParams,
      ...(listParams?.sourceLabel !== undefined && { sourceLabel: listParams.sourceLabel }),
      ...(listParams?.nextPageToken !== undefined && { nextPageToken: listParams.nextPageToken }),
      ...(pageSize !== undefined && { pageSize }),
      ...(listParams?.filterQuery !== undefined &&
        listParams.filterQuery !== '' && { filterQuery: listParams.filterQuery }),
      ...(listParams?.q !== undefined && listParams.q !== '' && { q: listParams.q }),
    };
    return handleRestFailures(restGET(hostPath, '/skills', allParams, opts)).then((response) => {
      if (isModArchResponse<SkillList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });
  };

export const getSkillFilterOptionList =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions): Promise<CatalogFilterOptionsList> =>
    handleRestFailures(restGET(hostPath, '/skills_filter_options', queryParams, opts)).then(
      (response) => {
        if (isModArchResponse<CatalogFilterOptionsList>(response)) {
          return response.data;
        }
        throw new Error('Invalid response format');
      },
    );

export const getSkill =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, skillId: string): Promise<Skill> =>
    handleRestFailures(restGET(hostPath, `/skills/${skillId}`, queryParams, opts)).then(
      (response) => {
        if (isModArchResponse<Skill>(response)) {
          return response.data;
        }
        throw new Error('Invalid response format');
      },
    );

export const getSkillMarketplace =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions): Promise<SkillMarketplaceResult> =>
    handleRestFailures(restGET(hostPath, '/claude/marketplace.json', queryParams, opts)).then(
      (response) => {
        if (isModArchResponse<SkillMarketplace>(response)) {
          const metadata = response.metadata ?? {};
          return {
            marketplace: response.data,
            marketplaceUrl:
              typeof metadata.marketplaceUrl === 'string' ? metadata.marketplaceUrl : '',
            external: metadata.external === true,
          };
        }
        throw new Error('Invalid response format');
      },
    );
