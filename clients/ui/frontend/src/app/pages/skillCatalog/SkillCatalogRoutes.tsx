import * as React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { SkillCatalogContextProvider } from '~/app/context/skillCatalog/SkillCatalogContext';
import SkillCatalogCoreLoader from './SkillCatalogCoreLoader';
import SkillCatalog from './screens/SkillCatalog';

const SkillCatalogRoutes: React.FC = () => (
  <SkillCatalogContextProvider>
    <Routes>
      <Route path="/*" element={<SkillCatalogCoreLoader />}>
        <Route index element={<SkillCatalog />} />
        <Route path="*" element={<Navigate to="." />} />
      </Route>
    </Routes>
  </SkillCatalogContextProvider>
);

export default SkillCatalogRoutes;
