import React from 'react';
import { screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderWithProviders } from '../../../test-utils';
import MobileSettingsPage from '../MobileSettingsPage';

// Mock the hooks and contexts
vi.mock('../../../hooks/useAuth', () => ({
  useAuth: () => ({
    user: {
      id: '1',
      username: 'testuser',
      email: 'test@example.com',
      preferred_language: 'spanish',
      current_level: 'A1',
      timezone: 'UTC',
      ai_provider: '',
      ai_model: '',
      ai_enabled: false,
    },
    refreshUser: vi.fn(),
  }),
}));

vi.mock('../../../contexts/ThemeContext', async importOriginal => {
  const actual = (await importOriginal()) as any;
  const mockTheme = { primaryColor: 'blue' };
  return {
    ...actual,
    useTheme: () => ({
      currentTheme: 'blue',
      setTheme: vi.fn(),
      themeNames: actual.themeNames || {
        blue: 'Blue',
        green: 'Green',
        red: 'Red',
      },
      themes: actual.themes || { blue: mockTheme, green: mockTheme, red: mockTheme },
      colorScheme: 'light',
      setColorScheme: vi.fn(),
      fontSize: 'medium',
      setFontSize: vi.fn(),
    }),
  };
});

vi.mock('../../../api/api', () => ({
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
  useGetV1SettingsAiProviders: () => ({
    isLoading: false,
    error: null,
    data: {
      providers: [
        {
          code: 'openai',
          name: 'OpenAI',
          url: 'https://api.openai.com',
          models: [{ code: 'gpt-4', name: 'GPT-4' }],
        },
      ],
    },
  }),
  useGetV1SettingsLanguages: () => ({
    data: [{ name: 'spanish' }, { name: 'french' }],
    isLoading: false,
    error: null,
  }),
  useGetV1SettingsLevels: () => ({
    data: {
      levels: ['A1', 'A2', 'B1'],
      level_descriptions: {
        A1: 'Beginner',
        A2: 'Elementary',
        B1: 'Intermediate',
      },
    },
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
  useGetV1PreferencesLearning: () => ({
    data: {
      focus_on_weak_areas: false,
      fresh_question_ratio: 0.5,
      known_question_penalty: 0.5,
      weak_area_boost: 2.0,
      review_interval_days: 7,
      daily_goal: 10,
      daily_reminder_enabled: false,
      tts_voice: '',
    },
    isLoading: false,
    error: null,
  }),
  usePutV1PreferencesLearning: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePostV1SettingsTestAi: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePutV1UserzProfile: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePostV1SettingsTestEmail: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useGetV1SettingsApiKeyProvider: () => ({
    data: { has_api_key: false },
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
  getGetV1PreferencesLearningQueryKey: () => ['learningPreferences'],
}));

vi.mock('../../../api/settingsApi', () => ({
  clearAllStories: vi.fn(),
  resetAccount: vi.fn(),
  clearAllAIChats: vi.fn(),
  clearAllSnippets: vi.fn(),
}));

describe.skip('MobileSettingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.clearAllTimers();
  });

  it('should render the settings page', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(
      () => {
        expect(screen.getByText('Settings')).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it('should render theme section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(
      () => {
        expect(screen.getByText('Theme')).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it('should render account information section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(
      () => {
        expect(screen.getByText('Account Information')).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it('should render learning preferences section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(
      () => {
        expect(screen.getByText('Learning Preferences')).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it('should render notifications section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(
      () => {
        expect(screen.getByText('Notifications')).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it('should render AI settings section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(
      () => {
        expect(screen.getByText('AI Settings')).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it('should render data management section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(
      () => {
        expect(screen.getByText('Data Management')).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });

  it('should render save button', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(
      () => {
        expect(
          screen.getByRole('button', { name: /Save Changes/i })
        ).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });
});
