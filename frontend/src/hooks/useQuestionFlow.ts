import { useCallback, useEffect, useRef, useState } from 'react';
import { useAuth } from './useAuth';
import type { Question } from '../api/api';
import {
  getV1QuizQuestion,
  getV1QuizQuestionId,
  type GetV1QuizQuestionParams,
} from '../api/api';

export type QuestionMode = 'quiz' | 'reading' | 'vocabulary';

export interface UseQuestionFlowOptions {
  mode: QuestionMode;
  questionId?: string | undefined;
}

export interface UseQuestionFlowReturn {
  question: Question | null;
  setQuestion: (q: Question | null) => void;
  isLoading: boolean;
  isGenerating: boolean;
  error: string | null;
  fetchQuestion: (force?: boolean) => Promise<void>;
  forceFetchNextQuestion: () => void;
  startPolling: () => void;
  stopPolling: () => void;
}

export function useQuestionFlow(
  options: UseQuestionFlowOptions
): UseQuestionFlowReturn {
  const { user } = useAuth();
  const { mode, questionId } = options;

  const [question, setQuestion] = useState<Question | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [forceNextQuestion, setForceNextQuestion] = useState(false);

  const forceFetchNextQuestion = useCallback(() => {
    setForceNextQuestion(true);
    setQuestion(null);
    fetchQuestion(true);
  }, []);

  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const isPollingRef = useRef(false);
  const currentQuestionRef = useRef<Question | null>(null);
  // Shared inflight map so multiple components asking for the "next" question
  // or the same id do not issue duplicate API requests.
  // key -> promise
  const sharedQuestionInflightRef = useRef<Map<
    string,
    Promise<unknown>
  > | null>(null);
  if (!sharedQuestionInflightRef.current)
    sharedQuestionInflightRef.current = new Map();
  const initialFetchRef = useRef(false);

  useEffect(() => {
    currentQuestionRef.current = question;
  }, [question]);

  const stopPolling = useCallback(() => {
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
    isPollingRef.current = false;
  }, []);

  const startPolling = useCallback(() => {
    if (isPollingRef.current) return;
    isPollingRef.current = true;

    pollingIntervalRef.current = setInterval(async () => {
      try {
        if (!currentQuestionRef.current && user && user.ai_enabled !== false) {
          await fetchQuestion(true);
        }
      } catch {
        // swallow; next tick will retry
      }
    }, 3000);
    // intentionally exclude fetchQuestion from deps to avoid resetting interval
  }, [user]);

  const buildParams = (): GetV1QuizQuestionParams => {
    if (mode === 'reading') {
      return { type: 'reading_comprehension' };
    }
    if (mode === 'vocabulary') {
      return { type: 'vocabulary' };
    }
    // Quiz mode should include all types
    const params: GetV1QuizQuestionParams = {};

    // Add user level if available for better question matching
    if (user?.current_level) {
      params.level = user.current_level;
    }

    return params;
  };

  const fetchQuestion = useCallback(
    async (force = false) => {
      if (!user) return;

      // If force is true, or if we had a question but now have no questionId, always fetch a new question
      const shouldFetchNext = force || forceNextQuestion || !questionId;

      if (!shouldFetchNext && question) {
        return;
      }

      // compute inflight key: specific id or 'next:mode' for next question
      const inflightKey = shouldFetchNext ? `next:${mode}` : `id:${questionId}`;
      const sharedInflight = sharedQuestionInflightRef.current!;
      const existing = sharedInflight.get(inflightKey);
      if (existing) {
        // wait for existing inflight request to finish and reuse result
        try {
          await existing;
        } catch {
          // ignore; we'll fall through and possibly retry
        }
        // If another inflight filled the question, return early unless force
        if (!shouldFetchNext && question) {
          return;
        }
      }
      setIsLoading(true);
      setError(null);
      try {
        // Fetch by specific id or next question by params
        // When force is true, always fetch the next question
        const p = (async () => {
          const params = buildParams();
          return shouldFetchNext
            ? await getV1QuizQuestion(params)
            : await getV1QuizQuestionId(Number(questionId));
        })();
        sharedInflight.set(inflightKey, p);
        const data = await p;
        sharedInflight.delete(inflightKey);

        if ((data as { status?: string })?.status === 'generating') {
          // If AI is disabled, don't show generating state - go directly to "no questions available"
          if (user?.ai_enabled === false) {
            setIsGenerating(false);
            setError('Enable AI in settings to generate questions');
            setQuestion(null);
            return;
          }
          setIsGenerating(true);
          setError((data as { message?: string })?.message || null);
          setQuestion(null);
          startPolling();
          return;
        }
        setIsGenerating(false);
        if ((data as Question)?.content && (data as Question)?.id) {
          const newQuestion = data as Question;

          // Reset the force flag after successful fetch
          if (forceNextQuestion) {
            setForceNextQuestion(false);
          }

          setQuestion(newQuestion);
        } else {
          setError('Invalid question format received');
          setQuestion(null);
        }
      } catch (err: unknown) {
        const maybe = err as { response?: { data?: { error?: string } } };
        const errorMessage =
          maybe?.response?.data?.error || 'Failed to fetch question';
        setError(errorMessage);
      } finally {
        setIsLoading(false);
      }
    },
    // intentionally exclude fetchQuestion from deps to avoid resetting interval
    // Note: questionId is intentionally excluded to prevent circular dependencies
    [user, mode, question, forceNextQuestion]
  );

  useEffect(() => {
    // Guard initial fetch so that React StrictMode double-mounts don't cause
    // duplicate network requests during development. Only fetch once per
    // mount unless questionId changes.
    // Only do initial fetch if we don't have a specific questionId (i.e., we're on /quiz not /quiz/34)
    if (user && !initialFetchRef.current && !questionId) {
      initialFetchRef.current = true;
      fetchQuestion();
    }
    return () => stopPolling();
    // intentionally exclude fetchQuestion from deps to avoid resetting interval
    // Note: questionId is intentionally excluded to prevent circular dependencies with useParams
  }, [user]);

  // Separate effect to handle questionId changes. We guard to ensure we only
  // fetch the specific question when the initial fetch (for no-id routes)
  // has already completed or when explicitly navigated to a `/quiz/:id` URL.
  useEffect(() => {
    if (!user) return;

    // If we have a questionId in the URL, we should fetch that specific
    // question unless we already have it. If the initial fetch hasn't run
    // yet, still allow fetching the specific question so copy/paste URLs work.
    if (questionId) {
      if (!question || question.id !== Number(questionId)) {
        fetchQuestion();
      }
      return;
    }

    // If there is no questionId and we already have a question, do nothing;
    // otherwise the initial fetch effect will handle fetching the next
    // question when appropriate.
  }, [questionId, user, question]);

  return {
    question,
    setQuestion,
    isLoading,
    isGenerating,
    error,
    fetchQuestion,
    forceFetchNextQuestion,
    startPolling,
    stopPolling,
  };
}
