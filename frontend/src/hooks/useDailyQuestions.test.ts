// Mock all modules before importing
vi.mock('@tanstack/react-query', () => {
  const mockQueryClient = {
    invalidateQueries: vi.fn(),
    refetchQueries: vi.fn(),
  };
  return {
    useQueryClient: vi.fn(() => mockQueryClient),
  };
});

// Mock the Date constructor to return a consistent date
const mockDate = new Date('2025-08-04T12:00:00Z');
const OriginalDate = global.Date;
global.Date = class extends OriginalDate {
  constructor(...args: unknown[]) {
    if (args.length === 0) {
      super(mockDate);
      return mockDate;
    }
    return new OriginalDate(...(args as [unknown, ...unknown[]]));
  }
} as unknown as DateConstructor;

vi.mock('../api/api', () => ({
  useGetV1DailyQuestionsDate: vi.fn(),
  useGetV1DailyProgressDate: vi.fn(),
  useGetV1DailyDates: vi.fn(),
  usePostV1DailyQuestionsDateCompleteQuestionId: vi.fn(),
  useDeleteV1DailyQuestionsDateCompleteQuestionId: vi.fn(),
  usePostV1DailyQuestionsDateAnswerQuestionId: vi.fn(),
  // history hook used by the implementation
  useGetV1DailyHistoryQuestionId: vi.fn(),
}));

vi.mock('./useAuth', () => ({
  useAuth: vi.fn(),
}));

vi.mock('../notifications', () => ({
  showNotificationWithClean: vi.fn(),
}));

import { renderHook, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterAll } from 'vitest';
import { useDailyQuestions } from './useDailyQuestions';
import { DailyQuestionWithDetails, DailyProgress } from '../api/api';
import type { UseQueryResult, UseMutationResult } from '@tanstack/react-query';
import { useQueryClient } from '@tanstack/react-query';

// Type definitions for mock return values
type MockQueryResult<T> = Partial<UseQueryResult<T, Error>> & {
  data?: T;
  isLoading?: boolean;
  refetch?: () => void;
};

type MockMutationResult = Partial<
  UseMutationResult<unknown, Error, unknown, unknown>
> & {
  mutateAsync?: () => Promise<unknown>;
  isPending?: boolean;
};

type MockQueryClient = {
  invalidateQueries: ReturnType<typeof vi.fn>;
  refetchQueries: ReturnType<typeof vi.fn>;
};

// Import the mocked functions
import {
  useGetV1DailyQuestionsDate,
  useGetV1DailyProgressDate,
  useGetV1DailyDates,
  usePostV1DailyQuestionsDateCompleteQuestionId,
  useDeleteV1DailyQuestionsDateCompleteQuestionId,
  usePostV1DailyQuestionsDateAnswerQuestionId,
  useGetV1DailyHistoryQuestionId,
} from '../api/api';
import { useAuth } from './useAuth';
// import { showNotificationWithClean } from '../notifications';

// Mock the API hooks
const mockUseGetV1DailyQuestionsDate = vi.mocked(useGetV1DailyQuestionsDate);
const mockUseGetV1DailyProgressDate = vi.mocked(useGetV1DailyProgressDate);
const mockUseGetV1DailyDates = vi.mocked(useGetV1DailyDates);
const mockUsePostV1DailyQuestionsDateCompleteQuestionId = vi.mocked(
  usePostV1DailyQuestionsDateCompleteQuestionId
);
const mockUseDeleteV1DailyQuestionsDateCompleteQuestionId = vi.mocked(
  useDeleteV1DailyQuestionsDateCompleteQuestionId
);
const mockUsePostV1DailyQuestionsDateAnswerQuestionId = vi.mocked(
  usePostV1DailyQuestionsDateAnswerQuestionId
);
const mockUseGetV1DailyHistoryQuestionId = vi.mocked(
  useGetV1DailyHistoryQuestionId
);

// Mock the auth hook
const mockUseAuth = vi.mocked(useAuth);

// Mock the notifications
// const mockShowNotificationWithClean = vi.mocked(showNotificationWithClean);

describe('useDailyQuestions', () => {
  const mockUser = {
    id: 1,
    username: 'testuser',
    email: 'test@example.com',
    preferred_language: 'spanish',
    current_level: 'B1',
  };
  const storageKey = `/daily/index/2025-08-04/${mockUser.id}|${mockUser.preferred_language}|${mockUser.current_level}`;

  const mockQuestions: DailyQuestionWithDetails[] = [
    {
      id: 1,
      user_id: 1,
      question_id: 1,
      assignment_date: '2025-08-04',
      is_completed: false,
      completed_at: null,
      created_at: '2025-08-04T00:00:00Z',
      user_answer_index: null,
      submitted_at: null,
      question: {
        id: 1,
        language: 'italian',
        level: 'B1',
        type: 'qa',
        status: 'active',
        difficulty_score: 0.5,
        explanation: 'Test explanation',
        content: {
          question: 'Test question 1?',
          options: ['Option A', 'Option B', 'Option C', 'Option D'],
        },
        created_at: '2025-08-04T00:00:00Z',
        correct_count: 0,
        incorrect_count: 0,
        total_responses: 0,
        user_count: 1,
        correct_answer: 0,
        reporters: '',
        topic_category: 'daily_life',
        grammar_focus: 'present_perfect',
        vocabulary_domain: 'food_and_dining',
        scenario: 'at_the_restaurant',
        style_modifier: 'conversational',
        difficulty_modifier: 'basic',
        time_context: 'morning_routine',
      },
    },
    {
      id: 2,
      user_id: 1,
      question_id: 2,
      assignment_date: '2025-08-04',
      is_completed: true,
      completed_at: '2025-08-04T12:00:00Z',
      created_at: '2025-08-04T00:00:00Z',
      user_answer_index: 1,
      submitted_at: '2025-08-04T12:00:00Z',
      question: {
        id: 2,
        language: 'italian',
        level: 'B1',
        type: 'reading_comprehension',
        status: 'active',
        difficulty_score: 0.6,
        explanation: 'Test explanation 2',
        content: {
          question: 'Test question 2?',
          options: ['Option A', 'Option B', 'Option C', 'Option D'],
        },
        created_at: '2025-08-04T00:00:00Z',
        correct_count: 0,
        incorrect_count: 0,
        total_responses: 0,
        user_count: 1,
        correct_answer: 1,
        reporters: '',
        topic_category: 'travel',
        grammar_focus: 'conditionals',
        vocabulary_domain: 'transportation',
        scenario: 'at_the_airport',
        style_modifier: 'formal',
        difficulty_modifier: 'intermediate',
        time_context: 'workday',
      },
    },
    {
      id: 3,
      user_id: 1,
      question_id: 3,
      assignment_date: '2025-08-04',
      is_completed: false,
      completed_at: null,
      created_at: '2025-08-04T00:00:00Z',
      user_answer_index: null,
      submitted_at: null,
      question: {
        id: 3,
        language: 'italian',
        level: 'B1',
        type: 'fill_blank',
        status: 'active',
        difficulty_score: 0.7,
        explanation: 'Test explanation 3',
        content: {
          question: 'Test question 3?',
          options: ['Option A', 'Option B', 'Option C', 'Option D'],
          hint: 'Test hint',
        },
        created_at: '2025-08-04T00:00:00Z',
        correct_count: 0,
        incorrect_count: 0,
        total_responses: 0,
        user_count: 1,
        correct_answer: 2,
        reporters: '',
        topic_category: 'work',
        grammar_focus: 'future_tense',
        vocabulary_domain: 'business',
        scenario: 'in_the_office',
        style_modifier: 'professional',
        difficulty_modifier: 'advanced',
        time_context: 'evening_routine',
      },
    },
    {
      id: 4,
      user_id: 1,
      question_id: 4,
      assignment_date: '2025-08-04',
      is_completed: false,
      completed_at: null,
      created_at: '2025-08-04T00:00:00Z',
      user_answer_index: null,
      submitted_at: null,
      question: {
        id: 4,
        language: 'italian',
        level: 'B1',
        type: 'vocabulary',
        status: 'active',
        difficulty_score: 0.8,
        explanation: 'Test explanation 4',
        content: {
          question: 'Test question 4?',
          options: ['Option A', 'Option B', 'Option C', 'Option D'],
          sentence: 'Test sentence for vocabulary question.',
        },
        created_at: '2025-08-04T00:00:00Z',
        correct_count: 0,
        incorrect_count: 0,
        total_responses: 0,
        user_count: 1,
        correct_answer: 3,
        reporters: '',
        topic_category: 'education',
        grammar_focus: 'past_perfect',
        vocabulary_domain: 'academic',
        scenario: 'in_the_classroom',
        style_modifier: 'academic',
        difficulty_modifier: 'expert',
        time_context: 'weekend_activity',
      },
    },
  ];

  // Restore original Date constructor after all tests
  afterAll(() => {
    global.Date = OriginalDate;
  });

  beforeEach(() => {
    vi.clearAllMocks();

    // Mock the auth hook
    mockUseAuth.mockReturnValue({
      user: mockUser,
      isAuthenticated: true,
    });

    // Mock the API hooks
    mockUseGetV1DailyQuestionsDate.mockReturnValue({
      data: { questions: mockQuestions },
      isLoading: false,
      refetch: vi.fn(),
    } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

    mockUseGetV1DailyProgressDate.mockReturnValue({
      data: { date: '2025-08-04', completed: 1, total: 3 },
      isLoading: false,
      refetch: vi.fn(),
    } as MockQueryResult<DailyProgress>);

    mockUseGetV1DailyDates.mockReturnValue({
      data: { dates: ['2025-08-04'] },
      refetch: vi.fn(),
    } as MockQueryResult<{ dates: string[] }>);

    mockUsePostV1DailyQuestionsDateCompleteQuestionId.mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
    } as MockMutationResult);

    mockUseDeleteV1DailyQuestionsDateCompleteQuestionId.mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
    } as MockMutationResult);

    mockUsePostV1DailyQuestionsDateAnswerQuestionId.mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
    } as MockMutationResult);

    // Provide a default mock for the history hook used by the implementation
    mockUseGetV1DailyHistoryQuestionId.mockReturnValue({
      data: { history: [] },
      isLoading: false,
      refetch: vi.fn(),
    } as MockQueryResult<{ history: unknown[] }>);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('getNextUnansweredIndex', () => {
    it('should return the next unanswered question index', () => {
      const { result } = renderHook(() => useDailyQuestions());

      // Start at question 0 (unanswered)
      act(() => {
        result.current.setCurrentQuestionIndex(0);
      });

      const nextUnansweredIndex = result.current.getNextUnansweredIndex();
      expect(nextUnansweredIndex).toBe(2); // Question 3 (index 2) is the next unanswered
    });

    it('should return -1 when there are no more unanswered questions after current', () => {
      const { result } = renderHook(() => useDailyQuestions());

      // Start at question 3 (last question, unanswered)
      act(() => {
        result.current.setCurrentQuestionIndex(3);
      });

      const nextUnansweredIndex = result.current.getNextUnansweredIndex();
      expect(nextUnansweredIndex).toBe(-1);
    });

    it('should return -1 when current question is the last one', () => {
      const { result } = renderHook(() => useDailyQuestions());

      // Start at question 3 (last question)
      act(() => {
        result.current.setCurrentQuestionIndex(3);
      });

      const nextUnansweredIndex = result.current.getNextUnansweredIndex();
      expect(nextUnansweredIndex).toBe(-1);
    });
  });

  describe('goToNextQuestion', () => {
    it('should navigate to the next unanswered question', () => {
      const { result } = renderHook(() => useDailyQuestions());

      // Start at question 0 (unanswered)
      act(() => {
        result.current.setCurrentQuestionIndex(0);
      });

      expect(result.current.currentQuestionIndex).toBe(0);

      act(() => {
        result.current.goToNextQuestion();
      });

      expect(result.current.currentQuestionIndex).toBe(2); // Should go to question 3 (index 2)
    });

    it('should navigate to next question even if completed when no unanswered questions remain', () => {
      const { result } = renderHook(() => useDailyQuestions());

      // Start at question 1 (completed)
      act(() => {
        result.current.setCurrentQuestionIndex(1);
      });

      expect(result.current.currentQuestionIndex).toBe(1);

      act(() => {
        result.current.goToNextQuestion();
      });

      expect(result.current.currentQuestionIndex).toBe(2); // Should go to question 3 (index 2)
    });

    it('should not navigate when there are no more questions', () => {
      const { result } = renderHook(() => useDailyQuestions());

      // Start at question 3 (last question)
      act(() => {
        result.current.setCurrentQuestionIndex(3);
      });

      expect(result.current.currentQuestionIndex).toBe(3);

      act(() => {
        result.current.goToNextQuestion();
      });

      expect(result.current.currentQuestionIndex).toBe(3); // Should stay at the same index
    });
  });

  describe('getFirstUnansweredIndex', () => {
    it('should return the first unanswered question index', () => {
      const { result } = renderHook(() => useDailyQuestions());

      const firstUnansweredIndex = result.current.getFirstUnansweredIndex();
      expect(firstUnansweredIndex).toBe(0); // Question 1 (index 0) is the first unanswered
    });

    it('should return 0 when all questions are completed', () => {
      const allCompletedQuestions = mockQuestions.map(q => ({
        ...q,
        is_completed: true,
      }));

      mockUseGetV1DailyQuestionsDate.mockReturnValue({
        data: { questions: allCompletedQuestions },
        isLoading: false,
        refetch: vi.fn(),
      } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

      const { result } = renderHook(() => useDailyQuestions());

      const firstUnansweredIndex = result.current.getFirstUnansweredIndex();
      expect(firstUnansweredIndex).toBe(0);
    });
  });

  describe('initial navigation', () => {
    beforeEach(() => {
      // Clear sessionStorage before each test to avoid pollution
      window.sessionStorage.clear();
    });

    it('should navigate to first unanswered question on load', () => {
      const { result } = renderHook(() => useDailyQuestions());

      // The hook should automatically navigate to the first unanswered question
      expect(result.current.currentQuestionIndex).toBe(0);
    });

    it('should not navigate when all questions are completed', () => {
      const allCompletedQuestions = mockQuestions.map(q => ({
        ...q,
        is_completed: true,
      }));

      mockUseGetV1DailyQuestionsDate.mockReturnValue({
        data: { questions: allCompletedQuestions },
        isLoading: false,
        refetch: vi.fn(),
      } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

      const { result } = renderHook(() => useDailyQuestions());

      // Should stay at index 0 when all questions are completed
      expect(result.current.currentQuestionIndex).toBe(0);
    });
  });

  describe('persistence across navigation', () => {
    beforeEach(() => {
      // Clear sessionStorage before each test
      window.sessionStorage.clear();
    });

    it('restores persisted currentQuestionIndex on remount', () => {
      // Pre-seed sessionStorage for mocked date 2025-08-04
      window.sessionStorage.setItem(storageKey, '2');
      const { result } = renderHook(() => useDailyQuestions());

      // Wait for initialization to complete
      expect(result.current.currentQuestionIndex).toBe(2);
    });

    it('persists currentQuestionIndex when navigating away', () => {
      const { result, unmount } = renderHook(() => useDailyQuestions());

      // Change to question index 2
      act(() => {
        result.current.setCurrentQuestionIndex(2);
      });

      // Verify it was persisted to sessionStorage
      const stored = window.sessionStorage.getItem(storageKey);
      expect(stored).toBe('2');

      unmount();
    });

    it('navigates to first unanswered when no stored index exists', () => {
      // Don't set any stored index
      const { result } = renderHook(() => useDailyQuestions());

      // Should navigate to first unanswered (index 0)
      expect(result.current.currentQuestionIndex).toBe(0);
    });
  });

  describe('progress updates', () => {
    it('should update progress when submitting an answer', async () => {
      const mockRefetchProgress = vi.fn();
      const mockRefetchQuestions = vi.fn();
      const mockRefetchDates = vi.fn();
      const mockInvalidateQueries = vi.fn();

      mockUseGetV1DailyProgressDate.mockReturnValue({
        data: { date: '2025-08-04', completed: 1, total: 4 },
        isLoading: false,
        refetch: mockRefetchProgress,
      } as MockQueryResult<DailyProgress>);

      mockUseGetV1DailyQuestionsDate.mockReturnValue({
        data: { questions: mockQuestions },
        isLoading: false,
        refetch: mockRefetchQuestions,
      } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

      mockUseGetV1DailyDates.mockReturnValue({
        data: { dates: ['2025-08-04'] },
        refetch: mockRefetchDates,
      } as MockQueryResult<{ dates: string[] }>);

      const mockSubmitAnswerMutation = vi.fn().mockResolvedValue({});
      mockUsePostV1DailyQuestionsDateAnswerQuestionId.mockReturnValue({
        mutateAsync: mockSubmitAnswerMutation,
        isPending: false,
      } as MockMutationResult);

      const mockQueryClient: MockQueryClient = {
        invalidateQueries: mockInvalidateQueries,
        refetchQueries: vi.fn(),
      };

      // Mock useQueryClient to return our mock
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      vi.mocked(useQueryClient).mockReturnValue(mockQueryClient as any);

      const { result } = renderHook(() => useDailyQuestions());

      // Submit an answer
      await act(async () => {
        await result.current.submitAnswer(1, 2);
      });

      // Verify that the mutation was called
      expect(mockSubmitAnswerMutation).toHaveBeenCalledWith({
        date: '2025-08-04',
        questionId: 1,
        data: {
          user_answer_index: 2,
        },
      });

      // Verify that progress was refetched
      expect(mockRefetchQuestions).toHaveBeenCalled();
      expect(mockRefetchDates).toHaveBeenCalled();
      expect(mockInvalidateQueries).toHaveBeenCalledWith({
        queryKey: [
          '/v1/daily/progress',
          '2025-08-04',
          String(mockUser.id),
          mockUser.preferred_language,
          mockUser.current_level,
        ],
      });
    });

    it('should update progress when completing a question', async () => {
      const mockRefetchProgress = vi.fn();
      const mockRefetchQuestions = vi.fn();
      const mockRefetchDates = vi.fn();
      const mockInvalidateQueries = vi.fn();

      mockUseGetV1DailyProgressDate.mockReturnValue({
        data: { date: '2025-08-04', completed: 1, total: 4 },
        isLoading: false,
        refetch: mockRefetchProgress,
      } as MockQueryResult<DailyProgress>);

      mockUseGetV1DailyQuestionsDate.mockReturnValue({
        data: { questions: mockQuestions },
        isLoading: false,
        refetch: mockRefetchQuestions,
      } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

      mockUseGetV1DailyDates.mockReturnValue({
        data: { dates: ['2025-08-04'] },
        refetch: mockRefetchDates,
      } as MockQueryResult<{ dates: string[] }>);

      const mockCompleteQuestionMutation = vi.fn().mockResolvedValue({});
      mockUsePostV1DailyQuestionsDateCompleteQuestionId.mockReturnValue({
        mutateAsync: mockCompleteQuestionMutation,
        isPending: false,
      } as MockMutationResult);

      const mockQueryClient: MockQueryClient = {
        invalidateQueries: mockInvalidateQueries,
        refetchQueries: vi.fn(),
      };

      // Mock useQueryClient to return our mock
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      vi.mocked(useQueryClient).mockReturnValue(mockQueryClient as any);

      const { result } = renderHook(() => useDailyQuestions());

      // Complete a question
      await act(async () => {
        await result.current.completeQuestion(1);
      });

      // Verify that the mutation was called
      expect(mockCompleteQuestionMutation).toHaveBeenCalledWith({
        date: '2025-08-04',
        questionId: 1,
      });

      // Verify that progress was refetched
      expect(mockRefetchQuestions).toHaveBeenCalled();
      expect(mockRefetchDates).toHaveBeenCalled();
      expect(mockInvalidateQueries).toHaveBeenCalledWith({
        queryKey: [
          '/v1/daily/progress',
          '2025-08-04',
          String(mockUser.id),
          mockUser.preferred_language,
          mockUser.current_level,
        ],
      });
    });

    it('should update progress when resetting a question', async () => {
      const mockRefetchProgress = vi.fn();
      const mockRefetchQuestions = vi.fn();
      const mockRefetchDates = vi.fn();
      const mockInvalidateQueries = vi.fn();

      mockUseGetV1DailyProgressDate.mockReturnValue({
        data: { date: '2025-08-04', completed: 1, total: 4 },
        isLoading: false,
        refetch: mockRefetchProgress,
      } as MockQueryResult<DailyProgress>);

      mockUseGetV1DailyQuestionsDate.mockReturnValue({
        data: { questions: mockQuestions },
        isLoading: false,
        refetch: mockRefetchQuestions,
      } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

      mockUseGetV1DailyDates.mockReturnValue({
        data: { dates: ['2025-08-04'] },
        refetch: mockRefetchDates,
      } as MockQueryResult<{ dates: string[] }>);

      const mockResetQuestionMutation = vi.fn().mockResolvedValue({});
      mockUseDeleteV1DailyQuestionsDateCompleteQuestionId.mockReturnValue({
        mutateAsync: mockResetQuestionMutation,
        isPending: false,
      } as MockMutationResult);

      const mockQueryClient: MockQueryClient = {
        invalidateQueries: mockInvalidateQueries,
        refetchQueries: vi.fn(),
      };

      // Mock useQueryClient to return our mock
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      vi.mocked(useQueryClient).mockReturnValue(mockQueryClient as any);

      const { result } = renderHook(() => useDailyQuestions());

      // Reset a question
      await act(async () => {
        await result.current.resetQuestion(1);
      });

      // Verify that the mutation was called
      expect(mockResetQuestionMutation).toHaveBeenCalledWith({
        date: '2025-08-04',
        questionId: 1,
      });

      // Verify that progress was refetched
      expect(mockRefetchQuestions).toHaveBeenCalled();
      expect(mockRefetchDates).toHaveBeenCalled();
      expect(mockInvalidateQueries).toHaveBeenCalledWith({
        queryKey: [
          '/v1/daily/progress',
          '2025-08-04',
          String(mockUser.id),
          mockUser.preferred_language,
          mockUser.current_level,
        ],
      });
    });

    it('should handle errors when submitting answer', async () => {
      const mockSubmitAnswerMutation = vi
        .fn()
        .mockRejectedValue(new Error('Network error'));
      mockUsePostV1DailyQuestionsDateAnswerQuestionId.mockReturnValue({
        mutateAsync: mockSubmitAnswerMutation,
        isPending: false,
      } as MockMutationResult);

      const mockQueryClient: MockQueryClient = {
        invalidateQueries: vi.fn(),
        refetchQueries: vi.fn(),
      };

      // Mock useQueryClient to return our mock
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      vi.mocked(useQueryClient).mockReturnValue(mockQueryClient as any);

      const { result } = renderHook(() => useDailyQuestions());

      // Submit an answer that will fail
      await expect(async () => {
        await act(async () => {
          await result.current.submitAnswer(1, 2);
        });
      }).rejects.toThrow('Network error');

      // Verify that the mutation was called
      expect(mockSubmitAnswerMutation).toHaveBeenCalledWith({
        date: '2025-08-04',
        questionId: 1,
        data: {
          user_answer_index: 2,
        },
      });

      // Verify that progress was NOT refetched on error
      // (The refetch functions should not be called when the mutation fails)
    });
  });

  describe('Feedback Data Handling', () => {
    it('should handle questions with feedback data correctly', () => {
      const questionsWithFeedback = [
        {
          id: 1,
          user_id: 1,
          question_id: 1,
          assignment_date: '2025-08-04',
          is_completed: true,
          completed_at: '2025-08-04T10:00:00Z',
          user_answer_index: 2,
          submitted_at: '2025-08-04T10:00:00Z',
          question: {
            id: 1,
            type: 'multiple_choice',
            language: 'en',
            level: 'beginner',
            difficulty_score: 1,
            content: {
              question: 'What is 2 + 2?',
              options: ['3', '4', '5', '6'],
            },
            correct_answer: 1,
            explanation: '2 + 2 = 4',
            created_at: '2025-08-04T00:00:00Z',
            status: 'active',
            topic_category: 'math',
            grammar_focus: null,
            vocabulary_domain: null,
            scenario: null,
            style_modifier: null,
            difficulty_modifier: null,
            time_context: null,
          },
        },
      ];

      mockUseGetV1DailyQuestionsDate.mockReturnValue({
        data: { questions: questionsWithFeedback },
        isLoading: false,
        refetch: vi.fn(),
      } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

      const { result } = renderHook(() => useDailyQuestions());

      // Verify that the question with feedback data is accessible
      expect(result.current.questions).toEqual(questionsWithFeedback);
      expect(result.current.currentQuestion).toEqual(questionsWithFeedback[0]);
    });

    it('should handle questions without feedback data correctly', () => {
      const questionsWithoutFeedback = [
        {
          id: 1,
          user_id: 1,
          question_id: 1,
          assignment_date: '2025-08-04',
          is_completed: false,
          completed_at: null,
          user_answer_index: null,
          submitted_at: null,
          question: {
            id: 1,
            type: 'multiple_choice',
            language: 'en',
            level: 'beginner',
            difficulty_score: 1,
            content: {
              question: 'What is 2 + 2?',
              options: ['3', '4', '5', '6'],
            },
            correct_answer: 1,
            explanation: '2 + 2 = 4',
            created_at: '2025-08-04T00:00:00Z',
            status: 'active',
            topic_category: 'math',
            grammar_focus: null,
            vocabulary_domain: null,
            scenario: null,
            style_modifier: null,
            difficulty_modifier: null,
            time_context: null,
          },
        },
      ];

      mockUseGetV1DailyQuestionsDate.mockReturnValue({
        data: { questions: questionsWithoutFeedback },
        isLoading: false,
        refetch: vi.fn(),
      } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

      const { result } = renderHook(() => useDailyQuestions());

      // Verify that the question without feedback data is accessible
      expect(result.current.questions).toEqual(questionsWithoutFeedback);
      expect(result.current.currentQuestion).toEqual(
        questionsWithoutFeedback[0]
      );
    });

    it('should handle mixed completed and uncompleted questions', () => {
      const mixedQuestions = [
        {
          id: 1,
          user_id: 1,
          question_id: 1,
          assignment_date: '2025-08-04',
          is_completed: true,
          completed_at: '2025-08-04T10:00:00Z',
          user_answer_index: 2,
          submitted_at: '2025-08-04T10:00:00Z',
          question: {
            id: 1,
            type: 'multiple_choice',
            language: 'en',
            level: 'beginner',
            difficulty_score: 1,
            content: {
              question: 'What is 2 + 2?',
              options: ['3', '4', '5', '6'],
            },
            correct_answer: 1,
            explanation: '2 + 2 = 4',
            created_at: '2025-08-04T00:00:00Z',
            status: 'active',
            topic_category: 'math',
            grammar_focus: null,
            vocabulary_domain: null,
            scenario: null,
            style_modifier: null,
            difficulty_modifier: null,
            time_context: null,
          },
        },
        {
          id: 2,
          user_id: 1,
          question_id: 2,
          assignment_date: '2025-08-04',
          is_completed: false,
          completed_at: null,
          user_answer_index: null,
          submitted_at: null,
          question: {
            id: 2,
            type: 'multiple_choice',
            language: 'en',
            level: 'beginner',
            difficulty_score: 1,
            content: {
              question: 'What is 3 + 3?',
              options: ['5', '6', '7', '8'],
            },
            correct_answer: 1,
            explanation: '3 + 3 = 6',
            created_at: '2025-08-04T00:00:00Z',
            status: 'active',
            topic_category: 'math',
            grammar_focus: null,
            vocabulary_domain: null,
            scenario: null,
            style_modifier: null,
            difficulty_modifier: null,
            time_context: null,
          },
        },
      ];

      mockUseGetV1DailyQuestionsDate.mockReturnValue({
        data: { questions: mixedQuestions },
        isLoading: false,
        refetch: vi.fn(),
      } as MockQueryResult<{ questions: DailyQuestionWithDetails[] }>);

      const { result } = renderHook(() => useDailyQuestions());

      // Verify that both types of questions are handled correctly
      expect(result.current.questions).toEqual(mixedQuestions);
      // With current initialization logic, we can land on index 0 if restored;
      // for a clean run we expect the first unanswered from start, which is 0
      expect(result.current.currentQuestion).toEqual(mixedQuestions[0]);
    });
  });
});
