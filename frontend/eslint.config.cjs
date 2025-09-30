module.exports = {
  ignores: [
    'dist',
    'node_modules',
    'tests/**',
    'playwright.config.ts',
    'vite.config.ts',
    'vitest.config.ts',
    'vite-env.d.ts',
    // generated API types — ignore to avoid linting generated code
    'src/api/**',
  ],
  languageOptions: {
    parser: require('@typescript-eslint/parser'),
    parserOptions: {
      project: './tsconfig.json',
      ecmaVersion: 2020,
      sourceType: 'module',
    },
  },
  plugins: {
    '@typescript-eslint': require('@typescript-eslint/eslint-plugin'),
    'react-refresh': require('eslint-plugin-react-refresh'),
    'react-hooks': require('eslint-plugin-react-hooks'),
  },
  rules: {
    '@typescript-eslint/no-unused-vars': 'error',
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/ban-ts-comment': 'warn',
  },
};


