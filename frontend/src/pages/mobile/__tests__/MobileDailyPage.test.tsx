import { describe, it, expect, beforeEach, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import MobileDailyPage from '../MobileDailyPage';
import { renderWithProviders } from '../../../test-utils';

const mockAuthStatusData = {
  authenticated: true,
  user: { id: 1, role: 'user' },
};

const mockRefetch = vi.fn();

// Mock auth hook
vi.mock('../../../hooks/useAuth', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com' },
    isAuthenticated: true,
  }),
}));

// Mock tanstack-query mutation hooks used in component
vi.mock('../../../api/api', () => ({
  usePostV1QuizQuestionIdReport: () => ({ mutate: vi.fn(), isPending: false }),
  usePostV1QuizQuestionIdMarkKnown: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  // Mock auth status for AuthProvider
  useGetV1AuthStatus: () => ({
    data: mockAuthStatusData, // ✅ Stable reference
    isLoading: false,
    refetch: mockRefetch, // ✅ Stable reference
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

const mockSubmitAnswer = vi.fn();
const mockGoToNextQuestion = vi.fn();
const mockGoToPreviousQuestion = vi.fn();

// Mock useDailyQuestions hook
vi.mock('../../../hooks/useDailyQuestions', () => ({
  useDailyQuestions: () => ({
    selectedDate: '2025-09-30',
    setSelectedDate: vi.fn(),
    currentQuestion: {
      id: 1,
      question_id: 101,
      user_id: 1,
      assignment_date: '2025-09-30',
      is_completed: false,
      question: {
        id: 101,
        language: 'Italian',
        level: 'A1',
        type: 'vocabulary',
        content: {
          question: 'Come stai?',
          sentence: 'Ciao, come stai oggi?',
          options: ['Bene', 'Male', 'Così così', 'Benissimo'],
        },
      },
    },
    submitAnswer: mockSubmitAnswer,
    goToNextQuestion: mockGoToNextQuestion,
    goToPreviousQuestion: mockGoToPreviousQuestion,
    hasNextQuestion: true,
    hasPreviousQuestion: false,
    isLoading: false,
    isSubmittingAnswer: false,
    currentQuestionIndex: 0,
    questions: [
      {
        id: 1,
        question_id: 101,
        user_id: 1,
        assignment_date: '2025-09-30',
        is_completed: false,
      },
      {
        id: 2,
        question_id: 102,
        user_id: 1,
        assignment_date: '2025-09-30',
        is_completed: false,
      },
    ],
  }),
}));

const renderComponent = () => {
  return renderWithProviders(<MobileDailyPage />);
};

describe('MobileDailyPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSubmitAnswer.mockResolvedValue({
      is_correct: true,
      correct_answer_index: 0,
      explanation: 'Great job!',
    });
  });

  it('renders without crashing', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Daily Challenge')).toBeInTheDocument();
    });
  });

  it('renders daily challenge header', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Daily Challenge')).toBeInTheDocument();
    });
  });

  it('shows question counter', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('1 of 2')).toBeInTheDocument();
    });
  });

  // Removed test for Daily Questions title which no longer exists

  it('renders current question', async () => {
    renderComponent();

    await waitFor(() => {
      // Should still have header badge but not duplicate within question card
      expect(screen.getAllByText('Italian - A1').length).toBe(1);
      expect(screen.getByText('Bene')).toBeInTheDocument();
    });
  });

  it('shows language and level badge', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getAllByText('Italian - A1').length).toBe(1);
    });
  });

  it('renders all answer options', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Bene')).toBeInTheDocument();
      expect(screen.getByText('Male')).toBeInTheDocument();
      expect(screen.getByText('Così così')).toBeInTheDocument();
      expect(screen.getByText('Benissimo')).toBeInTheDocument();
    });
  });

  it('shows submit button', async () => {
    renderComponent();

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Submit Answer/i })
      ).toBeInTheDocument();
    });
  });

  it('shows submit and next navigation', async () => {
    renderComponent();

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Submit Answer/i })
      ).toBeInTheDocument();
    });
  });

  it('shows next button after submitting answer', async () => {
    renderComponent();

    // Wait for component to render
    await waitFor(() => {
      expect(screen.getByText('Daily Challenge')).toBeInTheDocument();
    });

    // First select an answer option (look for Italian answer options)
    const answerOptions = screen.getAllByRole('button');
    const answerButton = answerOptions.find(
      button =>
        button.textContent &&
        ['Bene', 'Male', 'Così così', 'Benissimo'].includes(
          button.textContent.trim()
        )
    );

    if (answerButton) {
      fireEvent.click(answerButton);
    }

    // Now submit the answer
    const submitButton = screen.getByRole('button', { name: /Submit Answer/i });
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Next Question/i })
      ).toBeInTheDocument();
    });
  });
});
