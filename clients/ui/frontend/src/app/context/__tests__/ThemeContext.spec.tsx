import '@testing-library/jest-dom';
import React from 'react';
import { render, screen, fireEvent, renderHook } from '@testing-library/react';
import { ColorModeProvider, useColorMode, STORAGE_KEY } from '~/app/context/ThemeContext';

const TestConsumer: React.FC = () => {
  const { colorMode, toggleColorMode, setColorMode } = useColorMode();
  return (
    <div>
      <span data-testid="current-mode">{colorMode}</span>
      <button data-testid="toggle-btn" onClick={toggleColorMode}>
        Toggle
      </button>
      <button data-testid="set-dark-btn" onClick={() => setColorMode('dark')}>
        Set Dark
      </button>
      <button data-testid="set-light-btn" onClick={() => setColorMode('light')}>
        Set Light
      </button>
    </div>
  );
};

describe('ThemeContext / ColorModeProvider', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.className = '';
  });

  afterEach(() => {
    localStorage.clear();
    document.documentElement.className = '';
  });

  it('should throw an error when useColorMode is used outside of ColorModeProvider', () => {
    // Suppress console.error during expected error boundary / throw
    const consoleError = jest.spyOn(console, 'error').mockImplementation(() => undefined);
    expect(() => renderHook(() => useColorMode())).toThrow(
      'useColorMode must be used within a ColorModeProvider',
    );
    consoleError.mockRestore();
  });

  it('should default to light mode when localStorage is empty and no dark preference is matched', () => {
    render(
      <ColorModeProvider>
        <TestConsumer />
      </ColorModeProvider>,
    );

    expect(screen.getByTestId('current-mode')).toHaveTextContent('light');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(false);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(false);
  });

  it('should respect OS dark mode preference via matchMedia when localStorage is empty', () => {
    const originalMatchMedia = window.matchMedia;
    window.matchMedia = jest.fn().mockImplementation((query: string) => ({
      matches: query === '(prefers-color-scheme: dark)',
      media: query,
      onchange: null,
      addListener: jest.fn(),
      removeListener: jest.fn(),
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
      dispatchEvent: jest.fn(),
    }));

    render(
      <ColorModeProvider>
        <TestConsumer />
      </ColorModeProvider>,
    );

    expect(screen.getByTestId('current-mode')).toHaveTextContent('dark');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(true);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(true);

    window.matchMedia = originalMatchMedia;
  });

  it('should restore saved dark mode from localStorage on mount', () => {
    localStorage.setItem(STORAGE_KEY, 'dark');

    render(
      <ColorModeProvider>
        <TestConsumer />
      </ColorModeProvider>,
    );

    expect(screen.getByTestId('current-mode')).toHaveTextContent('dark');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(true);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(true);
  });

  it('should safely fall back to light mode when localStorage contains invalid value', () => {
    localStorage.setItem(STORAGE_KEY, 'invalid-theme-value');

    render(
      <ColorModeProvider>
        <TestConsumer />
      </ColorModeProvider>,
    );

    expect(screen.getByTestId('current-mode')).toHaveTextContent('light');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(false);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(false);
  });

  it('should toggle between light and dark modes, updating classList and localStorage', () => {
    render(
      <ColorModeProvider>
        <TestConsumer />
      </ColorModeProvider>,
    );

    const toggleBtn = screen.getByTestId('toggle-btn');
    const currentMode = screen.getByTestId('current-mode');

    // Initially light
    expect(currentMode).toHaveTextContent('light');

    // Toggle to dark
    fireEvent.click(toggleBtn);
    expect(currentMode).toHaveTextContent('dark');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(true);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(true);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('dark');

    // Toggle back to light
    fireEvent.click(toggleBtn);
    expect(currentMode).toHaveTextContent('light');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(false);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(false);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('light');
  });

  it('should allow setting color mode directly via setColorMode', () => {
    render(
      <ColorModeProvider>
        <TestConsumer />
      </ColorModeProvider>,
    );

    const setDarkBtn = screen.getByTestId('set-dark-btn');
    const setLightBtn = screen.getByTestId('set-light-btn');
    const currentMode = screen.getByTestId('current-mode');

    fireEvent.click(setDarkBtn);
    expect(currentMode).toHaveTextContent('dark');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(true);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(true);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('dark');

    fireEvent.click(setLightBtn);
    expect(currentMode).toHaveTextContent('light');
    expect(document.documentElement.classList.contains('dark-theme')).toBe(false);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(false);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('light');
  });

  it('should not mutate document.documentElement when isStandalone is false', () => {
    localStorage.setItem(STORAGE_KEY, 'dark');

    render(
      <ColorModeProvider isStandalone={false}>
        <TestConsumer />
      </ColorModeProvider>,
    );

    // Context still tracks colorMode
    expect(screen.getByTestId('current-mode')).toHaveTextContent('dark');
    // But does NOT stamp classes on document.documentElement in embedded/federated mode
    expect(document.documentElement.classList.contains('dark-theme')).toBe(false);
    expect(document.documentElement.classList.contains('pf-v6-theme-dark')).toBe(false);
  });
});
