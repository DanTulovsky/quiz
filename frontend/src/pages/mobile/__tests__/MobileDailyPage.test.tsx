import { describe, it, expect, beforeEach, vi } from 'vitest';
import { screen } from '@testing-library/react';
import MobileDailyPage from '../MobileDailyPage';
import { renderWithProviders } from '../../../test-utils';

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
        type: 'qa',
        content: {
          question: 'Come stai?',
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

  it('renders without crashing', () => {
    renderComponent();
    expect(screen.getByText('Daily Challenge')).toBeInTheDocument();
  });

  it('renders daily challenge header', () => {
    renderComponent();
    expect(screen.getByText('Daily Challenge')).toBeInTheDocument();
  });

  it('shows question counter', () => {
    renderComponent();
    expect(screen.getByText('1 of 2')).toBeInTheDocument();
  });

  it('shows Daily Questions title', () => {
    renderComponent();
    expect(screen.getByText('Daily Questions')).toBeInTheDocument();
  });

  it('renders current question', () => {
    renderComponent();
    expect(screen.getByText('Come stai?')).toBeInTheDocument();
  });

  it('shows language and level badge', () => {
    renderComponent();
    expect(screen.getByText('Italian - A1')).toBeInTheDocument();
  });

  it('renders all answer options', () => {
    renderComponent();

    expect(screen.getByText('Bene')).toBeInTheDocument();
    expect(screen.getByText('Male')).toBeInTheDocument();
    expect(screen.getByText('Così così')).toBeInTheDocument();
    expect(screen.getByText('Benissimo')).toBeInTheDocument();
  });

  it('shows submit button', () => {
    renderComponent();
    expect(
      screen.getByRole('button', { name: /Submit Answer/i })
    ).toBeInTheDocument();
  });

  it('shows previous and next navigation', () => {
    renderComponent();
    expect(
      screen.getByRole('button', { name: /Previous/i })
    ).toBeInTheDocument();
  });

  it('disables Previous button when no previous question', () => {
    renderComponent();
    const previousButton = screen.getByRole('button', { name: /Previous/i });
    expect(previousButton).toBeDisabled();
  });
});
