import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useQuestion } from '../contexts/useQuestion';
import { useQuestionUrlState } from '../hooks/useQuestionUrlState';
import {
  AnswerResponse as Feedback,
  postV1QuizAnswer,
  Question,
} from '../api/api';
import {
  Alert,
  Badge,
  Box,
  Button,
  Center,
  Container,
  Group,
  Paper,
  ScrollArea,
  Stack,
  Text,
} from '@mantine/core';
import LoadingSpinner from './LoadingSpinner';
import QuestionCard, { QuestionCardHandle } from './QuestionCard';
import { Chat } from './Chat';
import KeyboardShortcuts from './KeyboardShortcuts';
import QuestionPanel from './QuestionPanel';
import QuestionHeader from './QuestionHeader';
import { SUGGESTED_QUIZ_PROMPTS } from '../constants/prompts';
import { useQuestionFlow } from '../hooks/useQuestionFlow';
import { useTTS } from '../hooks/useTTS';

export type QuestionMode = 'quiz' | 'reading' | 'vocabulary';

interface Props {
  mode: QuestionMode;
}

export const QuestionPageBase: React.FC<Props> = ({ mode }) => {
  const { questionId } = useParams();
  const { user } = useAuth();

  const {
    setQuizQuestion,
    setReadingQuestion,
    quizFeedback,
    setQuizFeedback,
    readingFeedback,
    setReadingFeedback,
    // NOTE: selection/isSubmitted/showExplanation are intentionally kept local to
    // each page to avoid cross-page leakage of UI state.
  } = useQuestion();

  // Local UI state per page (isolation between pages)
  const [selectedAnswerLocal, setSelectedAnswerLocal] = useState<number | null>(
    null
  );
  const [isSubmittedLocal, setIsSubmittedLocal] = useState(false);
  const [showExplanationLocal, setShowExplanationLocal] = useState(false);

  const feedback = mode === 'quiz' ? quizFeedback : readingFeedback;
  const setFeedback = mode === 'quiz' ? setQuizFeedback : setReadingFeedback;

  const {
    question,
    setQuestion,
    isLoading,
    isGenerating,
    error,
    fetchQuestion,
    forceFetchNextQuestion,
  } = useQuestionFlow({ mode, questionId });

  // TTS hook for stopping audio on next question
  const { stopTTS } = useTTS();

  // URL state management for question navigation
  const { navigateToQuestion } = useQuestionUrlState({
    mode,
    question,
    isLoading,
  });

  const [isTransitioning, setIsTransitioning] = useState(false);
  const [isChatMaximized, setIsChatMaximized] = useState(false);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [isInputFocused, setIsInputFocused] = useState(false);
  const [isMarkKnownModalOpen, setIsMarkKnownModalOpen] = useState(false);
  const [isReportModalOpen, setIsReportModalOpen] = useState(false);
  const [isReportTextareaFocused, setIsReportTextareaFocused] = useState(false);
  const [maxOptions, setMaxOptions] = useState(0);
  const [workerError, setWorkerError] = useState<string | null>(null);
  const chatActionsRef = useRef<{
    clear: () => void;
    toggleMaximize: () => void;
  } | null>(null);
  const questionCardRef = useRef<QuestionCardHandle | null>(null);

  // Fetching is handled inside `useQuestionFlow` to avoid duplicate network
  // requests. Do not call `fetchQuestion` here.

  // Fetch worker status when generating to show any errors
  useEffect(() => {
    if (isGenerating) {
      const fetchWorkerStatus = async () => {
        try {
          const response = await fetch('/v1/quiz/worker-status', {
            credentials: 'include',
          });
          if (response.ok) {
            const data = await response.json();
            if (data.has_errors && data.last_error_details) {
              setWorkerError(data.last_error_details);
            } else {
              setWorkerError(null);
            }
          }
        } catch (err) {
          // Silently fail - worker status is not critical
        }
      };

      fetchWorkerStatus();
      // Poll every 5 seconds while generating
      const interval = setInterval(fetchWorkerStatus, 5000);
      return () => clearInterval(interval);
    } else {
      setWorkerError(null);
    }
  }, [isGenerating]);

  // Update the global QuestionContext when local question changes
  useEffect(() => {
    if (question) {
      if (mode === 'quiz') {
        setQuizQuestion(question);
      } else {
        setReadingQuestion(question);
      }
    } else {
      if (mode === 'quiz') {
        setQuizQuestion(null);
      } else {
        setReadingQuestion(null);
      }
    }
  }, [question, mode, setQuizQuestion, setReadingQuestion]);

  useEffect(() => {
    if (
      question &&
      user?.current_level &&
      question.level !== user.current_level
    ) {
      setQuestion(null);
      setFeedback(null);
      // reset local UI state
      setSelectedAnswerLocal(null);
      setIsSubmittedLocal(false);
      setShowExplanationLocal(false);
      fetchQuestion(true);
    }
  }, [question, user?.current_level, setQuestion, setFeedback, fetchQuestion]);

  useEffect(() => {
    if (questionId) {
      setFeedback(null);
      setSelectedAnswerLocal(null);
      setIsSubmittedLocal(false);
      setShowExplanationLocal(false);
    }
  }, [questionId, setFeedback]);

  const handleAnswerSubmit = async (
    qid: number,
    answerIndex: string
  ): Promise<Feedback> => {
    const response = await postV1QuizAnswer({
      question_id: qid,
      user_answer_index: parseInt(answerIndex, 10),
    });
    setFeedback(response);
    window.scrollTo({ top: 0, behavior: 'smooth' });
    return response;
  };

  useEffect(() => {
    setIsSubmittedLocal(!!feedback);
  }, [question?.id, feedback]);

  const handleAnswerSelect = (index: number) => {
    if (isSubmittedLocal) return;
    const options = (question as Question | null)?.content?.options;
    if (options && index >= 0 && index < options.length) {
      setSelectedAnswerLocal(index);
    }
  };

  const handleSubmit = async () => {
    if (
      selectedAnswerLocal === null ||
      selectedAnswerLocal === undefined ||
      !question?.id ||
      !question?.content?.options
    )
      return;
    setIsSubmittedLocal(true);
    await handleAnswerSubmit(question.id, selectedAnswerLocal.toString());
  };

  const startTransition = useCallback((after: () => void) => {
    setIsTransitioning(true);
    setTimeout(() => {
      after();
      setTimeout(() => setIsTransitioning(false), 300);
    }, 150);
  }, []);

  const clearQAState = useCallback(() => {
    setFeedback(null);
    setSelectedAnswerLocal(null);
    setIsSubmittedLocal(false);
    setShowExplanationLocal(false);
  }, [setFeedback]);

  const handleNextQuestion = useCallback(() => {
    // Stop TTS if playing
    stopTTS();

    startTransition(() => {
      clearQAState();
      // Clear question ID from URL to get next question
      navigateToQuestion(null);

      // Use the force fetch function to get a new question
      forceFetchNextQuestion();
      // Scroll will be handled by useEffect when question loads
    });
  }, [
    startTransition,
    clearQAState,
    navigateToQuestion,
    forceFetchNextQuestion,
    stopTTS,
  ]);

  const handleNewQuestion = useCallback(() => {
    // Stop TTS if playing
    stopTTS();

    startTransition(() => {
      clearQAState();
      // Clear question ID from URL to get next question
      navigateToQuestion(null);
      // Use the force fetch function to get a new question
      forceFetchNextQuestion();
      // Scroll will be handled by useEffect when question loads
    });
  }, [
    startTransition,
    clearQAState,
    navigateToQuestion,
    forceFetchNextQuestion,
    stopTTS,
  ]);

  // Scroll to top when a new question is loaded (desktop)
  useEffect(() => {
    // Only scroll when we have a question id and loading is complete
    if (!question?.id || isLoading || isGenerating) return;

    // Use multiple requestAnimationFrames to ensure DOM has fully updated
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        try {
          // Scroll window/document
          window.scrollTo({ top: 0, behavior: 'smooth' });
          document.documentElement.scrollTo?.({ top: 0, behavior: 'smooth' });
          document.body.scrollTo?.({ top: 0, behavior: 'smooth' });

          // Fallback: scroll first question element into view
          const firstQuestionElement = document.querySelector(
            '[data-testid="quiz-page-container"], [data-testid="reading-comprehension-page-container"]'
          );
          if (firstQuestionElement) {
            firstQuestionElement.scrollIntoView({
              behavior: 'smooth',
              block: 'start',
            });
          }
        } catch {
          // ignore (e.g., server-side rendering or environments without scrollTo)
        }
      });
    });
  }, [question?.id, isLoading, isGenerating]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        const active = document.activeElement as HTMLElement | null;
        if (
          active &&
          (active.tagName === 'INPUT' ||
            active.tagName === 'TEXTAREA' ||
            active.isContentEditable)
        ) {
          active.blur();
          setIsInputFocused(false);
        }
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleReportIssue = () => {
    questionCardRef.current?.openReport();
  };

  const handleMarkKnown = () => {
    questionCardRef.current?.openMarkKnown();
  };

  if (isLoading) {
    return (
      <Center h={300}>
        <Stack align='center' gap='xs'>
          <LoadingSpinner />
          <Text c='dimmed'>Loading your next question...</Text>
        </Stack>
      </Center>
    );
  }

  if (isGenerating) {
    return (
      <Center h='100vh' style={{ flexDirection: 'column' }}>
        <Stack align='center' gap='xl' style={{ maxWidth: 600, width: '100%', padding: '2rem' }}>
          <Stack align='center' gap='xs'>
            <LoadingSpinner />
            <Text>
              Generating your personalized{' '}
              {mode === 'reading' ? 'reading comprehension ' : ''}question...
            </Text>
            <Text size='sm' c='dimmed'>
              This may take a moment
            </Text>
          </Stack>
          {workerError && (
            <Alert
              color='red'
              title='Generation Error'
              style={{ width: '100%', marginTop: '3rem' }}
            >
              <Stack gap='xs'>
                <Text size='sm' fw={500}>
                  Worker encountered errors during question generation
                </Text>
                <ScrollArea.Autosize mah={200}>
                  <Text
                    size='xs'
                    style={{
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      fontFamily: 'monospace',
                    }}
                  >
                    {workerError}
                  </Text>
                </ScrollArea.Autosize>
              </Stack>
            </Alert>
          )}
        </Stack>
      </Center>
    );
  }

  if (error && !isGenerating) {
    // Special handling for AI disabled error
    if (error === 'Enable AI in settings to generate questions') {
      return (
        <Center h='100vh'>
          <Paper
            p='xl'
            radius='md'
            withBorder
            shadow='sm'
            style={{ minWidth: 340, textAlign: 'center' }}
          >
            <Stack align='center' gap='lg'>
              <Text fw={600} size='lg'>
                No questions available.
              </Text>
              <Text c='dimmed' size='md'>
                Enable AI in your{' '}
                <Text
                  component={Link}
                  to='/settings'
                  c='primary'
                  style={{ textDecoration: 'underline' }}
                >
                  settings
                </Text>{' '}
                to generate new questions.
              </Text>
              <Button
                component={Link}
                to='/settings'
                variant='filled'
                size='md'
                style={{ marginTop: 12 }}
              >
                Go to Settings
              </Button>
            </Stack>
          </Paper>
        </Center>
      );
    }
    return (
      <Center h={300}>
        <Paper p='lg' radius='md' withBorder shadow='sm'>
          <Stack align='center' gap='md'>
            <Text c='var(--mantine-color-error)' fw={500}>
              {error}
            </Text>
            <Button
              onClick={() => fetchQuestion(true)}
              variant='filled'
              color='primary'
            >
              Try Again
            </Button>
          </Stack>
        </Paper>
      </Center>
    );
  }

  if (!question) {
    return (
      <Center h={300}>
        <Text>No question available.</Text>
      </Center>
    );
  }

  const explanationAvailable = isSubmittedLocal && !!feedback?.explanation;

  return (
    <Container
      size='lg'
      py='xl'
      data-testid={
        mode === 'quiz'
          ? 'quiz-page-container'
          : 'reading-comprehension-page-container'
      }
      style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}
    >
      <Stack gap='xl' style={{ flex: 1, minHeight: 0 }}>
        <Group justify='flex-end' align='center'>
          <Button
            onClick={handleNewQuestion}
            variant='filled'
            size='md'
            disabled={isLoading || isGenerating || isTransitioning}
          >
            New Question{' '}
            <Badge ml={6} size='xs' color='gray' variant='filled' radius='sm'>
              N
            </Badge>
          </Button>
        </Group>

        <Box style={{ marginBottom: 0 }}>
          {question && (
            <QuestionHeader
              question={question}
              timezone={user?.timezone}
              showConfidence={mode === 'quiz'}
            />
          )}
          <QuestionPanel
            loading={isLoading}
            generating={isGenerating}
            transitioning={isTransitioning}
          >
            <QuestionCard
              ref={questionCardRef}
              question={question}
              onAnswer={handleAnswerSubmit}
              onNext={handleNextQuestion}
              feedback={feedback}
              selectedAnswer={selectedAnswerLocal}
              selectedAnswerQuestionId={question.id}
              onAnswerSelect={handleAnswerSelect}
              showExplanation={showExplanationLocal}
              setShowExplanation={setShowExplanationLocal}
              onMarkKnownModalChange={setIsMarkKnownModalOpen}
              onReportModalChange={setIsReportModalOpen}
              onReportTextareaFocusChange={setIsReportTextareaFocused}
              onShuffledOptionsChange={setMaxOptions}
            />
          </QuestionPanel>
        </Box>

        <Box>
          <Chat
            question={question}
            answerContext={feedback || undefined}
            isMaximized={isChatMaximized}
            setIsMaximized={setIsChatMaximized}
            showSuggestions={showSuggestions}
            setShowSuggestions={setShowSuggestions}
            onInputFocus={() => setIsInputFocused(true)}
            onInputBlur={() => setIsInputFocused(false)}
            onRegisterActions={({ clear, toggleMaximize }) => {
              chatActionsRef.current = { clear, toggleMaximize };
            }}
          />
        </Box>

        {!isChatMaximized && (
          <Box>
            <KeyboardShortcuts
              onAnswerSelect={handleAnswerSelect}
              onSubmit={handleSubmit}
              onNextQuestion={handleNextQuestion}
              onNewQuestion={handleNewQuestion}
              onToggleTTS={() => questionCardRef.current?.toggleTTS?.()}
              isSubmitted={isSubmittedLocal}
              hasSelectedAnswer={
                selectedAnswerLocal !== null &&
                selectedAnswerLocal !== undefined
              }
              maxOptions={maxOptions}
              onToggleExplanation={() => setShowExplanationLocal(prev => !prev)}
              explanationAvailable={explanationAvailable}
              ttsAvailable={
                !!(
                  question?.type === 'reading_comprehension' &&
                  question?.content?.passage
                )
              }
              onReportIssue={handleReportIssue}
              onMarkKnown={handleMarkKnown}
              onClearChat={() => chatActionsRef.current?.clear?.()}
              onToggleMaximize={() =>
                chatActionsRef.current?.toggleMaximize?.()
              }
              isQuickSuggestionsOpen={showSuggestions}
              quickSuggestionsCount={SUGGESTED_QUIZ_PROMPTS.length}
              isInputFocused={isInputFocused}
              isMarkKnownModalOpen={isMarkKnownModalOpen}
              isReportModalOpen={isReportModalOpen}
              isReportTextareaFocused={isReportTextareaFocused}
            />
          </Box>
        )}
      </Stack>
    </Container>
  );
};

export default QuestionPageBase;
