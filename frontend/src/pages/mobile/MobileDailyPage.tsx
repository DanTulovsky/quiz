import React, { useCallback, useState, useEffect, useRef } from 'react';
import { useParams } from 'react-router-dom';
import {
  Container,
  Paper,
  Stack,
  Text,
  Button,
  Group,
  Badge,
  ActionIcon,
  Alert,
  Loader,
  Center,
  Progress,
  Tooltip,
  LoadingOverlay,
  Box,
  Modal,
  Textarea,
} from '@mantine/core';
import { useAuth } from '../../hooks/useAuth';
import { IconCheck, IconX } from '@tabler/icons-react';
import { Volume2, VolumeX } from 'lucide-react';
import { useDailyQuestions } from '../../hooks/useDailyQuestions';
import DailyDatePicker from '../../components/DailyDatePicker';
import { useMantineTheme } from '@mantine/core';
import { useTTS } from '../../hooks/useTTS';
import { defaultVoiceForLanguage } from '../../utils/tts';
import { splitIntoParagraphs } from '../../utils/reading';
import {
  usePostV1QuizQuestionIdReport,
  usePostV1QuizQuestionIdMarkKnown,
} from '../../api/api';
import { showNotificationWithClean } from '../../notifications';

// Highlight occurrences of target word within a sentence with bold styling.
function highlightTargetWord(sentence: string, target: string) {
  if (!target) return sentence;
  const regex = new RegExp(
    `\\b${target.replace(/[.*+?^${}()|[\\]\\]/g, '\\$&')}\\b`,
    'gi'
  );
  const parts = sentence.split(regex);
  const matches = sentence.match(regex);
  if (!matches) return sentence;
  const result: React.ReactNode[] = [];
  for (let i = 0; i < parts.length; i++) {
    result.push(parts[i]);
    if (i < matches.length) {
      result.push(
        <strong key={i} style={{ color: '#1976d2', fontWeight: 700 }}>
          {matches[i]}
        </strong>
      );
    }
  }
  return result;
}

const MobileDailyPage: React.FC = () => {
  const { date: dateParam } = useParams();

  const {
    selectedDate,
    setSelectedDate,
    currentQuestion,
    submitAnswer,
    goToNextQuestion,
    hasNextQuestion,
    isLoading,
    isSubmittingAnswer,
    currentQuestionIndex,
    questions,
    availableDates,
    hasPreviousQuestion,
    goToPreviousQuestion,
    isAllCompleted,
  } = useDailyQuestions();

  // Local UI state
  const [selectedAnswerLocal, setSelectedAnswerLocal] = useState<number | null>(
    null
  );
  const [isSubmittedLocal, setIsSubmittedLocal] = useState(false);
  const [feedbackLocal, setFeedbackLocal] = useState<{
    is_correct: boolean;
    correct_answer_index: number;
    explanation?: string;
  } | null>(null);

  // Refs for buttons to enable scrolling
  const submitButtonRef = useRef<HTMLButtonElement>(null);
  const nextButtonRef = useRef<HTMLButtonElement>(null);

  // TTS state for reading comprehension passages
  const {
    isLoading: isTTSLoading,
    isPlaying: isTTSPlaying,
    playTTS,
    stopTTS,
  } = useTTS();

  // Set date from URL param
  useEffect(() => {
    if (dateParam && dateParam !== selectedDate) {
      setSelectedDate(dateParam);
    }
  }, [dateParam, selectedDate, setSelectedDate]);

  // Initialize local state when the current question changes. If the question
  // is already completed (e.g. user answered it on desktop first), pre-populate
  // the local state so the mobile UI reflects the completed status instead of
  // showing an empty unanswered question.
  useEffect(() => {
    if (!currentQuestion) {
      setSelectedAnswerLocal(null);
      setIsSubmittedLocal(false);
      setFeedbackLocal(null);
      return;
    }

    if (currentQuestion.is_completed) {
      // Pre-fill with the user ’s previous answer so the radio buttons and UI
      // reflect the completed state.
      const prevAnswerIdx = currentQuestion.user_answer_index ?? null;

      setSelectedAnswerLocal(prevAnswerIdx);
      setIsSubmittedLocal(true);

      if (
        prevAnswerIdx !== null &&
        typeof currentQuestion.question.correct_answer === 'number'
      ) {
        setFeedbackLocal({
          is_correct: prevAnswerIdx === currentQuestion.question.correct_answer,
          correct_answer_index: currentQuestion.question.correct_answer,
          explanation: currentQuestion.question.explanation,
        });
      } else {
        setFeedbackLocal(null);
      }
    } else {
      // Not completed – reset to blank state
      setSelectedAnswerLocal(null);
      setIsSubmittedLocal(false);
      setFeedbackLocal(null);
    }
  }, [currentQuestion?.id, currentQuestion?.is_completed]);

  // Scroll to top when a new daily question is loaded (mobile)
  useEffect(() => {
    if (!currentQuestion?.id) return;
    try {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    } catch {
      // ignore in non-browser environments
    }
  }, [currentQuestion?.id]);

  // TTS handler functions
  const handleTTSPlay = async (text: string) => {
    if (!text) return;

    // Determine the best voice: default for question.language -> fallback to 'echo'
    const finalVoice =
      defaultVoiceForLanguage(currentQuestion?.question.language || 'en') ||
      'echo';

    await playTTS(text, finalVoice);
  };

  const handleTTSStop = () => {
    stopTTS();
  };

  // Function to scroll to submit button (mobile only)
  const scrollToSubmitButton = useCallback(() => {
    if (submitButtonRef.current) {
      submitButtonRef.current.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
      });
    }
  }, []);

  const { isAuthenticated, user } = useAuth();

  const handleReport = async () => {
    if (isReported || reportMutation.isPending || !currentQuestion?.question_id)
      return;

    if (!isAuthenticated) {
      showNotificationWithClean({
        title: 'Error',
        message: 'You must be logged in to report a question.',
        color: 'red',
      });
      return;
    }

    setShowReportModal(true);
  };

  // Reporting & mark-known state (mobile parity)
  const [isReported, setIsReported] = useState(false);
  const [showMarkKnownModal, setShowMarkKnownModal] = useState(false);
  const [showReportModal, setShowReportModal] = useState(false);
  const [reportReason, setReportReason] = useState('');
  const [isReporting, setIsReporting] = useState(false);
  const [confidenceLevel, setConfidenceLevel] = useState<number | null>(null);
  const [isMarkingKnown, setIsMarkingKnown] = useState(false);

  const reportMutation = usePostV1QuizQuestionIdReport({
    mutation: {
      onSuccess: () => {
        setIsReported(true);
        setShowReportModal(false);
        setReportReason('');
        showNotificationWithClean({
          title: 'Success',
          message:
            'Question reported successfully. Thank you for your feedback!',
          color: 'green',
        });
      },
      onError: error => {
        showNotificationWithClean({
          title: 'Error',
          message: error?.error || 'Failed to report question.',
          color: 'red',
        });
      },
    },
  });

  const markKnownMutation = usePostV1QuizQuestionIdMarkKnown({
    mutation: {
      onSuccess: () => {
        setShowMarkKnownModal(false);
        const confidence = confidenceLevel;
        setConfidenceLevel(null);
        let message = 'Preference saved.';
        if (confidence === 1)
          message =
            'Saved with low confidence. You will see this question more often.';
        if (confidence === 2)
          message =
            'Saved with some confidence. You will see this question a bit more often.';
        if (confidence === 3)
          message =
            'Saved with neutral confidence. No change to how often you will see this question.';
        if (confidence === 4)
          message =
            'Saved with high confidence. You will see this question less often.';
        if (confidence === 5)
          message =
            'Saved with complete confidence. You will rarely see this question.';
        showNotificationWithClean({
          title: 'Success',
          message,
          color: 'green',
        });
      },
      onError: error => {
        showNotificationWithClean({
          title: 'Error',
          message: error?.error || 'Failed to mark question as known.',
          color: 'red',
        });
      },
    },
  });

  // Handle answer submission
  const handleAnswerSubmit = useCallback(async () => {
    if (!currentQuestion || selectedAnswerLocal === null) return;

    setIsSubmittedLocal(true);

    try {
      const response = await submitAnswer(
        currentQuestion.question_id,
        selectedAnswerLocal
      );
      setFeedbackLocal({
        is_correct: response.is_correct ?? false,
        correct_answer_index: response.correct_answer_index ?? 0,
        explanation: response.explanation,
      });

      // Scroll to next question button after feedback is shown
      setTimeout(() => {
        if (nextButtonRef.current) {
          nextButtonRef.current.scrollIntoView({
            behavior: 'smooth',
            block: 'center',
          });
        }
      }, 300); // Delay to allow feedback animation to complete
    } catch (error) {
      console.error('Failed to submit answer:', error);
    }
  }, [currentQuestion, selectedAnswerLocal, submitAnswer]);

  // Handle next question
  const handleNextQuestion = useCallback(() => {
    setSelectedAnswerLocal(null);
    setIsSubmittedLocal(false);
    setFeedbackLocal(null);
    goToNextQuestion();
  }, [goToNextQuestion]);

  // Handle previous question (only used after completion)
  const handlePreviousQuestion = useCallback(() => {
    goToPreviousQuestion();
  }, [goToPreviousQuestion]);

  if (isLoading && !currentQuestion) {
    return (
      <Center h='100%'>
        <Loader size='lg' />
      </Center>
    );
  }

  if (!currentQuestion || !questions) {
    return (
      <Center h='100%'>
        <Text>No daily questions available</Text>
      </Center>
    );
  }

  const canSubmit = selectedAnswerLocal !== null && !isSubmittedLocal;
  const showFeedback = isSubmittedLocal && feedbackLocal;

  const progressValue =
    questions.length > 0
      ? ((currentQuestionIndex + 1) / questions.length) * 100
      : 0;

  const theme = useMantineTheme();

  return (
    <Container size='sm'>
      <Stack gap='md'>
        {/* Daily Progress Header */}
        <Paper p='md' radius='md' withBorder>
          <Stack gap='xs'>
            {/* Top row: Daily Challenge (left) and language-level (right) */}
            <Group justify='space-between' align='center'>
              <Badge variant='light' color='orange'>
                Daily Challenge
              </Badge>

              {/* Language + level badge right aligned */}
              <Badge
                variant='outline'
                color='blue'
                data-testid='header-language-level-badge'
              >
                {currentQuestion?.question.language} -{' '}
                {user?.current_level || currentQuestion?.question.level}
              </Badge>
            </Group>

            {/* Second row: date picker and question counter */}
            <Group gap='sm' align='center' justify='space-between'>
              {/* Direct date picker - no intermediate popup */}
              <DailyDatePicker
                dropdownType='modal'
                selectedDate={selectedDate}
                onDateSelect={date => {
                  if (date) {
                    setSelectedDate(date);
                  }
                }}
                availableDates={availableDates}
                maxDate={new Date()}
                size='xs'
                clearable={false}
                hideOutsideDates
                withCellSpacing={false}
                firstDayOfWeek={1}
                style={{ width: '180px' }}
              />

              <Badge variant='outline'>
                {currentQuestionIndex + 1} of {questions.length}
              </Badge>
            </Group>
            {/* Removed redundant Daily Questions label */}
            <Progress value={progressValue} color='orange' />
          </Stack>
        </Paper>

        {/* Current Question */}
        <Paper p='md' radius='md' withBorder>
          <Stack gap='md'>
            <Group justify='space-between'>
              {/* TTS button for reading comprehension */}
              {currentQuestion.question.type === 'reading_comprehension' &&
                currentQuestion.question.content?.passage && (
                  <Tooltip
                    label={isTTSPlaying ? 'Stop audio' : 'Listen to passage'}
                  >
                    <ActionIcon
                      size='sm'
                      variant='subtle'
                      color={isTTSPlaying ? 'red' : 'blue'}
                      onClick={() => {
                        if (isTTSPlaying || isTTSLoading) {
                          handleTTSStop();
                        } else {
                          handleTTSPlay(
                            currentQuestion.question.content?.passage || ''
                          );
                        }
                      }}
                      disabled={false}
                      aria-label={
                        isTTSPlaying || isTTSLoading
                          ? 'Stop audio'
                          : 'Listen to passage'
                      }
                    >
                      {isTTSPlaying || isTTSLoading ? (
                        <VolumeX size={16} />
                      ) : (
                        <Volume2 size={16} />
                      )}
                    </ActionIcon>
                  </Tooltip>
                )}
            </Group>

            {/* Show passage for reading comprehension questions */}
            {currentQuestion.question.type === 'reading_comprehension' &&
              currentQuestion.question.content?.passage && (
                <Paper
                  p='md'
                  bg='var(--mantine-color-gray-0)'
                  radius='md'
                  withBorder
                  style={{ marginBottom: 8, position: 'relative' }}
                >
                  <LoadingOverlay
                    visible={isTTSLoading}
                    overlayProps={{ backgroundOpacity: 0.35, blur: 1 }}
                    zIndex={5}
                  />
                  <Text
                    size='md'
                    style={{
                      whiteSpace: 'pre-line',
                      lineHeight: 1.7,
                      fontWeight: 400,
                      letterSpacing: 0.2,
                    }}
                  >
                    {(() => {
                      try {
                        const { splitIntoParagraphs } = require('../../utils/reading');
                        const paras = splitIntoParagraphs(
                          currentQuestion.question.content.passage,
                          3
                        );
                        return paras.map((p: string, idx: number) => (
                          <div key={idx} style={{ marginBottom: idx < paras.length - 1 ? 12 : 0 }}>
                            {p}
                            {idx < paras.length - 1 ? <br /> : null}
                          </div>
                        ));
                      } catch (e) {
                        return currentQuestion.question.content.passage;
                      }
                    })()}
                  </Text>
                </Paper>
              )}

            {/* Vocabulary question: show sentence with highlighted target word */}
            {currentQuestion.question.type === 'vocabulary' &&
              (() => {
                const { sentence, question: qWord } =
                  currentQuestion.question.content || {};
                const baseSentence =
                  sentence || currentQuestion.question.content?.question || '';
                if (
                  baseSentence &&
                  qWord &&
                  baseSentence.trim() &&
                  qWord.trim()
                ) {
                  // If the sentence equals the target word, skip highlighting to keep text contiguous
                  const shouldHighlight =
                    baseSentence.trim().toLowerCase() !==
                    qWord.trim().toLowerCase();
                  return (
                    <>
                      {/* Make sentence/title sizing match desktop QuestionCard */}
                      <Text size='lg' fw={500}>
                        {shouldHighlight
                          ? highlightTargetWord(baseSentence, qWord)
                          : baseSentence}
                      </Text>
                      {/* Standalone vocabulary word removed to avoid duplicate display */}
                      {shouldHighlight && (
                        <Text
                          size='sm'
                          c='dimmed'
                          mt={4}
                          style={{ fontWeight: 500 }}
                        >
                          What does <strong>{qWord}</strong> mean in this
                          context?
                        </Text>
                      )}
                    </>
                  );
                }
                // Fallback: render question only
                return (
                  <Text size='lg' fw={500}>
                    {qWord || ''}
                  </Text>
                );
              })()}

            {/* Fallback: show the main question text when it hasn't been shown by specific handlers above */}
            {currentQuestion.question.content?.question &&
              currentQuestion.question.type !== 'vocabulary' &&
              currentQuestion.question.type !== 'reading_comprehension' && (
                <Text size='lg' fw={500} style={{ whiteSpace: 'pre-line' }}>
                  {currentQuestion.question.content.question}
                </Text>
              )}

            {/* Answer Options */}
            <Stack gap='sm'>
              {currentQuestion.question.content?.options ? (
                currentQuestion.question.content.options.map(
                  (option: string, index: number) => {
                    const isSelected = selectedAnswerLocal === index;
                    const isCorrect =
                      showFeedback &&
                      feedbackLocal.correct_answer_index === index;
                    const isIncorrect =
                      showFeedback &&
                      selectedAnswerLocal === index &&
                      !isCorrect;

                    return (
                      <Button
                        key={index}
                        variant={isSelected ? 'filled' : 'light'}
                        color={
                          isIncorrect ? 'red' : isCorrect ? 'green' : 'blue'
                        }
                        c={
                          isIncorrect ? 'red' : isCorrect ? 'green' : undefined
                        }
                        size='lg'
                        onClick={() => {
                          if (!isSubmittedLocal) {
                            setSelectedAnswerLocal(index);
                            // Scroll to submit button in mobile view after a brief delay
                            // to ensure the selection state has been updated
                            setTimeout(() => {
                              scrollToSubmitButton();
                            }, 100);
                          }
                        }}
                        disabled={isSubmittedLocal}
                        fullWidth
                        justify='flex-start'
                        leftSection={
                          showFeedback ? (
                            isCorrect ? (
                              <IconCheck size={16} />
                            ) : isIncorrect ? (
                              <IconX size={16} />
                            ) : null
                          ) : null
                        }
                        styles={{
                          root: {
                            height: 'auto',
                            padding: '12px 16px',
                            textAlign: 'left',
                            opacity:
                              isSubmittedLocal && !isCorrect && !isIncorrect
                                ? 0.6
                                : 1,
                          },
                          label: {
                            fontWeight: isCorrect || isIncorrect ? 600 : 400,
                            color: isCorrect
                              ? theme.colors.green[7]
                              : isIncorrect
                                ? theme.colors.red[7]
                                : undefined,
                          },
                        }}
                      >
                        <Text
                          style={{
                            wordBreak: 'break-word',
                            whiteSpace: 'normal',
                          }}
                        >
                          {option}
                        </Text>
                      </Button>
                    );
                  }
                )
              ) : (
                <Text c='dimmed' ta='center'>
                  Loading options...
                </Text>
              )}
            </Stack>

            {/* Feedback Section */}
            {showFeedback && (
              <Alert
                color={feedbackLocal.is_correct ? 'green' : 'red'}
                icon={
                  feedbackLocal.is_correct ? (
                    <IconCheck size={16} />
                  ) : (
                    <IconX size={16} />
                  )
                }
              >
                <Stack gap='xs'>
                  <Text size='sm' fw={500}>
                    {feedbackLocal.is_correct ? 'Correct!' : 'Incorrect'}
                  </Text>
                  {feedbackLocal.explanation && (
                    <Text size='sm'>{feedbackLocal.explanation}</Text>
                  )}
                </Stack>
              </Alert>
            )}
          </Stack>
        </Paper>

        {/* Action Buttons */}
        <Group grow>
          {/* Show Submit button until the answer is submitted */}
          {!isSubmittedLocal && (
            <Button
              ref={submitButtonRef}
              variant='filled'
              onClick={handleAnswerSubmit}
              disabled={!canSubmit}
              loading={isSubmittingAnswer}
            >
              Submit Answer
            </Button>
          )}

          {/* After submission */}
          {isSubmittedLocal && !isAllCompleted && (
            <Button
              ref={nextButtonRef}
              variant='light'
              onClick={handleNextQuestion}
              disabled={!hasNextQuestion}
            >
              Next Question
            </Button>
          )}

          {/* When all completed, allow navigating back and forth */}
          {isAllCompleted && (
            <>
              <Button
                variant='light'
                onClick={handlePreviousQuestion}
                disabled={!hasPreviousQuestion}
              >
                Previous
              </Button>
              <Button
                variant='light'
                onClick={handleNextQuestion}
                disabled={!hasNextQuestion}
              >
                Next
              </Button>
            </>
          )}
        </Group>
        {/* Bottom section: report issue and adjust frequency */}
        <Box
          style={{
            borderTop: '1px solid var(--mantine-color-default-border)',
            padding: '4px 16px',
            backgroundColor: 'var(--mantine-color-gray-0)',
            marginTop: '16px',
          }}
        >
          <Group justify='space-between' gap='xs'>
            <Button
              onClick={() => handleReport()}
              disabled={isReported || reportMutation.isPending}
              variant='subtle'
              color='gray'
              size='xs'
              data-testid='report-question-btn'
              style={{ flex: 1 }}
            >
              {isReported ? 'Reported' : 'Report issue'}
            </Button>

            <Button
              onClick={() => setShowMarkKnownModal(true)}
              variant='subtle'
              color='gray'
              size='xs'
              data-testid='mark-known-btn'
              style={{ flex: 1 }}
            >
              Adjust frequency
            </Button>
          </Group>
        </Box>

        {/* Mark Known Modal */}
        <Modal
          opened={showMarkKnownModal}
          onClose={() => setShowMarkKnownModal(false)}
          title='Adjust Question Frequency'
          size='sm'
          closeOnClickOutside={false}
          closeOnEscape={false}
        >
          <Stack gap='md'>
            <Text size='sm' c='dimmed'>
              Choose how often you want to see this question in future quizzes:
              1–2 show it more, 3 no change, 4–5 show it less.
            </Text>
            <Text size='sm' fw={500}>
              How confident are you about this question?
            </Text>
            <Group gap='xs' justify='space-between'>
              {[1, 2, 3, 4, 5].map(level => (
                <Button
                  key={level}
                  variant={confidenceLevel === level ? 'filled' : 'light'}
                  color={confidenceLevel === level ? 'teal' : 'gray'}
                  onClick={() => setConfidenceLevel(level)}
                  style={{ flex: 1, minHeight: '56px' }}
                  data-testid={`confidence-level-${level}`}
                >
                  {level}
                </Button>
              ))}
            </Group>
            <Group justify='space-between'>
              <Button
                variant='subtle'
                onClick={() => {
                  setShowMarkKnownModal(false);
                  setConfidenceLevel(null);
                }}
                data-testid='cancel-mark-known'
              >
                Cancel
              </Button>
              <Button
                onClick={() => {
                  if (!currentQuestion?.question_id || !confidenceLevel) return;
                  setIsMarkingKnown(true);
                  markKnownMutation.mutate({
                    id: currentQuestion.question_id,
                    data: { confidence_level: confidenceLevel },
                  });
                }}
                disabled={!confidenceLevel}
                loading={isMarkingKnown}
                color='teal'
                data-testid='submit-mark-known'
              >
                Save
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Report Modal */}
        <Modal
          opened={showReportModal}
          onClose={() => {
            setShowReportModal(false);
            setReportReason('');
          }}
          title='Report Issue with Question'
          size='sm'
          closeOnClickOutside={false}
          closeOnEscape={false}
        >
          <Stack gap='md'>
            <Text size='sm' c='dimmed'>
              Please let us know what's wrong with this question. Your feedback
              helps us improve the quality of our content.
            </Text>
            <Textarea
              placeholder='Describe the issue (optional, max 512 characters)...'
              value={reportReason}
              onChange={e => setReportReason(e.target.value)}
              maxLength={512}
              minRows={4}
              data-testid='report-reason-input'
              id='report-reason-textarea'
            />
            <Group justify='space-between'>
              <Button
                variant='subtle'
                onClick={() => {
                  setShowReportModal(false);
                  setReportReason('');
                }}
                data-testid='cancel-report'
              >
                Cancel
              </Button>
              <Button
                onClick={() => {
                  if (!currentQuestion?.question_id) return;
                  setIsReporting(true);
                  reportMutation.mutate({
                    id: currentQuestion.question_id,
                    data: { report_reason: reportReason },
                  });
                }}
                disabled={isReporting}
                loading={isReporting}
                color='red'
                data-testid='submit-report'
              >
                Report Question
              </Button>
            </Group>
          </Stack>
        </Modal>
      </Stack>
    </Container>
  );
};

export default MobileDailyPage;
