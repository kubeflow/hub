import * as React from 'react';

export type ThemeMode = 'light' | 'dark';

interface ThemeContextType {
  themeMode: ThemeMode;
  effectiveTheme: 'light' | 'dark';
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
}

const STORAGE_KEY = 'kubeflow_theme_mode';

const defaultThemeContext: ThemeContextType = {
  themeMode: 'light',
  effectiveTheme: 'light',
  setThemeMode: () => undefined,
  toggleTheme: () => undefined,
};

const ThemeContext = React.createContext<ThemeContextType | undefined>(undefined);

export const CustomThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [themeMode, setThemeModeState] = React.useState<ThemeMode>(() => {
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
    return 'light';
  });

  const effectiveTheme: 'light' | 'dark' = themeMode;

  React.useEffect(() => {
    const root = document.documentElement;
    if (effectiveTheme === 'dark') {
      root.classList.add('dark-theme');
      root.classList.add('pf-v6-theme-dark');
    } else {
      root.classList.remove('dark-theme');
      root.classList.remove('pf-v6-theme-dark');
    }
  }, [effectiveTheme]);

  const setThemeMode = React.useCallback((mode: ThemeMode) => {
    setThemeModeState(mode);
    try {
      localStorage.setItem(STORAGE_KEY, mode);
    } catch {
      // Ignore localStorage errors
    }
  }, []);

  const toggleTheme = React.useCallback(() => {
    const nextMode = effectiveTheme === 'light' ? 'dark' : 'light';
    setThemeMode(nextMode);
  }, [effectiveTheme, setThemeMode]);

  const contextValue = React.useMemo(
    () => ({ themeMode, effectiveTheme, setThemeMode, toggleTheme }),
    [themeMode, effectiveTheme, setThemeMode, toggleTheme],
  );

  return <ThemeContext.Provider value={contextValue}>{children}</ThemeContext.Provider>;
};

export const useCustomTheme = (): ThemeContextType => {
  const context = React.useContext(ThemeContext);
  return context || defaultThemeContext;
};
