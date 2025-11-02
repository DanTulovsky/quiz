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
import { initializeTTSRocks } from './utils/ttsRocksInit';

// Initialize TTS.Rocks when DOM is ready
if (typeof window !== 'undefined') {
  // Wait for TTS.Rocks script to load
  let initAttempts = 0;
  const maxAttempts = 100; // 10 seconds max wait
  const initTTS = () => {
    // Check if TTS is loaded and has the necessary properties
    if (window.TTS && typeof window.TTS.speak === 'function') {
      initializeTTSRocks();
      console.log(
        'TTS.Rocks initialized with endpoint:',
        window.TTS.openAISettings?.endpoint
      );
    } else if (window.TTS_LOADED || window.TTS) {
      // Script loaded but TTS might not be fully initialized yet, wait a bit more
      initAttempts++;
      if (initAttempts < maxAttempts) {
        setTimeout(initTTS, 100);
      } else {
        console.error(
          'TTS.Rocks script loaded but not fully initialized after',
          maxAttempts * 100,
          'ms'
        );
      }
    } else {
      // Script hasn't loaded yet
      initAttempts++;
      if (initAttempts < maxAttempts) {
        setTimeout(initTTS, 100);
      } else {
        console.error(
          'TTS.Rocks script failed to load after',
          maxAttempts * 100,
          'ms'
        );
      }
    }
  };

  // Wait for window load to ensure all scripts are loaded
  if (document.readyState === 'complete') {
    // Already loaded, initialize immediately
    setTimeout(initTTS, 100);
  } else {
    window.addEventListener('load', () => {
      setTimeout(initTTS, 100);
    });
    // Also try on DOMContentLoaded as fallback
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => {
        setTimeout(initTTS, 100);
      });
    }
  }
}

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
