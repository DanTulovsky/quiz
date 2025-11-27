import { describe, it, expect, beforeEach, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MobileQuestionPageBase from '../MobileQuestionPageBase';
import { renderWithProviders } from '../../test-utils';

const mockAuthStatusData = {
  authenticated: true,
  user: { id: 1, role: 'user' },
};

const mockRefetch = vi.fn();

// Mock all dependencies
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com' },
    isAuthenticated: true,
  }),
}));

vi.mock('../../contexts/useQuestion', () => ({
  useQuestion: () => ({
    quizQuestion: null,
    setQuizQuestion: vi.fn(),
    readingQuestion: null,
    setReadingQuestion: vi.fn(),
    quizFeedback: null,
    setQuizFeedback: vi.fn(),
    readingFeedback: null,
    setReadingFeedback: vi.fn(),
    selectedAnswer: null,
    setSelectedAnswer: vi.fn(),
    isSubmitted: false,
    setIsSubmitted: vi.fn(),
    showExplanation: false,
    setShowExplanation: vi.fn(),
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
    isLoading: false,
    isPlaying: false,
    isPaused: false,
    playTTS: vi.fn(),
    stopTTS: vi.fn(),
    pauseTTS: vi.fn(),
    resumeTTS: vi.fn(),
    restartTTS: vi.fn(),
  }),
}));

vi.mock('../../utils/tts', () => ({
  defaultVoiceForLanguage: vi.fn(() => 'it-IT'),
}));

// Mock SnippetHighlighter component
vi.mock('../SnippetHighlighter', () => ({
  SnippetHighlighter: ({
    text,
    component: Component,
    componentProps,
  }: {
    text: string;
    component?: React.ElementType;
    componentProps?: Record<string, unknown>;
  }) => {
    const ComponentToRender = Component || 'span';
    return <ComponentToRender {...componentProps}>{text}</ComponentToRender>;
  },
}));

vi.mock('../../api/api', () => ({
  postV1QuizAnswer: vi.fn(() =>
    Promise.resolve({
      is_correct: true,
      correct_answer_index: 0,
      explanation: 'Correct! "Bene" means "good/well"',
    })
  ),
  // Add mocks for tanstack-query mutation hooks used in component
  usePostV1QuizQuestionIdReport: () => ({ mutate: vi.fn(), isPending: false }),
  usePostV1QuizQuestionIdMarkKnown: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
  // Mock snippet hooks
  useGetV1SnippetsByQuestionQuestionId: () => ({
    data: { snippets: [] },
    isLoading: false,
    error: null,
  }),
  useGetV1PreferencesLearning: () => ({
    data: { tts_voice: 'it-IT-TestVoice' },
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

  // Mode and badge headers removed in mobile design

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

  it('resets adjust frequency selection when modal is reopened', async () => {
    const user = userEvent.setup();
    renderComponent('quiz');

    await user.click(screen.getByRole('button', { name: /adjust frequency/i }));
    await user.click(await screen.findByTestId('confidence-level-3'));
    const saveButton = await screen.findByRole('button', { name: /save/i });
    await waitFor(() => expect(saveButton).not.toBeDisabled());

    const closeButton = await screen.findByRole('button', { name: '' });
    await user.click(closeButton);

    await user.click(screen.getByRole('button', { name: /adjust frequency/i }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /save/i })).toBeDisabled()
    );
  });
});
