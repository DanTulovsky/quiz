import React, { useState, useEffect, useCallback } from 'react';
import {
  Container,
  Title,
  Text,
  Select,
  Button,
  Group,
  Paper,
  Stack,
  Badge,
  Modal,
  Box,
  Grid,
  Card,
  LoadingOverlay,
  Tabs,
  Table,
  ActionIcon,
} from '@mantine/core';
import {
  IconRefresh,
  IconX,
  IconInfoCircle,
  IconPlus,
  IconTrash,
} from '@tabler/icons-react';
import { useAuth } from '../../hooks/useAuth';
import { Navigate } from 'react-router-dom';
import { notifications } from '@mantine/notifications';
import { useUsersPaginated } from '../../api/admin';
import logger from '../../utils/logger';

interface User {
  id: number;
  username: string;
}

interface UserAnalytics {
  distribution: {
    high: number;
    medium: number;
    low: number;
    average: number;
  };
  highPriorityQuestions: Array<{
    priority_score: number;
    question_type: string;
    level: string;
    topic: string;
  }>;
  weakAreas: Array<{
    topic: string;
    correct_attempts: number;
    total_attempts: number;
  }>;
  learningPreferences: {
    focus_on_weak_areas: boolean;
    fresh_question_ratio: number;
    weak_area_boost: number;
    known_question_penalty: number;
  };
}

interface AggregateAnalytics {
  distribution: {
    high: number;
    medium: number;
    low: number;
    average: number;
  };
  performance: {
    calculationsPerSecond: number;
    avgCalculationTime: number;
    memoryUsage: number;
    lastCalculation: string;
  };
  preferencesUsage: {
    vocabulary: number;
    reading_comprehension: number;
    fill_in_blank: number;
    question_answer: number;
  };
  weakAreasByTopic: Array<{
    topic: string;
    avg_score: number;
  }>;
}

interface ComparisonData {
  user: User;
  distribution: {
    high: number;
    medium: number;
    low: number;
    average: number;
  };
  weakAreas: Array<{
    topic: string;
    correct_attempts: number;
    total_attempts: number;
  }>;
}

const AnalyticsPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const [activeTab, setActiveTab] = useState<string>('individual');
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [selectedUser, setSelectedUser] = useState<string>('');
  const [userAnalytics, setUserAnalytics] = useState<UserAnalytics | null>(
    null
  );
  const [aggregateAnalytics, setAggregateAnalytics] =
    useState<AggregateAnalytics | null>(null);
  const [selectedUsers, setSelectedUsers] = useState<User[]>([]);
  const [comparisonData, setComparisonData] = useState<ComparisonData[]>([]);
  const [loading, setLoading] = useState(false);
  const [infoModal, setInfoModal] = useState<{
    open: boolean;
    content: string;
  }>({
    open: false,
    content: '',
  });

  // Info content for tooltips
  const infoContent: Record<string, string> = {
    'priority-distribution': `
      <h2>Priority Score Distribution</h2>
      <p><b>What is a Priority Score?</b><br>
      A <b>priority score</b> is a number assigned to each question for each user. It represents how urgently the system thinks you should review or practice that question. Higher scores mean the question is more important for you to see soon, based on your past answers, accuracy, and learning preferences.</p>
      <p>This box shows how many of your questions are considered <b>high</b> (score &gt; 200), <b>medium</b> (100-200), or <b>low</b> (&lt; 100) priority. The <b>average score</b> gives a sense of your overall review urgency.</p>
      <ul>
        <li><b>High Priority:</b> Questions you should review soon.</li>
        <li><b>Medium Priority:</b> Questions to review, but less urgent.</li>
        <li><b>Low Priority:</b> Questions you know well or have seen recently.</li>
      </ul>
      <p><b>Why does this matter?</b><br>
      The system uses these scores to decide which questions to show you next, helping you focus on what you need most.</p>
    `,
    'high-priority-questions': `
      <h2>High Priority Questions</h2>
      <p>This box lists your questions with the <b>highest priority scores</b>. These are the questions the system believes are most important for you to review right now.</p>
      <p>Each entry shows:</p>
      <ul>
        <li><b>Score:</b> The priority score (higher = more important)</li>
        <li><b>Type/Level:</b> Question type and difficulty level</li>
        <li><b>Topic:</b> The subject area of the question</li>
      </ul>
      <p>These questions should be your focus for the next study session.</p>
    `,
    'weak-areas': `
      <h2>Weak Areas</h2>
      <p>This section identifies topics where you're <b>struggling the most</b> based on your answer accuracy.</p>
      <p>For each topic, you'll see:</p>
      <ul>
        <li><b>Accuracy Percentage:</b> How often you get questions right in this topic</li>
        <li><b>Attempt Count:</b> Total questions answered in this topic</li>
      </ul>
      <p><b>Lower accuracy percentages</b> indicate areas that need more practice. The system will prioritize these topics when generating new questions for you.</p>
    `,
    'learning-preferences': `
      <h2>Learning Preferences</h2>
      <p>This shows your current <b>learning strategy settings</b> and how they affect question selection:</p>
      <ul>
        <li><b>Focus on Weak Areas:</b> Whether the system prioritizes your struggling topics</li>
        <li><b>Fresh Question Ratio:</b> Percentage of new questions vs. review questions</li>
        <li><b>Weak Area Boost:</b> Multiplier for questions in your weak areas</li>
        <li><b>Known Question Penalty:</b> Reduction for questions you've marked as known</li>
      </ul>
      <p>These settings help personalize your learning experience.</p>
    `,
    'aggregate-priority-distribution': `
      <h2>Overall Priority Distribution</h2>
      <p>This shows the <b>system-wide distribution</b> of priority scores across all users and questions.</p>
      <p>It helps administrators understand:</p>
      <ul>
        <li>How many questions are high priority across the system</li>
        <li>Whether the priority system is working effectively</li>
        <li>If there are enough questions available at different priority levels</li>
      </ul>
      <p>A healthy distribution should have a good mix of high, medium, and low priority questions.</p>
    `,
    'system-performance': `
      <h2>System Performance</h2>
      <p>This tracks the <b>technical performance</b> of the priority calculation system:</p>
      <ul>
        <li><b>Calculations/sec:</b> How many priority scores are calculated per second</li>
        <li><b>Avg Calculation Time:</b> Average time to calculate one priority score</li>
        <li><b>Memory Usage:</b> How much memory the system is using</li>
        <li><b>Last Calculation:</b> When the system last updated priority scores</li>
      </ul>
      <p>These metrics help ensure the system is running efficiently.</p>
    `,
    'aggregate-learning-preferences': `
      <h2>Learning Preferences Usage</h2>
      <p>This shows how users are <b>distributed across different question types</b>:</p>
      <ul>
        <li><b>Vocabulary:</b> Percentage of vocabulary questions being used</li>
        <li><b>Reading Comprehension:</b> Percentage of reading questions</li>
        <li><b>Fill-in-the-Blank:</b> Percentage of fill-in-blank questions</li>
        <li><b>Question-Answer:</b> Percentage of Q&A questions</li>
      </ul>
      <p>This helps ensure a balanced learning experience across all question types.</p>
    `,
    'aggregate-weak-areas': `
      <h2>Weak Areas by Topic</h2>
      <p>This shows the <b>most common weak areas</b> across all users:</p>
      <ul>
        <li><b>Topic:</b> The subject area</li>
        <li><b>Average Score:</b> Average performance across all users in this topic</li>
      </ul>
      <p>Topics with lower average scores indicate areas where most users struggle, suggesting these topics might need more question generation or different approaches.</p>
    `,
    'comparison-priority-distribution': `
      <h2>Priority Distribution Comparison</h2>
      <p>This table compares <b>priority score distributions</b> between selected users:</p>
      <ul>
        <li><b>High/Medium/Low:</b> Number of questions in each priority category</li>
        <li><b>Average:</b> Average priority score for each user</li>
      </ul>
      <p>This helps identify which users have more urgent review needs and how the priority system is working for different users.</p>
    `,
    'comparison-weak-areas': `
      <h2>Weak Areas Comparison</h2>
      <p>This table compares <b>performance by topic</b> between selected users:</p>
      <ul>
        <li><b>Area:</b> The topic being compared</li>
        <li><b>Accuracy:</b> Percentage of correct answers for each user</li>
      </ul>
      <p>This helps identify which topics are challenging for different users and how learning patterns vary between individuals.</p>
    `,
  };

  // API hooks
  const { data: usersData, error: usersError } = useUsersPaginated({
    page: 1,
    pageSize: 1000, // Get all users for analytics
  });

  const loadUserAnalytics = useCallback(async () => {
    if (!selectedUser) return;

    setLoading(true);
    try {
      const response = await fetch(
        `/v1/admin/worker/analytics/user/${selectedUser}`
      );
      if (!response.ok) throw new Error('Failed to fetch user analytics');
      const data = await response.json();
      setUserAnalytics(data);
    } catch (error) {
      logger.error('Error loading user analytics:', error);
      notifications.show({
        title: 'Error',
        message: 'Failed to load user analytics',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [selectedUser]);

  const loadAggregateAnalytics = useCallback(async () => {
    setLoading(true);
    try {
      const [priorityResponse, perfResponse, prefsResponse, weakResponse] =
        await Promise.all([
          fetch('/v1/admin/worker/analytics/priority-scores'),
          fetch('/v1/admin/worker/analytics/system-health'),
          fetch('/v1/admin/worker/analytics/user-performance'),
          fetch('/v1/admin/worker/analytics/user-performance'),
        ]);

      if (!priorityResponse.ok)
        throw new Error('Failed to fetch aggregate analytics');

      const priorityData = await priorityResponse.json();
      const perfData = perfResponse.ok ? await perfResponse.json() : {};
      const prefsData = prefsResponse.ok ? await prefsResponse.json() : {};
      const weakData = weakResponse.ok ? await weakResponse.json() : {};

      setAggregateAnalytics({
        distribution: priorityData.distribution || {},
        performance: perfData.performance || {},
        preferencesUsage: prefsData.learningPreferences || {},
        weakAreasByTopic: weakData.weakAreas || [],
      });
    } catch (error) {
      logger.error('Error loading aggregate analytics:', error);
      notifications.show({
        title: 'Error',
        message: 'Failed to load aggregate analytics',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, []);

  const loadComparison = useCallback(async () => {
    if (selectedUsers.length === 0) return;

    setLoading(true);
    try {
      const userIds = selectedUsers.map(u => u.id).join(',');
      const response = await fetch(
        `/v1/admin/worker/analytics/comparison?user_ids=${userIds}`
      );
      if (!response.ok) throw new Error('Failed to fetch comparison data');
      const data = await response.json();
      setComparisonData(data.comparison || []);
    } catch (error) {
      logger.error('Error loading comparison:', error);
      notifications.show({
        title: 'Error',
        message: 'Failed to load comparison data',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [selectedUsers]);

  const addUserToComparison = () => {
    const user = allUsers.find(u => u.id?.toString() === selectedUser);
    if (user && !selectedUsers.find(u => u.id === user.id)) {
      setSelectedUsers([...selectedUsers, user]);
    }
  };

  const removeUserFromComparison = (userId: number) => {
    setSelectedUsers(selectedUsers.filter(u => u.id !== userId));
  };

  const clearComparison = () => {
    setSelectedUsers([]);
    setComparisonData([]);
  };

  const showInfoModal = (type: string) => {
    setInfoModal({
      open: true,
      content: infoContent[type] || 'Information not available',
    });
  };

  // Effects
  useEffect(() => {
    logger.debug('Users data:', usersData);
    if (usersData?.users) {
      logger.debug('Setting allUsers:', usersData.users);
      // Extract user data (now directly in the array)
      const extractedUsers = usersData.users as User[];
      logger.debug('Extracted users:', extractedUsers);
      setAllUsers(extractedUsers);
    }
  }, [usersData]);

  useEffect(() => {
    if (usersError) {
      logger.error('Error loading users:', usersError);
      notifications.show({
        title: 'Error',
        message: 'Failed to load users',
        color: 'red',
      });
    }
  }, [usersError]);

  useEffect(() => {
    if (selectedUser) {
      loadUserAnalytics();
    }
  }, [selectedUser, loadUserAnalytics]);

  useEffect(() => {
    if (selectedUsers.length > 0) {
      loadComparison();
    }
  }, [selectedUsers, loadComparison]);

  // Check if user is admin
  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }

  const isAdmin = user.roles?.some(role => role.name === 'admin') || false;
  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  return (
    <Container size='xl' py='md'>
      <LoadingOverlay visible={loading} />

      <Title order={1} mb='lg'>
        Priority System Analytics
      </Title>
      <Text c='dimmed' mb='lg'>
        Comprehensive analytics for the priority system, including individual
        user data and aggregate metrics.
      </Text>

      <Tabs
        value={activeTab}
        onChange={value => setActiveTab(value || 'individual')}
      >
        <Tabs.List>
          <Tabs.Tab value='individual'>Individual User</Tabs.Tab>
          <Tabs.Tab value='aggregate'>Aggregate Data</Tabs.Tab>
          <Tabs.Tab value='comparison'>User Comparison</Tabs.Tab>
        </Tabs.List>

        {/* Individual User Tab */}
        <Tabs.Panel value='individual' pt='md'>
          <Paper p='md' withBorder>
            <Title order={2} mb='md'>
              Individual User Analytics
            </Title>

            <Group mb='md'>
              <Select
                label='Select User:'
                placeholder='Select a user...'
                data={allUsers
                  .filter(u => u.id != null)
                  .map(u => ({
                    value: u.id.toString(),
                    label: u.username || 'Unknown User',
                  }))}
                value={selectedUser}
                onChange={setSelectedUser}
                searchable
                clearable
                nothingFound='No users found'
                style={{ minWidth: 200 }}
              />
              <Button
                onClick={loadUserAnalytics}
                leftSection={<IconRefresh size={14} />}
              >
                Refresh
              </Button>
            </Group>

            {userAnalytics && (
              <Grid>
                <Grid.Col span={6}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>Priority Score Distribution</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() => showInfoModal('priority-distribution')}
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    <Stack gap='xs'>
                      <Text>
                        <strong>High Priority (&gt;200):</strong>{' '}
                        {userAnalytics.distribution.high || 0}
                      </Text>
                      <Text>
                        <strong>Medium Priority (100-200):</strong>{' '}
                        {userAnalytics.distribution.medium || 0}
                      </Text>
                      <Text>
                        <strong>Low Priority (&lt;100):</strong>{' '}
                        {userAnalytics.distribution.low || 0}
                      </Text>
                      <Text>
                        <strong>Average Score:</strong>{' '}
                        {userAnalytics.distribution.average?.toFixed(2) ||
                          'N/A'}
                      </Text>
                    </Stack>
                  </Card>
                </Grid.Col>

                <Grid.Col span={6}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>High Priority Questions</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() => showInfoModal('high-priority-questions')}
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    {userAnalytics.highPriorityQuestions &&
                    userAnalytics.highPriorityQuestions.length > 0 ? (
                      <Stack gap='xs'>
                        {userAnalytics.highPriorityQuestions
                          .slice(0, 5)
                          .map((q, index) => (
                            <Text key={index} size='sm'>
                              <strong>
                                Score {q.priority_score?.toFixed(1)}:
                              </strong>{' '}
                              {q.question_type}/{q.level} -{' '}
                              {q.topic || 'No topic'}
                            </Text>
                          ))}
                      </Stack>
                    ) : (
                      <Text c='dimmed' fs='italic'>
                        No high priority questions
                      </Text>
                    )}
                  </Card>
                </Grid.Col>

                <Grid.Col span={6}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>Weak Areas</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() => showInfoModal('weak-areas')}
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    {userAnalytics.weakAreas &&
                    userAnalytics.weakAreas.length > 0 ? (
                      <Stack gap='xs'>
                        {userAnalytics.weakAreas
                          .slice(0, 5)
                          .map((area, index) => {
                            const accuracy =
                              area.total_attempts > 0
                                ? (
                                    (area.correct_attempts /
                                      area.total_attempts) *
                                    100
                                  ).toFixed(1)
                                : 0;
                            return (
                              <Text key={index} size='sm'>
                                <strong>{area.topic || 'Unknown'}:</strong>{' '}
                                {accuracy}% ({area.correct_attempts}/
                                {area.total_attempts})
                              </Text>
                            );
                          })}
                      </Stack>
                    ) : (
                      <Text c='dimmed' fs='italic'>
                        No weak areas data
                      </Text>
                    )}
                  </Card>
                </Grid.Col>

                <Grid.Col span={6}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>Learning Preferences</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() => showInfoModal('learning-preferences')}
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    {userAnalytics.learningPreferences ? (
                      <Stack gap='xs'>
                        <Text>
                          <strong>Focus on Weak Areas:</strong>{' '}
                          {userAnalytics.learningPreferences.focus_on_weak_areas
                            ? 'Yes'
                            : 'No'}
                        </Text>
                        <Text>
                          <strong>Fresh Question Ratio:</strong>{' '}
                          {(
                            userAnalytics.learningPreferences
                              .fresh_question_ratio * 100
                          ).toFixed(1)}
                          %
                        </Text>
                        <Text>
                          <strong>Weak Area Boost:</strong>{' '}
                          {userAnalytics.learningPreferences.weak_area_boost}
                        </Text>
                        <Text>
                          <strong>Known Question Penalty:</strong>{' '}
                          {
                            userAnalytics.learningPreferences
                              .known_question_penalty
                          }
                        </Text>
                      </Stack>
                    ) : (
                      <Text c='dimmed' fs='italic'>
                        No preferences data
                      </Text>
                    )}
                  </Card>
                </Grid.Col>
              </Grid>
            )}
          </Paper>
        </Tabs.Panel>

        {/* Aggregate Data Tab */}
        <Tabs.Panel value='aggregate' pt='md'>
          <Paper p='md' withBorder>
            <Group justify='space-between' mb='md'>
              <Title order={2}>Aggregate Analytics</Title>
              <Button
                onClick={loadAggregateAnalytics}
                leftSection={<IconRefresh size={14} />}
              >
                Refresh Data
              </Button>
            </Group>

            {aggregateAnalytics && (
              <Grid>
                <Grid.Col span={6}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>Overall Priority Distribution</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() =>
                          showInfoModal('aggregate-priority-distribution')
                        }
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    <Stack gap='xs'>
                      <Text>
                        <strong>High Priority (&gt;200):</strong>{' '}
                        {aggregateAnalytics.distribution.high || 0}
                      </Text>
                      <Text>
                        <strong>Medium Priority (100-200):</strong>{' '}
                        {aggregateAnalytics.distribution.medium || 0}
                      </Text>
                      <Text>
                        <strong>Low Priority (&lt;100):</strong>{' '}
                        {aggregateAnalytics.distribution.low || 0}
                      </Text>
                      <Text>
                        <strong>Average Score:</strong>{' '}
                        {aggregateAnalytics.distribution.average?.toFixed(2) ||
                          'N/A'}
                      </Text>
                    </Stack>
                  </Card>
                </Grid.Col>

                <Grid.Col span={6}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>System Performance</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() => showInfoModal('system-performance')}
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    <Stack gap='xs'>
                      <Text>
                        <strong>Calculations/sec:</strong>{' '}
                        {aggregateAnalytics.performance.calculationsPerSecond?.toFixed(
                          2
                        ) || 'N/A'}
                      </Text>
                      <Text>
                        <strong>Avg Calculation Time:</strong>{' '}
                        {aggregateAnalytics.performance.avgCalculationTime?.toFixed(
                          2
                        ) || 'N/A'}
                        ms
                      </Text>
                      <Text>
                        <strong>Memory Usage:</strong>{' '}
                        {aggregateAnalytics.performance.memoryUsage?.toFixed(
                          1
                        ) || 'N/A'}
                        MB
                      </Text>
                      <Text>
                        <strong>Last Calculation:</strong>{' '}
                        {aggregateAnalytics.performance.lastCalculation ||
                          'N/A'}
                      </Text>
                    </Stack>
                  </Card>
                </Grid.Col>

                <Grid.Col span={6}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>Learning Preferences Usage</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() =>
                          showInfoModal('aggregate-learning-preferences')
                        }
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    <Stack gap='xs'>
                      <Text>
                        <strong>Vocabulary:</strong>{' '}
                        {aggregateAnalytics.preferencesUsage.vocabulary || 0}%
                      </Text>
                      <Text>
                        <strong>Reading Comprehension:</strong>{' '}
                        {aggregateAnalytics.preferencesUsage
                          .reading_comprehension || 0}
                        %
                      </Text>
                      <Text>
                        <strong>Fill-in-the-Blank:</strong>{' '}
                        {aggregateAnalytics.preferencesUsage.fill_in_blank || 0}
                        %
                      </Text>
                      <Text>
                        <strong>Question-Answer:</strong>{' '}
                        {aggregateAnalytics.preferencesUsage.question_answer ||
                          0}
                        %
                      </Text>
                    </Stack>
                  </Card>
                </Grid.Col>

                <Grid.Col span={6}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>Weak Areas by Topic</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() => showInfoModal('aggregate-weak-areas')}
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    {aggregateAnalytics.weakAreasByTopic &&
                    aggregateAnalytics.weakAreasByTopic.length > 0 ? (
                      <Stack gap='xs'>
                        {aggregateAnalytics.weakAreasByTopic
                          .slice(0, 5)
                          .map((area, index) => (
                            <Text key={index} size='sm'>
                              <strong>{area.topic || 'Unknown'}:</strong>{' '}
                              {area.avg_score?.toFixed(1) || 'N/A'} avg score
                            </Text>
                          ))}
                      </Stack>
                    ) : (
                      <Text c='dimmed' fs='italic'>
                        No weak areas data
                      </Text>
                    )}
                  </Card>
                </Grid.Col>
              </Grid>
            )}
          </Paper>
        </Tabs.Panel>

        {/* User Comparison Tab */}
        <Tabs.Panel value='comparison' pt='md'>
          <Paper p='md' withBorder>
            <Title order={2} mb='md'>
              User Comparison
            </Title>

            <Group mb='md'>
              <Select
                label='Add User:'
                placeholder='Select user...'
                data={allUsers
                  .filter(u => u.id != null)
                  .map(u => ({
                    value: u.id.toString(),
                    label: u.username,
                  }))}
                value={selectedUser}
                onChange={setSelectedUser}
                style={{ minWidth: 200 }}
              />
              <Button
                onClick={addUserToComparison}
                leftSection={<IconPlus size={14} />}
              >
                Add
              </Button>
              <Button
                onClick={clearComparison}
                variant='outline'
                leftSection={<IconTrash size={14} />}
              >
                Clear All
              </Button>
            </Group>

            {selectedUsers.length > 0 && (
              <Box mb='md'>
                <Text size='sm' fw={500} mb='xs'>
                  Selected Users:
                </Text>
                <Group gap='xs'>
                  {selectedUsers.map(user => (
                    <Badge
                      key={user.id}
                      rightSection={
                        <ActionIcon
                          size='xs'
                          variant='subtle'
                          onClick={() => removeUserFromComparison(user.id)}
                        >
                          <IconX size={10} />
                        </ActionIcon>
                      }
                    >
                      {user.username}
                    </Badge>
                  ))}
                </Group>
              </Box>
            )}

            {comparisonData.length > 0 && (
              <Grid>
                <Grid.Col span={12}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>Priority Distribution Comparison</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() =>
                          showInfoModal('comparison-priority-distribution')
                        }
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    <Table>
                      <Table.Thead>
                        <Table.Tr>
                          <Table.Th>Metric</Table.Th>
                          {comparisonData.map(u => (
                            <Table.Th key={u.user.id}>
                              {u.user.username}
                            </Table.Th>
                          ))}
                        </Table.Tr>
                      </Table.Thead>
                      <Table.Tbody>
                        {[
                          { key: 'high', label: 'High', color: 'green' },
                          { key: 'medium', label: 'Medium', color: 'yellow' },
                          { key: 'low', label: 'Low', color: 'red' },
                          { key: 'average', label: 'Avg', color: 'blue' },
                        ].map(({ key, label, color }) => (
                          <Table.Tr key={key}>
                            <Table.Td>
                              <Badge color={color}>{label}</Badge>
                            </Table.Td>
                            {comparisonData.map(u => {
                              const val =
                                u.distribution[
                                  key as keyof typeof u.distribution
                                ];
                              const displayVal =
                                key === 'average' && val != null
                                  ? Number(val).toFixed(1)
                                  : val;
                              return (
                                <Table.Td key={u.user.id}>
                                  {displayVal != null ? displayVal : '-'}
                                </Table.Td>
                              );
                            })}
                          </Table.Tr>
                        ))}
                      </Table.Tbody>
                    </Table>
                  </Card>
                </Grid.Col>

                <Grid.Col span={12}>
                  <Card withBorder>
                    <Group justify='space-between' mb='sm'>
                      <Title order={4}>Weak Areas Comparison</Title>
                      <ActionIcon
                        size='sm'
                        variant='subtle'
                        onClick={() => showInfoModal('comparison-weak-areas')}
                      >
                        <IconInfoCircle size={16} />
                      </ActionIcon>
                    </Group>
                    {(() => {
                      // Collect all unique weak area topics
                      const allTopics = new Set<string>();
                      comparisonData.forEach(u => {
                        (u.weakAreas || []).forEach(a =>
                          allTopics.add(a.topic || 'Unknown')
                        );
                      });
                      const topicsArr = Array.from(allTopics);

                      if (topicsArr.length === 0) {
                        return (
                          <Text c='dimmed' fs='italic'>
                            No weak areas data available
                          </Text>
                        );
                      }

                      return (
                        <Table>
                          <Table.Thead>
                            <Table.Tr>
                              <Table.Th>Area</Table.Th>
                              {comparisonData.map(u => (
                                <Table.Th key={u.user.id}>
                                  {u.user.username}
                                </Table.Th>
                              ))}
                            </Table.Tr>
                          </Table.Thead>
                          <Table.Tbody>
                            {topicsArr.map(topic => (
                              <Table.Tr key={topic}>
                                <Table.Td>{topic.replace(/_/g, ' ')}</Table.Td>
                                {comparisonData.map(u => {
                                  const area = (u.weakAreas || []).find(
                                    a => (a.topic || 'Unknown') === topic
                                  );
                                  const val =
                                    area && area.total_attempts > 0
                                      ? (
                                          (area.correct_attempts /
                                            area.total_attempts) *
                                          100
                                        ).toFixed(1) + '%'
                                      : '-';
                                  return (
                                    <Table.Td key={u.user.id}>{val}</Table.Td>
                                  );
                                })}
                              </Table.Tr>
                            ))}
                          </Table.Tbody>
                        </Table>
                      );
                    })()}
                  </Card>
                </Grid.Col>
              </Grid>
            )}
          </Paper>
        </Tabs.Panel>
      </Tabs>

      {/* Info Modal */}
      <Modal
        opened={infoModal.open}
        onClose={() => setInfoModal({ open: false, content: '' })}
        title='Information'
        size='lg'
      >
        <div dangerouslySetInnerHTML={{ __html: infoModal.content }} />
      </Modal>
    </Container>
  );
};

export default AnalyticsPage;
