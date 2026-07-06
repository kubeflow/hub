import { CatalogSourceStatus } from '~/concepts/modelCatalogSettings/const';

export type CatalogSource = {
  id: string;
  name: string;
  labels: string[];
  enabled?: boolean;
  status?:
    | CatalogSourceStatus.AVAILABLE
    | CatalogSourceStatus.PARTIALLY_AVAILABLE
    | CatalogSourceStatus.ERROR
    | CatalogSourceStatus.DISABLED;
  error?: string;
  assetType?: CatalogAssetType;
};

export type PaginationParams = {
  size: number;
  pageSize: number;
  nextPageToken: string;
};

export type CatalogSourceList = PaginationParams & { items?: CatalogSource[] };

export type CatalogAssetType = 'models' | 'mcp_servers' | 'agents';

export type CatalogSourceListParams = {
  assetType?: CatalogAssetType;
};

export type CatalogLabel = {
  name: string | null;
  displayName?: string;
  description?: string;
};

export type CatalogLabelList = PaginationParams & { items: CatalogLabel[] };

export type CatalogLabelListParams = {
  assetType?: CatalogAssetType;
};

export enum SourceLabel {
  other = 'null',
}
