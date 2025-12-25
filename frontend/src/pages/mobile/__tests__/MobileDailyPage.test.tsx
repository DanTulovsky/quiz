import React from 'react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import MobileDailyPage from '../MobileDailyPage';
import { renderWithProviders } from '../../../test-utils';

type MockSnippetProps = {
  component?: React.ElementType;
  componentProps?: Record<string, unknown>;
  text: string;
  [key: string]: unknown;
};

const snippetMock = vi.fn<(props: MockSnippetProps) => void>();
const mockMarkKnownMutate = vi.fn();

vi.mock('../../../components/SnippetHighlighter', () => ({
  __esModule: true,
  SnippetHighlighter: (props: MockSnippetProps) => {
    snippetMock(props);
    const { component: Component, componentProps = {}, text } = props;
    if (Component) {
      return React.createElement(Component, componentProps, text);
    }
    return React.createElement('span', componentProps, text);
  },
}));

// Mock react-router-dom to provide stable useParams
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ date: undefined }),
  };
});

// Mock Mantine's DatePickerInput to avoid potential rendering issues
vi.mock('@mantine/dates', async () => {
  const actual = await vi.importActual('@mantine/dates');
  return {
    ...actual,
    DatePickerInput: ({
      onChange,
      value,
    }: {
      onChange?: (date: Date | null) => void;
      value?: Date | null;
    }) => (
      <input
        data-testid='date-picker-input'
        type='text'
        value={value ? value.toISOString() : ''}
        onChange={e => onChange && onChange(new Date(e.target.value))}
      />
    ),
  };
});

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
    mutate: mockMarkKnownMutate,
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

const mockSubmitAnswer = vi.fn();
const mockGoToNextQuestion = vi.fn();
const mockGoToPreviousQuestion = vi.fn();
const mockSetSelectedDate = vi.fn();

// Create stable question object to prevent infinite loops in useEffect
const stableCurrentQuestion = {
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
};

const mockDailyQuestionsState = {
  selectedDate: '2025-09-30',
  setSelectedDate: mockSetSelectedDate,
  currentQuestion: stableCurrentQuestion,
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
  availableDates: ['2025-09-30', '2025-09-29'],
  isAllCompleted: false,
};

// Mock useDailyQuestions hook
vi.mock('../../../hooks/useDailyQuestions', () => ({
  useDailyQuestions: () => mockDailyQuestionsState,
}));

const renderComponent = () => {
  return renderWithProviders(<MobileDailyPage />);
};

describe('MobileDailyPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    snippetMock.mockClear();
    mockMarkKnownMutate.mockReset();
    mockDailyQuestionsState.currentQuestion = stableCurrentQuestion;
    mockDailyQuestionsState.questions = [
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
    ];
    mockSubmitAnswer.mockResolvedValue({
      is_correct: true,
      correct_answer_index: 0,
      explanation: 'Great job!',
    });
    mockDailyQuestionsState.isAllCompleted = false;
    mockDailyQuestionsState.hasPreviousQuestion = false;
    mockDailyQuestionsState.hasNextQuestion = true;
  });

  it('resets adjust frequency modal state when a new question loads', async () => {
    mockMarkKnownMutate.mockImplementation(() => {
      // Do not resolve to simulate a stuck request
    });

    const { rerender } = renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Daily Challenge')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('mark-known-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('confidence-level-3')).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId('confidence-level-3'));

    const saveButton = screen.getByTestId('submit-mark-known');
    fireEvent.click(saveButton);
    expect(saveButton).toBeDisabled();

    mockDailyQuestionsState.currentQuestion = {
      ...stableCurrentQuestion,
      id: 2,
      question_id: 202,
    };

    rerender(<MobileDailyPage />);

    await waitFor(() => {
      expect(screen.queryByTestId('submit-mark-known')).not.toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('mark-known-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('submit-mark-known')).toBeInTheDocument();
    });
    const reopenedSaveButton = screen.getByTestId('submit-mark-known');
    expect(reopenedSaveButton).toBeDisabled();
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
      expect(screen.getByText('1 OF 2')).toBeInTheDocument();
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

  it('renders top navigation when all questions are completed', async () => {
    mockDailyQuestionsState.isAllCompleted = true;
    mockDailyQuestionsState.hasPreviousQuestion = true;
    mockDailyQuestionsState.hasNextQuestion = true;

    renderComponent();

    await waitFor(() => {
      expect(
        screen.getByTestId('mobile-daily-top-navigation')
      ).toBeInTheDocument();
    });
  });

  it('hides top navigation while questions remain incomplete', async () => {
    mockDailyQuestionsState.isAllCompleted = false;

    renderComponent();

    await waitFor(() => {
      expect(
        screen.queryByTestId('mobile-daily-top-navigation')
      ).not.toBeInTheDocument();
    });
  });

  it('highlights reading comprehension passages with SnippetHighlighter', async () => {
    const readingQuestion = {
      id: 3,
      question_id: 303,
      user_id: 1,
      assignment_date: '2025-09-30',
      is_completed: false,
      question: {
        id: 303,
        language: 'Italian',
        level: 'A2',
        type: 'reading_comprehension' as const,
        content: {
          question: 'Qual è il tema principale del testo?',
          passage:
            'Prima frase del brano. Seconda frase del brano. Terza frase del brano. Quarta frase del brano.',
          options: ['La famiglia', 'Il lavoro', 'Le vacanze', 'Lo sport'],
        },
      },
    };

    mockDailyQuestionsState.currentQuestion = readingQuestion as unknown as typeof stableCurrentQuestion;
    mockDailyQuestionsState.questions = [readingQuestion];

    renderComponent();

    await waitFor(() => {
      expect(
        screen.getByText('Qual è il tema principale del testo?')
      ).toBeInTheDocument();
    });

    const passageCall = snippetMock.mock.calls.find(call => {
      const props = call[0];
      const componentProps = props.componentProps as
        | { style?: { lineHeight?: number } }
        | undefined;
      return componentProps?.style?.lineHeight === 1.7;
    });

    expect(passageCall).toBeDefined();
    expect(passageCall?.[0].text).toContain('Prima frase del brano');
  });
});
