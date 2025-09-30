import React, { useState, useEffect, useCallback } from 'react';
import {
  Container,
  Title,
  Text,
  Table,
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
} from '@mantine/core';
import {
  IconPlayerPlay,
  IconPlayerPause,
  IconRefresh,
  IconTrash,
} from '@tabler/icons-react';
import { useAuth } from '../../hooks/useAuth';
import { Navigate } from 'react-router-dom';
import { notifications } from '@mantine/notifications';
import logger from '../../utils/logger';

interface WorkerStatus {
  id: number;
  worker_instance: string;
  is_running: boolean;
  is_paused: boolean;
  current_activity: string | null;
  last_heartbeat: string;
  last_run_start: string;
  last_run_end: string | null;
  last_run_finish: string;
  last_run_error: string | null;
  total_questions_processed: number;
  total_questions_generated: number;
  total_runs: number;
  created_at: string;
  updated_at: string;
}

interface RunHistory {
  start_time: string;
  duration: number;
  status: string;
  details: string;
  count?: number;
}

interface AIConcurrency {
  active_requests: number;
  max_concurrent: number;
  queued_requests: number;
  total_requests: number;
  user_active_count: Record<string, number>;
}

interface PriorityAnalytics {
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
}

interface UserPerformanceAnalytics {
  weakAreas: Array<{
    topic: string;
    correct_attempts: number;
    total_attempts: number;
  }>;
  learningPreferences: {
    focusOnWeakAreas: boolean;
    freshQuestionRatio: number;
    weakAreaBoost: number;
    knownQuestionPenalty: number;
  };
}

interface GenerationIntelligence {
  gapAnalysis: Array<{
    question_type: string;
    level: string;
    available: number;
    demand: number;
  }>;
  generationSuggestions: Array<{
    priority: string;
    count: number;
    question_type: string;
    level: string;
  }>;
}

interface SystemHealthAnalytics {
  performance: {
    calculationsPerSecond: number;
    avgCalculationTime: number;
    avgQueryTime: number;
    memoryUsage: number;
  };
  backgroundJobs: {
    priorityUpdates: number;
    lastUpdate: string;
    queueSize: number;
    status: string;
  };
}

interface ActivityLog {
  timestamp: string;
  level: string;
  message: string;
  username?: string;
}

const WorkerAdminPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const [workerStatus, setWorkerStatus] = useState<WorkerStatus | null>(null);
  const [aiConcurrency, setAiConcurrency] = useState<AIConcurrency | null>(
    null
  );
  const [runHistory, setRunHistory] = useState<RunHistory[]>([]);
  const [activityLogs, setActivityLogs] = useState<ActivityLog[]>([]);
  const [priorityAnalytics, setPriorityAnalytics] =
    useState<PriorityAnalytics | null>(null);
  const [userPerformanceAnalytics, setUserPerformanceAnalytics] =
    useState<UserPerformanceAnalytics | null>(null);
  const [generationIntelligence, setGenerationIntelligence] =
    useState<GenerationIntelligence | null>(null);
  const [systemHealthAnalytics, setSystemHealthAnalytics] =
    useState<SystemHealthAnalytics | null>(null);
  const [globalPaused, setGlobalPaused] = useState(false);
  const [logPaused, setLogPaused] = useState(false);
  const [loading, setLoading] = useState(false);
  const [errorModal, setErrorModal] = useState<{
    open: boolean;
    title: string;
    content: string;
  }>({
    open: false,
    title: '',
    content: '',
  });
  const [confirmModal, setConfirmModal] = useState<{
    open: boolean;
    title: string;
    message: string;
    onConfirm: (() => void) | null;
  }>({
    open: false,
    title: '',
    message: '',
    onConfirm: null,
  });

  // Utility functions
  const formatDateTime = (dateString: string) => {
    if (!dateString || dateString.startsWith('0001-01-01')) {
      return 'N/A';
    }
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const formatDuration = (durationNs: number) => {
    if (!durationNs) return 'N/A';
    const ms = durationNs / 1000000;
    if (ms < 1000) return ms.toFixed(0) + ' ms';
    return (ms / 1000).toFixed(2) + ' s';
  };

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'ERROR':
        return '#dc3545';
      case 'WARN':
        return '#fd7e14';
      case 'INFO':
        return '#28a745';
      default:
        return '#d4d4d4';
    }
  };

  const formatActiveUsers = (userActiveCount: Record<string, number>) => {
    if (!userActiveCount || typeof userActiveCount !== 'object') {
      return '0';
    }

    const entries = Object.entries(userActiveCount);
    if (entries.length === 0) {
      return '0';
    }

    return entries.map(([user, count]) => `${user}: ${count}`).join(', ');
  };

  const consolidateRepeatedLogs = (history: RunHistory[]) => {
    const consolidated: (RunHistory & { count: number })[] = [];
    const messageMap = new Map<string, RunHistory & { count: number }>();

    // Process history in reverse order to keep the most recent
    for (let i = history.length - 1; i >= 0; i--) {
      const run = history[i];
      const key = `${run.status}:${run.details}`;

      if (messageMap.has(key)) {
        messageMap.get(key)!.count++;
      } else {
        const consolidatedRun = { ...run, count: 1 };
        messageMap.set(key, consolidatedRun);
        consolidated.unshift(consolidatedRun);
      }
    }

    return consolidated;
  };

  // API functions
  const updateDashboard = useCallback(async () => {
    try {
      const [detailsResponse, statusResponse] = await Promise.all([
        fetch('/v1/admin/worker/details'),
        fetch('/v1/admin/worker/status'),
      ]);

      if (!statusResponse.ok) throw new Error('Network response was not ok');

      const detailsData = await detailsResponse.json();
      const statusData = await statusResponse.json();
      const status = statusData; // Use the actual status response
      const history = detailsData.history || [];

      // Get global pause status from details
      let globalPaused = false;
      if (detailsResponse.ok) {
        globalPaused = detailsData.global_paused || false;
      }

      setWorkerStatus(status);
      setGlobalPaused(globalPaused);
      setRunHistory(history);

      // Update AI Concurrency
      try {
        const aiResponse = await fetch('/v1/admin/worker/ai-concurrency');
        if (aiResponse.ok) {
          const aiData = await aiResponse.json();
          setAiConcurrency(aiData.ai_concurrency || {});
        }
      } catch (error) {
        logger.error('Error fetching AI concurrency stats:', error);
      }
    } catch (error) {
      logger.error('Error updating dashboard:', error);
      notifications.show({
        title: 'Error',
        message: 'Failed to load dashboard data',
        color: 'red',
      });
    }
  }, []);

  const updateActivityLog = useCallback(async () => {
    if (logPaused) return;

    try {
      const response = await fetch('/v1/admin/worker/logs');
      if (!response.ok) throw new Error('Network response was not ok');
      const data = await response.json();
      setActivityLogs(data.logs || []);
    } catch (error) {
      logger.error('Error fetching activity logs:', error);
    }
  }, [logPaused]);

  const updatePriorityAnalytics = useCallback(async () => {
    try {
      const response = await fetch(
        '/v1/admin/worker/analytics/priority-scores'
      );
      if (!response.ok) throw new Error('Failed to fetch priority analytics');
      const data = await response.json();
      setPriorityAnalytics(data);
    } catch (error) {
      logger.error('Error updating priority analytics:', error);
    }
  }, []);

  const updateUserPerformanceAnalytics = useCallback(async () => {
    try {
      const response = await fetch(
        '/v1/admin/worker/analytics/user-performance'
      );
      if (!response.ok)
        throw new Error('Failed to fetch user performance analytics');
      const data = await response.json();
      setUserPerformanceAnalytics(data);
    } catch (error) {
      logger.error('Error updating user performance analytics:', error);
    }
  }, []);

  const updateGenerationIntelligence = useCallback(async () => {
    try {
      const response = await fetch(
        '/v1/admin/worker/analytics/generation-intelligence'
      );
      if (!response.ok)
        throw new Error('Failed to fetch generation intelligence');
      const data = await response.json();
      setGenerationIntelligence(data);
    } catch (error) {
      logger.error('Error updating generation intelligence:', error);
    }
  }, []);

  const updateSystemHealthAnalytics = useCallback(async () => {
    try {
      const response = await fetch('/v1/admin/worker/analytics/system-health');
      if (!response.ok)
        throw new Error('Failed to fetch system health analytics');
      const data = await response.json();
      setSystemHealthAnalytics(data);
    } catch (error) {
      logger.error('Error updating system health analytics:', error);
    }
  }, []);

  const postAction = useCallback(
    async (url: string, actionName: string) => {
      setConfirmModal({
        open: true,
        title: `Confirm ${actionName}`,
        message: `Are you sure you want to ${actionName.toLowerCase()}?`,
        onConfirm: async () => {
          setLoading(true);
          try {
            const response = await fetch(url, { method: 'POST' });
            const data = await response.json();

            // Force immediate dashboard update
            await updateDashboard();

            notifications.show({
              title: 'Action Result',
              message: data.message,
              color: 'green',
            });
          } catch (error) {
            logger.error(`Error during ${actionName}:`, error);
            notifications.show({
              title: 'Error',
              message: `Failed to ${actionName.toLowerCase()}.`,
              color: 'red',
            });
          } finally {
            setLoading(false);
            setConfirmModal({
              open: false,
              title: '',
              message: '',
              onConfirm: null,
            });
          }
        },
      });
    },
    [updateDashboard]
  );

  const showErrorDetails = (details: string) => {
    setErrorModal({
      open: true,
      title:
        details.includes('error') || details.includes('failed')
          ? 'Error Details'
          : 'Run Details',
      content: details,
    });
  };

  // Effects for periodic updates
  useEffect(() => {
    updateDashboard();
    updateActivityLog();
    updatePriorityAnalytics();
    updateUserPerformanceAnalytics();
    updateGenerationIntelligence();
    updateSystemHealthAnalytics();

    const dashboardInterval = setInterval(updateDashboard, 5000);
    const logInterval = setInterval(updateActivityLog, 2000);
    const analyticsInterval = setInterval(() => {
      updatePriorityAnalytics();
      updateUserPerformanceAnalytics();
      updateGenerationIntelligence();
      updateSystemHealthAnalytics();
    }, 30000);

    return () => {
      clearInterval(dashboardInterval);
      clearInterval(logInterval);
      clearInterval(analyticsInterval);
    };
  }, [
    updateDashboard,
    updateActivityLog,
    updatePriorityAnalytics,
    updateUserPerformanceAnalytics,
    updateGenerationIntelligence,
    updateSystemHealthAnalytics,
  ]);

  const handleTriggerRun = () => {
    postAction('/v1/admin/worker/trigger', 'Trigger Manual Run');
  };

  const handleInstancePause = async () => {
    const action = workerStatus?.is_paused ? 'resume' : 'pause';
    const actionName = workerStatus?.is_paused
      ? 'Resume Worker'
      : 'Pause Worker';
    await postAction(`/v1/admin/worker/${action}`, actionName);
  };

  const handleGlobalPause = async () => {
    const action = globalPaused ? 'resume' : 'pause';
    const actionName = globalPaused ? 'Resume Global' : 'Pause Global';

    // Update the state immediately for better UX
    setGlobalPaused(!globalPaused);

    await postAction(`/v1/admin/worker/${action}`, actionName);
  };

  const clearLogs = () => {
    setActivityLogs([]);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'success':
        return 'green';
      case 'error':
        return 'red';
      case 'running':
        return 'blue';
      default:
        return 'gray';
    }
  };

  const getOverallStatus = () => {
    if (!workerStatus) return { status: 'Unknown', color: 'gray' };
    if (!workerStatus.is_running) return { status: 'Stopped', color: 'red' };
    if (workerStatus.is_paused) return { status: 'Paused', color: 'yellow' };
    return { status: 'Running', color: 'green' };
  };

  // Check if user is admin
  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }

  const isAdmin = user.roles?.some(role => role.name === 'admin') || false;
  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  const overallStatus = getOverallStatus();

  return (
    <Container size='xl' py='md'>
      <LoadingOverlay visible={loading} />

      <Title order={1} mb='lg'>
        Worker Administration
      </Title>

      <Stack gap='lg'>
        {/* Worker Status */}
        <Paper p='md' withBorder>
          <Title order={2} mb='md'>
            Worker Status
          </Title>
          <Table>
            <Table.Tbody>
              <Table.Tr>
                <Table.Td>
                  <strong>Status</strong>
                </Table.Td>
                <Table.Td>
                  <Badge color={overallStatus.color}>
                    {overallStatus.status}
                  </Badge>
                </Table.Td>
              </Table.Tr>
              <Table.Tr>
                <Table.Td>
                  <strong>Last Heartbeat</strong>
                </Table.Td>
                <Table.Td>
                  {workerStatus
                    ? formatDateTime(workerStatus.last_heartbeat)
                    : 'N/A'}
                </Table.Td>
              </Table.Tr>
              <Table.Tr>
                <Table.Td>
                  <strong>Total Questions Generated</strong>
                </Table.Td>
                <Table.Td>
                  {workerStatus?.total_questions_generated || 0}
                </Table.Td>
              </Table.Tr>
              <Table.Tr>
                <Table.Td>
                  <strong>Total Runs</strong>
                </Table.Td>
                <Table.Td>{workerStatus?.total_runs || 0}</Table.Td>
              </Table.Tr>
              <Table.Tr>
                <Table.Td>
                  <strong>Current Activity</strong>
                </Table.Td>
                <Table.Td>
                  <Text>{workerStatus?.current_activity || 'Idle'}</Text>
                </Table.Td>
              </Table.Tr>
            </Table.Tbody>
          </Table>
        </Paper>

        {/* AI Concurrency Monitor */}
        <Paper p='md' withBorder>
          <Title order={2} mb='md'>
            AI Concurrency Monitor
          </Title>
          <Table>
            <Table.Tbody>
              <Table.Tr>
                <Table.Td>
                  <strong>Active Requests</strong>
                </Table.Td>
                <Table.Td>{aiConcurrency?.active_requests || 0}</Table.Td>
              </Table.Tr>
              <Table.Tr>
                <Table.Td>
                  <strong>Max Concurrent</strong>
                </Table.Td>
                <Table.Td>{aiConcurrency?.max_concurrent || 0}</Table.Td>
              </Table.Tr>
              <Table.Tr>
                <Table.Td>
                  <strong>Queued Requests</strong>
                </Table.Td>
                <Table.Td>{aiConcurrency?.queued_requests || 0}</Table.Td>
              </Table.Tr>
              <Table.Tr>
                <Table.Td>
                  <strong>Total Requests</strong>
                </Table.Td>
                <Table.Td>{aiConcurrency?.total_requests || 0}</Table.Td>
              </Table.Tr>
              <Table.Tr>
                <Table.Td>
                  <strong>Active Users</strong>
                </Table.Td>
                <Table.Td>
                  {aiConcurrency
                    ? formatActiveUsers(aiConcurrency.user_active_count)
                    : '0'}
                </Table.Td>
              </Table.Tr>
            </Table.Tbody>
          </Table>
        </Paper>

        {/* Actions */}
        <Paper p='md' withBorder>
          <Title order={2} mb='md'>
            Actions
          </Title>
          <Group>
            <Button
              onClick={handleTriggerRun}
              leftSection={<IconPlayerPlay size={16} />}
            >
              Trigger Manual Run
            </Button>
            <Button
              onClick={handleInstancePause}
              leftSection={
                workerStatus?.is_paused ? (
                  <IconPlayerPlay size={16} />
                ) : (
                  <IconPlayerPause size={16} />
                )
              }
              disabled={globalPaused}
            >
              {workerStatus?.is_paused ? 'Resume Worker' : 'Pause Worker'}
            </Button>
            <Button
              onClick={handleGlobalPause}
              leftSection={
                globalPaused ? (
                  <IconPlayerPlay size={16} />
                ) : (
                  <IconPlayerPause size={16} />
                )
              }
            >
              {globalPaused ? 'Resume Global' : 'Pause Global'}
            </Button>
          </Group>
        </Paper>

        {/* Live Activity Log */}
        <Paper p='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Title order={2}>Live Activity Log</Title>
            <Group>
              <Button
                size='sm'
                variant='outline'
                onClick={clearLogs}
                leftSection={<IconTrash size={14} />}
              >
                Clear
              </Button>
              <Button
                size='sm'
                variant='outline'
                onClick={() => setLogPaused(!logPaused)}
                leftSection={
                  logPaused ? (
                    <IconPlayerPlay size={14} />
                  ) : (
                    <IconPlayerPause size={14} />
                  )
                }
              >
                {logPaused ? 'Resume' : 'Pause'}
              </Button>
            </Group>
          </Group>
          <Box
            style={{
              background: '#1e1e1e',
              color: '#d4d4d4',
              fontFamily:
                'SFMono-Regular, Consolas, Liberation Mono, Menlo, Courier, monospace',
              fontSize: '12px',
              padding: '10px',
              height: '300px',
              overflowY: 'auto',
              borderRadius: '4px',
              border: '1px solid #333',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}
          >
            {activityLogs.length === 0 ? (
              <Text c='dimmed' fs='italic'>
                No activity logs yet...
              </Text>
            ) : (
              activityLogs
                .slice()
                .reverse()
                .map((log, index) => (
                  <Box key={index} mb={2}>
                    <Text span c='dimmed' size='xs'>
                      {new Date(log.timestamp).toLocaleTimeString()}
                    </Text>{' '}
                    <Text span c={getLevelColor(log.level)} fw='bold' size='xs'>
                      [{log.level}]
                    </Text>
                    {log.username && (
                      <Text span c='blue' size='xs'>
                        [{log.username}]
                      </Text>
                    )}{' '}
                    <Text span c='gray.3' size='xs'>
                      {log.message}
                    </Text>
                  </Box>
                ))
            )}
          </Box>
        </Paper>

        {/* Run History */}
        <Paper p='md' withBorder>
          <Title order={2} mb='md'>
            Run History (Current Session)
          </Title>
          <Box style={{ overflowX: 'auto' }}>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th style={{ width: '150px' }}>Start Time</Table.Th>
                  <Table.Th style={{ width: '100px' }}>Duration</Table.Th>
                  <Table.Th style={{ width: '120px' }}>Status</Table.Th>
                  <Table.Th style={{ minWidth: '300px' }}>Details</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {runHistory.length === 0 ? (
                  <Table.Tr>
                    <Table.Td colSpan={4} ta='center' c='dimmed'>
                      No run history available
                    </Table.Td>
                  </Table.Tr>
                ) : (
                  consolidateRepeatedLogs(runHistory).map((run, index) => (
                    <Table.Tr key={index}>
                      <Table.Td>{formatDateTime(run.start_time)}</Table.Td>
                      <Table.Td>{formatDuration(run.duration)}</Table.Td>
                      <Table.Td>
                        <Badge color={getStatusColor(run.status)}>
                          {run.status}
                          {run.count && run.count > 1 ? ` (${run.count}x)` : ''}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Text
                          style={{
                            cursor: 'pointer',
                            textDecoration: 'underline',
                            wordBreak: 'break-word',
                            whiteSpace: 'pre-wrap',
                            maxWidth: '400px',
                          }}
                          onClick={() => showErrorDetails(run.details)}
                        >
                          {run.details}
                        </Text>
                      </Table.Td>
                    </Table.Tr>
                  ))
                )}
              </Table.Tbody>
            </Table>
          </Box>
        </Paper>

        {/* Priority System Analytics */}
        <Paper p='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Title order={2}>Priority System Analytics</Title>
            <Button
              size='sm'
              variant='outline'
              onClick={updatePriorityAnalytics}
              leftSection={<IconRefresh size={14} />}
            >
              Refresh
            </Button>
          </Group>
          <Grid>
            <Grid.Col span={6}>
              <Card withBorder>
                <Title order={4} mb='sm'>
                  Priority Score Distribution
                </Title>
                {priorityAnalytics ? (
                  <Stack gap='xs'>
                    <Text>
                      <strong>High Priority (&gt;200):</strong>{' '}
                      {priorityAnalytics.distribution.high || 0}
                    </Text>
                    <Text>
                      <strong>Medium Priority (100-200):</strong>{' '}
                      {priorityAnalytics.distribution.medium || 0}
                    </Text>
                    <Text>
                      <strong>Low Priority (&lt;100):</strong>{' '}
                      {priorityAnalytics.distribution.low || 0}
                    </Text>
                    <Text>
                      <strong>Average Score:</strong>{' '}
                      {priorityAnalytics.distribution.average?.toFixed(2) ||
                        'N/A'}
                    </Text>
                  </Stack>
                ) : (
                  <Text c='dimmed' fs='italic'>
                    No priority data available
                  </Text>
                )}
              </Card>
            </Grid.Col>
            <Grid.Col span={6}>
              <Card withBorder>
                <Title order={4} mb='sm'>
                  Top High Priority Questions
                </Title>
                {priorityAnalytics?.highPriorityQuestions &&
                priorityAnalytics.highPriorityQuestions.length > 0 ? (
                  <Stack gap='xs'>
                    {priorityAnalytics.highPriorityQuestions
                      .slice(0, 5)
                      .map((q, index) => (
                        <Text key={index} size='sm'>
                          <strong>Score {q.priority_score.toFixed(1)}:</strong>{' '}
                          {q.question_type}/{q.level} - {q.topic || 'No topic'}
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
          </Grid>
        </Paper>

        {/* User Performance Analytics */}
        <Paper p='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Title order={2}>User Performance Analytics</Title>
            <Button
              size='sm'
              variant='outline'
              onClick={updateUserPerformanceAnalytics}
              leftSection={<IconRefresh size={14} />}
            >
              Refresh
            </Button>
          </Group>
          <Grid>
            <Grid.Col span={6}>
              <Card withBorder>
                <Title order={4} mb='sm'>
                  Weak Areas by Topic
                </Title>
                {userPerformanceAnalytics?.weakAreas &&
                userPerformanceAnalytics.weakAreas.length > 0 ? (
                  <Stack gap='xs'>
                    {userPerformanceAnalytics.weakAreas
                      .slice(0, 5)
                      .map((area, index) => {
                        const accuracy = (
                          (area.correct_attempts / area.total_attempts) *
                          100
                        ).toFixed(1);
                        return (
                          <Text key={index} size='sm'>
                            <strong>{area.topic}:</strong> {accuracy}% (
                            {area.correct_attempts}/{area.total_attempts})
                          </Text>
                        );
                      })}
                  </Stack>
                ) : (
                  <Text c='dimmed' fs='italic'>
                    No weak areas identified
                  </Text>
                )}
              </Card>
            </Grid.Col>
            <Grid.Col span={6}>
              <Card withBorder>
                <Title order={4} mb='sm'>
                  Learning Preference Usage
                </Title>
                {userPerformanceAnalytics?.learningPreferences ? (
                  <Stack gap='xs'>
                    <Text>
                      <strong>Focus on Weak Areas:</strong>{' '}
                      {userPerformanceAnalytics.learningPreferences
                        .focusOnWeakAreas
                        ? 'Enabled'
                        : 'Disabled'}
                    </Text>
                    <Text>
                      <strong>Fresh Question Ratio:</strong>{' '}
                      {(
                        userPerformanceAnalytics.learningPreferences
                          .freshQuestionRatio * 100
                      ).toFixed(0)}
                      %
                    </Text>
                    <Text>
                      <strong>Weak Area Boost:</strong>{' '}
                      {
                        userPerformanceAnalytics.learningPreferences
                          .weakAreaBoost
                      }
                      x
                    </Text>
                    <Text>
                      <strong>Known Question Penalty:</strong>{' '}
                      {(
                        userPerformanceAnalytics.learningPreferences
                          .knownQuestionPenalty * 100
                      ).toFixed(0)}
                      %
                    </Text>
                  </Stack>
                ) : (
                  <Text c='dimmed' fs='italic'>
                    No preference data available
                  </Text>
                )}
              </Card>
            </Grid.Col>
          </Grid>
        </Paper>

        {/* Question Generation Intelligence */}
        <Paper p='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Title order={2}>Question Generation Intelligence</Title>
            <Button
              size='sm'
              variant='outline'
              onClick={updateGenerationIntelligence}
              leftSection={<IconRefresh size={14} />}
            >
              Refresh
            </Button>
          </Group>
          <Grid>
            <Grid.Col span={6}>
              <Card withBorder>
                <Title order={4} mb='sm'>
                  Question Type Gaps
                </Title>
                {generationIntelligence?.gapAnalysis &&
                generationIntelligence.gapAnalysis.length > 0 ? (
                  <Stack gap='xs'>
                    {generationIntelligence.gapAnalysis
                      .slice(0, 5)
                      .map((gap, index) => (
                        <Text key={index} size='sm'>
                          <strong>
                            {gap.question_type}/{gap.level}:
                          </strong>{' '}
                          {gap.available} available, {gap.demand} needed
                        </Text>
                      ))}
                  </Stack>
                ) : (
                  <Text c='dimmed' fs='italic'>
                    No significant gaps identified
                  </Text>
                )}
              </Card>
            </Grid.Col>
            <Grid.Col span={6}>
              <Card withBorder>
                <Title order={4} mb='sm'>
                  Generation Suggestions
                </Title>
                {generationIntelligence?.generationSuggestions &&
                generationIntelligence.generationSuggestions.length > 0 ? (
                  <Stack gap='xs'>
                    {generationIntelligence.generationSuggestions
                      .slice(0, 5)
                      .map((suggestion, index) => (
                        <Text key={index} size='sm'>
                          <strong>Priority {suggestion.priority}:</strong>{' '}
                          Generate {suggestion.count} {suggestion.question_type}
                          /{suggestion.level}
                        </Text>
                      ))}
                  </Stack>
                ) : (
                  <Text c='dimmed' fs='italic'>
                    No generation suggestions
                  </Text>
                )}
              </Card>
            </Grid.Col>
          </Grid>
        </Paper>

        {/* System Health & Performance */}
        <Paper p='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Title order={2}>System Health & Performance</Title>
            <Button
              size='sm'
              variant='outline'
              onClick={updateSystemHealthAnalytics}
              leftSection={<IconRefresh size={14} />}
            >
              Refresh
            </Button>
          </Group>
          <Grid>
            <Grid.Col span={6}>
              <Card withBorder>
                <Title order={4} mb='sm'>
                  Performance Metrics
                </Title>
                {systemHealthAnalytics?.performance ? (
                  <Stack gap='xs'>
                    <Text>
                      <strong>Priority Calculations/sec:</strong>{' '}
                      {systemHealthAnalytics.performance.calculationsPerSecond?.toFixed(
                        2
                      ) || 'N/A'}
                    </Text>
                    <Text>
                      <strong>Average Calculation Time:</strong>{' '}
                      {systemHealthAnalytics.performance.avgCalculationTime?.toFixed(
                        2
                      ) || 'N/A'}
                      ms
                    </Text>
                    <Text>
                      <strong>Database Query Time:</strong>{' '}
                      {systemHealthAnalytics.performance.avgQueryTime?.toFixed(
                        2
                      ) || 'N/A'}
                      ms
                    </Text>
                    <Text>
                      <strong>Memory Usage:</strong>{' '}
                      {systemHealthAnalytics.performance.memoryUsage?.toFixed(
                        1
                      ) || 'N/A'}
                      MB
                    </Text>
                  </Stack>
                ) : (
                  <Text c='dimmed' fs='italic'>
                    No performance data available
                  </Text>
                )}
              </Card>
            </Grid.Col>
            <Grid.Col span={6}>
              <Card withBorder>
                <Title order={4} mb='sm'>
                  Background Jobs
                </Title>
                {systemHealthAnalytics?.backgroundJobs ? (
                  <Stack gap='xs'>
                    <Text>
                      <strong>Priority Score Updates:</strong>{' '}
                      {systemHealthAnalytics.backgroundJobs.priorityUpdates ||
                        0}
                      /min
                    </Text>
                    <Text>
                      <strong>Last Update:</strong>{' '}
                      {systemHealthAnalytics.backgroundJobs.lastUpdate || 'N/A'}
                    </Text>
                    <Text>
                      <strong>Queue Size:</strong>{' '}
                      {systemHealthAnalytics.backgroundJobs.queueSize || 0}
                    </Text>
                    <Text>
                      <strong>Status:</strong>{' '}
                      <Badge
                        color={
                          systemHealthAnalytics.backgroundJobs.status ===
                          'healthy'
                            ? 'green'
                            : 'red'
                        }
                      >
                        {systemHealthAnalytics.backgroundJobs.status ||
                          'Unknown'}
                      </Badge>
                    </Text>
                  </Stack>
                ) : (
                  <Text c='dimmed' fs='italic'>
                    No background job data available
                  </Text>
                )}
              </Card>
            </Grid.Col>
          </Grid>
        </Paper>
      </Stack>

      {/* Error Modal */}
      <Modal
        opened={errorModal.open}
        onClose={() => setErrorModal({ open: false, title: '', content: '' })}
        title={errorModal.title}
        size='lg'
      >
        <Text
          style={{
            whiteSpace: 'pre-wrap',
            fontFamily:
              'SFMono-Regular, Consolas, Liberation Mono, Menlo, Courier, monospace',
            fontSize: '14px',
            lineHeight: '1.6',
            backgroundColor: '#f8f9fa',
            padding: '16px',
            borderRadius: '8px',
            border: '1px solid #e9ecef',
          }}
        >
          {errorModal.content}
        </Text>
      </Modal>

      {/* Confirmation Modal */}
      <Modal
        opened={confirmModal.open}
        onClose={() =>
          setConfirmModal({
            open: false,
            title: '',
            message: '',
            onConfirm: null,
          })
        }
        title={confirmModal.title}
      >
        <Stack>
          <Text>{confirmModal.message}</Text>
          <Group justify='flex-end'>
            <Button
              variant='outline'
              onClick={() =>
                setConfirmModal({
                  open: false,
                  title: '',
                  message: '',
                  onConfirm: null,
                })
              }
            >
              Cancel
            </Button>
            <Button
              color='red'
              onClick={() => {
                if (confirmModal.onConfirm) {
                  confirmModal.onConfirm();
                }
              }}
            >
              Confirm
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
};

export default WorkerAdminPage;
