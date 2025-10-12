import React, { useState, useEffect, useCallback } from 'react';
import {
  Container,
  Title,
  Text,
  Card,
  Group,
  Button,
  Stack,
  Badge,
  Grid,
  LoadingOverlay,
  Tabs,
  Table,
  Pagination,
  Select,
  SimpleGrid,
  ThemeIcon,
  Paper,
  Alert,
} from '@mantine/core';
import {
  IconRefresh,
  IconAlertTriangle,
  IconCheck,
  IconX,
  IconMail,
  IconClock,
  IconUsers,
  IconChartBar,
  IconSend,
  IconSearch,
} from '@tabler/icons-react';
import { useAuth } from '../../hooks/useAuth';
import { Navigate } from 'react-router-dom';
import { notifications } from '@mantine/notifications';
import { useUsersPaginated } from '../../api/admin';

interface NotificationStats {
  total_notifications_sent: number;
  total_notifications_failed: number;
  success_rate: number;
  users_with_notifications_enabled: number;
  total_users: number;
  notifications_sent_today: number;
  notifications_sent_this_week: number;
  notifications_by_type: Record<string, number>;
  errors_by_type: Record<string, number>;
  upcoming_notifications: number;
  unresolved_errors: number;
}

interface NotificationError {
  id: number;
  user_id?: number;
  username?: string;
  notification_type: string;
  error_type: string;
  error_message: string;
  email_address?: string;
  occurred_at: string;
  resolved_at?: string;
  resolution_notes?: string;
}

interface SentNotification {
  id: number;
  user_id: number;
  username: string;
  email_address: string;
  notification_type: string;
  subject: string;
  template_name: string;
  sent_at: string;
  status: string;
  error_message?: string;
  retry_count: number;
}

interface PaginationData {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
}

const NotificationsPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const [stats, setStats] = useState<NotificationStats | null>(null);
  const [errors, setErrors] = useState<NotificationError[]>([]);
  const [sent, setSent] = useState<SentNotification[]>([]);
  const [loading] = useState(false);
  const [activeTab, setActiveTab] = useState<string>('stats');

  // Pagination state
  const [errorsPagination, setErrorsPagination] = useState<PaginationData>({
    page: 1,
    page_size: 20,
    total: 0,
    total_pages: 0,
  });
  const [sentPagination, setSentPagination] = useState<PaginationData>({
    page: 1,
    page_size: 20,
    total: 0,
    total_pages: 0,
  });

  // Filter state
  const [errorsFilters, setErrorsFilters] = useState({
    error_type: '',
    notification_type: '',
    resolved: '',
  });
  const [sentFilters, setSentFilters] = useState({
    notification_type: '',
    status: '',
  });

  // Force send state
  const [selectedUsername, setSelectedUsername] = useState<string>('');
  const [isSending, setIsSending] = useState(false);
  const [searchQuery, setSearchQuery] = useState<string>('');

  // API hooks
  const { data: usersData } = useUsersPaginated({
    page: 1,
    pageSize: 50,
    search: searchQuery || undefined,
  });

  // API functions
  const fetchStats = useCallback(async () => {
    try {
      const response = await fetch('/v1/admin/worker/notifications/stats');
      if (!response.ok) throw new Error('Failed to fetch notification stats');
      const data = await response.json();
      setStats(data);
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: 'Failed to load notification statistics: ' + error,
        color: 'red',
      });
    }
  }, []);

  // Force send notification function
  const forceSendNotification = useCallback(async (username: string) => {
    setIsSending(true);
    try {
      const response = await fetch(
        '/v1/admin/worker/notifications/force-send',
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ username }),
        }
      );

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error || 'Failed to send notification');
      }

      const data = await response.json();
      notifications.show({
        title: 'Success',
        message: `Notification sent to ${data.user.username}`,
        color: 'green',
      });

      // Clear the selected username
      setSelectedUsername('');
    } catch (error) {
      notifications.show({
        title: 'Error',
        message:
          error instanceof Error
            ? error.message
            : 'Failed to send notification: ' + error,
        color: 'red',
      });
    } finally {
      setIsSending(false);
    }
  }, []);

  // Effects for data loading
  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  // Handle errors tab data loading
  useEffect(() => {
    const isErrorsTab = activeTab === 'errors';
    if (isErrorsTab) {
      const fetchErrorsData = async () => {
        try {
          const params = new URLSearchParams({
            page: errorsPagination.page.toString(),
            page_size: errorsPagination.page_size.toString(),
            ...(errorsFilters.error_type && {
              error_type: errorsFilters.error_type,
            }),
            ...(errorsFilters.notification_type && {
              notification_type: errorsFilters.notification_type,
            }),
            ...(errorsFilters.resolved && { resolved: errorsFilters.resolved }),
          });

          const response = await fetch(
            `/v1/admin/worker/notifications/errors?${params}`
          );
          if (!response.ok)
            throw new Error('Failed to fetch notification errors');
          const data = await response.json();
          setErrors(data.errors || []);
          setErrorsPagination(prev => data.pagination || prev);
        } catch (error) {
          notifications.show({
            title: 'Error',
            message: 'Failed to load notification errors: ' + error,
            color: 'red',
          });
        }
      };
      fetchErrorsData();
    }
  }, [
    activeTab,
    errorsPagination.page,
    errorsPagination.page_size,
    errorsFilters,
  ]);

  // Handle sent tab data loading
  useEffect(() => {
    const isSentTab = activeTab === 'sent';
    if (isSentTab) {
      const fetchSentData = async () => {
        try {
          const params = new URLSearchParams({
            page: sentPagination.page.toString(),
            page_size: sentPagination.page_size.toString(),
            ...(sentFilters.notification_type && {
              notification_type: sentFilters.notification_type,
            }),
            ...(sentFilters.status && { status: sentFilters.status }),
          });

          const response = await fetch(
            `/v1/admin/worker/notifications/sent?${params}`
          );
          if (!response.ok)
            throw new Error('Failed to fetch sent notifications');
          const data = await response.json();
          setSent(data.notifications || []);
          setSentPagination(prev => data.pagination || prev);
        } catch (error) {
          notifications.show({
            title: 'Error',
            message: 'Failed to load sent notifications: ' + error,
            color: 'red',
          });
        }
      };
      fetchSentData();
    }
  }, [activeTab, sentPagination.page, sentPagination.page_size, sentFilters]);

  // Check if user is admin
  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }

  const isAdmin = user.roles?.some(role => role.name === 'admin') || false;
  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  const formatDateTime = (dateString: string) => {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'sent':
      case 'success':
        return 'green';
      case 'failed':
      case 'error':
        return 'red';
      case 'pending':
        return 'yellow';
      case 'cancelled':
        return 'gray';
      default:
        return 'blue';
    }
  };

  const getErrorTypeColor = (errorType: string) => {
    switch (errorType) {
      case 'smtp_error':
        return 'red';
      case 'template_error':
        return 'orange';
      case 'user_not_found':
        return 'blue';
      case 'email_disabled':
        return 'gray';
      default:
        return 'yellow';
    }
  };

  return (
    <Container size='xl' py='md'>
      <LoadingOverlay visible={loading} />

      <Title order={1} mb='lg'>
        Notification Management
      </Title>

      <Tabs
        value={activeTab}
        onChange={value => setActiveTab(value || 'stats')}
      >
        <Tabs.List>
          <Tabs.Tab value='stats' leftSection={<IconChartBar size={16} />}>
            Statistics
          </Tabs.Tab>
          <Tabs.Tab
            value='errors'
            leftSection={<IconAlertTriangle size={16} />}
          >
            Errors
          </Tabs.Tab>
          <Tabs.Tab value='sent' leftSection={<IconMail size={16} />}>
            Sent
          </Tabs.Tab>
          <Tabs.Tab value='force-send' leftSection={<IconSend size={16} />}>
            Force Send
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value='stats' pt='md'>
          {stats && (
            <Stack gap='lg'>
              {/* Overview Cards */}
              <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }}>
                <Card withBorder>
                  <Group>
                    <ThemeIcon size='lg' color='green'>
                      <IconCheck size={20} />
                    </ThemeIcon>
                    <div>
                      <Text size='xs' c='dimmed' tt='uppercase'>
                        Total Sent
                      </Text>
                      <Text fw={700} size='xl'>
                        {stats.total_notifications_sent.toLocaleString()}
                      </Text>
                    </div>
                  </Group>
                </Card>

                <Card withBorder>
                  <Group>
                    <ThemeIcon size='lg' color='red'>
                      <IconX size={20} />
                    </ThemeIcon>
                    <div>
                      <Text size='xs' c='dimmed' tt='uppercase'>
                        Failed
                      </Text>
                      <Text fw={700} size='xl'>
                        {stats.total_notifications_failed.toLocaleString()}
                      </Text>
                    </div>
                  </Group>
                </Card>

                <Card withBorder>
                  <Group>
                    <ThemeIcon size='lg' color='blue'>
                      <IconUsers size={20} />
                    </ThemeIcon>
                    <div>
                      <Text size='xs' c='dimmed' tt='uppercase'>
                        Users Enabled
                      </Text>
                      <Text fw={700} size='xl'>
                        {stats.users_with_notifications_enabled} /{' '}
                        {stats.total_users}
                      </Text>
                    </div>
                  </Group>
                </Card>

                <Card withBorder>
                  <Group>
                    <ThemeIcon size='lg' color='yellow'>
                      <IconClock size={20} />
                    </ThemeIcon>
                    <div>
                      <Text size='xs' c='dimmed' tt='uppercase'>
                        Success Rate
                      </Text>
                      <Text fw={700} size='xl'>
                        {(stats.success_rate * 100).toFixed(1)}%
                      </Text>
                    </div>
                  </Group>
                </Card>
              </SimpleGrid>

              {/* Detailed Stats */}
              <Grid>
                <Grid.Col span={6}>
                  <Card withBorder>
                    <Title order={4} mb='md'>
                      Recent Activity
                    </Title>
                    <Stack gap='xs'>
                      <Group justify='space-between'>
                        <Text>Sent Today:</Text>
                        <Badge color='green'>
                          {stats.notifications_sent_today}
                        </Badge>
                      </Group>
                      <Group justify='space-between'>
                        <Text>Sent This Week:</Text>
                        <Badge color='blue'>
                          {stats.notifications_sent_this_week}
                        </Badge>
                      </Group>
                      <Group justify='space-between'>
                        <Text>Upcoming:</Text>
                        <Badge color='yellow'>
                          {stats.upcoming_notifications}
                        </Badge>
                      </Group>
                      <Group justify='space-between'>
                        <Text>Unresolved Errors:</Text>
                        <Badge color='red'>{stats.unresolved_errors}</Badge>
                      </Group>
                    </Stack>
                  </Card>
                </Grid.Col>

                <Grid.Col span={6}>
                  <Card withBorder>
                    <Title order={4} mb='md'>
                      Notifications by Type
                    </Title>
                    <Stack gap='xs'>
                      {Object.entries(stats.notifications_by_type).map(
                        ([type, count]) => (
                          <Group key={type} justify='space-between'>
                            <Text tt='capitalize'>
                              {type.replace('_', ' ')}:
                            </Text>
                            <Badge>{count}</Badge>
                          </Group>
                        )
                      )}
                    </Stack>
                  </Card>
                </Grid.Col>
              </Grid>

              <Group justify='flex-end'>
                <Button
                  leftSection={<IconRefresh size={16} />}
                  onClick={fetchStats}
                  variant='outline'
                >
                  Refresh Stats
                </Button>
              </Group>
            </Stack>
          )}
        </Tabs.Panel>

        <Tabs.Panel value='errors' pt='md'>
          <Stack gap='md'>
            {/* Filters */}
            <Paper p='md' withBorder>
              <Group>
                <Select
                  label='Error Type'
                  placeholder='All error types'
                  data={[
                    { value: 'smtp_error', label: 'SMTP Error' },
                    { value: 'template_error', label: 'Template Error' },
                    { value: 'user_not_found', label: 'User Not Found' },
                    { value: 'email_disabled', label: 'Email Disabled' },
                    { value: 'other', label: 'Other' },
                  ]}
                  value={errorsFilters.error_type}
                  onChange={value =>
                    setErrorsFilters({
                      ...errorsFilters,
                      error_type: value || '',
                    })
                  }
                  clearable
                />
                <Select
                  label='Notification Type'
                  placeholder='All notification types'
                  data={[
                    { value: 'daily_reminder', label: 'Daily Reminder' },
                    { value: 'test_email', label: 'Test Email' },
                  ]}
                  value={errorsFilters.notification_type}
                  onChange={value =>
                    setErrorsFilters({
                      ...errorsFilters,
                      notification_type: value || '',
                    })
                  }
                  clearable
                />
                <Select
                  label='Resolution Status'
                  placeholder='All statuses'
                  data={[
                    { value: 'false', label: 'Unresolved' },
                    { value: 'true', label: 'Resolved' },
                  ]}
                  value={errorsFilters.resolved}
                  onChange={value =>
                    setErrorsFilters({
                      ...errorsFilters,
                      resolved: value || '',
                    })
                  }
                  clearable
                />
                <Button
                  leftSection={<IconRefresh size={16} />}
                  onClick={() => {
                    // Reset to page 1 when applying filters
                    setErrorsPagination(prev => ({ ...prev, page: 1 }));
                  }}
                  variant='outline'
                  style={{ marginTop: 'auto' }}
                >
                  Apply Filters
                </Button>
              </Group>
            </Paper>

            {/* Errors Table */}
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>User</Table.Th>
                  <Table.Th>Type</Table.Th>
                  <Table.Th>Error</Table.Th>
                  <Table.Th>Email</Table.Th>
                  <Table.Th>Occurred</Table.Th>
                  <Table.Th>Status</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {errors.map(error => (
                  <Table.Tr key={error.id}>
                    <Table.Td>
                      <Text size='sm' fw={500}>
                        {error.username || `User ${error.user_id}`}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Badge color={getErrorTypeColor(error.error_type)}>
                        {error.error_type.replace('_', ' ')}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <Text size='sm' lineClamp={2}>
                        {error.error_message}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text size='sm' c='dimmed'>
                        {error.email_address || 'N/A'}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text size='sm'>{formatDateTime(error.occurred_at)}</Text>
                    </Table.Td>
                    <Table.Td>
                      <Badge color={error.resolved_at ? 'green' : 'red'}>
                        {error.resolved_at ? 'Resolved' : 'Unresolved'}
                      </Badge>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>

            {/* Pagination */}
            <Group justify='center'>
              <Pagination
                total={errorsPagination.total_pages}
                value={errorsPagination.page}
                onChange={page =>
                  setErrorsPagination({ ...errorsPagination, page })
                }
              />
            </Group>
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value='sent' pt='md'>
          <Stack gap='md'>
            {/* Filters */}
            <Paper p='md' withBorder>
              <Group>
                <Select
                  label='Notification Type'
                  placeholder='All notification types'
                  data={[
                    { value: 'daily_reminder', label: 'Daily Reminder' },
                    { value: 'test_email', label: 'Test Email' },
                  ]}
                  value={sentFilters.notification_type}
                  onChange={value =>
                    setSentFilters({
                      ...sentFilters,
                      notification_type: value || '',
                    })
                  }
                  clearable
                />
                <Select
                  label='Status'
                  placeholder='All statuses'
                  data={[
                    { value: 'sent', label: 'Sent' },
                    { value: 'failed', label: 'Failed' },
                    { value: 'bounced', label: 'Bounced' },
                  ]}
                  value={sentFilters.status}
                  onChange={value =>
                    setSentFilters({ ...sentFilters, status: value || '' })
                  }
                  clearable
                />
                <Button
                  leftSection={<IconRefresh size={16} />}
                  onClick={() => {
                    // Reset to page 1 when applying filters
                    setSentPagination(prev => ({ ...prev, page: 1 }));
                  }}
                  variant='outline'
                  style={{ marginTop: 'auto' }}
                >
                  Apply Filters
                </Button>
              </Group>
            </Paper>

            {/* Sent Table */}
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>User</Table.Th>
                  <Table.Th>Type</Table.Th>
                  <Table.Th>Subject</Table.Th>
                  <Table.Th>Email</Table.Th>
                  <Table.Th>Sent At</Table.Th>
                  <Table.Th>Status</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {sent.map(notification => (
                  <Table.Tr key={notification.id}>
                    <Table.Td>
                      <Text size='sm' fw={500}>
                        {notification.username}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Badge>
                        {notification.notification_type.replace('_', ' ')}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <Text size='sm' lineClamp={1}>
                        {notification.subject}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text size='sm' c='dimmed'>
                        {notification.email_address}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text size='sm'>
                        {formatDateTime(notification.sent_at)}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Badge color={getStatusColor(notification.status)}>
                        {notification.status}
                      </Badge>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>

            {/* Pagination */}
            <Group justify='center'>
              <Pagination
                total={sentPagination.total_pages}
                value={sentPagination.page}
                onChange={page =>
                  setSentPagination({ ...sentPagination, page })
                }
              />
            </Group>
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value='force-send' pt='md'>
          <Stack gap='lg'>
            <Alert
              title='Force Send Notifications'
              color='blue'
              icon={<IconSend size={16} />}
            >
              This feature allows you to force send a daily reminder
              notification to any user, bypassing the normal time and date
              checks. The user must have daily reminders enabled.
            </Alert>

            <Paper p='md' withBorder>
              <Stack gap='md'>
                <Text fw={500} size='lg'>
                  Select User
                </Text>

                <Select
                  label='Username'
                  placeholder='Search for a user...'
                  value={selectedUsername}
                  onChange={value => setSelectedUsername(value || '')}
                  data={
                    usersData?.users
                      ?.map((user: { username?: string }) => user?.username)
                      .filter(Boolean) || []
                  }
                  searchable
                  clearable
                  rightSection={<IconSearch size={16} />}
                />

                <Group>
                  <Button
                    leftSection={<IconSend size={16} />}
                    onClick={() => forceSendNotification(selectedUsername)}
                    disabled={!selectedUsername || isSending}
                    loading={isSending}
                    color='blue'
                  >
                    Send Notification
                  </Button>

                  <Button
                    variant='outline'
                    onClick={() => {
                      setSelectedUsername('');
                      setSearchQuery('');
                    }}
                    disabled={isSending}
                  >
                    Clear
                  </Button>
                </Group>
              </Stack>
            </Paper>
          </Stack>
        </Tabs.Panel>
      </Tabs>
    </Container>
  );
};

export default NotificationsPage;
