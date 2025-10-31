import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../../hooks/useAuth';
import { useWordOfTheDay } from '../../hooks/useWordOfTheDay';
import {
  Container,
  Stack,
  Text,
  Center,
  Paper,
  Button,
  Group,
  Badge,
  Title,
  Card,
  ActionIcon,
  ThemeIcon,
} from '@mantine/core';
import { ChevronLeft, ChevronRight, Calendar } from 'lucide-react';
import LoadingSpinner from '../../components/LoadingSpinner';

const MobileWordOfTheDayPage: React.FC = () => {
  const { date: dateParam } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();

  const {
    selectedDate,
    word,
    isLoading,
    goToPreviousDay,
    goToNextDay,
    goToToday,
    canGoPrevious,
    canGoNext,
  } = useWordOfTheDay(dateParam);

  // Update URL when date changes (using effect to sync URL)
  React.useEffect(() => {
    if (dateParam !== selectedDate) {
      navigate(`/m/word-of-day/${selectedDate}`, { replace: true });
    }
  }, [selectedDate, dateParam, navigate]);

  // Swipe handlers (implement later if needed)

  // Format date for display
  const formatDisplayDate = (dateStr: string): string => {
    const date = new Date(dateStr + 'T00:00:00');
    return date.toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  if (!user) {
    return (
      <Container size='xs' py='lg' px='md'>
        <Center h='60vh'>
          <Text>Please log in to view your word of the day.</Text>
        </Center>
      </Container>
    );
  }

  return (
    <Container size='xs' py='lg' px='md'>
      <Stack gap='md'>
        {/* Header */}
        <Group justify='space-between' align='center'>
          <Title order={3}>Word of the Day</Title>
          <Button
            variant='subtle'
            size='xs'
            leftSection={<Calendar size={14} />}
            onClick={goToToday}
          >
            Today
          </Button>
        </Group>

        {/* Navigation buttons */}
        <Group justify='space-between' align='center'>
          <ActionIcon
            variant='subtle'
            size='lg'
            onClick={goToPreviousDay}
            disabled={!canGoPrevious || isLoading}
          >
            <ChevronLeft size={24} />
          </ActionIcon>

          <Text size='sm' c='dimmed' ta='center' style={{ flex: 1 }}>
            {formatDisplayDate(selectedDate)}
          </Text>

          <ActionIcon
            variant='subtle'
            size='lg'
            onClick={goToNextDay}
            disabled={!canGoNext || isLoading}
          >
            <ChevronRight size={24} />
          </ActionIcon>
        </Group>

        {/* Word display */}
        {isLoading ? (
          <Center h='50vh'>
            <LoadingSpinner />
          </Center>
        ) : word ? (
          <Card
            shadow='md'
            padding='lg'
            radius='md'
            style={{
              background: `var(--mantine-primary-color-0)`,
              border: `2px solid var(--mantine-primary-color-4)`,
              minHeight: '400px',
              position: 'relative',
              overflow: 'visible',
              wordWrap: 'break-word',
            }}
          >
            <Stack gap='md'>
              {/* Word */}
              <Title
                order={1}
                ta='center'
                style={{
                  lineHeight: 1.2,
                  fontSize: 'clamp(1.5rem, 8vw, 2.5rem)',
                  wordBreak: 'break-word',
                  overflowWrap: 'anywhere',
                  hyphens: 'auto',
                }}
                c='primary'
              >
                {word.word}
              </Title>

              {/* Translation */}
              <Text
                size='xl'
                ta='center'
                c='primary'
                style={{ fontStyle: 'italic' }}
              >
                {word.translation}
              </Text>

              {/* Example sentence */}
              {word.sentence && (
                <Paper
                  p='md'
                  radius='md'
                  style={{
                    background: 'var(--mantine-color-body)',
                    borderLeft: '3px solid var(--mantine-primary-color-4)',
                    marginTop: '16px',
                  }}
                  data-allow-translate='true'
                >
                  <Text
                    size='md'
                    style={{ lineHeight: 1.8, fontStyle: 'italic' }}
                  >
                    {word.sentence}
                  </Text>
                </Paper>
              )}

              {/* Explanation */}
              {word.explanation && (
                <Paper
                  p='sm'
                  radius='md'
                  style={{
                    background: 'var(--mantine-color-body)',
                    borderLeft: '3px solid var(--mantine-primary-color-4)',
                    marginTop: '8px',
                  }}
                >
                  <Text size='sm'>{word.explanation}</Text>
                </Paper>
              )}

              {/* Metadata badges */}
              <Group gap='xs' justify='center' mt='md'>
                {word.language && (
                  <Badge size='sm' variant='light' color='primary'>
                    {word.language}
                  </Badge>
                )}
                {word.level && (
                  <Badge size='sm' variant='light' color='primary'>
                    {word.level}
                  </Badge>
                )}
                {word.source_type && (
                  <Badge size='sm' variant='light' color='primary'>
                    {word.source_type === 'vocabulary_question'
                      ? 'Vocabulary'
                      : 'Snippet'}
                  </Badge>
                )}
              </Group>
            </Stack>
          </Card>
        ) : (
          <Center h='50vh'>
            <Stack align='center' gap='md'>
              <ThemeIcon size={48} radius='xl' color='gray' variant='light'>
                <Text size='lg'>📚</Text>
              </ThemeIcon>
              <Text size='md' c='dimmed' ta='center'>
                No word available for this date.
              </Text>
            </Stack>
          </Center>
        )}

        {/* Navigation hint */}
        <Text size='xs' c='dimmed' ta='center'>
          Use arrows to navigate between days
        </Text>
      </Stack>
    </Container>
  );
};

export default MobileWordOfTheDayPage;
