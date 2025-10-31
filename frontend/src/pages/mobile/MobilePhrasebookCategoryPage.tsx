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
  Select,
  Box,
  ActionIcon,
  Divider,
  Paper,
  Badge,
} from '@mantine/core';
import { useNavigate, useParams } from 'react-router-dom';
import {
  IconArrowLeft,
  IconChevronLeft,
  IconChevronRight,
  IconAlertCircle,
  IconSearch,
  IconX,
  IconCopy,
  IconVolume,
} from '@tabler/icons-react';
import { useTheme } from '../../contexts/ThemeContext';
import { fontScaleMap } from '../../theme/theme';
import { useAuth } from '../../hooks/useAuth';
import { playTTSOnce } from '../../hooks/useTTS';
import { defaultVoiceForLanguage } from '../../utils/tts';
import { getTermForLanguage } from '../../utils/phrasebook';
import { ensureLanguagesLoaded } from '../../utils/locale';
import {
  loadCategoryData,
  getCategoryInfo,
  getNextCategory,
  getPreviousCategory,
  type PhrasebookData,
  type CategoryInfo,
} from '../../utils/phrasebook';

type PhrasebookWord = {
  term?: string;
  en?: string;
  it?: string;
  fr?: string;
  de?: string;
  ru?: string;
  ja?: string;
  zh?: string;
  [key: string]: string | undefined;
  icon?: string;
  note?: string;
};

const MobilePhrasebookCategoryPage = () => {
  const { category } = useParams<{ category: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();
  const { fontSize } = useTheme();
  const [data, setData] = useState<PhrasebookData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedSection, setSelectedSection] = useState<string>('all');

  const [categoryInfo, setCategoryInfo] = useState<CategoryInfo | null>(null);
  const [nextCategory, setNextCategory] = useState<CategoryInfo | null>(null);
  const [previousCategory, setPreviousCategory] = useState<CategoryInfo | null>(
    null
  );

  // Term resolution is handled by shared helper

  // Helper function to get the translation (English) for display
  const getTranslation = (word: PhrasebookWord): string => {
    return word.term || word.en || '';
  };

  // Process sections: filter by search and section selection
  const processedSections = useMemo(() => {
    if (!data || !user?.preferred_language) return [];

    const query = searchQuery.toLowerCase();
    const languageCode = user.preferred_language;

    let sections = data.sections;

    // Filter by selected section
    if (selectedSection !== 'all') {
      sections = sections.filter(
        (_, index) => index.toString() === selectedSection
      );
    }

    return sections
      .map(section => {
        // Add original index and language-specific terms
        const wordsWithData = section.words.map((word, index) => ({
          ...word,
          originalIndex: index,
          displayTerm: getTermForLanguage(word as PhrasebookWord, languageCode),
          translation: getTranslation(word),
        }));

        // Filter words by search query
        let filteredWords = wordsWithData;
        if (query) {
          filteredWords = wordsWithData.filter(
            word =>
              word.displayTerm.toLowerCase().includes(query) ||
              word.translation.toLowerCase().includes(query) ||
              (word.note && word.note.toLowerCase().includes(query))
          );
        }

        return {
          ...section,
          words: filteredWords,
        };
      })
      .filter(section => section.words.length > 0); // Remove empty sections
  }, [data, searchQuery, selectedSection, user?.preferred_language]);

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

      await playTTSOnce(text, finalVoice);
    } catch {
      // Error handling is already done in playTTSOnce
    }
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

  // Create section options for the dropdown
  const sectionOptions = useMemo(() => {
    if (!data) return [];
    return [
      { value: 'all', label: 'All Sections' },
      ...data.sections.map((section, index) => ({
        value: index.toString(),
        label: section.title,
      })),
    ];
  }, [data]);

  if (loading) {
    return (
      <Center h='50vh'>
        <Loader size='lg' />
      </Center>
    );
  }

  if (error || !categoryInfo) {
    return (
      <Container size='lg' py='md' px='xs'>
        <Alert icon={<IconAlertCircle size='1rem' />} title='Error' color='red'>
          {error || 'Category not found'}
        </Alert>
        <Button
          onClick={() => navigate('/m/phrasebook')}
          leftSection={<IconArrowLeft size='1rem' />}
          mt='md'
          fullWidth
        >
          Back to Phrasebook
        </Button>
      </Container>
    );
  }

  return (
    <Container size='lg' py='md' px='xs'>
      <Stack gap='md'>
        {/* Header */}
        <Stack gap='sm'>
          <Button
            onClick={() => navigate('/m/phrasebook')}
            leftSection={<IconArrowLeft size={18} />}
            variant='subtle'
            size='sm'
            px={0}
          >
            Back to Phrasebook
          </Button>
          <Group gap='sm'>
            <Text size='2rem' style={{ lineHeight: 1 }}>
              {categoryInfo.emoji}
            </Text>
            <Title order={3}>{categoryInfo.name}</Title>
          </Group>
        </Stack>

        {/* Search */}
        <TextInput
          placeholder='Search terms...'
          leftSection={<IconSearch size={18} />}
          rightSection={
            searchQuery ? (
              <ActionIcon
                variant='subtle'
                onClick={() => setSearchQuery('')}
                size='sm'
              >
                <IconX size={16} />
              </ActionIcon>
            ) : null
          }
          value={searchQuery}
          onChange={e => setSearchQuery(e.currentTarget.value)}
        />

        {/* Section Filter */}
        {sectionOptions.length > 1 && (
          <Select
            label='Section'
            placeholder='Select section'
            data={sectionOptions}
            value={selectedSection}
            onChange={value => setSelectedSection(value || 'all')}
            clearable={false}
          />
        )}

        <Divider />

        {/* Phrases List */}
        <Stack gap='sm'>
          {processedSections.length === 0 ? (
            <Paper withBorder p='xl'>
              <Center>
                <Text c='dimmed'>No terms found</Text>
              </Center>
            </Paper>
          ) : (
            processedSections.map((section, sectionIndex) => (
              <Stack key={sectionIndex} gap='sm'>
                {/* Section Header (only show if showing all sections) */}
                {selectedSection === 'all' && (
                  <Group gap='xs' mt={sectionIndex > 0 ? 'md' : 0}>
                    <Title order={5}>{section.title}</Title>
                    <Badge size='sm' variant='light'>
                      {section.words.length}
                    </Badge>
                  </Group>
                )}

                {/* Words in Section */}
                {section.words.map((word, index) => (
                  <Paper
                    key={index}
                    withBorder
                    p='sm'
                    radius='md'
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                    }}
                    data-allow-translate='true'
                  >
                    {/* Icon */}
                    {word.icon && (
                      <Box
                        style={{
                          fontSize: `${20 * fontScaleMap[fontSize]}px`,
                          lineHeight: 1,
                        }}
                      >
                        {word.icon}
                      </Box>
                    )}

                    {/* Text Content */}
                    <Box style={{ flex: 1, minWidth: 0 }}>
                      <Text size='md' fw={500} style={{ lineHeight: 1.3 }}>
                        {word.displayTerm}
                      </Text>
                      <Text size='sm' c='blue' style={{ lineHeight: 1.3 }}>
                        {word.translation}
                      </Text>
                      {word.note && (
                        <Text
                          size='xs'
                          c='dimmed'
                          fs='italic'
                          style={{ lineHeight: 1.3 }}
                          mt={2}
                        >
                          {word.note}
                        </Text>
                      )}
                    </Box>

                    {/* Action Buttons */}
                    <Group gap={4} wrap='nowrap' style={{ flexShrink: 0 }}>
                      <ActionIcon
                        variant='subtle'
                        size='lg'
                        onClick={() => playWordTTS(word.displayTerm)}
                        aria-label='Pronounce word'
                      >
                        <IconVolume size={20} />
                      </ActionIcon>
                      <ActionIcon
                        variant='subtle'
                        size='lg'
                        onClick={() => copyToClipboard(word.displayTerm)}
                        aria-label='Copy term'
                      >
                        <IconCopy size={20} />
                      </ActionIcon>
                    </Group>
                  </Paper>
                ))}
              </Stack>
            ))
          )}
        </Stack>

        {/* Navigation */}
        <Divider mt='md' />
        <Group justify='space-between' align='center'>
          {previousCategory ? (
            <Button
              variant='light'
              leftSection={<IconChevronLeft size={18} />}
              onClick={() => navigate(`/m/phrasebook/${previousCategory.id}`)}
              size='sm'
              style={{ flex: 1 }}
            >
              {previousCategory.name}
            </Button>
          ) : (
            <div style={{ flex: 1 }} />
          )}

          {nextCategory ? (
            <Button
              variant='light'
              rightSection={<IconChevronRight size={18} />}
              onClick={() => navigate(`/m/phrasebook/${nextCategory.id}`)}
              size='sm'
              style={{ flex: 1 }}
            >
              {nextCategory.name}
            </Button>
          ) : (
            <div style={{ flex: 1 }} />
          )}
        </Group>
      </Stack>
    </Container>
  );
};

export default MobilePhrasebookCategoryPage;
