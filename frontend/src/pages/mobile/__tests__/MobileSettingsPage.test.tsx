import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MantineProvider } from '@mantine/core';
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

vi.mock('../../../contexts/ThemeContext', () => ({
  useTheme: () => ({
    currentTheme: 'blue',
    setTheme: vi.fn(),
    themeNames: {
      blue: 'Blue',
      green: 'Green',
      red: 'Red',
    },
    colorScheme: 'light',
    setColorScheme: vi.fn(),
    fontSize: 'medium',
    setFontSize: vi.fn(),
  }),
}));

vi.mock('../../../api/api', () => ({
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
  }),
  useGetV1SettingsLevels: () => ({
    data: {
      levels: ['A1', 'A2', 'B1'],
      level_descriptions: { A1: 'Beginner', A2: 'Elementary', B1: 'Intermediate' },
    },
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
  }),
  usePostV1SettingsTestAi: () => ({
    mutateAsync: vi.fn(),
  }),
  usePutV1UserzProfile: () => ({
    mutateAsync: vi.fn(),
  }),
  usePostV1SettingsTestEmail: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useGetV1SettingsApiKeyProvider: () => ({
    data: { has_api_key: false },
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

const renderWithProviders = (children: React.ReactNode) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter
        future={{
          v7_startTransition: false,
          v7_relativeSplatPath: false,
        }}
      >
        <MantineProvider>{children}</MantineProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
};

describe('MobileSettingsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render the settings page', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Settings')).toBeInTheDocument();
    });
  });

  it('should render theme section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Theme')).toBeInTheDocument();
    });
  });

  it('should render account information section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Account Information')).toBeInTheDocument();
    });
  });

  it('should render learning preferences section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Learning Preferences')).toBeInTheDocument();
    });
  });

  it('should render notifications section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Notifications')).toBeInTheDocument();
    });
  });

  it('should render AI settings section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(() => {
      expect(screen.getByText('AI Settings')).toBeInTheDocument();
    });
  });

  it('should render data management section', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(() => {
      expect(screen.getByText('Data Management')).toBeInTheDocument();
    });
  });

  it('should render save button', async () => {
    renderWithProviders(<MobileSettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Save Changes/i })
      ).toBeInTheDocument();
    });
  });
});
