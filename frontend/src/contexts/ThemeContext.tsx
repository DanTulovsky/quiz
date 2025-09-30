// NOTE: Only export ThemeProvider and useTheme from this file. Do NOT export or re-export theme constants/types here. See theme.ts for theme constants.
import React, {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from 'react';
import { ThemeName, ColorScheme } from '../theme/theme';
import { themes, themeNames } from '../theme/theme';
import { ThemeContextType } from './ThemeContext.types';
import logger from '../utils/logger';

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

interface ThemeProviderProps {
  children: ReactNode;
}

export const ThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
  const [currentTheme, setCurrentTheme] = useState<ThemeName>('blue');
  const [colorScheme, setColorSchemeState] = useState<ColorScheme>('light');

  // Load theme and colorScheme from localStorage on mount
  useEffect(() => {
    const savedTheme = localStorage.getItem('quiz-theme') as ThemeName;

    if (savedTheme && themes[savedTheme]) {
      setCurrentTheme(savedTheme);
    } else {
      // Clear potentially corrupted theme data
      if (savedTheme) {
        localStorage.removeItem('quiz-theme');
      }
      setCurrentTheme('blue');
    }

    const savedScheme = localStorage.getItem(
      'quiz-color-scheme'
    ) as ColorScheme;
    if (savedScheme === 'light' || savedScheme === 'dark') {
      setColorSchemeState(savedScheme);
    } else {
      // Optionally, use system preference as default
      const prefersDark = window.matchMedia(
        '(prefers-color-scheme: dark)'
      ).matches;
      setColorSchemeState(prefersDark ? 'dark' : 'light');
    }
  }, []);

  // Save theme to localStorage when it changes
  const setTheme = (theme: ThemeName) => {
    if (!themes[theme]) {
      logger.error('Invalid theme:', theme);
      return;
    }
    setCurrentTheme(theme);
    localStorage.setItem('quiz-theme', theme);
  };

  // Save colorScheme to localStorage when it changes
  const setColorScheme = (scheme: ColorScheme) => {
    setColorSchemeState(scheme);
    localStorage.setItem('quiz-color-scheme', scheme);
  };

  const value: ThemeContextType = {
    currentTheme,
    setTheme,
    themeNames,
    themes,
    colorScheme,
    setColorScheme,
  };

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
};

export const useTheme = (): ThemeContextType => {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};
