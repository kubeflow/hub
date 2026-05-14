import * as React from 'react';
import { Content, ContentVariants, Divider, StackItem } from '@patternfly/react-core';
import ModelCatalogStringFilter from '~/app/pages/modelCatalog/components/ModelCatalogStringFilter';
import { ModelCatalogContext } from '~/app/context/modelCatalog/ModelCatalogContext';
import {
  ModelCatalogStringFilterKey,
  MODEL_CATALOG_VALIDATED_CONFIGURATION_NAME_MAPPING,
} from '~/concepts/modelCatalog/const';
import { CatalogFilterOptions, ModelCatalogStringFilterOptions } from '~/app/modelCatalogTypes';

const filterKey = ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION;

type ValidatedConfigurationFilterProps = {
  filters?: Extract<CatalogFilterOptions, Partial<ModelCatalogStringFilterOptions>>;
};

const ValidatedConfigurationFilter: React.FC<ValidatedConfigurationFilterProps> = ({ filters }) => {
  const { filterData } = React.useContext(ModelCatalogContext);
  const validatedConfiguration = filters?.[filterKey];
  const filterValues = validatedConfiguration?.values ?? [];
  const hasMultipleOptions = filterValues.length > 1;
  const hasSelection = filterData[filterKey].length > 0;

  if (!validatedConfiguration) {
    return null;
  }

  return (
    <>
      <StackItem>
        <ModelCatalogStringFilter<ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION>
          title="Validated configuration"
          filterKey={filterKey}
          filterToNameMapping={MODEL_CATALOG_VALIDATED_CONFIGURATION_NAME_MAPPING}
          filters={validatedConfiguration}
        />
        {hasMultipleOptions && hasSelection && (
          <Content component={ContentVariants.small} className="pf-v6-u-mt-sm">
            Showing models with all selected configurations
          </Content>
        )}
      </StackItem>
      <Divider />
    </>
  );
};

export default ValidatedConfigurationFilter;
