import * as React from 'react';
import { Td } from '@patternfly/react-table';
import { SourceConfigsTable, SourceVisibilityLabelInfo } from '~/app/shared/catalogSettings';
import { CatalogSourceConfig } from '~/app/modelCatalogTypes';
import { ModelCatalogSettingsContext } from '~/app/context/modelCatalogSettings/ModelCatalogSettingsContext';
import {
  ADD_SOURCE_TITLE,
  manageSourceUrl,
} from '~/app/routes/modelCatalogSettings/modelCatalogSettings';
import {
  CATALOG_SOURCE_TYPE_LABELS,
  ModelVisibilityBadgeColor,
} from '~/concepts/modelCatalogSettings/const';
import { hasSourceFilters, getOrganizationDisplay } from '~/concepts/modelCatalogSettings/utils';
import { TABLE_COLUMN_LABELS } from '~/app/pages/modelCatalogSettings/constants';
import CatalogSourceStatus from '~/app/pages/modelCatalogSettings/components/CatalogSourceStatus';
import { catalogSourceConfigsColumns } from './CatalogSourceConfigsTableColumns';

type CatalogSourceConfigsTableProps = {
  catalogSourceConfigs: CatalogSourceConfig[];
  onAddSource: () => void;
  onDeleteSource: (sourceId: string) => Promise<void>;
};

const getVisibilityLabel = (config: CatalogSourceConfig): SourceVisibilityLabelInfo =>
  hasSourceFilters(config)
    ? {
        text: 'Filtered',
        color: ModelVisibilityBadgeColor.FILTERED,
        testId: `model-visibility-filtered-${config.id}`,
      }
    : {
        text: 'All models',
        color: ModelVisibilityBadgeColor.UNFILTERED,
        variant: 'outline',
        testId: `model-visibility-unfiltered-${config.id}`,
      };

const StatusComponent: React.FC<{ sourceConfig: CatalogSourceConfig }> = ({ sourceConfig }) => (
  <CatalogSourceStatus catalogSourceConfig={sourceConfig} />
);

const CatalogSourceConfigsTable: React.FC<CatalogSourceConfigsTableProps> = ({
  catalogSourceConfigs,
  onAddSource,
  onDeleteSource,
}) => {
  const {
    apiState,
    refreshCatalogSourceConfigs,
    refreshCatalogSources,
    catalogSourcesLoadError,
    catalogSources,
    markSourcePending,
  } = React.useContext(ModelCatalogSettingsContext);

  const handleToggleUpdate = React.useCallback(
    async (checked: boolean, config: CatalogSourceConfig) => {
      if (checked) {
        const previousStatus = catalogSources?.items?.find((s) => s.id === config.id)?.status ?? '';
        markSourcePending(config.id, previousStatus);
      }
      await apiState.api.updateCatalogSourceConfig({}, config.id, { enabled: checked });
      refreshCatalogSourceConfigs();
      if (checked) {
        refreshCatalogSources();
      }
    },
    [
      apiState.api,
      catalogSources,
      markSourcePending,
      refreshCatalogSourceConfigs,
      refreshCatalogSources,
    ],
  );

  const renderExtraCells = React.useCallback(
    (config: CatalogSourceConfig) => (
      <Td dataLabel="Organization" style={{ verticalAlign: 'middle' }}>
        <span data-testid={`source-organization-${config.id}`}>
          {getOrganizationDisplay(config, config.isDefault ?? false)}
        </span>
      </Td>
    ),
    [],
  );

  const deleteModalBody = React.useCallback(
    (config: CatalogSourceConfig) => (
      <>
        The <strong>{config.name}</strong> repository will be deleted, and its models will be
        removed from the model catalog.
      </>
    ),
    [],
  );

  return (
    <SourceConfigsTable
      sourceConfigs={catalogSourceConfigs}
      columns={catalogSourceConfigsColumns}
      onAddSource={onAddSource}
      addSourceLabel={ADD_SOURCE_TITLE}
      onDeleteSource={onDeleteSource}
      apiAvailable={apiState.apiAvailable}
      onToggleUpdate={handleToggleUpdate}
      loadError={catalogSourcesLoadError}
      getManageSourceUrl={manageSourceUrl}
      renderExtraCells={renderExtraCells}
      visibilityColumnLabel={TABLE_COLUMN_LABELS.MODEL_VISIBILITY}
      getVisibilityLabel={getVisibilityLabel}
      getSourceTypeLabel={(config) => CATALOG_SOURCE_TYPE_LABELS[config.type]}
      StatusComponent={StatusComponent}
      deleteModalBody={deleteModalBody}
    />
  );
};

export default CatalogSourceConfigsTable;
