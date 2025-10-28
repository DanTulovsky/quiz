import React, {useState} from 'react';
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
  Code,
  SegmentedControl,
  Paper,
  SimpleGrid,
  Tooltip,
  Divider,
} from '@mantine/core';
import {
  IconSearch,
  IconEye,
  IconRefresh,
  IconBug,
  IconFilter,
  IconCheck,
  IconClock,
  IconX,
} from '@tabler/icons-react';
import {
  useGetV1AdminBackendFeedback,
  usePatchV1AdminBackendFeedbackId,
  FeedbackUpdateRequest,
  FeedbackUpdateRequestStatus,
} from '../../api/api';
import {notifications} from '@mantine/notifications';

interface FeedbackReport {
  id: number;
  user_id: number;
  feedback_text: string;
  feedback_type: string;
  context_data?: Record<string, unknown>;
  screenshot_data?: string;
  screenshot_url?: string;
  status: string;
  admin_notes?: string;
  assigned_to_user_id?: number;
  resolved_at?: string;
  resolved_by_user_id?: number;
  created_at: string;
  updated_at: string;
}

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

  const {data, isLoading, refetch} = useGetV1AdminBackendFeedback({
    page,
    page_size: pageSize,
    ...(statusFilter && {status: statusFilter}),
    ...(typeFilter && {feedback_type: typeFilter}),
  });

  const {mutate: updateFeedback, isPending: isUpdating} =
    usePatchV1AdminBackendFeedbackId();

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
        return <IconBug size={14} />;
      case 'in_progress':
        return <IconClock size={14} />;
      case 'resolved':
        return <IconCheck size={14} />;
      case 'dismissed':
        return <IconX size={14} />;
      default:
        return <IconBug size={14} />;
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
              ? (error as {response?: {data?: {message?: string}}})
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
    };
  }, [data]);

  const totalPages = Math.ceil((data?.total || 0) / pageSize);

  return (
    <Container size='xl' py='xl'>
      <Stack gap='xl'>
        {/* Stats Cards */}
        <SimpleGrid cols={{base: 1, sm: 2, md: 4}} spacing='md'>
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
              <Button
                leftSection={<IconRefresh size={16} />}
                onClick={() => refetch()}
                variant='subtle'
                size='sm'
              >
                Refresh
              </Button>
            </Group>

            <Group gap='md'>
              <TextInput
                placeholder='Search feedback...'
                leftSection={<IconSearch size={16} />}
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                style={{flex: 1}}
              />

              <Select
                placeholder='Status'
                data={[
                  {value: '', label: 'All Statuses'},
                  {value: 'new', label: 'New'},
                  {value: 'in_progress', label: 'In Progress'},
                  {value: 'resolved', label: 'Resolved'},
                  {value: 'dismissed', label: 'Dismissed'},
                ]}
                value={statusFilter}
                onChange={value => {
                  setStatusFilter(value || '');
                  setPage(1);
                }}
                leftSection={<IconFilter size={16} />}
                clearable
              />

              <Select
                placeholder='Type'
                data={[
                  {value: '', label: 'All Types'},
                  {value: 'bug', label: 'Bug Report'},
                  {value: 'feature_request', label: 'Feature Request'},
                  {value: 'general', label: 'General'},
                  {value: 'improvement', label: 'Improvement'},
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
                    <Table.Tr key={item.id}>
                      <Table.Td>{item.id}</Table.Td>
                      <Table.Td>{item.user_id}</Table.Td>
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
                          <Text size='sm' truncate style={{maxWidth: 300}}>
                            {item.feedback_text}
                          </Text>
                        </Tooltip>
                      </Table.Td>
                      <Table.Td>
                        {item.screenshot_data || item.screenshot_url ? (
                          <Badge color='blue'>Has Screenshot</Badge>
                        ) : (
                          <Text size='sm' c='dimmed'>
                            -
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        <Text size='sm'>
                          {new Date(item.created_at).toLocaleDateString()}
                        </Text>
                      </Table.Td>
                      <Table.Td>
                        <ActionIcon
                          variant='subtle'
                          onClick={() => handleViewDetails(item)}
                          title='View Details'
                        >
                          <IconEye size={18} />
                        </ActionIcon>
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
              <Group justify='space-between'>
                <Stack gap='xs'>
                  <Text size='sm' fw={500}>
                    Status
                  </Text>
                  <Badge
                    color={getStatusColor(selectedFeedback.status)}
                    size='lg'
                    leftSection={getStatusIcon(selectedFeedback.status)}
                  >
                    {selectedFeedback.status}
                  </Badge>
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
                  <Text size='sm' style={{whiteSpace: 'pre-wrap'}}>
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
                        selectedFeedback.screenshot_data ||
                        selectedFeedback.screenshot_url
                      }
                      alt='Feedback screenshot'
                      radius='md'
                    />
                  </Stack>
                )}

              {/* Context Data */}
              {selectedFeedback.context_data && (
                <Stack gap='xs'>
                  <Text size='sm' fw={500}>
                    Context Data
                  </Text>
                  <Code
                    block
                    p='md'
                    style={{maxHeight: 200, overflow: 'auto'}}
                  >
                    {JSON.stringify(selectedFeedback.context_data, null, 2)}
                  </Code>
                </Stack>
              )}

              {/* Update Form */}
              <Divider />
              <Stack gap='xs'>
                <Text size='sm' fw={500}>
                  Update Status
                </Text>
                <SegmentedControl
                  data={[
                    {value: 'new', label: 'New'},
                    {value: 'in_progress', label: 'In Progress'},
                    {value: 'resolved', label: 'Resolved'},
                    {value: 'dismissed', label: 'Dismissed'},
                  ]}
                  value={updateStatus}
                  onChange={setUpdateStatus}
                  fullWidth
                />
              </Stack>

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
                <Text size='xs'>User ID: {selectedFeedback.user_id}</Text>
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
      </Stack>
    </Container>
  );
};

export default FeedbackManagementPage;
