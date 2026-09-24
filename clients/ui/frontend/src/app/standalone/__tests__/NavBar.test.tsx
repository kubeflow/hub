import '@testing-library/jest-dom';
import React from 'react';
import { render, screen } from '@testing-library/react';
import {
  ModularArchConfig,
  DeploymentMode,
  useNamespaceSelector,
  useModularArchContext,
} from 'mod-arch-core';
import { useThemeContext } from 'mod-arch-kubeflow';
import { CustomThemeProvider } from '~/app/context/ThemeContext';
import NavBar from '~/app/standalone/NavBar';

// Mock the utilities
jest.mock('mod-arch-core', () => ({
  ...jest.requireActual('mod-arch-core'),
  useNamespaceSelector: jest.fn(),
  useModularArchContext: jest.fn(),
}));

jest.mock('mod-arch-kubeflow', () => ({
  useThemeContext: jest.fn(),
}));

const mockUseNamespaceSelector = useNamespaceSelector as jest.MockedFunction<
  typeof useNamespaceSelector
>;
const mockUseModularArchContext = useModularArchContext as jest.MockedFunction<
  typeof useModularArchContext
>;
const mockUseThemeContext = useThemeContext as jest.MockedFunction<typeof useThemeContext>;

const createMockConfig = (
  mandatoryNamespace?: string,
  deploymentMode: DeploymentMode = DeploymentMode.Standalone,
): ModularArchConfig => ({
  deploymentMode,
  URL_PREFIX: 'test',
  BFF_API_VERSION: 'v1',
  ...(mandatoryNamespace && { mandatoryNamespace }),
});

describe('NavBar mandatory namespace functionality', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Mock fetch for script loading
    global.fetch = jest.fn().mockResolvedValue({ ok: false });

    // Set up default mocks for hooks
    mockUseThemeContext.mockReturnValue({
      isMUITheme: false,
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
    // Clean up fetch stub explicitly
    // @ts-expect-error – fetch might be undefined in node
    delete global.fetch;
  });

  it('should disable namespace selection when mandatory namespace is set', async () => {
    const mandatoryNamespace = 'mandatory-namespace';
    const config = createMockConfig(mandatoryNamespace);

    // Mock useModularArchContext to return the config with mandatory namespace
    mockUseModularArchContext.mockReturnValue({
      config,
    } as ReturnType<typeof useModularArchContext>);

    // Mock useNamespaceSelector to return only the mandatory namespace
    mockUseNamespaceSelector.mockReturnValue({
      namespaces: [{ name: mandatoryNamespace }],
      preferredNamespace: { name: mandatoryNamespace },
      updatePreferredNamespace: jest.fn(),
      clearStoredNamespace: jest.fn(),
      namespacesLoaded: true,
      namespacesLoadError: undefined,
      initializationError: undefined,
    } as ReturnType<typeof useNamespaceSelector>);

    render(
      <CustomThemeProvider>
        <NavBar onLogout={jest.fn()} />
      </CustomThemeProvider>,
    );

    const namespaceButton = await screen.findByText(mandatoryNamespace);
    expect(namespaceButton).toBeInTheDocument();

    // The MenuToggle button should be disabled due to mandatory namespace
    const menuToggle = namespaceButton.closest('button');
    expect(menuToggle).toBeDisabled();
  });

  it('should allow namespace selection when no mandatory namespace is set', async () => {
    const config = createMockConfig();
    const mockNamespaces = [{ name: 'namespace-1' }, { name: 'namespace-2' }];

    // Mock useModularArchContext to return the config without mandatory namespace
    mockUseModularArchContext.mockReturnValue({
      config,
    } as ReturnType<typeof useModularArchContext>);

    // Mock useNamespaceSelector to return multiple namespaces
    mockUseNamespaceSelector.mockReturnValue({
      namespaces: mockNamespaces,
      preferredNamespace: { name: 'namespace-1' },
      updatePreferredNamespace: jest.fn(),
      clearStoredNamespace: jest.fn(),
      namespacesLoaded: true,
      namespacesLoadError: undefined,
      initializationError: undefined,
    } as ReturnType<typeof useNamespaceSelector>);

    render(
      <CustomThemeProvider>
        <NavBar onLogout={jest.fn()} />
      </CustomThemeProvider>,
    );

    // Check that the namespace selector is present and enabled
    const namespaceButton = screen.getByText('namespace-1');
    expect(namespaceButton).toBeInTheDocument();

    // The MenuToggle button should be enabled (not disabled)
    const menuToggle = namespaceButton.closest('button');
    expect(menuToggle).not.toBeDisabled();
  });

  it('should render theme toggle button', () => {
    render(
      <CustomThemeProvider>
        <NavBar onLogout={jest.fn()} />
      </CustomThemeProvider>,
    );

    const themeToggle = screen.getByRole('button', { name: /toggle dark mode/i });
    expect(themeToggle).toBeInTheDocument();
  });
});
