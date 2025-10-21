import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { vi } from 'vitest';
import ProgressPage from './ProgressPage';

// Mock the hooks and API calls
vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({
    user: {
      id: 1,
      username: 'testuser',
      ai_provider: 'openai',
      ai_model: 'gpt-4',
      ai_enabled: true,
    },
    isAuthenticated: true,
    isLoading: false,
    login: vi.fn(),
    loginWithUser: vi.fn(),
    logout: vi.fn(),
    updateSettings: vi.fn(),
    refreshUser: vi.fn(),
  }),
}));

// Mock the API hooks
vi.mock('../api/api', () => ({
  ...vi.importActual('../api/api'),
  useGetV1QuizProgress: () => ({
    data: {
      current_level: 'B1',
      total_questions: 100,
      correct_answers: 80,
      accuracy_rate: 0.8,
      performance_by_topic: {
        grammar: {
          total_attempts: 50,
          correct_attempts: 40,
          average_response_time_ms: 2000,
          last_updated: '2024-01-01',
        },
      },
      weak_areas: ['grammar'],
      recent_activity: [
        {
          question_id: 1,
          is_correct: true,
          created_at: '2024-01-01T10:00:00Z',
        },
        {
          question_id: 2,
          is_correct: false,
          created_at: '2024-01-01T11:00:00Z',
        },
      ],
    },
    isLoading: false,
  }),
  useGetV1QuizAiTokenUsageDaily: () => ({
    data: [
      {
        usage_date: '2024-01-01',
        total_tokens: 1000,
        total_requests: 10,
        provider: 'openai',
        model: 'gpt-4',
        usage_type: 'chat',
      },
    ],
    isLoading: false,
  }),
  useGetV1SettingsAiProviders: () => ({
    data: {
      providers: [
        { code: 'openai', name: 'OpenAI', usage_supported: true },
        { code: 'google', name: 'Google', usage_supported: false },
        { code: 'anthropic', name: 'Anthropic', usage_supported: true },
      ],
    },
    isLoading: false,
  }),
}));

describe('ProgressPage', () => {
  const createWrapper = () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });

    return ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <MantineProvider>{children}</MantineProvider>
        </BrowserRouter>
      </QueryClientProvider>
    );
  };

  it('renders without crashing', () => {
    const Wrapper = createWrapper();

    expect(() => {
      render(
        <Wrapper>
          <ProgressPage />
        </Wrapper>
      );
    }).not.toThrow();
  });

  it('displays progress information', () => {
    const Wrapper = createWrapper();

    render(
      <Wrapper>
        <ProgressPage />
      </Wrapper>
    );

    // Check that the main container exists
    expect(
      document.querySelector('.mantine-Container-root')
    ).toBeInTheDocument();
  });
});
