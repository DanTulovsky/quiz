import React, { ReactNode, useMemo } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { ThemeProvider, useTheme } from './contexts/ThemeContext';
import { QuestionProvider } from './contexts/QuestionContext';
import { AuthProvider } from './contexts/AuthProvider';
import { TranslationProvider } from './contexts/TranslationContext';

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
}

interface ProvidersProps {
  children: ReactNode;
}

// Component that uses the theme context to set the Mantine theme
const ThemedProviders: React.FC<ProvidersProps> = ({ children }) => {
  const { currentTheme, themes } = useTheme();

  // ✅ Create QueryClient only once
  const queryClient = useMemo(() => createQueryClient(), []);

  return (
    <MantineProvider theme={themes[currentTheme]}>
      <Notifications />
      <QueryClientProvider client={queryClient}>
        <BrowserRouter
          future={{
            v7_startTransition: false,
            v7_relativeSplatPath: false,
          }}
        >
          <AuthProvider>
            <TranslationProvider>
              <QuestionProvider>{children}</QuestionProvider>
            </TranslationProvider>
          </AuthProvider>
        </BrowserRouter>
      </QueryClientProvider>
    </MantineProvider>
  );
};

export function AllProviders({ children }: ProvidersProps) {
  return (
    <ThemeProvider>
      <ThemedProviders>{children}</ThemedProviders>
    </ThemeProvider>
  );
}

export function renderWithProviders(
  ui: React.ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, { wrapper: AllProviders, ...options });
}
