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

// Import the mocked API module to access the mock function
import * as apiModule from '../../api/api';

// Mock data with stable references
const mockAuthStatusData = {
  authenticated: true,
  user: { id: 1, role: 'user' },
};

const mockRefetch = vi.fn();

// Mock the dependencies
vi.mock('../../hooks/useAuth');

vi.mock('../../api/api', () => {
  // Create the mock function inside the mock
  const mockPostV1SnippetsFunction = vi.fn();

  return {
    // Keep all other mocks as they were
    useGetV1AuthStatus: () => ({
      data: mockAuthStatusData,
      isLoading: false,
      refetch: mockRefetch,
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
    useGetV1SettingsAiProviders: () => ({
      data: { providers: [] },
      isLoading: false,
    }),
    usePostV1SettingsTestAi: () => ({
      mutateAsync: vi.fn(),
      isPending: false,
    }),
    usePostV1SettingsTestEmail: () => ({
      mutateAsync: vi.fn(),
      isPending: false,
    }),
    useGetV1SettingsLevels: () => ({
      data: { levels: ['A1', 'A2', 'B1', 'B2', 'C1', 'C2'] },
      isLoading: false,
    }),
    useGetV1SettingsLanguages: () => ({
      data: ['en', 'es', 'fr', 'de', 'it'],
      isLoading: false,
    }),
    useGetV1SettingsApiKeyAvailability: () => ({
      data: { has_api_key: false },
      isLoading: false,
    }),
    usePostV1Snippets: () => ({
      mutateAsync: vi.fn(),
      isPending: false,
    }),
    useDeleteV1SnippetsId: () => ({
      mutateAsync: vi.fn(),
      isPending: false,
    }),
    usePutV1SnippetsId: () => ({
      mutateAsync: vi.fn(),
      isPending: false,
    }),
    useGetV1Snippets: () => ({
      data: { snippets: [], total: 0, limit: 20, offset: 0 },
      isLoading: false,
    }),
    postV1Snippets: mockPostV1SnippetsFunction,
  };
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
    translation: {
      translatedText: 'Translated text',
      sourceLanguage: 'en',
      targetLanguage: 'es',
    },
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

  it('should show save button when translation is available', () => {
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

    expect(screen.getByText('Save')).toBeInTheDocument();
  });

  it('should call save API when save button is clicked', async () => {
    const user = userEvent.setup();

    // Set up the mock for this test
    apiModule.postV1Snippets.mockResolvedValue({});

    render(
      <TestWrapper>
        <TranslationPopup
          selection={{
            text: 'Bonjour',
            x: 100,
            y: 100,
            width: 50,
            height: 20,
          }}
          onClose={mockOnClose}
        />
      </TestWrapper>
    );

    const saveButton = screen.getByText('Save');
    await user.click(saveButton);

    expect(apiModule.postV1Snippets).toHaveBeenCalledWith({
      original_text: 'Bonjour',
      translated_text: 'Translated text',
      source_language: 'en',
      target_language: 'es',
    });
  });

  it('should show loading state while saving', async () => {
    const user = userEvent.setup();

    // Set up the mock for this test
    apiModule.postV1Snippets.mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 100))
    );

    render(
      <TestWrapper>
        <TranslationPopup
          selection={{
            text: 'Bonjour',
            x: 100,
            y: 100,
            width: 50,
            height: 20,
          }}
          onClose={mockOnClose}
        />
      </TestWrapper>
    );

    const saveButton = screen.getByText('Save');
    await user.click(saveButton);

    expect(screen.getByText('Saving...')).toBeInTheDocument();
    expect(screen.getByTestId('loader')).toBeInTheDocument();
  });

  it('should show saved state after successful save', async () => {
    const user = userEvent.setup();

    // Set up the mock for this test
    apiModule.postV1Snippets.mockResolvedValue({});

    render(
      <TestWrapper>
        <TranslationPopup
          selection={{
            text: 'Bonjour',
            x: 100,
            y: 100,
            width: 50,
            height: 20,
          }}
          onClose={mockOnClose}
        />
      </TestWrapper>
    );

    const saveButton = screen.getByText('Save');
    await user.click(saveButton);

    // Wait for the save operation to complete and check that button shows "Saved!"
    await waitFor(() => {
      const savedButton = screen.getByRole('button', { name: /Saved!/ });
      expect(savedButton).toBeInTheDocument();
      expect(savedButton).toBeDisabled();
    });
  });

  it('should show error message when save fails', async () => {
    const user = userEvent.setup();

    // Set up the mock for this test
    apiModule.postV1Snippets.mockRejectedValue(new Error('Save failed'));

    render(
      <TestWrapper>
        <TranslationPopup
          selection={{
            text: 'Bonjour',
            x: 100,
            y: 100,
            width: 50,
            height: 20,
          }}
          onClose={mockOnClose}
        />
      </TestWrapper>
    );

    const saveButton = screen.getByText('Save');
    await user.click(saveButton);

    // Check that error message appears inline in the component
    await waitFor(() => {
      expect(screen.getByText('Save failed')).toBeInTheDocument();
    });

    // Save button should be enabled again after error
    expect(saveButton).not.toBeDisabled();
  });


  it('should show save button even when no translation', () => {
    // Mock translation context to return no translation
    vi.mocked(
      vi.importActual('../../contexts/TranslationContext')
    ).useTranslation = () => ({
      translateText: vi.fn().mockResolvedValue({
        translatedText: '',
        sourceLanguage: 'en',
        targetLanguage: 'es',
      }),
      translation: null,
      isLoading: false,
      error: null,
    });

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

    // Save button should be present even when there's no translation
    expect(screen.getByText('Save')).toBeInTheDocument();
  });
});
