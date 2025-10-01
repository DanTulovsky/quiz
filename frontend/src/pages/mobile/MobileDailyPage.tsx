import React, { useCallback, useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { parseLocalDateString } from '../../utils/time';
import {
  Container,
  Paper,
  Stack,
  Text,
  Button,
  Group,
  Badge,
  ActionIcon,
  Popover,
  Alert,
  Loader,
  Center,
  Progress,
  Tooltip,
  LoadingOverlay,
} from '@mantine/core';
import { IconCheck, IconX, IconCalendar } from '@tabler/icons-react';
import { Volume2, VolumeX } from 'lucide-react';
import { useDailyQuestions } from '../../hooks/useDailyQuestions';
import { useDisclosure } from '@mantine/hooks';
import DailyDatePicker from '../../components/DailyDatePicker';
import { useMantineTheme } from '@mantine/core';
import { useTTS } from '../../hooks/useTTS';
import { defaultVoiceForLanguage } from '../../utils/tts';

const MobileDailyPage: React.FC = () => {
  const { date: dateParam } = useParams();

  const {
    selectedDate,
    setSelectedDate,
    currentQuestion,
    submitAnswer,
    goToNextQuestion,
    goToPreviousQuestion,
    hasNextQuestion,
    hasPreviousQuestion,
    isLoading,
    isSubmittingAnswer,
    currentQuestionIndex,
    questions,
    availableDates,
  } = useDailyQuestions();

  // Popover control for date picker
  const [pickerOpened, { toggle: togglePicker, close: closePicker }] =
    useDisclosure(false);

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

  // Reset state when question changes
  useEffect(() => {
    setSelectedAnswerLocal(null);
    setIsSubmittedLocal(false);
    setFeedbackLocal(null);
  }, [currentQuestion?.id]);

  // Scroll to top when a new daily question is loaded (mobile)
  useEffect(() => {
    if (!currentQuestion?.id) return;
    try {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    } catch (e) {
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

  // Handle answer submission
  const handleAnswerSubmit = useCallback(async () => {
    if (!currentQuestion || selectedAnswerLocal === null) return;

    setIsSubmittedLocal(true);

    try {
      const response = await submitAnswer(
        currentQuestion.question_id,
        selectedAnswerLocal
      );
      setFeedbackLocal(response);
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

  // Handle previous question
  const handlePrevQuestion = useCallback(() => {
    setSelectedAnswerLocal(null);
    setIsSubmittedLocal(false);
    setFeedbackLocal(null);
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
            <Group justify='space-between' align='center'>
              <Badge variant='light' color='orange'>
                Daily Challenge
              </Badge>

              {/* Right side: date picker icon and question counter */}
              <Group gap='sm' align='center'>
                {/* Date picker popover */}
                <Popover
                  opened={pickerOpened}
                  onChange={opened => {
                    if (!opened) closePicker();
                  }}
                  position='bottom'
                  offset={4}
                >
                  <Popover.Target>
                    <ActionIcon
                      variant='light'
                      size='lg'
                      onClick={togglePicker}
                      style={{ position: 'relative', padding: 6 }}
                      aria-label='Select date'
                    >
                      <IconCalendar size={18} />
                      {selectedDate && (
                        <Badge
                          size='xs'
                          color='blue'
                          variant='filled'
                          style={{
                            position: 'absolute',
                            top: -2,
                            right: -2,
                            transform: 'scale(0.75)',
                            pointerEvents: 'none',
                          }}
                        >
                          {parseLocalDateString(selectedDate).getDate()}
                        </Badge>
                      )}
                    </ActionIcon>
                  </Popover.Target>

                  <Popover.Dropdown p={0}>
                    <DailyDatePicker
                      dropdownType='modal'
                      selectedDate={selectedDate}
                      onDateSelect={date => {
                        if (date) {
                          setSelectedDate(date);
                          closePicker();
                        }
                      }}
                      availableDates={availableDates}
                      maxDate={new Date()}
                      size='sm'
                      clearable={false}
                      hideOutsideDates
                      withCellSpacing={false}
                      firstDayOfWeek={1}
                    />
                  </Popover.Dropdown>
                </Popover>

                <Badge variant='outline'>
                  {currentQuestionIndex + 1} of {questions.length}
                </Badge>
              </Group>
            </Group>
            {/* Removed redundant Daily Questions label */}
            <Progress value={progressValue} color='orange' />
          </Stack>
        </Paper>

        {/* Current Question */}
        <Paper p='md' radius='md' withBorder>
          <Stack gap='md'>
            <Group justify='space-between'>
              <Badge color='blue'>
                {currentQuestion.question.language} -{' '}
                {currentQuestion.question.level}
              </Badge>

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
                    {currentQuestion.question.content.passage}
                  </Text>
                </Paper>
              )}

            <Text size='lg' fw={500}>
              {currentQuestion.question.content?.question}
            </Text>

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
                        onClick={() =>
                          !isSubmittedLocal && setSelectedAnswerLocal(index)
                        }
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
          <Button
            variant='light'
            onClick={handlePrevQuestion}
            disabled={!hasPreviousQuestion}
          >
            Previous
          </Button>

          {!isSubmittedLocal ? (
            <Button
              variant='filled'
              onClick={handleAnswerSubmit}
              disabled={!canSubmit}
              loading={isSubmittingAnswer}
            >
              Submit Answer
            </Button>
          ) : (
            <Button
              variant='filled'
              onClick={handleNextQuestion}
              disabled={!hasNextQuestion}
            >
              Next Question
            </Button>
          )}
        </Group>
      </Stack>
    </Container>
  );
};

export default MobileDailyPage;
