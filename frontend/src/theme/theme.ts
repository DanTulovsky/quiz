import { MantineThemeOverride } from '@mantine/core';

export type ColorScheme = 'light' | 'dark';

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

// Helper function to create a theme with validation
const createValidTheme = (primaryColor: string) => {
  if (
    !validPrimaryColors.includes(
      primaryColor as (typeof validPrimaryColors)[number]
    )
  ) {
    // Avoid noisy console in production; rely on dev logger
    return { ...baseTheme, primaryColor: 'blue' }; // fallback to blue
  }
  const theme = { ...baseTheme, primaryColor };
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

// Default theme
export const theme = themes.blue;

// Theme type
export type ThemeName = keyof typeof themes;
