import { useState, useEffect, useRef } from 'react';
import {
  Container,
  Title,
  Text,
  Select,
  Card,
  Group,
  Stack,
  Badge,
  Alert,
  Loader,
  Center,
  Divider,
  Table,
} from '@mantine/core';
import { IconBook } from '@tabler/icons-react';
import {
  loadVerbConjugations,
  loadVerbConjugation,
  VerbConjugation,
} from '../utils/verbConjugations';
import { HoverTranslation } from '../components/HoverTranslation';
import { useAuth } from '../hooks/useAuth';
import TTSButton from '../components/TTSButton';
import {
  useGetV1SettingsLanguages,
  useGetV1PreferencesLearning,
} from '../api/api';
import { defaultVoiceForLanguage } from '../utils/tts';

export function VerbConjugationPage() {
  const { user } = useAuth();
  const [availableVerbs, setAvailableVerbs] = useState<VerbConjugation[]>([]);
  const [selectedVerb, setSelectedVerb] = useState<string>('');
  const [selectedTense, setSelectedTense] = useState<string>('');
  const [verbData, setVerbData] = useState<VerbConjugation | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const tenseRefs = useRef<{ [key: string]: HTMLDivElement | null }>({});
  const isProgrammaticScroll = useRef(false);

  // Use React Query to fetch languages (with proper caching)
  const { data: languages } = useGetV1SettingsLanguages();
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();

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
          // Reset tense selection when verb changes
          setSelectedTense('');
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

  // Scroll to selected tense
  useEffect(() => {
    if (selectedTense && tenseRefs.current[selectedTense]) {
      const element = tenseRefs.current[selectedTense];
      if (element) {
        isProgrammaticScroll.current = true;
        const elementRect = element.getBoundingClientRect();
        const absoluteElementTop = elementRect.top + window.pageYOffset;
        const offset = 100; // Adjust this value to fine-tune positioning
        window.scrollTo({
          top: absoluteElementTop - offset,
          behavior: 'smooth',
        });

        // Reset the flag after a short delay to allow the smooth scroll to complete
        setTimeout(() => {
          isProgrammaticScroll.current = false;
        }, 1000);
      }
    }
  }, [selectedTense]);

  // Reset tense selection when manually scrolling (not programmatic)
  useEffect(() => {
    const handleScroll = () => {
      if (!isProgrammaticScroll.current) {
        setSelectedTense('');
      } else {
      }
    };

    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const getLanguageName = (code: string): string => {
    const languageInfo = languages?.find(lang => lang.code === code);
    if (languageInfo) {
      return (
        languageInfo.name.charAt(0).toUpperCase() + languageInfo.name.slice(1)
      );
    }
    return code.toUpperCase();
  };

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
    <Card
      key={tense.tenseId}
      withBorder
      p='md'
      mb='sm'
      ref={el => (tenseRefs.current[tense.tenseId] = el)}
    >
      <Stack gap='md'>
        <Group>
          <Text fw={600} size='lg'>
            {tense.tenseName}
          </Text>
          <Badge size='sm' variant='light' color='blue'>
            {tense.tenseNameEn}
          </Badge>
        </Group>

        <Text size='sm' c='dimmed'>
          {tense.description}
        </Text>

        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Pronoun</Table.Th>
              <Table.Th>Conjugation</Table.Th>
              <Table.Th>Example</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {tense.conjugations.map((conjugation, index) => (
              <Table.Tr key={index}>
                <Table.Td>
                  <Text fw={500} size='sm'>
                    {conjugation.pronoun}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Text size='sm' c='blue' fw={500}>
                    {conjugation.form}
                  </Text>
                </Table.Td>
                <Table.Td>
                  <Group gap='xs' align='center'>
                    <TTSButton
                      getText={() => conjugation.exampleSentence}
                      getVoice={() => {
                        const saved = (
                          userLearningPrefs?.tts_voice || ''
                        ).trim();
                        if (saved) return saved;
                        const voice = defaultVoiceForLanguage(
                          user?.preferred_language || undefined
                        );
                        return voice || undefined;
                      }}
                      size='sm'
                      ariaLabel='Pronounce example'
                    />
                    <HoverTranslation
                      text={conjugation.exampleSentence}
                      targetLanguage='en'
                    >
                      {conjugation.exampleSentence}
                    </HoverTranslation>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </Stack>
    </Card>
  );

  const renderVerb = (verb: VerbConjugation) => (
    <Stack gap='xs'>{verb.tenses.map(renderTense)}</Stack>
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
    <Container size='lg' py='xl'>
      <Stack gap='xl'>
        <Group>
          <IconBook size={32} />
          <div>
            <Title order={1}>Verb Conjugations</Title>
            <Text c='dimmed'>
              Comprehensive verb conjugation tables with examples for essential
              verbs
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
            <Group justify='space-between' align='flex-end'>
              <Group>
                <Text size='lg' fw={500}>
                  {getLanguageName(userLanguage)} Verb Conjugations
                </Text>
                <Badge color='blue' variant='light'>
                  {availableVerbs.length} verbs available
                </Badge>
              </Group>

              <Group>
                <Select
                  key={selectedTense || 'empty'}
                  data={verbData.tenses.map(tense => ({
                    value: tense.tenseId,
                    label: `${tense.tenseName} (${tense.tenseNameEn})`,
                  }))}
                  value={selectedTense}
                  onChange={value => {
                    setSelectedTense(value || '');
                  }}
                  placeholder='Choose a tense'
                  clearable
                  maxDropdownHeight={400}
                  style={{ minWidth: '250px' }}
                  comboboxProps={{ width: 'auto', dropdownPadding: 4 }}
                />
                <Select
                  data={availableVerbs.map(verb => ({
                    value: verb.infinitive,
                    label: `${verb.infinitive} (${verb.infinitiveEn})`,
                  }))}
                  value={selectedVerb}
                  onChange={value => setSelectedVerb(value || '')}
                  placeholder='Choose a verb'
                  searchable
                  style={{ minWidth: '250px' }}
                />
              </Group>
            </Group>

            <Divider />

            {renderVerb(verbData)}
          </Stack>
        )}
      </Stack>
    </Container>
  );
}
