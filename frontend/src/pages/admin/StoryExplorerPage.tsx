import React, { useMemo, useState } from 'react';
import {
  Container,
  Title,
  Text,
  Group,
  Card,
  Button,
  Select,
  TextInput,
  Table,
  Pagination,
  Loader,
  Alert,
  Modal,
  Stack,
  Tabs,
  Badge,
} from '@mantine/core';
import { Navigate } from 'react-router-dom';
import { IconBook, IconSearch, IconFilter } from '@tabler/icons-react';
import { useAuth } from '../../hooks/useAuth';
import {
  useAdminStories,
  useAdminStory,
  useAdminStorySection,
  useUsersPaginated,
  useAdminDeleteStory,
} from '../../api/admin';
import {
  useGetV1SettingsLanguages,
  type Story,
  type GetV1AdminBackendUserzPaginated200UsersItem,
} from '../../api/api';
import StoryReadingView from '../../components/StoryReadingView';
import StorySectionView from '../../components/StorySectionView';

const StoryExplorerPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();

  // Filters and pagination
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 20;
  const [filters, setFilters] = useState({
    search: '',
    language: '',
    status: '',
    user: '',
  });

  // Admin guard
  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }
  const isAdmin = user.roles?.some(r => r.name === 'admin') || false;
  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  // Languages for filter
  const { data: languagesData } = useGetV1SettingsLanguages();

  // Users for filter (first 1000)
  const { data: usersData, isLoading: isLoadingUsers } = useUsersPaginated({
    page: 1,
    pageSize: 1000,
  });
  const userOptions = useMemo(() => {
    const users = usersData?.users || [];
    return users
      .filter((u: GetV1AdminBackendUserzPaginated200UsersItem) => u?.user?.id)
      .map((u: GetV1AdminBackendUserzPaginated200UsersItem) => ({
        value: String(u.user!.id),
        label: u.user!.username || `user-${u.user!.id}`,
      }));
  }, [usersData]);

  // Fetch stories
  const {
    data: storiesResp,
    isLoading,
    error,
  } = useAdminStories(
    currentPage,
    pageSize,
    filters.search || undefined,
    filters.language || undefined,
    filters.status || undefined,
    filters.user ? parseInt(filters.user) : undefined
  );
  const stories = storiesResp?.stories || [];
  const totalStories = storiesResp?.pagination?.total || 0;

  // Modal state for viewing a story
  const [viewModalOpen, setViewModalOpen] = useState(false);
  const [selectedStoryId, setSelectedStoryId] = useState<number | null>(null);
  const { data: selectedStory } = useAdminStory(selectedStoryId);

  // Section tab state
  const [sectionIndex, setSectionIndex] = useState(0);
  const currentSectionId = selectedStory?.sections?.[sectionIndex]?.id || null;
  const { data: currentSectionWithQuestions } = useAdminStorySection(
    currentSectionId ? Number(currentSectionId) : null
  );

  const openView = (storyId: number) => {
    setSelectedStoryId(storyId);
    setSectionIndex(0);
    setViewModalOpen(true);
  };

  // Delete story (admin)
  const deleteStoryMutation = useAdminDeleteStory();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [storyToDelete, setStoryToDelete] = useState<number | null>(null);
  const triggerDelete = (storyId: number) => {
    setStoryToDelete(storyId);
    setConfirmOpen(true);
  };
  const confirmDelete = async () => {
    if (!storyToDelete) return;
    try {
      await deleteStoryMutation.mutateAsync(storyToDelete);
    } finally {
      setStoryToDelete(null);
    }
  };

  const clearFilters = () => {
    setFilters({ search: '', language: '', status: '', user: '' });
    setCurrentPage(1);
  };

  if (isLoading) {
    return (
      <Container size='xl' py='md'>
        <Group justify='center' mih={400}>
          <Loader size='lg' data-testid='loader' />
        </Group>
      </Container>
    );
  }

  if (error) {
    const message = (error as Error)?.message || 'Failed to load stories';
    return (
      <Container size='xl' py='md'>
        <Alert color='red' title='Error'>
          {message}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size='xl' py='md'>
      <Title order={1} mb='lg'>
        Story Explorer
      </Title>
      <Text c='dimmed' mb='xl'>
        Browse users' stories. Use filters to narrow results and click a row to
        view the full story.
      </Text>

      {/* Filters */}
      <Card shadow='sm' padding='md' radius='md' withBorder mb='md'>
        <Group justify='space-between' mb='md'>
          <Title order={4}>Filters</Title>
          <Button
            variant='light'
            size='sm'
            leftSection={<IconFilter size={16} />}
            onClick={clearFilters}
          >
            Clear All Filters
          </Button>
        </Group>
        <Group gap='sm' align='center' style={{ flexWrap: 'wrap' }}>
          <TextInput
            placeholder='Search title...'
            value={filters.search}
            onChange={e => setFilters({ ...filters, search: e.target.value })}
            leftSection={<IconSearch size={14} />}
            size='xs'
            style={{ minWidth: 180 }}
          />
          <Select
            placeholder='Language'
            value={filters.language}
            onChange={v => setFilters({ ...filters, language: v || '' })}
            data={[
              { value: '', label: 'All Languages' },
              ...(languagesData?.map(lang => ({
                value: lang,
                label: lang.charAt(0).toUpperCase() + lang.slice(1),
              })) || []),
            ]}
            clearable
            size='xs'
            style={{ minWidth: 160 }}
          />
          <Select
            placeholder='Status'
            value={filters.status}
            onChange={v => setFilters({ ...filters, status: v || '' })}
            data={[
              { value: '', label: 'All Statuses' },
              { value: 'active', label: 'Active' },
              { value: 'archived', label: 'Archived' },
              { value: 'completed', label: 'Completed' },
            ]}
            clearable
            size='xs'
            style={{ minWidth: 160 }}
          />
          <Select
            placeholder='User'
            value={filters.user || null}
            onChange={v => setFilters({ ...filters, user: v || '' })}
            data={userOptions}
            searchable
            clearable
            disabled={isLoadingUsers}
            size='xs'
            style={{ minWidth: 200 }}
          />
        </Group>
      </Card>

      {/* Stories Table */}
      <Card withBorder>
        <Card.Section p='md' bg='blue.0'>
          <Group>
            <IconBook />
            <Title order={2} size='h3'>
              Stories ({totalStories})
            </Title>
          </Group>
        </Card.Section>

        <div style={{ maxHeight: '600px', overflowY: 'auto' }}>
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>ID</Table.Th>
                <Table.Th>Title</Table.Th>
                <Table.Th>Language</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>User ID</Table.Th>
                <Table.Th>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {stories.map((s: Story) => (
                <Table.Tr
                  key={s.id}
                  onClick={() => openView(Number(s.id))}
                  style={{ cursor: 'pointer' }}
                >
                  <Table.Td>{s.id}</Table.Td>
                  <Table.Td>
                    <Text size='sm' style={{ maxWidth: 300 }}>
                      {s.title}
                    </Text>
                  </Table.Td>
                  <Table.Td>{s.language}</Table.Td>
                  <Table.Td>
                    <Badge
                      variant='light'
                      color={
                        s.status === 'active'
                          ? 'green'
                          : s.status === 'completed'
                            ? 'blue'
                            : 'gray'
                      }
                    >
                      {s.status}
                    </Badge>
                  </Table.Td>
                  <Table.Td>{s.user_id}</Table.Td>
                  <Table.Td>
                    <Group gap='xs' wrap='nowrap'>
                      <Button
                        size='xs'
                        variant='subtle'
                        onClick={e => {
                          e.stopPropagation();
                          openView(Number(s.id));
                        }}
                      >
                        View
                      </Button>
                      <Button
                        size='xs'
                        variant='outline'
                        color='red'
                        onClick={e => {
                          e.stopPropagation();
                          triggerDelete(Number(s.id));
                        }}
                        disabled={deleteStoryMutation.isPending}
                      >
                        {deleteStoryMutation.isPending && storyToDelete === s.id
                          ? 'Deleting…'
                          : 'Delete'}
                      </Button>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </div>

        <Group justify='space-between' p='md'>
          <Text size='sm' c='dimmed'>
            {totalStories > 0 ? (
              <>
                Showing {(currentPage - 1) * pageSize + 1} to{' '}
                {Math.min(currentPage * pageSize, totalStories)} of{' '}
                {totalStories} stories
              </>
            ) : (
              'No stories found'
            )}
          </Text>
          {totalStories > 0 && (
            <Pagination
              total={Math.ceil(totalStories / pageSize)}
              value={currentPage}
              onChange={setCurrentPage}
              size='sm'
            />
          )}
        </Group>
      </Card>

      {/* View Modal */}
      <Modal
        opened={viewModalOpen}
        onClose={() => setViewModalOpen(false)}
        title='View Story'
        size='xl'
      >
        <Stack>
          {!selectedStory ? (
            <Group justify='center' mih={200}>
              <Loader />
            </Group>
          ) : (
            <Tabs defaultValue='reading'>
              <Tabs.List>
                <Tabs.Tab value='reading'>Reading View</Tabs.Tab>
                <Tabs.Tab value='sections'>Sections</Tabs.Tab>
              </Tabs.List>
              <Tabs.Panel value='reading' pt='xs'>
                <StoryReadingView story={selectedStory} isGenerating={false} />
              </Tabs.Panel>
              <Tabs.Panel value='sections' pt='xs'>
                <Stack gap='sm'>
                  <Group>
                    <Select
                      label='Section'
                      placeholder='Select a section'
                      value={String(sectionIndex)}
                      onChange={v => setSectionIndex(parseInt(v || '0'))}
                      data={(selectedStory.sections || []).map((sec, idx) => ({
                        value: String(idx),
                        label: `Section ${sec.section_number}`,
                      }))}
                      style={{ minWidth: 200 }}
                    />
                  </Group>
                  <StorySectionView
                    section={
                      (selectedStory.sections || [])[sectionIndex] || null
                    }
                    sectionWithQuestions={currentSectionWithQuestions || null}
                    sectionIndex={sectionIndex}
                    totalSections={selectedStory.sections?.length || 0}
                    canGenerateToday={false}
                    isGenerating={false}
                    onGenerateNext={() => {}}
                    generationDisabledReason='Generation disabled in admin view'
                    onPrevious={() => setSectionIndex(i => Math.max(0, i - 1))}
                    onNext={() =>
                      setSectionIndex(i =>
                        Math.min(
                          (selectedStory.sections?.length || 1) - 1,
                          i + 1
                        )
                      )
                    }
                    onFirst={() => setSectionIndex(0)}
                    onLast={() =>
                      setSectionIndex(
                        Math.max(0, (selectedStory.sections?.length || 1) - 1)
                      )
                    }
                  />
                </Stack>
              </Tabs.Panel>
            </Tabs>
          )}
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        opened={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title='Delete Story'
        centered
      >
        <Stack>
          <Text size='sm' c='dimmed'>
            This will permanently delete the story and all its sections and
            questions.
          </Text>
          <Group justify='flex-end'>
            <Button variant='outline' onClick={() => setConfirmOpen(false)}>
              Cancel
            </Button>
            <Button
              color='red'
              loading={deleteStoryMutation.isPending}
              onClick={() => {
                setConfirmOpen(false);
                confirmDelete();
              }}
            >
              Delete
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
};

export default StoryExplorerPage;
