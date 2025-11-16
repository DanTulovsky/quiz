import React, { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Container,
  Group,
  Select,
  Stack,
  Text,
  Textarea,
  Title,
  Divider,
  Badge,
  Paper,
  ScrollArea,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import {
  useGeneratePracticeSentence,
  useGetPracticeSentence,
  usePracticeHistory,
  usePracticeStats,
  useSubmitTranslation,
  TranslationDirection,
  SentenceResponse,
} from '../api/translationPracticeApi';
import { useAuth } from '../hooks/useAuth';

const directionOptions = [
  { label: 'English → Learning language', value: 'en_to_learning' },
  { label: 'Learning language → English', value: 'learning_to_en' },
] as const;

const DEFAULT_HISTORY_LIMIT = 20;

const TranslationPracticePage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const [direction, setDirection] = useState<TranslationDirection>('learning_to_en');
  const [topic, setTopic] = useState('');
  const [answer, setAnswer] = useState('');
  const [currentSentence, setCurrentSentence] = useState<SentenceResponse | null>(null);
  const [loadingExisting, setLoadingExisting] = useState(false);

  const learningLanguage = user?.preferred_language || '';
  const level = (user?.current_level as string) || '';

  const { mutateAsync: generateSentence, isPending: isGenerating } = useGeneratePracticeSentence();
  const { mutateAsync: submitTranslation, isPending: isSubmitting } = useSubmitTranslation();
  const { data: stats } = usePracticeStats();
  const { data: history } = usePracticeHistory(DEFAULT_HISTORY_LIMIT);

  // fetch from existing content on demand (not mounted auto-query)
  const { refetch: refetchExisting } = useGetPracticeSentence(
    useMemo(
      () => ({
        language: learningLanguage || undefined,
        level: level || undefined,
        direction,
        enabled: false,
      }),
      [learningLanguage, level, direction]
    )
  );

  const canRequest = isAuthenticated && learningLanguage && level;

  const handleGenerate = async () => {
    if (!canRequest) {
      notifications.show({
        color: 'red',
        title: 'Missing settings',
        message: 'Please set your learning language and level in Settings.',
      });
      return;
    }
    try {
      const sentence = await generateSentence({
        language: learningLanguage,
        level,
        direction,
        topic: topic.trim() || undefined,
      });
      setCurrentSentence(sentence);
      setAnswer('');
    } catch (e) {
      notifications.show({
        color: 'red',
        title: 'Failed to generate',
        message: 'Could not generate a sentence. Please try again.',
      });
    }
  };

  const handleFromExisting = async () => {
    if (!canRequest) {
      notifications.show({
        color: 'red',
        title: 'Missing settings',
        message: 'Please set your learning language and level in Settings.',
      });
      return;
    }
    setLoadingExisting(true);
    try {
      const { data, error } = await refetchExisting();
      if (error) throw error;
      if (data) {
        setCurrentSentence(data);
        setAnswer('');
      }
    } catch {
      notifications.show({
        color: 'yellow',
        title: 'No sentence found',
        message: 'Could not find a suitable sentence from existing content.',
      });
    } finally {
      setLoadingExisting(false);
    }
  };

  const handleSubmit = async () => {
    if (!currentSentence) return;
    const trimmed = answer.trim();
    if (!trimmed) {
      notifications.show({
        color: 'red',
        title: 'Enter a translation',
        message: 'Please write your translation before submitting.',
      });
      return;
    }
    try {
      const resp = await submitTranslation({
        sentence_id: currentSentence.id,
        original_sentence: currentSentence.sentence_text,
        user_translation: trimmed,
        translation_direction: direction,
      });
      // Show feedback inline via notification and keep sentence visible
      notifications.show({
        color: 'teal',
        title: resp.ai_score != null ? `Feedback (score ${resp.ai_score.toFixed(1)}/5)` : 'Feedback',
        message: resp.ai_feedback,
        withCloseButton: true,
        autoClose: false,
      });
    } catch {
      notifications.show({
        color: 'red',
        title: 'Submit failed',
        message: 'Could not submit your translation. Please try again.',
      });
    }
  };

  return (
    <Container size="lg" pt="md" pb="xl">
      <Group justify="space-between" align="center" mb="md">
        <Title order={2}>Translation Practice</Title>
        <Group gap="xs">
          <Select
            data={directionOptions as unknown as { label: string; value: string }[]}
            value={direction}
            onChange={v => setDirection((v as TranslationDirection) || 'learning_to_en')}
            aria-label="Translation direction"
            w={280}
          />
          <Button variant="light" loading={isGenerating} onClick={handleGenerate}>
            Generate with AI
          </Button>
          <Button variant="light" loading={loadingExisting} onClick={handleFromExisting}>
            From existing content
          </Button>
        </Group>
      </Group>

      <Stack gap="md">
        <Card withBorder>
          <Stack gap="sm">
            <Group justify="space-between" align="baseline">
              <Title order={4}>Prompt</Title>
              <Group gap="xs">
                {learningLanguage ? <Badge>{learningLanguage}</Badge> : null}
                {level ? <Badge variant="light">Level {level}</Badge> : null}
              </Group>
            </Group>
            <Textarea
              label="Optional topic"
              placeholder="e.g., travel, ordering food, work"
              value={topic}
              onChange={e => setTopic(e.currentTarget.value)}
              autosize
              minRows={1}
              maxRows={3}
            />
            <Divider />
            <Text fw={600}>Text to translate</Text>
            <Paper withBorder p="md" bg="gray.0">
              {currentSentence ? (
                <Text>{currentSentence.sentence_text}</Text>
              ) : (
                <Text c="dimmed">Click Generate or From existing content to load a sentence.</Text>
              )}
            </Paper>
            <Textarea
              label="Your translation"
              placeholder="Write your translation here"
              value={answer}
              onChange={e => setAnswer(e.currentTarget.value)}
              autosize
              minRows={3}
            />
            <Group justify="flex-end">
              <Button onClick={handleSubmit} loading={isSubmitting} disabled={!currentSentence}>
                Submit for feedback
              </Button>
            </Group>
          </Stack>
        </Card>

        <Group align="start" grow>
          <Card withBorder>
            <Stack gap="xs">
              <Title order={5}>Stats</Title>
              <Text size="sm">
                Total sessions: {stats?.total_sessions ?? 0}
                {stats?.average_score != null ? ` • Avg score: ${stats.average_score.toFixed(2)}` : ''}
              </Text>
              <Group gap="xs">
                <Badge color="green" variant="light">
                  Excellent {stats?.excellent_count ?? 0}
                </Badge>
                <Badge color="blue" variant="light">
                  Good {stats?.good_count ?? 0}
                </Badge>
                <Badge color="yellow" variant="light">
                  Needs work {stats?.needs_improvement_count ?? 0}
                </Badge>
              </Group>
            </Stack>
          </Card>
          <Card withBorder>
            <Stack gap="xs">
              <Title order={5}>History</Title>
              <ScrollArea h={220}>
                <Stack gap="sm">
                  {(history?.sessions || []).map(s => (
                    <Paper key={s.id} withBorder p="sm">
                      <Text size="sm" fw={600}>
                        {s.original_sentence}
                      </Text>
                      <Text size="sm" c="dimmed">
                        You: {s.user_translation}
                      </Text>
                      <Group justify="space-between" mt="xs">
                        <Badge variant="light">{s.translation_direction.replaceAll('_', ' ')}</Badge>
                        {s.ai_score != null ? (
                          <Badge color="teal" variant="light">
                            {s.ai_score.toFixed(1)}/5
                          </Badge>
                        ) : null}
                      </Group>
                    </Paper>
                  ))}
                  {!history?.sessions?.length ? (
                    <Text size="sm" c="dimmed">
                      No practice yet. Submit a translation to see history here.
                    </Text>
                  ) : null}
                </Stack>
              </ScrollArea>
            </Stack>
          </Card>
        </Group>
      </Stack>
    </Container>
  );
};

export default TranslationPracticePage;

import React, { useMemo, useState } from 'react';
import { Button, Card, Group, SegmentedControl, Stack, Text, Textarea, Title, Select, Divider, Badge, Paper } from '@mantine/core';
import { IconBolt, IconDatabase, IconSend } from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';
import {
  TranslationDirection,
  useGeneratePracticeSentence,
  useGetPracticeSentence,
  useSubmitTranslation,
  usePracticeHistory,
  usePracticeStats,
} from '../api/translationPracticeApi';

const directionOptions: { label: string; value: TranslationDirection }[] = [
  { label: 'English → Learning', value: 'en_to_learning' },
  { label: 'Learning → English', value: 'learning_to_en' },
];

const levels = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2'];

const TranslationPracticePage: React.FC = () => {
  const { user } = useAuth();
  const [direction, setDirection] = useState<TranslationDirection>('learning_to_en');
  const [level, setLevel] = useState<string>(user?.current_level || 'A2');
  const [topic, setTopic] = useState<string>('');

  const [sentenceId, setSentenceId] = useState<number | null>(null);
  const [originalSentence, setOriginalSentence] = useState<string>('');
  const [userTranslation, setUserTranslation] = useState<string>('');
  const [feedback, setFeedback] = useState<{ text: string; score?: number | null } | null>(null);

  const learningLanguage = user?.preferred_language || 'es';

  const { mutateAsync: generateSentence, isLoading: isGenerating } = useGeneratePracticeSentence();
  const { mutateAsync: submitTranslation, isLoading: isSubmitting } = useSubmitTranslation();

  const getSentenceQuery = useGetPracticeSentence({
    language: learningLanguage,
    level,
    direction,
    enabled: false,
  });

  const statsQuery = usePracticeStats();
  const historyQuery = usePracticeHistory(10);

  const onGenerateClick = async () => {
    setFeedback(null);
    const resp = await generateSentence({
      language: learningLanguage,
      level,
      direction,
      topic: topic?.trim() || undefined,
    });
    setSentenceId(resp.id);
    setOriginalSentence(resp.sentence_text);
    setUserTranslation('');
  };

  const onFromContentClick = async () => {
    setFeedback(null);
    const resp = await getSentenceQuery.refetch();
    if (resp.data) {
      setSentenceId(resp.data.id);
      setOriginalSentence(resp.data.sentence_text);
      setUserTranslation('');
    }
  };

  const onSubmitClick = async () => {
    if (!sentenceId || !originalSentence.trim() || !userTranslation.trim()) return;
    const resp = await submitTranslation({
      sentence_id: sentenceId,
      original_sentence: originalSentence,
      user_translation: userTranslation,
      translation_direction: direction,
    });
    setFeedback({ text: resp.ai_feedback, score: resp.ai_score ?? null });
    historyQuery.refetch();
    statsQuery.refetch();
  };

  const directionHint = useMemo(() => {
    return direction === 'en_to_learning'
      ? `Translate from English into ${learningLanguage.toUpperCase()}`
      : `Translate from ${learningLanguage.toUpperCase()} into English`;
  }, [direction, learningLanguage]);

  return (
    <Stack gap="md">
      <Title order={2}>Translation Practice</Title>
      <Card withBorder>
        <Stack>
          <Group wrap="wrap" justify="space-between">
            <SegmentedControl
              data={directionOptions}
              value={direction}
              onChange={(val) => setDirection(val as TranslationDirection)}
            />
            <Group>
              <Select
                label="Level"
                placeholder="Select level"
                data={levels}
                value={level}
                onChange={(val) => setLevel(val || 'A2')}
                withinPortal
              />
            </Group>
          </Group>
          <Text size="sm" c="dimmed">{directionHint}</Text>
          <Group>
            <Textarea
              placeholder="Optional topic or keywords (e.g., travel, ordering food)"
              value={topic}
              onChange={(e) => setTopic(e.currentTarget.value)}
              autosize
              minRows={1}
              maxRows={2}
            />
          </Group>
          <Group>
            <Button leftSection={<IconBolt size={16} />} loading={isGenerating} onClick={onGenerateClick}>
              Generate sentence
            </Button>
            <Button variant="default" leftSection={<IconDatabase size={16} />} onClick={onFromContentClick}>
              From existing content
            </Button>
          </Group>
        </Stack>
      </Card>

      {originalSentence && (
        <Card withBorder>
          <Stack>
            <Group justify="space-between" align="center">
              <Title order={4} m={0}>Original sentence</Title>
              <Badge variant="light">{level}</Badge>
            </Group>
            <Paper p="sm" withBorder>
              <Text>{originalSentence}</Text>
            </Paper>
            <Textarea
              label="Your translation"
              placeholder="Type your translation here"
              value={userTranslation}
              onChange={(e) => setUserTranslation(e.currentTarget.value)}
              autosize
              minRows={3}
            />
            <Group>
              <Button leftSection={<IconSend size={16} />} loading={isSubmitting} onClick={onSubmitClick}>
                Submit for feedback
              </Button>
            </Group>
          </Stack>
        </Card>
      )}

      {feedback && (
        <Card withBorder>
          <Stack>
            <Group justify="space-between">
              <Title order={4} m={0}>AI Feedback</Title>
              {typeof feedback.score === 'number' && (
                <Badge color={feedback.score >= 4 ? 'green' : feedback.score >= 3 ? 'yellow' : 'red'}>
                  Score: {feedback.score.toFixed(1)} / 5
                </Badge>
              )}
            </Group>
            <Divider />
            <Text style={{ whiteSpace: 'pre-wrap' }}>{feedback.text}</Text>
          </Stack>
        </Card>
      )}

      <Group align="start" grow>
        <Card withBorder>
          <Title order={5}>Recent practice</Title>
          <Divider my="sm" />
          <Stack gap="xs">
            {(historyQuery.data?.sessions || []).map((s) => (
              <Stack key={s.id} gap={4}>
                <Text size="sm" c="dimmed">{new Date(s.created_at).toLocaleString()}</Text>
                <Text size="sm"><strong>Original:</strong> {s.original_sentence}</Text>
                <Text size="sm"><strong>Your translation:</strong> {s.user_translation}</Text>
                {s.ai_score != null && (
                  <Badge size="sm" variant="light">Score: {s.ai_score.toFixed(1)}</Badge>
                )}
                <Divider my={4} />
              </Stack>
            ))}
            {historyQuery.data?.sessions?.length === 0 && <Text size="sm" c="dimmed">No history yet</Text>}
          </Stack>
        </Card>

        <Card withBorder>
          <Title order={5}>Stats</Title>
          <Divider my="sm" />
          <Stack gap={6}>
            <Text size="sm">Total: {statsQuery.data?.total_sessions ?? 0}</Text>
            <Text size="sm">Avg score: {statsQuery.data?.average_score?.toFixed?.(1) ?? '—'}</Text>
            <Text size="sm">Excellent (≥4): {statsQuery.data?.excellent_count ?? 0}</Text>
            <Text size="sm">Good (3–3.9): {statsQuery.data?.good_count ?? 0}</Text>
            <Text size="sm">Needs work (<3): {statsQuery.data?.needs_improvement_count ?? 0}</Text>
          </Stack>
        </Card>
      </Group>
    </Stack>
  );
};

export default TranslationPracticePage;
