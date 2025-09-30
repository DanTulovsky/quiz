import React, { useCallback, useEffect, useState } from 'react';
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
  Tooltip,
  ActionIcon,
  LoadingOverlay,
} from '@mantine/core';
import { IconCheck, IconX } from '@tabler/icons-react';
import { Volume2, VolumeX } from 'lucide-react';
import { useQuestionFlow } from '../hooks/useQuestionFlow';
import { useTTS } from '../hooks/useTTS';
import { defaultVoiceForLanguage } from '../utils/tts';

export type QuestionMode = 'quiz' | 'reading' | 'vocabulary';

interface Props {
  mode: QuestionMode;
}

const MobileQuestionPageBase: React.FC<Props> = ({ mode }) => {
  const { questionId } = useParams();

  const { quizFeedback, setQuizFeedback, readingFeedback, setReadingFeedback } =
    useQuestion();

  const feedback = mode === 'quiz' ? quizFeedback : readingFeedback;
  const setFeedback = mode === 'quiz' ? setQuizFeedback : setReadingFeedback;

  // Local UI state
  const [selectedAnswerLocal, setSelectedAnswerLocal] = useState<number | null>(
    null
  );
  const [isSubmittedLocal, setIsSubmittedLocal] = useState(false);

  // TTS state for reading comprehension passages
  const {
    isLoading: isTTSLoading,
    isPlaying: isTTSPlaying,
    playTTS,
    stopTTS,
  } = useTTS();

  const { question, isLoading, error, forceFetchNextQuestion } =
    useQuestionFlow({ mode, questionId });

  // URL state management for question navigation
  useQuestionUrlState({
    mode,
    question,
    isLoading,
  });

  const [isTransitioning, setIsTransitioning] = useState(false);

  // Handle answer submission
  const handleAnswerSubmit = useCallback(async () => {
    if (!question || selectedAnswerLocal === null) return;

    setIsSubmittedLocal(true);

    try {
      const response = await postV1QuizAnswer({
        question_id: question.id || 0,
        user_answer_index: selectedAnswerLocal,
      });

      setFeedback(response);
    } catch (error) {
      console.error('Failed to submit answer:', error);
    }
  }, [question, selectedAnswerLocal, setFeedback]);

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
  const handleTTSPlay = async (text: string) => {
    if (!text) return;

    // Determine the best voice: default for question.language -> fallback to 'echo'
    const finalVoice =
      defaultVoiceForLanguage(question?.language || 'en') || 'echo';

    await playTTS(text, finalVoice);
  };

  const handleTTSStop = () => {
    stopTTS();
  };

  // Reset state when question changes
  useEffect(() => {
    setSelectedAnswerLocal(null);
    setIsSubmittedLocal(false);
    setFeedback(null);
  }, [question?.id, setFeedback]);

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

  const canSubmit = selectedAnswerLocal !== null && !isSubmittedLocal;
  const showFeedback = isSubmittedLocal && feedback;

  const modeLabel =
    mode === 'quiz' ? 'Quiz' : mode === 'vocabulary' ? 'Vocabulary' : 'Reading';

  return (
    <Container size='sm'>
      <Stack gap='md'>
        {/* Question Header */}
        <Paper p='md' radius='md' withBorder>
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
                  <LoadingOverlay
                    visible={isTTSLoading}
                    overlayProps={{ backgroundOpacity: 0.35, blur: 1 }}
                    zIndex={5}
                  />
                  <Box
                    style={{
                      position: 'absolute',
                      top: 8,
                      right: 8,
                      zIndex: 10,
                    }}
                  >
                    <Group gap={4} align='center'>
                      <Badge
                        size='xs'
                        color='gray'
                        variant='filled'
                        radius='sm'
                        title='Shortcut: P'
                      >
                        P
                      </Badge>
                      <Tooltip
                        label={
                          isTTSPlaying ? 'Stop audio' : 'Listen to passage'
                        }
                      >
                        <ActionIcon
                          size='sm'
                          variant='subtle'
                          color={isTTSPlaying ? 'red' : 'blue'}
                          onClick={() => {
                            if (isTTSPlaying || isTTSLoading) {
                              handleTTSStop();
                            } else {
                              handleTTSPlay(question.content?.passage || '');
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
                    </Group>
                  </Box>
                  <Text
                    size='sm'
                    style={{
                      whiteSpace: 'pre-line',
                      lineHeight: 1.6,
                      fontWeight: 400,
                      letterSpacing: 0.1,
                    }}
                  >
                    {question.content.passage}
                  </Text>
                </Paper>
              )}

            {/* For reading comprehension, place the question after the passage */}
            {question.type === 'reading_comprehension' && (
              <Text size='lg' fw={500}>
                {question.content?.question}
              </Text>
            )}

            {/* For vocabulary, show sentence context */}
            {question.type === 'vocabulary' && question.content?.sentence && (
              <>
                <Text size='sm' c='dimmed' fs='italic'>
                  "{question.content.sentence}"
                </Text>
                <Text size='lg' fw={500}>
                  {question.content?.question}
                </Text>
              </>
            )}

            {/* For other question types */}
            {question.type !== 'reading_comprehension' &&
              question.type !== 'vocabulary' && (
                <Text size='lg' fw={500}>
                  {question.content?.question}
                </Text>
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
            variant='filled'
            onClick={handleNextQuestion}
            loading={isTransitioning}
            fullWidth
          >
            Next Question
          </Button>
        )}
      </Stack>
    </Container>
  );
};

export default MobileQuestionPageBase;
