import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../hooks/useAuth';
import {
  Badge,
  Popover,
  Text,
  Stack,
  Group,
  ActionIcon,
  Code,
  Paper,
  ScrollArea,
} from '@mantine/core';
import {
  IconAlertTriangle,
  IconCircleCheck,
  IconPlayerPause,
  IconClock,
} from '@tabler/icons-react';

interface WorkerStatusData {
  has_errors: boolean;
  error_message: string;
  global_paused: boolean;
  user_paused: boolean;
  healthy_workers: number;
  total_workers: number;
  last_error_details: string;
  worker_running: boolean;
}

const WorkerStatus: React.FC = () => {
  const { isAuthenticated } = useAuth();
  const [status, setStatus] = useState<WorkerStatusData | null>(null);
  const [showDetails, setShowDetails] = useState(false);

  const fetchWorkerStatus = useCallback(async () => {
    if (!isAuthenticated) {
      return;
    }

    try {
      const response = await fetch('/v1/quiz/worker-status', {
        credentials: 'include',
      });

      if (response.ok) {
        const data = await response.json();
        setStatus(data);
      } else {
      }
    } catch {}
  }, [isAuthenticated]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchWorkerStatus();
      // Poll every 30 seconds
      const interval = setInterval(fetchWorkerStatus, 30000);
      return () => clearInterval(interval);
    }
  }, [isAuthenticated, fetchWorkerStatus]);

  if (!isAuthenticated || !status) {
    return null;
  }

  const shouldShowIndicator = () => {
    return (
      status.has_errors ||
      status.user_paused ||
      status.global_paused ||
      !status.worker_running
    );
  };

  if (!shouldShowIndicator()) {
    return null;
  }

  const getIndicatorIcon = () => {
    if (status.user_paused || status.global_paused) {
      return <IconPlayerPause size={14} />;
    }
    if (status.has_errors) {
      return <IconAlertTriangle size={14} />;
    }
    if (!status.worker_running) {
      return <IconClock size={14} />;
    }
    return <IconCircleCheck size={14} />;
  };

  const getIndicatorColor = () => {
    if (status.user_paused || status.global_paused) {
      return 'yellow';
    }
    if (status.has_errors) {
      return 'red';
    }
    if (!status.worker_running) {
      return 'gray';
    }
    return 'green';
  };

  const getStatusMessage = () => {
    if (status.user_paused) {
      return 'Question generation paused for your account';
    }
    if (status.global_paused) {
      return 'Question generation globally paused';
    }
    if (status.has_errors) {
      return status.error_message || 'Worker experiencing errors';
    }
    if (!status.worker_running) {
      return 'Worker not currently running';
    }
    return 'Question generation working normally';
  };

  return (
    <Popover
      width={600}
      position='bottom-end'
      withArrow
      shadow='md'
      opened={showDetails}
      onChange={setShowDetails}
    >
      <Popover.Target>
        <ActionIcon
          variant='subtle'
          color={getIndicatorColor()}
          onClick={() => setShowDetails(!showDetails)}
          title={getStatusMessage()}
        >
          {getIndicatorIcon()}
        </ActionIcon>
      </Popover.Target>

      <Popover.Dropdown>
        <Paper p='md'>
          <Stack gap='md'>
            <Group justify='space-between'>
              <Text fw={600} size='sm'>
                Worker Status
              </Text>
              <Badge color={getIndicatorColor()} variant='light' size='sm'>
                {status.has_errors
                  ? 'Error'
                  : status.user_paused || status.global_paused
                    ? 'Paused'
                    : status.worker_running
                      ? 'Running'
                      : 'Stopped'}
              </Badge>
            </Group>

            <Stack gap='xs'>
              <Group justify='space-between' align='flex-start'>
                <Text size='sm' c='dimmed' style={{ flexShrink: 0 }}>
                  Status:
                </Text>
                <Text
                  size='sm'
                  c={getIndicatorColor()}
                  style={{ flex: 1, wordBreak: 'break-word' }}
                >
                  {getStatusMessage()}
                </Text>
              </Group>

              <Group justify='space-between'>
                <Text size='sm' c='dimmed'>
                  Workers:
                </Text>
                <Text size='sm'>
                  {status.healthy_workers}/{status.total_workers} healthy
                </Text>
              </Group>

              {status.last_error_details && (
                <Stack gap='xs'>
                  <Text size='sm' c='dimmed' fw={500}>
                    Latest Error:
                  </Text>
                  <ScrollArea.Autosize mah={200}>
                    <Code
                      block
                      p='sm'
                      style={{
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontSize: '12px',
                        lineHeight: 1.5,
                        maxWidth: '100%',
                      }}
                    >
                      {status.last_error_details}
                    </Code>
                  </ScrollArea.Autosize>
                </Stack>
              )}

              <Text size='xs' c='dimmed'>
                Last updated: {new Date().toLocaleTimeString()}
              </Text>
            </Stack>
          </Stack>
        </Paper>
      </Popover.Dropdown>
    </Popover>
  );
};

export default WorkerStatus;
