import React, { useState } from 'react';
import {
  Container,
  Stack,
  Table,
  Badge,
  Text,
  Select,
  TextInput,
  Group,
  Button,
  ActionIcon,
  Pagination,
  Modal,
  Textarea,
  Image,
  Paper,
  SimpleGrid,
  Tooltip,
  Divider,
  Box,
} from '@mantine/core';
import { IconRefresh, IconCheck, IconClock, IconX } from '@tabler/icons-react';
import * as TablerIcons from '@tabler/icons-react';
import {
  useGetV1AdminBackendFeedback,
  usePatchV1AdminBackendFeedbackId,
  useDeleteV1AdminBackendFeedbackId,
  FeedbackUpdateRequest,
  FeedbackUpdateRequestStatus,
  GetV1AdminBackendFeedbackStatus,
  FeedbackReport,
} from '../../api/api';
import { customInstance } from '../../api/axios';
import { notifications } from '@mantine/notifications';
import { Alert } from '@mantine/core';
import { useUsersPaginated } from '../../api/admin';

const tablerIconMap = TablerIcons as unknown as Record<
  string,
  React.ComponentType<React.SVGProps<SVGSVGElement>>
>;
const IconSearch = tablerIconMap.IconSearch || (() => null);
const IconEye = tablerIconMap.IconEye || (() => null);
const IconBug = tablerIconMap.IconBug || (() => null);
const IconFilter = tablerIconMap.IconFilter || (() => null);
const IconAlertTriangle = tablerIconMap.IconAlertTriangle || (() => null);
const IconTrash = tablerIconMap.IconTrash || (() => null);

const FeedbackManagementPage: React.FC = () => {
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [typeFilter, setTypeFilter] = useState<string>('');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedFeedback, setSelectedFeedback] =
    useState<FeedbackReport | null>(null);
  const [detailModalOpened, setDetailModalOpened] = useState(false);
  const [updateStatus, setUpdateStatus] = useState('');
  const [updateNotes, setUpdateNotes] = useState('');
  const [deleteModalOpened, setDeleteModalOpened] = useState(false);
  const [feedbackToDelete, setFeedbackToDelete] =
    useState<FeedbackReport | null>(null);
  const [deleteAllResolvedModalOpened, setDeleteAllResolvedModalOpened] =
    useState(false);
  const [deleteAllDismissedModalOpened, setDeleteAllDismissedModalOpened] =
    useState(false);
  const [deleteAllModalOpened, setDeleteAllModalOpened] = useState(false);

  // Fetch users for username display
  const { data: usersData } = useUsersPaginated({
    page: 1,
    pageSize: 1000,
  });

  // Create a mapping from user_id to username
  const userIdToUsername = React.useMemo(() => {
    const mapping: Record<number, string> = {};
    if (usersData?.users) {
      for (const userData of usersData.users) {
        if (userData?.user?.id && userData?.user?.username) {
          mapping[userData.user.id] = userData.user.username;
        }
      }
    }
    return mapping;
  }, [usersData]);

  const { data, isLoading, refetch } = useGetV1AdminBackendFeedback(
    {
      page,
      page_size: pageSize,
      ...(statusFilter && {
        status: statusFilter as GetV1AdminBackendFeedbackStatus,
      }),
      ...(typeFilter && { feedback_type: typeFilter }),
    },
    {
      query: {
        refetchInterval: 30000, // Poll every 30 seconds
        refetchOnWindowFocus: true, // Refresh when returning to page
      },
    }
  );

  const { mutate: updateFeedback, isPending: isUpdating } =
    usePatchV1AdminBackendFeedbackId();

  const { mutate: deleteFeedback, isPending: isDeleting } =
    useDeleteV1AdminBackendFeedbackId();

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'new':
        return 'blue';
      case 'in_progress':
        return 'yellow';
      case 'resolved':
        return 'green';
      case 'dismissed':
        return 'gray';
      default:
        return 'blue';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'new':
        return <IconBug style={{ width: 14, height: 14 }} />;
      case 'in_progress':
        return <IconClock size={14} />;
      case 'resolved':
        return <IconCheck size={14} />;
      case 'dismissed':
        return <IconX size={14} />;
      default:
        return <IconBug style={{ width: 14, height: 14 }} />;
    }
  };

  const getTypeLabel = (type: string) => {
    switch (type) {
      case 'bug':
        return 'Bug Report';
      case 'feature_request':
        return 'Feature Request';
      case 'general':
        return 'General Feedback';
      case 'improvement':
        return 'Improvement';
      default:
        return type;
    }
  };

  const handleViewDetails = (feedback: FeedbackReport) => {
    setSelectedFeedback(feedback);
    setUpdateStatus(feedback.status);
    setUpdateNotes(feedback.admin_notes || '');
    setDetailModalOpened(true);
  };

  const handleDeleteClick = (feedback: FeedbackReport | null | undefined) => {
    if (!feedback) return;
    setFeedbackToDelete(feedback);
    setDeleteModalOpened(true);
  };

  const handleDeleteConfirm = () => {
    if (!feedbackToDelete) return;

    deleteFeedback(
      { id: feedbackToDelete.id },
      {
        onSuccess: () => {
          notifications.show({
            title: 'Success',
            message: 'Feedback deleted successfully',
            color: 'green',
          });
          refetch();
          setDeleteModalOpened(false);
          setFeedbackToDelete(null);
        },
        onError: (error: unknown) => {
          const errorMessage =
            error && typeof error === 'object' && 'response' in error
              ? (error as { response?: { data?: { message?: string } } })
                  .response?.data?.message
              : undefined;
          notifications.show({
            title: 'Error',
            message: errorMessage || 'Failed to delete feedback',
            color: 'red',
          });
        },
      }
    );
  };

  const handleDeleteAllResolved = async () => {
    try {
      await customInstance<{ deleted_count: number }>({
        url: '/v1/admin/backend/feedback',
        method: 'DELETE',
        params: { status: 'resolved' },
      });
      notifications.show({
        title: 'Success',
        message: 'All resolved feedback reports deleted',
        color: 'green',
      });
      refetch();
      setDeleteAllResolvedModalOpened(false);
    } catch (error: unknown) {
      const errorMessage =
        error && typeof error === 'object' && 'response' in error
          ? (error as { response?: { data?: { message?: string } } }).response
              ?.data?.message
          : undefined;
      notifications.show({
        title: 'Error',
        message: errorMessage || 'Failed to delete resolved feedback',
        color: 'red',
      });
    }
  };

  const handleDeleteAllDismissed = async () => {
    try {
      await customInstance<{ deleted_count: number }>({
        url: '/v1/admin/backend/feedback',
        method: 'DELETE',
        params: { status: 'dismissed' },
      });
      notifications.show({
        title: 'Success',
        message: 'All dismissed feedback reports deleted',
        color: 'green',
      });
      refetch();
      setDeleteAllDismissedModalOpened(false);
    } catch (error: unknown) {
      const errorMessage =
        error && typeof error === 'object' && 'response' in error
          ? (error as { response?: { data?: { message?: string } } }).response
              ?.data?.message
          : undefined;
      notifications.show({
        title: 'Error',
        message: errorMessage || 'Failed to delete dismissed feedback',
        color: 'red',
      });
    }
  };

  const handleDeleteAll = async () => {
    try {
      await customInstance<{ deleted_count: number }>({
        url: '/v1/admin/backend/feedback',
        method: 'DELETE',
        params: { all: 'true' },
      });
      notifications.show({
        title: 'Success',
        message: 'All feedback reports deleted',
        color: 'green',
      });
      refetch();
      setDeleteAllModalOpened(false);
    } catch (error: unknown) {
      const errorMessage =
        error && typeof error === 'object' && 'response' in error
          ? (error as { response?: { data?: { message?: string } } }).response
              ?.data?.message
          : undefined;
      notifications.show({
        title: 'Error',
        message: errorMessage || 'Failed to delete all feedback',
        color: 'red',
      });
    }
  };

  const handleSaveUpdate = () => {
    if (!selectedFeedback) return;

    const updates: Partial<FeedbackUpdateRequest> = {
      status: updateStatus as FeedbackUpdateRequestStatus,
    };

    if (updateNotes !== (selectedFeedback.admin_notes || '')) {
      updates.admin_notes = updateNotes;
    }

    // If status is being changed to resolved, set resolved_at
    if (updateStatus === 'resolved' && selectedFeedback.status !== 'resolved') {
      updates.resolved_at = new Date().toISOString();
    }

    updateFeedback(
      {
        id: selectedFeedback.id,
        data: updates,
      },
      {
        onSuccess: () => {
          notifications.show({
            title: 'Success',
            message: 'Feedback updated successfully',
            color: 'green',
          });
          refetch();
          setDetailModalOpened(false);
        },
        onError: (error: unknown) => {
          const errorMessage =
            error && typeof error === 'object' && 'response' in error
              ? (error as { response?: { data?: { message?: string } } })
                  .response?.data?.message
              : undefined;
          notifications.show({
            title: 'Error',
            message: errorMessage || 'Failed to update feedback',
            color: 'red',
          });
        },
      }
    );
  };

  const filteredItems =
    data?.items?.filter(item => {
      if (!searchTerm) return true;
      const searchLower = searchTerm.toLowerCase();
      return (
        item.feedback_text.toLowerCase().includes(searchLower) ||
        item.id.toString().includes(searchLower) ||
        item.user_id.toString().includes(searchLower)
      );
    }) || [];

  const stats = React.useMemo(() => {
    const items = data?.items || [];
    return {
      total: data?.total || 0,
      new: items.filter(i => i.status === 'new').length,
      inProgress: items.filter(i => i.status === 'in_progress').length,
      resolved: items.filter(i => i.status === 'resolved').length,
      dismissed: items.filter(i => i.status === 'dismissed').length,
    };
  }, [data]);

  const totalPages = Math.ceil((data?.total || 0) / pageSize);

  return (
    <Container size='xl' py='xl'>
      <Stack gap='xl'>
        {/* Stats Cards */}
        <SimpleGrid cols={{ base: 1, sm: 2, md: 4 }} spacing='md'>
          <Paper p='md' withBorder radius='md'>
            <Stack gap='xs'>
              <Text size='sm' c='dimmed'>
                Total
              </Text>
              <Text size='xl' fw={700}>
                {stats.total}
              </Text>
            </Stack>
          </Paper>
          <Paper p='md' withBorder radius='md'>
            <Stack gap='xs'>
              <Text size='sm' c='dimmed'>
                New
              </Text>
              <Text size='xl' fw={700} c='blue'>
                {stats.new}
              </Text>
            </Stack>
          </Paper>
          <Paper p='md' withBorder radius='md'>
            <Stack gap='xs'>
              <Text size='sm' c='dimmed'>
                In Progress
              </Text>
              <Text size='xl' fw={700} c='yellow'>
                {stats.inProgress}
              </Text>
            </Stack>
          </Paper>
          <Paper p='md' withBorder radius='md'>
            <Stack gap='xs'>
              <Text size='sm' c='dimmed'>
                Resolved
              </Text>
              <Text size='xl' fw={700} c='green'>
                {stats.resolved}
              </Text>
            </Stack>
          </Paper>
        </SimpleGrid>

        {/* Filters */}
        <Paper p='md' withBorder radius='md'>
          <Stack gap='md'>
            <Group justify='space-between'>
              <Text fw={600} size='lg'>
                Feedback Reports
              </Text>
              <Group gap='xs'>
                {stats.resolved > 0 && (
                  <Button
                    leftSection={
                      <IconTrash style={{ width: 16, height: 16 }} />
                    }
                    onClick={() => setDeleteAllResolvedModalOpened(true)}
                    variant='outline'
                    color='red'
                    size='sm'
                  >
                    Delete All Resolved ({stats.resolved})
                  </Button>
                )}
                {stats.dismissed > 0 && (
                  <Button
                    leftSection={
                      <IconTrash style={{ width: 16, height: 16 }} />
                    }
                    onClick={() => setDeleteAllDismissedModalOpened(true)}
                    variant='outline'
                    color='gray'
                    size='sm'
                  >
                    Delete All Dismissed ({stats.dismissed})
                  </Button>
                )}
                {stats.total > 0 && (
                  <Button
                    leftSection={
                      <IconTrash style={{ width: 16, height: 16 }} />
                    }
                    onClick={() => setDeleteAllModalOpened(true)}
                    variant='outline'
                    color='red'
                    size='sm'
                  >
                    Delete All ({stats.total})
                  </Button>
                )}
                <Button
                  leftSection={<IconRefresh size={16} />}
                  onClick={() => refetch()}
                  variant='subtle'
                  size='sm'
                >
                  Refresh
                </Button>
              </Group>
            </Group>

            <Group gap='md'>
              <TextInput
                placeholder='Search feedback...'
                leftSection={<IconSearch style={{ width: 16, height: 16 }} />}
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                style={{ flex: 1 }}
              />

              <Select
                placeholder='Status'
                data={[
                  { value: '', label: 'All Statuses' },
                  { value: 'new', label: 'New' },
                  { value: 'in_progress', label: 'In Progress' },
                  { value: 'resolved', label: 'Resolved' },
                  { value: 'dismissed', label: 'Dismissed' },
                ]}
                value={statusFilter}
                onChange={value => {
                  setStatusFilter(value || '');
                  setPage(1);
                }}
                leftSection={<IconFilter style={{ width: 16, height: 16 }} />}
                clearable
              />

              <Select
                placeholder='Type'
                data={[
                  { value: '', label: 'All Types' },
                  { value: 'bug', label: 'Bug Report' },
                  { value: 'feature_request', label: 'Feature Request' },
                  { value: 'general', label: 'General' },
                  { value: 'improvement', label: 'Improvement' },
                ]}
                value={typeFilter}
                onChange={value => {
                  setTypeFilter(value || '');
                  setPage(1);
                }}
                clearable
              />
            </Group>
          </Stack>
        </Paper>

        {/* Table */}
        <Paper p='md' withBorder radius='md'>
          {isLoading ? (
            <Text>Loading...</Text>
          ) : (
            <>
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>ID</Table.Th>
                    <Table.Th>User</Table.Th>
                    <Table.Th>Type</Table.Th>
                    <Table.Th>Status</Table.Th>
                    <Table.Th>Feedback</Table.Th>
                    <Table.Th>Screenshot</Table.Th>
                    <Table.Th>Created</Table.Th>
                    <Table.Th>Actions</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {filteredItems.map(item => (
                    <Table.Tr
                      key={item.id}
                      onClick={() => handleViewDetails(item)}
                      style={{ cursor: 'pointer' }}
                    >
                      <Table.Td>{item.id}</Table.Td>
                      <Table.Td>
                        {userIdToUsername[item.user_id] ||
                          `User ${item.user_id}`}
                      </Table.Td>
                      <Table.Td>
                        <Badge variant='light'>
                          {getTypeLabel(item.feedback_type)}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Badge
                          color={getStatusColor(item.status)}
                          leftSection={getStatusIcon(item.status)}
                        >
                          {item.status}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        <Tooltip label={item.feedback_text} withArrow>
                          <Text size='sm' truncate style={{ maxWidth: 300 }}>
                            {item.feedback_text}
                          </Text>
                        </Tooltip>
                      </Table.Td>
                      <Table.Td>
                        {item.screenshot_data || item.screenshot_url ? (
                          <Tooltip
                            label='Click to view full screenshot'
                            withArrow
                          >
                            <Box
                              style={{
                                width: 60,
                                height: 40,
                                borderRadius: 4,
                                overflow: 'hidden',
                                border: '1px solid #e0e0e0',
                              }}
                            >
                              <Image
                                src={
                                  item.screenshot_url ||
                                  (item.screenshot_data?.startsWith('data:')
                                    ? item.screenshot_data
                                    : `data:image/jpeg;base64,${item.screenshot_data}`)
                                }
                                alt='Screenshot thumbnail'
                                fit='cover'
                                style={{ width: '100%', height: '100%' }}
                              />
                            </Box>
                          </Tooltip>
                        ) : (
                          <Text size='sm' c='dimmed'>
                            —
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        <Text size='sm'>
                          {new Date(item.created_at).toLocaleDateString()}
                        </Text>
                      </Table.Td>
                      <Table.Td onClick={e => e.stopPropagation()}>
                        <Group gap='xs'>
                          <ActionIcon
                            variant='subtle'
                            onClick={() => handleViewDetails(item)}
                            title='View Details'
                          >
                            <IconEye style={{ width: 18, height: 18 }} />
                          </ActionIcon>
                          <ActionIcon
                            variant='subtle'
                            color='red'
                            onClick={() => handleDeleteClick(item)}
                            title='Delete'
                          >
                            <IconTrash style={{ width: 18, height: 18 }} />
                          </ActionIcon>
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>

              {totalPages > 1 && (
                <Group justify='center' mt='md'>
                  <Pagination
                    value={page}
                    onChange={setPage}
                    total={totalPages}
                    siblings={1}
                    boundaries={1}
                  />
                </Group>
              )}
            </>
          )}
        </Paper>

        {/* Detail Modal */}
        <Modal
          opened={detailModalOpened}
          onClose={() => setDetailModalOpened(false)}
          title={`Feedback #${selectedFeedback?.id}`}
          size='xl'
        >
          {selectedFeedback && (
            <Stack gap='md'>
              {/* Status and Type */}
              <Group justify='space-between' align='flex-start'>
                <Stack gap='xs'>
                  <Text size='sm' fw={500}>
                    Status
                  </Text>
                  <Select
                    data={[
                      { value: 'new', label: 'New' },
                      { value: 'in_progress', label: 'In Progress' },
                      { value: 'resolved', label: 'Resolved' },
                      { value: 'dismissed', label: 'Dismissed' },
                    ]}
                    value={updateStatus}
                    onChange={value => setUpdateStatus(value || '')}
                    style={{ minWidth: 200 }}
                  />
                </Stack>
                <Stack gap='xs'>
                  <Text size='sm' fw={500}>
                    Type
                  </Text>
                  <Badge variant='light' size='lg'>
                    {getTypeLabel(selectedFeedback.feedback_type)}
                  </Badge>
                </Stack>
              </Group>

              {/* Feedback Text */}
              <Stack gap='xs'>
                <Text size='sm' fw={500}>
                  Feedback
                </Text>
                <Paper p='md' withBorder>
                  <Text size='sm' style={{ whiteSpace: 'pre-wrap' }}>
                    {selectedFeedback.feedback_text}
                  </Text>
                </Paper>
              </Stack>

              {/* Screenshot */}
              {(selectedFeedback.screenshot_data ||
                selectedFeedback.screenshot_url) && (
                <Stack gap='xs'>
                  <Text size='sm' fw={500}>
                    Screenshot
                  </Text>
                  <Image
                    src={
                      selectedFeedback.screenshot_url ||
                      (selectedFeedback.screenshot_data?.startsWith('data:')
                        ? selectedFeedback.screenshot_data
                        : `data:image/jpeg;base64,${selectedFeedback.screenshot_data}`)
                    }
                    alt='Feedback screenshot'
                    radius='md'
                    fit='contain'
                    style={{ maxWidth: '100%', maxHeight: 600 }}
                    fallbackSrc='data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="400" height="300"%3E%3Crect width="400" height="300" fill="%23f0f0f0"/%3E%3Ctext x="50%25" y="50%25" text-anchor="middle" dy=".3em" fill="%23999"%3EFailed to load image%3C/text%3E%3C/svg%3E'
                  />
                </Stack>
              )}

              {/* Context Data */}
              {selectedFeedback.context_data && (
                <Stack gap='xs'>
                  <Text size='sm' fw={500}>
                    Context Data
                  </Text>
                  <Table withTableBorder withColumnBorders striped>
                    <Table.Tbody>
                      {Object.entries(selectedFeedback.context_data).map(
                        ([key, value]) => (
                          <Table.Tr key={key}>
                            <Table.Td
                              style={{
                                fontWeight: 500,
                                width: '30%',
                                verticalAlign: 'top',
                              }}
                            >
                              {key}
                            </Table.Td>
                            <Table.Td
                              style={{
                                fontFamily: 'monospace',
                                fontSize: '0.9em',
                                wordBreak: 'break-word',
                              }}
                            >
                              {typeof value === 'object'
                                ? JSON.stringify(value, null, 2)
                                : String(value)}
                            </Table.Td>
                          </Table.Tr>
                        )
                      )}
                    </Table.Tbody>
                  </Table>
                </Stack>
              )}

              {/* Admin Notes */}
              <Divider />
              <Stack gap='xs'>
                <Text size='sm' fw={500}>
                  Admin Notes
                </Text>
                <Textarea
                  placeholder='Add notes about this feedback...'
                  value={updateNotes}
                  onChange={e => setUpdateNotes(e.target.value)}
                  minRows={3}
                />
              </Stack>

              {/* Metadata */}
              <Group justify='space-between' c='dimmed'>
                <Text size='xs'>
                  User:{' '}
                  {userIdToUsername[selectedFeedback.user_id] || 'Unknown'} (ID:{' '}
                  {selectedFeedback.user_id})
                </Text>
                <Text size='xs'>
                  Created:{' '}
                  {new Date(selectedFeedback.created_at).toLocaleString()}
                </Text>
              </Group>

              {/* Actions */}
              <Group justify='flex-end' mt='md'>
                <Button
                  variant='subtle'
                  onClick={() => setDetailModalOpened(false)}
                  disabled={isUpdating}
                >
                  Close
                </Button>
                <Button
                  onClick={handleSaveUpdate}
                  loading={isUpdating}
                  leftSection={<IconCheck />}
                >
                  Save Changes
                </Button>
              </Group>
            </Stack>
          )}
        </Modal>

        {/* Delete Confirmation Modal */}
        <Modal
          opened={deleteModalOpened}
          onClose={() => {
            setDeleteModalOpened(false);
            setFeedbackToDelete(null);
          }}
          title='Delete Feedback Report'
          size='sm'
        >
          <Stack gap='md'>
            <Alert
              icon={<IconAlertTriangle style={{ width: 16, height: 16 }} />}
              color='red'
            >
              Are you sure you want to delete feedback #{feedbackToDelete?.id}?
              This action cannot be undone.
            </Alert>
            <Group justify='flex-end'>
              <Button
                variant='subtle'
                onClick={() => {
                  setDeleteModalOpened(false);
                  setFeedbackToDelete(null);
                }}
                disabled={isDeleting}
              >
                Cancel
              </Button>
              <Button
                color='red'
                onClick={handleDeleteConfirm}
                loading={isDeleting}
              >
                Delete
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Delete All Resolved Confirmation Modal */}
        <Modal
          opened={deleteAllResolvedModalOpened}
          onClose={() => setDeleteAllResolvedModalOpened(false)}
          title='Delete All Resolved Reports'
          size='sm'
        >
          <Stack gap='md'>
            <Alert
              icon={<IconAlertTriangle style={{ width: 16, height: 16 }} />}
              color='red'
            >
              Are you sure you want to delete all {stats.resolved} resolved
              feedback reports? This action cannot be undone.
            </Alert>
            <Group justify='flex-end'>
              <Button
                variant='subtle'
                onClick={() => setDeleteAllResolvedModalOpened(false)}
              >
                Cancel
              </Button>
              <Button
                color='red'
                onClick={handleDeleteAllResolved}
                leftSection={<IconTrash style={{ width: 16, height: 16 }} />}
              >
                Delete All Resolved
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Delete All Dismissed Confirmation Modal */}
        <Modal
          opened={deleteAllDismissedModalOpened}
          onClose={() => setDeleteAllDismissedModalOpened(false)}
          title='Delete All Dismissed Reports'
          size='sm'
        >
          <Stack gap='md'>
            <Alert
              icon={<IconAlertTriangle style={{ width: 16, height: 16 }} />}
              color='yellow'
            >
              Are you sure you want to delete all {stats.dismissed} dismissed
              feedback reports? This action cannot be undone.
            </Alert>
            <Group justify='flex-end'>
              <Button
                variant='subtle'
                onClick={() => setDeleteAllDismissedModalOpened(false)}
              >
                Cancel
              </Button>
              <Button
                color='gray'
                onClick={handleDeleteAllDismissed}
                leftSection={<IconTrash style={{ width: 16, height: 16 }} />}
              >
                Delete All Dismissed
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Delete All Confirmation Modal */}
        <Modal
          opened={deleteAllModalOpened}
          onClose={() => setDeleteAllModalOpened(false)}
          title='Delete All Feedback Reports'
          size='sm'
        >
          <Stack gap='md'>
            <Alert
              icon={<IconAlertTriangle style={{ width: 16, height: 16 }} />}
              color='red'
            >
              Are you sure you want to delete ALL {stats.total} feedback reports
              regardless of status? This action cannot be undone.
            </Alert>
            <Group justify='flex-end'>
              <Button
                variant='subtle'
                onClick={() => setDeleteAllModalOpened(false)}
              >
                Cancel
              </Button>
              <Button
                color='red'
                onClick={handleDeleteAll}
                leftSection={<IconTrash style={{ width: 16, height: 16 }} />}
              >
                Delete All Feedback
              </Button>
            </Group>
          </Stack>
        </Modal>
      </Stack>
    </Container>
  );
};

export default FeedbackManagementPage;
