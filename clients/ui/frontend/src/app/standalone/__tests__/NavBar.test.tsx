import '@testing-library/jest-dom';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import {
  ModularArchConfig,
  DeploymentMode,
  useNamespaceSelector,
  useModularArchContext,
} from 'mod-arch-core';
import { useThemeContext } from 'mod-arch-kubeflow';
import { ColorModeProvider } from '~/app/context/ThemeContext';
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

const renderNavBar = (props?: Partial<React.ComponentProps<typeof NavBar>>) =>
  render(
    <ColorModeProvider>
      <NavBar onLogout={jest.fn()} {...props} />
    </ColorModeProvider>,
  );

describe('NavBar mandatory namespace functionality', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
    document.documentElement.className = '';
    // Mock fetch for script loading
    global.fetch = jest.fn().mockResolvedValue({ ok: false });

    // Set up default mocks for hooks
    mockUseThemeContext.mockReturnValue({
      isMUITheme: false,
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
    localStorage.clear();
    document.documentElement.className = '';
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

    renderNavBar();

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

    renderNavBar();

    // Check that the namespace selector is present and enabled
    const namespaceButton = screen.getByText('namespace-1');
    expect(namespaceButton).toBeInTheDocument();

    // The MenuToggle button should be enabled (not disabled)
    const menuToggle = namespaceButton.closest('button');
    expect(menuToggle).not.toBeDisabled();
  });

  it('should render and toggle theme mode on button click', () => {
    renderNavBar();

    const themeToggle = screen.getByRole('button', { name: /toggle dark mode/i });
    expect(themeToggle).toBeInTheDocument();
    expect(themeToggle).toHaveAttribute('title', 'Switch to Dark Mode');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(false);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(false);

    // Toggle to Dark Mode
    fireEvent.click(themeToggle);

    expect(document.documentElement.classList.contains('dark-theme')).toBe(true);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(true);
    expect(localStorage.getItem('kubeflow_theme_mode')).toBe('dark');
    expect(themeToggle).toHaveAttribute('title', 'Switch to Light Mode');

    // Toggle back to Light Mode
    fireEvent.click(themeToggle);

    expect(document.documentElement.classList.contains('dark-theme')).toBe(false);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(false);
    expect(localStorage.getItem('kubeflow_theme_mode')).toBe('light');
    expect(themeToggle).toHaveAttribute('title', 'Switch to Dark Mode');
  });
});
