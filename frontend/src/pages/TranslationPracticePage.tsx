import React, { useEffect, useMemo, useRef, useState, useCallback } from 'react';
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
  Box,
  Transition,
  Tooltip,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { useHotkeys } from 'react-hotkeys-hook';
import { useMediaQuery, useLocalStorage } from '@mantine/hooks';
import {
  useGeneratePracticeSentence,
  useGetPracticeSentence,
  usePracticeHistory,
  usePracticeStats,
  useSubmitTranslation,
  TranslationDirection,
  SentenceResponse,
} from '../api/translationPracticeApi';
import { AXIOS_INSTANCE } from '../api/axios';
import { useAuth } from '../hooks/useAuth';
import * as TablerIcons from '@tabler/icons-react';
import TTSButton from '../components/TTSButton';
import { defaultVoiceForLanguage } from '../utils/tts';
import { useGetV1PreferencesLearning } from '../api/api';

/* eslint-disable @typescript-eslint/no-explicit-any */
const IconChevronRight = TablerIcons.IconChevronRight as unknown as any;
const IconChevronLeft = TablerIcons.IconChevronLeft as unknown as any;
const IconKeyboard = TablerIcons.IconKeyboard as unknown as any;
/* eslint-enable @typescript-eslint/no-explicit-any */

function toTitle(s: string) {
  if (!s) return '';
  return s.charAt(0).toUpperCase() + s.slice(1);
}

const textInputTypes = new Set([
  'text',
  'search',
  'email',
  'number',
  'password',
  'tel',
  'url',
]);

const isTypingTarget = (target: EventTarget | null): boolean => {
  const element = target as HTMLElement | null;
  if (!element) return false;
  if (element.isContentEditable) return true;
  const tagName = element.tagName;
  if (tagName === 'TEXTAREA' || tagName === 'SELECT') {
    return true;
  }
  if (tagName === 'INPUT') {
    const input = element as HTMLInputElement;
    return textInputTypes.has(input.type.toLowerCase());
  }
  return false;
};

const TranslationPracticePage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const [direction, setDirection] = useState<TranslationDirection>('learning_to_en');
  const [actualDirection, setActualDirection] = useState<TranslationDirection>('learning_to_en');
  const [topic, setTopic] = useState('');
  const [answer, setAnswer] = useState('');
  const [currentSentence, setCurrentSentence] = useState<SentenceResponse | null>(null);
  const [loadingExisting, setLoadingExisting] = useState(false);
  const [feedback, setFeedback] = useState<{ text: string; score?: number | null } | null>(null);
  const [lastGenerationType, setLastGenerationType] = useState<'ai' | 'existing' | null>(null);
  const [isInputFocused, setIsInputFocused] = useState(false);
  const [isDirectionFocused, setIsDirectionFocused] = useState(false);
  const [isDirectionDropdownOpen, setIsDirectionDropdownOpen] = useState(false);
  const [tabCycleIndex, setTabCycleIndex] = useState(0);
  const [shortcutsExpanded, setShortcutsExpanded] = useLocalStorage({
    key: 'translation-practice-shortcuts-expanded',
    defaultValue: false,
  });
  const isSmallScreen = useMediaQuery('(max-width: 1200px) or (max-height: 700px)');

  const feedbackTitleRef = useRef<HTMLHeadingElement | null>(null);
  const feedbackCardRef = useRef<HTMLDivElement | null>(null);
  const historyViewportRef = useRef<HTMLDivElement | null>(null);
  const topicInputRef = useRef<HTMLTextAreaElement | null>(null);
  const translationInputRef = useRef<HTMLTextAreaElement | null>(null);
  const historySearchInputRef = useRef<HTMLInputElement | null>(null);
  const historyCardRef = useRef<HTMLDivElement | null>(null);

  const HISTORY_PAGE_SIZE = 20;
  const [historyOffset, setHistoryOffset] = useState<number>(0);
  const [historySearch, setHistorySearch] = useState<string>('');

  const learningLanguage = user?.preferred_language || '';
  const level = (user?.current_level as string) || '';

  const languageDisplay = toTitle(learningLanguage || 'learning language');
  const directionOptions = useMemo(
    () => [
      { label: `English → ${languageDisplay}`, value: 'en_to_learning' },
      { label: `${languageDisplay} → English`, value: 'learning_to_en' },
      { label: 'Random', value: 'random' },
    ],
    [languageDisplay]
  );

  const { mutateAsync: generateSentence, isPending: isGenerating } = useGeneratePracticeSentence();
  const { mutateAsync: submitTranslation, isPending: isSubmitting } = useSubmitTranslation();
  const { data: stats } = usePracticeStats();
  const { data: history } = usePracticeHistory(HISTORY_PAGE_SIZE, historyOffset, historySearch.trim() || undefined);

  // Reset to first page when search changes
  useEffect(() => {
    setHistoryOffset(0);
  }, [historySearch]);
  const { data: learningPrefs } = useGetV1PreferencesLearning();

  // fetch from existing content on demand (not mounted auto-query)
  const { refetch: refetchExisting } = useGetPracticeSentence(
    useMemo(
      () => ({
        language: learningLanguage || undefined,
        level: level || undefined,
        direction: actualDirection,
        enabled: false,
      }),
      [learningLanguage, level, actualDirection]
    )
  );

  const canRequest = isAuthenticated && learningLanguage && level;

  // Determine if "from existing content" should be disabled
  // It's disabled when the source language would be English (en_to_learning or random)
  const isFromExistingDisabled = useMemo(() => {
    return direction === 'en_to_learning' || direction === 'random';
  }, [direction]);

  const handleGenerate = useCallback(async () => {
    if (!canRequest) {
      notifications.show({
        color: 'red',
        title: 'Missing settings',
        message: 'Please set your learning language and level in Settings.',
      });
      return;
    }
    try {
      // If random is selected, pick a random direction
      let dirToUse: 'en_to_learning' | 'learning_to_en';
      if (direction === 'random') {
        dirToUse = Math.random() < 0.5 ? 'en_to_learning' : 'learning_to_en';
        setActualDirection(dirToUse);
      } else {
        setActualDirection(direction);
        dirToUse = direction as 'en_to_learning' | 'learning_to_en';
      }

      const sentence = await generateSentence({
        language: learningLanguage,
        level,
        direction: dirToUse,
        topic: topic.trim() || undefined,
      });
      setCurrentSentence(sentence);
      setAnswer('');
      setFeedback(null);
      setLastGenerationType('ai');
      // Focus translation input after generation
      setTimeout(() => {
        translationInputRef.current?.focus();
      }, 100);
    } catch (e) {
      notifications.show({
        color: 'red',
        title: 'Failed to generate',
        message: 'Could not generate a sentence. Please try again.',
      });
    }
  }, [canRequest, direction, learningLanguage, level, topic, generateSentence]);

  const handleFromExisting = useCallback(async () => {
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
      // If random is selected, pick a random direction
      let dirToUse: 'en_to_learning' | 'learning_to_en';
      if (direction === 'random') {
        dirToUse = Math.random() < 0.5 ? 'en_to_learning' : 'learning_to_en';
        setActualDirection(dirToUse);
      } else {
        setActualDirection(direction);
        dirToUse = direction as 'en_to_learning' | 'learning_to_en';
      }

      // Make API call directly with the direction we want to use
      const qs = new URLSearchParams({
        language: learningLanguage,
        level,
        direction: dirToUse,
      });
      const resp = await AXIOS_INSTANCE.get(`/v1/translation-practice/sentence?${qs.toString()}`, {
        headers: { Accept: 'application/json' },
      });
      const data = resp.data as SentenceResponse;

      if (data) {
        setCurrentSentence(data);
        setAnswer('');
        setFeedback(null);
        setLastGenerationType('existing');
        // Focus translation input after loading
        setTimeout(() => {
          translationInputRef.current?.focus();
        }, 100);
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
  }, [canRequest, direction, learningLanguage, level]);

  const handleSubmit = useCallback(async () => {
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
        translation_direction: actualDirection,
      });
      // Show feedback inline (no notification)
      setFeedback({ text: resp.ai_feedback, score: resp.ai_score ?? null });
      // Scroll to feedback after it appears - position with offset to keep header visible
      setTimeout(() => {
        if (feedbackCardRef.current) {
          const cardRect = feedbackCardRef.current.getBoundingClientRect();
          const currentScroll = window.pageYOffset || document.documentElement.scrollTop;
          const headerOffset = 100; // Offset to keep header visible
          const targetPosition = currentScroll + cardRect.top - headerOffset;
          window.scrollTo({ top: targetPosition, behavior: 'smooth' });
        }
      }, 200);
    } catch (error) {
      console.error('Submit translation error:', error);
      notifications.show({
        color: 'red',
        title: 'Submit failed',
        message: 'Could not submit your translation. Please try again.',
      });
    }
  }, [currentSentence, answer, actualDirection, submitTranslation]);

  // Focus feedback header when feedback appears for accessibility
  useEffect(() => {
    if (feedback && feedbackTitleRef.current) {
      feedbackTitleRef.current.focus();
    }
  }, [feedback]);

  // Auto-load from existing content on mount
  useEffect(() => {
    if (canRequest && !currentSentence) {
      handleFromExisting();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canRequest]);

  // Handle "n" key - generate another of the same type
  const handleNextSameType = useCallback(() => {
    if (lastGenerationType === 'ai') {
      handleGenerate();
    } else if (lastGenerationType === 'existing') {
      handleFromExisting();
    }
  }, [lastGenerationType, handleGenerate, handleFromExisting]);

  // Tab cycling between topic and translation fields
  const handleTabCycle = useCallback(() => {
    if (tabCycleIndex === 0) {
      translationInputRef.current?.focus();
      setTabCycleIndex(1);
    } else {
      topicInputRef.current?.focus();
      setTabCycleIndex(0);
    }
  }, [tabCycleIndex]);

  // Keyboard shortcuts
  useHotkeys(
    'a',
    e => {
      if (isInputFocused || isDirectionFocused) return;
      e.preventDefault();
      handleGenerate();
    },
    { enableOnFormTags: false, preventDefault: true },
    [isInputFocused, isDirectionFocused, handleGenerate]
  );

  useHotkeys(
    'e',
    e => {
      if (isInputFocused || isDirectionFocused) return;
      if (isFromExistingDisabled) return; // Don't trigger if disabled
      e.preventDefault();
      handleFromExisting();
    },
    { enableOnFormTags: false, preventDefault: true },
    [isInputFocused, isDirectionFocused, isFromExistingDisabled, handleFromExisting]
  );

  useHotkeys(
    'n',
    e => {
      if (isInputFocused || isDirectionFocused) return;
      e.preventDefault();
      handleNextSameType();
    },
    { enableOnFormTags: false, preventDefault: true },
    [isInputFocused, isDirectionFocused, handleNextSameType]
  );

  useHotkeys(
    'tab',
    e => {
      const activeElement = document.activeElement;
      if (
        activeElement === topicInputRef.current ||
        activeElement === translationInputRef.current
      ) {
        e.preventDefault();
        handleTabCycle();
      }
    },
    { enableOnFormTags: true, preventDefault: true },
    [handleTabCycle]
  );

  useHotkeys(
    'mod+enter',
    e => {
      e.preventDefault();
      handleSubmit();
    },
    { enableOnFormTags: true, preventDefault: true },
    [handleSubmit]
  );

  useHotkeys(
    'escape',
    e => {
      const activeElement = document.activeElement as HTMLElement | null;
      if (isTypingTarget(activeElement)) {
        e.preventDefault();
        activeElement?.blur();
        setIsInputFocused(false);
        setIsDirectionFocused(false);
      }
    },
    { enableOnFormTags: true, preventDefault: true }
  );

  useHotkeys(
    't',
    e => {
      if (isInputFocused || isDirectionFocused) return;
      e.preventDefault();
      window.scrollTo({ top: 0, behavior: 'smooth' });
    },
    { enableOnFormTags: false, preventDefault: true },
    [isInputFocused, isDirectionFocused]
  );


  useHotkeys(
    'h',
    e => {
      if (isInputFocused || isDirectionFocused) return;
      e.preventDefault();
      historyCardRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      setTimeout(() => {
        historySearchInputRef.current?.focus();
      }, 100);
    },
    { enableOnFormTags: false, preventDefault: true },
    [isInputFocused, isDirectionFocused]
  );

  useHotkeys(
    'd',
    e => {
      if (isInputFocused) return;
      e.preventDefault();
      // Focus the Select component - find the actual input element inside
      const selectWrapper = document.querySelector('[aria-label="Translation direction"]');
      if (selectWrapper) {
        // Try to find the input element inside the Select
        const input = selectWrapper.querySelector('input') as HTMLElement;
        if (input) {
          input.focus();
          input.click(); // Also click to open the dropdown
          setIsDirectionFocused(true);
        } else {
          // Fallback: try to focus the wrapper itself
          (selectWrapper as HTMLElement).focus();
          (selectWrapper as HTMLElement).click();
          setIsDirectionFocused(true);
        }
      }
    },
    { enableOnFormTags: false, preventDefault: true },
    [isInputFocused]
  );

  // Handle direction dropdown navigation when focused but dropdown is closed
  // When dropdown is open, let Mantine handle arrow keys naturally (don't intercept)
  useHotkeys(
    ['arrowup', 'arrowdown'],
    e => {
      // Don't intercept if dropdown is open - let Mantine handle it
      if (isDirectionDropdownOpen) return;
      if (!isDirectionFocused) return;

      // Only handle when dropdown is closed - open it and let Mantine handle navigation
      const selectWrapper = document.querySelector('[aria-label="Translation direction"]') as HTMLElement;
      if (!selectWrapper) return;

      e.preventDefault();
      // Open the dropdown - Mantine will handle arrow key navigation once it's open
      selectWrapper.click();
      setIsDirectionDropdownOpen(true);
    },
    {
      enableOnFormTags: true,
      preventDefault: false, // Don't prevent default - we'll do it conditionally
      enabled: isDirectionFocused && !isDirectionDropdownOpen,
    },
    [isDirectionFocused, isDirectionDropdownOpen]
  );

  useHotkeys(
    'enter',
    e => {
      if (isDirectionFocused && isDirectionDropdownOpen) {
        // Let Mantine handle the selection first, then blur
        setTimeout(() => {
          setIsDirectionFocused(false);
          setIsDirectionDropdownOpen(false);
          const selectWrapper = document.querySelector('[aria-label="Translation direction"]') as HTMLElement;
          if (selectWrapper) {
            const input = selectWrapper.querySelector('input') as HTMLElement;
            if (input) {
              input.blur();
            } else {
              selectWrapper.blur();
            }
          }
        }, 50);
      }
    },
    { enableOnFormTags: true, preventDefault: false },
    [isDirectionFocused, isDirectionDropdownOpen]
  );

  // Handle '<' and '>' shortcuts to collapse/expand the shortcuts panel
  useHotkeys(
    'shift+comma',
    e => {
      e.preventDefault();
      setShortcutsExpanded(true);
    },
    { enableOnFormTags: false, preventDefault: true },
    []
  );

  useHotkeys(
    'shift+period',
    e => {
      e.preventDefault();
      setShortcutsExpanded(false);
    },
    { enableOnFormTags: false, preventDefault: true },
    []
  );
  // Pagination helpers
  const totalPages = history?.total ? Math.ceil(history.total / HISTORY_PAGE_SIZE) : 0;
  const currentPage = Math.floor(historyOffset / HISTORY_PAGE_SIZE) + 1;
  const hasNextPage = history ? historyOffset + HISTORY_PAGE_SIZE < history.total : false;
  const hasPrevPage = historyOffset > 0;

  const handlePrevPage = () => {
    setHistoryOffset(prev => Math.max(0, prev - HISTORY_PAGE_SIZE));
    historyViewportRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const handleNextPage = () => {
    if (hasNextPage) {
      setHistoryOffset(prev => prev + HISTORY_PAGE_SIZE);
      historyViewportRef.current?.scrollTo({ top: 0, behavior: 'smooth' });
    }
  };
  // Server-side search is now handled by the API, so we use the sessions directly
  const sessions = history?.sessions || [];
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

  const isExpanded = shortcutsExpanded && !isSmallScreen;

  return (
    <Box style={{ position: 'relative' }}>
      <Container size="lg" pt="md" pb="xl">
        <Group justify="space-between" align="center" mb="md">
          <Title order={2}>Translation Practice</Title>
          <Group gap="xs">
            <Box style={{ position: 'relative' }}>
              <Select
                data={directionOptions as unknown as { label: string; value: string }[]}
                value={direction}
                onChange={v => {
                  const newDir = (v as TranslationDirection) || 'learning_to_en';
                  setDirection(newDir);
                  if (newDir !== 'random') {
                    setActualDirection(newDir);
                  }
                  setIsDirectionFocused(false);
                  setIsDirectionDropdownOpen(false);
                }}
              onFocus={() => setIsDirectionFocused(true)}
              onBlur={() => {
                setIsDirectionFocused(false);
                setIsDirectionDropdownOpen(false);
              }}
              onDropdownOpen={() => {
                setIsDirectionFocused(true);
                setIsDirectionDropdownOpen(true);
              }}
              onDropdownClose={() => {
                setIsDirectionDropdownOpen(false);
                // Keep focused state briefly to allow Enter key to work
                setTimeout(() => setIsDirectionFocused(false), 100);
              }}
                aria-label="Translation direction"
                w={280}
              />
              <Badge
                size="xs"
                color="gray"
                variant="filled"
                radius="sm"
                style={{
                  position: 'absolute',
                  right: 8,
                  top: '50%',
                  transform: 'translateY(-50%)',
                  zIndex: 1,
                  pointerEvents: 'none',
                }}
              >
                D
              </Badge>
            </Box>
            <Button variant="light" loading={isGenerating} onClick={handleGenerate}>
              Generate with AI{' '}
              <Badge ml={6} size="xs" color="gray" variant="filled" radius="sm">
                A
              </Badge>
            </Button>
            <Tooltip
              label={
                isFromExistingDisabled
                  ? 'From existing content is only available when translating from your learning language to English, not from English to your learning language.'
                  : 'Load a sentence from existing content (stories, vocabulary, reading comprehension)'
              }
              withArrow
              withinPortal={false}
            >
              <Button
                variant="light"
                loading={loadingExisting}
                onClick={handleFromExisting}
                disabled={isFromExistingDisabled}
              >
                From existing content{' '}
                <Badge ml={6} size="xs" color="gray" variant="filled" radius="sm">
                  E
                </Badge>
              </Button>
            </Tooltip>
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
              ref={topicInputRef}
              label="Optional topic"
              placeholder="e.g., travel, ordering food, work"
              value={topic}
              onChange={e => setTopic(e.currentTarget.value)}
              onFocus={() => {
                setIsInputFocused(true);
                setTabCycleIndex(0);
              }}
              onBlur={() => setIsInputFocused(false)}
              autosize
              minRows={1}
              maxRows={3}
            />
            <Divider />
            <Group justify="space-between" align="center">
              <Text fw={600}>Text to translate</Text>
              {currentSentence && (
                <TTSButton
                  getText={() => currentSentence?.sentence_text || ''}
                  getVoice={() => {
                    // Determine language based on translation direction
                    // If translating from English to learning language, text is in English
                    // If translating from learning language to English, text is in learning language
                    const textLanguage =
                      actualDirection === 'en_to_learning' ? 'english' : learningLanguage;

                    // For English text, always use the default English voice
                    // (don't use saved preference which might be for learning language)
                    if (textLanguage === 'english') {
                      return defaultVoiceForLanguage('english') || undefined;
                    }

                    // For learning language text, prefer user setting, fall back to default voice
                    const saved = (learningPrefs?.tts_voice || '').trim();
                    if (saved) return saved;

                    const voice = defaultVoiceForLanguage(learningLanguage);
                    return voice || undefined;
                  }}
                  size="sm"
                  ariaLabel="Play text to translate"
                />
              )}
            </Group>
            <Paper withBorder p="md" bg="gray.0">
              {currentSentence ? (
                <Stack gap={6}>
                  <Text>{currentSentence.sentence_text}</Text>
                  {(() => {
                    const sourceType = currentSentence.source_type?.toLowerCase();
                    if (sourceType === 'ai_generated') {
                      return (
                        <Group gap="xs" align="center">
                          <Badge size="xs" variant="light" color="blue">
                            AI Generated
                          </Badge>
                        </Group>
                      );
                    }
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
              ref={translationInputRef}
              label="Your translation"
              placeholder="Write your translation here"
              value={answer}
              onChange={e => setAnswer(e.currentTarget.value)}
              onFocus={() => {
                setIsInputFocused(true);
                setTabCycleIndex(1);
              }}
              onBlur={() => setIsInputFocused(false)}
              autosize
              minRows={3}
            />
            <Group justify="flex-end">
              {lastGenerationType && (
                <Tooltip
                  label={`Generate another sentence using the same method (${lastGenerationType === 'ai' ? 'AI generation' : 'from existing content'})`}
                  withArrow
                  withinPortal={false}
                >
                  <Button
                    variant="light"
                    onClick={handleNextSameType}
                    disabled={!currentSentence || isGenerating || loadingExisting}
                    loading={isGenerating || loadingExisting}
                  >
                    Generate another{' '}
                    <Badge ml={6} size="xs" color="gray" variant="filled" radius="sm">
                      N
                    </Badge>
                  </Button>
                </Tooltip>
              )}
              <Button onClick={handleSubmit} loading={isSubmitting} disabled={!currentSentence}>
                Submit for feedback{' '}
                <Badge ml={6} size="xs" color="gray" variant="filled" radius="sm">
                  ⌘↵
                </Badge>
              </Button>
            </Group>
          </Stack>
        </Card>

        {/* AI Feedback section placed directly below translation card */}
        {feedback && (
          <Card withBorder ref={feedbackCardRef}>
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
              <Paper p="sm" withBorder>
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  components={{
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    p: ({ children }: any) => (
                      <Box mb="md" component="p">
                        {children}
                      </Box>
                    ),
                  }}
                >
                  {feedback.text}
                </ReactMarkdown>
              </Paper>
            </Stack>
          </Card>
        )}

        <Stack>
        <Card withBorder ref={historyCardRef}>
          <Stack gap="xs">
            <Group justify="space-between" align="center">
              <Title order={5}>History</Title>
              <Group gap="xs">
                <Text size="xs" c="dimmed">
                  {history?.total !== undefined
                    ? `Showing ${historyOffset + 1}-${Math.min(historyOffset + sessions.length, history.total)} of ${history.total}${historySearch.trim() ? ' (filtered)' : ''}`
                    : 'Loading...'}
                </Text>
              </Group>
            </Group>
            <TextInput
              ref={historySearchInputRef}
              placeholder="Search history (original, your translation, feedback, direction)"
              value={historySearch}
              onChange={e => setHistorySearch(e.currentTarget.value)}
              onFocus={() => setIsInputFocused(true)}
              onBlur={() => setIsInputFocused(false)}
              rightSection={
                <Badge size="xs" color="gray" variant="filled" radius="sm" style={{ pointerEvents: 'none' }}>
                  H
                </Badge>
              }
            />
            <ScrollArea
              style={{ height: '60vh' }}
              viewportRef={historyViewportRef}
            >
              {(sessions.length ?? 0) > 0 ? (
                <Accordion variant="separated">
                  {sessions.map(s => (
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
                            <Badge variant="light">
                              {s.translation_direction === 'en_to_learning'
                                ? `English → ${languageDisplay}`
                                : s.translation_direction === 'learning_to_en'
                                  ? `${languageDisplay} → English`
                                  : s.translation_direction.replaceAll('_', ' ')}
                            </Badge>
                            {s.ai_score != null ? (
                              <Badge
                                color={s.ai_score >= 4 ? 'green' : s.ai_score >= 3 ? 'yellow' : 'red'}
                                variant="light"
                              >
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
                            <ReactMarkdown
                              remarkPlugins={[remarkGfm]}
                              components={{
                                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                p: ({ children }: any) => (
                                  <Box mb="md" component="p">
                                    {children}
                                  </Box>
                                ),
                              }}
                            >
                              {s.ai_feedback}
                            </ReactMarkdown>
                          </Paper>
                        </Stack>
                      </Accordion.Panel>
                    </Accordion.Item>
                  ))}
                </Accordion>
              ) : (
                <Text size="sm" c="dimmed">
                  {historySearch.trim() ? 'No results found.' : 'No practice yet. Submit a translation to see history here.'}
                </Text>
              )}
            </ScrollArea>
            {history && history.total > 0 && (
              <Group justify="space-between" align="center" mt="xs">
                <Button
                  variant="light"
                  size="sm"
                  onClick={handlePrevPage}
                  disabled={!hasPrevPage}
                >
                  ← Previous
                </Button>
                <Text size="sm" c="dimmed">
                  Page {currentPage} of {totalPages}
                </Text>
                <Button
                  variant="light"
                  size="sm"
                  onClick={handleNextPage}
                  disabled={!hasNextPage}
                >
                  Next →
                </Button>
              </Group>
            )}
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

    {/* Scroll to top indicator at bottom */}
    <Group justify="center" style={{ marginTop: 8, marginBottom: 8 }}>
      <Badge size="xs" color="gray" variant="filled" radius="sm" style={{ opacity: 0.85, marginRight: 1 }}>
        ↑
      </Badge>
      <Badge size="xs" color="gray" variant="filled" radius="sm">
        T
      </Badge>
    </Group>

    {/* Keyboard Shortcuts Panel */}
    <Box
      style={{
        position: 'fixed',
        right: 0,
        top: '50%',
        transform: 'translateY(-50%)',
        zIndex: 1000,
        pointerEvents: 'none',
      }}
    >
      <Group gap={0} align="stretch" style={{ pointerEvents: 'auto' }}>
        <Button
          onClick={() => setShortcutsExpanded(!shortcutsExpanded)}
          variant="subtle"
          size="sm"
          p={8}
          style={{
            borderRadius: '8px 0 0 8px',
            border: '1px solid var(--mantine-color-default-border)',
            borderRight: 'none',
            background: 'var(--mantine-color-body)',
            minWidth: 'auto',
            height: 'auto',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
          title={isExpanded ? 'Collapse keyboard shortcuts' : 'Expand keyboard shortcuts'}
        >
          {isExpanded ? <IconChevronRight size={16} /> : <IconChevronLeft size={16} />}
        </Button>

        <Transition mounted={isExpanded} transition="slide-left" duration={200} timingFunction="ease">
          {styles => (
            <Box
              style={{
                ...styles,
                borderRadius: '0 8px 8px 0',
                border: '1px solid var(--mantine-color-default-border)',
                background: 'var(--mantine-color-body)',
                padding: 'var(--mantine-spacing-sm)',
                maxWidth: '280px',
                minWidth: '240px',
              }}
            >
              <Stack gap="xs">
                <Group gap="xs" align="center">
                  <IconKeyboard size={16} />
                  <Text size="sm" fw={500}>
                    Keyboard Shortcuts
                  </Text>
                </Group>

                {/* Expand/Collapse shortcuts info */}
                <Group gap="xs" align="center">
                  <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                    {'<'}
                  </Badge>
                  <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                    {'>'}
                  </Badge>
                  <Text size="xs" c="dimmed">
                    Expand / Collapse panel
                  </Text>
                </Group>

                <Stack gap="xs">
                  {isInputFocused || isDirectionFocused ? (
                    <>
                      <Group gap="xs" align="center">
                        <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                          Esc
                        </Badge>
                        <Text size="xs" c="dimmed">
                          Reset focus / enable hotkeys
                        </Text>
                      </Group>
                      {(isInputFocused || isDirectionFocused) && (
                        <Group gap="xs" align="center">
                          <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                            ⌘↵
                          </Badge>
                          <Text size="xs" c="dimmed">
                            Submit for feedback
                          </Text>
                        </Group>
                      )}
                    </>
                  ) : (
                    <>
                      <Group gap="xs" align="center">
                        <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                          A
                        </Badge>
                        <Text size="xs" c="dimmed">
                          Generate with AI
                        </Text>
                      </Group>
                      <Group gap="xs" align="center">
                        <Badge
                          size="sm"
                          variant="light"
                          style={{ minWidth: '32px', opacity: isFromExistingDisabled ? 0.5 : 1 }}
                        >
                          E
                        </Badge>
                        <Text size="xs" c={isFromExistingDisabled ? 'dimmed' : 'dimmed'} style={{ opacity: isFromExistingDisabled ? 0.5 : 1 }}>
                          From existing content
                          {isFromExistingDisabled && ' (disabled when translating from English)'}
                        </Text>
                      </Group>
                      <Group gap="xs" align="center">
                        <Badge
                          size="sm"
                          variant="light"
                          style={{ minWidth: '32px', opacity: lastGenerationType ? 1 : 0.5 }}
                        >
                          N
                        </Badge>
                        <Text size="xs" c="dimmed" style={{ opacity: lastGenerationType ? 1 : 0.5 }}>
                          {lastGenerationType
                            ? `Generate another (${lastGenerationType === 'ai' ? 'AI' : 'existing'})`
                            : 'Generate another (available after generating a sentence)'}
                        </Text>
                      </Group>
                      <Group gap="xs" align="center">
                        <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                          Tab
                        </Badge>
                        <Text size="xs" c="dimmed">
                          Cycle: topic ↔ translation
                        </Text>
                      </Group>
                      <Group gap="xs" align="center">
                        <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                          ⌘↵
                        </Badge>
                        <Text size="xs" c="dimmed">
                          Submit for feedback
                        </Text>
                      </Group>
                      <Group gap="xs" align="center">
                        <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                          Esc
                        </Badge>
                        <Text size="xs" c="dimmed">
                          Deselect / blur inputs
                        </Text>
                      </Group>
                      <Group gap="xs" align="center">
                        <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                          T
                        </Badge>
                        <Text size="xs" c="dimmed">
                          Scroll to top
                        </Text>
                      </Group>
                      <Group gap="xs" align="center">
                        <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                          H
                        </Badge>
                        <Text size="xs" c="dimmed">
                          Scroll to history
                        </Text>
                      </Group>
                      <Group gap="xs" align="center">
                        <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                          D
                        </Badge>
                        <Text size="xs" c="dimmed">
                          Focus direction dropdown
                        </Text>
                      </Group>
                      {isDirectionFocused && (
                        <>
                          <Group gap="xs" align="center">
                            <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                              ↑↓
                            </Badge>
                            <Text size="xs" c="dimmed">
                              Navigate options
                            </Text>
                          </Group>
                          <Group gap="xs" align="center">
                            <Badge size="sm" variant="light" style={{ minWidth: '32px' }}>
                              ↵
                            </Badge>
                            <Text size="xs" c="dimmed">
                              Select option
                            </Text>
                          </Group>
                        </>
                      )}
                    </>
                  )}
                </Stack>
              </Stack>
            </Box>
          )}
        </Transition>
      </Group>
    </Box>
    </Box>
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
