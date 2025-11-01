import React, { useCallback, useEffect, useState, useRef } from 'react';
import { splitIntoParagraphs } from '../utils/passage';
import { useMediaQuery } from '@mantine/hooks';
import { useParams } from 'react-router-dom';
import { useQuestion } from '../contexts/useQuestion';
import { useQuestionUrlState } from '../hooks/useQuestionUrlState';
import { postV1QuizAnswer } from '../api/api';
import {
  Container,
  Paper,
  Stack,
  Text,
  Button,
  Group,
  Badge,
  Alert,
  Loader,
  Center,
  Box,
  Modal,
  Textarea,
} from '@mantine/core';
import { useAuth } from '../hooks/useAuth';
import { IconCheck, IconX } from '@tabler/icons-react';
import TTSButton from './TTSButton';
import { useQuestionFlow } from '../hooks/useQuestionFlow';
import { defaultVoiceForLanguage } from '../utils/tts';
import {
  usePostV1QuizQuestionIdReport,
  usePostV1QuizQuestionIdMarkKnown,
  useGetV1PreferencesLearning,
} from '../api/api';
import { showNotificationWithClean } from '../notifications';
import { SnippetHighlighter } from './SnippetHighlighter';
import { useQuestionSnippets } from '../hooks/useQuestionSnippets';

export type QuestionMode = 'quiz' | 'reading' | 'vocabulary';

interface Props {
  mode: QuestionMode;
}

const MobileQuestionPageBase: React.FC<Props> = ({ mode }) => {
  const isSmallScreen = useMediaQuery('(max-width: 768px)');
  const { questionId } = useParams();
  const { user } = useAuth();

  const {
    quizFeedback,
    setQuizFeedback,
    readingFeedback,
    setReadingFeedback,
    setQuizQuestion,
    setReadingQuestion,
  } = useQuestion();

  const feedback = mode === 'quiz' ? quizFeedback : readingFeedback;
  const setFeedback = mode === 'quiz' ? setQuizFeedback : setReadingFeedback;

  // Early return if not authenticated - prevents flash of error state before redirect
  if (!user) {
    return null;
  }

  // Local UI state
  const [selectedAnswerLocal, setSelectedAnswerLocal] = useState<number | null>(
    null
  );
  const [isSubmittedLocal, setIsSubmittedLocal] = useState(false);
  // Local submitting state to avoid leaving the UI disabled if the network fails
  const [isSubmittingLocal, setIsSubmittingLocal] = useState(false);

  // Refs for buttons to enable scrolling
  const submitButtonRef = useRef<HTMLButtonElement>(null);
  const nextButtonRef = useRef<HTMLButtonElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const questionPaperRef = useRef<HTMLDivElement>(null);

  const { question, isLoading, error, forceFetchNextQuestion } =
    useQuestionFlow({ mode, questionId });

  // Fetch snippets for the current question
  const { snippets } = useQuestionSnippets(question?.id);

  // Fetch user learning preferences for TTS voice
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();

  // Update the global QuestionContext when local question changes (same as desktop)
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

  // Reporting & mark-known state (mobile parity with desktop QuestionCard)
  const [isReported, setIsReported] = useState(false);
  const [showMarkKnownModal, setShowMarkKnownModal] = useState(false);
  const [showReportModal, setShowReportModal] = useState(false);
  const [reportReason, setReportReason] = useState('');
  const [isReporting, setIsReporting] = useState(false);
  const [confidenceLevel, setConfidenceLevel] = useState<number | null>(null);
  const [isMarkingKnown, setIsMarkingKnown] = useState(false);

  // URL state management for question navigation
  useQuestionUrlState({
    mode,
    question,
    isLoading,
  });

  const [isTransitioning, setIsTransitioning] = useState(false);

  // Function to scroll to submit button (mobile only)
  const scrollToSubmitButton = useCallback(() => {
    if (submitButtonRef.current) {
      submitButtonRef.current.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
      });
    }
  }, []);

  // Handle answer submission
  const handleAnswerSubmit = useCallback(async () => {
    if (!question || selectedAnswerLocal === null) return;

    // Prevent duplicate submissions and show loading state
    if (isSubmittingLocal) return;
    setIsSubmittingLocal(true);

    try {
      const response = await postV1QuizAnswer({
        question_id: question.id || 0,
        user_answer_index: selectedAnswerLocal,
      });

      setFeedback(response);
      // Mark as submitted only on success so the options remain selectable on failure
      setIsSubmittedLocal(true);

      // Scroll to next question button after feedback is shown
      setTimeout(() => {
        if (nextButtonRef.current) {
          nextButtonRef.current.scrollIntoView({
            behavior: 'smooth',
            block: 'center',
          });
        }
      }, 300); // Delay to allow feedback animation to complete
    } catch {
      showNotificationWithClean({
        title: 'Error',
        message: 'Failed to submit answer. Please try again.',
        color: 'red',
      });
      // Ensure we do not leave the UI in a submitted/disabled state
      setIsSubmittedLocal(false);
    } finally {
      setIsSubmittingLocal(false);
    }
  }, [question, selectedAnswerLocal, setFeedback, isSubmittingLocal]);

  // Handle next question
  const handleNextQuestion = useCallback(async () => {
    setIsTransitioning(true);
    setSelectedAnswerLocal(null);
    setIsSubmittedLocal(false);
    setFeedback(null);

    try {
      await forceFetchNextQuestion();
    } finally {
      setIsTransitioning(false);
    }
  }, [forceFetchNextQuestion, setFeedback]);

  // TTS handler functions

  const { isAuthenticated } = useAuth();

  const handleReport = async () => {
    if (isReported || reportMutation.isPending || !question?.id) return;

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

  const handleSubmitReport = async () => {
    if (!question?.id) return;

    setIsReporting(true);
    try {
      reportMutation.mutate({
        id: question.id,
        data: { report_reason: reportReason },
      });
    } finally {
      setIsReporting(false);
    }
  };

  const handleMarkAsKnown = async () => {
    if (!question?.id || !confidenceLevel) return;

    setIsMarkingKnown(true);
    try {
      markKnownMutation.mutate({
        id: question.id,
        data: { confidence_level: confidenceLevel },
      });
    } finally {
      setIsMarkingKnown(false);
    }
  };

  // API hooks for reporting / mark known
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

  // Reset state when question changes
  useEffect(() => {
    setSelectedAnswerLocal(null);
    setIsSubmittedLocal(false);
    setFeedback(null);
    setIsReported(false);
  }, [question?.id, setFeedback]);

  // Scroll to top when a new question is loaded (mobile)
  useEffect(() => {
    // Only scroll when we have a question id and loading is complete
    if (!question?.id || isLoading) return;

    const scrollToTop = () => {
      try {
        // Find AppShell.Main - try multiple selectors
        let appShellMain: HTMLElement | null = null;
        const possibleSelectors = [
          '[data-mantine-component="AppShell.Main"]',
          'main[role="main"]',
          '.mantine-AppShell-main',
          'main',
        ];

        for (const selector of possibleSelectors) {
          const element = document.querySelector(selector) as HTMLElement;
          if (
            element &&
            (containerRef.current
              ? element.contains(containerRef.current)
              : true)
          ) {
            appShellMain = element;
            break;
          }
        }

        // Get header height
        const header = document.querySelector('header') as HTMLElement;
        const headerHeight = header?.offsetHeight || 50;

        if (appShellMain && questionPaperRef.current) {
          // Get the question paper's position relative to AppShell.Main
          const appShellRect = appShellMain.getBoundingClientRect();
          const paperRect = questionPaperRef.current.getBoundingClientRect();

          // Calculate scroll needed to put paper at top (accounting for header)
          const currentScroll = appShellMain.scrollTop;
          const paperTopRelativeToAppShell =
            paperRect.top - appShellRect.top + currentScroll;
          const targetScroll = Math.max(
            0,
            paperTopRelativeToAppShell - headerHeight
          );

          // Instant scroll first
          appShellMain.scrollTop = targetScroll;

          // Then smooth scroll
          setTimeout(() => {
            appShellMain?.scrollTo({ top: targetScroll, behavior: 'smooth' });
          }, 100);
        } else if (questionPaperRef.current) {
          // Fallback: scroll window and use scrollIntoView with offset
          const paperRect = questionPaperRef.current.getBoundingClientRect();
          const currentScroll =
            window.scrollY || document.documentElement.scrollTop;
          const targetScroll = Math.max(
            0,
            currentScroll + paperRect.top - headerHeight
          );

          window.scrollTo({ top: targetScroll, behavior: 'auto' });
          document.documentElement.scrollTop = targetScroll;
          document.body.scrollTop = targetScroll;

          setTimeout(() => {
            window.scrollTo({ top: targetScroll, behavior: 'smooth' });
          }, 100);
        }
      } catch {
        // ignore (e.g., server-side rendering or environments without scrollTo)
      }
    };

    // Use multiple requestAnimationFrames with delay to ensure DOM has fully updated
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        // Add a small delay to ensure rendering is complete
        setTimeout(scrollToTop, 50);
      });
    });
  }, [question?.id, isLoading]);

  if (isLoading && !question) {
    return (
      <Center h='100%'>
        <Loader size='lg' />
      </Center>
    );
  }

  if (error) {
    return (
      <Container size='sm'>
        <Alert color='red' title='Error' icon={<IconX size={16} />}>
          {error}
        </Alert>
      </Container>
    );
  }

  if (!question) {
    return (
      <Center h='100%'>
        <Text>No question available</Text>
      </Center>
    );
  }

  const canSubmit =
    selectedAnswerLocal !== null && !isSubmittedLocal && !isSubmittingLocal;
  const showFeedback = isSubmittedLocal && feedback;

  const modeLabel =
    mode === 'quiz' ? 'Quiz' : mode === 'vocabulary' ? 'Vocabulary' : 'Reading';

  return (
    <Container ref={containerRef} size='sm'>
      <Stack gap='md'>
        {/* Question Header */}
        <Paper ref={questionPaperRef} p='md' radius='md' withBorder>
          <Stack gap='xs'>
            <Group justify='space-between'>
              <Badge variant='light' color='blue'>
                {modeLabel}
              </Badge>
              <Badge variant='outline'>
                {question.language} - {question.level}
              </Badge>
            </Group>
            {/* Show passage for reading comprehension questions */}
            {question.type === 'reading_comprehension' &&
              question.content?.passage && (
                <Paper
                  p='md'
                  bg='var(--mantine-color-body)'
                  radius='md'
                  withBorder
                  style={{ position: 'relative' }}
                >
                  {/* Loading state handled by TTSButton component */}
                  <Box
                    style={{
                      position: 'absolute',
                      top: 8,
                      right: 8,
                      zIndex: 10,
                    }}
                  >
                    <TTSButton
                      getText={() => question.content?.passage || ''}
                      getVoice={() => {
                        const saved = (
                          userLearningPrefs?.tts_voice || ''
                        ).trim();
                        if (saved) return saved;
                        const voice = defaultVoiceForLanguage(
                          question.language
                        );
                        return voice || undefined;
                      }}
                      size='sm'
                      ariaLabel='Passage audio'
                    />
                  </Box>
                  {(() => {
                    const per = isSmallScreen ? 2 : 4;
                    const paras = splitIntoParagraphs(
                      question.content.passage,
                      per
                    );
                    return (
                      <div data-allow-translate='true'>
                        {paras.map((p, i) => (
                          <SnippetHighlighter
                            key={i}
                            text={p}
                            snippets={snippets}
                            component={Text}
                            componentProps={{
                              size: 'md',
                              style: {
                                whiteSpace: 'pre-line',
                                lineHeight: 1.7,
                                fontWeight: 400,
                                letterSpacing: 0.2,
                                marginBottom: i === paras.length - 1 ? 0 : 10,
                              },
                            }}
                          />
                        ))}
                      </div>
                    );
                  })()}
                </Paper>
              )}

            {/* For reading comprehension, place the question after the passage */}
            {question.type === 'reading_comprehension' && (
              <SnippetHighlighter
                text={question.content?.question || ''}
                snippets={snippets}
                component={Text}
                componentProps={{
                  size: 'xl',
                  fw: 600,
                  mb: 'sm',
                  style: { lineHeight: 1.5 },
                  'data-testid': 'reading-comprehension-question',
                }}
              />
            )}

            {/* For vocabulary, show sentence context */}
            {question.type === 'vocabulary' && question.content?.sentence && (
              <div data-allow-translate='true'>
                {/* Vocabulary sentence with snippet highlighting */}
                <SnippetHighlighter
                  text={question.content.sentence}
                  snippets={snippets}
                  targetWord={question.content.question}
                  component={Text}
                  componentProps={{
                    size: 'lg',
                    'data-testid': 'vocab-sentence',
                    style: { marginBottom: 8 },
                  }}
                />
                {/* Prompt: What does X mean in this context? */}
                <Text
                  size='sm'
                  c='dimmed'
                  mt={6}
                  mb={10}
                  style={{ fontWeight: 500 }}
                >
                  What does <strong>{question.content.question}</strong> mean in
                  this context?
                </Text>
              </div>
            )}

            {/* For other question types (quiz, etc.) */}
            {question.type !== 'reading_comprehension' &&
              question.type !== 'vocabulary' && (
                <div data-allow-translate='true'>
                  <SnippetHighlighter
                    text={question.content?.question || ''}
                    snippets={snippets}
                    component={Text}
                    componentProps={{
                      size: 'lg',
                      fw: 500,
                    }}
                  />
                </div>
              )}
          </Stack>
        </Paper>

        {/* Answer Options */}
        <Paper p='md' radius='md' withBorder>
          <Stack gap='sm'>
            {question.content?.options ? (
              question.content.options.map((option: string, index: number) => {
                const isSelected = selectedAnswerLocal === index;
                const isCorrect =
                  showFeedback && feedback.correct_answer_index === index;
                const isIncorrect =
                  showFeedback && selectedAnswerLocal === index && !isCorrect;

                return (
                  <Button
                    key={index}
                    variant={isSelected ? 'filled' : 'light'}
                    color={isIncorrect ? 'red' : isCorrect ? 'green' : 'blue'}
                    size='lg'
                    onClick={() => {
                      if (!isSubmittedLocal && !isSubmittingLocal) {
                        setSelectedAnswerLocal(index);
                        // Scroll to submit button in mobile view after a brief delay
                        // to ensure the selection state has been updated
                        setTimeout(() => {
                          scrollToSubmitButton();
                        }, 100);
                      }
                    }}
                    disabled={isSubmittedLocal || isSubmittingLocal}
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
                      },
                    }}
                  >
                    <Text
                      style={{ wordBreak: 'break-word', whiteSpace: 'normal' }}
                    >
                      {option}
                    </Text>
                  </Button>
                );
              })
            ) : (
              <Text c='dimmed' ta='center'>
                Loading options...
              </Text>
            )}
          </Stack>
        </Paper>

        {/* Feedback Section */}
        {showFeedback && (
          <Paper p='md' radius='md' withBorder>
            <Stack gap='sm'>
              <Group>
                {feedback.is_correct ? (
                  <>
                    <IconCheck size={16} color='green' />
                    <Text size='sm' c='green' fw={500}>
                      Correct!
                    </Text>
                  </>
                ) : (
                  <>
                    <IconX size={16} color='red' />
                    <Text size='sm' c='red' fw={500}>
                      Incorrect
                    </Text>
                  </>
                )}
              </Group>
              {feedback.explanation && (
                <Text size='sm'>{feedback.explanation}</Text>
              )}
            </Stack>
          </Paper>
        )}

        {/* Action Buttons */}
        {!isSubmittedLocal ? (
          <Button
            ref={submitButtonRef}
            variant='filled'
            onClick={handleAnswerSubmit}
            disabled={!canSubmit}
            loading={isLoading}
            fullWidth
          >
            Submit Answer
          </Button>
        ) : (
          <Button
            ref={nextButtonRef}
            variant='filled'
            onClick={handleNextQuestion}
            loading={isTransitioning}
            fullWidth
          >
            Next Question
          </Button>
        )}
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
              onClick={handleReport}
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
                onClick={handleMarkAsKnown}
                disabled={!confidenceLevel || isMarkingKnown}
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
                onClick={handleSubmitReport}
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

export default MobileQuestionPageBase;
