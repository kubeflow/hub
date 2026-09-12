import * as React from 'react';

export type ThemeMode = 'light' | 'dark' | 'system';

interface ThemeContextType {
  themeMode: ThemeMode;
  effectiveTheme: 'light' | 'dark';
  setThemeMode: (mode: ThemeMode) => void;
  toggleTheme: () => void;
}

const STORAGE_KEY = 'kubeflow_theme_mode';

const ThemeContext = React.createContext<ThemeContextType | undefined>(undefined);

export const CustomThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [themeMode, setThemeModeState] = React.useState<ThemeMode>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'light' || saved === 'dark' || saved === 'system') {
      return saved as ThemeMode;
    }
    return 'system';
  });

  const [systemPrefersDark, setSystemPrefersDark] = React.useState<boolean>(() => {
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  });

  React.useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (e: MediaQueryListEvent) => {
      setSystemPrefersDark(e.matches);
    };

    if (mediaQuery && mediaQuery.addEventListener) {
      mediaQuery.addEventListener('change', handleChange);
      return () => {
        mediaQuery.removeEventListener('change', handleChange);
      };
    }
    return undefined;
  }, []);

  const effectiveTheme: 'light' | 'dark' =
    themeMode === 'system' ? (systemPrefersDark ? 'dark' : 'light') : themeMode;

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

  const setThemeMode = (mode: ThemeMode) => {
    setThemeModeState(mode);
    localStorage.setItem(STORAGE_KEY, mode);
  };

  const toggleTheme = () => {
    const nextMode = effectiveTheme === 'light' ? 'dark' : 'light';
    setThemeMode(nextMode);
  };

  return (
    <ThemeContext.Provider value={{ themeMode, effectiveTheme, setThemeMode, toggleTheme }}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useCustomTheme = (): ThemeContextType => {
  const context = React.useContext(ThemeContext);
  if (!context) {
    throw new Error('useCustomTheme must be used within a CustomThemeProvider');
  }
  return context;
};
