import { useEffect, useRef } from 'react';
import { useNavigate, useParams, useLocation } from 'react-router-dom';
import type { Question } from '../api/api';
import type { QuestionMode } from '../components/QuestionPageBase';
import { isMobilePath } from '../utils/device';

export interface UseQuestionUrlStateOptions {
  mode: QuestionMode;
  question: Question | null;
  isLoading: boolean;
}

export interface UseQuestionUrlStateReturn {
  questionId: string | undefined;
  navigateToQuestion: (questionId: number | null) => void;
}

/**
 * Hook to manage URL state for questions across different modes (quiz, vocabulary, reading).
 * Automatically updates the URL to include the current question ID so users can copy/paste URLs
 * to return to the same question.
 */
export function useQuestionUrlState(
  options: UseQuestionUrlStateOptions
): UseQuestionUrlStateReturn {
  const { mode, question, isLoading } = options;
  const { questionId: questionIdFromParams } = useParams();
  const location = useLocation();
  const navigate = useNavigate();

  // Detect if we're on mobile path
  const isMobile = isMobilePath();
  const mobilePrefix = isMobile ? '/m' : '';

  const basePath =
    mode === 'quiz'
      ? `${mobilePrefix}/quiz`
      : mode === 'reading'
        ? `${mobilePrefix}/reading-comprehension`
        : `${mobilePrefix}/vocabulary`;

  // Extract question ID from the current pathname
  const questionIdMatch = location.pathname.match(
    new RegExp(`^${basePath.replace(/\//g, '\\/')}/(\\d+)$`)
  );
  const questionId = questionIdMatch?.[1] || undefined;

  // Log when questionId changes
  const prevQuestionId = useRef(questionId);
  if (prevQuestionId.current !== questionId) {
    prevQuestionId.current = questionId;
  }

  // Track whether we previously had a question so we only clear the URL when
  // a question was present and then becomes null (user navigated away or asked
  // for a new question). This avoids clearing the URL on initial mount when
  // the page was opened at `/quiz/:id` and question is still loading.
  const prevHadQuestionRef = useRef<boolean>(!!question);
  useEffect(() => {
    prevHadQuestionRef.current = !!question;
  }, [question]);

  /**
   * Navigate to a specific question or clear the question ID from URL
   */
  const navigateToQuestion = (newQuestionId: number | null) => {
    const targetPath = newQuestionId
      ? `${basePath}/${newQuestionId}`
      : basePath;

    // Only navigate if the path is actually different
    if (window.location.pathname !== targetPath) {
      navigate(targetPath, { replace: true });
    } else {
    }
  };

  /**
   * Update URL when question changes (but not while loading)
   */
  useEffect(() => {
    // Don't update URL if we're currently loading
    if (isLoading) {
      return;
    }

    // If we have a question and it doesn't match the URL, update the URL
    if (question?.id && questionId !== question.id.toString()) {
      navigateToQuestion(question.id);
    } else if (!question && questionId) {
      // If we have no question but still have questionId in URL, only clear it
      // if we previously had a question loaded — this avoids clearing the
      // URL on initial mount when loading a specific question via copy/paste.
      if (prevHadQuestionRef.current) {
        navigateToQuestion(null);
      }
    }
  }, [
    question?.id,
    questionId,
    isLoading,
    navigateToQuestion,
    location.pathname,
    questionIdFromParams,
  ]);

  return {
    questionId,
    navigateToQuestion,
  };
}
