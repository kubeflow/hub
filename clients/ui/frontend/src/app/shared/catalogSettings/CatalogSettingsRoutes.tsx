import * as React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import type { CatalogSettingsDefinition } from './types';

type CatalogSettingsRoutesProps = {
  definition: CatalogSettingsDefinition;
};

/**
 * Mounts the standard 4-route tree for a catalog settings section:
 *   /          → ListPage
 *   add-source → ManagePage
 *   manage-source/:catalogSourceId → ManagePage
 *   *          → redirect to /
 *
 * Wraps routes in the catalog's ContextProvider (from definition).
 */
const CatalogSettingsRoutes: React.FC<CatalogSettingsRoutesProps> = ({ definition }) => {
  const { ContextProvider, ListPage, ManagePage } = definition;
  return (
    <ContextProvider>
      <Routes>
        <Route path="/" element={<ListPage />} />
        <Route path="add-source" element={<ManagePage />} />
        <Route path="manage-source/:catalogSourceId" element={<ManagePage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </ContextProvider>
  );
};

export default CatalogSettingsRoutes;
