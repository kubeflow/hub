import React from 'react';
import { Stack } from '@patternfly/react-core';
import type { CatalogLabelList, CatalogSourceList } from '~/app/shared/types/catalogTypes';
import { SourceLabel } from '~/app/shared/types/catalogTypes';
import {
  filterEnabledCatalogSources,
  getUniqueSourceLabels,
  hasSourcesWithoutLabels,
  orderLabelsByPriority,
} from './utils/catalogSourceUtils';

type CatalogAllItemsViewProps = {
  searchTerm: string;
  catalogSources: CatalogSourceList | null;
  catalogLabels: CatalogLabelList | null;
  pageSize?: number;
  otherSectionKey?: string;
  onShowMore: (label: string) => void;
  renderCategorySection: (
    label: string,
    searchTerm: string,
    pageSize: number,
    onShowMore: (label: string) => void,
  ) => React.ReactNode;
};

const CatalogAllItemsView: React.FC<CatalogAllItemsViewProps> = ({
  searchTerm,
  catalogSources,
  catalogLabels,
  pageSize = 4,
  otherSectionKey,
  onShowMore,
  renderCategorySection,
}) => {
  const sourceLabels = React.useMemo(() => {
    const enabledSources = filterEnabledCatalogSources(catalogSources);
    const uniqueLabels = getUniqueSourceLabels(enabledSources);
    // Order labels according to catalogLabels priority
    return orderLabelsByPriority(uniqueLabels, catalogLabels);
  }, [catalogSources, catalogLabels]);

  const hasSourcesWithoutLabelsValue = React.useMemo(
    () => hasSourcesWithoutLabels(catalogSources),
    [catalogSources],
  );

  return (
    <Stack hasGutter>
      {sourceLabels.map((label) => (
        <React.Fragment key={label}>
          {renderCategorySection(label, searchTerm, pageSize, onShowMore)}
        </React.Fragment>
      ))}
      {hasSourcesWithoutLabelsValue && (
        <React.Fragment key={otherSectionKey ?? 'other'}>
          {renderCategorySection(SourceLabel.other, searchTerm, pageSize, onShowMore)}
        </React.Fragment>
      )}
    </Stack>
  );
};

export default CatalogAllItemsView;
