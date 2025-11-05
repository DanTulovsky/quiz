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
import {
  IconRefresh,
  IconCheck,
  IconClock,
  IconX,
  IconExternalLink,
} from '@tabler/icons-react';
import * as TablerIcons from '@tabler/icons-react';
import {
  useGetV1AdminBackendFeedback,
  usePatchV1AdminBackendFeedbackId,
  useDeleteV1AdminBackendFeedbackId,
  usePostV1AdminBackendFeedbackIdLinearIssue,
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
  const [errorModalOpened, setErrorModalOpened] = useState(false);
  const [errorDetails, setErrorDetails] = useState<{
    title: string;
    message: string;
    details?: string;
  } | null>(null);

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

  const { mutate: createLinearIssue, isPending: isCreatingLinearIssue } =
    usePostV1AdminBackendFeedbackIdLinearIssue();
  const [creatingLinearIssueForId, setCreatingLinearIssueForId] = useState<
    number | null
  >(null);

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

  const handleCreateLinearIssue = (feedbackId?: number) => {
    const feedbackIdToUse = feedbackId || selectedFeedback?.id;
    if (!feedbackIdToUse) return;

    setCreatingLinearIssueForId(feedbackIdToUse);
    createLinearIssue(
      { id: feedbackIdToUse },
      {
        onSuccess: response => {
          setCreatingLinearIssueForId(null);
          notifications.show({
            title: 'Linear Issue Created',
            message: (
              <div>
                Successfully created Linear issue.{' '}
                <a
                  href={response.issue_url}
                  target='_blank'
                  rel='noopener noreferrer'
                  style={{ textDecoration: 'underline' }}
                >
                  View Issue
                </a>
              </div>
            ),
            color: 'green',
          });
        },
        onError: (error: unknown) => {
          setCreatingLinearIssueForId(null);

          // Extract error details
          let errorTitle = 'Error';
          let errorMessage = 'Failed to create Linear issue';
          let errorDetails: string | undefined;

          if (error && typeof error === 'object' && 'response' in error) {
            const response = (
              error as { response?: { data?: { message?: string } } }
            ).response;
            if (response?.data) {
              const data = response.data;

              // Try to parse the error message for Linear API errors
              if (data.message) {
                errorMessage = data.message;

                // Check if it contains Linear validation errors
                if (data.message.includes('Argument Validation Error')) {
                  errorTitle = 'Linear Validation Error';

                  // Try to extract user-presentable message from extensions
                  try {
                    // Look for userPresentableMessage in the error
                    const extensionsMatch = data.message.match(
                      /userPresentableMessage":"([^"]+)"/
                    );
                    if (extensionsMatch) {
                      errorMessage = extensionsMatch[1];
                    }

                    // Extract validation errors - try to parse the full JSON
                    try {
                      // Find the extensions JSON object
                      const extensionsStart =
                        data.message.indexOf('Extensions: {');
                      if (extensionsStart !== -1) {
                        const extensionsStr = data.message.substring(
                          extensionsStart + 12
                        ); // Skip "Extensions: "
                        // Try to find the end of the JSON object (matching braces)
                        let braceCount = 0;
                        let endIndex = -1;
                        for (let i = 0; i < extensionsStr.length; i++) {
                          if (extensionsStr[i] === '{') braceCount++;
                          if (extensionsStr[i] === '}') braceCount--;
                          if (braceCount === 0) {
                            endIndex = i + 1;
                            break;
                          }
                        }

                        if (endIndex > 0) {
                          try {
                            const extensionsJson = JSON.parse(
                              extensionsStr.substring(0, endIndex)
                            ) as {
                              validationErrors?: Array<{
                                property?: string;
                                constraints?: Record<string, string>;
                                value?: string;
                              }>;
                            };
                            if (
                              extensionsJson.validationErrors &&
                              Array.isArray(extensionsJson.validationErrors)
                            ) {
                              const validationMessages =
                                extensionsJson.validationErrors
                                  .map(ve => {
                                    const property = ve.property || 'unknown';
                                    const constraint = ve.constraints
                                      ? (Object.values(
                                          ve.constraints
                                        )[0] as string)
                                      : 'Invalid value';
                                    const value = ve.value
                                      ? ` (value: "${ve.value}")`
                                      : '';
                                    return `• ${property}: ${constraint}${value}`;
                                  })
                                  .join('\n');
                              if (validationMessages) {
                                errorDetails = validationMessages;
                              }
                            }
                          } catch {
                            // JSON parsing failed, fall through to regex parsing
                          }
                        }
                      }

                      // Fallback to regex parsing if JSON parsing failed
                      if (!errorDetails) {
                        const validationErrorsMatch = data.message.match(
                          /"validationErrors":\[(.*?)\]/
                        );
                        if (validationErrorsMatch) {
                          try {
                            // Try to extract individual validation errors
                            const validationStr = validationErrorsMatch[1];
                            const propertyMatches = [
                              ...validationStr.matchAll(
                                /"property":"([^"]+)"/g
                              ),
                            ];
                            const constraintMatches = [
                              ...validationStr.matchAll(
                                /"constraints":\{"isUuid":"([^"]+)"/g
                              ),
                            ];
                            const valueMatches = [
                              ...validationStr.matchAll(/"value":"([^"]+)"/g),
                            ];

                            const messages: string[] = [];
                            for (let i = 0; i < propertyMatches.length; i++) {
                              const property =
                                propertyMatches[i]?.[1] || 'unknown';
                              const constraint =
                                constraintMatches[i]?.[1] || 'Invalid value';
                              const value = valueMatches[i]?.[1] || '';
                              messages.push(
                                `• ${property}: ${constraint}${value ? ` (value: "${value}")` : ''}`
                              );
                            }

                            if (messages.length > 0) {
                              errorDetails = messages.join('\n');
                            }
                          } catch {
                            // If parsing fails, just show the raw message
                          }
                        }
                      }
                    } catch {
                      // If parsing fails, use the original message
                    }
                  } catch {
                    // If parsing fails, use the original message
                  }
                }
              }
            }
          }

          // Show notification
          notifications.show({
            title: errorTitle,
            message: errorMessage,
            color: 'red',
          });

          // Show detailed error modal
          setErrorDetails({
            title: errorTitle,
            message: errorMessage,
            details: errorDetails,
          });
          setErrorModalOpened(true);
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
                          <Tooltip label='Create Linear Issue'>
                            <ActionIcon
                              variant='subtle'
                              color='blue'
                              onClick={() => handleCreateLinearIssue(item.id)}
                              title='Create Linear Issue'
                              disabled={creatingLinearIssueForId === item.id}
                            >
                              <IconExternalLink
                                style={{
                                  width: 18,
                                  height: 18,
                                  opacity:
                                    creatingLinearIssueForId === item.id
                                      ? 0.5
                                      : 1,
                                }}
                              />
                            </ActionIcon>
                          </Tooltip>
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
              {/* Action Buttons at Top */}
              <Group justify='flex-end' mb='xs'>
                <Button
                  variant='subtle'
                  onClick={() => setDetailModalOpened(false)}
                  disabled={isUpdating || isCreatingLinearIssue}
                >
                  Close
                </Button>
                <Button
                  onClick={() => handleCreateLinearIssue()}
                  loading={isCreatingLinearIssue}
                  disabled={isUpdating}
                  leftSection={<IconExternalLink size={16} />}
                  variant='outline'
                >
                  Create Linear Issue
                </Button>
                <Button
                  onClick={handleSaveUpdate}
                  loading={isUpdating}
                  disabled={isCreatingLinearIssue}
                  leftSection={<IconCheck />}
                >
                  Save Changes
                </Button>
              </Group>

              <Divider />

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
                  disabled={isUpdating || isCreatingLinearIssue}
                >
                  Close
                </Button>
                <Button
                  onClick={handleCreateLinearIssue}
                  loading={isCreatingLinearIssue}
                  disabled={isUpdating}
                  leftSection={<IconExternalLink size={16} />}
                  variant='outline'
                >
                  Create Linear Issue
                </Button>
                <Button
                  onClick={handleSaveUpdate}
                  loading={isUpdating}
                  disabled={isCreatingLinearIssue}
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

        {/* Error Details Modal */}
        <Modal
          opened={errorModalOpened}
          onClose={() => {
            setErrorModalOpened(false);
            setErrorDetails(null);
          }}
          title={errorDetails?.title || 'Error'}
          size='lg'
        >
          {errorDetails && (
            <Stack gap='md'>
              <Alert color='red' icon={<IconAlertTriangle size={16} />}>
                <Text fw={500} mb='xs'>
                  {errorDetails.message}
                </Text>
                {errorDetails.details && (
                  <Paper
                    p='md'
                    mt='md'
                    withBorder
                    style={{ backgroundColor: '#f8f9fa' }}
                  >
                    <Text
                      size='sm'
                      style={{
                        whiteSpace: 'pre-wrap',
                        fontFamily: 'monospace',
                      }}
                    >
                      {errorDetails.details}
                    </Text>
                  </Paper>
                )}
              </Alert>

              <Text size='sm' c='dimmed'>
                Please check your Linear configuration:
                <ul style={{ marginTop: '8px', paddingLeft: '20px' }}>
                  <li>Team ID must be a valid UUID (not a team name)</li>
                  <li>Project ID must be a valid UUID (not a project name)</li>
                  <li>API key must be valid and have proper permissions</li>
                </ul>
              </Text>

              <Group justify='flex-end' mt='md'>
                <Button
                  onClick={() => {
                    setErrorModalOpened(false);
                    setErrorDetails(null);
                  }}
                >
                  Close
                </Button>
              </Group>
            </Stack>
          )}
        </Modal>
      </Stack>
    </Container>
  );
};

export default FeedbackManagementPage;
