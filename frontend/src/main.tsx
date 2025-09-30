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

import { CacheProvider } from '@emotion/react';
import createCache from '@emotion/cache';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 5 * 60 * 1000, // 5 minutes
      refetchOnWindowFocus: false,
    },
  },
});

// Detect the nonce from the first <style nonce> tag in the document
const nonce =
  document.querySelector('style[nonce]')?.getAttribute('nonce') || undefined;
const cache = createCache({ key: 'mantine', nonce });

// Component that uses the theme context to set the Mantine theme
export const ThemedApp: React.FC = () => {
  const { currentTheme, themes, colorScheme } = useTheme();

  return (
    <MantineProvider
      theme={themes[currentTheme]}
      forceColorScheme={colorScheme}
    >
      <Notifications />
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
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
