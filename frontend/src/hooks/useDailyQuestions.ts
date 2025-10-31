import { useState, useEffect, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { useAuth } from './useAuth';
import {
  useGetV1DailyQuestionsDate,
  useGetV1DailyDates,
  useGetV1DailyProgressDate,
  usePostV1DailyQuestionsDateCompleteQuestionId,
  useDeleteV1DailyQuestionsDateCompleteQuestionId,
  usePostV1DailyQuestionsDateAnswerQuestionId,
  DailyQuestionWithDetails,
  DailyQuestionHistory,
  DailyProgress,
  AnswerResponse,
  useGetV1DailyHistoryQuestionId,
} from '../api/api';
import { showNotificationWithClean } from '../notifications';

export interface UseDailyQuestionsReturn {
  // State
  selectedDate: string;
  setSelectedDate: (date: string) => void;
  questions: DailyQuestionWithDetails[] | undefined;
  progress: DailyProgress | undefined;
  availableDates: string[] | undefined;
  currentQuestionIndex: number;
  setCurrentQuestionIndex: (index: number) => void;

  // Loading states
  isLoading: boolean;
  isProgressLoading: boolean;
  isCompletingQuestion: boolean;
  isResettingQuestion: boolean;
  isSubmittingAnswer: boolean;
  isHistoryLoading: boolean;

  // Actions
  completeQuestion: (questionId: number) => Promise<void>;
  resetQuestion: (questionId: number) => Promise<void>;
  submitAnswer: (
    questionId: number,
    userAnswerIndex: number
  ) => Promise<AnswerResponse>;
  goToNextQuestion: () => void;
  goToPreviousQuestion: () => void;
  getQuestionHistory: (questionId: number) => Promise<void>;

  // Computed
  currentQuestion: DailyQuestionWithDetails | undefined;
  hasNextQuestion: boolean;
  hasPreviousQuestion: boolean;
  isAllCompleted: boolean;
  getNextUnansweredIndex: () => number;
  getFirstUnansweredIndex: () => number;

  // History
  questionHistory: DailyQuestionHistory[] | undefined;
}

// Format a date as YYYY-MM-DD in the user's local timezone to avoid UTC
// rollover issues (which can shift dates and cause duplicate fetches).
const formatDateLocal = (d: Date): string => {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const formatDateForAPI = (date: Date | string): string => {
  if (typeof date === 'string') {
    // If the backend already returns YYYY-MM-DD, use it as-is.
    if (/^\d{4}-\d{2}-\d{2}$/.test(date)) return date;
    return formatDateLocal(new Date(date));
  }
  return formatDateLocal(date);
};

const getCurrentDateString = (): string => {
  return formatDateForAPI(new Date());
};

export const useDailyQuestions = (): UseDailyQuestionsReturn => {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  const [selectedDate, setSelectedDate] = useState<string>(
    getCurrentDateString()
  );
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState<number>(0);
  const [hasInitialized, setHasInitialized] = useState<boolean>(false);
  const storageKey = `/daily/index/${selectedDate}`;

  // API hooks
  const {
    data: questionsResponse,
    isLoading: isQuestionsLoading,
    refetch: refetchQuestions,
  } = useGetV1DailyQuestionsDate(selectedDate, {
    query: {
      enabled: !!user,
      refetchOnWindowFocus: false,
      queryKey: [`/v1/daily/questions/${selectedDate}`, user?.id],
    },
  });

  const { data: progress, isLoading: isProgressLoading } =
    useGetV1DailyProgressDate(selectedDate, {
      query: {
        enabled: !!user,
        refetchOnWindowFocus: false,
        queryKey: [`/v1/daily/progress/${selectedDate}`, user?.id],
      },
    });

  const { data: availableDatesResponse, refetch: refetchDates } =
    useGetV1DailyDates({
      query: {
        enabled: !!user,
        refetchOnWindowFocus: false,
        queryKey: ['/v1/daily/dates', user?.id],
      },
    });

  const {
    mutateAsync: completeQuestionMutation,
    isPending: isCompletingQuestion,
  } = usePostV1DailyQuestionsDateCompleteQuestionId();

  const { mutateAsync: resetQuestionMutation, isPending: isResettingQuestion } =
    useDeleteV1DailyQuestionsDateCompleteQuestionId();

  const { mutateAsync: submitAnswerMutation, isPending: isSubmittingAnswer } =
    usePostV1DailyQuestionsDateAnswerQuestionId();

  // Question history hook
  const [historyQuestionId, setHistoryQuestionId] = useState<number | null>(
    null
  );
  const { data: questionHistoryResponse, isLoading: isHistoryLoading } =
    useGetV1DailyHistoryQuestionId(historyQuestionId || 0, {
      query: {
        enabled: !!historyQuestionId,
        refetchOnWindowFocus: false,
        queryKey: [`/v1/daily/history/${historyQuestionId}`, user?.id],
      },
    });

  // expose only the history array to consumers
  const questionHistory = questionHistoryResponse?.history;

  // Reset question index when date changes
  useEffect(() => {
    setCurrentQuestionIndex(0);
    setHasInitialized(false);
  }, [selectedDate]);

  // Persist current index per date to survive navigation/unmounts
  // Only persist after initialization to avoid overwriting the stored value on mount
  useEffect(() => {
    if (!hasInitialized) return;
    try {
      window.sessionStorage.setItem(storageKey, String(currentQuestionIndex));
    } catch {}
  }, [currentQuestionIndex, storageKey, hasInitialized]);

  // Actions
  const completeQuestion = useCallback(
    async (questionId: number) => {
      try {
        await completeQuestionMutation({
          date: selectedDate,
          questionId: questionId,
        });

        // Invalidate queries to update UI
        await Promise.all([
          refetchQuestions(),
          queryClient.invalidateQueries({
            queryKey: [`/v1/daily/progress/${selectedDate}`, user?.id],
          }),
          refetchDates(),
        ]);

        // Force refetch progress data
        await queryClient.refetchQueries({
          queryKey: [`/v1/daily/progress/${selectedDate}`, user?.id],
        });

        showNotificationWithClean({
          title: 'Question Completed',
          message: 'Great job! Question marked as completed.',
          color: 'green',
        });
      } catch (error) {
        showNotificationWithClean({
          title: 'Error',
          message: 'Failed to mark question as completed. Please try again.',
          color: 'red',
        });
        throw error;
      }
    },
    [
      selectedDate,
      completeQuestionMutation,
      refetchQuestions,
      queryClient,
      refetchDates,
      user?.id,
    ]
  );

  const resetQuestion = useCallback(
    async (questionId: number) => {
      try {
        await resetQuestionMutation({
          date: selectedDate,
          questionId: questionId,
        });

        // Invalidate queries to update UI
        await Promise.all([
          refetchQuestions(),
          queryClient.invalidateQueries({
            queryKey: [`/v1/daily/progress/${selectedDate}`, user?.id],
          }),
          refetchDates(),
          // Invalidate history for this specific question since its state changed
          queryClient.invalidateQueries({
            queryKey: [`/v1/daily/history/${questionId}`, user?.id],
          }),
        ]);

        showNotificationWithClean({
          title: 'Question Reset',
          message: 'Question has been reset. You can answer it again.',
          color: 'blue',
        });
      } catch (error) {
        showNotificationWithClean({
          title: 'Error',
          message: 'Failed to reset question. Please try again.',
          color: 'red',
        });
        throw error;
      }
    },
    [
      selectedDate,
      resetQuestionMutation,
      refetchQuestions,
      queryClient,
      refetchDates,
      user?.id,
    ]
  );

  const getQuestionHistory = useCallback(
    async (questionId: number) => {
      try {
        // Ensure we re-query fresh data each time the history modal is opened
        // by invalidating any cached history for this question first.
        await queryClient.invalidateQueries({
          queryKey: [`/v1/daily/history/${questionId}`, user?.id],
        });

        // Setting the historyQuestionId enables the generated query hook
        // which will fetch the (now invalidated) data.
        setHistoryQuestionId(questionId);
      } catch {
        showNotificationWithClean({
          title: 'Error',
          message: 'Failed to load question history. Please try again.',
          color: 'red',
        });
      }
    },
    [queryClient, user?.id]
  );

  const submitAnswer = useCallback(
    async (questionId: number, userAnswerIndex: number) => {
      try {
        const response = await submitAnswerMutation({
          date: selectedDate,
          questionId: questionId,
          data: {
            user_answer_index: userAnswerIndex,
          },
        });

        // Invalidate queries to update UI
        await Promise.all([
          refetchQuestions(),
          queryClient.invalidateQueries({
            queryKey: [`/v1/daily/progress/${selectedDate}`, user?.id],
          }),
          refetchDates(),
          // Invalidate history for this specific question so it updates with the new answer
          queryClient.invalidateQueries({
            queryKey: [`/v1/daily/history/${questionId}`, user?.id],
          }),
        ]);

        // Force refetch progress data
        await queryClient.refetchQueries({
          queryKey: [`/v1/daily/progress/${selectedDate}`, user?.id],
        });

        return response;
      } catch (error) {
        showNotificationWithClean({
          title: 'Error',
          message: 'Failed to submit answer. Please try again.',
          color: 'red',
        });
        throw error;
      }
    },
    [
      selectedDate,
      submitAnswerMutation,
      refetchQuestions,
      queryClient,
      refetchDates,
      user?.id,
    ]
  );

  // Extract questions array from response
  const questions = questionsResponse?.questions;

  const goToPreviousQuestion = useCallback(() => {
    if (currentQuestionIndex > 0) {
      setCurrentQuestionIndex(prev => prev - 1);
    }
  }, [currentQuestionIndex]);

  // New computed values for revised daily questions behavior
  const isAllCompleted = useCallback(() => {
    if (!questions || questions.length === 0) return false;
    return questions.every(q => q.is_completed);
  }, [questions]);

  const getNextUnansweredIndex = useCallback(() => {
    if (!questions) return -1;
    const nextIndex = questions.findIndex(
      (q, index) => index > currentQuestionIndex && !q.is_completed
    );
    return nextIndex >= 0 ? nextIndex : -1;
  }, [questions, currentQuestionIndex]);

  const getFirstUnansweredIndex = useCallback(() => {
    if (!questions) return 0;
    const firstUnanswered = questions.findIndex(q => !q.is_completed);
    return firstUnanswered >= 0 ? firstUnanswered : 0;
  }, [questions]);

  const goToNextQuestion = useCallback(() => {
    if (!questions) return;

    const nextUnansweredIndex = getNextUnansweredIndex();
    if (nextUnansweredIndex >= 0) {
      setCurrentQuestionIndex(nextUnansweredIndex);
    } else if (currentQuestionIndex < questions.length - 1) {
      // If no unanswered questions after current, go to next question anyway
      setCurrentQuestionIndex(prev => prev + 1);
    }
  }, [questions, currentQuestionIndex, getNextUnansweredIndex]);

  // Initialize current index on first load: restore persisted index if valid,
  // otherwise navigate to first unanswered question.
  useEffect(() => {
    if (
      questionsResponse?.questions &&
      questionsResponse.questions.length > 0 &&
      !hasInitialized
    ) {
      // Try restore
      let restoredIndex: number | null = null;
      try {
        const raw = window.sessionStorage.getItem(storageKey);
        if (raw != null) restoredIndex = Number(raw);
      } catch {}

      if (
        restoredIndex != null &&
        restoredIndex >= 0 &&
        restoredIndex < questionsResponse.questions.length
      ) {
        setCurrentQuestionIndex(restoredIndex);
      } else {
        const allCompleted = questionsResponse.questions.every(
          q => q.is_completed
        );
        if (!allCompleted) {
          const firstUnanswered = getFirstUnansweredIndex();
          if (firstUnanswered !== currentQuestionIndex) {
            setCurrentQuestionIndex(firstUnanswered);
          }
        }
      }
      setHasInitialized(true);
    }
  }, [
    questionsResponse?.questions,
    getFirstUnansweredIndex,
    currentQuestionIndex,
    hasInitialized,
    storageKey,
  ]);

  // Computed values
  const currentQuestion = questions?.[currentQuestionIndex];
  const hasNextQuestion = questions
    ? currentQuestionIndex < questions.length - 1
    : false;
  const hasPreviousQuestion = currentQuestionIndex > 0;
  const isLoading = isQuestionsLoading;

  // Convert availableDatesResponse to string array
  const availableDates = availableDatesResponse?.dates?.map(date =>
    formatDateForAPI(date)
  );

  return {
    // State
    selectedDate,
    setSelectedDate,
    questions,
    progress,
    availableDates,
    currentQuestionIndex,
    setCurrentQuestionIndex,

    // Loading states
    isLoading,
    isProgressLoading,
    isCompletingQuestion,
    isResettingQuestion,
    isSubmittingAnswer,
    isHistoryLoading,

    // Actions
    completeQuestion,
    resetQuestion,
    submitAnswer,
    goToNextQuestion,
    goToPreviousQuestion,
    getQuestionHistory,

    // Computed
    currentQuestion,
    hasNextQuestion,
    hasPreviousQuestion,
    isAllCompleted: isAllCompleted(),
    getNextUnansweredIndex,
    getFirstUnansweredIndex,

    // History
    questionHistory,
  };
};
