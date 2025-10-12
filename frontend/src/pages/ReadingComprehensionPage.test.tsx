import { render, screen, act, waitFor } from '@testing-library/react';
import { vi, Mock } from 'vitest';
import ReadingComprehensionPage from './ReadingComprehensionPage';
import * as useAuthModule from '../hooks/useAuth';
import type { User, Question } from '../api/api';
import * as api from '../api/api';
import { MemoryRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { QuestionProvider } from '../contexts/QuestionContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../api/api', async (importOriginal: () => Promise<unknown>) => {
  const actual = (await importOriginal()) as typeof api;
  return {
    ...actual,
    getV1QuizQuestion: vi.fn(),
  };
});

function mockUseAuth(user: User) {
  vi.spyOn(useAuthModule, 'useAuth').mockReturnValue({
    user,
    isAuthenticated: !!user,
    isLoading: false,
    login: vi.fn(),
    logout: vi.fn(),
    updateSettings: vi.fn(),
    refreshUser: vi.fn(),
    loginWithUser: vi.fn(),
  });
}

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

describe('ReadingComprehensionPage - lifecycle safety', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('mounts without throwing and shows loading/generating UI for AI-enabled users', async () => {
    mockUseAuth({ username: 'test', ai_enabled: true });
    (api.getV1QuizQuestion as Mock).mockResolvedValue({
      status: 'generating',
      message: 'No questions available.',
    });

    let unmount: () => void;
    act(() => {
      const renderResult = render(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <ReadingComprehensionPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
      unmount = renderResult.unmount;
    });

    // Wait for the loading/generating UI to appear
    await waitFor(() => {
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    });

    // Clean up to prevent polling timers from causing warnings
    unmount();
  });

  it('unmounts cleanly without calling undefined cleanup (regression for stopPolling)', () => {
    mockUseAuth({ username: 'test', ai_enabled: true });
    const question: Question = {
      id: 1,
      type: 'reading_comprehension',
      content: {
        question: 'Q?',
        options: ['A', 'B', 'C', 'D'],
        passage: '...',
      },
    };
    (api.getV1QuizQuestion as Mock).mockResolvedValue(question);

    let unmount: () => void;
    act(() => {
      const renderResult = render(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <ReadingComprehensionPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
      unmount = renderResult.unmount;
    });

    expect(() => unmount()).not.toThrow();
  });
});
