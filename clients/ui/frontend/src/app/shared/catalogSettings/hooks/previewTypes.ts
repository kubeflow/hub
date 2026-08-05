export enum CatalogSettingsPreviewTab {
  INCLUDED = 'included',
  EXCLUDED = 'excluded',
}

export const DEFAULT_PREVIEW_PAGE_SIZE = 20;

export type CatalogSettingsPreviewTabState<TItem> = {
  items: TItem[];
  nextPageToken?: string;
  hasMore: boolean;
};

export type CatalogSettingsPreviewResult<TItem, TSummary> = {
  items: TItem[];
  summary: TSummary;
  nextPageToken?: string;
};

export const getTargetPreviewTab = <TTab extends string>(
  isFreshPreview: boolean,
  switchToTab: TTab | undefined,
  activeTab: TTab,
  includedTab: TTab,
): TTab => {
  if (isFreshPreview) {
    return includedTab;
  }
  return switchToTab ?? activeTab;
};

export const createInitialPreviewTabState = <TItem>(): CatalogSettingsPreviewTabState<TItem> => ({
  items: [],
  nextPageToken: undefined,
  hasMore: false,
});
