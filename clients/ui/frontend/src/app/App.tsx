import * as React from 'react';
import '@patternfly/react-core/dist/styles/base.css';
import './app.css';
import {
  Alert,
  Bullseye,
  Button,
  Page,
  PageSidebar,
  Spinner,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import {
  DeploymentMode,
  logout,
  useModularArchContext,
  useNamespaceSelector,
  useSettings,
} from 'mod-arch-core';
import NavBar from '~/app/standalone/NavBar';
import ToastNotifications from '~/app/standalone/ToastNotifications';
import AppNavSidebar from '~/app/standalone/AppNavSidebar';
import AppRoutes from '~/app/AppRoutes';
import { AppContext } from '~/app/context/AppContext';
import { CustomThemeProvider } from '~/app/context/ThemeContext';
import { ModelRegistrySelectorContextProvider } from '~/app/context/ModelRegistrySelectorContext';
import { TransferJobNotificationsProvider } from '~/app/context/TransferJobNotificationsContext';

const App: React.FC = () => {
  const {
    configSettings,
    userSettings,
    loaded: configLoaded,
    loadError: configError,
  } = useSettings();

  const { namespacesLoaded, namespacesLoadError, initializationError } = useNamespaceSelector();

  const username = userSettings?.userId;
  const { config } = useModularArchContext();
  const { deploymentMode } = config;
  const isStandalone = deploymentMode === DeploymentMode.Standalone;

  const contextValue = React.useMemo(
    () =>
      configSettings && userSettings
        ? {
            config: configSettings!,
            user: userSettings!,
          }
        : null,
    [configSettings, userSettings],
  );

  const error = configError || namespacesLoadError || initializationError;

  const sidebar = <PageSidebar isSidebarOpen={false} />;

  // We lack the critical data to startup the app
  if (error) {
    // There was an error fetching critical data
    return (
      <div className="error-page-container">
        <Stack hasGutter>
          <StackItem>
            <Alert variant="danger" isInline title="General loading error">
              <p>
                {configError?.message ||
                  namespacesLoadError?.message ||
                  initializationError?.message ||
                  'Unknown error occurred during startup'}
              </p>
              <p>Logging out and logging back in may solve the issue</p>
            </Alert>
          </StackItem>
          <StackItem>
            <Button
              variant="secondary"
              onClick={() => logout().then(() => window.location.reload())}
            >
              Logout
            </Button>
          </StackItem>
        </Stack>
      </div>
    );
  }

  // Waiting on the API to finish
  const loading =
    !configLoaded || !userSettings || !configSettings || !contextValue || !namespacesLoaded;

  return loading ? (
    <Bullseye>
      <Spinner />
    </Bullseye>
  ) : (
    <CustomThemeProvider>
      <AppContext.Provider value={contextValue}>
        <TransferJobNotificationsProvider>
          <Page
            mainContainerId="primary-app-container"
            masthead={
              isStandalone ? (
                <NavBar
                  username={username}
                  onLogout={() => {
                    logout().then(() => window.location.reload());
                  }}
                />
              ) : (
                ''
              )
            }
            isManagedSidebar={isStandalone}
            sidebar={isStandalone ? <AppNavSidebar /> : sidebar}
          >
            <ModelRegistrySelectorContextProvider>
              <AppRoutes />
            </ModelRegistrySelectorContextProvider>
            <ToastNotifications />
          </Page>
        </TransferJobNotificationsProvider>
      </AppContext.Provider>
    </CustomThemeProvider>
  );
};

export default App;
