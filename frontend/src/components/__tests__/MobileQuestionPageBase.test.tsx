import { describe, it, expect, beforeEach, vi } from 'vitest';
import { screen } from '@testing-library/react';
import MobileQuestionPageBase from '../MobileQuestionPageBase';
import { renderWithProviders } from '../../test-utils';

// Mock all dependencies
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com' },
  }),
}));

vi.mock('../../contexts/useQuestion', () => ({
  useQuestion: () => ({
    quizFeedback: null,
    setQuizFeedback: vi.fn(),
    readingFeedback: null,
    setReadingFeedback: vi.fn(),
  }),
}));

vi.mock('../../hooks/useQuestionUrlState', () => ({
  useQuestionUrlState: () => ({
    navigateToQuestion: vi.fn(),
  }),
}));

vi.mock('../../hooks/useQuestionFlow', () => ({
  useQuestionFlow: () => ({
    question: {
      id: 1,
      language: 'Italian',
      level: 'A1',
      type: 'qa',
      content: {
        question: 'Come stai?',
        options: ['Bene', 'Male', 'Così così', 'Benissimo'],
      },
    },
    isLoading: false,
    error: null,
    forceFetchNextQuestion: vi.fn(),
  }),
}));

vi.mock('../../hooks/useTTS', () => ({
  useTTS: () => ({
    prebufferTTS: vi.fn(),
    cancelPrebuffer: vi.fn(),
    isBuffering: false,
    bufferingProgress: 0,
  }),
}));

vi.mock('../../utils/tts', () => ({
  defaultVoiceForLanguage: vi.fn(() => 'it-IT'),
}));

vi.mock('../../api/api', () => ({
  postV1QuizAnswer: vi.fn(() =>
    Promise.resolve({
      is_correct: true,
      correct_answer_index: 0,
      explanation: 'Correct! "Bene" means "good/well"',
    })
  ),
}));

const renderComponent = (mode: 'quiz' | 'reading' | 'vocabulary') => {
  return renderWithProviders(<MobileQuestionPageBase mode={mode} />);
};

describe('MobileQuestionPageBase', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders without crashing', () => {
    renderComponent('quiz');
    expect(screen.getByText('Quiz')).toBeInTheDocument();
  });

  it('renders question text', () => {
    renderComponent('quiz');
    expect(screen.getByText('Come stai?')).toBeInTheDocument();
  });

  it('renders all answer options', () => {
    renderComponent('quiz');

    expect(screen.getByText('Bene')).toBeInTheDocument();
    expect(screen.getByText('Male')).toBeInTheDocument();
    expect(screen.getByText('Così così')).toBeInTheDocument();
    expect(screen.getByText('Benissimo')).toBeInTheDocument();
  });

  it('shows correct mode label for quiz', () => {
    renderComponent('quiz');
    expect(screen.getByText('Quiz')).toBeInTheDocument();
  });

  it('shows correct mode label for vocabulary', () => {
    renderComponent('vocabulary');
    expect(screen.getByText('Vocabulary')).toBeInTheDocument();
  });

  it('shows correct mode label for reading', () => {
    renderComponent('reading');
    expect(screen.getByText('Reading')).toBeInTheDocument();
  });

  it('shows language and level badges', () => {
    renderComponent('quiz');
    expect(screen.getByText('Italian - A1')).toBeInTheDocument();
  });

  it('shows submit button', () => {
    renderComponent('quiz');
    expect(
      screen.getByRole('button', { name: /Submit Answer/i })
    ).toBeInTheDocument();
  });

  it('disables submit button initially', () => {
    renderComponent('quiz');
    const submitButton = screen.getByRole('button', { name: /Submit Answer/i });
    expect(submitButton).toBeDisabled();
  });
});
