import {
  SkillCatalogSourceConfig,
  SkillCatalogSourceConfigList,
  SkillCatalogSourceConfigPayload,
  SkillCatalogSourcePreviewRequest,
  SkillCatalogSourcePreviewResult,
} from '~/app/skillCatalogTypes';
import { createSourceConfigService } from '~/app/shared/catalogSettings/api/createSourceConfigService';

const service = createSourceConfigService<
  SkillCatalogSourceConfig,
  SkillCatalogSourceConfigList,
  SkillCatalogSourceConfigPayload,
  SkillCatalogSourcePreviewRequest,
  SkillCatalogSourcePreviewResult
>({
  previewExtraQueryParams: { assetType: 'skills' },
});

export const getSkillCatalogSourceConfigs = service.getSourceConfigs;
export const createSkillCatalogSourceConfig = service.createSourceConfig;
export const getSkillCatalogSourceConfig = service.getSourceConfig;
export const updateSkillCatalogSourceConfig = service.updateSourceConfig;
export const deleteSkillCatalogSourceConfig = service.deleteSourceConfig;
export const previewSkillCatalogSource = service.previewSource;
