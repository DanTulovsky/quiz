import React, { useCallback, useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
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
  Progress,
} from '@mantine/core';
import { IconCheck, IconX } from '@tabler/icons-react';
import { useDailyQuestions } from '../../hooks/useDailyQuestions';

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

  return (
    <Container size='sm'>
      <Stack gap='md'>
        {/* Daily Progress Header */}
        <Paper p='md' radius='md' withBorder>
          <Stack gap='xs'>
            <Group justify='space-between'>
              <Badge variant='light' color='orange'>
                Daily Challenge
              </Badge>
              <Badge variant='outline'>
                {currentQuestionIndex + 1} of {questions.length}
              </Badge>
            </Group>
            <Text size='lg' fw={500}>
              Daily Questions
            </Text>
            <Progress value={progressValue} color='orange' />
          </Stack>
        </Paper>

        {/* Current Question */}
        <Paper p='md' radius='md' withBorder>
          <Stack gap='md'>
            <Group>
              <Badge color='blue'>
                {currentQuestion.question.language} -{' '}
                {currentQuestion.question.level}
              </Badge>
            </Group>

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
                        <Text>{option}</Text>
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
