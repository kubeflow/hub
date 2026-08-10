import * as React from 'react';
import CatalogSettingsRoutes from '~/app/shared/catalogSettings/CatalogSettingsRoutes';
import { modelCatalogSettingsDefinition } from './definition';

const ModelCatalogSettingsRoutes: React.FC = () => (
  <CatalogSettingsRoutes definition={modelCatalogSettingsDefinition} />
);

export default ModelCatalogSettingsRoutes;
