import { APIOptions, handleRestFailures, restDELETE } from 'mod-arch-core';
import {
  CatalogSourceConfig,
  CatalogSourceConfigList,
  CatalogSourceConfigPayload,
  CatalogSourcePreviewRequest,
  CatalogSourcePreviewResult,
} from '~/app/modelCatalogTypes';
import { createSourceConfigService } from '~/app/shared/catalogSettings/api/createSourceConfigService';

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

export const deleteCatalogSourceCredentials =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, sourceId: string): Promise<void> =>
    handleRestFailures(
      restDELETE(hostPath, `/source_configs/${sourceId}/credentials`, {}, queryParams, {
        ...opts,
        parseJSON: false,
      }),
    ).then(() => undefined);
