import React from 'react';
import ReactDOM from 'react-dom/client';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { BrowserRouter } from 'react-router-dom';
import App from './App.tsx';
import './index.css';
import { AuthProvider } from './contexts/AuthProvider.tsx';
import { ThemeProvider, useTheme } from './contexts/ThemeContext.tsx';
import { QuestionProvider } from './contexts/QuestionContext';
import UpdatePrompt from './components/UpdatePrompt';
import { createThemeWithFontScale } from './theme/theme';

import { CacheProvider } from '@emotion/react';
import createCache from '@emotion/cache';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 5 * 60 * 1000, // 5 minutes
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: false,
    },
  },
});

// Add global error handler for unhandled promise rejections
if (typeof window !== 'undefined') {
  window.addEventListener('unhandledrejection', event => {
    // Only prevent default for axios errors to avoid console noise
    if (
      event.reason &&
      typeof event.reason === 'object' &&
      event.reason.isAxiosError
    ) {
      console.warn(
        'Axios error caught by global handler:',
        event.reason.message
      );
      event.preventDefault();
      return;
    }
    // Let other errors through for debugging
    console.error('Unhandled promise rejection:', event.reason);
  });
}

// Detect the nonce from the first <style nonce> tag in the document
const nonce =
  document.querySelector('style[nonce]')?.getAttribute('nonce') || undefined;
const cache = createCache({ key: 'mantine', nonce });

// Component that uses the theme context to set the Mantine theme
export const ThemedApp: React.FC = () => {
  const { currentTheme, colorScheme, fontSize } = useTheme();

  // Apply font scaling to the theme
  const scaledTheme = createThemeWithFontScale(currentTheme, fontSize);

  return (
    <MantineProvider theme={scaledTheme} forceColorScheme={colorScheme}>
      <Notifications />
      <QueryClientProvider client={queryClient}>
        <BrowserRouter
          future={{
            v7_startTransition: false,
            v7_relativeSplatPath: false,
          }}
        >
          <AuthProvider>
            {/* Remove global QuestionProvider here */}
            <App />
            <UpdatePrompt />
          </AuthProvider>
        </BrowserRouter>
      </QueryClientProvider>
    </MantineProvider>
  );
};

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <CacheProvider value={cache}>
      <ThemeProvider>
        <QuestionProvider>
          <ThemedApp />
        </QuestionProvider>
      </ThemeProvider>
    </CacheProvider>
  </React.StrictMode>
);
