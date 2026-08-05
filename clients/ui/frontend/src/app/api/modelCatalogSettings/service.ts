import {
  CatalogSourceConfig,
  CatalogSourceConfigList,
  CatalogSourceConfigPayload,
  CatalogSourcePreviewRequest,
  CatalogSourcePreviewResult,
} from '~/app/modelCatalogTypes';
import { createSourceConfigService } from '~/app/shared/catalogSettings';

const service = createSourceConfigService<
  CatalogSourceConfig,
  CatalogSourceConfigList,
  CatalogSourceConfigPayload,
  CatalogSourcePreviewRequest,
  CatalogSourcePreviewResult
>();

export const getCatalogSourceConfigs = service.getSourceConfigs;
export const createCatalogSourceConfig = service.createSourceConfig;
export const getCatalogSourceConfig = service.getSourceConfig;
export const updateCatalogSourceConfig = service.updateSourceConfig;
export const deleteCatalogSourceConfig = service.deleteSourceConfig;
export const previewCatalogSource = service.previewSource;
