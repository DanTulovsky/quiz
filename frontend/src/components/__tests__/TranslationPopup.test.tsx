import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import userEvent from '@testing-library/user-event';
import { TranslationPopup } from '../TranslationPopup';
import { AuthProvider } from '../../contexts/AuthProvider';
import { TranslationProvider } from '../../contexts/TranslationContext';
import { ThemeProvider } from '../../contexts/ThemeContext';

// Mock document and window for tests
Object.defineProperty(window, 'innerWidth', {
  writable: true,
  configurable: true,
  value: 1024,
});

Object.defineProperty(window, 'innerHeight', {
  writable: true,
  configurable: true,
  value: 768,
});

// Create a simple test wrapper
const TestWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });

  return (
    <MantineProvider>
      <Notifications />
      <QueryClientProvider client={queryClient}>
        <BrowserRouter
          future={{
            v7_startTransition: false,
            v7_relativeSplatPath: false,
          }}
        >
          <ThemeProvider>
            <AuthProvider>
              <TranslationProvider>{children}</TranslationProvider>
            </AuthProvider>
          </ThemeProvider>
        </BrowserRouter>
      </QueryClientProvider>
    </MantineProvider>
  );
};

// Mock the translation context
vi.mock('../../contexts/TranslationContext', () => ({
  TranslationProvider: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  useTranslation: () => ({
    translateText: vi.fn().mockResolvedValue({
      translatedText: 'Translated text',
      sourceLanguage: 'en',
      targetLanguage: 'es',
    }),
    isLoading: false,
    error: null,
  }),
}));

// Mock the text selection hook
vi.mock('../../hooks/useTextSelection', () => ({
  useTextSelection: () => ({
    selection: {
      text: 'Hello world',
      x: 100,
      y: 100,
      width: 50,
      height: 20,
    },
    isVisible: true,
    clearSelection: vi.fn(),
  }),
}));

// Mock the API module for AuthProvider
vi.mock('../../api/api', () => ({
  useGetV1AuthStatus: () => ({
    data: { authenticated: true, user: { id: 1, role: 'user' } },
    isLoading: false,
    refetch: vi.fn(),
  }),
  usePostV1AuthLogin: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePostV1AuthLogout: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePutV1Settings: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}));

// Mock the auth provider
vi.mock('../../contexts/AuthProvider', () => ({
  AuthProvider: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  useAuth: () => ({
    user: null,
    isLoading: false,
    login: vi.fn(),
    logout: vi.fn(),
    updateSettings: vi.fn(),
  }),
}));

// Mock ThemeProvider
vi.mock('../../contexts/ThemeContext', () => ({
  useTheme: () => ({
    currentTheme: 'blue',
    themes: {
      blue: {},
    },
    colorScheme: 'light',
    setTheme: vi.fn(),
  }),
  ThemeProvider: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

describe('TranslationPopup', () => {
  const mockOnClose = vi.fn();

  beforeEach(() => {
    mockOnClose.mockClear();
  });

  it('should render without crashing', () => {
    render(
      <TestWrapper>
        <TranslationPopup
          selection={{
            text: 'Hello world',
            x: 100,
            y: 100,
            width: 50,
            height: 20,
          }}
          onClose={mockOnClose}
        />
      </TestWrapper>
    );

    expect(screen.getByText('Translation')).toBeInTheDocument();
    expect(screen.getByText('"Hello world"')).toBeInTheDocument();
  });

  it('should show language dropdown', () => {
    render(
      <TestWrapper>
        <TranslationPopup
          selection={{
            text: 'Hello world',
            x: 100,
            y: 100,
            width: 50,
            height: 20,
          }}
          onClose={mockOnClose}
        />
      </TestWrapper>
    );

    const selectElement = screen.getByDisplayValue('English');
    expect(selectElement).toBeInTheDocument();
  });

  it('should handle dropdown interaction without closing popup', async () => {
    const user = userEvent.setup();

    render(
      <TestWrapper>
        <TranslationPopup
          selection={{
            text: 'Hello world',
            x: 100,
            y: 100,
            width: 50,
            height: 20,
          }}
          onClose={mockOnClose}
        />
      </TestWrapper>
    );

    const selectElement = screen.getByDisplayValue('English');

    // Click on the select to focus it
    await user.click(selectElement);

    // Wait a bit to see if popup closes during interaction
    await waitFor(
      () => {
        expect(mockOnClose).not.toHaveBeenCalled();
      },
      { timeout: 200 }
    );

    // Try to click on a dropdown option (this is what fails in real usage)
    const dropdownOption = screen.queryByText('Spanish');
    if (dropdownOption) {
      await user.click(dropdownOption);
    }

    // Wait again to see if popup closes after dropdown interaction
    await waitFor(
      () => {
        expect(mockOnClose).not.toHaveBeenCalled();
      },
      { timeout: 200 }
    );

    // Popup should still be visible
    expect(screen.getByText('Translation')).toBeInTheDocument();
  });

  it('should close popup when clicking outside', async () => {
    const user = userEvent.setup();

    render(
      <TestWrapper>
        <TranslationPopup
          selection={{
            text: 'Hello world',
            x: 100,
            y: 100,
            width: 50,
            height: 20,
          }}
          onClose={mockOnClose}
        />
      </TestWrapper>
    );

    // Click outside the popup (on the document body)
    await user.click(document.body);

    expect(mockOnClose).toHaveBeenCalled();
  });
});
