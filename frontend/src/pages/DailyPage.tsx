import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useDailyQuestions } from '../hooks/useDailyQuestions';
import { useQuestion } from '../contexts/useQuestion';
import { AnswerResponse as Feedback } from '../api/api';

import { showNotificationWithClean } from '../notifications';
import {
  Container,
  Stack,
  Text,
  Center,
  Paper,
  Box,
  Button,
  Group,
  Badge,
  Title,
  Progress,
  Grid,
} from '@mantine/core';
import { ChevronLeft, ChevronRight, Clock as HistoryIcon } from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';
import QuestionCard, { QuestionCardHandle } from '../components/QuestionCard';
import { Chat } from '../components/Chat';
import KeyboardShortcuts from '../components/KeyboardShortcuts';

import DailyDatePicker from '../components/DailyDatePicker';
import DailyCompletionScreen from '../components/DailyCompletionScreen';
import { QuestionHistoryModal } from '../components/QuestionHistoryModal';
import { SUGGESTED_QUIZ_PROMPTS } from '../constants/prompts';
import logger from '../utils/logger';
import QuestionPanel from '../components/QuestionPanel';
import QuestionHeader from '../components/QuestionHeader';

// Reuse shared prompts
const suggestedPrompts = SUGGESTED_QUIZ_PROMPTS;

const DailyPage: React.FC = () => {
  const { date } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const chatActionsRef = useRef<{
    clear: () => void;
    toggleMaximize: () => void;
  } | null>(null);
  const questionCardRef = useRef<QuestionCardHandle | null>(null);

  // Daily questions hook
  const {
    selectedDate,
    setSelectedDate,
    questions,
    progress,
    availableDates,

    currentQuestionIndex,
    isLoading,
    goToNextQuestion,
    goToPreviousQuestion,
    currentQuestion,
    hasNextQuestion,
    hasPreviousQuestion,
    isAllCompleted,
    submitAnswer,
    getQuestionHistory,
    questionHistory,
    isHistoryLoading,
  } = useDailyQuestions();

  // Keep feedback local to DailyPage to avoid affecting QuizPage or other pages
  // (previously this used quizFeedback from shared context which leaked state).
  useQuestion();
  const [feedbackLocal, setFeedbackLocal] = useState<Feedback | null>(null);

  // Local selection/UI state (isolated per-page)
  const [selectedAnswerLocal, setSelectedAnswerLocal] = useState<number | null>(
    null
  );
  const [isSubmittedLocal, setIsSubmittedLocal] = useState(false);
  const [showExplanationLocal, setShowExplanationLocal] = useState(false);

  // Local state
  const [isAnswerSubmitting, setIsAnswerSubmitting] = useState(false);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const [isChatMaximized, setIsChatMaximized] = useState(false);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [isInputFocused, setIsInputFocused] = useState(false);
  const [isMarkKnownModalOpen, setIsMarkKnownModalOpen] = useState(false);
  const [isReportModalOpen, setIsReportModalOpen] = useState(false);
  const [isReportTextareaFocused, setIsReportTextareaFocused] = useState(false);
  const [showCompletionScreen, setShowCompletionScreen] = useState(false);
  const [selectedAnswerQuestionIdLocal, setSelectedAnswerQuestionIdLocal] =
    useState<number | null>(null);
  const [maxOptions, setMaxOptions] = useState(0);
  const [isHistoryModalOpen, setIsHistoryModalOpen] = useState(false);

  // Set selected date from URL parameter
  useEffect(() => {
    if (date) {
      setSelectedDate(date);
    }
  }, [date, setSelectedDate]);

  // Update URL when date changes
  useEffect(() => {
    if (selectedDate !== date) {
      navigate(`/daily/${selectedDate}`, { replace: true });
    }
  }, [selectedDate, date, navigate]);

  // Reset question-related state when question changes
  useEffect(() => {
    setSelectedAnswerLocal(null);
    setSelectedAnswerQuestionIdLocal(null);
    setIsSubmittedLocal(false);
    setShowExplanationLocal(false);
    setFeedbackLocal(null);
    setShowCompletionScreen(false);
  }, [currentQuestionIndex, setSelectedAnswerQuestionIdLocal]);

  // Set feedback for completed questions
  useEffect(() => {
    if (
      currentQuestion?.is_completed &&
      currentQuestion.user_answer_index !== null
    ) {
      // Convert daily question data to feedback format
      const feedbackData: Feedback = {
        user_answer_index: currentQuestion.user_answer_index,
        correct_answer_index: currentQuestion.question.correct_answer,
        is_correct:
          currentQuestion.user_answer_index ===
          currentQuestion.question.correct_answer,
        user_answer:
          currentQuestion.question.content?.options?.[
            currentQuestion.user_answer_index!
          ] || undefined,
        explanation: currentQuestion.question.explanation,
      };

      setFeedbackLocal(feedbackData);
      // Do NOT set selectedAnswer here; let QuestionCard map original -> shuffled
      setIsSubmittedLocal(true);
      setShowExplanationLocal(true);
    }
  }, [currentQuestion]);

  const handleAnswerSelect = useCallback(
    (answerIndex: number) => {
      // Allow updating selectedAnswer even when isSubmitted is true
      // This is needed for completed questions where we need to convert
      // from original index to shuffled index
      setSelectedAnswerLocal(answerIndex);
      if (currentQuestion?.question?.id != null) {
        setSelectedAnswerQuestionIdLocal(currentQuestion.question.id);
      }
    },
    [currentQuestion]
  );

  const handleAnswerSubmit = useCallback(
    async (questionId: number, answerIndex: string): Promise<Feedback> => {
      if (!currentQuestion || isSubmittedLocal || isAnswerSubmitting) {
        throw new Error('Cannot submit answer: invalid state');
      }

      try {
        setIsAnswerSubmitting(true);

        // Use the hook's submitAnswer function which properly handles cache invalidation
        const response = await submitAnswer(
          questionId,
          parseInt(answerIndex, 10)
        );

        // Set the feedback state so the QuestionCard can display the result
        setFeedbackLocal(response);
        setIsSubmittedLocal(true);
        setShowExplanationLocal(true);

        // Scroll to top after submitting answer
        window.scrollTo({
          top: 0,
          behavior: 'smooth',
        });

        return response;
      } catch (error) {
        logger.error('Failed to submit answer:', error);
        showNotificationWithClean({
          title: 'Error',
          message: 'Failed to submit answer. Please try again.',
          color: 'red',
        });
        throw error;
      } finally {
        setIsAnswerSubmitting(false);
      }
    },
    [
      currentQuestion,
      isSubmittedLocal,
      isAnswerSubmitting,
      submitAnswer,
      setFeedbackLocal,
      setIsSubmittedLocal,
      setShowExplanationLocal,
    ]
  );

  const handleNextQuestion = useCallback(() => {
    setIsTransitioning(true);
    setTimeout(() => {
      goToNextQuestion();
      setIsTransitioning(false);
      // Scroll to top after new question is loaded
      window.scrollTo({
        top: 0,
        behavior: 'smooth',
      });
    }, 150);
  }, [goToNextQuestion]);

  const handlePreviousQuestion = useCallback(() => {
    setIsTransitioning(true);
    setTimeout(() => {
      goToPreviousQuestion();
      setIsTransitioning(false);
    }, 150);
  }, [goToPreviousQuestion]);

  const handleDateSelect = useCallback(
    (newDate: string | null) => {
      if (newDate) {
        setSelectedDate(newDate);
      }
    },
    [setSelectedDate]
  );

  const handleSubmit = useCallback(() => {
    if (
      selectedAnswerLocal !== null &&
      selectedAnswerLocal !== undefined &&
      !isSubmittedLocal &&
      currentQuestion
    ) {
      handleAnswerSubmit(
        currentQuestion.question_id,
        selectedAnswerLocal.toString()
      );
    }
  }, [
    selectedAnswerLocal,
    isSubmittedLocal,
    currentQuestion,
    handleAnswerSubmit,
  ]);

  const handleNext = useCallback(() => {
    handleNextQuestion();
  }, [handleNextQuestion]);

  const handleReportIssue = useCallback(() => {
    questionCardRef.current?.openReport();
  }, []);

  const handleMarkKnown = useCallback(() => {
    questionCardRef.current?.openMarkKnown();
  }, []);

  if (!user) {
    return (
      <Center h={300}>
        <Text>Please log in to access daily questions.</Text>
      </Center>
    );
  }

  if (isLoading) {
    return (
      <Center h={300}>
        <LoadingSpinner />
      </Center>
    );
  }

  if (!questions || questions.length === 0) {
    return (
      <Container size='lg' py='xl'>
        <Stack gap='xl' align='center'>
          <Title order={2}>No Daily Questions Available</Title>
          <Text size='lg' ta='center' c='dimmed'>
            {selectedDate === new Date().toISOString().split('T')[0]
              ? "Today's questions haven't been assigned yet. Check back later!"
              : 'No questions were assigned for this date.'}
          </Text>
          <DailyDatePicker
            selectedDate={selectedDate}
            onDateSelect={handleDateSelect}
            availableDates={availableDates}
            progressData={progress ? { [selectedDate]: progress } : {}}
            maxDate={new Date()}
            size='sm'
            style={{ width: '250px' }}
            clearable
            hideOutsideDates
            withCellSpacing={false}
            firstDayOfWeek={1}
          />
        </Stack>
      </Container>
    );
  }

  const explanationAvailable = isSubmittedLocal && !!feedbackLocal?.explanation;

  // Prefer per-assignment per-user stats for display in QuestionCard when available
  const questionForCard = currentQuestion?.question
    ? {
        ...currentQuestion.question,
        total_responses:
          currentQuestion.user_total_responses ??
          currentQuestion.question.total_responses,
        correct_count:
          currentQuestion.user_correct_count ??
          currentQuestion.question.correct_count,
        incorrect_count:
          currentQuestion.user_incorrect_count ??
          currentQuestion.question.incorrect_count,
      }
    : currentQuestion?.question;

  return (
    <Container
      size='lg'
      py='xl'
      data-testid='daily-page-container'
      style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}
    >
      <Stack gap='xl' style={{ flex: 1, minHeight: 0 }}>
        {/* Header with progress and navigation */}
        <Paper p='md' withBorder radius='md'>
          <Grid>
            <Grid.Col span={{ base: 12, md: 6 }}>
              <Stack gap='xs'>
                <Group justify='space-between' align='center'>
                  <Title order={3}>Daily Questions</Title>
                  {/* moved date picker + history to right column for layout */}
                </Group>
                {progress && (
                  <Group justify='flex-start' align='center'>
                    <Text size='sm' c='dimmed'>
                      Progress: {progress.completed}/{progress.total} completed
                    </Text>
                  </Group>
                )}
                {/* progress moved below to span full header width */}
              </Stack>
            </Grid.Col>
            <Grid.Col span={{ base: 12, md: 6 }}>
              {/* Right side: date picker and history, aligned to the far right */}
              <Group justify='flex-end' align='center' mb='sm'>
                <DailyDatePicker
                  selectedDate={selectedDate}
                  onDateSelect={handleDateSelect}
                  availableDates={availableDates}
                  progressData={progress ? { [selectedDate]: progress } : {}}
                  maxDate={new Date()}
                  size='sm'
                  style={{ width: '200px' }}
                  clearable
                  hideOutsideDates
                  withCellSpacing={false}
                  firstDayOfWeek={1}
                />
                <Button
                  size='xs'
                  variant='subtle'
                  onClick={() =>
                    setSelectedDate(new Date().toISOString().split('T')[0])
                  }
                  ml={8}
                >
                  Today
                </Button>
                {progress && currentQuestion && (
                  <Button
                    leftSection={<HistoryIcon size={16} />}
                    variant='light'
                    size='xs'
                    onClick={() => {
                      if (currentQuestion.question_id) {
                        getQuestionHistory(currentQuestion.question_id);
                        setIsHistoryModalOpen(true);
                      }
                    }}
                    disabled={!currentQuestion.question_id}
                    ml={8}
                  >
                    <Box
                      style={{
                        display: 'inline-flex',
                        gap: 6,
                        alignItems: 'center',
                      }}
                    >
                      <Text size='xs'>History</Text>
                      <Badge size='xs' color='gray' variant='light'>
                        H
                      </Badge>
                    </Box>
                  </Button>
                )}
              </Group>
              {isAllCompleted && (
                <Group
                  justify='flex-end'
                  align='center'
                  h='100%'
                  mb='lg'
                  style={{ position: 'relative', zIndex: 2, marginBottom: 24 }}
                >
                  <Button
                    leftSection={<ChevronLeft size={16} />}
                    variant='light'
                    onClick={handlePreviousQuestion}
                    disabled={!hasPreviousQuestion || isTransitioning}
                    size='sm'
                  >
                    Previous{' '}
                    <Badge
                      ml={6}
                      size='xs'
                      color='gray'
                      variant='filled'
                      radius='sm'
                    >
                      ←
                    </Badge>
                  </Button>
                  <Button
                    rightSection={<ChevronRight size={16} />}
                    variant='light'
                    onClick={handleNextQuestion}
                    disabled={!hasNextQuestion || isTransitioning}
                    size='sm'
                  >
                    Next{' '}
                    <Badge
                      ml={6}
                      size='xs'
                      color='gray'
                      variant='filled'
                      radius='sm'
                    >
                      →
                    </Badge>
                  </Button>
                </Group>
              )}
            </Grid.Col>
          </Grid>

          {/* Full-width progress bar inside header */}
          {progress && (
            <Box style={{ marginTop: 24, position: 'relative', zIndex: 1 }}>
              <Progress
                value={(progress.completed / progress.total) * 100}
                size='md'
                radius='xl'
                color={progress.completed === progress.total ? 'green' : 'blue'}
              />
            </Box>
          )}
        </Paper>

        {/* Question content */}
        <Box style={{ marginBottom: 0 }}>
          {/* Question header with tags/timestamps/confidence */}
          {currentQuestion?.question && (
            <QuestionHeader
              question={currentQuestion.question}
              timezone={user?.timezone}
              showConfidence
            />
          )}

          {/* Question completion status */}
          <Group justify='flex-start' align='center' mb='md'>
            {currentQuestion?.is_completed ? (
              <Badge color='green' variant='filled' size='lg'>
                ✓ Completed
              </Badge>
            ) : (
              <Badge color='gray' variant='light' size='lg'>
                Incomplete
              </Badge>
            )}
          </Group>

          <QuestionPanel
            loading={isLoading}
            transitioning={isTransitioning}
            maxHeight='85vh'
          >
            {showCompletionScreen ? (
              <DailyCompletionScreen
                selectedDate={selectedDate}
                onDateSelect={handleDateSelect}
                availableDates={availableDates}
                progressData={progress ? { [selectedDate]: progress } : {}}
              />
            ) : !currentQuestion ? (
              <Center w='100%' h='100%'>
                <Text>No questions available for this date.</Text>
              </Center>
            ) : (
              <QuestionCard
                ref={questionCardRef}
                question={questionForCard}
                onAnswer={handleAnswerSubmit}
                onNext={handleNextQuestion}
                feedback={currentQuestion.is_completed ? feedbackLocal : null}
                selectedAnswer={selectedAnswerLocal}
                selectedAnswerQuestionId={selectedAnswerQuestionIdLocal}
                groupScopeId={currentQuestion.id}
                onAnswerSelect={handleAnswerSelect}
                showExplanation={showExplanationLocal}
                setShowExplanation={setShowExplanationLocal}
                onMarkKnownModalChange={setIsMarkKnownModalOpen}
                onReportModalChange={setIsReportModalOpen}
                onReportTextareaFocusChange={setIsReportTextareaFocused}
                isLastQuestion={!hasNextQuestion}
                // Only treat the card as fully read-only when the entire day's
                // questions are completed. If this question is completed but
                // there are still unanswered questions in progress, allow the
                // Next Question button to be shown.
                isReadOnly={!!currentQuestion.is_completed && isAllCompleted}
                onShuffledOptionsChange={setMaxOptions}
              />
            )}
          </QuestionPanel>
        </Box>

        {/* Chat section */}
        <Box>
          {currentQuestion?.question && (
            <Chat
              question={currentQuestion.question}
              answerContext={feedbackLocal || undefined}
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
          )}
        </Box>

        {/* Keyboard shortcuts */}
        {!isChatMaximized && (
          <Box>
            <KeyboardShortcuts
              onAnswerSelect={handleAnswerSelect}
              onSubmit={handleSubmit}
              onNextQuestion={handleNext}
              onNewQuestion={() => {}} // Not applicable for daily questions
              onPreviousQuestion={handlePreviousQuestion}
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
                  currentQuestion?.question?.type === 'reading_comprehension' &&
                  currentQuestion?.question?.content?.passage
                )
              }
              onReportIssue={handleReportIssue}
              onMarkKnown={handleMarkKnown}
              onClearChat={() => chatActionsRef.current?.clear?.()}
              onToggleMaximize={() =>
                chatActionsRef.current?.toggleMaximize?.()
              }
              isQuickSuggestionsOpen={showSuggestions}
              quickSuggestionsCount={suggestedPrompts.length}
              isInputFocused={isInputFocused}
              isReportTextareaFocused={isReportTextareaFocused}
              isMarkKnownModalOpen={isMarkKnownModalOpen}
              isReportModalOpen={isReportModalOpen}
              isHistoryOpen={isHistoryModalOpen}
              onShowHistory={() => {
                if (currentQuestion?.question_id) {
                  getQuestionHistory(currentQuestion.question_id);
                  setIsHistoryModalOpen(true);
                }
              }}
              onHideHistory={() => setIsHistoryModalOpen(false)}
              enablePrevNextArrows={isAllCompleted}
            />
          </Box>
        )}
      </Stack>

      {/* Question History Modal */}
      <QuestionHistoryModal
        opened={isHistoryModalOpen}
        onClose={() => setIsHistoryModalOpen(false)}
        history={questionHistory || []}
        isLoading={isHistoryLoading}
        questionText={currentQuestion?.question?.content?.question}
      />
    </Container>
  );
};

export default DailyPage;
