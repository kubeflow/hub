import { APIOptions, handleRestFailures, isModArchResponse, restGET } from 'mod-arch-core';
import {
  CatalogLabelList,
  CatalogSourceList,
  CatalogSourceListParams,
  CatalogLabelListParams,
} from '~/app/shared/types/catalogTypes';
import {
  CatalogArtifactList,
  CatalogFilterOptionsList,
  CatalogModel,
  CatalogModelList,
  CatalogPerformanceArtifactList,
  ModelCatalogFilterStates,
  PerformanceArtifactsParams,
} from '~/app/modelCatalogTypes';
import { filtersToFilterQuery } from '~/app/pages/modelCatalog/utils/modelCatalogUtils';

export const getCatalogModelsBySource =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (
    opts: APIOptions,
    sourceId?: string,
    sourceLabel?: string,
    paginationParams?: {
      pageSize?: string;
      nextPageToken?: string;
      orderBy?: string;
      sortOrder?: string;
    },
    searchKeyword?: string,
    filterData?: ModelCatalogFilterStates,
    filterOptions?: CatalogFilterOptionsList | null,
    filterQuery?: string,
    performanceParams?: {
      targetRPS?: number;
      latencyProperty?: string;
      orderBy?: string;
    },
  ): Promise<CatalogModelList> => {
    const computedFilterQuery =
      filterQuery ??
      (filterData && filterOptions
        ? filtersToFilterQuery(filterData, filterOptions, 'models', !!performanceParams)
        : '');

    const allParams = {
      source: sourceId,
      sourceLabel,
      ...paginationParams,
      ...(searchKeyword && { q: searchKeyword }),
      ...queryParams,
      ...(computedFilterQuery && { filterQuery: computedFilterQuery }),
      ...(performanceParams?.targetRPS !== undefined && { targetRPS: performanceParams.targetRPS }),
      ...(performanceParams?.latencyProperty && {
        latencyProperty: performanceParams.latencyProperty,
      }),
      ...(performanceParams?.orderBy && {
        orderBy: performanceParams.orderBy,
      }),
    };
    return handleRestFailures(restGET(hostPath, '/models', allParams, opts)).then((response) => {
      if (isModArchResponse<CatalogModelList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });
  };

export const getCatalogFilterOptionList =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions): Promise<CatalogFilterOptionsList> =>
    handleRestFailures(restGET(hostPath, '/models/filter_options', queryParams, opts)).then(
      (response) => {
        if (isModArchResponse<CatalogFilterOptionsList>(response)) {
          return response.data;
        }
        throw new Error('Invalid response format');
      },
    );

export const getListSources =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, listParams?: CatalogSourceListParams): Promise<CatalogSourceList> =>
    handleRestFailures(restGET(hostPath, '/sources', { ...queryParams, ...listParams }, opts)).then(
      (response) => {
        if (isModArchResponse<CatalogSourceList>(response)) {
          return response.data;
        }
        throw new Error('Invalid response format');
      },
    );

export const getCatalogModel =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, sourceId: string, modelName: string): Promise<CatalogModel> =>
    handleRestFailures(
      restGET(hostPath, `/sources/${sourceId}/models/${modelName}`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<CatalogModel>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getListCatalogModelArtifacts =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, sourceId: string, modelName: string): Promise<CatalogArtifactList> =>
    handleRestFailures(
      restGET(hostPath, `/sources/${sourceId}/artifacts/${modelName}`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<CatalogArtifactList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getPerformanceArtifacts =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (
    opts: APIOptions,
    sourceId: string,
    modelName: string,
    params?: PerformanceArtifactsParams,
    filterData?: ModelCatalogFilterStates,
    filterOptions?: CatalogFilterOptionsList | null,
  ): Promise<CatalogPerformanceArtifactList> => {
    const allParams: Record<string, unknown> = {
      ...queryParams,
      ...(params?.targetRPS !== undefined && { targetRPS: params.targetRPS }),
      ...(params?.rpsProperty && { rpsProperty: params.rpsProperty }),
      ...(params?.latencyProperty && { latencyProperty: params.latencyProperty }),
      ...(params?.hardwareCountProperty && { hardwareCountProperty: params.hardwareCountProperty }),
      ...(params?.hardwareTypeProperty && { hardwareTypeProperty: params.hardwareTypeProperty }),
      ...(params?.pageSize && { pageSize: params.pageSize }),
      ...(params?.orderBy && { orderBy: params.orderBy }),
      ...(params?.sortOrder && { sortOrder: params.sortOrder }),
      ...(params?.nextPageToken && { nextPageToken: params.nextPageToken }),
      ...(filterData &&
        filterOptions && {
          filterQuery: filtersToFilterQuery(filterData, filterOptions, 'artifacts'),
        }),
    };
    return handleRestFailures(
      restGET(hostPath, `/sources/${sourceId}/performance_artifacts/${modelName}`, allParams, opts),
    ).then((response) => {
      if (isModArchResponse<CatalogPerformanceArtifactList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });
  };

export const getCatalogLabels =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, listParams?: CatalogLabelListParams): Promise<CatalogLabelList> =>
    handleRestFailures(restGET(hostPath, '/labels', { ...queryParams, ...listParams }, opts)).then(
      (response) => {
        if (isModArchResponse<CatalogLabelList>(response)) {
          return response.data;
        }
        throw new Error('Invalid response format');
      },
    );
