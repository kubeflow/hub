import * as React from 'react';
import CatalogSettingsRoutes from '~/app/shared/catalogSettings/CatalogSettingsRoutes';
import { skillCatalogSettingsDefinition } from './definition';

const SkillCatalogSettingsRoutes: React.FC = () => (
  <CatalogSettingsRoutes definition={skillCatalogSettingsDefinition} />
);

export default SkillCatalogSettingsRoutes;
