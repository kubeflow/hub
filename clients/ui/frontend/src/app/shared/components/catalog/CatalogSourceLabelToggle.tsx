import * as React from 'react';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';
import type { CatalogLabelList, CatalogSourceList } from '~/app/modelCatalogTypes';
import { SourceLabel } from '~/app/modelCatalogTypes';
import {
  filterEnabledCatalogSources,
  getLabelDisplayName,
  getUniqueSourceLabels,
  hasSourcesWithoutLabels,
  orderLabelsByPriority,
} from '~/app/pages/modelCatalog/utils/modelCatalogUtils';

type SourceLabelBlock = {
  id: string;
  label: string | undefined;
  displayName: string;
};

type CatalogSourceLabelToggleProps = {
  catalogSources: CatalogSourceList | null;
  catalogLabels: CatalogLabelList | null;
  selectedSourceLabel: string | undefined;
  onSelectSourceLabel: (label: string | undefined) => void;
  allBlockLabel?: string | undefined;
  allBlockDisplayName: string;
  className?: string;
  testId?: string;
  ariaLabel?: string;
  hideWhenSingleCategory?: boolean;
  getLabelDisplayNameOverride?: (label: string) => string;
  getTestId?: (blockId: string) => string;
};

const CatalogSourceLabelToggle: React.FC<CatalogSourceLabelToggleProps> = ({
  catalogSources,
  catalogLabels,
  selectedSourceLabel,
  onSelectSourceLabel,
  allBlockLabel,
  allBlockDisplayName,
  className,
  testId,
  ariaLabel = 'Category selection',
  hideWhenSingleCategory = false,
  getLabelDisplayNameOverride,
  getTestId,
}) => {
  const blocks: SourceLabelBlock[] = React.useMemo(() => {
    if (!catalogSources) {
      return [];
    }

    const enabledSources = filterEnabledCatalogSources(catalogSources);
    const uniqueLabels = getUniqueSourceLabels(enabledSources);
    const hasNoLabels = hasSourcesWithoutLabels(enabledSources);
    const orderedLabels = orderLabelsByPriority(uniqueLabels, catalogLabels);

    const allBlock: SourceLabelBlock = {
      id: 'all',
      label: allBlockLabel,
      displayName: allBlockDisplayName,
    };

    const getDisplayName = (label: string): string =>
      getLabelDisplayNameOverride
        ? getLabelDisplayNameOverride(label)
        : getLabelDisplayName(label, catalogLabels);

    const labelBlocks: SourceLabelBlock[] = orderedLabels.map((label) => ({
      id: `label-${label}`,
      label,
      displayName: getDisplayName(label),
    }));

    const result: SourceLabelBlock[] = [allBlock, ...labelBlocks];

    if (hasNoLabels) {
      result.push({
        id: 'no-labels',
        label: SourceLabel.other,
        displayName: getDisplayName(SourceLabel.other),
      });
    }

    return result;
  }, [
    catalogSources,
    catalogLabels,
    allBlockLabel,
    allBlockDisplayName,
    getLabelDisplayNameOverride,
  ]);

  if (!catalogSources) {
    return null;
  }

  if (hideWhenSingleCategory && blocks.length - 1 <= 1) {
    return null;
  }

  return (
    <ToggleGroup aria-label={ariaLabel} className={className} data-testid={testId}>
      {blocks.map((block) => (
        <ToggleGroupItem
          buttonId={block.id}
          data-testid={getTestId ? getTestId(block.id) : block.id}
          key={block.id}
          text={block.displayName}
          isSelected={block.label === selectedSourceLabel}
          onChange={() => onSelectSourceLabel(block.label)}
        />
      ))}
    </ToggleGroup>
  );
};

export default CatalogSourceLabelToggle;
