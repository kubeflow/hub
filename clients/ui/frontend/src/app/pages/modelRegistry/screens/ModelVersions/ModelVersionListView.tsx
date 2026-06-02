import * as React from 'react';
import {
  Alert,
  Button,
  Dropdown,
  DropdownItem,
  DropdownList,
  MenuToggle,
  MenuToggleElement,
  ToolbarItem,
} from '@patternfly/react-core';
import { EllipsisVIcon } from '@patternfly/react-icons';
import { useNavigate } from 'react-router-dom';
import { ProjectObjectType, typedEmptyImage, ToolbarFilter, FilterState } from 'mod-arch-shared';
import { ModelVersion, RegisteredModel } from '~/app/types';
import { ModelRegistrySelectorContext } from '~/app/context/ModelRegistrySelectorContext';
import EmptyModelRegistryState from '~/app/pages/modelRegistry/screens/components/EmptyModelRegistryState';
import {
  modelVersionArchiveUrl,
  registerVersionForModelUrl,
} from '~/app/pages/modelRegistry/screens/routeUtils';
import {
  filterModelVersions,
  getTextValue,
  sortModelVersionsByCreateTime,
} from '~/app/pages/modelRegistry/screens/utils';
import ModelVersionsTable from '~/app/pages/modelRegistry/screens/ModelVersions/ModelVersionsTable';
import { filterArchiveVersions, filterLiveVersions } from '~/app/utils';
import {
  ModelRegistryVersionsFilterDataType,
  ModelRegistryVersionsFilterOptions,
  modelVersionsFilterConfig,
  modelVersionsVisibleFilterKeys,
  modelVersionsInitialFilterValues,
} from '~/app/pages/modelRegistry/screens/const';

type ModelVersionListViewProps = {
  modelVersions: ModelVersion[];
  registeredModel: RegisteredModel;
  isArchiveModel?: boolean;
  refresh: () => void;
};

const ModelVersionListView: React.FC<ModelVersionListViewProps> = ({
  modelVersions,
  registeredModel: rm,
  isArchiveModel,
  refresh,
}) => {
  const unfilteredModelVersions = isArchiveModel
    ? modelVersions
    : filterLiveVersions(modelVersions);

  const archiveModelVersions = filterArchiveVersions(modelVersions);
  const navigate = useNavigate();
  const { preferredModelRegistry } = React.useContext(ModelRegistrySelectorContext);
  const [filterValues, setFilterValues] = React.useState<
    FilterState<ModelRegistryVersionsFilterOptions>
  >(modelVersionsInitialFilterValues);

  const onFilterChange = React.useCallback(
    (key: ModelRegistryVersionsFilterOptions, value: string | string[]) =>
      setFilterValues((prev) => ({ ...prev, [key]: value })),
    [],
  );

  const onClearAllFilters = React.useCallback(
    () => setFilterValues(modelVersionsInitialFilterValues),
    [],
  );

  const [isArchivedModelVersionKebabOpen, setIsArchivedModelVersionKebabOpen] =
    React.useState(false);

  const filterData: ModelRegistryVersionsFilterDataType = {
    [ModelRegistryVersionsFilterOptions.keyword]: getTextValue(
      filterValues[ModelRegistryVersionsFilterOptions.keyword],
    ),
    [ModelRegistryVersionsFilterOptions.author]: getTextValue(
      filterValues[ModelRegistryVersionsFilterOptions.author],
    ),
  };
  const filteredModelVersions = filterModelVersions(unfilteredModelVersions, filterData);
  const date = rm.lastUpdateTimeSinceEpoch && new Date(parseInt(rm.lastUpdateTimeSinceEpoch));

  if (unfilteredModelVersions.length === 0) {
    if (isArchiveModel) {
      return (
        <EmptyModelRegistryState
          testid="empty-archive-model-versions"
          title="No versions"
          headerIcon={() => (
            <img
              src={typedEmptyImage(ProjectObjectType.registeredModels, 'MissingVersion')}
              alt="missing version"
            />
          )}
          description={`${rm.name} has no registered versions.`}
        />
      );
    }
    return (
      <EmptyModelRegistryState
        testid="empty-model-versions"
        title="No versions"
        headerIcon={() => (
          <img
            src={typedEmptyImage(ProjectObjectType.registeredModels, 'MissingVersion')}
            alt="missing version"
          />
        )}
        description={`${rm.name} has no registered versions. Register a version to this model.`}
        primaryActionText="Register new version"
        primaryActionOnClick={() => {
          navigate(registerVersionForModelUrl(rm.id, preferredModelRegistry?.name));
        }}
        secondaryActionText={
          archiveModelVersions.length !== 0 ? 'View archived versions' : undefined
        }
        secondaryActionOnClick={() => {
          navigate(modelVersionArchiveUrl(rm.id, preferredModelRegistry?.name));
        }}
      />
    );
  }

  const toolbarActions = !isArchiveModel ? (
    <>
      <ToolbarItem>
        <Button
          variant="primary"
          onClick={() => {
            navigate(registerVersionForModelUrl(rm.id, preferredModelRegistry?.name));
          }}
        >
          Register new version
        </Button>
      </ToolbarItem>
      <ToolbarItem>
        <Dropdown
          isOpen={isArchivedModelVersionKebabOpen}
          onSelect={() => setIsArchivedModelVersionKebabOpen(false)}
          onOpenChange={(isOpen: boolean) => setIsArchivedModelVersionKebabOpen(isOpen)}
          toggle={(tr: React.Ref<MenuToggleElement>) => (
            <MenuToggle
              data-testid="model-versions-table-kebab-action"
              ref={tr}
              variant="plain"
              onClick={() => setIsArchivedModelVersionKebabOpen(!isArchivedModelVersionKebabOpen)}
              isExpanded={isArchivedModelVersionKebabOpen}
              aria-label="View archived versions"
            >
              <EllipsisVIcon />
            </MenuToggle>
          )}
          shouldFocusToggleOnSelect
          popperProps={{ appendTo: 'inline' }}
        >
          <DropdownList>
            <DropdownItem
              onClick={() => navigate(modelVersionArchiveUrl(rm.id, preferredModelRegistry?.name))}
            >
              View archived versions
            </DropdownItem>
          </DropdownList>
        </Dropdown>
      </ToolbarItem>
    </>
  ) : undefined;

  return (
    <>
      {isArchiveModel && (
        <Alert
          variant="warning"
          isInline
          title={`All the versions have been archived along with the model on ${
            date
              ? `${date.toLocaleString('en-US', {
                  month: 'long',
                  timeZone: 'UTC',
                })} ${date.getUTCDate()}, ${date.getUTCFullYear()}`
              : '--'
          }. They are now read-only and can only be restored together with the model.`}
        />
      )}
      <ModelVersionsTable
        refresh={refresh}
        isArchiveModel={isArchiveModel}
        clearFilters={onClearAllFilters}
        modelVersions={sortModelVersionsByCreateTime(filteredModelVersions)}
        toolbarContent={
          <ToolbarFilter
            filterConfig={modelVersionsFilterConfig}
            visibleFilterKeys={modelVersionsVisibleFilterKeys}
            filterValues={filterValues}
            onFilterChange={onFilterChange}
            onClearAllFilters={onClearAllFilters}
            toolbarActions={toolbarActions}
            testIdPrefix="model-versions-table"
          />
        }
      />
    </>
  );
};

export default ModelVersionListView;
