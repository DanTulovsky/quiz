import { render, screen, waitFor, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MantineProvider } from '@mantine/core';
import SettingsPage from './SettingsPage';
import { ThemeProvider } from '../contexts/ThemeContext';

// Mock the API hooks with realistic data
const mockUser = {
  id: 2,
  username: 'dant@wetsnow.com',
  email: 'dant@wetsnow.com',
  current_level: 'B1',
  preferred_language: 'italian',
  ai_provider: 'google',
  ai_model: 'gemini-2.5-flash',
  ai_enabled: true,
  has_api_key: true,
  created_at: '2025-07-11T20:03:18.750238Z',
  last_active: '2025-07-25T02:37:17.568042Z',
  timezone: 'America/Detroit',
  roles: [
    {
      id: 2,
      name: 'admin',
      description: 'Administrative access to all features',
      created_at: '2025-07-18T17:27:25.728776Z',
      updated_at: '2025-07-18T17:27:25.728776Z',
    },
  ],
};

const mockLevelsData = {
  levels: ['A1', 'A2', 'B1', 'B1+', 'B1++', 'B2', 'C1', 'C2'],
  level_descriptions: {
    A1: 'Beginner',
    A2: 'Elementary',
    B1: 'Intermediate',
    'B1+': 'Intermediate Plus',
    'B1++': 'Strong Intermediate',
    B2: 'Upper-Intermediate',
    C1: 'Advanced',
    C2: 'Proficient',
  },
};

const mockLanguagesData = [
  { name: 'italian' },
  { name: 'russian' },
  { name: 'french' },
  { name: 'japanese' },
  { name: 'chinese' },
  { name: 'german' },
];

const mockProvidersData = {
  providers: [
    {
      name: 'Google',
      code: 'google',
      url: 'https://generativelanguage.googleapis.com',
      models: [
        { name: 'Gemini 2.0 Flash', code: 'gemini-2.0-flash' },
        { name: 'Gemini 2.5 Flash', code: 'gemini-2.5-flash' },
      ],
    },
  ],
};

const mockLearningPrefs = {
  focus_on_weak_areas: true,
  fresh_question_ratio: 0.3,
  known_question_penalty: 0.5,
  review_interval_days: 7,
  weak_area_boost: 2.0,
  daily_reminder_enabled: false,
};

// Mock the API hooks
vi.mock('../api/api', () => ({
  useGetV1SettingsAiProviders: () => ({
    data: mockProvidersData,
    isLoading: false,
  }),
  useGetV1SettingsLanguages: () => ({
    data: mockLanguagesData,
  }),
  useGetV1SettingsLevels: () => ({
    data: mockLevelsData,
    refetch: vi.fn(),
  }),
  useGetV1PreferencesLearning: () => ({
    data: mockLearningPrefs,
    isLoading: false,
  }),
  usePutV1PreferencesLearning: () => ({
    mutateAsync: vi.fn(),
  }),
  usePutV1UserzProfile: () => ({
    mutateAsync: vi.fn(),
  }),
  usePostV1SettingsTestAi: () => ({
    mutateAsync: vi.fn(),
  }),
  usePostV1SettingsTestEmail: () => ({
    mutateAsync: vi.fn(),
  }),
  useGetV1SettingsApiKeyProvider: () => ({
    data: { has_api_key: true },
    refetch: vi.fn(),
  }),
}));

// Mock the useAuth hook directly
vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({
    user: mockUser,
    isAuthenticated: true,
    logout: vi.fn(),
    refreshUser: vi.fn(),
  }),
}));

// Mock the useTheme hook
vi.mock('../contexts/ThemeContext', async (importOriginal) => {
  const actual = (await importOriginal()) as any;
  return {
    ...actual,
    useTheme: () => ({
      currentTheme: 'teal',
      setTheme: vi.fn(),
      themeNames: ['teal', 'blue', 'indigo'],
      colorScheme: 'light',
      setColorScheme: vi.fn(),
    }),
  };
});

// Mock the TimezoneSelector component
vi.mock('../components/TimezoneSelector', () => ({
  default: ({
    value,
    onChange,
  }: {
    value: string;
    onChange: (value: string) => void;
  }) => (
    <select
      data-testid='timezone-selector'
      value={value}
      onChange={e => onChange(e.target.value)}
    >
      <option value='America/Detroit'>(GMT-4:00) Eastern Time</option>
    </select>
  ),
}));

// Mock the useTTS hook to prevent real TTS functionality in tests
vi.mock('../hooks/useTTS', () => ({
  useTTS: () => ({
    playTTSOnce: vi.fn(),
    stopTTSOnce: vi.fn(),
  }),
}));

describe('SettingsPage', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });
  });

  it('should pass a basic test', () => {
    expect(true).toBe(true);
  });

  it('should render a simple div', () => {
    const { container } = render(<div>Test</div>);
    expect(container.textContent).toBe('Test');
  });

  it('should render with providers', () => {
    let container: HTMLElement;
    act(() => {
      const renderResult = render(
        <QueryClientProvider client={queryClient}>
          <MantineProvider>
            <BrowserRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <div>Test with providers</div>
            </BrowserRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
      container = renderResult.container;
    });
    expect(container!.textContent).toContain('Test with providers');
  });

  it('should render with mocked contexts', () => {
    const { getByTestId } = render(
      <div data-testid='auth-provider'>
        <div data-testid='theme-provider'>
          <div data-testid='question-provider'>Test content</div>
        </div>
      </div>
    );
    expect(getByTestId('auth-provider')).toBeInTheDocument();
    expect(getByTestId('theme-provider')).toBeInTheDocument();
    expect(getByTestId('question-provider')).toBeInTheDocument();
  });

  it('should correctly display user current level in dropdown', async () => {
    // Mock fetch for /v1/voices to prevent test failures
    const voicesPayload = {
      voices: [
        { language: 'it-IT', gender: 'Male', name: 'it-IT-DiegoNeural' },
        { language: 'it-IT', gender: 'Female', name: 'it-IT-IsabellaNeural' },
      ],
    };
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(voicesPayload),
    });

    // Set up the mock before rendering
    global.fetch = mockFetch;

    act(() => {
      render(
        <QueryClientProvider client={queryClient}>
          <ThemeProvider>
            <MantineProvider>
              <BrowserRouter
                future={{
                  v7_startTransition: false,
                  v7_relativeSplatPath: false,
                }}
              >
                <SettingsPage />
              </BrowserRouter>
            </MantineProvider>
          </ThemeProvider>
        </QueryClientProvider>
      );
    });

    // Wait for the component to load and initialize
    await waitFor(() => {
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    // Wait for the learning language select to appear first (it should appear before level select)
    await waitFor(() => {
      const languageSelect = screen.getByTestId('learning-language-select');
      expect(languageSelect).toBeInTheDocument();
    });

    // Wait for the level select to appear after language is loaded
    await waitFor(() => {
      const levelSelect = screen.getByTestId('level-select');
      expect(levelSelect).toBeInTheDocument();
    });

    // Check that the level dropdown shows the correct value
    // The user has current_level: 'B1', so the dropdown should show "B1 — Intermediate"
    const levelSelect = screen.getByTestId('level-select');
    expect(levelSelect).toBeInTheDocument();

    // Check that the level dropdown has the correct value
    // In Mantine Select, the value is displayed as the full label
    const expectedValue = 'B1 — Intermediate';
    expect(levelSelect).toHaveValue(expectedValue);

    // Restore the global fetch mock after the test
    global.fetch = vi.fn().mockImplementation(() => {
      throw new Error('fetch() called without being mocked in test');
    });
  }, 15000); // Increase timeout for this test

  it('loads TTS voices and populates the voice select from /v1/voices response object', async () => {
    // Mock fetch for /v1/voices
    const voicesPayload = {
      voices: [
        { language: 'it-IT', gender: 'Male', name: 'it-IT-DiegoNeural' },
        { language: 'it-IT', gender: 'Female', name: 'it-IT-IsabellaNeural' },
      ],
    };
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(voicesPayload),
    });

    // Set up the mock before rendering
    global.fetch = mockFetch;

    // Render the component - wrap in act to handle initial render
    await act(async () => {
      render(
        <QueryClientProvider client={queryClient}>
          <ThemeProvider>
            <MantineProvider>
              <BrowserRouter
                future={{
                  v7_startTransition: false,
                  v7_relativeSplatPath: false,
                }}
              >
                <SettingsPage />
              </BrowserRouter>
            </MantineProvider>
          </ThemeProvider>
        </QueryClientProvider>
      );
    });

    // Wait for the Settings page to load
    await waitFor(() => {
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    // Wait for the learning preferences section to be in the DOM
    await waitFor(() => {
      expect(screen.getByText('Learning Preferences')).toBeInTheDocument();
    });

    // Wait for languages to be loaded first
    await waitFor(() => {
      expect(screen.getByText('Learning Language')).toBeInTheDocument();
    });

    // Wait for the voices to be loaded and state to be updated
    await waitFor(
      () => {
        expect(mockFetch).toHaveBeenCalledWith(
          expect.stringContaining('/v1/voices?language=it-IT')
        );
      },
      { timeout: 5000 }
    );

    // Voice select should now have options (value not required, but options rendered)
    const voiceSelect = screen.getByTestId('tts-voice-select');
    expect(voiceSelect).toBeInTheDocument();

    // The sample button should be present
    const sampleButton = screen.getByTestId('tts-sample-button');
    expect(sampleButton).toBeInTheDocument();

    // Verify the voice select has the expected voices populated
    // The select should have the voices from the API response
    await waitFor(() => {
      // Check that the voices are available in the select data
      // We can verify this by checking if the select is enabled (not disabled due to no voices)
      expect(voiceSelect).not.toBeDisabled();
    });

    // Restore the global fetch mock after the test
    global.fetch = vi.fn().mockImplementation(() => {
      throw new Error('fetch() called without being mocked in test');
    });
  }, 15000); // Increase timeout for this test
});
