import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Container,
  Group,
  Accordion,
  Select,
  Stack,
  Text,
  TextInput,
  Textarea,
  Title,
  Divider,
  Badge,
  Paper,
  ScrollArea,
  Anchor,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
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

function toTitle(s: string) {
  if (!s) return '';
  return s.charAt(0).toUpperCase() + s.slice(1);
}

const DEFAULT_HISTORY_LIMIT = 20;

const TranslationPracticePage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const [direction, setDirection] = useState<TranslationDirection>('learning_to_en');
  const [topic, setTopic] = useState('');
  const [answer, setAnswer] = useState('');
  const [currentSentence, setCurrentSentence] = useState<SentenceResponse | null>(null);
  const [loadingExisting, setLoadingExisting] = useState(false);
  const [feedback, setFeedback] = useState<{ text: string; score?: number | null } | null>(null);
  const feedbackTitleRef = useRef<HTMLHeadingElement | null>(null);
  const historyViewportRef = useRef<HTMLDivElement | null>(null);
  const [historyLimit, setHistoryLimit] = useState<number>(DEFAULT_HISTORY_LIMIT);
  const SERVER_HISTORY_MAX = 100;
  const [historySearch, setHistorySearch] = useState<string>('');

  const learningLanguage = user?.preferred_language || '';
  const level = (user?.current_level as string) || '';

  const { mutateAsync: generateSentence, isPending: isGenerating } = useGeneratePracticeSentence();
  const { mutateAsync: submitTranslation, isPending: isSubmitting } = useSubmitTranslation();
  const { data: stats } = usePracticeStats();
  const { data: history } = usePracticeHistory(historyLimit);

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
      setFeedback(null);
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
        setFeedback(null);
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
      // Show feedback inline (no notification)
      setFeedback({ text: resp.ai_feedback, score: resp.ai_score ?? null });
    } catch {
      notifications.show({
        color: 'red',
        title: 'Submit failed',
        message: 'Could not submit your translation. Please try again.',
      });
    }
  };

  // Focus feedback header when feedback appears for accessibility
  useEffect(() => {
    if (feedback && feedbackTitleRef.current) {
      feedbackTitleRef.current.focus();
    }
  }, [feedback]);
  // Infinite scroll: grow historyLimit when scrolled near bottom (max 100 per API)
  const onHistoryScroll = () => {
    const el = historyViewportRef.current;
    if (!el) return;
    const nearBottom = el.scrollTop + el.clientHeight >= el.scrollHeight - 80;
    if (nearBottom && historyLimit < 100) {
      setHistoryLimit(prev => Math.min(prev + 25, 100));
    }
  };
  const filteredSessions = useMemo(() => {
    const q = historySearch.trim().toLowerCase();
    const list = history?.sessions || [];
    if (!q) return list;
    return list.filter(s => {
      return (
        s.original_sentence?.toLowerCase().includes(q) ||
        s.user_translation?.toLowerCase().includes(q) ||
        s.ai_feedback?.toLowerCase().includes(q) ||
        s.translation_direction?.toLowerCase().includes(q)
      );
    });
  }, [history?.sessions, historySearch]);
  const getSourceInfo = (sourceType?: string | null, sourceId?: number | null) => {
    if (!sourceType || !sourceId) return null;
    const type = sourceType.toLowerCase();
    if (type === 'vocabulary_question') {
      return { label: 'Vocabulary question', href: `/vocabulary/${sourceId}` };
    }
    if (type === 'reading_comprehension') {
      return { label: 'Reading comprehension', href: `/reading-comprehension/${sourceId}` };
    }
    if (type === 'story_section') {
      // Best-effort jump: Story page, passing sectionId as query for now
      return { label: 'Story section', href: `/story?sectionId=${sourceId}` };
    }
    if (type === 'snippet') {
      return { label: 'Snippet', href: `/snippets` };
    }
    if (type === 'phrasebook') {
      return { label: 'Phrasebook', href: `/phrasebook` };
    }
    return { label: type.replaceAll('_', ' '), href: undefined as unknown as string };
  };
  const languageDisplay = toTitle(learningLanguage || 'learning language');
  const directionOptions = useMemo(
    () => [
      { label: `English → ${languageDisplay}`, value: 'en_to_learning' },
      { label: `${languageDisplay} → English`, value: 'learning_to_en' },
    ],
    [languageDisplay]
  );

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
                <Stack gap={6}>
                  <Text>{currentSentence.sentence_text}</Text>
                  {(() => {
                    const info = getSourceInfo(currentSentence.source_type as unknown as string, currentSentence.source_id as unknown as number);
                    return info ? (
                      <Text size="xs" c="dimmed">
                        From existing content: {info.href ? <Anchor href={info.href}>{info.label}</Anchor> : info.label}
                      </Text>
                    ) : null;
                  })()}
                </Stack>
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

        {/* AI Feedback section placed directly below translation card */}
        {feedback && (
          <Card withBorder>
            <Stack>
              <Group justify="space-between" align="center">
                <Title order={4} m={0} ref={feedbackTitleRef} tabIndex={-1}>
                  AI Feedback
                </Title>
                {typeof feedback.score === 'number' && (
                  <Badge color={feedback.score >= 4 ? 'green' : feedback.score >= 3 ? 'yellow' : 'red'}>
                    Score: {feedback.score.toFixed(1)} / 5
                  </Badge>
                )}
              </Group>
              <Divider />
              <ScrollArea h={480} type="auto">
                <Paper p="sm" withBorder>
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>{feedback.text}</ReactMarkdown>
                </Paper>
              </ScrollArea>
            </Stack>
          </Card>
        )}

        <Stack>
        <Card withBorder>
          <Stack gap="xs">
            <Group justify="space-between" align="center">
              <Title order={5}>History</Title>
              <Group gap="xs">
                <Text size="xs" c="dimmed">
                  Showing {filteredSessions.length} of {history?.sessions?.length ?? 0} loaded • Server max {SERVER_HISTORY_MAX}
                </Text>
                <Button
                  size="xs"
                  variant="subtle"
                  onClick={() => historyViewportRef.current?.scrollTo({ top: 0, behavior: 'smooth' })}
                >
                  Top
                </Button>
                <Button
                  size="xs"
                  variant="subtle"
                  onClick={() =>
                    historyViewportRef.current?.scrollTo({
                      top: historyViewportRef.current.scrollHeight,
                      behavior: 'smooth',
                    })
                  }
                >
                  Bottom
                </Button>
                <Button
                  size="xs"
                  variant="light"
                  onClick={() => setHistoryLimit(SERVER_HISTORY_MAX)}
                  disabled={historyLimit >= SERVER_HISTORY_MAX}
                >
                  Load all ({SERVER_HISTORY_MAX})
                </Button>
              </Group>
            </Group>
            <TextInput
              placeholder="Search history (original, your translation, feedback, direction)"
              value={historySearch}
              onChange={e => setHistorySearch(e.currentTarget.value)}
            />
            <ScrollArea
              style={{ height: '60vh' }}
              viewportRef={historyViewportRef}
              viewportProps={{ onScroll: onHistoryScroll }}
            >
              {(filteredSessions.length ?? 0) > 0 ? (
                <Accordion variant="separated">
                  {filteredSessions.map(s => (
                    <Accordion.Item value={String(s.id)} key={s.id}>
                      <Accordion.Control>
                        <Group justify="space-between" align="flex-start" wrap="nowrap">
                          <Stack gap={2} style={{ flex: 1, minWidth: 0 }}>
                            <Text size="sm" fw={600} lineClamp={1}>
                              {s.original_sentence}
                            </Text>
                            <Text size="xs" c="dimmed" lineClamp={1}>
                              You: {s.user_translation}
                            </Text>
                          </Stack>
                          <Group gap="xs" wrap="nowrap">
                            <Badge variant="light">{s.translation_direction.replaceAll('_', ' ')}</Badge>
                            {s.ai_score != null ? (
                              <Badge color="teal" variant="light">
                                {s.ai_score.toFixed(1)}/5
                              </Badge>
                            ) : null}
                          </Group>
                        </Group>
                      </Accordion.Control>
                      <Accordion.Panel>
                        <Stack gap="xs">
                          <Text size="xs" c="dimmed">
                            {new Date(s.created_at).toLocaleString()}
                          </Text>
                          <Paper withBorder p="sm">
                            <Text size="sm" fw={600}>
                              Original
                            </Text>
                            <Text size="sm">{s.original_sentence}</Text>
                          </Paper>
                          <Paper withBorder p="sm">
                            <Text size="sm" fw={600}>
                              Your translation
                            </Text>
                            <Text size="sm">{s.user_translation}</Text>
                          </Paper>
                          <Paper withBorder p="sm">
                            <Group justify="space-between" align="center">
                              <Text size="sm" fw={600}>
                                AI Feedback
                              </Text>
                              {s.ai_score != null ? (
                                <Badge color={s.ai_score >= 4 ? 'green' : s.ai_score >= 3 ? 'yellow' : 'red'}>
                                  {s.ai_score.toFixed(1)} / 5
                                </Badge>
                              ) : null}
                            </Group>
                            <Divider my="xs" />
                            <ReactMarkdown remarkPlugins={[remarkGfm]}>{s.ai_feedback}</ReactMarkdown>
                          </Paper>
                        </Stack>
                      </Accordion.Panel>
                    </Accordion.Item>
                  ))}
                </Accordion>
              ) : (
                <Text size="sm" c="dimmed">
                  No practice yet. Submit a translation to see history here.
                </Text>
              )}
            </ScrollArea>
          </Stack>
        </Card>
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
        </Stack>
      </Stack>
    </Container>
  );
};

export default TranslationPracticePage;

/* Duplicate component block removed below
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
            <Text size="sm">Needs work (&lt;3): {statsQuery.data?.needs_improvement_count ?? 0}</Text>
          </Stack>
        </Card>
      </Group>
    </Stack>
  );
};

export default TranslationPracticePage;
*/
