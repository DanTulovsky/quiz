import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { vi, Mock } from 'vitest';
import QuizPage from '../pages/QuizPage';
import * as useAuthModule from '../hooks/useAuth';
import * as api from '../api/api';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { QuestionProvider } from '../contexts/QuestionContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { User, Question } from '../api/api';

vi.mock('../api/api', async (importOriginal: () => Promise<unknown>) => {
  const actual = (await importOriginal()) as typeof api;
  return {
    ...actual,
    getV1QuizQuestion: vi.fn(),
    getV1QuizQuestionId: vi.fn(),
  };
});

function mockUseAuth(user: User | null) {
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
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

describe('Question + URL integration', () => {
  afterEach(() => vi.clearAllMocks());

  it('loads specific question when visiting /quiz/:id and does not fetch next', async () => {
    mockUseAuth({ username: 'test', ai_enabled: true, current_level: 'A1' });

    const question: Question = {
      id: 34,
      level: 'A1',
      content: { question: 'Question 34?', options: ['A', 'B'] },
    };

    (api.getV1QuizQuestionId as Mock).mockResolvedValue(question);
    (api.getV1QuizQuestion as Mock).mockResolvedValue({});

    render(
      <QueryClientProvider client={createTestQueryClient()}>
        <MantineProvider>
          <MemoryRouter initialEntries={['/quiz/34']}>
            <Routes>
              <Route
                path='/quiz/:questionId'
                element={
                  <QuestionProvider>
                    <QuizPage />
                  </QuestionProvider>
                }
              />
            </Routes>
          </MemoryRouter>
        </MantineProvider>
      </QueryClientProvider>
    );

    await waitFor(() =>
      expect(screen.getByText('Question 34?')).toBeInTheDocument()
    );

    expect(api.getV1QuizQuestionId).toHaveBeenCalledWith(34);
    expect(api.getV1QuizQuestion).not.toHaveBeenCalled();
  });

  it('fetches next question when visiting /quiz and displays it', async () => {
    mockUseAuth({ username: 'test', ai_enabled: true, current_level: 'A1' });

    const nextQuestion: Question = {
      id: 7,
      level: 'A1',
      content: { question: 'Question 7?', options: ['X', 'Y'] },
    };

    (api.getV1QuizQuestion as Mock).mockResolvedValue(nextQuestion);
    (api.getV1QuizQuestionId as Mock).mockResolvedValue({});

    render(
      <QueryClientProvider client={createTestQueryClient()}>
        <MantineProvider>
          <MemoryRouter initialEntries={['/quiz']}>
            <Routes>
              <Route
                path='/quiz'
                element={
                  <QuestionProvider>
                    <QuizPage />
                  </QuestionProvider>
                }
              />
            </Routes>
          </MemoryRouter>
        </MantineProvider>
      </QueryClientProvider>
    );

    await waitFor(() =>
      expect(screen.getByText('Question 7?')).toBeInTheDocument()
    );

    expect(api.getV1QuizQuestion).toHaveBeenCalled();
  });
});
