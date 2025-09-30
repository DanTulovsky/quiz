import React, { useState } from 'react';
import {
  Container,
  Title,
  Text,
  Stack,
  Paper,
  Group,
  Button,
  Select,
  Card,
  Badge,
  Table,
  LoadingOverlay,
  Grid,
  Box,
  Tooltip,
  Modal,
  Divider,
} from '@mantine/core';
import {
  IconRefresh,
  IconUser,
  IconCheck,
  IconClock,
  IconInfoCircle,
} from '@tabler/icons-react';
import { useAuth } from '../../hooks/useAuth';
import { Navigate, useSearchParams } from 'react-router-dom';
import { notifications } from '@mantine/notifications';
import DailyDatePicker from '../../components/DailyDatePicker';
import { useUsersPaginated } from '../../api/admin';
import {
  useGetV1AdminWorkerDailyUsersUserIdQuestionsDate,
  usePostV1AdminWorkerDailyUsersUserIdQuestionsDateRegenerate,
  useGetV1DailyDates,
  DailyQuestionWithDetails,
} from '../../api/api';
import { formatDateForAPI } from '../../utils/time';

const DailyAdminPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const [searchParams, setSearchParams] = useSearchParams();
  const [selectedUser, setSelectedUser] = useState<number | null>(null);
  const [selectedDate, setSelectedDate] = useState<string>(
    formatDateForAPI(new Date())
  );
  const [isRegenerating, setIsRegenerating] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [selectedAssignment, setSelectedAssignment] =
    useState<DailyQuestionWithDetails | null>(null);

  // API hooks for users dropdown
  const {
    data: usersData,
    isLoading: isLoadingUsers,
    error: usersError,
  } = useUsersPaginated({
    page: 1,
    pageSize: 100, // Get first 100 users for dropdown
  });

  // API hooks for daily questions
  const {
    data: dailyQuestions,
    isLoading: isLoadingQuestions,
    refetch: refetchQuestions,
  } = useGetV1AdminWorkerDailyUsersUserIdQuestionsDate(
    selectedUser!,
    selectedDate,
    {
      query: {
        enabled: !!selectedUser,
      },
    }
  );

  // API hook for available dates
  const { data: availableDatesResponse } = useGetV1DailyDates({
    query: {
      enabled: !!user,
    },
  });

  const regenerateMutation =
    usePostV1AdminWorkerDailyUsersUserIdQuestionsDateRegenerate();

  // Determine admin status (defer early returns until after hooks)
  const isAdmin = user?.roles?.some(role => role.name === 'admin') || false;

  const handleDateSelect = (date: string | null) => {
    if (date) {
      setSelectedDate(date);
    }
  };

  // Convert availableDatesResponse to string array
  const availableDates = availableDatesResponse?.dates?.map(
    (date: string | Date) =>
      typeof date === 'string' ? formatDateForAPI(date) : formatDateForAPI(date)
  );

  const handleUserSelect = (value: string | null) => {
    setSelectedUser(value ? parseInt(value) : null);
  };

  // Initialize selected user from localStorage (URL username resolved in a separate effect)
  React.useEffect(() => {
    if (selectedUser != null) return; // already set
    const fromUrl = searchParams.get('user');
    if (fromUrl) {
      // It's a username; wait for usersData and resolve in another effect
    }
    const stored = localStorage.getItem('dailyAdmin.selectedUserId');
    if (stored) {
      const parsed = parseInt(stored, 10);
      if (!Number.isNaN(parsed)) {
        setSelectedUser(parsed);
      }
    }
    // Intentionally omit dependencies to avoid repeated initialization
  }, []);

  // Resolve username from URL to user ID when users list is available
  React.useEffect(() => {
    if (selectedUser != null) return; // already set
    const fromUrl = searchParams.get('user');
    if (!fromUrl) return;
    const list: Array<{ user: { id: number; username: string } }> | undefined =
      usersData?.users;
    if (!list || list.length === 0) return;
    const found = list.find(
      item => item?.user?.username?.toLowerCase() === fromUrl.toLowerCase()
    );
    if (found) {
      setSelectedUser(found.user.id);
    }
  }, [usersData, searchParams, selectedUser]);

  // Keep URL (username) and localStorage (id) in sync with selection
  React.useEffect(() => {
    const next = new URLSearchParams(searchParams);
    if (selectedUser != null) {
      localStorage.setItem('dailyAdmin.selectedUserId', String(selectedUser));
      // Try to find the username from the loaded users list; fall back to id
      const list:
        | Array<{ user: { id: number; username: string } }>
        | undefined = usersData?.users;
      const userObj = list?.find(u => u?.user?.id === selectedUser)?.user;
      const urlValue = userObj?.username ?? String(selectedUser);
      next.set('user', urlValue);
    } else {
      next.delete('user');
      localStorage.removeItem('dailyAdmin.selectedUserId');
    }
    // Only update if changed to avoid loops
    if (next.toString() !== searchParams.toString()) {
      setSearchParams(next, { replace: true });
    }
  }, [selectedUser, usersData, searchParams, setSearchParams]);

  const handleRegenerate = async () => {
    if (!selectedUser) {
      notifications.show({
        title: 'Error',
        message: 'Please select a user first',
        color: 'red',
      });
      return;
    }

    setIsRegenerating(true);
    try {
      await regenerateMutation.mutateAsync({
        userId: selectedUser,
        date: selectedDate,
      });

      notifications.show({
        title: 'Success',
        message: 'Daily questions regenerated successfully',
        color: 'green',
      });

      // Refetch questions to show updated data
      refetchQuestions();
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to regenerate daily questions',
        color: 'red',
      });
    } finally {
      setIsRegenerating(false);
    }
  };

  // Prepare user dropdown data
  const userSelectData =
    usersData?.users
      ?.filter(
        (item: { user?: { id?: number; username?: string } }) =>
          item?.user && item.user.id && item.user.username
      )
      ?.map(
        (item: { user: { id: number; username: string; email: string } }) => ({
          value: item.user.id.toString(),
          label: `${item.user.username} (${item.user.email || 'No email'})`,
        })
      ) || [];

  const selectedUserData = usersData?.users?.find(
    (item: { user: { id: number; username: string; email: string } }) =>
      item?.user?.id === selectedUser
  )?.user;

  // Get questions from API
  const questions = dailyQuestions?.questions || [];

  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }
  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  return (
    <Container size='xl' py='md'>
      <Stack gap='lg'>
        <Group justify='space-between' align='center' wrap='wrap'>
          <Title order={1}>Daily Questions Administration</Title>
          <DailyDatePicker
            selectedDate={selectedDate}
            onDateSelect={handleDateSelect}
            availableDates={availableDates}
            placeholder='Pick date'
            maxDate={new Date()}
            size='sm'
            style={{ width: '200px' }}
            clearable
            hideOutsideDates
            withCellSpacing={false}
            firstDayOfWeek={1}
            isAdminMode={true}
          />
          <Button
            size='xs'
            variant='subtle'
            onClick={() => setSelectedDate(formatDateForAPI(new Date()))}
            ml={8}
          >
            Today
          </Button>
        </Group>

        {/* User Selection */}
        <Paper p='md' withBorder>
          <Title order={3} mb='md'>
            User Selection
          </Title>
          <Grid>
            <Grid.Col span={{ base: 12, md: 6 }}>
              <Select
                label='Select User'
                placeholder='Choose a user to view daily questions'
                data={userSelectData}
                value={selectedUser?.toString() || null}
                onChange={handleUserSelect}
                searchable
                clearable
                disabled={isLoadingUsers}
                leftSection={<IconUser size={16} />}
                error={usersError ? 'Failed to load users' : undefined}
              />
            </Grid.Col>
            <Grid.Col span={{ base: 12, md: 6 }}>
              {selectedUserData && (
                <Box>
                  <Text size='sm' c='dimmed' mb='xs'>
                    Selected User
                  </Text>
                  <Group>
                    <Badge variant='light' color='blue'>
                      {selectedUserData.username}
                    </Badge>
                    <Text size='sm'>
                      {selectedUserData.preferred_language} -{' '}
                      {selectedUserData.current_level}
                    </Text>
                  </Group>
                </Box>
              )}
            </Grid.Col>
          </Grid>
        </Paper>

        {/* Actions */}
        {selectedUser && (
          <Paper p='md' withBorder>
            <Group justify='space-between' align='center'>
              <div>
                <Title order={3}>Actions</Title>
                <Text size='sm' c='dimmed'>
                  Manage daily questions for {selectedUserData?.username} on{' '}
                  {selectedDate}
                </Text>
              </div>
              <Button
                leftSection={<IconRefresh size={16} />}
                onClick={handleRegenerate}
                loading={isRegenerating}
                color='orange'
              >
                Regenerate Questions
              </Button>
            </Group>
          </Paper>
        )}

        {/* Questions Display */}
        {selectedUser && (
          <Paper p='md' withBorder>
            <LoadingOverlay visible={isLoadingQuestions} />
            <Group justify='space-between' mb='md'>
              <Title order={3}>
                Daily Questions for {selectedUserData?.username}
              </Title>
              <Badge variant='light'>{selectedDate}</Badge>
            </Group>

            {questions.length === 0 ? (
              <Card p='xl' ta='center'>
                <Text c='dimmed'>
                  No daily questions found for this user and date.
                </Text>
                <Button
                  mt='md'
                  variant='light'
                  leftSection={<IconRefresh size={16} />}
                  onClick={handleRegenerate}
                  loading={isRegenerating}
                >
                  Generate Questions
                </Button>
              </Card>
            ) : (
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th style={{ width: '45%' }}>Question</Table.Th>
                    <Table.Th style={{ minWidth: 110 }}>Type</Table.Th>
                    <Table.Th style={{ minWidth: 70 }}>Level</Table.Th>
                    <Table.Th style={{ minWidth: 150 }}>
                      <Tooltip
                        label='From left to right: impressions (times assigned/shown), correct answers, incorrect answers for this user.'
                        withArrow
                      >
                        <Group gap={4} wrap='nowrap'>
                          <Text>Stats</Text>
                          <IconInfoCircle size={12} />
                        </Group>
                      </Tooltip>
                    </Table.Th>
                    <Table.Th style={{ minWidth: 120 }}>
                      <Tooltip
                        label='Whether the question was completed for this date.'
                        withArrow
                      >
                        <Group gap={4} wrap='nowrap'>
                          <Text>Status</Text>
                          <IconInfoCircle size={12} />
                        </Group>
                      </Tooltip>
                    </Table.Th>
                    <Table.Th style={{ minWidth: 200 }}>
                      <Tooltip
                        label='When the question was completed (if completed).'
                        withArrow
                      >
                        <Group gap={4} wrap='nowrap'>
                          <Text>Completed At</Text>
                          <IconInfoCircle size={12} />
                        </Group>
                      </Tooltip>
                    </Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {questions.map(dq => (
                    <Table.Tr key={dq.id}>
                      <Table.Td
                        style={{ cursor: 'pointer' }}
                        onClick={() => {
                          setSelectedAssignment(dq);
                          setDetailsOpen(true);
                        }}
                      >
                        <Text size='sm' lineClamp={1}>
                          {typeof dq.question?.content === 'object' &&
                          'question' in dq.question.content
                            ? dq.question.content.question
                            : 'Question content'}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <Badge variant='light' size='sm'>
                          {dq.question?.type}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Badge variant='outline' size='sm'>
                          {dq.question?.level}
                        </Badge>
                      </Table.Td>
                      <Table.Td style={{ whiteSpace: 'nowrap' }}>
                        <Group gap='xs' wrap='nowrap'>
                          <Badge color='gray' variant='light' size='sm'>
                            {dq.user_shown_count ?? 0}
                          </Badge>
                          <Badge color='green' variant='light' size='sm'>
                            {dq.user_correct_count ?? 0}
                          </Badge>
                          <Badge color='red' variant='light' size='sm'>
                            {dq.user_incorrect_count ?? 0}
                          </Badge>
                        </Group>
                      </Table.Td>
                      <Table.Td style={{ whiteSpace: 'nowrap' }}>
                        <Group gap='xs' wrap='nowrap'>
                          {dq.is_completed ? (
                            <IconCheck size={16} color='green' />
                          ) : (
                            <IconClock size={16} color='orange' />
                          )}
                          <Text
                            size='sm'
                            c={dq.is_completed ? 'green' : 'orange'}
                            style={{ whiteSpace: 'nowrap' }}
                          >
                            {dq.is_completed ? 'Completed' : 'Pending'}
                          </Text>
                        </Group>
                      </Table.Td>
                      <Table.Td style={{ whiteSpace: 'nowrap' }}>
                        <Text
                          size='sm'
                          c='dimmed'
                          style={{ whiteSpace: 'nowrap' }}
                        >
                          {dq.completed_at
                            ? new Date(dq.completed_at).toLocaleString()
                            : 'Not completed'}
                        </Text>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            )}
          </Paper>
        )}

        {/* Centered Question Details Modal */}
        <Modal
          opened={detailsOpen}
          onClose={() => setDetailsOpen(false)}
          title='Question Details'
          size='xl'
          centered
        >
          {selectedAssignment && (
            <Stack gap='md'>
              <Group gap='xs'>
                {selectedAssignment.question?.type && (
                  <Badge variant='light' color='blue'>
                    {selectedAssignment.question.type}
                  </Badge>
                )}
                {selectedAssignment.question?.level && (
                  <Badge variant='outline'>
                    {selectedAssignment.question.level}
                  </Badge>
                )}
                {selectedAssignment.question?.language && (
                  <Badge variant='light' color='gray'>
                    {selectedAssignment.question.language}
                  </Badge>
                )}
              </Group>

              {/* Passage (reading comprehension) */}
              {typeof selectedAssignment.question?.content === 'object' &&
                'passage' in selectedAssignment.question.content &&
                selectedAssignment.question.content.passage && (
                  <Paper p='sm' withBorder>
                    <Text size='sm'>
                      {selectedAssignment.question.content.passage}
                    </Text>
                  </Paper>
                )}

              <Text size='lg' fw={600}>
                {typeof selectedAssignment.question?.content === 'object' &&
                'question' in selectedAssignment.question.content
                  ? selectedAssignment.question.content.question
                  : 'Question'}
              </Text>

              <Group gap='xs' wrap='nowrap' style={{ whiteSpace: 'nowrap' }}>
                <Badge color='gray' variant='light'>
                  {selectedAssignment.user_shown_count ?? 0}
                </Badge>
                <Badge color='green' variant='light'>
                  {selectedAssignment.user_correct_count ?? 0}
                </Badge>
                <Badge color='red' variant='light'>
                  {selectedAssignment.user_incorrect_count ?? 0}
                </Badge>
                <Tooltip
                  label='Shown / Correct / Incorrect for this user'
                  withArrow
                >
                  <IconInfoCircle size={14} />
                </Tooltip>
              </Group>

              {Array.isArray(selectedAssignment.question?.content?.options) && (
                <Stack gap={6}>
                  {selectedAssignment.question?.content?.options?.map(
                    (opt: string, idx: number) => (
                      <Group key={idx} gap='xs'>
                        <Badge variant='light' color='gray'>
                          {idx + 1}
                        </Badge>
                        <Text>{opt}</Text>
                        {typeof selectedAssignment.question?.correct_answer ===
                          'number' &&
                          selectedAssignment.question.correct_answer ===
                            idx && <Badge color='green'>Correct</Badge>}
                      </Group>
                    )
                  )}
                </Stack>
              )}

              {selectedAssignment.question?.explanation && (
                <>
                  <Divider />
                  <Text size='sm' c='dimmed'>
                    Explanation
                  </Text>
                  <Text>{selectedAssignment.question.explanation}</Text>
                </>
              )}

              <Divider />

              <Group justify='space-between'>
                <Group gap='xs'>
                  {selectedAssignment.is_completed ? (
                    <>
                      <IconCheck size={16} color='green' />
                      <Text c='green'>Completed</Text>
                    </>
                  ) : (
                    <>
                      <IconClock size={16} color='orange' />
                      <Text c='orange'>Pending</Text>
                    </>
                  )}
                </Group>
                <Text size='sm' c='dimmed'>
                  {selectedAssignment.completed_at
                    ? new Date(selectedAssignment.completed_at).toLocaleString()
                    : 'Not completed'}
                </Text>
              </Group>
            </Stack>
          )}
        </Modal>

        {!selectedUser && (
          <Card p='xl' ta='center'>
            <Text c='dimmed' size='lg'>
              Please select a user to view their daily questions.
            </Text>
          </Card>
        )}
      </Stack>
    </Container>
  );
};

export default DailyAdminPage;
