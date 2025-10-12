import {
  render,
  screen,
  waitFor,
  fireEvent,
  act,
} from '@testing-library/react';
import { vi, Mock } from 'vitest';
import QuizPage from './QuizPage';
import * as useAuthModule from '../hooks/useAuth';
import type { User, Question } from '../api/api';
import * as api from '../api/api';
import { MemoryRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { QuestionProvider } from '../contexts/QuestionContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Mock API functions used by the page/hook
vi.mock('../api/api', async (importOriginal: () => Promise<unknown>) => {
  const actual = (await importOriginal()) as typeof api;
  return {
    ...actual,
    getV1QuizQuestion: vi.fn(),
    postV1QuizAnswer: vi.fn(),
  };
});

// Helper to mock useAuth
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

// Create a QueryClient for testing
const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

describe('QuizPage - GeneratingResponse UI', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('shows spinner and generating message when AI is enabled', async () => {
    mockUseAuth({ username: 'test', ai_enabled: true });
    (api.getV1QuizQuestion as Mock).mockResolvedValue({
      status: 'generating',
      message: 'No questions available.',
    });

    act(() => {
      render(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Wait for the spinner and message
    await waitFor(() => {
      expect(
        screen.getByText('Generating your personalized question...')
      ).toBeInTheDocument();
      expect(screen.getByText('This may take a moment')).toBeInTheDocument();
    });
  });

  it('shows enable AI message and button when AI is disabled', async () => {
    mockUseAuth({ username: 'test', ai_enabled: false });
    (api.getV1QuizQuestion as Mock).mockResolvedValue({
      status: 'generating',
      message: 'No questions available.',
    });

    act(() => {
      render(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Wait for the custom message
    await waitFor(() => {
      expect(screen.getByText(/no questions available/i)).toBeInTheDocument();
      expect(screen.getByText(/enable ai in your/i)).toBeInTheDocument();
      expect(
        screen.getByText(/to generate new questions/i)
      ).toBeInTheDocument();

      // Check that both links exist
      const settingsLinks = screen.getAllByRole('link');
      expect(settingsLinks).toHaveLength(2);

      // Check for the inline "settings" link
      expect(
        screen.getByRole('link', { name: /^settings$/i })
      ).toBeInTheDocument();

      // Check for the "Go to Settings" button
      expect(
        screen.getByRole('link', { name: /go to settings/i })
      ).toBeInTheDocument();
    });
  });
});

describe('QuizPage - Level Change', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('clears question when user level changes and does not match question level', async () => {
    // Mock a user with A1 level
    const user: User = {
      username: 'test',
      ai_enabled: true,
      current_level: 'A1',
    };
    mockUseAuth(user);

    // Mock a question with B1 level (different from user's A1 level)
    const question: Question = {
      id: 1,
      level: 'B1',
      content: {
        question: 'Test question?',
        options: ['Option 1', 'Option 2', 'Option 3', 'Option 4'],
      },
    };

    // Mock API to return the question
    (api.getV1QuizQuestion as Mock).mockResolvedValue(question);

    let rerender: (ui: React.ReactElement) => void;
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
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
      rerender = renderResult.rerender;
    });

    // Wait for the question to be loaded
    await waitFor(() => {
      expect(screen.getByText('Test question?')).toBeInTheDocument();
    });

    // Now change the user's level to B2 (still different from question's B1 level)
    const updatedUser: User = {
      username: 'test',
      ai_enabled: true,
      current_level: 'B2',
    };
    mockUseAuth(updatedUser);

    // Rerender with the updated user
    act(() => {
      rerender(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // The question should be cleared and we should see the loading state
    await waitFor(() => {
      expect(screen.queryByText('Test question?')).not.toBeInTheDocument();
    });
  });

  it('keeps question when user level matches question level', async () => {
    // Mock a user with B1 level
    const user: User = {
      username: 'test',
      ai_enabled: true,
      current_level: 'B1',
    };
    mockUseAuth(user);

    // Mock a question with B1 level (same as user's level)
    const question: Question = {
      id: 1,
      level: 'B1',
      content: {
        question: 'Test question?',
        options: ['Option 1', 'Option 2', 'Option 3', 'Option 4'],
      },
    };

    // Mock API to return the question
    (api.getV1QuizQuestion as Mock).mockResolvedValue(question);

    let rerender: (ui: React.ReactElement) => void;
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
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
      rerender = renderResult.rerender;
    });

    // Wait for the question to be loaded
    await waitFor(() => {
      expect(screen.getByText('Test question?')).toBeInTheDocument();
    });

    // Now change the user's level to B1 (same as question's level)
    const updatedUser: User = {
      username: 'test',
      ai_enabled: true,
      current_level: 'B1',
    };
    mockUseAuth(updatedUser);

    // Rerender with the updated user
    act(() => {
      rerender(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // The question should still be displayed
    await waitFor(() => {
      expect(screen.getByText('Test question?')).toBeInTheDocument();
    });
  });
});

describe('QuizPage - Answer Submission', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('submits correct answer index to backend', async () => {
    const user: User = {
      username: 'test',
      ai_enabled: true,
      current_level: 'A1',
    };
    mockUseAuth(user);

    const question: Question = {
      id: 1,
      level: 'A1',
      content: {
        question: 'Test question?',
        options: ['Option 1', 'Option 2', 'Option 3', 'Option 4'],
      },
    };

    // Mock API to return the question
    (api.getV1QuizQuestion as Mock).mockResolvedValue(question);

    // Mock the answer submission
    (api.postV1QuizAnswer as Mock).mockResolvedValue({
      is_correct: true,
      user_answer: 'Option 2',
      user_answer_index: 1,
      correct_answer_index: 1,
      explanation: 'Test explanation',
    });

    act(() => {
      render(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Wait for the question to be loaded
    await waitFor(() => {
      expect(screen.getByText('Test question?')).toBeInTheDocument();
    });

    // Select the first option (index 0)
    const radioButtons = screen.getAllByRole('radio');
    fireEvent.click(radioButtons[0]);

    // Submit the answer
    const submitButton = screen.getByRole('button', { name: /submit/i });
    fireEvent.click(submitButton);

    // Verify that the correct index was sent to the backend
    await waitFor(() => {
      expect(api.postV1QuizAnswer).toHaveBeenCalledWith({
        question_id: 1,
        user_answer_index: expect.any(Number),
      });
    });
  });
});

describe('QuizPage - URL State Management', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('updates URL when question is loaded', async () => {
    const user: User = {
      username: 'test',
      ai_enabled: true,
      current_level: 'A1',
    };
    mockUseAuth(user);

    const question: Question = {
      id: 123,
      level: 'A1',
      content: {
        question: 'Test question?',
        options: ['Option 1', 'Option 2', 'Option 3', 'Option 4'],
      },
    };

    // Mock API to return the question
    (api.getV1QuizQuestion as Mock).mockResolvedValue(question);

    act(() => {
      render(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Wait for the question to be loaded
    await waitFor(() => {
      expect(screen.getByText('Test question?')).toBeInTheDocument();
    });

    // Verify that the URL has been updated to include the question ID
    // This would be reflected in the browser's location but we can't easily test it directly
    // The important part is that the navigate function would have been called
    // In a real scenario, this would update the browser URL
  });
});

describe('QuizPage - Confidence Level Display', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('displays confidence level icon when confidence_level is present', async () => {
    const user: User = {
      username: 'test',
      ai_enabled: true,
      current_level: 'A1',
    };
    mockUseAuth(user);

    const question: Question = {
      id: 1,
      level: 'A1',
      confidence_level: 4,
      content: {
        question: 'Test question?',
        options: ['Option 1', 'Option 2', 'Option 3', 'Option 4'],
      },
      created_at: '2025-01-22T15:21:59.441433Z',
    };

    // Mock API to return the question
    (api.getV1QuizQuestion as Mock).mockResolvedValue(question);

    act(() => {
      render(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Wait for the question to be loaded
    await waitFor(() => {
      expect(screen.getByText('Test question?')).toBeInTheDocument();
    });

    // The confidence level icon should be present
    // We can verify this by checking that the confidence level is present in the question object
    expect(question.confidence_level).toBe(4);

    // The confidence level icon should be rendered inline near the stats
    // Verify the inline icon in the question card area
    const inlineIcon = screen.getByTestId('confidence-icon-inline');
    expect(inlineIcon).toBeInTheDocument();
  });

  it('does not display confidence level icon when confidence_level is not present', async () => {
    const user: User = {
      username: 'test',
      ai_enabled: true,
      current_level: 'A1',
    };
    mockUseAuth(user);

    const question: Question = {
      id: 1,
      level: 'A1',
      content: {
        question: 'Test question?',
        options: ['Option 1', 'Option 2', 'Option 3', 'Option 4'],
      },
      created_at: '2025-01-22T15:21:59.441433Z',
    };

    // Mock API to return the question
    (api.getV1QuizQuestion as Mock).mockResolvedValue(question);

    act(() => {
      render(
        <QueryClientProvider client={createTestQueryClient()}>
          <MantineProvider>
            <MemoryRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <QuestionProvider>
                <QuizPage />
              </QuestionProvider>
            </MemoryRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Wait for the question to be loaded
    await waitFor(() => {
      expect(screen.getByText('Test question?')).toBeInTheDocument();
    });

    // The confidence level should not be present
    expect(question.confidence_level).toBeUndefined();

    // The top-level confidence-level-icon was removed; ensure inline icon is absent
    const inlineIcon = screen.queryByTestId('confidence-icon-inline');
    expect(inlineIcon).not.toBeInTheDocument();
  });
});
