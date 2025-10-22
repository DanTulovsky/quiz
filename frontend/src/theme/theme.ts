import { MantineThemeOverride } from '@mantine/core';

export type ColorScheme = 'light' | 'dark';
export type FontSize = 'small' | 'medium' | 'large' | 'extra-large';

// Font scale multipliers for each size option
export const fontScaleMap: Record<FontSize, number> = {
  small: 0.875,
  medium: 1.0,
  large: 1.125,
  'extra-large': 1.25,
};

// Valid Mantine primary colors
const validPrimaryColors = [
  'blue',
  'indigo',
  'pink',
  'red',
  'orange',
  'yellow',
  'lime',
  'green',
  'teal',
  'cyan',
] as const;

// Base theme configuration that all themes will extend
const baseTheme: MantineThemeOverride = {
  fontFamily: 'Inter, system-ui, sans-serif',
  headings: {
    fontFamily: 'Inter, system-ui, sans-serif',
  },
  defaultRadius: 'md',
  components: {
    Button: {
      defaultProps: {
        size: 'md',
      },
    },
    Card: {
      defaultProps: {
        shadow: 'sm',
        radius: 'md',
        withBorder: true,
      },
    },
    Paper: {
      defaultProps: {
        shadow: 'sm',
        radius: 'md',
      },
    },
    TextInput: {
      defaultProps: {
        size: 'md',
      },
    },
    PasswordInput: {
      defaultProps: {
        size: 'md',
      },
    },
    Select: {
      defaultProps: {
        size: 'md',
      },
    },
    Textarea: {
      defaultProps: {
        size: 'md',
      },
    },
    Badge: {
      defaultProps: {
        radius: 'sm',
      },
    },
    ActionIcon: {
      defaultProps: {
        variant: 'subtle',
      },
    },
    NavLink: {
      defaultProps: {
        variant: 'filled',
      },
    },
  },
};

// Helper function to create a theme with validation and optional font scaling
const createValidTheme = (primaryColor: string, fontScale: number = 1.0) => {
  if (
    !validPrimaryColors.includes(
      primaryColor as (typeof validPrimaryColors)[number]
    )
  ) {
    // Avoid noisy console in production; rely on dev logger
    return { ...baseTheme, primaryColor: 'blue' }; // fallback to blue
  }

  // Apply font scaling if different from default
  const theme = { ...baseTheme, primaryColor };
  if (fontScale !== 1.0) {
    // Scale the base font sizes
    theme.fontSizes = {
      xs: `${0.75 * fontScale}rem`,
      sm: `${0.875 * fontScale}rem`,
      md: `${1 * fontScale}rem`,
      lg: `${1.125 * fontScale}rem`,
      xl: `${1.25 * fontScale}rem`,
    };

    // Scale heading sizes
    theme.headings = {
      ...theme.headings,
      sizes: {
        h1: { fontSize: `${2.125 * fontScale}rem` },
        h2: { fontSize: `${1.625 * fontScale}rem` },
        h3: { fontSize: `${1.375 * fontScale}rem` },
        h4: { fontSize: `${1.125 * fontScale}rem` },
        h5: { fontSize: `${1 * fontScale}rem` },
        h6: { fontSize: `${0.875 * fontScale}rem` },
      },
    };
  }

  return theme;
};

// Available themes with different primary colors
export const themes = {
  blue: createValidTheme('blue'),
  indigo: createValidTheme('indigo'),
  pink: createValidTheme('pink'),
  red: createValidTheme('red'),
  orange: createValidTheme('orange'),
  yellow: createValidTheme('yellow'),
  lime: createValidTheme('lime'),
  green: createValidTheme('green'),
  teal: createValidTheme('teal'),
  cyan: createValidTheme('cyan'),
};

// Theme names for display
export const themeNames = {
  blue: 'Blue',
  indigo: 'Indigo',
  pink: 'Pink',
  red: 'Red',
  orange: 'Orange',
  yellow: 'Yellow',
  lime: 'Lime',
  green: 'Green',
  teal: 'Teal',
  cyan: 'Cyan',
};

// Helper function to create a theme with font scaling applied
export const createThemeWithFontScale = (
  themeName: ThemeName,
  fontSize: FontSize
): MantineThemeOverride => {
  const fontScale = fontScaleMap[fontSize];
  return createValidTheme(themeName, fontScale);
};

// Default theme
export const theme = themes.blue;

// Theme type
export type ThemeName = keyof typeof themes;
