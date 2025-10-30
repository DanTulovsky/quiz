import { useState, useEffect } from 'react';
import {
  Container,
  Title,
  Text,
  Select,
  Group,
  Stack,
  Badge,
  Alert,
  Loader,
  Center,
  ActionIcon,
  Accordion,
  Box,
  Paper,
  Button,
} from '@mantine/core';
import {
  IconBook,
  IconVolume,
  IconChevronDown,
  IconChevronUp,
} from '@tabler/icons-react';
import {
  loadVerbConjugations,
  loadVerbConjugation,
  VerbConjugation,
} from '../../utils/verbConjugations';
import { HoverTranslation } from '../../components/HoverTranslation';
import { useAuth } from '../../hooks/useAuth';
import { playTTSOnce } from '../../hooks/useTTS';
import { useGetV1SettingsLanguages } from '../../api/api';

export default function MobileVerbConjugationPage() {
  const { user } = useAuth();
  const [availableVerbs, setAvailableVerbs] = useState<VerbConjugation[]>([]);
  const [selectedVerb, setSelectedVerb] = useState<string>('');
  const [verbData, setVerbData] = useState<VerbConjugation | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [accordionValue, setAccordionValue] = useState<
    string | string[] | null
  >(null);
  const [isAllExpanded, setIsAllExpanded] = useState(false);

  // Use React Query to fetch languages (with proper caching)
  const { data: languages } = useGetV1SettingsLanguages();

  // Convert user's preferred language to language code using server data
  const getLanguageCode = (languageName: string): string => {
    const languageInfo = languages?.find(
      lang => lang.name.toLowerCase() === languageName.toLowerCase()
    );
    return languageInfo?.code || 'it'; // Default to Italian
  };

  // Get user's preferred language code
  const userLanguage = user?.preferred_language
    ? getLanguageCode(user.preferred_language)
    : 'it';

  // Load available verbs for the user's language on component mount
  useEffect(() => {
    // Don't load verbs until languages array is populated (unless user has no preferred language)
    // This prevents loading default 'it' verbs before we know the actual language code
    if (!user?.preferred_language || (languages && languages.length > 0)) {
      const loadVerbs = async () => {
        setLoading(true);
        setError(null);
        try {
          const data = await loadVerbConjugations(userLanguage);
          if (data && data.verbs) {
            setAvailableVerbs(data.verbs);
            if (data.verbs.length > 0) {
              setSelectedVerb(data.verbs[0].infinitive);
            }
          }
        } catch (err) {
          setError('Failed to load available verbs');
          console.error('Error loading verbs:', err);
        } finally {
          setLoading(false);
        }
      };

      loadVerbs();
    }
  }, [userLanguage, user?.preferred_language, languages]);

  // Load specific verb conjugations when verb changes
  useEffect(() => {
    // Don't load verb data until languages array is populated (unless user has no preferred language)
    // This prevents loading verbs with wrong language code
    if (
      selectedVerb &&
      (!user?.preferred_language || (languages && languages.length > 0))
    ) {
      const loadVerbData = async () => {
        setLoading(true);
        setError(null);
        try {
          // Find the verb object to get its slug if available
          const verbObj = availableVerbs.find(
            v => v.infinitive === selectedVerb
          );
          const slug = verbObj?.slug;
          const data = await loadVerbConjugation(
            userLanguage,
            selectedVerb,
            slug
          );
          setVerbData(data);
          // Reset accordion when verb changes
          setAccordionValue(null);
        } catch (err) {
          setError('Failed to load verb conjugations');
          console.error('Error loading verb conjugations:', err);
        } finally {
          setLoading(false);
        }
      };

      loadVerbData();
    }
  }, [
    selectedVerb,
    userLanguage,
    user?.preferred_language,
    languages,
    availableVerbs,
  ]);

  const getLanguageName = (code: string): string => {
    const languageInfo = languages?.find(lang => lang.code === code);
    if (languageInfo) {
      return (
        languageInfo.name.charAt(0).toUpperCase() + languageInfo.name.slice(1)
      );
    }
    return code.toUpperCase();
  };

  const handleTTSPlay = async (text: string) => {
    try {
      // Get the language info to find the correct TTS voice
      const languageInfo = languages?.find(lang => lang.code === userLanguage);

      if (!languageInfo) {
        console.error('Language info not found for:', userLanguage);
        return;
      }

      // Use the TTS voice from language settings
      const voice = languageInfo.tts_voice || languageInfo.tts_locale;

      if (!voice) {
        console.error('No TTS voice configured for language:', userLanguage);
        return;
      }

      await playTTSOnce(text, voice);
    } catch (error) {
      console.error('TTS playback failed:', error);
    }
  };

  const handleExpandAll = () => {
    if (verbData) {
      if (isAllExpanded) {
        // Collapse all
        setAccordionValue(null);
        setIsAllExpanded(false);
      } else {
        // Expand all
        const allTenseIds = verbData.tenses.map(tense => tense.tenseId);
        setAccordionValue(allTenseIds);
        setIsAllExpanded(true);
      }
    }
  };

  // Update expand all state when accordion value changes
  useEffect(() => {
    if (verbData) {
      const allTenseIds = verbData.tenses.map(tense => tense.tenseId);
      const isAllOpen =
        Array.isArray(accordionValue) &&
        allTenseIds.every(id => accordionValue.includes(id));
      setIsAllExpanded(isAllOpen);
    }
  }, [accordionValue, verbData]);

  const renderTense = (tense: {
    tenseId: string;
    tenseName: string;
    tenseNameEn: string;
    description: string;
    conjugations: Array<{
      pronoun: string;
      form: string;
      exampleSentence: string;
      exampleSentenceEn: string;
    }>;
  }) => (
    <Accordion.Item key={tense.tenseId} value={tense.tenseId}>
      <Accordion.Control>
        <Group justify='space-between' wrap='nowrap'>
          <Text fw={600} size='sm'>
            {tense.tenseName}
          </Text>
          <Badge size='sm' variant='light' color='blue'>
            {tense.tenseNameEn}
          </Badge>
        </Group>
      </Accordion.Control>
      <Accordion.Panel>
        <Stack gap='xs'>
          <Text size='xs' c='dimmed' mb='sm'>
            {tense.description}
          </Text>

          {tense.conjugations.map((conjugation, index) => (
            <Paper key={index} p='sm' withBorder>
              <Stack gap='xs'>
                <Group justify='space-between' wrap='nowrap'>
                  <Text fw={500} size='sm'>
                    {conjugation.pronoun}
                  </Text>
                  <Text size='sm' c='blue' fw={500}>
                    {conjugation.form}
                  </Text>
                </Group>

                <Group gap='xs' align='flex-start' wrap='nowrap'>
                  <ActionIcon
                    variant='subtle'
                    color='blue'
                    size='sm'
                    onClick={() => handleTTSPlay(conjugation.exampleSentence)}
                    style={{ flexShrink: 0, marginTop: '2px' }}
                  >
                    <IconVolume size={16} />
                  </ActionIcon>
                  <Box style={{ flex: 1, minWidth: 0 }}>
                    <HoverTranslation
                      text={conjugation.exampleSentence}
                      targetLanguage='en'
                    >
                      {conjugation.exampleSentence}
                    </HoverTranslation>
                  </Box>
                </Group>
              </Stack>
            </Paper>
          ))}
        </Stack>
      </Accordion.Panel>
    </Accordion.Item>
  );

  if (error) {
    return (
      <Container size='md' py='xl'>
        <Alert color='red' title='Error'>
          {error}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size='lg' py='md'>
      <Stack gap='md'>
        <Group>
          <IconBook size={28} />
          <div>
            <Title order={2}>Verb Conjugations</Title>
            <Text size='sm' c='dimmed'>
              {getLanguageName(userLanguage)} verb conjugation tables
            </Text>
          </div>
        </Group>

        {loading && (
          <Center py='xl'>
            <Loader size='lg' />
          </Center>
        )}

        {verbData && !loading && (
          <Stack gap='md'>
            <Group justify='space-between' align='center'>
              <Badge color='blue' variant='light' size='lg'>
                {availableVerbs.length} VERBS
              </Badge>
              <Button
                variant='light'
                size='sm'
                leftSection={
                  isAllExpanded ? (
                    <IconChevronUp size={16} />
                  ) : (
                    <IconChevronDown size={16} />
                  )
                }
                onClick={handleExpandAll}
                disabled={!verbData || verbData.tenses.length === 0}
              >
                {isAllExpanded ? 'Collapse All' : 'Expand All'}
              </Button>
            </Group>

            <Select
              data={availableVerbs.map(verb => ({
                value: verb.infinitive,
                label: `${verb.infinitive} (${verb.infinitiveEn})`,
              }))}
              value={selectedVerb}
              onChange={value => setSelectedVerb(value || '')}
              placeholder='Choose a verb'
              searchable
              size='md'
            />

            <Accordion
              value={accordionValue}
              onChange={setAccordionValue}
              chevronPosition='right'
              multiple
            >
              {verbData.tenses.map(renderTense)}
            </Accordion>
          </Stack>
        )}
      </Stack>
    </Container>
  );
}
