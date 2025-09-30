import React, { useState, useMemo } from 'react';
import {
  Check,
  X,
  Eye,
  EyeOff,
  Lightbulb,
  ChevronRight,
  BookOpen,
  Volume2,
  VolumeX,
} from 'lucide-react';
import { Question, AnswerResponse as Feedback } from '../api/api';
import * as Api from '../api/api';
import { defaultVoiceForLanguage } from '../utils/tts';
import logger from '../utils/logger';
import { useAuth } from '../hooks/useAuth';
import { useTTS } from '../hooks/useTTS';
import {
  usePostV1QuizQuestionIdReport,
  usePostV1QuizQuestionIdMarkKnown,
  useGetV1DailyHistoryQuestionId,
  ErrorResponse,
} from '../api/api';
import { showNotificationWithClean } from '../notifications';
import { QuestionHistoryChart } from './QuestionHistoryChart';
import {
  Stack,
  Group,
  Radio,
  Button,
  Paper,
  Text,
  Badge,
  Alert,
  Title,
  Transition,
  Box,
  Modal,
  Tooltip,
  Textarea,
  ActionIcon,
  LoadingOverlay,
} from '@mantine/core';
import { useElementSize } from '@mantine/hooks';
import { useHotkeys } from 'react-hotkeys-hook';
// Tabler icons package provides named exports under '@tabler/icons-react' in this repo's setup.
// Keep the import but fall back to a lightweight local mapping when types are missing.
import * as TablerIcons from '@tabler/icons-react';

const tablerIconMap = TablerIcons as unknown as Record<
  string,
  React.ComponentType<React.SVGProps<SVGSVGElement>>
>;
type IconProps = React.SVGProps<SVGSVGElement> & { size?: number };
const IconMoodSad: React.ComponentType<IconProps> =
  (tablerIconMap.IconMoodSad as unknown as React.ComponentType<IconProps>) ||
  ((props: IconProps) => {
    const { size, ...rest } = props;
    const s = size ?? 16;
    return (
      <svg
        xmlns='http://www.w3.org/2000/svg'
        viewBox='0 0 24 24'
        width={s}
        height={s}
        {...rest}
      />
    );
  });
const IconMoodNeutral: React.ComponentType<IconProps> =
  (tablerIconMap.IconMoodNeutral as unknown as React.ComponentType<IconProps>) ||
  ((props: IconProps) => {
    const { size, ...rest } = props;
    const s = size ?? 16;
    return (
      <svg
        xmlns='http://www.w3.org/2000/svg'
        viewBox='0 0 24 24'
        width={s}
        height={s}
        {...rest}
      />
    );
  });
const IconMoodSmile: React.ComponentType<IconProps> =
  (tablerIconMap.IconMoodSmile as unknown as React.ComponentType<IconProps>) ||
  ((props: IconProps) => {
    const { size, ...rest } = props;
    const s = size ?? 16;
    return (
      <svg
        xmlns='http://www.w3.org/2000/svg'
        viewBox='0 0 24 24'
        width={s}
        height={s}
        {...rest}
      />
    );
  });
const IconMoodHappy: React.ComponentType<IconProps> =
  (tablerIconMap.IconMoodHappy as unknown as React.ComponentType<IconProps>) ||
  ((props: IconProps) => {
    const { size, ...rest } = props;
    const s = size ?? 16;
    return (
      <svg
        xmlns='http://www.w3.org/2000/svg'
        viewBox='0 0 24 24'
        width={s}
        height={s}
        {...rest}
      />
    );
  });
const IconMoodCry: React.ComponentType<IconProps> =
  (tablerIconMap.IconMoodCry as unknown as React.ComponentType<IconProps>) ||
  ((props: IconProps) => {
    const { size, ...rest } = props;
    const s = size ?? 16;
    return (
      <svg
        xmlns='http://www.w3.org/2000/svg'
        viewBox='0 0 24 24'
        width={s}
        height={s}
        {...rest}
      />
    );
  });

export interface QuestionCardProps {
  question: Question;
  onAnswer: (questionId: number, answer: string) => Promise<Feedback>;
  onNext: () => void;
  feedback?: Feedback | null;
  selectedAnswer?: number | null;
  // The question id for which selectedAnswer is valid.
  selectedAnswerQuestionId?: number | null;
  // Optional scope id for isolating radio group instances (e.g., daily item id)
  groupScopeId?: number | string;
  onAnswerSelect?: (index: number) => void;
  showExplanation: boolean;
  setShowExplanation: React.Dispatch<React.SetStateAction<boolean>>;
  onMarkKnownModalChange?: (isOpen: boolean) => void;
  onReportModalChange?: (isOpen: boolean) => void;
  onReportTextareaFocusChange?: (isFocused: boolean) => void;
  isLastQuestion?: boolean;
  isReadOnly?: boolean;
  onShuffledOptionsChange?: (len: number) => void;
  prebuffering?: boolean;
  prebufferingProgress?: number; // 0..1
}

export type QuestionCardHandle = {
  openReport: () => void;
  openMarkKnown: () => void;
  toggleTTS: () => void;
};

// Debug logger (removed; kept as comment for quick re-enable)
// const debugSelection = (..._args: unknown[]) => {};

// Utility to highlight the target word in the sentence
function highlightTargetWord(sentence: string, target: string) {
  if (!target) return sentence;
  // Use regex to match the word boundary (case-insensitive)
  const regex = new RegExp(
    `\\b${target.replace(/[.*+?^${}()|[\\]\\]/g, '\\$&')}\\b`,
    'gi'
  );
  const parts = sentence.split(regex);
  const matches = sentence.match(regex);
  if (!matches) return sentence;
  const result: React.ReactNode[] = [];
  for (let i = 0; i < parts.length; i++) {
    result.push(parts[i]);
    if (i < matches.length) {
      result.push(
        <strong key={i} style={{ color: '#1976d2', fontWeight: 700 }}>
          {matches[i]}
        </strong>
      );
    }
  }
  return result;
}

const QuestionCard = React.forwardRef<QuestionCardHandle, QuestionCardProps>(
  (
    {
      question,
      onAnswer,
      onNext,
      feedback,
      selectedAnswer,
      selectedAnswerQuestionId,
      groupScopeId,
      onAnswerSelect,
      showExplanation,
      setShowExplanation,
      onMarkKnownModalChange,
      onReportModalChange,
      onReportTextareaFocusChange,
      isLastQuestion = false,
      isReadOnly = false,
      onShuffledOptionsChange,
    },
    ref
  ) => {
    const { ref: bottomBarRef, height: bottomBarHeight } = useElementSize();
    const [isSubmitted, setIsSubmitted] = useState(!!feedback);
    const [isLoading, setIsLoading] = useState(false);
    const [localFeedback, setLocalFeedback] = useState<Feedback | null>(
      feedback || null
    );
    const [isReported, setIsReported] = useState(false);
    const [showMarkKnownModal, setShowMarkKnownModal] = useState(false);
    const [showReportModal, setShowReportModal] = useState(false);
    const [confidenceLevel, setConfidenceLevel] = useState<number | null>(null);
    const [isMarkingKnown, setIsMarkingKnown] = useState(false);
    const [reportReason, setReportReason] = useState('');
    const [isReporting, setIsReporting] = useState(false);
    const {
      isLoading: isTTSLoading,
      isPlaying: isTTSPlaying,
      playTTS,
      stopTTS,
      // prebufferTTS intentionally unused in this component — prebuffering is
      // orchestrated at page-level to avoid duplicate calls.
      isBuffering,
      bufferingProgress,
    } = useTTS();
    const { isAuthenticated } = useAuth();
    React.useImperativeHandle(ref, () => ({
      openReport: () => setShowReportModal(true),
      openMarkKnown: () => setShowMarkKnownModal(true),
      toggleTTS: () => {
        // Only apply to reading comprehension with a passage
        if (question.type !== 'reading_comprehension') return;
        const passage = question.content?.passage || '';
        if (!passage) return;
        if (isTTSPlaying || isTTSLoading) {
          handleTTSStop();
        } else {
          handleTTSPlay(passage);
        }
      },
    }));

    // Create shuffled options and mapping when question changes
    const { shuffledOptions, shuffledToOriginalMap } = useMemo(() => {
      if (!question.content?.options || question.content.options.length === 0) {
        return {
          shuffledOptions: [],
          shuffledToOriginalMap: new Map(),
        };
      }

      // Create a copy of options with their original indices
      const optionsWithIndices = question.content.options.map(
        (option, index) => ({
          option,
          originalIndex: index,
        })
      );

      // Use deterministic shuffling based on question ID to ensure consistency
      const shuffled = [...optionsWithIndices];
      const seed = question.id || 0; // Use question ID as seed for deterministic shuffling

      // Simple deterministic shuffle using the seed
      for (let i = shuffled.length - 1; i > 0; i--) {
        const j = (seed + i) % (i + 1);
        [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]];
      }

      // Create mapping arrays
      const shuffledToOriginalMap = new Map<number, number>();
      const shuffledOptions: string[] = [];

      shuffled.forEach((item, shuffledIndex) => {
        shuffledToOriginalMap.set(shuffledIndex, item.originalIndex);
        shuffledOptions.push(item.option);
      });

      // debugSelection('shuffle', { questionId: question.id, options: question.content?.options, shuffledOptions });

      return {
        shuffledOptions,
        shuffledToOriginalMap,
      };
    }, [question.id, question.content?.options]); // Include question.id in dependencies

    // Build reverse map: original index -> shuffled index
    const originalToShuffledMap = useMemo(() => {
      const map = new Map<number, number>();
      shuffledToOriginalMap.forEach((originalIndex, shuffledIndex) => {
        map.set(originalIndex, shuffledIndex);
      });
      return map;
    }, [shuffledToOriginalMap]);

    // Notify parent of shuffledOptions length
    React.useEffect(() => {
      if (onShuffledOptionsChange) {
        onShuffledOptionsChange(shuffledOptions.length);
      }
    }, [shuffledOptions.length, onShuffledOptionsChange]);

    // Reset local state when question changes
    React.useEffect(() => {
      setIsSubmitted(!!feedback);
      setLocalFeedback(feedback || null);
      // debugSelection('reset-on-question-change', { questionId: question.id, hasFeedback: !!feedback });
    }, [question.id, feedback]);

    // Notify parent when modal state changes
    React.useEffect(() => {
      onMarkKnownModalChange?.(showMarkKnownModal);
    }, [showMarkKnownModal, onMarkKnownModalChange]);

    React.useEffect(() => {
      onReportModalChange?.(showReportModal);
    }, [showReportModal, onReportModalChange]);

    // Keyboard shortcuts for confidence levels
    useHotkeys(
      ['1', '2', '3', '4', '5'],
      event => {
        if (showMarkKnownModal) {
          event.preventDefault();
          event.stopPropagation();
          const level = parseInt(event.key);
          setConfidenceLevel(level);
          return false;
        }
      },
      {
        enableOnFormTags: true,
        preventDefault: true,
        enabled: showMarkKnownModal,
      }
    );

    // Escape to close modal or stop TTS
    useHotkeys(
      'escape',
      event => {
        if (isTTSPlaying || isTTSLoading) {
          event.preventDefault();
          event.stopPropagation();
          handleTTSStop();
          return false;
        }
        if (showMarkKnownModal) {
          event.preventDefault();
          event.stopPropagation();
          setShowMarkKnownModal(false);
          setConfidenceLevel(null);
          return false;
        }
        if (showReportModal) {
          event.preventDefault();
          event.stopPropagation();
          setShowReportModal(false);
          setReportReason('');
          return false;
        }
      },
      { enableOnFormTags: true, preventDefault: true }
    );

    // Enter to submit
    useHotkeys(
      'enter',
      event => {
        if (showMarkKnownModal && confidenceLevel) {
          event.preventDefault();
          event.stopPropagation();
          handleMarkAsKnown();
          return false;
        }
        if (showReportModal) {
          event.preventDefault();
          event.stopPropagation();
          handleSubmitReport();
          return false;
        }
      },
      { enableOnFormTags: false, preventDefault: true }
    );

    // 'i' key to focus text area in report modal
    useHotkeys(
      'i',
      event => {
        if (showReportModal) {
          event.preventDefault();
          event.stopPropagation();
          const textarea = document.getElementById(
            'report-reason-textarea'
          ) as HTMLTextAreaElement;
          if (textarea) {
            textarea.focus();
          }
          return false;
        }
      },
      { enableOnFormTags: false, preventDefault: true }
    );

    const reportMutation = usePostV1QuizQuestionIdReport({
      mutation: {
        onSuccess: () => {
          setIsReported(true);
          setShowReportModal(false);
          setReportReason('');
          showNotificationWithClean({
            title: 'Success',
            message:
              'Question reported successfully. Thank you for your feedback!',
            color: 'green',
          });
        },
        onError: error => {
          showNotificationWithClean({
            title: 'Error',
            message: error?.error || 'Failed to report question.',
            color: 'red',
          });
        },
      },
    });

    const markKnownMutation = usePostV1QuizQuestionIdMarkKnown({
      mutation: {
        onSuccess: () => {
          setShowMarkKnownModal(false);
          const confidence = confidenceLevel;
          setConfidenceLevel(null);

          // Message based on confidence level reflecting scheduling logic
          let message = 'Preference saved.';
          if (confidence === 1) {
            message =
              'Saved with low confidence. You will see this question more often.';
          } else if (confidence === 2) {
            message =
              'Saved with some confidence. You will see this question a bit more often.';
          } else if (confidence === 3) {
            message =
              'Saved with neutral confidence. No change to how often you will see this question.';
          } else if (confidence === 4) {
            message =
              'Saved with high confidence. You will see this question less often.';
          } else if (confidence === 5) {
            message =
              'Saved with complete confidence. You will rarely see this question.';
          }

          showNotificationWithClean({
            title: 'Success',
            message,
            color: 'green',
          });
        },
        onError: (error: ErrorResponse) => {
          showNotificationWithClean({
            title: 'Error',
            message: error?.error || 'Failed to mark question as known.',
            color: 'red',
          });
        },
      },
    });

    // Fetch question history
    const { data: historyData, isLoading: isHistoryLoading } =
      useGetV1DailyHistoryQuestionId(question.id);

    const handleReport = async () => {
      if (isReported || reportMutation.isPending || !question.id) return;

      if (!isAuthenticated) {
        showNotificationWithClean({
          title: 'Error',
          message: 'You must be logged in to report a question.',
          color: 'red',
        });
        return;
      }

      // Show the report modal instead of directly reporting
      setShowReportModal(true);
    };

    const handleSubmitReport = async () => {
      if (!question.id) return;

      setIsReporting(true);
      try {
        reportMutation.mutate({
          id: question.id,
          data: { report_reason: reportReason },
        });
      } finally {
        setIsReporting(false);
      }
    };

    const handleMarkAsKnown = async () => {
      if (!question.id || !confidenceLevel) return;

      setIsMarkingKnown(true);
      try {
        markKnownMutation.mutate({
          id: question.id,
          data: { confidence_level: confidenceLevel },
        });
      } finally {
        setIsMarkingKnown(false);
      }
    };

    const handleSubmit = async () => {
      // Use selectedAnswer state as the ONLY source of truth
      const selectedValue = selectedAnswer;

      // Validate the selected value
      if (selectedValue === null || selectedValue === undefined || !question.id)
        return;

      // Additional validation: ensure the selected value is within bounds
      const maxIndex = shuffledOptions ? shuffledOptions.length - 1 : 0;
      if (selectedValue < 0 || selectedValue > maxIndex) {
        logger.warn(
          `Invalid selected value: ${selectedValue}, max index: ${maxIndex}`
        );
        return;
      }

      // Convert shuffled index to original index
      const originalIndex = shuffledToOriginalMap.get(selectedValue);

      if (typeof originalIndex !== 'number') return;

      setIsLoading(true);
      try {
        // Send the original index instead of the answer text
        const feedbackData = await onAnswer(
          question.id,
          originalIndex.toString()
        );
        setLocalFeedback(feedbackData);
        setIsSubmitted(true);
      } catch (error) {
        logger.error('Failed to submit answer:', error);
      } finally {
        setIsLoading(false);
      }
    };

    const handleNextQuestion = () => {
      onNext();
    };

    // TTS Functions
    // Safely attempt to read optional learning preferences hook. Some tests
    // partially mock `../api/api` and do not provide this export, and their
    // mock throws on missing property access. Guard with try/catch.
    let userLearningPrefs: unknown | undefined = undefined;
    try {
      const maybeHook = (Api as unknown as Record<string, unknown>)[
        'useGetV1PreferencesLearning'
      ];
      if (typeof maybeHook === 'function') {
        const result = (maybeHook as () => unknown)();
        userLearningPrefs = (result as { data?: { tts_voice?: string } })?.data;
      }
    } catch {
      userLearningPrefs = undefined;
    }

    const handleTTSPlay = async (text: string) => {
      // Determine the best voice: user preference (if any) -> default for question.language -> fallback to 'echo'
      let preferredVoice: string | undefined;
      try {
        const saved = (
          userLearningPrefs as unknown as { tts_voice?: string } | undefined
        )?.tts_voice;
        if (saved && saved.trim()) {
          preferredVoice = saved.trim();
        }
        preferredVoice =
          preferredVoice || defaultVoiceForLanguage(question.language);
      } catch {
        preferredVoice = undefined;
      }
      // Ensure we always pass a sensible fallback voice to the TTS hook
      const finalVoice =
        preferredVoice ?? defaultVoiceForLanguage(question.language) ?? 'echo';
      await playTTS(text, finalVoice);
    };

    const handleTTSStop = () => {
      stopTTS();
    };

    // NOTE: prebuffering is handled at page level (QuestionPageBase) to avoid
    // duplicate prebuffer requests from both page and card. Do not start
    // prebuffer here.

    const currentFeedback = localFeedback || feedback;

    // Compute which shuffled index should be shown as selected in the UI.
    // Falls back to mapping backend's original indices when parent has not
    // provided a selectedAnswer (e.g., completed questions rendered read-only).
    const computedSelectedShuffledIndex = React.useMemo(() => {
      // Only use parent-selected value if it belongs to this question.
      // Treat `null` as "do not trust" to avoid leaking a previous question's selection.
      // Only accept a parent-provided selectedAnswer when the parent also
      // provides the matching question id. This prevents selection leakage
      // when switching between different pages (e.g., Daily -> Reading).
      if (
        selectedAnswer !== null &&
        selectedAnswer !== undefined &&
        selectedAnswerQuestionId === question.id
      ) {
        // debugSelection('use-parent-selected', { questionId: question.id, selectedAnswer, selectedAnswerQuestionId });
        return selectedAnswer;
      }
      if (
        isSubmitted &&
        typeof currentFeedback?.user_answer_index === 'number'
      ) {
        const mapped = originalToShuffledMap.get(
          currentFeedback.user_answer_index
        );
        if (typeof mapped === 'number') {
          // debugSelection('map-feedback-to-shuffled', { questionId: question.id, user_answer_index: currentFeedback.user_answer_index, mapped });
          return mapped;
        }
      }
      // debugSelection('no-selection', { questionId: question.id });
      return null;
    }, [
      selectedAnswer,
      selectedAnswerQuestionId,
      question.id,
      isSubmitted,
      currentFeedback?.user_answer_index,
      originalToShuffledMap,
    ]);

    // Ensure the selected radio matches the user's answer after submission,
    // even when options are shuffled. We convert backend indices (original order)
    // to the currently displayed shuffled indices and push the value to parent.
    React.useEffect(() => {
      if (!currentFeedback) return;
      if (!isSubmitted) return;
      if (typeof currentFeedback.user_answer_index !== 'number') return;

      const mappedShuffledIndex = originalToShuffledMap.get(
        currentFeedback.user_answer_index
      );

      if (
        typeof mappedShuffledIndex === 'number' &&
        mappedShuffledIndex !== selectedAnswer
      ) {
        // debugSelection('sync-parent-with-mapped', { questionId: question.id, mappedShuffledIndex, prevSelectedAnswer: selectedAnswer });
        onAnswerSelect?.(mappedShuffledIndex);
      }
    }, [
      isSubmitted,
      currentFeedback,
      originalToShuffledMap,
      selectedAnswer,
      onAnswerSelect,
    ]);

    const renderQuestion = () => {
      if (!shuffledOptions || shuffledOptions.length === 0) {
        return (
          <Text c='error'>Error: No options available for this question.</Text>
        );
      }

      return (
        <Radio.Group
          key={`${groupScopeId ?? 'q'}-${question.id}`}
          value={
            computedSelectedShuffledIndex !== null &&
            computedSelectedShuffledIndex !== undefined
              ? computedSelectedShuffledIndex.toString()
              : undefined
          }
          onChange={value => {
            // debugSelection('onChange', { questionId: question.id, value, isSubmitted, isReadOnly });
            if (!isSubmitted && !isReadOnly) onAnswerSelect?.(Number(value));
          }}
          name={`multiple-choice-${groupScopeId ?? 'q'}-${question.id}`}
          withAsterisk={false}
        >
          <Stack gap='sm'>
            {shuffledOptions.map((option, shuffledIndex) => {
              // Determine if this option is the user's answer or the correct answer
              const originalIndex = shuffledToOriginalMap.get(shuffledIndex);

              // Detect if this option is the one the user selected. Rely on the
              // computed shuffled index that already accounts for mapping from
              // feedback indices → displayed order instead of comparing against
              // the original index only. This makes the badge robust even when
              // tests (or backend) provide a shuffled index.
              const isUserAnswer =
                isSubmitted &&
                ((typeof currentFeedback?.user_answer_index === 'number' &&
                  originalIndex === currentFeedback.user_answer_index) ||
                  (computedSelectedShuffledIndex !== null &&
                    shuffledIndex === computedSelectedShuffledIndex));

              // Use correct_answer_index to find which shuffled option is correct
              const isCorrectAnswer =
                isSubmitted &&
                typeof currentFeedback?.correct_answer_index === 'number' &&
                originalIndex === currentFeedback.correct_answer_index;

              return (
                <Radio
                  key={`${question.id}-${shuffledIndex}-${option}`}
                  value={shuffledIndex.toString()}
                  label={
                    <div
                      style={{
                        color: isSubmitted ? '#000' : undefined,
                        display: 'flex',
                        alignItems: 'flex-start',
                        gap: 8,
                        whiteSpace: 'normal',
                        wordBreak: 'break-word',
                        overflowWrap: 'anywhere',
                        // Ensure long words wrap on small screens
                      }}
                    >
                      <div style={{ flex: '0 0 auto' }}>
                        <Badge
                          size='xs'
                          color='gray'
                          variant='outline'
                          radius='sm'
                          mr={6}
                        >
                          {shuffledIndex + 1}
                        </Badge>
                      </div>
                      <div style={{ flex: '1 1 auto', minWidth: 0 }}>
                        {/* option text and badges share the same inline container to aid test selectors */}
                        <div style={{ display: 'inline' }}>
                          {option}
                          {isUserAnswer && (
                            <Badge
                              ml={8}
                              size='xs'
                              color='blue'
                              variant='filled'
                              radius='sm'
                            >
                              Your answer
                            </Badge>
                          )}
                          {isCorrectAnswer && (
                            <Badge
                              ml={8}
                              size='xs'
                              color='green'
                              variant='filled'
                              radius='sm'
                            >
                              Correct answer
                            </Badge>
                          )}
                        </div>
                      </div>
                    </div>
                  }
                  disabled={isSubmitted}
                  data-testid={`option-${shuffledIndex}`}
                />
              );
            })}
          </Stack>
        </Radio.Group>
      );
    };

    // Always show all stats, defaulting to 0
    const shown =
      typeof question.total_responses === 'number'
        ? question.total_responses
        : 0;
    const correct =
      typeof question.correct_count === 'number' ? question.correct_count : 0;
    const wrong =
      typeof question.incorrect_count === 'number'
        ? question.incorrect_count
        : 0;

    return (
      <Box
        data-testid='question-card'
        style={{
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
          margin: 0,
          padding: 0,
          position: 'relative',
          minHeight: '400px', // Match the Card's min-height
        }}
      >
        {/* Content area (allow full height, natural overflow at page level) */}
        <Box
          style={{
            flex: '0 1 auto',
            overflow: 'visible',
            padding: '24px',
            paddingBottom: `${bottomBarHeight + 16}px`,
            display: 'flex',
            flexDirection: 'column',
            position: 'relative',
          }}
        >
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: '16px',
              flex: 1,
            }}
          >
            {/* Question text and context at the top (except reading comprehension) */}
            {question.type !== 'reading_comprehension' &&
              (question.type === 'vocabulary' &&
              question.content?.sentence &&
              question.content?.question ? (
                <>
                  <Title order={3} data-testid='question-content' mb={0}>
                    {highlightTargetWord(
                      question.content.sentence,
                      question.content.question
                    )}
                  </Title>
                  <Text
                    size='sm'
                    c='dimmed'
                    mt={-8}
                    mb={8}
                    style={{ fontWeight: 500 }}
                  >
                    What does <strong>{question.content.question}</strong> mean
                    in this context?
                  </Text>
                </>
              ) : (
                <Title order={3} data-testid='question-content' mb={0}>
                  {question.content?.question}
                </Title>
              ))}

            {/* Show passage for reading comprehension questions */}
            {question.type === 'reading_comprehension' &&
              question.content?.passage && (
                <Paper
                  p='lg'
                  bg='var(--mantine-color-body)'
                  radius='md'
                  withBorder
                  style={{ marginBottom: 8, position: 'relative' }}
                >
                  <LoadingOverlay
                    visible={isTTSLoading}
                    overlayProps={{ backgroundOpacity: 0.35, blur: 1 }}
                    zIndex={5}
                  />
                  <Box
                    style={{
                      position: 'absolute',
                      top: 12,
                      right: 12,
                      zIndex: 10,
                    }}
                  >
                    <Group gap={6} align='center'>
                      <Badge
                        size='xs'
                        color='gray'
                        variant='filled'
                        radius='sm'
                        title='Shortcut: P'
                      >
                        P
                      </Badge>
                      <Tooltip
                        label={
                          isTTSPlaying ? 'Stop audio' : 'Listen to passage'
                        }
                      >
                        <ActionIcon
                          size='md'
                          variant='subtle'
                          color={isTTSPlaying ? 'red' : 'blue'}
                          onClick={() => {
                            if (isTTSPlaying || isTTSLoading) {
                              handleTTSStop();
                            } else {
                              handleTTSPlay(question.content?.passage || '');
                            }
                          }}
                          disabled={false}
                          aria-label={
                            isTTSPlaying || isTTSLoading
                              ? 'Stop audio'
                              : 'Listen to passage'
                          }
                        >
                          {isTTSPlaying || isTTSLoading ? (
                            <VolumeX size={18} />
                          ) : (
                            <Volume2 size={18} />
                          )}
                        </ActionIcon>
                      </Tooltip>
                      {/* Small inline buffering indicator (progress bar) */}
                      <Box
                        ml={8}
                        style={{ display: 'inline-flex', alignItems: 'center' }}
                      >
                        {isBuffering ? (
                          <div
                            style={{
                              display: 'inline-flex',
                              alignItems: 'center',
                              gap: 8,
                            }}
                          >
                            <div style={{ width: 96 }}>
                              <div
                                style={{
                                  height: 6,
                                  background:
                                    'var(--mantine-color-default-border)',
                                  borderRadius: 6,
                                  overflow: 'hidden',
                                }}
                              >
                                <div
                                  style={{
                                    width: `${Math.round((bufferingProgress ?? 0) * 100)}%`,
                                    height: '100%',
                                    background:
                                      'linear-gradient(90deg, #4dabf7, #1971c2)',
                                    transition: 'width 160ms linear',
                                  }}
                                  data-testid='tts-buffer-progress'
                                />
                              </div>
                            </div>
                            <Text size='xs' c='dimmed'>
                              Buffering
                            </Text>
                          </div>
                        ) : null}
                      </Box>
                    </Group>
                  </Box>
                  <Text
                    size='lg'
                    style={{
                      whiteSpace: 'pre-line',
                      lineHeight: 1.8,
                      fontWeight: 500,
                      letterSpacing: 0.1,
                    }}
                    className='reading-passage-text'
                  >
                    {question.content.passage}
                  </Text>
                </Paper>
              )}

            {/* For reading comprehension, place the question after the passage */}
            {question.type === 'reading_comprehension' && (
              <Title order={3} data-testid='question-content' mb={0}>
                {question.content?.question}
              </Title>
            )}

            {/* Show hint for fill_blank questions */}
            {question.type === 'fill_blank' && question.content?.hint && (
              <Alert color='blue' icon={<Lightbulb size={18} />}>
                <Text size='sm'>
                  <strong>Hint:</strong> {question.content.hint}
                </Text>
              </Alert>
            )}

            {/* Answer options */}
            {renderQuestion()}

            {/* Inline feedback summary */}
            <Transition
              mounted={isSubmitted && !!currentFeedback}
              transition='slide-up'
              duration={400}
              timingFunction='ease'
            >
              {styles => (
                <div style={styles}>
                  {currentFeedback && (
                    <>
                      <Alert
                        color={currentFeedback.is_correct ? 'success' : 'error'}
                        icon={
                          currentFeedback.is_correct ? (
                            <Check size={20} />
                          ) : (
                            <X size={20} />
                          )
                        }
                        title={
                          currentFeedback.is_correct ? 'Correct!' : 'Incorrect'
                        }
                        withCloseButton={false}
                        style={
                          currentFeedback.is_correct
                            ? {
                                backgroundColor:
                                  'var(--mantine-color-green-light, #e6fcf5)',
                              }
                            : {
                                backgroundColor:
                                  'var(--mantine-color-red-light, #fff5f5)',
                              }
                        }
                      >
                        <Text size='sm'>
                          {currentFeedback.is_correct
                            ? 'Great job! You got it right.'
                            : "Don't worry, let's learn from this."}
                        </Text>
                        {currentFeedback?.explanation && (
                          <Button
                            mt='sm'
                            size='xs'
                            leftSection={<Lightbulb size={16} />}
                            rightSection={
                              <Group gap={4}>
                                {showExplanation ? (
                                  <EyeOff size={16} />
                                ) : (
                                  <Eye size={16} />
                                )}
                                <Badge
                                  ml={6}
                                  size='xs'
                                  color='gray'
                                  variant='filled'
                                  radius='sm'
                                  title='Shortcut: E'
                                >
                                  E
                                </Badge>
                              </Group>
                            }
                            variant={
                              currentFeedback.is_correct ? 'light' : 'outline'
                            }
                            color={
                              currentFeedback.is_correct ? 'success' : 'error'
                            }
                            onClick={() => setShowExplanation(v => !v)}
                          >
                            Explanation
                          </Button>
                        )}
                      </Alert>
                      <Transition
                        mounted={
                          showExplanation && !!currentFeedback?.explanation
                        }
                        transition='slide-down'
                        duration={300}
                        timingFunction='ease'
                      >
                        {explanationStyles => (
                          <Paper
                            mt='sm'
                            p='md'
                            radius='md'
                            shadow='xs'
                            withBorder
                            bg='var(--mantine-color-body)'
                            style={explanationStyles}
                          >
                            <Group mb='xs'>
                              <Lightbulb size={18} />
                              <Text fw={500}>Explanation</Text>
                            </Group>
                            <Text size='sm' style={{ whiteSpace: 'pre-line' }}>
                              {currentFeedback.explanation}
                            </Text>
                          </Paper>
                        )}
                      </Transition>
                    </>
                  )}
                </div>
              )}
            </Transition>

            {/* Difficulty Adjustment Notice */}
            <Transition
              mounted={isSubmitted && !!currentFeedback?.next_difficulty}
              transition='slide-up'
              duration={400}
              timingFunction='ease'
            >
              {styles => (
                <Alert
                  color='primary'
                  icon={<ChevronRight size={18} />}
                  style={styles}
                >
                  <Group>
                    <Text fw={600}>Difficulty Adjusted</Text>
                    <Badge color='primary' variant='light'>
                      {currentFeedback?.next_difficulty}
                    </Badge>
                  </Group>
                  <Text size='sm'>
                    Next question will be {currentFeedback?.next_difficulty}{' '}
                    level
                  </Text>
                </Alert>
              )}
            </Transition>
          </div>

          {/* Submit/Next button */}
          {!isReadOnly && (
            <Group justify='flex-end' mt='xl' mb='xl'>
              {!isSubmitted ? (
                <Button
                  onClick={handleSubmit}
                  disabled={
                    selectedAnswer === null ||
                    selectedAnswer === undefined ||
                    isLoading
                  }
                  loading={isLoading}
                  variant='filled'
                  data-testid='submit-button'
                >
                  Submit{' '}
                  <Badge
                    ml={6}
                    size='xs'
                    color='gray'
                    variant='filled'
                    radius='sm'
                  >
                    ↵
                  </Badge>
                </Button>
              ) : (
                <Button onClick={handleNextQuestion} variant='filled'>
                  {isLastQuestion ? 'Complete Questions' : 'Next Question'}{' '}
                  <Badge
                    ml={6}
                    size='xs'
                    color='gray'
                    variant='filled'
                    radius='sm'
                  >
                    ↵
                  </Badge>
                </Button>
              )}
            </Group>
          )}

          {/* Question history chart */}
          <QuestionHistoryChart
            history={historyData?.history || []}
            isLoading={isHistoryLoading}
            questionId={question.id}
          />
        </Box>

        {/* Fixed bottom row: report issue (left), stats (right) */}
        <Box
          ref={bottomBarRef}
          style={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            borderTop: '1px solid var(--mantine-color-default-border)',
            padding: '12px 24px',
            backgroundColor: 'var(--mantine-color-body)',
            borderBottomLeftRadius: '10px',
            borderBottomRightRadius: '10px',
          }}
        >
          <Group justify='space-between' gap={8}>
            <Group gap={8}>
              <Button
                onClick={handleReport}
                disabled={isReported || reportMutation.isPending}
                variant='subtle'
                color='gray'
                size='xs'
                data-testid='report-question-btn'
              >
                {isReported ? 'Reported' : 'Report issue with question'}{' '}
                <Badge
                  ml={4}
                  size='xs'
                  color='gray'
                  variant='filled'
                  radius='sm'
                >
                  R
                </Badge>
              </Button>
              <Button
                onClick={() => setShowMarkKnownModal(true)}
                variant='subtle'
                color='blue'
                size='xs'
                leftSection={<BookOpen size={14} />}
                data-testid='mark-known-btn'
              >
                Adjust question frequency{' '}
                <Badge
                  ml={4}
                  size='xs'
                  color='gray'
                  variant='filled'
                  radius='sm'
                >
                  K
                </Badge>
              </Button>
            </Group>
            <Group gap={8} align='center'>
              {/* Confidence Level Icon */}
              {question.confidence_level && (
                <Tooltip
                  label={`Confidence Level: ${question.confidence_level}/5`}
                  position='top'
                  withArrow
                  openDelay={200}
                  closeDelay={0}
                >
                  <div data-testid='confidence-icon-inline'>
                    {(() => {
                      const confidenceIcons = {
                        1: <IconMoodCry size={16} />,
                        2: <IconMoodSad size={16} />,
                        3: <IconMoodNeutral size={16} />,
                        4: <IconMoodSmile size={16} />,
                        5: <IconMoodHappy size={16} />,
                      };
                      return (
                        confidenceIcons[
                          question.confidence_level as keyof typeof confidenceIcons
                        ] || <IconMoodNeutral size={16} />
                      );
                    })()}
                  </div>
                </Tooltip>
              )}
              <Text c='dimmed' size='xs'>
                Shown: {shown} | Correct: {correct} | Wrong: {wrong}
              </Text>
            </Group>
          </Group>
        </Box>

        {/* Adjust Frequency / Confidence Modal */}
        <Modal
          opened={showMarkKnownModal}
          onClose={() => {
            setShowMarkKnownModal(false);
            setConfidenceLevel(null);
          }}
          title='Adjust Question Frequency'
          size='sm'
          closeOnClickOutside={false}
          closeOnEscape={false}
        >
          <Stack gap='md'>
            <Text size='sm' c='dimmed'>
              Choose how often you want to see this question in future quizzes:
              1–2 show it more, 3 no change, 4–5 show it less.
            </Text>

            <Text size='sm' fw={500}>
              How confident are you about this question?
            </Text>

            <Group gap='xs' justify='space-between'>
              {[
                {
                  level: 1,
                  label: 'Not very confident',
                  description: 'Low confidence',
                  icon: <IconMoodCry size={24} />,
                },
                {
                  level: 2,
                  label: 'Somewhat confident',
                  description: 'Some confidence',
                  icon: <IconMoodSad size={24} />,
                },
                {
                  level: 3,
                  label: 'Moderately confident',
                  description: 'Medium confidence',
                  icon: <IconMoodNeutral size={24} />,
                },
                {
                  level: 4,
                  label: 'Very confident',
                  description: 'High confidence',
                  icon: <IconMoodSmile size={24} />,
                },
                {
                  level: 5,
                  label: 'Extremely confident',
                  description: 'Complete confidence',
                  icon: <IconMoodHappy size={24} />,
                },
              ].map(({ level, label, icon }) => (
                <Tooltip
                  key={level}
                  label={label}
                  position='top'
                  withArrow
                  openDelay={200}
                  closeDelay={0}
                >
                  <Button
                    variant={confidenceLevel === level ? 'filled' : 'light'}
                    color={confidenceLevel === level ? 'teal' : 'gray'}
                    onClick={() => setConfidenceLevel(level)}
                    style={{
                      flex: 1,
                      minHeight: '60px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      position: 'relative',
                    }}
                    data-testid={`confidence-level-${level}`}
                  >
                    <div style={{ position: 'relative' }}>
                      <Badge
                        size='xl'
                        variant={confidenceLevel === level ? 'filled' : 'light'}
                        color={confidenceLevel === level ? 'teal' : 'gray'}
                        style={{
                          minWidth: '40px',
                          height: '40px',
                          textAlign: 'center',
                          fontSize: '20px',
                          fontWeight: 'bold',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          padding: '8px 12px',
                        }}
                      >
                        {icon}
                      </Badge>
                      <Badge
                        size='xs'
                        variant='filled'
                        color='gray'
                        style={{
                          position: 'absolute',
                          bottom: '-4px',
                          right: '-4px',
                          minWidth: '16px',
                          height: '16px',
                          fontSize: '10px',
                          fontWeight: 'bold',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          zIndex: 1,
                        }}
                      >
                        {level}
                      </Badge>
                    </div>
                  </Button>
                </Tooltip>
              ))}
            </Group>

            <Group justify='space-between' mt='md'>
              <Button
                variant='subtle'
                onClick={() => {
                  setShowMarkKnownModal(false);
                  setConfidenceLevel(null);
                }}
                data-testid='cancel-mark-known'
              >
                Cancel{' '}
                <Badge
                  ml={4}
                  size='xs'
                  color='gray'
                  variant='filled'
                  radius='sm'
                >
                  Esc
                </Badge>
              </Button>
              <Button
                onClick={handleMarkAsKnown}
                disabled={!confidenceLevel || isMarkingKnown}
                loading={isMarkingKnown}
                color='teal'
                data-testid='submit-mark-known'
              >
                Save{' '}
                <Badge
                  ml={4}
                  size='xs'
                  color='gray'
                  variant='filled'
                  radius='sm'
                >
                  ↵
                </Badge>
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Report Question Modal */}
        <Modal
          opened={showReportModal}
          onClose={() => {
            setShowReportModal(false);
            setReportReason('');
          }}
          title='Report Issue with Question'
          size='lg'
          closeOnClickOutside={false}
          closeOnEscape={false}
        >
          <Stack gap='md'>
            <Text size='sm' c='dimmed'>
              Please let us know what's wrong with this question. Your feedback
              helps us improve the quality of our content.
            </Text>

            <Box pos='relative'>
              <Textarea
                placeholder='Describe the issue (optional, max 512 characters)...'
                value={reportReason}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                  setReportReason(e.target.value)
                }
                onFocus={() => onReportTextareaFocusChange?.(true)}
                onBlur={() => onReportTextareaFocusChange?.(false)}
                maxLength={512}
                minRows={8}
                maxRows={12}
                data-testid='report-reason-input'
                id='report-reason-textarea'
              />
              <Badge
                size='xs'
                variant='light'
                color='gray'
                style={{
                  position: 'absolute',
                  top: '8px',
                  right: '8px',
                  zIndex: 1,
                  pointerEvents: 'none',
                }}
              >
                I
              </Badge>
            </Box>

            <Group
              justify='space-between'
              mt='md'
              style={{ flexWrap: 'nowrap' }}
            >
              <Button
                variant='subtle'
                onClick={() => {
                  setShowReportModal(false);
                  setReportReason('');
                }}
                data-testid='cancel-report'
                style={{ flexShrink: 0 }}
              >
                Cancel{' '}
                <Badge
                  ml={4}
                  size='xs'
                  color='gray'
                  variant='filled'
                  radius='sm'
                >
                  Esc
                </Badge>
              </Button>
              <Button
                onClick={handleSubmitReport}
                disabled={isReporting}
                loading={isReporting}
                color='red'
                data-testid='submit-report'
                style={{ flexShrink: 0 }}
              >
                Report Question{' '}
                <Badge
                  ml={4}
                  size='xs'
                  color='gray'
                  variant='filled'
                  radius='sm'
                >
                  ↵
                </Badge>
              </Button>
            </Group>
          </Stack>
        </Modal>
      </Box>
    );
  }
);

QuestionCard.displayName = 'QuestionCard';

export default QuestionCard;
