import * as React from 'react';

export type ColorMode = 'light' | 'dark';

export interface ColorModeContextType {
  colorMode: ColorMode;
  setColorMode: (mode: ColorMode) => void;
  toggleColorMode: () => void;
}

export const STORAGE_KEY = 'kubeflow_theme_mode';

const ColorModeContext = React.createContext<ColorModeContextType | undefined>(undefined);

export interface ColorModeProviderProps {
  children: React.ReactNode;
  isStandalone?: boolean;
}

export const ColorModeProvider: React.FC<ColorModeProviderProps> = ({
  children,
  isStandalone = true,
}) => {
  const [colorMode, setColorModeState] = React.useState<ColorMode>(() => {
    if (typeof window === 'undefined') {
      return 'light';
    }
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved === 'dark' || saved === 'light') {
        return saved;
      }
    } catch {
      // Ignore localStorage errors in sandboxed environments
    }
    if (
      typeof window.matchMedia === 'function' &&
      window.matchMedia('(prefers-color-scheme: dark)').matches
    ) {
      return 'dark';
    }
    return 'light';
  });

  React.useEffect(() => {
    if (!isStandalone) {
      return;
    }
    const root = document.documentElement;
    if (colorMode === 'dark') {
      root.classList.add('dark-theme', 'pf-v6-theme-dark');
    } else {
      root.classList.remove('dark-theme', 'pf-v6-theme-dark');
    }
  }, [colorMode, isStandalone]);

  const setColorMode = React.useCallback(
    (mode: ColorMode) => {
      setColorModeState(mode);
      if (isStandalone) {
        try {
          localStorage.setItem(STORAGE_KEY, mode);
        } catch {
          // Ignore localStorage errors
        }
      }
    },
    [isStandalone],
  );

  const toggleColorMode = React.useCallback(() => {
    setColorMode(colorMode === 'light' ? 'dark' : 'light');
  }, [colorMode, setColorMode]);

  const contextValue = React.useMemo(
    () => ({ colorMode, setColorMode, toggleColorMode }),
    [colorMode, setColorMode, toggleColorMode],
  );

  return <ColorModeContext.Provider value={contextValue}>{children}</ColorModeContext.Provider>;
};

export const useColorMode = (): ColorModeContextType => {
  const context = React.useContext(ColorModeContext);
  if (!context) {
    throw new Error('useColorMode must be used within a ColorModeProvider');
  }
  return context;
};

// Aliases for backwards compatibility
export const CustomThemeProvider = ColorModeProvider;
export const useCustomTheme = useColorMode;
export type ThemeMode = ColorMode;
export type ThemeContextType = ColorModeContextType;
