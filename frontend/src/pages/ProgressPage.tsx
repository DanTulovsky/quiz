import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { getV1QuizProgress } from '../api/api';
import { showNotificationWithClean } from '../notifications';
import {
  Container,
  Text,
  Card,
  Stack,
  Title,
  Group,
  RingProgress,
  Center,
  Divider,
  ThemeIcon,
  List,
  Button,
  Loader,
  Table,
  Progress,
  Tooltip,
  useMantineTheme,
  Modal,
  ActionIcon,
  Box,
} from '@mantine/core';
import {
  IconCheck,
  IconX,
  IconTarget,
  IconTrophy,
  IconBrain,
  IconInfoCircle,
} from '@tabler/icons-react';
import { getV1QuizQuestionId } from '../api/api';

interface LocalUserProgress {
  current_level: string;
  total_questions: number;
  correct_answers: number;
  accuracy_rate: number;
  performance_by_topic: Record<string, PerformanceMetrics>;
  weak_areas: string[];
  recent_activity: UserResponse[];
  suggested_level?: string;
  // Worker-related properties
  worker_status?: {
    status: 'busy' | 'idle' | 'error';
    last_heartbeat?: string;
    error_message?: string;
  };
  learning_preferences?: {
    focus_on_weak_areas: boolean;
    fresh_question_ratio: number;
    known_question_penalty: number;
    review_interval_days: number;
    weak_area_boost: number;
  };
  priority_insights?: {
    total_questions_in_queue: number;
    high_priority_questions: number;
    medium_priority_questions: number;
    low_priority_questions: number;
  };
  generation_focus?: {
    current_generation_model?: string;
    generation_rate?: number;
    last_generation_time?: string;
  };
  high_priority_topics?: string[];
  gap_analysis?: Record<string, unknown>;
  priority_distribution?: Record<string, number>;
}

interface PerformanceMetrics {
  total_attempts: number;
  correct_attempts: number;
  average_response_time_ms: number;
  last_updated: string;
}

interface UserResponse {
  question_id: number;
  is_correct: boolean;
  created_at: string;
}

interface RecentActivityItemProps {
  activity: UserResponse;
  navigate: ReturnType<typeof useNavigate>;
}

const questionTextCache: Record<number, string> = {};

const RecentActivityItem: React.FC<RecentActivityItemProps> = ({
  activity,
  navigate,
}) => {
  const theme = useMantineTheme();
  const cachedText = questionTextCache[activity.question_id];
  const isErrorMessage =
    cachedText === 'Question unavailable' ||
    cachedText === 'Failed to load question';
  const [questionText, setQuestionText] = useState<string | null>(
    cachedText && !isErrorMessage ? cachedText : null
  );

  // Clear cache for error messages
  useEffect(() => {
    if (isErrorMessage) {
      delete questionTextCache[activity.question_id];
    }
  }, [activity.question_id, isErrorMessage]);
  const [loading, setLoading] = useState(false);
  const [opened, setOpened] = useState(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  let isDark = false;
  if ('colorScheme' in theme && typeof theme['colorScheme'] === 'string') {
    isDark = theme['colorScheme'] === 'dark';
  } else if (typeof window !== 'undefined' && window.matchMedia) {
    isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  }
  const checkColor = theme.colors.green[isDark ? 5 : 7];
  const xColor = theme.colors.red[isDark ? 5 : 7];

  // Fetch question text on tooltip open
  const handleTooltipOpen = async () => {
    if (!questionText && !loading) {
      setLoading(true);
      try {
        const resp = await getV1QuizQuestionId(activity.question_id);
        // All question types now have a standardized question field
        const text = resp.content?.question || 'Question text not available';
        questionTextCache[activity.question_id] = text;
        setQuestionText(text);

        // Only show tooltip once we have content
        setOpened(true);
      } catch {
        const errorText = 'Failed to load question';
        setQuestionText(errorText);
        setOpened(true);
      } finally {
        setLoading(false);
      }
    } else {
      // Show tooltip immediately if we already have content
      setOpened(true);
    }
  };

  // Debounce tooltip close to avoid flicker
  const handleTooltipClose = () => {
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    timeoutRef.current = setTimeout(() => setOpened(false), 100);
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString();
  };

  return (
    <Tooltip
      label={loading ? <Loader size='xs' /> : questionText || 'Loading...'}
      opened={opened}
      withArrow
      position='right'
      transitionProps={{ duration: 150 }}
      multiline
      maw={320}
    >
      <Group
        gap={8}
        align='center'
        style={{ cursor: 'pointer' }}
        onClick={() => navigate(`/quiz/${activity.question_id}`)}
        tabIndex={0}
        role='button'
        aria-label={`Go to question ${activity.question_id}`}
        onKeyDown={e => {
          if (e.key === 'Enter' || e.key === ' ')
            navigate(`/quiz/${activity.question_id}`);
        }}
        onMouseEnter={handleTooltipOpen}
        onMouseLeave={handleTooltipClose}
      >
        {activity.is_correct ? (
          <IconCheck
            size={18}
            stroke={2.2}
            color={checkColor}
            style={{ verticalAlign: 'middle' }}
          />
        ) : (
          <IconX
            size={18}
            stroke={2.2}
            color={xColor}
            style={{ verticalAlign: 'middle' }}
          />
        )}
        <Text fz='md' span>
          Answered question #{activity.question_id}
        </Text>
        <Text fz='md' c='dimmed' span ml='auto'>
          {formatDate(activity.created_at)}
        </Text>
      </Group>
    </Tooltip>
  );
};

// Info content components for each section
const CurrentLevelInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Current Level</Title>
    <Text size='sm'>
      Your current learning level based on your performance and the questions
      you've answered correctly. This level determines the difficulty of
      questions you'll receive.
    </Text>
    <Text size='sm' c='dimmed'>
      The system tracks your progress across different topics and adjusts your
      level accordingly. Higher levels mean more challenging questions and more
      complex language concepts.
    </Text>
    <Text size='sm' fw={600}>
      How it's calculated:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Accuracy rate across all topics</List.Item>
      <List.Item>Performance in weak areas</List.Item>
      <List.Item>Consistency of recent answers</List.Item>
      <List.Item>Overall question difficulty mastered</List.Item>
    </List>
  </Stack>
);

const QuestionsAnsweredInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Questions Answered</Title>
    <Text size='sm'>
      The total number of questions you've attempted across all topics and
      difficulty levels. This includes both correct and incorrect answers.
    </Text>
    <Text size='sm' c='dimmed'>
      More questions answered means more data for the system to understand your
      learning patterns and provide better personalized content.
    </Text>
    <Text size='sm' fw={600}>
      What this tells us:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Your engagement with the learning system</List.Item>
      <List.Item>Breadth of topics you've explored</List.Item>
      <List.Item>Consistency of your learning habits</List.Item>
      <List.Item>Data points for personalized recommendations</List.Item>
    </List>
  </Stack>
);

const AccuracyInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Accuracy Rate</Title>
    <Text size='sm'>
      The percentage of questions you've answered correctly across all topics
      and difficulty levels. This is a key metric for understanding your overall
      learning progress.
    </Text>
    <Text size='sm' c='dimmed'>
      A higher accuracy rate indicates better mastery of the material, while
      lower rates suggest areas that need more practice or review.
    </Text>
    <Text size='sm' fw={600}>
      Accuracy ranges:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>90%+ Excellent mastery</List.Item>
      <List.Item>80-89% Good understanding</List.Item>
      <List.Item>70-79% Solid progress</List.Item>
      <List.Item>60-69% Needs improvement</List.Item>
      <List.Item>&lt;60% Requires focused practice</List.Item>
    </List>
  </Stack>
);

const SuggestedLevelInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Suggested Level</Title>
    <Text size='sm'>
      The level we recommend you move to based on your current performance and
      learning patterns. This suggestion is based on comprehensive analysis of
      your progress.
    </Text>
    <Text size='sm' c='dimmed'>
      The system analyzes your accuracy, consistency, weak areas, and overall
      learning trajectory to determine the optimal next level for your continued
      growth.
    </Text>
    <Text size='sm' fw={600}>
      Factors considered:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Recent performance trends</List.Item>
      <List.Item>Mastery of current level concepts</List.Item>
      <List.Item>Readiness for more challenging content</List.Item>
      <List.Item>Balance of challenge and success</List.Item>
    </List>
  </Stack>
);

const PerformanceByTopicInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Performance by Topic</Title>
    <Text size='sm'>
      Detailed breakdown of your performance across different language learning
      topics. This helps identify your strengths and areas that need more
      attention.
    </Text>
    <Text size='sm' c='dimmed'>
      Each topic represents a specific aspect of language learning, such as
      grammar, vocabulary, pronunciation, or cultural understanding.
    </Text>
    <Text size='sm' fw={600}>
      What each column means:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>
        <strong>Topic:</strong> The specific language learning area
      </List.Item>
      <List.Item>
        <strong>Attempts:</strong> Total questions answered in this topic
      </List.Item>
      <List.Item>
        <strong>Correct:</strong> Number of correct answers
      </List.Item>
      <List.Item>
        <strong>Accuracy:</strong> Percentage of correct answers with visual
        indicator
      </List.Item>
    </List>
    <Text size='sm' fw={600}>
      Color coding:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>🟢 Green (80%+): Excellent mastery</List.Item>
      <List.Item>🟡 Yellow (60-79%): Good progress</List.Item>
      <List.Item>🔴 Red (&lt;60%): Needs improvement</List.Item>
    </List>
  </Stack>
);

const WorkerStatusInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Worker Status</Title>
    <Text size='sm'>
      Real-time status of the AI system that generates personalized questions
      for your learning. This shows whether the system is actively working on
      your content.
    </Text>
    <Text size='sm' c='dimmed'>
      The worker analyzes your learning patterns and creates questions tailored
      to your needs, focusing on areas where you need practice.
    </Text>
    <Text size='sm' fw={600}>
      Status meanings:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>
        <strong>Idle:</strong> System is ready, waiting for your next question
      </List.Item>
      <List.Item>
        <strong>Active:</strong> System is actively processing or generating
        content
      </List.Item>
      <List.Item>
        <strong>Error:</strong> Temporary issue with question generation
      </List.Item>
      <List.Item>
        <strong>Paused:</strong> System temporarily paused (user or global)
      </List.Item>
    </List>
    <Text size='sm' fw={600}>
      What the worker does:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Analyzes your learning patterns</List.Item>
      <List.Item>Identifies knowledge gaps</List.Item>
      <List.Item>Generates personalized questions</List.Item>
      <List.Item>Adapts difficulty based on performance</List.Item>
    </List>
  </Stack>
);

const LearningPreferencesInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Learning Focus</Title>
    <Text size='sm'>
      Your personalized learning strategy settings that determine how the system
      tailors questions to your specific needs and preferences.
    </Text>
    <Text size='sm' c='dimmed'>
      These preferences help the AI understand how you learn best and adjust the
      content accordingly for optimal learning outcomes.
    </Text>
    <Text size='sm' fw={600}>
      Key settings:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>
        <strong>Focus on weak areas:</strong> Prioritizes topics where you
        struggle
      </List.Item>
      <List.Item>
        <strong>Fresh questions ratio:</strong> Percentage of new vs. review
        questions
      </List.Item>
      <List.Item>
        <strong>Review interval:</strong> Days between reviewing known material
      </List.Item>
      <List.Item>
        <strong>Weak area boost:</strong> How much to emphasize difficult topics
      </List.Item>
    </List>
    <Text size='sm' fw={600}>
      Benefits:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Targeted practice in problem areas</List.Item>
      <List.Item>Balanced mix of new and review content</List.Item>
      <List.Item>Optimal spacing for long-term retention</List.Item>
      <List.Item>Personalized learning pace</List.Item>
    </List>
  </Stack>
);

const PriorityInsightsInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Question Priority</Title>
    <Text size='sm'>
      Shows how the system prioritizes questions for your learning journey.
      Questions are ranked based on their importance for your current learning
      goals.
    </Text>
    <Text size='sm' c='dimmed'>
      The AI analyzes your performance and creates a queue of questions
      optimized for your learning progress and identified weak areas.
    </Text>
    <Text size='sm' fw={600}>
      Priority levels:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>
        <strong>High Priority:</strong> Critical for addressing weak areas
      </List.Item>
      <List.Item>
        <strong>Medium Priority:</strong> Important for balanced learning
      </List.Item>
      <List.Item>
        <strong>Low Priority:</strong> Review or reinforcement questions
      </List.Item>
    </List>
    <Text size='sm' fw={600}>
      How priorities are determined:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Your recent performance patterns</List.Item>
      <List.Item>Identified knowledge gaps</List.Item>
      <List.Item>Learning preferences and goals</List.Item>
      <List.Item>Optimal learning sequence</List.Item>
    </List>
  </Stack>
);

const GenerationFocusInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>AI Generation</Title>
    <Text size='sm'>
      Information about the AI system that creates personalized questions for
      your learning. This shows the current model and generation activity.
    </Text>
    <Text size='sm' c='dimmed'>
      The AI continuously analyzes your learning patterns and generates
      questions specifically tailored to your needs and current level.
    </Text>
    <Text size='sm' fw={600}>
      What you'll see:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>
        <strong>Model:</strong> The AI model currently generating your questions
      </List.Item>
      <List.Item>
        <strong>Generation Rate:</strong> Questions created per minute
      </List.Item>
      <List.Item>
        <strong>Last Generation:</strong> When new questions were last created
      </List.Item>
    </List>
    <Text size='sm' fw={600}>
      How it works:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Analyzes your learning patterns</List.Item>
      <List.Item>Identifies knowledge gaps</List.Item>
      <List.Item>Creates contextually relevant questions</List.Item>
      <List.Item>Adapts difficulty based on performance</List.Item>
    </List>
  </Stack>
);

const WeakAreasInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Areas to Improve</Title>
    <Text size='sm'>
      Topics or concepts where you've shown lower performance and need
      additional practice. These are identified through analysis of your answer
      patterns and accuracy rates.
    </Text>
    <Text size='sm' c='dimmed'>
      Focusing on weak areas is crucial for balanced learning and preventing
      knowledge gaps that could hinder your progress to higher levels.
    </Text>
    <Text size='sm' fw={600}>
      How weak areas are identified:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Lower accuracy rates in specific topics</List.Item>
      <List.Item>Consistent mistakes in certain areas</List.Item>
      <List.Item>Topics you've avoided or struggled with</List.Item>
      <List.Item>Gaps in foundational knowledge</List.Item>
    </List>
    <Text size='sm' fw={600}>
      Benefits of practicing weak areas:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Builds stronger foundation</List.Item>
      <List.Item>Prevents knowledge gaps</List.Item>
      <List.Item>Improves overall performance</List.Item>
      <List.Item>Enables progression to higher levels</List.Item>
    </List>
  </Stack>
);

const PracticeWeakAreasNotImplementedInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Practice Weak Areas</Title>
    <Text size='sm'>This feature is not implemented yet.</Text>
  </Stack>
);

const RecentActivityInfo: React.FC = () => (
  <Stack gap='md'>
    <Title order={4}>Recent Activity</Title>
    <Text size='sm'>
      A chronological list of your most recent question attempts, showing your
      learning activity and progress over time.
    </Text>
    <Text size='sm' c='dimmed'>
      This helps you track your learning patterns, see your improvement over
      time, and identify any trends in your performance.
    </Text>
    <Text size='sm' fw={600}>
      What you'll see:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>
        <strong>✅ Green check:</strong> Correct answer
      </List.Item>
      <List.Item>
        <strong>❌ Red X:</strong> Incorrect answer
      </List.Item>
      <List.Item>
        <strong>Question ID:</strong> Unique identifier for each question
      </List.Item>
      <List.Item>
        <strong>Date:</strong> When you answered the question
      </List.Item>
    </List>
    <Text size='sm' fw={600}>
      How to use this information:
    </Text>
    <List size='sm' spacing='xs'>
      <List.Item>Track your learning consistency</List.Item>
      <List.Item>Identify patterns in mistakes</List.Item>
      <List.Item>Monitor improvement over time</List.Item>
      <List.Item>Click on any question to review it</List.Item>
    </List>
  </Stack>
);

const ProgressPage: React.FC = () => {
  const navigate = useNavigate();
  const [progress, setProgress] = useState<LocalUserProgress | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [modalOpened, setModalOpened] = useState(false);
  const [modalContent, setModalContent] = useState<React.ReactNode>(null);

  useEffect(() => {
    loadProgress();
  }, []);

  const openInfoModal = (content: React.ReactNode) => {
    setModalContent(content);
    setModalOpened(true);
  };

  const loadProgress = async () => {
    try {
      const data = await getV1QuizProgress();
      setProgress(data as unknown as LocalUserProgress);
    } catch (error: unknown) {
      const message =
        (error as { response?: { data?: { error?: string } } })?.response?.data
          ?.error || 'Failed to load progress';
      showNotificationWithClean({
        title: 'Error',
        message: message,
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const formatTopic = (topic: string) => {
    return topic
      .split('_')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  };

  if (isLoading) {
    return (
      <Center py='xl'>
        <Loader size='lg' />
      </Center>
    );
  }

  if (!progress) {
    return (
      <Container size='lg' py='xl'>
        <Stack align='center' gap='md'>
          <Text c='dimmed' size='lg'>
            Unable to load progress data
          </Text>
          <Button onClick={loadProgress} variant='filled'>
            Try Again
          </Button>
        </Stack>
      </Container>
    );
  }

  return (
    <Container size='lg' py='xl'>
      <Stack gap={40}>
        <Group justify='center' gap={24}>
          {/* Current Level */}
          <Card
            shadow='sm'
            padding='md'
            radius='lg'
            withBorder
            style={{
              maxWidth: 220,
              minWidth: 180,
              background: 'var(--mantine-color-body)',
            }}
          >
            <Stack gap={4} align='center'>
              <Group gap={6} mb={2} justify='center' align='center'>
                <ThemeIcon
                  size='lg'
                  radius='xl'
                  color='primary'
                  variant='light'
                >
                  <IconTarget size={20} />
                </ThemeIcon>
                <Text size='sm' c='dimmed'>
                  Current Level
                </Text>
                <ActionIcon
                  size='xs'
                  variant='subtle'
                  color='gray'
                  onClick={() => openInfoModal(<CurrentLevelInfo />)}
                  style={{ marginLeft: 'auto' }}
                >
                  <IconInfoCircle size={14} />
                </ActionIcon>
              </Group>
              <Title order={2} size='h3' c='primary'>
                {progress.current_level}
              </Title>
            </Stack>
          </Card>

          {/* Questions Answered */}
          <Card
            shadow='sm'
            padding='md'
            radius='lg'
            withBorder
            style={{
              maxWidth: 220,
              minWidth: 180,
              background: 'var(--mantine-color-body)',
            }}
          >
            <Stack gap={4} align='center'>
              <Group gap={6} mb={2} justify='center' align='center'>
                <ThemeIcon size='lg' radius='xl' color='gray' variant='light'>
                  <IconBrain size={20} />
                </ThemeIcon>
                <Text size='sm' c='dimmed'>
                  Questions Answered
                </Text>
                <ActionIcon
                  size='xs'
                  variant='subtle'
                  color='gray'
                  onClick={() => openInfoModal(<QuestionsAnsweredInfo />)}
                  style={{ marginLeft: 'auto' }}
                >
                  <IconInfoCircle size={14} />
                </ActionIcon>
              </Group>
              <Title order={2} size='h3'>
                {progress.total_questions}
              </Title>
            </Stack>
          </Card>

          {/* Accuracy */}
          <Card
            shadow='sm'
            padding='md'
            radius='lg'
            withBorder
            style={{
              maxWidth: 220,
              minWidth: 180,
              background: 'var(--mantine-color-body)',
            }}
          >
            <Stack gap={4} align='center'>
              <Group gap={6} mb={2} justify='center' align='center'>
                <Text size='sm' c='dimmed'>
                  Accuracy
                </Text>
                <ActionIcon
                  size='xs'
                  variant='subtle'
                  color='gray'
                  onClick={() => openInfoModal(<AccuracyInfo />)}
                  style={{ marginLeft: 'auto' }}
                >
                  <IconInfoCircle size={14} />
                </ActionIcon>
              </Group>
              <RingProgress
                size={56}
                thickness={6}
                sections={[
                  { value: progress.accuracy_rate * 100, color: 'teal' },
                ]}
                label={
                  <Text ta='center' size='sm' fw={700}>
                    {(progress.accuracy_rate * 100).toFixed(1)}%
                  </Text>
                }
              />
            </Stack>
          </Card>

          {/* Suggested Level */}
          <Card
            shadow='sm'
            padding='md'
            radius='lg'
            withBorder
            style={{
              maxWidth: 220,
              minWidth: 180,
              background: 'var(--mantine-color-body)',
            }}
          >
            <Stack gap={4} align='center'>
              <Group gap={6} mb={2} justify='center' align='center'>
                <ThemeIcon
                  size='lg'
                  radius='xl'
                  color='primary'
                  variant='light'
                >
                  <IconTrophy size={20} />
                </ThemeIcon>
                <Text size='sm' c='primary'>
                  Suggested Level
                </Text>
                <ActionIcon
                  size='xs'
                  variant='subtle'
                  color='gray'
                  onClick={() => openInfoModal(<SuggestedLevelInfo />)}
                  style={{ marginLeft: 'auto' }}
                >
                  <IconInfoCircle size={14} />
                </ActionIcon>
              </Group>
              <Title order={2} size='h3' c='primary'>
                {progress.suggested_level || progress.current_level}
              </Title>
              <Text size='xs' c='primary' mt={-4}>
                {progress.suggested_level
                  ? 'We suggest moving to this level!'
                  : 'Keep up the great work!'}
              </Text>
            </Stack>
          </Card>
        </Group>

        {/* Performance by Topic - full width */}
        <Card
          shadow='sm'
          padding='xl'
          radius='lg'
          withBorder
          style={{ background: 'var(--mantine-color-body)' }}
        >
          <Stack gap='md'>
            <Group justify='space-between' align='center' mb={2}>
              <Title order={3} size='h4'>
                Performance by Topic
              </Title>
              <ActionIcon
                size='sm'
                variant='subtle'
                color='gray'
                onClick={() => openInfoModal(<PerformanceByTopicInfo />)}
              >
                <IconInfoCircle size={16} />
              </ActionIcon>
            </Group>
            <Divider mb={4} />
            <Table striped highlightOnHover withColumnBorders>
              <Table.Thead
                style={{ background: 'var(--mantine-color-table-th-bg)' }}
              >
                <Table.Tr>
                  <Table.Th>Topic</Table.Th>
                  <Table.Th>Attempts</Table.Th>
                  <Table.Th>Correct</Table.Th>
                  <Table.Th>Accuracy</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {progress.performance_by_topic &&
                Object.entries(progress.performance_by_topic).length > 0 ? (
                  Object.entries(progress.performance_by_topic).map(
                    ([topic, metrics]: [string, PerformanceMetrics]) => {
                      const accuracy =
                        (metrics.correct_attempts / metrics.total_attempts) *
                        100;
                      return (
                        <Table.Tr key={topic}>
                          <Table.Td>{formatTopic(topic)}</Table.Td>
                          <Table.Td>{metrics.total_attempts}</Table.Td>
                          <Table.Td>{metrics.correct_attempts}</Table.Td>
                          <Table.Td>
                            <Group gap={4} align='center'>
                              <Text size='sm'>{accuracy.toFixed(1)}%</Text>
                              <Progress
                                value={accuracy}
                                size='sm'
                                w={60}
                                color={
                                  accuracy >= 80
                                    ? 'teal'
                                    : accuracy >= 60
                                      ? 'yellow'
                                      : 'error'
                                }
                                radius='xl'
                              />
                            </Group>
                          </Table.Td>
                        </Table.Tr>
                      );
                    }
                  )
                ) : (
                  <Table.Tr>
                    <Table.Td colSpan={4}>
                      <Text ta='center' c='dimmed' py='md'>
                        No performance data available yet. Answer some questions
                        to see your topic performance!
                      </Text>
                    </Table.Td>
                  </Table.Tr>
                )}
              </Table.Tbody>
            </Table>
          </Stack>
        </Card>

        {/* Learning System Insights */}
        {progress.worker_status ||
        progress.learning_preferences ||
        progress.priority_insights ||
        progress.generation_focus ? (
          <Card
            shadow='sm'
            padding='xl'
            radius='lg'
            withBorder
            style={{ background: 'var(--mantine-color-body)' }}
          >
            <Stack gap='md'>
              <Group justify='space-between' align='center' mb={2}>
                <Title order={3} size='h4'>
                  Learning System Insights
                </Title>
              </Group>
              <Divider mb={4} />

              <Group align='flex-start' gap={24} wrap='wrap'>
                {/* Worker Status */}
                {progress.worker_status && (
                  <Card
                    shadow='sm'
                    padding='md'
                    radius='md'
                    withBorder
                    style={{
                      flex: 1,
                      minWidth: 250,
                      background: 'var(--mantine-color-body)',
                    }}
                  >
                    <Stack gap='sm'>
                      <Group gap={6} align='center'>
                        <ThemeIcon
                          size='md'
                          radius='xl'
                          color={
                            progress.worker_status.status === 'busy'
                              ? 'blue'
                              : progress.worker_status.status === 'error'
                                ? 'red'
                                : 'green'
                          }
                          variant='light'
                        >
                          <IconBrain size={16} />
                        </ThemeIcon>
                        <Text size='sm' fw={600}>
                          Worker Status
                        </Text>
                        <ActionIcon
                          size='xs'
                          variant='subtle'
                          color='gray'
                          onClick={() => openInfoModal(<WorkerStatusInfo />)}
                          style={{ marginLeft: 'auto' }}
                        >
                          <IconInfoCircle size={12} />
                        </ActionIcon>
                      </Group>
                      <Text
                        size='lg'
                        fw={700}
                        c={
                          progress.worker_status.status === 'busy'
                            ? 'blue'
                            : progress.worker_status.status === 'error'
                              ? 'red'
                              : 'green'
                        }
                      >
                        {progress.worker_status.status === 'busy'
                          ? 'Active'
                          : progress.worker_status.status === 'error'
                            ? 'Error'
                            : 'Idle'}
                      </Text>
                      {progress.worker_status.error_message && (
                        <Text size='xs' c='red'>
                          {progress.worker_status.error_message}
                        </Text>
                      )}
                      {progress.worker_status.last_heartbeat && (
                        <Text size='xs' c='dimmed'>
                          Last active:{' '}
                          {new Date(
                            progress.worker_status.last_heartbeat
                          ).toLocaleString()}
                        </Text>
                      )}
                    </Stack>
                  </Card>
                )}

                {/* Learning Preferences */}
                {progress.learning_preferences && (
                  <Card
                    shadow='sm'
                    padding='md'
                    radius='md'
                    withBorder
                    style={{
                      flex: 1,
                      minWidth: 250,
                      background: 'var(--mantine-color-body)',
                    }}
                  >
                    <Stack gap='sm'>
                      <Group gap={6} align='center'>
                        <ThemeIcon
                          size='md'
                          radius='xl'
                          color='yellow'
                          variant='light'
                        >
                          <IconTarget size={16} />
                        </ThemeIcon>
                        <Text size='sm' fw={600}>
                          Learning Focus
                        </Text>
                        <ActionIcon
                          size='xs'
                          variant='subtle'
                          color='gray'
                          onClick={() =>
                            openInfoModal(<LearningPreferencesInfo />)
                          }
                          style={{ marginLeft: 'auto' }}
                        >
                          <IconInfoCircle size={12} />
                        </ActionIcon>
                      </Group>
                      <Text size='sm'>
                        {progress.learning_preferences.focus_on_weak_areas
                          ? 'Focusing on weak areas'
                          : 'Balanced learning'}
                      </Text>
                      <Text size='xs' c='dimmed'>
                        Fresh questions:{' '}
                        {Math.round(
                          progress.learning_preferences.fresh_question_ratio *
                            100
                        )}
                        %
                      </Text>
                      <Text size='xs' c='dimmed'>
                        Review interval:{' '}
                        {progress.learning_preferences.review_interval_days}{' '}
                        days
                      </Text>
                    </Stack>
                  </Card>
                )}

                {/* Priority Insights */}
                {progress.priority_insights && (
                  <Card
                    shadow='sm'
                    padding='md'
                    radius='md'
                    withBorder
                    style={{
                      flex: 1,
                      minWidth: 250,
                      background: 'var(--mantine-color-body)',
                    }}
                  >
                    <Stack gap='sm'>
                      <Group gap={6} align='center'>
                        <ThemeIcon
                          size='md'
                          radius='xl'
                          color='purple'
                          variant='light'
                        >
                          <IconTrophy size={16} />
                        </ThemeIcon>
                        <Text size='sm' fw={600}>
                          Question Priority
                        </Text>
                        <ActionIcon
                          size='xs'
                          variant='subtle'
                          color='gray'
                          onClick={() =>
                            openInfoModal(<PriorityInsightsInfo />)
                          }
                          style={{ marginLeft: 'auto' }}
                        >
                          <IconInfoCircle size={12} />
                        </ActionIcon>
                      </Group>
                      <Text size='lg' fw={700}>
                        {progress.priority_insights.total_questions_in_queue ||
                          0}
                      </Text>
                      <Text size='xs' c='dimmed'>
                        High:{' '}
                        {progress.priority_insights.high_priority_questions ||
                          0}{' '}
                        | Medium:{' '}
                        {progress.priority_insights.medium_priority_questions ||
                          0}{' '}
                        | Low:{' '}
                        {progress.priority_insights.low_priority_questions || 0}
                      </Text>
                    </Stack>
                  </Card>
                )}

                {/* Generation Focus */}
                {progress.generation_focus && (
                  <Card
                    shadow='sm'
                    padding='md'
                    radius='md'
                    withBorder
                    style={{
                      flex: 1,
                      minWidth: 250,
                      background: 'var(--mantine-color-body)',
                    }}
                  >
                    <Stack gap='sm'>
                      <Group gap={6} align='center'>
                        <ThemeIcon
                          size='md'
                          radius='xl'
                          color='teal'
                          variant='light'
                        >
                          <IconBrain size={16} />
                        </ThemeIcon>
                        <Text size='sm' fw={600}>
                          AI Generation
                        </Text>
                        <ActionIcon
                          size='xs'
                          variant='subtle'
                          color='gray'
                          onClick={() => openInfoModal(<GenerationFocusInfo />)}
                          style={{ marginLeft: 'auto' }}
                        >
                          <IconInfoCircle size={12} />
                        </ActionIcon>
                      </Group>
                      <Text size='sm' fw={600}>
                        {progress.generation_focus.current_generation_model ||
                          'Default'}
                      </Text>
                      {progress.generation_focus.generation_rate && (
                        <Text size='xs' c='dimmed'>
                          {progress.generation_focus.generation_rate.toFixed(1)}{' '}
                          questions/min
                        </Text>
                      )}
                      {progress.generation_focus.last_generation_time && (
                        <Text size='xs' c='dimmed'>
                          Last:{' '}
                          {new Date(
                            progress.generation_focus.last_generation_time
                          ).toLocaleString()}
                        </Text>
                      )}
                    </Stack>
                  </Card>
                )}
              </Group>
            </Stack>
          </Card>
        ) : null}

        {/* Areas to Improve & Recent Activity - side by side */}
        <Group align='flex-start' gap={24} wrap='wrap'>
          {/* Areas to Improve */}
          <Card
            shadow='sm'
            padding='lg'
            radius='lg'
            withBorder
            style={{
              flex: 1,
              minWidth: 320,
              maxWidth: 400,
              background: 'var(--mantine-color-body)',
            }}
          >
            <Stack gap='md'>
              <Group justify='space-between' align='center' mb={2}>
                <Title order={3} size='h4'>
                  Areas to Improve
                </Title>
                <ActionIcon
                  size='sm'
                  variant='subtle'
                  color='gray'
                  onClick={() => openInfoModal(<WeakAreasInfo />)}
                >
                  <IconInfoCircle size={16} />
                </ActionIcon>
              </Group>
              <Divider mb={4} />
              {progress.weak_areas && progress.weak_areas.length > 0 ? (
                <List
                  spacing='xs'
                  size='sm'
                  center
                  icon={
                    <ThemeIcon color='error' size={20} radius='xl'>
                      <IconX size={12} />
                    </ThemeIcon>
                  }
                >
                  {progress.weak_areas.map((area: string, index: number) => (
                    <List.Item key={index}>{formatTopic(area)}</List.Item>
                  ))}
                </List>
              ) : (
                <Text c='dimmed'>
                  No specific weak areas found. Keep it up!
                </Text>
              )}
              <Button
                onClick={() =>
                  openInfoModal(<PracticeWeakAreasNotImplementedInfo />)
                }
                variant='filled'
                size='md'
                fullWidth
                mt='sm'
                disabled={
                  !progress.weak_areas || progress.weak_areas.length === 0
                }
              >
                Practice Weak Areas
              </Button>
            </Stack>
          </Card>

          {/* Recent Activity */}
          <Card
            shadow='sm'
            padding='lg'
            radius='lg'
            withBorder
            style={{
              flex: 2,
              minWidth: 320,
              background: 'var(--mantine-color-body)',
            }}
          >
            <Stack gap='md'>
              <Group justify='space-between' align='center' mb={2}>
                <Title order={3} size='h4'>
                  Recent Activity
                </Title>
                <ActionIcon
                  size='sm'
                  variant='subtle'
                  color='gray'
                  onClick={() => openInfoModal(<RecentActivityInfo />)}
                >
                  <IconInfoCircle size={16} />
                </ActionIcon>
              </Group>
              <Divider mb={4} />
              {progress.recent_activity &&
              progress.recent_activity.length > 0 ? (
                <Stack gap={4}>
                  {progress.recent_activity.map((activity: UserResponse) => (
                    <RecentActivityItem
                      key={activity.question_id}
                      activity={activity}
                      navigate={navigate}
                    />
                  ))}
                </Stack>
              ) : (
                <Text c='dimmed' size='sm'>
                  No recent activity found. Start answering questions to see
                  your progress!
                </Text>
              )}
            </Stack>
          </Card>
        </Group>
      </Stack>

      {/* Info Modal */}
      <Modal
        opened={modalOpened}
        onClose={() => setModalOpened(false)}
        title='Information'
        size='lg'
        centered
      >
        <Box p='md'>{modalContent}</Box>
      </Modal>
    </Container>
  );
};

export default ProgressPage;
