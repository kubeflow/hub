import * as React from 'react';
import CatalogSettingsRoutes from '~/app/shared/catalogSettings/CatalogSettingsRoutes';
import { mcpCatalogSettingsDefinition } from './definition';

const McpCatalogSettingsRoutes: React.FC = () => (
  <CatalogSettingsRoutes definition={mcpCatalogSettingsDefinition} />
);

export default McpCatalogSettingsRoutes;
