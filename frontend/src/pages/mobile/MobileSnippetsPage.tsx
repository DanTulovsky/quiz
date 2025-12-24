import React, {
  useState,
  useRef,
  useCallback,
  useMemo,
  useEffect,
} from 'react';
import { useLocation } from 'react-router-dom';
import {
  Container,
  Title,
  TextInput,
  Select,
  Button,
  Group,
  Stack,
  Text,
  Badge,
  ActionIcon,
  Modal,
  Textarea,
  Loader,
  Center,
  Alert,
  Paper,
  Divider,
  Anchor,
  Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconSearch,
  IconEdit,
  IconTrash,
  IconPlus,
  IconX,
  IconExternalLink,
  IconLanguage,
} from '@tabler/icons-react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../../hooks/useAuth';
import { usePagination } from '../../hooks/usePagination';
import { useTranslation } from '../../contexts/TranslationContext';
import { PaginationControls } from '../../components/PaginationControls';
import {
  useDeleteV1SnippetsId,
  usePutV1SnippetsId,
  usePostV1Snippets,
  useGetV1SettingsLanguages,
  useGetV1SettingsLevels,
  useGetV1Story,
} from '../../api/api';
import { customInstance } from '../../api/axios';

const MobileSnippetsPage: React.FC = () => {
  const { user } = useAuth();
  const location = useLocation();
  const { translateText } = useTranslation();

  // Fetch available languages
  const { data: languages = [] } = useGetV1SettingsLanguages();

  // Fetch user's stories for the dropdown
  const { data: stories = [] } = useGetV1Story();

  // Fetch available levels for the user's language
  const { data: levelsData, isLoading: levelsLoading } = useGetV1SettingsLevels(
    user?.preferred_language
      ? { language: user.preferred_language }
      : undefined,
    {
      query: {
        enabled: !!user?.preferred_language,
      },
    }
  );
  const levels = levelsData?.levels || [];
  const levelDescriptions = levelsData?.level_descriptions || {};

  const [searchQuery, setSearchQuery] = useState('');
  const [activeSearchQuery, setActiveSearchQuery] = useState('');
  const [selectedStoryId, setSelectedStoryId] = useState<string | null>(null);
  const [selectedLevel, setSelectedLevel] = useState<string | null>(null);
  const [selectedSourceLang, setSelectedSourceLang] = useState<string | null>(
    null
  );
  const [translatingSnippetId, setTranslatingSnippetId] = useState<
    number | null
  >(null);
  const [snippetTranslations, setSnippetTranslations] = useState<
    Map<number, string>
  >(new Map());

  // Handle URL search parameters
  useEffect(() => {
    const urlParams = new URLSearchParams(location.search);
    const q = urlParams.get('q');
    if (q) {
      setSearchQuery(q);
      setActiveSearchQuery(q);
    }
  }, [location.search]);
  const [editModalOpened, { open: openEditModal, close: closeEditModal }] =
    useDisclosure(false);
  const [addModalOpened, { open: openAddModal, close: closeAddModal }] =
    useDisclosure(false);
  const [
    deleteModalOpened,
    { open: openDeleteModal, close: closeDeleteModal },
  ] = useDisclosure(false);
  const [snippetToDelete, setSnippetToDelete] = useState<number | null>(null);
  const [editingSnippet, setEditingSnippet] = useState<{
    id: number;
    original_text: string;
    translated_text: string;
    context: string | null;
    source_language: string;
    target_language: string;
    difficulty_level: string | null;
    created_at: string;
  } | null>(null);

  // Create language options for dropdowns
  const languageOptions = useMemo(
    () =>
      languages.map(lang => ({
        value: lang.code,
        label: lang.name,
      })),
    [languages]
  );

  // Create story options for dropdown
  const storyOptions = useMemo(
    () =>
      stories.map(story => ({
        value: String(story.id),
        label: story.title || 'Untitled Story',
      })),
    [stories]
  );

  // Level options from API
  const levelOptions = useMemo(
    () =>
      levels.map(level => ({
        value: level,
        label: levelDescriptions[level]
          ? `${level} - ${levelDescriptions[level]}`
          : level,
      })),
    [levels, levelDescriptions]
  );

  // Calculate dynamic width for dropdowns based on content
  const storyDropdownWidth = useMemo(() => {
    if (storyOptions.length === 0) return 200;
    const maxLength = Math.max(
      ...storyOptions.map(option => option.label.length)
    );
    // Base width + character width (approximately 8px per character) + padding
    return Math.min(Math.max(maxLength * 8 + 40, 200), 350);
  }, [storyOptions]);

  const levelDropdownWidth = useMemo(() => {
    if (levelOptions.length === 0) return 150;
    const maxLength = Math.max(
      ...levelOptions.map(option => option.label.length)
    );
    // Base width + character width (approximately 8px per character) + padding
    return Math.min(Math.max(maxLength * 8 + 40, 150), 250);
  }, [levelOptions]);

  // Add snippet form state
  const [newSnippet, setNewSnippet] = useState({
    original_text: '',
    translated_text: '',
    source_language: 'IT', // Will be updated when languages load
    target_language: 'EN', // Will be updated when languages load
    context: '',
  });

  const [editForm, setEditForm] = useState({
    original_text: '',
    translated_text: '',
    source_language: 'IT', // Will be updated when languages load
    target_language: 'EN', // Will be updated when languages load
    context: '',
  });

  // Update form defaults when languages are loaded
  useEffect(() => {
    if (languageOptions.length > 0) {
      // Set source to first language, target to second if available, otherwise same as source
      const sourceLang = languageOptions[0].value;
      const targetLang =
        languageOptions.length > 1 ? languageOptions[1].value : sourceLang;

      setNewSnippet(prev => ({
        ...prev,
        source_language: sourceLang,
        target_language: targetLang,
      }));
      setEditForm(prev => ({
        ...prev,
        source_language: sourceLang,
        target_language: targetLang,
      }));
    }
  }, [languageOptions]);

  const queryClient = useQueryClient();
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Use pagination hook for snippets
  const {
    data: snippets,
    isLoading: snippetsLoading,
    isFetching: snippetsFetching,
    isError,
    pagination: snippetsPagination,
    goToPage: goToSnippetsPage,
    goToNextPage: goToNextSnippetsPage,
    goToPreviousPage: goToPreviousSnippetsPage,
    reset: resetSnippets,
  } = usePagination(
    activeSearchQuery
      ? [
          '/v1/snippets/search',
          activeSearchQuery,
          selectedStoryId ?? '',
          selectedLevel ?? '',
          selectedSourceLang ?? '',
        ].filter(Boolean) as string[]
      : ['/v1/snippets', selectedStoryId ?? '', selectedLevel ?? '', selectedSourceLang ?? ''].filter(Boolean) as string[],
    async ({ limit, offset }) => {
      if (activeSearchQuery.trim()) {
        // Use search API
        const params: {
          q: string;
          limit: number;
          offset: number;
          story_id?: number;
          level?: string;
          source_lang?: string;
        } = {
          q: activeSearchQuery.trim(),
          limit,
          offset,
        };
        if (selectedStoryId) {
          params.story_id = parseInt(selectedStoryId, 10);
        }
        if (selectedLevel) {
          params.level = selectedLevel;
        }
        if (selectedSourceLang) {
          params.source_lang = selectedSourceLang;
        }
        const responseData = await customInstance({
          url: '/v1/snippets/search',
          method: 'GET',
          params,
        });
        return {
          items: responseData.snippets || [],
          total: responseData.total || 0,
        };
      } else {
        // Use regular snippets API
        const params: {
          limit: number;
          offset: number;
          story_id?: number;
          level?: string;
          source_lang?: string;
        } = {
          limit,
          offset,
        };
        if (selectedStoryId) {
          params.story_id = parseInt(selectedStoryId, 10);
        }
        if (selectedLevel) {
          params.level = selectedLevel;
        }
        if (selectedSourceLang) {
          params.source_lang = selectedSourceLang;
        }
        const responseData = await customInstance({
          url: '/v1/snippets',
          method: 'GET',
          params,
        });
        return {
          items: responseData.snippets || [],
          total: responseData.total || 0,
        };
      }
    },
    {
      initialLimit: 20,
      enableInfiniteScroll: false,
    }
  );

  // Reset pagination when component mounts to ensure fresh data
  useEffect(() => {
    resetSnippets();
  }, []);

  // Reset pagination when filters change
  useEffect(() => {
    resetSnippets();
  }, [selectedStoryId, selectedLevel, resetSnippets]);

  const isLoading = snippetsLoading;
  const isFetching = snippetsFetching;

  const totalCount = snippetsPagination.totalItems;

  // Mutations (use generated hooks directly; avoid creating hooks inside callbacks)
  const deleteSnippetMutation = useDeleteV1SnippetsId(
    {
      mutation: {
        onSuccess: () => {
          resetSnippets();
        },
      },
    },
    queryClient
  );

  const updateSnippetMutation = usePutV1SnippetsId(
    {
      mutation: {
        onSuccess: () => {
          resetSnippets();
          closeEditModal();
        },
      },
    },
    queryClient
  );

  const createSnippetMutation = usePostV1Snippets(
    {
      mutation: {
        onSuccess: () => {
          resetSnippets();
          closeAddModal();
          setNewSnippet({
            original_text: '',
            translated_text: '',
            source_language: 'IT', // Will be updated by useEffect if languages are loaded
            target_language: 'EN', // Will be updated by useEffect if languages are loaded
            context: '',
          });
        },
      },
    },
    queryClient
  );

  // Handle search input change
  const handleSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setSearchQuery(e.target.value);
    },
    []
  );

  // Handle Enter key press to trigger search
  const handleKeyPress = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter' && searchQuery.trim()) {
        setActiveSearchQuery(searchQuery);
        resetSnippets(); // Reset pagination when searching
      }
    },
    [searchQuery, resetSnippets]
  );

  // Clear search
  const handleClearSearch = () => {
    setSearchQuery('');
    setActiveSearchQuery('');
    resetSnippets(); // Reset pagination when clearing search
    // Focus back to search input
    setTimeout(() => {
      searchInputRef.current?.focus();
    }, 0);
  };

  // Handle search button click
  const handleSearch = () => {
    setActiveSearchQuery(searchQuery);
    resetSnippets(); // Reset pagination when searching
  };

  const handleEdit = (snippet: {
    id: number;
    original_text: string;
    translated_text: string;
    context: string | null;
    source_language: string;
    target_language: string;
    difficulty_level: string | null;
    created_at: string;
  }) => {
    setEditingSnippet(snippet);
    setEditForm({
      original_text: snippet.original_text,
      translated_text: snippet.translated_text,
      source_language: snippet.source_language,
      target_language: snippet.target_language,
      context: snippet.context || '',
    });
    openEditModal();
  };

  const handleSaveEdit = () => {
    if (editingSnippet) {
      updateSnippetMutation.mutate({
        id: editingSnippet.id,
        data: {
          original_text: editForm.original_text,
          translated_text: editForm.translated_text,
          source_language: editForm.source_language,
          target_language: editForm.target_language,
          context: editForm.context || null,
        },
      });
    }
  };

  const handleSaveNew = () => {
    createSnippetMutation.mutate({
      data: {
        original_text: newSnippet.original_text,
        translated_text: newSnippet.translated_text,
        source_language: newSnippet.source_language,
        target_language: newSnippet.target_language,
        context: newSnippet.context || null,
      },
    });
  };

  const handleAddNew = () => {
    openAddModal();
  };

  const handleDelete = (id: number) => {
    setSnippetToDelete(id);
    openDeleteModal();
  };

  const confirmDelete = () => {
    if (snippetToDelete) {
      deleteSnippetMutation.mutate({ id: snippetToDelete });
      closeDeleteModal();
      setSnippetToDelete(null);
    }
  };

  // Handle context translation
  const handleTranslateContext = async (snippetId: number, context: string) => {
    setTranslatingSnippetId(snippetId);
    try {
      const result = await translateText(context, 'en'); // Translate to English
      // Store the translation locally for this snippet
      setSnippetTranslations(
        prev => new Map(prev.set(snippetId, result.translatedText))
      );
    } catch (err) {
      console.error('Failed to translate context:', err);
    } finally {
      setTranslatingSnippetId(null);
    }
  };

  const getSnippetLink = useCallback(
    (snippet: {
      question_id?: number;
      story_id?: number;
      section_id?: number;
    }) => {
      if (snippet.question_id) {
        return {
          href: `/m/quiz/${snippet.question_id}`,
          label: 'View Question',
        };
      } else if (snippet.story_id) {
        // If we have both story_id and section_id, link to the specific section
        if (snippet.section_id) {
          return {
            href: `/m/story/${snippet.story_id}/section/${snippet.section_id}`,
            label: 'View Story Section',
          };
        } else {
          // Just story_id, link to the story
          return {
            href: `/m/story/${snippet.story_id}`,
            label: 'View Story',
          };
        }
      }
      return null;
    },
    []
  );

  if (isLoading) {
    return (
      <Container size='lg' py='md'>
        <Center h={200}>
          <Loader size='lg' />
        </Center>
      </Container>
    );
  }

  if (isError) {
    return (
      <Container size='lg' py='md'>
        <Alert color='red' title='Error'>
          Failed to load snippets. Please try again later.
        </Alert>
      </Container>
    );
  }

  return (
    <Container size='lg' py='md' px='xs'>
      <Stack gap='md'>
        {/* Header */}
        <Stack gap='xs'>
          <Group justify='space-between' align='center'>
            <Title order={2}>Snippets</Title>
            <Badge variant='light' color='blue' size='lg'>
              {totalCount}
            </Badge>
          </Group>
          <Text size='sm' c='dimmed'>
            Your saved words and phrases
          </Text>
        </Stack>

        {/* Add New Button */}
        <Button
          leftSection={<IconPlus size={18} />}
          variant='filled'
          onClick={handleAddNew}
          fullWidth
        >
          Add New Snippet
        </Button>

        {/* Search */}
        <Stack gap='xs'>
          <TextInput
            ref={searchInputRef}
            placeholder='Search snippets...'
            leftSection={<IconSearch size={18} />}
            value={searchQuery}
            onChange={handleSearchChange}
            onKeyDown={handleKeyPress}
            disabled={isLoading || isFetching}
            rightSection={
              (searchQuery || activeSearchQuery) && (
                <ActionIcon
                  variant='subtle'
                  onClick={handleClearSearch}
                  size='sm'
                >
                  <IconX size={16} />
                </ActionIcon>
              )
            }
          />
          <Button
            variant='light'
            leftSection={<IconSearch size={16} />}
            onClick={handleSearch}
            disabled={!searchQuery.trim() || isLoading || isFetching}
            fullWidth
          >
            Search
          </Button>
        </Stack>

        <Divider />

        {/* Filters */}
        <Stack gap='xs'>
          <Select
            placeholder='Filter by story'
            data={storyOptions}
            value={selectedStoryId}
            onChange={setSelectedStoryId}
            clearable
            disabled={isLoading || isFetching}
            style={{ width: storyDropdownWidth }}
            searchable
          />
          <Select
            placeholder='Filter by level'
            data={
              levelOptions.length > 0
                ? levelOptions
                : [
                    {
                      value: 'loading',
                      label: 'Loading levels...',
                      disabled: true,
                    },
                  ]
            }
            value={selectedLevel}
            onChange={setSelectedLevel}
            clearable
            disabled={isLoading || isFetching || levelsLoading}
            style={{ width: levelDropdownWidth }}
            searchable
          />
          <Select
            placeholder='Filter by source language'
            data={languageOptions}
            value={selectedSourceLang}
            onChange={setSelectedSourceLang}
            clearable
            disabled={isLoading || isFetching}
            searchable
          />
          {(selectedStoryId || selectedLevel || selectedSourceLang) && (
            <Button
              variant='subtle'
              onClick={() => {
                setSelectedStoryId(null);
                setSelectedLevel(null);
                setSelectedSourceLang(null);
              }}
              fullWidth
            >
              Clear Filters
            </Button>
          )}
        </Stack>

        <Divider />

        {/* Snippets List */}
        {snippets && snippets.length > 0 ? (
          <Stack gap='sm'>
            {snippets.map(snippet => (
              <Paper
                key={snippet.id}
                withBorder
                p='sm'
                radius='md'
                style={{
                  transition: 'all 0.2s',
                }}
                data-allow-translate='true'
              >
                <Stack gap='xs'>
                  <Group
                    justify='space-between'
                    align='flex-start'
                    wrap='nowrap'
                  >
                    <Stack gap={2} style={{ flex: 1, minWidth: 0 }}>
                      <Text
                        size='md'
                        fw={500}
                        style={{ wordBreak: 'break-word' }}
                      >
                        {snippet.original_text}
                      </Text>
                      <Text
                        size='sm'
                        c='blue'
                        style={{ wordBreak: 'break-word' }}
                      >
                        {snippet.translated_text}
                      </Text>
                    </Stack>
                  </Group>

                  {snippet.context && (
                    <Stack gap='xs'>
                      <Group gap='xs' align='flex-start'>
                        <Tooltip
                          label='Translate'
                          position='top'
                          withArrow
                          arrowSize={6}
                        >
                          <ActionIcon
                            size='sm'
                            variant='subtle'
                            color='blue'
                            onClick={() =>
                              handleTranslateContext(
                                snippet.id,
                                snippet.context!
                              )
                            }
                            loading={translatingSnippetId === snippet.id}
                          >
                            <IconLanguage size={14} />
                          </ActionIcon>
                        </Tooltip>
                        <Text
                          size='xs'
                          c='dimmed'
                          style={{
                            fontStyle: 'italic',
                            wordWrap: 'break-word',
                            overflowWrap: 'break-word',
                            whiteSpace: 'normal',
                            flex: 1,
                          }}
                        >
                          "{snippet.context}"
                        </Text>
                      </Group>
                      {/* Show translation if available for this snippet */}
                      {snippetTranslations.has(snippet.id) && (
                        <Text
                          size='xs'
                          c='blue'
                          style={{
                            wordWrap: 'break-word',
                            overflowWrap: 'break-word',
                            whiteSpace: 'normal',
                            paddingLeft: '32px',
                          }}
                        >
                          "{snippetTranslations.get(snippet.id)}"
                        </Text>
                      )}
                    </Stack>
                  )}

                  <Group justify='space-between' align='center' wrap='wrap'>
                    <Group gap='xs' wrap='wrap'>
                      <Badge variant='light' color='gray' size='sm'>
                        {snippet.source_language.toUpperCase()} →{' '}
                        {snippet.target_language.toUpperCase()}
                      </Badge>
                      {snippet.difficulty_level &&
                        snippet.difficulty_level.toLowerCase() !==
                          'unknown' && (
                          <Badge variant='light' color='blue' size='sm'>
                            {snippet.difficulty_level}
                          </Badge>
                        )}
                      {getSnippetLink(snippet) && (
                        <Anchor
                          href={getSnippetLink(snippet)?.href}
                          size='xs'
                          c='blue'
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 4,
                          }}
                        >
                          <IconExternalLink size={12} />
                          {getSnippetLink(snippet)?.label}
                        </Anchor>
                      )}
                    </Group>

                    <Group gap={4}>
                      <ActionIcon
                        variant='light'
                        color='blue'
                        size='sm'
                        onClick={() => handleEdit(snippet)}
                      >
                        <IconEdit size={14} />
                      </ActionIcon>
                      <ActionIcon
                        variant='light'
                        color='red'
                        size='sm'
                        onClick={() => handleDelete(snippet.id)}
                      >
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Group>
                </Stack>
              </Paper>
            ))}

            <PaginationControls
              pagination={snippetsPagination}
              onPageChange={goToSnippetsPage}
              onNext={goToNextSnippetsPage}
              onPrevious={goToPreviousSnippetsPage}
              isLoading={isLoading || isFetching}
              variant='mobile'
            />
          </Stack>
        ) : (
          <Center h={200}>
            <Stack align='center' gap='md'>
              <Text c='dimmed' size='lg' ta='center'>
                {activeSearchQuery
                  ? 'No snippets found matching your search.'
                  : 'No snippets yet'}
              </Text>
              <Text c='dimmed' size='sm' ta='center' px='md'>
                Use the translation popup while reading to save words and
                phrases
              </Text>
            </Stack>
          </Center>
        )}
      </Stack>

      {/* Edit Modal */}
      <Modal
        opened={editModalOpened}
        onClose={closeEditModal}
        title='Edit Snippet'
        size='100%'
        fullScreen
        styles={{
          content: {
            height: '100vh',
            display: 'flex',
            flexDirection: 'column',
          },
          body: {
            flex: 1,
            overflow: 'auto',
            padding: '16px',
          },
        }}
      >
        {editingSnippet && (
          <Stack gap='md'>
            <TextInput
              label='Original Text'
              placeholder='Enter the original text...'
              value={editForm.original_text}
              onChange={event =>
                setEditForm(prev => ({
                  ...prev,
                  original_text: event.target.value,
                }))
              }
              required
            />

            <TextInput
              label='Translation'
              placeholder='Enter the translation...'
              value={editForm.translated_text}
              onChange={event =>
                setEditForm(prev => ({
                  ...prev,
                  translated_text: event.target.value,
                }))
              }
              required
            />

            <Select
              label='Source Language'
              data={languageOptions}
              value={editForm.source_language}
              onChange={value => {
                if (value) {
                  const newSourceLang = value;
                  setEditForm(prev => ({
                    ...prev,
                    source_language: newSourceLang,
                    // If source and target become the same, update target to something else
                    target_language:
                      newSourceLang === prev.target_language &&
                      languageOptions.length > 1
                        ? languageOptions.find(
                            opt => opt.value !== newSourceLang
                          )?.value || prev.target_language
                        : prev.target_language,
                  }));
                }
              }}
              disabled={languageOptions.length === 0}
              error={
                editForm.source_language === editForm.target_language
                  ? 'Source and target languages must be different'
                  : undefined
              }
              clearable={false}
              searchable={true}
            />

            <Select
              label='Target Language'
              data={languageOptions}
              value={editForm.target_language}
              onChange={value => {
                if (value) {
                  const newTargetLang = value;
                  setEditForm(prev => ({
                    ...prev,
                    target_language: newTargetLang,
                    // If source and target become the same, update source to something else
                    source_language:
                      newTargetLang === prev.source_language &&
                      languageOptions.length > 1
                        ? languageOptions.find(
                            opt => opt.value !== newTargetLang
                          )?.value || prev.source_language
                        : prev.source_language,
                  }));
                }
              }}
              disabled={languageOptions.length === 0}
              error={
                editForm.source_language === editForm.target_language
                  ? 'Source and target languages must be different'
                  : undefined
              }
              clearable={false}
              searchable={true}
            />

            <Textarea
              label='Context/Notes'
              placeholder='Add context or notes about this snippet...'
              value={editForm.context}
              onChange={event =>
                setEditForm(prev => ({
                  ...prev,
                  context: event.target.value,
                }))
              }
              minRows={3}
            />

            <Group justify='space-between' mt='md'>
              <Button variant='light' onClick={closeEditModal} fullWidth>
                Cancel
              </Button>
              <Button
                onClick={handleSaveEdit}
                loading={updateSnippetMutation.isPending}
                disabled={
                  !editForm.original_text ||
                  !editForm.translated_text ||
                  editForm.source_language === editForm.target_language
                }
                fullWidth
              >
                Save Changes
              </Button>
            </Group>
          </Stack>
        )}
      </Modal>

      {/* Add New Modal */}
      <Modal
        opened={addModalOpened}
        onClose={closeAddModal}
        title='Add New Snippet'
        size='100%'
        fullScreen
        styles={{
          content: {
            height: '100vh',
            display: 'flex',
            flexDirection: 'column',
          },
          body: {
            flex: 1,
            overflow: 'auto',
            padding: '16px',
          },
        }}
      >
        <Stack gap='md'>
          <TextInput
            label='Original Text'
            placeholder='Enter the original text...'
            value={newSnippet.original_text}
            onChange={event =>
              setNewSnippet(prev => ({
                ...prev,
                original_text: event.target.value,
              }))
            }
            required
          />

          <TextInput
            label='Translation'
            placeholder='Enter the translation...'
            value={newSnippet.translated_text}
            onChange={event =>
              setNewSnippet(prev => ({
                ...prev,
                translated_text: event.target.value,
              }))
            }
            required
          />

          <Select
            label='Source Language'
            data={languageOptions}
            value={newSnippet.source_language}
            onChange={value => {
              if (value) {
                const newSourceLang = value;
                setNewSnippet(prev => ({
                  ...prev,
                  source_language: newSourceLang,
                  // If source and target become the same, update target to something else
                  target_language:
                    newSourceLang === prev.target_language &&
                    languageOptions.length > 1
                      ? languageOptions.find(opt => opt.value !== newSourceLang)
                          ?.value || prev.target_language
                      : prev.target_language,
                }));
              }
            }}
            disabled={languageOptions.length === 0}
            error={
              newSnippet.source_language === newSnippet.target_language
                ? 'Source and target languages must be different'
                : undefined
            }
            clearable={false}
            searchable={true}
          />

          <Select
            label='Target Language'
            data={languageOptions}
            value={newSnippet.target_language}
            onChange={value => {
              if (value) {
                const newTargetLang = value;
                setNewSnippet(prev => ({
                  ...prev,
                  target_language: newTargetLang,
                  // If source and target become the same, update source to something else
                  source_language:
                    newTargetLang === prev.source_language &&
                    languageOptions.length > 1
                      ? languageOptions.find(opt => opt.value !== newTargetLang)
                          ?.value || prev.source_language
                      : prev.source_language,
                }));
              }
            }}
            disabled={languageOptions.length === 0}
            error={
              newSnippet.source_language === newSnippet.target_language
                ? 'Source and target languages must be different'
                : undefined
            }
            clearable={false}
            searchable={true}
          />

          <Textarea
            label='Context/Notes'
            placeholder='Add context or notes about this snippet...'
            value={newSnippet.context}
            onChange={event =>
              setNewSnippet(prev => ({
                ...prev,
                context: event.target.value,
              }))
            }
            minRows={3}
          />

          <Group justify='flex-end' mt='md'>
            <Button variant='light' onClick={closeAddModal} fullWidth>
              Cancel
            </Button>
            <Button
              onClick={handleSaveNew}
              loading={createSnippetMutation.isPending}
              disabled={
                !newSnippet.original_text ||
                !newSnippet.translated_text ||
                newSnippet.source_language === newSnippet.target_language
              }
              fullWidth
            >
              Add Snippet
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        title='Delete Snippet'
        size='sm'
        centered
        zIndex={2000}
      >
        <Stack gap='md'>
          <Text>
            Are you sure you want to delete this snippet? This action cannot be
            undone.
          </Text>

          <Group justify='flex-end'>
            <Button variant='light' onClick={closeDeleteModal}>
              Cancel
            </Button>
            <Button
              color='red'
              onClick={confirmDelete}
              loading={deleteSnippetMutation.isPending}
            >
              Delete
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
};

export default MobileSnippetsPage;
