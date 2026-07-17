import * as React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { AgentsCatalogContextProvider } from '~/app/context/agentsCatalog/AgentsCatalogContext';
import AgentsCatalogCoreLoader from './AgentsCatalogCoreLoader';
import AgentsCatalog from './screens/AgentsCatalog';
import AgentDetailsPage from './screens/AgentDetailsPage';

const AgentsCatalogRoutes: React.FC = () => (
  <AgentsCatalogContextProvider>
    <Routes>
      <Route path="/*" element={<AgentsCatalogCoreLoader />}>
        <Route index element={<AgentsCatalog />} />
        <Route path=":agentId">
          <Route index element={<Navigate to="overview" replace />} />
          <Route path="overview" element={<AgentDetailsPage />} />
          <Route path="*" element={<Navigate to="." />} />
        </Route>
        <Route path="*" element={<Navigate to="." />} />
      </Route>
    </Routes>
  </AgentsCatalogContextProvider>
);

export default AgentsCatalogRoutes;
