/* eslint-disable @typescript-eslint/no-explicit-any */
// This file intentionally uses `any` in a few places for flexible table/data handling.

import React, { useState, useEffect, useMemo, useRef } from 'react';
import {
  Container,
  Title,
  Text,
  Button,
  Group,
  Paper,
  Stack,
  Badge,
  Modal,
  Card,
  Table,
  Select,
  TextInput,
  Checkbox,
  MultiSelect,
  Pagination,
  Center,
  Loader,
  Alert,
  SimpleGrid,
  ThemeIcon,
  Textarea,
  Divider,
  Tooltip,
  ActionIcon,
} from '@mantine/core';
import * as TablerIcons from '@tabler/icons-react';

const tablerIconMap = TablerIcons as unknown as Record<
  string,
  React.ComponentType<any>
>;
const IconSearch: React.ComponentType<any> =
  tablerIconMap.IconSearch || (() => null);
const IconFilter: React.ComponentType<any> =
  tablerIconMap.IconFilter || (() => null);
const IconQuestionMark: React.ComponentType<any> =
  tablerIconMap.IconQuestionMark || (() => null);
const IconAlertTriangle: React.ComponentType<any> =
  tablerIconMap.IconAlertTriangle || (() => null);
const IconUsers: React.ComponentType<any> =
  tablerIconMap.IconUsers || tablerIconMap.IconUser || (() => null);
const IconReport: React.ComponentType<any> =
  tablerIconMap.IconReport || tablerIconMap.IconFlag || (() => null);
const IconDatabase: React.ComponentType<any> =
  tablerIconMap.IconDatabase || (() => null);
const IconAlertCircle: React.ComponentType<any> =
  tablerIconMap.IconAlertCircle || (() => null);
const IconInfoCircle: React.ComponentType<any> =
  tablerIconMap.IconInfoCircle || (() => null);
import { useAuth } from '../../hooks/useAuth';
import { Navigate } from 'react-router-dom';
import { notifications } from '@mantine/notifications';
import {
  useAllQuestions,
  useReportedQuestions,
  useUsersPaginated,
  useUpdateQuestion,
  useDeleteQuestion,
  useAssignUsersToQuestion,
  useUnassignUsersFromQuestion,
  useMarkQuestionAsFixed,
  useFixQuestionWithAI,
  useClearUserDataForUser,
  useClearDatabase,
  useClearUserData,
  useUsersForQuestion,
  QuestionWithStats,
} from '../../api/admin';
import { formatDateCreated } from '../../utils/time';
import AIFixModal from '../../components/AIFixModal';
// AXIOS_INSTANCE no longer used; generated API client hooks are used instead
import {
  useGetV1SettingsLanguages,
  useGetV1SettingsLevels,
} from '../../api/api';

// Add this type for the levels API response
interface LevelsApiResponse {
  levels: string[];
  level_descriptions: Record<string, string>;
}

const DataExplorerPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();

  const [editModalOpen, setEditModalOpen] = useState(false);
  const [selectedQuestion, setSelectedQuestion] =
    useState<QuestionWithStats | null>(null);
  const [deleteQuestionConfirmOpen, setDeleteQuestionConfirmOpen] =
    useState(false);
  const [questionToDelete, setQuestionToDelete] =
    useState<QuestionWithStats | null>(null);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [userToDelete, setUserToDelete] = useState<{
    id: number;
    username: string;
  } | null>(null);
  const [clearUserDataConfirmOpen, setClearUserDataConfirmOpen] =
    useState(false);
  const [clearDatabaseConfirmOpen, setClearDatabaseConfirmOpen] =
    useState(false);

  // Pagination and filtering state
  const [currentPage, setCurrentPage] = useState(1);
  const [reportedCurrentPage, setReportedCurrentPage] = useState(1);
  const [questionFilters, setQuestionFilters] = useState({
    search: '',
    type: '',
    status: '',
    language: '',
    level: '',
    user: '',
    id: '',
  });
  const [reportedQuestionFilters, setReportedQuestionFilters] = useState({
    search: '',
    type: '',
    language: '',
    level: '',
  });
  const pageSize = 20;

  // API hooks for languages and levels
  const { data: languagesData } = useGetV1SettingsLanguages();
  const { data: levelsData } = useGetV1SettingsLevels<LevelsApiResponse>({
    language: questionFilters.language || undefined,
  });

  // API hooks
  // Get users for the dropdown (first page only for dropdown)
  const {
    data: usersData,
    isLoading: isLoadingUsers,
    error: usersError,
  } = useUsersPaginated({
    page: 1,
    pageSize: 100, // Get first 100 users for dropdown
  });
  const {
    data: questionsData,
    isLoading: isLoadingQuestions,
    error: questionsError,
    isFetching: isFetchingQuestions,
  } = useAllQuestions(
    currentPage,
    pageSize,
    questionFilters.search || undefined,
    questionFilters.type || undefined,
    questionFilters.status || undefined,
    questionFilters.language || undefined,
    questionFilters.level || undefined,
    questionFilters.user ? parseInt(questionFilters.user) : undefined
  );
  const {
    data: reportedQuestionsData,
    isLoading: isLoadingReportedQuestions,
    error: reportedQuestionsError,
  } = useReportedQuestions(
    reportedCurrentPage,
    pageSize,
    reportedQuestionFilters.search || undefined,
    reportedQuestionFilters.type || undefined,
    reportedQuestionFilters.language || undefined,
    reportedQuestionFilters.level || undefined
  );

  const updateQuestionMutation = useUpdateQuestion();
  const markQuestionAsFixedMutation = useMarkQuestionAsFixed();
  const fixQuestionWithAIMutation = useFixQuestionWithAI();
  const deleteQuestionMutation = useDeleteQuestion();
  const clearUserDataMutation = useClearUserDataForUser();
  const clearDatabaseMutation = useClearDatabase();
  const clearAllUserDataMutation = useClearUserData();

  const users = usersData?.users || [];
  const questionsList: QuestionWithStats[] = useMemo(
    () => (questionsData?.questions as QuestionWithStats[]) || [],
    [questionsData?.questions]
  );
  const reportedQuestions: QuestionWithStats[] =
    (reportedQuestionsData?.questions as QuestionWithStats[]) || [];
  const questions = questionsList;
  const totalQuestions = questionsData?.pagination?.total || 0;
  const totalReportedQuestions = reportedQuestionsData?.pagination?.total || 0;

  // Scroll position tracking
  const scrollRef = useRef<HTMLDivElement>(null);
  const [scrollPosition, setScrollPosition] = useState(0);

  // AI Fix modal state must be declared at top-level (before any early returns)
  const [aiModalOpen, setAIModalOpen] = useState(false);
  const [aiOriginal, setAIOriginal] = useState<Record<string, unknown> | null>(
    null
  );
  const [aiSuggestion, setAISuggestion] = useState<Record<
    string,
    unknown
  > | null>(null);
  const [aiLoading, setAILoading] = useState(false);
  const [aiAdditionalContext, setAIAdditionalContext] = useState<string>('');
  const [aiContextOpen, setAIContextOpen] = useState(false);

  // Extract an error message from unknown error shapes (axios or Error)
  const extractErrorMessage = (e: unknown): string => {
    if (e instanceof Error) return e.message;
    if (e && typeof e === 'object') {
      const maybe = e as Record<string, unknown>;
      const resp = maybe.response as Record<string, unknown> | undefined;
      const data = resp?.data as Record<string, unknown> | undefined;
      if (data && typeof data.error === 'string') return data.error;
    }
    try {
      return String(e);
    } catch {
      return 'Unknown error';
    }
  };

  // Preserve scroll position when data changes
  useEffect(() => {
    if (scrollRef.current && scrollPosition > 0) {
      scrollRef.current.scrollTop = scrollPosition;
    }
  }, [questions, scrollPosition]);

  const isLoading =
    isLoadingUsers || isLoadingQuestions || isLoadingReportedQuestions;
  const error = usersError || questionsError || reportedQuestionsError;

  // Check if user is admin
  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }

  const isAdmin = user.roles?.some(role => role.name === 'admin') || false;
  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  if (isLoading) {
    return (
      <Container size='xl' py='md'>
        <Center style={{ height: '50vh' }}>
          <Loader size='lg' data-testid='loader' />
        </Center>
      </Container>
    );
  }

  if (error) {
    return (
      <Container size='xl' py='md'>
        <Alert icon={<IconAlertTriangle size={16} />} title='Error' color='red'>
          Failed to load data explorer data: {error.message}
        </Alert>
      </Container>
    );
  }

  // Summary statistics
  const totalUsers = users.length;

  const confirmClearUserData = async () => {
    if (!userToDelete) return;

    try {
      await clearUserDataMutation.mutateAsync(userToDelete.id);
      notifications.show({
        title: 'Success',
        message: `User data cleared for ${userToDelete.username}`,
        color: 'green',
      });
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to clear user data',
        color: 'red',
      });
    } finally {
      setDeleteConfirmOpen(false);
      setUserToDelete(null);
    }
  };

  const handleClearAllUserData = async () => {
    try {
      await clearAllUserDataMutation.mutateAsync();
      notifications.show({
        title: 'Success',
        message: 'All user data cleared successfully',
        color: 'green',
      });
      setClearUserDataConfirmOpen(false);
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to clear all user data',
        color: 'red',
      });
    }
  };

  const handleClearDatabase = async () => {
    try {
      await clearDatabaseMutation.mutateAsync();
      notifications.show({
        title: 'Success',
        message: 'Database cleared successfully',
        color: 'green',
      });
      setClearDatabaseConfirmOpen(false);
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to clear database',
        color: 'red',
      });
    }
  };

  const handleMarkQuestionAsFixed = async (questionId: number) => {
    try {
      await markQuestionAsFixedMutation.mutateAsync(questionId);
      notifications.show({
        title: 'Success',
        message: 'Question marked as fixed',
        color: 'green',
      });
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : 'Failed to mark question as fixed';
      notifications.show({
        title: 'Error',
        message: errorMessage,
        color: 'red',
      });
    }
  };

  const handleFixQuestionWithAI = async () => {
    // Open small context modal and let admin enter optional additional context
    setAIAdditionalContext('');
    setAIOriginal(null);
    setAISuggestion(null);
    setAILoading(false);
    setAIContextOpen(true);
  };

  const applyAISuggestion = async () => {
    if (!aiSuggestion || !selectedQuestion) return;
    setAILoading(true);
    try {
      // Construct update payload from suggestion and normalize/clean fields so
      // we don't send nested `content.content` or duplicate top-level keys.
      const suggestionRaw = aiSuggestion as Record<string, unknown>;
      let payloadContent: Record<string, unknown> = {};

      if (
        suggestionRaw['content'] &&
        typeof suggestionRaw['content'] === 'object'
      ) {
        payloadContent = suggestionRaw['content'] as Record<string, unknown>;
        // unwrap nested content.content if present
        if (
          payloadContent['content'] &&
          typeof payloadContent['content'] === 'object'
        ) {
          payloadContent = payloadContent['content'] as Record<string, unknown>;
        }
      } else {
        payloadContent = { ...suggestionRaw };
        // Remove top-level ai metadata if present
        delete payloadContent['change_reason'];
      }

      // Remove duplicate fields that could be included inside content
      delete payloadContent['correct_answer'];
      delete payloadContent['explanation'];

      // Ensure options is an array if missing/null
      if (!Array.isArray(payloadContent['options'])) {
        payloadContent['options'] = [];
      }

      const updateData: {
        content?: Record<string, unknown>;
        correct_answer?: number;
        explanation?: string;
      } = {
        content: payloadContent,
      };

      // correct_answer/explanation should be taken from normalized suggestion
      if (typeof suggestionRaw['correct_answer'] === 'number') {
        updateData.correct_answer = suggestionRaw['correct_answer'] as number;
      }
      if (typeof suggestionRaw['explanation'] === 'string') {
        updateData.explanation = suggestionRaw['explanation'] as string;
      }

      await handleUpdateQuestion(selectedQuestion.id, updateData);

      // Mark as fixed via API (use mutation hook for consistency)
      await markQuestionAsFixedMutation.mutateAsync(selectedQuestion.id);

      notifications.show({
        title: 'Success',
        message: 'AI suggestion applied and question marked fixed',
        color: 'green',
      });
      setAIModalOpen(false);
      setAIOriginal(null);
      setAISuggestion(null);
    } catch (errUnknown) {
      const message = extractErrorMessage(errUnknown);
      notifications.show({ title: 'Error', message, color: 'red' });
    } finally {
      setAILoading(false);
    }
  };

  const handleUpdateQuestion = async (
    questionId: number,
    data: {
      content?: Record<string, unknown>;
      correct_answer?: number;
      explanation?: string;
    },
    opts: { showNotification?: boolean } = { showNotification: true }
  ) => {
    try {
      await updateQuestionMutation.mutateAsync({ questionId, data });
      if (opts.showNotification !== false) {
        notifications.show({
          title: 'Success',
          message: 'Question updated successfully',
          color: 'green',
        });
      }
      setEditModalOpen(false);
      setSelectedQuestion(null);
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to update question',
        color: 'red',
      });
    }
  };

  return (
    <Container size='xl' py='md'>
      <Title order={1} mb='lg'>
        Data Explorer
      </Title>
      <Text color='dimmed' mb='xl'>
        An overview of all users and their question data in the system.
      </Text>

      {/* Summary Statistics */}
      <SimpleGrid cols={{ base: 1, sm: 3 }} mb='xl'>
        <Paper p='md' withBorder>
          <Group>
            <ThemeIcon size='lg' color='blue'>
              <IconUsers size={20} />
            </ThemeIcon>
            <div>
              <Text size='xs' color='dimmed' tt='uppercase' fw={700}>
                Total Users
              </Text>
              <Text size='xl' fw={700}>
                {totalUsers}
              </Text>
            </div>
          </Group>
        </Paper>

        <Paper p='md' withBorder>
          <Group>
            <ThemeIcon size='lg' color='green'>
              <IconQuestionMark size={20} />
            </ThemeIcon>
            <div>
              <Text size='xs' color='dimmed' tt='uppercase' fw={700}>
                Total Questions
              </Text>
              <Text size='xl' fw={700}>
                {totalQuestions}
              </Text>
            </div>
          </Group>
        </Paper>

        <Paper p='md' withBorder>
          <Group>
            <ThemeIcon size='lg' color='red'>
              <IconReport size={20} />
            </ThemeIcon>
            <div>
              <Text size='xs' color='dimmed' tt='uppercase' fw={700}>
                Reported Questions
              </Text>
              <Text size='xl' fw={700}>
                {totalReportedQuestions}
              </Text>
            </div>
          </Group>
        </Paper>
      </SimpleGrid>

      {/* Action Buttons */}
      <Group mb='xl'>
        <Button
          leftSection={<IconUsers size={16} />}
          color='orange'
          onClick={() => setClearUserDataConfirmOpen(true)}
        >
          Clear User Data
        </Button>
        <Button
          leftSection={<IconDatabase size={16} />}
          color='red'
          onClick={() => setClearDatabaseConfirmOpen(true)}
        >
          Clear Database
        </Button>
      </Group>

      {/* Reported Questions Section */}
      <Card mb='xl' withBorder>
        <Card.Section p='md' bg='yellow.0'>
          <Title order={2} size='h3'>
            Reported Questions ({totalReportedQuestions})
          </Title>
          <Text size='sm' color='dimmed'>
            Questions that have been reported as problematic by users.
          </Text>
        </Card.Section>

        {/* Reported Questions Table */}
        {reportedQuestions && reportedQuestions.length > 0 ? (
          <>
            <div style={{ maxHeight: '400px', overflowY: 'auto' }}>
              <Table>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>
                      <div>
                        <div>Question</div>
                        <TextInput
                          size='xs'
                          placeholder='Search questions...'
                          value={reportedQuestionFilters.search}
                          onChange={e =>
                            setReportedQuestionFilters({
                              ...reportedQuestionFilters,
                              search: e.target.value,
                            })
                          }
                          leftSection={<IconSearch size={12} />}
                          style={{ marginTop: '4px' }}
                        />
                      </div>
                    </Table.Th>
                    <Table.Th>
                      <div>
                        <div>Type</div>
                        <Select
                          size='xs'
                          placeholder='All Types'
                          value={reportedQuestionFilters.type}
                          onChange={value =>
                            setReportedQuestionFilters({
                              ...reportedQuestionFilters,
                              type: value || '',
                            })
                          }
                          searchable
                          clearable
                          data={[
                            { value: '', label: 'All Types' },
                            { value: 'vocabulary', label: 'Vocabulary' },
                            { value: 'fill_blank', label: 'Fill in Blank' },
                            {
                              value: 'reading_comprehension',
                              label: 'Reading Comprehension',
                            },
                            { value: 'qa', label: 'Question Answer' },
                          ]}
                          style={{ marginTop: '4px' }}
                        />
                      </div>
                    </Table.Th>
                    <Table.Th>
                      <div>
                        <div>Language</div>
                        <Select
                          size='xs'
                          placeholder='All Languages'
                          value={reportedQuestionFilters.language}
                          onChange={value =>
                            setReportedQuestionFilters({
                              ...reportedQuestionFilters,
                              language: value || '',
                            })
                          }
                          searchable
                          clearable
                          data={[
                            { value: '', label: 'All Languages' },
                            ...(languagesData?.map(lang => ({
                              value: lang,
                              label:
                                lang.charAt(0).toUpperCase() + lang.slice(1),
                            })) || []),
                          ]}
                          style={{ marginTop: '4px' }}
                        />
                      </div>
                    </Table.Th>
                    <Table.Th>
                      <div>
                        <div>Level</div>
                        <Select
                          size='xs'
                          placeholder='All Levels'
                          value={reportedQuestionFilters.level}
                          onChange={value =>
                            setReportedQuestionFilters({
                              ...reportedQuestionFilters,
                              level: value || '',
                            })
                          }
                          searchable
                          clearable
                          data={[
                            { value: '', label: 'All Levels' },
                            ...(levelsData?.levels?.map(level => ({
                              value: level,
                              label:
                                `${level} - ${levelsData.level_descriptions?.[level] || ''}`.trim(),
                            })) || []),
                          ]}
                          style={{ marginTop: '4px' }}
                        />
                      </div>
                    </Table.Th>
                    <Table.Th>
                      <div>
                        <div>Report Info</div>
                      </div>
                    </Table.Th>
                    <Table.Th style={{ textAlign: 'center' }}>
                      <div>
                        <div>Actions</div>
                        <Button
                          size='xs'
                          variant='subtle'
                          onClick={() =>
                            setReportedQuestionFilters({
                              search: '',
                              type: '',
                              language: '',
                              level: '',
                            })
                          }
                          style={{ marginTop: '4px' }}
                        >
                          Clear All
                        </Button>
                      </div>
                    </Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {reportedQuestions
                    .filter(question => question?.id)
                    .map((question: QuestionWithStats) => (
                      <Table.Tr key={question.id}>
                        <Table.Td>
                          <div>
                            <strong>#{question.id}</strong> - {question.type} (
                            {question.language}/{question.level})
                          </div>
                          <div
                            style={{
                              color: '#6c757d',
                              fontSize: '11px',
                              marginTop: '2px',
                            }}
                          >
                            {question.content?.question?.substring(0, 60) ||
                              question.content?.sentence?.substring(0, 60) ||
                              'No text available'}
                            ...
                          </div>
                        </Table.Td>
                        <Table.Td>
                          <Badge
                            color={question.is_reported ? 'red' : 'green'}
                            variant='light'
                            size='sm'
                          >
                            {question.type}
                          </Badge>
                        </Table.Td>
                        <Table.Td>{question.language || 'N/A'}</Table.Td>
                        <Table.Td>{question.level || 'N/A'}</Table.Td>
                        <Table.Td>
                          {question.report_reasons &&
                          question.report_reasons !== 'No reason provided' ? (
                            <Tooltip
                              label={question.report_reasons}
                              multiline
                              w={300}
                            >
                              <ActionIcon
                                size='sm'
                                color='orange'
                                variant='light'
                              >
                                <IconAlertCircle />
                              </ActionIcon>
                            </Tooltip>
                          ) : (
                            <Tooltip label='No report reason provided'>
                              <ActionIcon
                                size='sm'
                                color='gray'
                                variant='light'
                              >
                                <IconInfoCircle />
                              </ActionIcon>
                            </Tooltip>
                          )}
                        </Table.Td>
                        <Table.Td>
                          <Group
                            gap='xs'
                            justify='center'
                            style={{ flexWrap: 'nowrap' }}
                          >
                            <Button
                              size='xs'
                              variant='subtle'
                              color='blue'
                              onClick={() => {
                                // Convert ReportedQuestion to QuestionWithStats for the modal
                                const questionWithStats: QuestionWithStats = {
                                  ...question,
                                  usage_count: question.usage_count || 0,
                                };
                                setSelectedQuestion(questionWithStats);
                                setEditModalOpen(true);
                              }}
                              style={{ flexShrink: 0 }}
                            >
                              Details
                            </Button>
                            <Button
                              size='xs'
                              variant='light'
                              color='green'
                              onClick={() =>
                                handleMarkQuestionAsFixed(question.id)
                              }
                              loading={
                                markQuestionAsFixedMutation?.isPending ?? false
                              }
                              style={{ flexShrink: 0 }}
                            >
                              Fixed
                            </Button>
                            <Button
                              size='xs'
                              variant='light'
                              color='violet'
                              onClick={() => {
                                const questionWithStats: QuestionWithStats = {
                                  ...question,
                                  usage_count: question.usage_count || 0,
                                };
                                setSelectedQuestion(questionWithStats);
                                handleFixQuestionWithAI();
                              }}
                              loading={
                                fixQuestionWithAIMutation?.isPending ?? false
                              }
                              style={{ flexShrink: 0 }}
                            >
                              AI Fix
                            </Button>
                          </Group>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                </Table.Tbody>
              </Table>
            </div>

            {/* Pagination for reported questions */}
            <Group justify='space-between' p='md'>
              <Text size='sm' c='dimmed'>
                {totalReportedQuestions > 0 ? (
                  <>
                    Showing {(reportedCurrentPage - 1) * pageSize + 1} to{' '}
                    {Math.min(
                      reportedCurrentPage * pageSize,
                      totalReportedQuestions
                    )}{' '}
                    of {totalReportedQuestions} reported questions
                  </>
                ) : (
                  'No reported questions found'
                )}
              </Text>
              {totalReportedQuestions > 0 && (
                <Pagination
                  total={Math.ceil(totalReportedQuestions / pageSize)}
                  value={reportedCurrentPage}
                  onChange={setReportedCurrentPage}
                  size='sm'
                />
              )}
            </Group>
          </>
        ) : (
          <Text c='dimmed' ta='center' py='md'>
            No reported questions found.
          </Text>
        )}
      </Card>

      {/* Questions Section */}
      <Card mb='xl' withBorder>
        <Card.Section p='md' bg='blue.0'>
          <Title order={2} size='h3'>
            All Questions ({totalQuestions})
          </Title>
          <Text size='sm' color='dimmed'>
            Browse and manage all questions in the system with filtering and
            pagination.
          </Text>
        </Card.Section>

        {/* Filters Section: keep filters outside the scrollable area so they remain visible */}
        <Card shadow='sm' padding='md' radius='md' withBorder mb='md'>
          <Group justify='space-between' mb='md'>
            <Title order={4}>Filters</Title>
            <Button
              variant='light'
              size='sm'
              leftSection={<IconFilter size={16} />}
              onClick={() =>
                setQuestionFilters({
                  search: '',
                  type: '',
                  status: '',
                  language: '',
                  level: '',
                  user: '',
                  id: '',
                })
              }
            >
              Clear All Filters
            </Button>
          </Group>

          <Group gap='sm' align='center' style={{ flexWrap: 'wrap' }}>
            <TextInput
              placeholder='Search questions...'
              value={questionFilters.search}
              onChange={e =>
                setQuestionFilters({
                  ...questionFilters,
                  search: e.target.value,
                })
              }
              leftSection={<IconSearch size={14} />}
              size='xs'
              style={{ minWidth: 160 }}
            />

            <Select
              placeholder='Type'
              value={questionFilters.type}
              onChange={value =>
                setQuestionFilters({ ...questionFilters, type: value || '' })
              }
              data={[
                { value: '', label: 'All Types' },
                { value: 'vocabulary', label: 'Vocabulary' },
                { value: 'fill_blank', label: 'Fill in Blank' },
                {
                  value: 'reading_comprehension',
                  label: 'Reading Comprehension',
                },
                { value: 'qa', label: 'Question Answer' },
              ]}
              clearable
              size='xs'
              style={{ minWidth: 140 }}
            />

            <Select
              placeholder='Language'
              value={questionFilters.language}
              onChange={value =>
                setQuestionFilters({
                  ...questionFilters,
                  language: value || '',
                })
              }
              data={[
                { value: '', label: 'All Languages' },
                ...(languagesData?.map(lang => ({
                  value: lang,
                  label: lang.charAt(0).toUpperCase() + lang.slice(1),
                })) || []),
              ]}
              clearable
              size='xs'
              style={{ minWidth: 140 }}
            />

            <Select
              placeholder='Level'
              value={questionFilters.level}
              onChange={value =>
                setQuestionFilters({ ...questionFilters, level: value || '' })
              }
              data={[
                { value: '', label: 'All Levels' },
                ...(levelsData?.levels?.map(level => ({
                  value: level,
                  label:
                    `${level} - ${levelsData.level_descriptions?.[level] || ''}`.trim(),
                })) || []),
              ]}
              clearable
              size='xs'
              style={{ minWidth: 160 }}
            />

            <Select
              placeholder='User'
              value={questionFilters.user || null}
              onChange={value =>
                setQuestionFilters({ ...questionFilters, user: value || '' })
              }
              data={
                users
                  .filter((u: any) => u.user && u.user.id)
                  .map((u: any) => ({
                    value: u.user!.id.toString(),
                    label: u.user!.username || `user-${u.user!.id}`,
                  })) || []
              }
              searchable
              clearable
              {...({ nothingFound: 'No users found' } as any)}
              disabled={isLoadingUsers}
              size='xs'
              style={{ minWidth: 180 }}
            />
          </Group>
        </Card>

        {/* Questions Table */}
        <div
          ref={scrollRef}
          style={{
            maxHeight: '600px',
            overflowY: 'auto',
            position: 'relative',
          }}
          onScroll={e => {
            const target = e.target as HTMLDivElement;
            setScrollPosition(target.scrollTop);
          }}
        >
          {isFetchingQuestions && (
            <div
              style={{
                position: 'absolute',
                top: '10px',
                right: '10px',
                zIndex: 10,
                background: 'rgba(255, 255, 255, 0.9)',
                padding: '4px 8px',
                borderRadius: '4px',
                fontSize: '12px',
              }}
            >
              <Loader size='xs' />
            </div>
          )}
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>ID</Table.Th>
                <Table.Th>Type</Table.Th>
                <Table.Th>Created</Table.Th>
                <Table.Th>Question</Table.Th>
                <Table.Th>Language</Table.Th>
                <Table.Th>Level</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Users</Table.Th>
                <Table.Th>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {questions
                .filter(question => question?.id)
                .map((question: QuestionWithStats) => {
                  return (
                    <Table.Tr key={question.id}>
                      <Table.Td>{question.id}</Table.Td>
                      <Table.Td>
                        <Badge
                          color={question.is_reported ? 'red' : 'green'}
                          variant='light'
                          size='sm'
                        >
                          {question.type}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        {formatDateCreated(question.created_at)}
                      </Table.Td>
                      <Table.Td>
                        <Text size='sm' style={{ maxWidth: 200 }}>
                          {question.content.question?.substring(0, 60) ||
                            question.content.sentence?.substring(0, 60) ||
                            'No text available'}
                          {((question.content.question?.length || 0) > 60 ||
                            (question.content.sentence?.length || 0) > 60) &&
                            '...'}
                        </Text>
                      </Table.Td>
                      <Table.Td>{question.language || 'N/A'}</Table.Td>
                      <Table.Td>{question.level || 'N/A'}</Table.Td>
                      <Table.Td>
                        <Badge
                          color={
                            question.status === 'reported' ? 'red' : 'green'
                          }
                          variant='light'
                          size='sm'
                        >
                          {question.status === 'reported'
                            ? 'Reported'
                            : 'Active'}
                        </Badge>
                      </Table.Td>

                      <Table.Td>
                        <QuestionUserCount question={question} />
                      </Table.Td>
                      <Table.Td>
                        <Group gap='xs'>
                          <Button
                            size='xs'
                            variant='subtle'
                            color='blue'
                            onClick={() => {
                              setSelectedQuestion(question);
                              setEditModalOpen(true);
                            }}
                          >
                            Details
                          </Button>
                          <Button
                            size='xs'
                            variant='light'
                            color='red'
                            onClick={() => {
                              setQuestionToDelete(question);
                              setDeleteQuestionConfirmOpen(true);
                            }}
                          >
                            Delete
                          </Button>
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
            </Table.Tbody>
          </Table>
        </div>

        {/* Pagination */}
        <Group justify='space-between' p='md'>
          <Text size='sm' c='dimmed'>
            {totalQuestions > 0 ? (
              <>
                Showing {(currentPage - 1) * pageSize + 1} to{' '}
                {Math.min(currentPage * pageSize, totalQuestions)} of{' '}
                {totalQuestions} questions
              </>
            ) : (
              'No questions found'
            )}
          </Text>
          {totalQuestions > 0 && (
            <Pagination
              total={Math.ceil(totalQuestions / pageSize)}
              value={currentPage}
              onChange={setCurrentPage}
              size='sm'
            />
          )}
        </Group>
      </Card>

      {/* Delete Confirmation Modal */}
      <Modal
        opened={deleteConfirmOpen}
        onClose={() => setDeleteConfirmOpen(false)}
        title='Clear User Data'
      >
        <Alert icon={<IconAlertTriangle size={16} />} color='red' mb='md'>
          Are you sure you want to clear all questions, responses, and stats for
          user "{userToDelete?.username}"? This will NOT delete the user, only
          their activity data. This action cannot be undone.
        </Alert>
        <Group justify='flex-end'>
          <Button variant='subtle' onClick={() => setDeleteConfirmOpen(false)}>
            Cancel
          </Button>
          <Button color='red' onClick={confirmClearUserData}>
            Clear Data
          </Button>
        </Group>
      </Modal>

      {/* Delete Question Confirmation Modal */}
      <Modal
        opened={deleteQuestionConfirmOpen}
        onClose={() => {
          setDeleteQuestionConfirmOpen(false);
          setQuestionToDelete(null);
        }}
        title='Delete Question'
        size='sm'
      >
        <Alert icon={<IconAlertTriangle size={16} />} color='red' mb='md'>
          Are you sure you want to permanently delete question #
          {questionToDelete?.id}?
        </Alert>
        <Group justify='flex-end'>
          <Button
            variant='subtle'
            onClick={() => {
              setDeleteQuestionConfirmOpen(false);
              setQuestionToDelete(null);
            }}
          >
            Cancel
          </Button>
          <Button
            color='red'
            onClick={() => {
              if (!questionToDelete) return;
              deleteQuestionMutation.mutate(questionToDelete.id, {
                onSuccess: () => {
                  notifications.show({
                    title: 'Deleted',
                    message: `Question #${questionToDelete.id} deleted`,
                    color: 'green',
                  });
                },
                onError: (err: unknown) => {
                  const msg = extractErrorMessage(err);
                  notifications.show({
                    title: 'Error',
                    message: msg,
                    color: 'red',
                  });
                },
                onSettled: () => {
                  setDeleteQuestionConfirmOpen(false);
                  setQuestionToDelete(null);
                },
              });
            }}
            loading={deleteQuestionMutation?.isPending ?? false}
          >
            Delete Question
          </Button>
        </Group>
      </Modal>

      {/* Clear User Data Confirmation Modal */}
      <Modal
        opened={clearUserDataConfirmOpen}
        onClose={() => setClearUserDataConfirmOpen(false)}
        title='Clear All User Data'
        size='md'
      >
        <Stack gap='md'>
          <Alert icon={<IconAlertTriangle size={16} />} color='orange'>
            <Text size='sm' fw={500} mb='xs'>
              This will clear ALL user activity data including:
            </Text>
            <Text size='sm' component='ul' mt='xs'>
              <li>All questions and their responses</li>
              <li>All user performance metrics and statistics</li>
              <li>All user progress and learning data</li>
              <li>All question assignments and usage data</li>
            </Text>
            <Text size='sm' fw={500} mt='md' c='red'>
              This action cannot be undone. Users will remain but all their
              activity data will be permanently deleted.
            </Text>
          </Alert>
          <Group justify='flex-end' gap='xs'>
            <Button
              variant='light'
              onClick={() => setClearUserDataConfirmOpen(false)}
            >
              Cancel
            </Button>
            <Button
              color='orange'
              onClick={handleClearAllUserData}
              loading={clearAllUserDataMutation?.isPending ?? false}
            >
              Clear All User Data
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Clear Database Confirmation Modal */}
      <Modal
        opened={clearDatabaseConfirmOpen}
        onClose={() => setClearDatabaseConfirmOpen(false)}
        title='Clear Entire Database'
        size='md'
      >
        <Stack gap='md'>
          <Alert icon={<IconAlertTriangle size={16} />} color='red'>
            <Text size='sm' fw={500} mb='xs'>
              This will completely clear the ENTIRE database including:
            </Text>
            <Text size='sm' component='ul' mt='xs'>
              <li>All users and their accounts</li>
              <li>All questions and their responses</li>
              <li>All performance metrics and statistics</li>
              <li>All user progress and learning data</li>
              <li>All question assignments and usage data</li>
              <li>All system configuration and settings</li>
            </Text>
            <Text size='sm' fw={500} mt='md' c='red'>
              This action cannot be undone. You will need to recreate admin
              users and all data will be permanently deleted.
            </Text>
          </Alert>
          <Group justify='flex-end' gap='xs'>
            <Button
              variant='light'
              onClick={() => setClearDatabaseConfirmOpen(false)}
            >
              Cancel
            </Button>
            <Button
              color='red'
              onClick={handleClearDatabase}
              loading={clearDatabaseMutation?.isPending ?? false}
            >
              Clear Entire Database
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Edit Question Modal */}
      <Modal
        opened={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        title='Edit Question'
        size='xl'
      >
        {selectedQuestion && (
          <QuestionEditForm
            question={selectedQuestion}
            onSave={data => handleUpdateQuestion(selectedQuestion.id, data)}
            onCancel={() => setEditModalOpen(false)}
          />
        )}
      </Modal>
      {/* AI Fix Modal */}
      <AIFixModal
        opened={aiModalOpen}
        original={aiOriginal}
        suggestion={aiSuggestion}
        loading={aiLoading || fixQuestionWithAIMutation.isPending}
        onClose={() => setAIModalOpen(false)}
        onApply={applyAISuggestion}
      />

      {/* Additional small modal to collect admin-provided context before calling AI */}
      <Modal
        opened={aiContextOpen}
        onClose={() => setAIContextOpen(false)}
        title='AI Fix - Additional Context'
        size='md'
      >
        <Stack>
          <Textarea
            placeholder='Optional: provide additional context or instructions for the AI (e.g. what to focus on)'
            value={aiAdditionalContext}
            onChange={e => setAIAdditionalContext(e.target.value)}
            autosize
            minRows={3}
          />
          <Group position='right'>
            <Button variant='subtle' onClick={() => setAIContextOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={async () => {
                // Close context modal and open main AI modal with loader while calling AI
                setAIContextOpen(false);
                setAIModalOpen(true);
                setAIOriginal(null);
                setAISuggestion(null);
                setAILoading(true);
                try {
                  const resp = await fixQuestionWithAIMutation.mutateAsync({
                    questionId: selectedQuestion?.id ?? 0,
                    additionalContext: aiAdditionalContext,
                  });
                  const original = (resp as any)?.original;
                  const suggestion = (resp as any)?.suggestion;
                  setAIOriginal(original as Record<string, unknown> | null);
                  setAISuggestion(suggestion as Record<string, unknown> | null);
                } catch (err) {
                  notifications.show({
                    title: 'Error',
                    message: extractErrorMessage(err),
                    color: 'red',
                  });
                } finally {
                  setAILoading(false);
                }
              }}
              color='violet'
              loading={fixQuestionWithAIMutation.isPending || aiLoading}
            >
              Send to AI
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
};

// Question Edit Form Component

const QuestionUserCount: React.FC<{ question: QuestionWithStats }> = ({
  question,
}) => {
  return (
    <Text size='xs' fw={500}>
      {question.user_count || 0}
    </Text>
  );
};

const QuestionUsersSection: React.FC<{
  questionId: number;
}> = ({ questionId }) => {
  const { data, isLoading, error } = useUsersForQuestion(questionId);
  const { data: allUsersData } = useUsersPaginated({ page: 1, pageSize: 1000 });
  const assignUsersMutation = useAssignUsersToQuestion();
  const unassignUsersMutation = useUnassignUsersFromQuestion();

  if (isLoading) {
    return <Loader size='sm' />;
  }

  if (error) {
    return <Text c='dimmed'>Error loading users: {error.message}</Text>;
  }

  if (!data) {
    return <Text c='dimmed'>No user data available</Text>;
  }

  const { users = [], total_count = 0 } = data;
  const allUsers = allUsersData?.users || [];

  // Create options for the multi-select
  const userOptions = allUsers
    .filter(
      (userData: { user?: { id: number; username: string } }) =>
        userData?.user && userData.user.id
    )
    .map((userData: { user?: { id: number; username: string } }) => ({
      value: userData.user!.id.toString(),
      label: userData.user!.username,
    }));

  // Get currently assigned user IDs
  const assignedUserIds = users
    .filter((user: { id: number; username: string }) => user && user.id)
    .map((user: { id: number; username: string }) => user.id.toString());

  const handleUserAssignmentChange = (selectedUserIds: string[]) => {
    const currentUserIds = new Set(assignedUserIds);
    const newUserIds = new Set(selectedUserIds);

    // Find users to assign (in new but not in current)
    const usersToAssign = selectedUserIds
      .filter((id: string) => !currentUserIds.has(id))
      .map((id: string) => parseInt(id));

    // Find users to unassign (in current but not in new)
    const usersToUnassign = assignedUserIds
      .filter((id: string) => !newUserIds.has(id))
      .map((id: string) => parseInt(id));

    // Perform the mutations
    if (usersToAssign.length > 0) {
      assignUsersMutation.mutate(
        { questionId, userIDs: usersToAssign },
        {
          onSuccess: () => {
            notifications.show({
              title: 'Success',
              message: `Assigned ${usersToAssign.length} user(s) to question`,
              color: 'green',
            });
          },
          onError: () => {
            notifications.show({
              title: 'Error',
              message: 'Failed to assign users to question',
              color: 'red',
            });
          },
        }
      );
    }

    if (usersToUnassign.length > 0) {
      unassignUsersMutation.mutate(
        { questionId, userIDs: usersToUnassign },
        {
          onSuccess: () => {
            notifications.show({
              title: 'Success',
              message: `Unassigned ${usersToUnassign.length} user(s) from question`,
              color: 'green',
            });
          },
          onError: () => {
            notifications.show({
              title: 'Error',
              message: 'Failed to unassign users from question',
              color: 'red',
            });
          },
        }
      );
    }
  };

  return (
    <div>
      <Text size='sm' fw={500} mb='xs'>
        Assigned Users ({total_count})
      </Text>
      <Text size='xs' color='dimmed' mb='xs'>
        User assignments are saved automatically when you make changes
      </Text>
      <MultiSelect
        data={userOptions}
        value={assignedUserIds}
        onChange={handleUserAssignmentChange}
        placeholder='Select users to assign...'
        searchable
        clearable
        disabled={
          assignUsersMutation.isPending || unassignUsersMutation.isPending
        }
        rightSection={
          assignUsersMutation.isPending || unassignUsersMutation.isPending ? (
            <Loader size='xs' />
          ) : null
        }
      />
    </div>
  );
};

const QuestionEditForm: React.FC<{
  question: QuestionWithStats;
  onSave: (data: {
    content?: Record<string, unknown>;
    correct_answer?: number;
    explanation?: string;
  }) => void;
  onCancel: () => void;
}> = ({ question, onSave, onCancel }) => {
  // Use the correct_answer index directly since it's the index of the correct option
  const findCorrectOptionIndex = () => {
    if (typeof question.correct_answer === 'number') {
      return question.correct_answer;
    }

    // Fallback: try to find by text matching if correct_answer is a string
    if (
      typeof question.correct_answer === 'string' &&
      question.content.options
    ) {
      const exactIndex = question.content.options.findIndex(
        option => option === question.correct_answer
      );
      if (exactIndex !== -1) return exactIndex;

      const caseInsensitiveIndex = question.content.options.findIndex(
        option => option.toLowerCase() === question.correct_answer.toLowerCase()
      );
      if (caseInsensitiveIndex !== -1) return caseInsensitiveIndex;
    }

    return 0; // Default to first option if no match found
  };

  const [formData, setFormData] = useState({
    type: question.type,
    question: question.content.question || '',
    sentence: question.content.sentence || '',
    passage: question.content.passage || '',
    options: question.content.options || ['', '', '', ''],
    correct_option: findCorrectOptionIndex(),
    explanation: question.explanation || '',
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const content: Record<string, unknown> = {};

    if (formData.type === 'vocabulary') {
      content.sentence = formData.sentence;
      content.question = formData.question;
    } else if (formData.type === 'reading_comprehension') {
      content.passage = formData.passage;
      content.question = formData.question;
    } else {
      content.question = formData.question;
    }

    content.options = formData.options.filter(opt => opt.trim() !== '');

    onSave({
      content,
      correct_answer: formData.correct_option,
      explanation: formData.explanation,
    });
  };

  return (
    <form onSubmit={handleSubmit}>
      <Stack gap='md'>
        <Select
          label='Question Type'
          value={formData.type}
          onChange={value =>
            setFormData({ ...formData, type: value || 'vocabulary' })
          }
          data={[
            { value: 'vocabulary', label: 'Vocabulary' },
            { value: 'fill_blank', label: 'Fill in the Blank' },
            { value: 'reading_comprehension', label: 'Reading Comprehension' },
            { value: 'qa', label: 'Question & Answer' },
          ]}
        />

        {formData.type === 'vocabulary' && (
          <>
            <Textarea
              label='Sentence (with target word)'
              value={formData.sentence}
              onChange={e =>
                setFormData({ ...formData, sentence: e.target.value })
              }
              rows={4}
            />
            <TextInput
              label='Target Word'
              value={formData.question}
              onChange={e =>
                setFormData({ ...formData, question: e.target.value })
              }
            />
          </>
        )}

        {formData.type === 'reading_comprehension' && (
          <>
            <Textarea
              label='Reading Passage'
              value={formData.passage}
              onChange={e =>
                setFormData({ ...formData, passage: e.target.value })
              }
              rows={4}
            />
            <Textarea
              label='Question about the passage'
              value={formData.question}
              onChange={e =>
                setFormData({ ...formData, question: e.target.value })
              }
              rows={3}
            />
          </>
        )}

        {formData.type !== 'vocabulary' &&
          formData.type !== 'reading_comprehension' && (
            <Textarea
              label='Question'
              value={formData.question}
              onChange={e =>
                setFormData({ ...formData, question: e.target.value })
              }
              rows={3}
            />
          )}

        <div>
          <Text size='sm' fw={500} mb='xs'>
            Answer Options
          </Text>
          {formData.options.map((option, index) => (
            <Group key={index} mb='xs'>
              <Checkbox
                checked={formData.correct_option === index}
                onChange={() =>
                  setFormData({ ...formData, correct_option: index })
                }
              />
              <TextInput
                value={option}
                onChange={e => {
                  const newOptions = [...formData.options];
                  newOptions[index] = e.target.value;
                  setFormData({ ...formData, options: newOptions });
                }}
                placeholder={`Option ${index + 1}`}
                style={{ flex: 1 }}
              />
            </Group>
          ))}
        </div>

        <Textarea
          label='Explanation'
          value={formData.explanation}
          onChange={e =>
            setFormData({ ...formData, explanation: e.target.value })
          }
          rows={3}
        />

        <Divider />

        {/* Report Information Section */}
        {question.report_reasons && (
          <>
            <div>
              <Text size='sm' fw={500} mb='xs'>
                Report Information
              </Text>
              <Paper p='xs' withBorder>
                <Text size='sm' c='dimmed'>
                  <strong>Report Reasons:</strong>
                </Text>
                <Text size='sm' mt='xs'>
                  {question.report_reasons}
                </Text>
              </Paper>
            </div>
            <Divider />
          </>
        )}

        <QuestionUsersSection questionId={question.id} />

        <Group justify='flex-end'>
          <Button variant='subtle' onClick={onCancel}>
            Cancel
          </Button>
          <Button type='submit' color='green'>
            Save Changes
          </Button>
        </Group>
      </Stack>
    </form>
  );
};

export default DataExplorerPage;
