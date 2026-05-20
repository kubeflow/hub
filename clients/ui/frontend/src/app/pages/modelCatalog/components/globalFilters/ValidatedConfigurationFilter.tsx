import * as React from 'react';
import { Divider, StackItem } from '@patternfly/react-core';
import ModelCatalogStringFilter from '~/app/pages/modelCatalog/components/ModelCatalogStringFilter';
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
  const validatedConfiguration = filters?.[filterKey];

  if (!validatedConfiguration) {
    return null;
  }

  return (
    <>
      <StackItem>
        <ModelCatalogStringFilter<ModelCatalogStringFilterKey.VALIDATED_CONFIGURATION>
          title="Validated arguments"
          filterKey={filterKey}
          filterToNameMapping={MODEL_CATALOG_VALIDATED_CONFIGURATION_NAME_MAPPING}
          filters={validatedConfiguration}
        />
      </StackItem>
      <Divider />
    </>
  );
};

export default ValidatedConfigurationFilter;
