import {
  APIOptions,
  assembleModArchBody,
  handleRestFailures,
  isModArchResponse,
  restCREATE,
  restDELETE,
  restGET,
  restPATCH,
} from 'mod-arch-core';
import { PreviewCatalogSourceQueryParams } from '~/app/modelCatalogTypes';

export type SourceConfigServiceOptions = {
  /** Extra query params applied last on preview (e.g. MCP assetType). */
  previewExtraQueryParams?: Record<string, unknown>;
};

export type SourceConfigService<TConfig, TConfigList, TPayload, TPreviewRequest, TPreviewResult> = {
  getSourceConfigs: (
    hostPath: string,
    queryParams?: Record<string, unknown>,
  ) => (opts: APIOptions) => Promise<TConfigList>;
  createSourceConfig: (
    hostPath: string,
    queryParams?: Record<string, unknown>,
  ) => (opts: APIOptions, data: TPayload) => Promise<TConfig>;
  getSourceConfig: (
    hostPath: string,
    queryParams?: Record<string, unknown>,
  ) => (opts: APIOptions, sourceId: string) => Promise<TConfig>;
  updateSourceConfig: (
    hostPath: string,
    queryParams?: Record<string, unknown>,
  ) => (opts: APIOptions, sourceId: string, data: Partial<TPayload>) => Promise<TConfig>;
  deleteSourceConfig: (
    hostPath: string,
    queryParams?: Record<string, unknown>,
  ) => (opts: APIOptions, sourceId: string) => Promise<void>;
  previewSource: (
    hostPath: string,
    queryParams?: Record<string, unknown>,
  ) => (
    opts: APIOptions,
    data: TPreviewRequest,
    additionalQueryParams?: PreviewCatalogSourceQueryParams,
  ) => Promise<TPreviewResult>;
};

export const createSourceConfigService = <
  TConfig,
  TConfigList,
  TPayload,
  TPreviewRequest,
  TPreviewResult,
>(
  options: SourceConfigServiceOptions = {},
): SourceConfigService<TConfig, TConfigList, TPayload, TPreviewRequest, TPreviewResult> => {
  const { previewExtraQueryParams } = options;

  return {
    getSourceConfigs:
      (hostPath, queryParams = {}) =>
      (opts) =>
        handleRestFailures(restGET(hostPath, '/source_configs', queryParams, opts)).then(
          (response) => {
            if (isModArchResponse<TConfigList>(response)) {
              return response.data;
            }
            throw new Error('Invalid response format');
          },
        ),

    createSourceConfig:
      (hostPath, queryParams = {}) =>
      (opts, data) =>
        handleRestFailures(
          restCREATE(hostPath, '/source_configs', assembleModArchBody(data), queryParams, opts),
        ).then((response) => {
          if (isModArchResponse<TConfig>(response)) {
            return response.data;
          }
          throw new Error('Invalid response format');
        }),

    getSourceConfig:
      (hostPath, queryParams = {}) =>
      (opts, sourceId) =>
        handleRestFailures(
          restGET(hostPath, `/source_configs/${sourceId}`, queryParams, opts),
        ).then((response) => {
          if (isModArchResponse<TConfig>(response)) {
            return response.data;
          }
          throw new Error('Invalid response format');
        }),

    updateSourceConfig:
      (hostPath, queryParams = {}) =>
      (opts, sourceId, data) =>
        handleRestFailures(
          restPATCH(
            hostPath,
            `/source_configs/${sourceId}`,
            assembleModArchBody(data),
            queryParams,
            opts,
          ),
        ).then((response) => {
          if (isModArchResponse<TConfig>(response)) {
            return response.data;
          }
          throw new Error('Invalid response format');
        }),

    deleteSourceConfig:
      (hostPath, queryParams = {}) =>
      (opts, sourceId) =>
        handleRestFailures(
          restDELETE(hostPath, `/source_configs/${sourceId}`, {}, queryParams, opts),
        ),

    previewSource:
      (hostPath, queryParams = {}) =>
      (opts, data, additionalQueryParams) =>
        handleRestFailures(
          restCREATE(
            hostPath,
            '/source_preview',
            assembleModArchBody(data),
            {
              ...queryParams,
              ...additionalQueryParams,
              ...previewExtraQueryParams,
            },
            opts,
          ),
        ).then((response) => {
          if (isModArchResponse<TPreviewResult>(response)) {
            return response.data;
          }
          throw new Error('Invalid response format');
        }),
  };
};
