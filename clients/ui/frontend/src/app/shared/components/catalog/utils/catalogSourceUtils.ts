import type {
  CatalogLabelList,
  CatalogSourceList,
  CatalogLabel,
} from '~/app/shared/types/catalogTypes';
import { SourceLabel } from '~/app/shared/types/catalogTypes';
import { CatalogSourceStatus } from '~/concepts/modelCatalogSettings/const';

/**
 * Checks whether a catalog source status indicates that items are available.
 * Sources with 'available' or 'partially-available' status have discoverable items.
 */
export const isSourceStatusActive = (status: string | undefined): boolean =>
  status === CatalogSourceStatus.AVAILABLE || status === CatalogSourceStatus.PARTIALLY_AVAILABLE;

export const filterEnabledCatalogSources = (
  catalogSources: CatalogSourceList | null,
): CatalogSourceList | null => {
  if (!catalogSources) {
    return null;
  }

  const filteredItems = catalogSources.items?.filter(
    (source) => source.enabled !== false && isSourceStatusActive(source.status),
  );

  return {
    ...catalogSources,
    items: filteredItems || [],
    size: filteredItems?.length || 0,
  };
};

export const getUniqueSourceLabels = (catalogSources: CatalogSourceList | null): string[] => {
  if (!catalogSources || !catalogSources.items) {
    return [];
  }

  const allLabels = new Set<string>();

  catalogSources.items.forEach((source) => {
    if (source.enabled && isSourceStatusActive(source.status) && source.labels.length > 0) {
      source.labels.forEach((label) => {
        if (label.trim()) {
          allLabels.add(label.trim());
        }
      });
    }
  });

  return Array.from(allLabels);
};

export const hasSourcesWithoutLabels = (catalogSources: CatalogSourceList | null): boolean => {
  if (!catalogSources || !catalogSources.items) {
    return false;
  }

  return catalogSources.items.some((source) => {
    if (source.enabled !== false && isSourceStatusActive(source.status)) {
      return source.labels.length === 0 || source.labels.every((label) => !label.trim());
    }
    return false;
  });
};

/**
 * Orders source labels according to the order in the catalog labels list.
 * Labels that appear in catalogLabels are ordered first (in the order they appear in the API),
 * followed by any labels found on sources that don't appear in catalogLabels.
 */
export const orderLabelsByPriority = (
  sourceLabels: string[],
  catalogLabels: CatalogLabelList | null,
): string[] => {
  if (!catalogLabels?.items) {
    return sourceLabels;
  }

  const orderedLabels: string[] = [];
  const remainingLabels = new Set(sourceLabels);

  catalogLabels.items.forEach((catalogLabel) => {
    if (catalogLabel.name === null) {
      return;
    }

    if (remainingLabels.has(catalogLabel.name)) {
      orderedLabels.push(catalogLabel.name);
      remainingLabels.delete(catalogLabel.name);
    }
  });

  orderedLabels.push(...Array.from(remainingLabels));

  return orderedLabels;
};

/**
 * Finds a label from the catalog labels list that matches the given source label name.
 * Handles the special case where sourceLabel is 'null' (other/unlabeled sources).
 */
export const findLabelData = (
  sourceLabel: string | undefined,
  catalogLabels: CatalogLabelList | null,
): CatalogLabel | undefined => {
  if (!catalogLabels?.items || !sourceLabel) {
    return undefined;
  }

  if (sourceLabel === SourceLabel.other) {
    return catalogLabels.items.find((label) => label.name === null);
  }

  return catalogLabels.items.find((label) => label.name === sourceLabel);
};

/**
 * Gets the display name for a source label, using the catalog labels data if available.
 * Falls back to the raw label name with the given suffix appended if no display name is found.
 */
export const getLabelDisplayName = (
  sourceLabel: string | undefined,
  catalogLabels: CatalogLabelList | null,
  otherFallback = 'Other models',
  categorySuffix = 'models',
): string => {
  if (!sourceLabel) {
    return '';
  }

  const labelData = findLabelData(sourceLabel, catalogLabels);

  if (labelData?.displayName) {
    return labelData.displayName;
  }

  if (sourceLabel === SourceLabel.other) {
    return otherFallback;
  }

  return sourceLabel.toLowerCase().endsWith(categorySuffix)
    ? sourceLabel
    : `${sourceLabel} ${categorySuffix}`;
};

/**
 * Gets the description for a source label from the catalog labels data.
 */
export const getLabelDescription = (
  sourceLabel: string | undefined,
  catalogLabels: CatalogLabelList | null,
): string | undefined => {
  const labelData = findLabelData(sourceLabel, catalogLabels);
  return labelData?.description;
};

export const getActiveSourceLabels = (
  catalogSources: CatalogSourceList | null,
  catalogLabels: CatalogLabelList | null,
): string[] => {
  const enabledSources = filterEnabledCatalogSources(catalogSources);
  const uniqueLabels = getUniqueSourceLabels(enabledSources);
  const orderedLabels = orderLabelsByPriority(uniqueLabels, catalogLabels);

  if (hasSourcesWithoutLabels(enabledSources)) {
    return [...orderedLabels, SourceLabel.other];
  }

  return orderedLabels;
};

/**
 * Checks if there are any catalog sources that have models/items available.
 * Returns true if at least one source has status AVAILABLE or PARTIALLY_AVAILABLE.
 */
export const hasSourcesWithModels = (catalogSources: CatalogSourceList | null): boolean => {
  if (!catalogSources?.items) {
    return false;
  }

  return catalogSources.items.some((source) => isSourceStatusActive(source.status));
};

/**
 * Filters catalog sources to only include those with available items.
 * A source has items if its status is AVAILABLE or PARTIALLY_AVAILABLE.
 */
export const filterSourcesWithModels = (
  catalogSources: CatalogSourceList | null,
): CatalogSourceList | null => {
  if (!catalogSources) {
    return null;
  }

  const filteredItems = catalogSources.items?.filter((source) =>
    isSourceStatusActive(source.status),
  );

  return {
    ...catalogSources,
    items: filteredItems || [],
    size: filteredItems?.length || 0,
  };
};
