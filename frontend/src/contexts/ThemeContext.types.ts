import { ThemeName, ColorScheme } from '../theme/theme';
import { themes, themeNames } from '../theme/theme';

export interface ThemeContextType {
  currentTheme: ThemeName;
  setTheme: (theme: ThemeName) => void;
  themeNames: typeof themeNames;
  themes: typeof themes;
  colorScheme: ColorScheme;
  setColorScheme: (scheme: ColorScheme) => void;
}
