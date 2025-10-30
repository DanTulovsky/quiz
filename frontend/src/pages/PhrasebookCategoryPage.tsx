import React, { useEffect, useState, useMemo } from 'react';
import {
  Container,
  Title,
  Text,
  Stack,
  Button,
  Group,
  Loader,
  Center,
  Alert,
  TextInput,
  Table,
  Paper,
  Badge,
  ActionIcon,
  Tooltip,
  Box,
  UnstyledButton,
} from '@mantine/core';
import { useNavigate, useParams, Link } from 'react-router-dom';
import {
  IconArrowLeft,
  IconChevronLeft,
  IconChevronRight,
  IconAlertCircle,
  IconSearch,
  IconX,
  IconCopy,
  IconDownload,
  IconHash,
  IconPrinter,
  IconChevronUp,
  IconChevronDown,
  IconSelector,
  IconSpeakerphone,
} from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';
import { playTTSOnce } from '../hooks/useTTS';
import { useTheme } from '../contexts/ThemeContext';
import { fontScaleMap } from '../theme/theme';
import { defaultVoiceForLanguage } from '../utils/tts';
import { getTermForLanguage } from '../utils/phrasebook';
import { ensureLanguagesLoaded } from '../utils/locale';
import {
  loadCategoryData,
  getCategoryInfo,
  getNextCategory,
  getPreviousCategory,
  type PhrasebookData,
  type CategoryInfo,
} from '../utils/phrasebook';

type SortField = 'term' | 'translation' | 'default';
type SortDirection = 'asc' | 'desc';

const PhrasebookCategoryPage = () => {
  const { category } = useParams<{ category: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const { fontSize } = useTheme();

  const [data, setData] = useState<PhrasebookData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<SortField>('default');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

  const [categoryInfo, setCategoryInfo] = useState<CategoryInfo | null>(null);
  const [nextCategory, setNextCategory] = useState<CategoryInfo | null>(null);
  const [previousCategory, setPreviousCategory] = useState<CategoryInfo | null>(
    null
  );

  // Translation (English) for display alongside the term

  // Helper function to get the translation (English) for display
  const getTranslation = (word: PhrasebookWord): string => {
    return word.term || word.en || '';
  };

  // Process sections: filter by search and sort within each section
  const processedSections = useMemo(() => {
    if (!data || !user?.preferred_language) return [];

    const query = searchQuery.toLowerCase();
    const languageCode = user.preferred_language;

    return data.sections
      .map(section => {
        // Add original index and language-specific terms
        const wordsWithIndex = section.words.map((word, index) => ({
          ...word,
          originalIndex: index,
          displayTerm: getTermForLanguage(word as any, languageCode),
          translation: getTranslation(word),
        }));

        // Filter words by search query
        let filteredWords = wordsWithIndex;
        if (query) {
          filteredWords = wordsWithIndex.filter(
            word =>
              word.displayTerm.toLowerCase().includes(query) ||
              word.translation.toLowerCase().includes(query) ||
              (word.note && word.note.toLowerCase().includes(query))
          );
        }

        // Sort words within section
        const sortedWords = [...filteredWords].sort((a, b) => {
          let compareResult = 0;

          switch (sortField) {
            case 'default':
              compareResult = a.originalIndex - b.originalIndex;
              break;
            case 'term':
              compareResult = a.displayTerm.localeCompare(b.displayTerm);
              break;
            case 'translation':
              compareResult = a.translation.localeCompare(b.translation);
              break;
          }

          return sortDirection === 'asc' ? compareResult : -compareResult;
        });

        return {
          ...section,
          words: sortedWords,
        };
      })
      .filter(section => section.words.length > 0); // Remove empty sections
  }, [data, searchQuery, sortField, sortDirection]);

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      // Toggle direction if same field
      setSortDirection(prev => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      // New field, default to ascending
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      console.error('Failed to copy text: ', err);
    }
  };

  const playWordTTS = async (text: string) => {
    try {
      // Determine the best voice: user preference -> default for user's language -> fallback to 'echo'
      let preferredVoice: string | undefined;
      try {
        const saved = (
          user?.learning_preferences as unknown as
            | { tts_voice?: string }
            | undefined
        )?.tts_voice;
        if (saved && saved.trim()) {
          preferredVoice = saved.trim();
        }
        preferredVoice =
          preferredVoice || defaultVoiceForLanguage(user?.preferred_language);
      } catch {
        preferredVoice = undefined;
      }
      // Ensure we always pass a sensible fallback voice to the TTS hook
      const finalVoice =
        preferredVoice ??
        defaultVoiceForLanguage(user?.preferred_language) ??
        'echo';

      console.log('Playing TTS:', {
        text,
        finalVoice,
        userLanguage: user?.preferred_language,
      });
      await playTTSOnce(text, finalVoice);
    } catch {
      // Error handling is already done in playTTSOnce
    }
  };

  const copyAllToClipboard = async () => {
    const text = processedSections
      .map(section =>
        section.words
          .map(word => `${word.displayTerm}\t${word.translation}`)
          .join('\n')
      )
      .join('\n');
    await copyToClipboard(text);
  };

  const exportToCSV = () => {
    const csv = [
      ['Term', 'Translation', 'Section', 'Note'].join(','),
      ...processedSections.flatMap(section =>
        section.words.map(word =>
          [
            `"${word.displayTerm}"`,
            `"${word.translation}"`,
            `"${section.title}"`,
            `"${word.note || ''}"`,
          ].join(',')
        )
      ),
    ].join('\n');

    const blob = new Blob([csv], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${category}-${user?.preferred_language || 'phrasebook'}.csv`;
    a.click();
    window.URL.revokeObjectURL(url);
  };

  const handlePrint = () => {
    window.print();
  };

  const getSortIcon = (field: SortField) => {
    if (sortField !== field) {
      return <IconSelector size='0.9rem' />;
    }
    return sortDirection === 'asc' ? (
      <IconChevronUp size='0.9rem' />
    ) : (
      <IconChevronDown size='0.9rem' />
    );
  };

  useEffect(() => {
    const fetchData = async () => {
      if (!category || !user?.preferred_language) {
        setError('Missing category or user language');
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        // Ensure runtime language map is loaded for normalization
        await ensureLanguagesLoaded();
        // Load category info and navigation
        const [
          categoryInfoData,
          nextCategoryData,
          previousCategoryData,
          categoryData,
        ] = await Promise.all([
          getCategoryInfo(category),
          getNextCategory(category),
          getPreviousCategory(category),
          loadCategoryData(category),
        ]);

        setCategoryInfo(categoryInfoData || null);
        setNextCategory(nextCategoryData);
        setPreviousCategory(previousCategoryData);

        if (categoryData) {
          setData(categoryData);
        } else {
          setError(`Failed to load data for ${category}`);
        }
      } catch (err) {
        setError(
          err instanceof Error ? err.message : 'Failed to load category data'
        );
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [category, user?.preferred_language]);

  if (loading) {
    return (
      <Center h='50vh'>
        <Loader size='lg' />
      </Center>
    );
  }

  if (error || !categoryInfo) {
    return (
      <Container size='lg' py='xl'>
        <Alert icon={<IconAlertCircle size='1rem' />} title='Error' color='red'>
          {error || 'Category not found'}
        </Alert>
        <Button
          component={Link}
          to='/phrasebook'
          leftSection={<IconArrowLeft size='1rem' />}
          mt='md'
        >
          Back to Phrasebook
        </Button>
      </Container>
    );
  }

  return (
    <>
      <Container size='xl' py='xl' className='screen-only'>
        <Stack gap='xl'>
          {/* Sticky Header */}
          <Box
            className='phrasebook-sticky-header'
            style={{
              position: 'sticky',
              zIndex: 100,
              backgroundColor: 'var(--mantine-color-body)',
              paddingTop: 'var(--mantine-spacing-sm)',
              paddingBottom: 'var(--mantine-spacing-sm)',
              borderBottom: '1px solid var(--mantine-color-gray-3)',
              marginBottom: 'var(--mantine-spacing-sm)',
            }}
          >
            <Stack gap='sm'>
              {/* Header */}
              <div>
                <Button
                  component={Link}
                  to='/phrasebook'
                  leftSection={<IconArrowLeft size='1rem' />}
                  variant='subtle'
                  size='sm'
                  mb='xs'
                >
                  Back to Phrasebook
                </Button>
                <Group gap='sm' mb='xs'>
                  <Text size='2rem' style={{ lineHeight: 1 }}>
                    {categoryInfo.emoji}
                  </Text>
                  <Title order={2}>{categoryInfo.name}</Title>
                </Group>
              </div>

              {/* Navigation and Toolbar */}
              <Group justify='space-between' align='center'>
                {/* Left: Previous Category */}
                {previousCategory ? (
                  <Button
                    variant='light'
                    leftSection={<IconChevronLeft size='0.9rem' />}
                    onClick={() =>
                      navigate(`/phrasebook/${previousCategory.id}`)
                    }
                    size='sm'
                  >
                    {previousCategory.name}
                  </Button>
                ) : (
                  <div />
                )}

                {/* Center: Search and Actions */}
                <Group gap='xs'>
                  <TextInput
                    placeholder='Search terms...'
                    leftSection={<IconSearch size='0.9rem' />}
                    rightSection={
                      searchQuery ? (
                        <ActionIcon
                          variant='subtle'
                          onClick={() => setSearchQuery('')}
                          size='sm'
                        >
                          <IconX size='0.8rem' />
                        </ActionIcon>
                      ) : null
                    }
                    value={searchQuery}
                    onChange={e => setSearchQuery(e.currentTarget.value)}
                    size='sm'
                    style={{ width: 300 }}
                  />
                  <Tooltip label='Copy to clipboard'>
                    <ActionIcon
                      variant='default'
                      size='sm'
                      aria-label='Copy to clipboard'
                      onClick={copyAllToClipboard}
                    >
                      <IconCopy size='0.9rem' />
                    </ActionIcon>
                  </Tooltip>
                  <Tooltip label='Download as CSV'>
                    <ActionIcon
                      variant='default'
                      size='sm'
                      aria-label='Download as CSV'
                      onClick={exportToCSV}
                    >
                      <IconDownload size='0.9rem' />
                    </ActionIcon>
                  </Tooltip>
                  <Tooltip label='Print'>
                    <ActionIcon
                      variant='default'
                      size='sm'
                      aria-label='Print'
                      onClick={handlePrint}
                    >
                      <IconPrinter size='0.9rem' />
                    </ActionIcon>
                  </Tooltip>
                </Group>

                {/* Right: Next Category */}
                {nextCategory ? (
                  <Button
                    variant='light'
                    rightSection={<IconChevronRight size='0.9rem' />}
                    onClick={() => navigate(`/phrasebook/${nextCategory.id}`)}
                    size='sm'
                  >
                    {nextCategory.name}
                  </Button>
                ) : (
                  <div />
                )}
              </Group>
            </Stack>
          </Box>

          {/* Tables by Section */}
          <Stack gap='xl'>
            {processedSections.length === 0 ? (
              <Paper withBorder p='xl'>
                <Center>
                  <Text c='dimmed'>No terms found</Text>
                </Center>
              </Paper>
            ) : (
              processedSections.map((section, sectionIndex) => (
                <div key={sectionIndex}>
                  <Group gap='sm' mb='md'>
                    <Title order={3}>{section.title}</Title>
                    <Badge size='lg' variant='light'>
                      {section.words.length}
                    </Badge>
                  </Group>

                  <Paper withBorder>
                    <Table highlightOnHover striped>
                      <Table.Thead>
                        <Table.Tr>
                          <Table.Th style={{ width: '40%' }}>
                            <Group gap='xs'>
                              <UnstyledButton
                                onClick={() => handleSort('term')}
                              >
                                <Group gap='xs'>
                                  <Text fw={600} size='sm'>
                                    Term
                                  </Text>
                                  {getSortIcon('term')}
                                </Group>
                              </UnstyledButton>
                              <Tooltip label='Sort by original order'>
                                <ActionIcon
                                  variant='subtle'
                                  size='sm'
                                  onClick={() => handleSort('default')}
                                  color={
                                    sortField === 'default' ? 'blue' : 'gray'
                                  }
                                >
                                  <IconHash size='0.8rem' />
                                </ActionIcon>
                              </Tooltip>
                            </Group>
                          </Table.Th>
                          <Table.Th style={{ width: '40%' }}>
                            <UnstyledButton
                              onClick={() => handleSort('translation')}
                            >
                              <Group gap='xs'>
                                <Text fw={600} size='sm'>
                                  English
                                </Text>
                                {getSortIcon('translation')}
                              </Group>
                            </UnstyledButton>
                          </Table.Th>
                          <Table.Th
                            style={{ width: '20%', textAlign: 'center' }}
                          >
                            <Text fw={600} size='sm'>
                              Actions
                            </Text>
                          </Table.Th>
                        </Table.Tr>
                      </Table.Thead>
                      <Table.Tbody>
                        {section.words.map((word, index) => {
                          const icon = word.icon ? (
                            <Box
                              style={{
                                fontSize: `${24 * fontScaleMap[fontSize]}px`,
                                lineHeight: 1,
                              }}
                            >
                              {word.icon}
                            </Box>
                          ) : null;
                          return (
                            <Table.Tr key={index}>
                              <Table.Td>
                                <Group gap='sm' wrap='nowrap'>
                                  {icon && <Box>{icon}</Box>}
                                  <Box style={{ flex: 1 }}>
                                    <Text fw={500}>{word.displayTerm}</Text>
                                    {word.note && (
                                      <Text size='xs' c='dimmed' fs='italic'>
                                        {word.note}
                                      </Text>
                                    )}
                                  </Box>
                                </Group>
                              </Table.Td>
                              <Table.Td>
                                <Text>{word.translation}</Text>
                              </Table.Td>
                              <Table.Td>
                                <Center>
                                  <Group gap='xs'>
                                    <Tooltip label='Copy term'>
                                      <ActionIcon
                                        variant='subtle'
                                        size='sm'
                                        onClick={() =>
                                          copyToClipboard(word.displayTerm)
                                        }
                                      >
                                        <IconCopy size='0.8rem' />
                                      </ActionIcon>
                                    </Tooltip>
                                    <Tooltip label='Pronounce word'>
                                      <ActionIcon
                                        variant='subtle'
                                        size='sm'
                                        onClick={() =>
                                          playWordTTS(word.displayTerm)
                                        }
                                      >
                                        <IconSpeakerphone size='0.8rem' />
                                      </ActionIcon>
                                    </Tooltip>
                                  </Group>
                                </Center>
                              </Table.Td>
                            </Table.Tr>
                          );
                        })}
                      </Table.Tbody>
                    </Table>
                  </Paper>
                </div>
              ))
            )}
          </Stack>
        </Stack>
      </Container>

      {/* Print-friendly version */}
      <Box className='print-only'>
        <Title order={1} mb='xs'>
          {categoryInfo.name} - {user?.preferred_language}
        </Title>
        <Text size='sm' c='dimmed' mb='xl'>
          Generated from Quiz App Phrasebook
        </Text>

        {processedSections.map((section, sectionIndex) => (
          <div key={sectionIndex} style={{ marginBottom: '2rem' }}>
            <Title order={2} mb='md'>
              {section.title}
            </Title>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Term</Table.Th>
                  <Table.Th>English</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {section.words.map((word, index) => {
                  const icon = word.icon ? (
                    <Box
                      style={{
                        fontSize: `${24 * fontScaleMap[fontSize]}px`,
                        lineHeight: 1,
                      }}
                    >
                      {word.icon}
                    </Box>
                  ) : null;
                  return (
                    <Table.Tr key={index}>
                      <Table.Td>
                        <Group gap='sm' wrap='nowrap'>
                          {icon && <Box>{icon}</Box>}
                          <Box>
                            {word.displayTerm}
                            {word.note && (
                              <>
                                <br />
                                <Text size='xs' c='dimmed' fs='italic'>
                                  {word.note}
                                </Text>
                              </>
                            )}
                          </Box>
                        </Group>
                      </Table.Td>
                      <Table.Td>{word.translation}</Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </div>
        ))}
      </Box>

      <style>{`
        @media screen {
          .print-only {
            display: none;
          }
        }

        @media print {
          .screen-only {
            display: none !important;
          }
          .print-only {
            display: block;
            padding: 20px;
          }
          table {
            page-break-inside: auto;
          }
          tr {
            page-break-inside: avoid;
            page-break-after: auto;
          }
          thead {
            display: table-header-group;
          }
        }
      `}</style>
    </>
  );
};

export default PhrasebookCategoryPage;
