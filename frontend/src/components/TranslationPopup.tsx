import React, { useState, useEffect, useRef, useMemo } from 'react';
import {
  Paper,
  Text,
  Loader,
  Button,
  Group,
  Select,
  Stack,
  Tooltip,
  Portal,
  ActionIcon,
  Box,
} from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from '../contexts/TranslationContext';
import { TextSelection } from '../hooks/useTextSelection';
import * as TablerIcons from '@tabler/icons-react';
import {
  postV1Snippets,
  Question,
  useGetV1PreferencesLearning,
  useGetV1SettingsLanguages,
} from '../api/api';
import type { LanguageInfo } from '../api/api';
import { useTheme } from '../contexts/ThemeContext';
import { fontScaleMap } from '../theme/theme';
import TTSButton from './TTSButton';
import { defaultVoiceForLanguage } from '../utils/tts';

type TablerIconComponent = React.ComponentType<
  React.SVGProps<SVGSVGElement> & {
    size?: string | number;
    stroke?: string | number;
    title?: string;
  }
>;

const tablerIcons = TablerIcons as Record<string, TablerIconComponent>;
const IconX = tablerIcons.IconX;
const IconBookmark = tablerIcons.IconBookmark;
const IconCopy = tablerIcons.IconCopy;

// Type for story context when no question is available
interface StoryContext {
  story_id: number;
  section_id?: number;
}

interface TranslationPopupProps {
  selection: TextSelection;
  onClose: () => void;
  currentQuestion?: Question | StoryContext | null;
  // When true, saving requires a valid question id to be present
  requireQuestionId?: boolean;
}

export const TranslationPopup: React.FC<TranslationPopupProps> = ({
  selection,
  onClose,
  currentQuestion,
  requireQuestionId = false,
}) => {
  const queryClient = useQueryClient();
  const { fontSize } = useTheme();

  // Load saved language from localStorage or use browser language or default to 'en'
  const [targetLanguage, setTargetLanguage] = useState('en');
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [isSaved, setIsSaved] = useState(false);
  const [copySuccess, setCopySuccess] = useState<string | null>(null);
  const {
    translateText,
    translation,
    isLoading: translationLoading,
    error: translationError,
  } = useTranslation();
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();
  const { data: languagesData, isLoading: languagesLoading } =
    useGetV1SettingsLanguages();
  const popupRef = useRef<HTMLDivElement>(null);
  const mountedRef = useRef(true);
  const savedTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const copySuccessTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const scale = fontScaleMap[fontSize] ?? 1;
  const baseWidth = 320 * scale;
  const [viewportSize, setViewportSize] = useState(() => ({
    width: typeof window !== 'undefined' ? window.innerWidth : baseWidth,
    height: typeof window !== 'undefined' ? window.innerHeight : 600,
  }));

  useEffect(() => {
    const handleResize = () => {
      setViewportSize({
        width: window.innerWidth,
        height: window.innerHeight,
      });
    };

    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
    };
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return () => {};
    }

    const body = document.body;

    const scrollX =
      window.scrollX ||
      window.pageXOffset ||
      document.documentElement.scrollLeft ||
      0;
    const scrollY =
      window.scrollY ||
      window.pageYOffset ||
      document.documentElement.scrollTop ||
      0;

    const originalBodyOverflow = body.style.overflow;
    const originalBodyPaddingRight = body.style.paddingRight;

    const scrollbarWidth =
      window.innerWidth - document.documentElement.clientWidth;

    body.style.overflow = 'hidden';

    if (scrollbarWidth > 0) {
      body.style.paddingRight = `${scrollbarWidth}px`;
    }

    if (scrollX || scrollY) {
      window.scrollTo(scrollX, scrollY);
    }

    return () => {
      body.style.overflow = originalBodyOverflow;
      body.style.paddingRight = originalBodyPaddingRight;
      if (scrollX || scrollY) {
        window.scrollTo(scrollX, scrollY);
      }
    };
  }, []);

  const viewportWidth = Number.isFinite(viewportSize.width)
    ? viewportSize.width
    : baseWidth;
  const popupMargin = 10;
  const popupWidth = Math.min(
    baseWidth,
    Math.max(viewportWidth - popupMargin * 2, 200)
  );
  const resolvedPopupWidth =
    Number.isFinite(popupWidth) && popupWidth > 0 ? popupWidth : 320;
  // Helper function to convert language code to language name for TTS
  const codeToLanguageName = (code: string): string => {
    const mapping: Record<string, string> = {
      en: 'english',
      es: 'spanish',
      fr: 'french',
      de: 'german',
      it: 'italian',
      pt: 'portuguese',
      ru: 'russian',
      ja: 'japanese',
      ko: 'korean',
      zh: 'chinese',
      hi: 'hindi',
    };
    return mapping[code] || code;
  };

  const languagesByCode = useMemo(() => {
    const map = new Map<string, LanguageInfo>();
    if (!languagesData) {
      return map;
    }
    languagesData.forEach(lang => {
      map.set(lang.code, lang);
    });
    return map;
  }, [languagesData]);

  const savedVoice = useMemo(
    () => (userLearningPrefs?.tts_voice || '').trim(),
    [userLearningPrefs?.tts_voice]
  );

  const getLanguageLabel = (code?: string): string | undefined => {
    if (!code) return undefined;
    const entry = languagesByCode.get(code);
    const rawName = entry?.name || codeToLanguageName(code);
    if (!rawName) return undefined;
    return rawName.charAt(0).toUpperCase() + rawName.slice(1);
  };

  const getPreferredVoice = (code?: string): string | undefined => {
    if (savedVoice) {
      return savedVoice;
    }

    if (!code) {
      return undefined;
    }

    const entry = languagesByCode.get(code);
    if (entry) {
      const fromEntry = defaultVoiceForLanguage(entry);
      if (fromEntry) {
        return fromEntry;
      }
    }

    const languageName = codeToLanguageName(code);
    return defaultVoiceForLanguage(languageName) || undefined;
  };

  const getMetadata = (variant: 'original' | 'translated', code?: string) => {
    const languageLabel = getLanguageLabel(code);
    const baseTitle = variant === 'original' ? 'Original text' : 'Translation';

    return {
      title: languageLabel ? `${baseTitle} — ${languageLabel}` : baseTitle,
      language: languageLabel,
    };
  };

  // Language options for the dropdown - dynamically generated from API
  const languageOptions = languagesData
    ? languagesData.map(lang => ({
        value: lang.code,
        label: lang.name.charAt(0).toUpperCase() + lang.name.slice(1),
      }))
    : [];

  // Track mounted state
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      // Clean up any pending timeouts
      if (savedTimeoutRef.current) {
        clearTimeout(savedTimeoutRef.current);
        savedTimeoutRef.current = null;
      }
      if (copySuccessTimeoutRef.current) {
        clearTimeout(copySuccessTimeoutRef.current);
        copySuccessTimeoutRef.current = null;
      }
    };
  }, []);

  // Initialize targetLanguage from localStorage or browser language once languages are loaded
  useEffect(() => {
    if (!languagesLoading && languagesData && languagesData.length > 0) {
      const availableCodes = languagesData.map(lang => lang.code);

      // Try to load saved language from localStorage
      const saved = localStorage.getItem('quiz-translation-target-lang');
      if (saved && availableCodes.includes(saved)) {
        setTargetLanguage(saved);
        return;
      }

      // Try to detect user's preferred language from browser
      const browserLang = navigator.language.split('-')[0];
      if (availableCodes.includes(browserLang)) {
        setTargetLanguage(browserLang);
        return;
      }

      // Default to 'en' if available, otherwise first language
      if (availableCodes.includes('en')) {
        setTargetLanguage('en');
      } else {
        setTargetLanguage(availableCodes[0]);
      }
    }
  }, [languagesLoading, languagesData]);

  // Translate text when selection or target language changes
  useEffect(() => {
    const performTranslation = async () => {
      if (selection?.text && selection.text.length > 1) {
        try {
          await translateText(selection.text, targetLanguage);
        } catch (err) {
          console.error('Translation failed:', err);
          // Error is already handled in context, just log here
        }
      }
    };

    // Reduced debounce since selection hook already has delay
    const timeoutId = setTimeout(performTranslation, 100);

    return () => clearTimeout(timeoutId);
  }, [selection?.text, targetLanguage, translateText]);

  // Prevent iOS context menu from appearing
  useEffect(() => {
    const handleContextMenu = (event: Event) => {
      // Prevent the default context menu on mobile devices
      const target = event.target as Element;
      if (
        target.closest('[data-selectable-text]') ||
        target.closest('.selectable-text')
      ) {
        event.preventDefault();
        event.stopPropagation();
      }
    };

    document.addEventListener('contextmenu', handleContextMenu, {
      passive: false,
    });

    return () => {
      document.removeEventListener('contextmenu', handleContextMenu);
    };
  }, []);

  // Handle clicks outside to close popup
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Element;

      // Don't close if clicking inside the popup
      const isInsidePopup = popupRef.current?.contains(target);

      // Don't close if clicking on Select dropdown elements (rendered in Portal)
      // Check for various Mantine Select-related class names and attributes
      const isSelectElement =
        target.closest('.mantine-Select-dropdown') ||
        target.closest('.mantine-Popover-dropdown') ||
        target.closest('[role="option"]') ||
        target.closest('.mantine-Select-item') ||
        target.closest('.mantine-Select-input') ||
        target.closest('.mantine-Select-root') ||
        target.closest('[data-combobox-target]') ||
        target.closest('[data-combobox-dropdown]') ||
        target.closest('[role="listbox"]') ||
        // Check if the target is within any Select component
        target.getAttribute('data-translation-select') === 'true' ||
        target.closest('[data-translation-select="true"]');

      // Determine if click is within the original text selection bounds
      const isWithinSelectionBounds =
        selection?.boundingRect &&
        event.clientX >= selection.boundingRect.left &&
        event.clientX <= selection.boundingRect.right &&
        event.clientY >= selection.boundingRect.top &&
        event.clientY <= selection.boundingRect.bottom;

      // Don't close if clicking directly on the selected text itself
      const isSelectedText =
        selection &&
        isWithinSelectionBounds &&
        target.closest('[data-translation-enabled]');

      // Fallback: if no bounding box data, preserve previous text-based detection
      const isSelectedTextFallback =
        !selection?.boundingRect &&
        selection &&
        target.textContent?.includes(selection.text) &&
        target.closest('[data-translation-enabled]');

      if (
        isInsidePopup ||
        isSelectElement ||
        isSelectedText ||
        isSelectedTextFallback
      ) {
        return;
      }

      onClose();
    };

    // Use a small delay to prevent immediate closing when popup opens
    const timeoutId = setTimeout(() => {
      document.addEventListener('mousedown', handleClickOutside);
    }, 100);

    return () => {
      clearTimeout(timeoutId);
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [onClose, selection]);

  // Calculate popup position to stay within viewport
  const position = useMemo(() => {
    return {
      left: '50%',
      top: '50%',
      transform: 'translate(-50%, -50%)',
    };
  }, []);

  // Determine whether a valid question id is available
  const hasQuestionId = Boolean(currentQuestion && 'id' in currentQuestion);
  const saveDisabled =
    isSaving || isSaved || (requireQuestionId && !hasQuestionId);

  const handleSaveSnippet = async () => {
    if (!translation || !selection.text) return;

    setIsSaving(true);
    setSaveError(null);

    try {
      const payload = {
        original_text: selection.text,
        translated_text: translation.translatedText,
        source_language: translation.sourceLanguage,
        target_language: targetLanguage,
        context: selection.sentence, // Add the extracted sentence as context
        ...(currentQuestion &&
          'id' in currentQuestion && {
            question_id: (currentQuestion as Question).id,
          }),
        ...(currentQuestion &&
          'section_id' in currentQuestion && {
            section_id: (currentQuestion as StoryContext).section_id,
          }),
        ...(currentQuestion &&
          'story_id' in currentQuestion && {
            story_id: (currentQuestion as StoryContext).story_id,
          }),
      };

      await postV1Snippets(payload);

      // Invalidate and refetch relevant snippet queries to refresh highlights immediately
      if (currentQuestion && 'id' in currentQuestion) {
        const questionId = (currentQuestion as Question).id;
        // Invalidate question snippets
        queryClient.invalidateQueries({
          queryKey: [`/v1/snippets/by-question/${questionId}`],
        });
        // Force immediate refetch to show highlights right away
        await queryClient.refetchQueries({
          queryKey: [`/v1/snippets/by-question/${questionId}`],
        });
      }
      if (currentQuestion && 'section_id' in currentQuestion) {
        const sectionId = (currentQuestion as StoryContext).section_id;
        // Invalidate section snippets
        queryClient.invalidateQueries({
          queryKey: [`/v1/snippets/by-section/${sectionId}`],
        });
        // Force immediate refetch to show highlights right away
        await queryClient.refetchQueries({
          queryKey: [`/v1/snippets/by-section/${sectionId}`],
        });
      }
      if (currentQuestion && 'story_id' in currentQuestion) {
        const storyId = (currentQuestion as StoryContext).story_id;
        // Invalidate story snippets
        queryClient.invalidateQueries({
          queryKey: [`/v1/snippets/by-story/${storyId}`],
        });
        // Force immediate refetch to show highlights right away
        await queryClient.refetchQueries({
          queryKey: [`/v1/snippets/by-story/${storyId}`],
        });
      }

      // Also invalidate and refetch general snippets list in case any components show all snippets
      queryClient.invalidateQueries({
        queryKey: ['/v1/snippets'],
      });
      await queryClient.refetchQueries({
        queryKey: ['/v1/snippets'],
      });

      // Refetch all snippet-related queries using predicate to catch any active queries
      // This ensures highlights appear immediately even if the query key format differs slightly
      await queryClient.refetchQueries({
        predicate: query => {
          const key = query.queryKey[0]?.toString() || '';
          return (
            key.includes('/v1/snippets/by-question/') ||
            key.includes('/v1/snippets/by-section/') ||
            key.includes('/v1/snippets/by-story/') ||
            key === '/v1/snippets'
          );
        },
      });

      if (mountedRef.current) {
        setIsSaved(true);
        // Clear any existing timeout
        if (savedTimeoutRef.current) {
          clearTimeout(savedTimeoutRef.current);
        }
        // Close popup after a brief delay to show "Saved!" feedback
        setTimeout(() => {
          if (mountedRef.current) {
            onClose();
          }
        }, 300);
      }
    } catch (error) {
      if (mountedRef.current) {
        setSaveError(
          error instanceof Error ? error.message : 'Failed to save snippet'
        );
      }
    } finally {
      if (mountedRef.current) {
        setIsSaving(false);
      }
    }
  };

  // Enhanced close handler
  const handleClose = () => {
    onClose();
  };

  // Handle copy to clipboard
  const handleCopy = async (text: string, type: 'original' | 'translated') => {
    try {
      await navigator.clipboard.writeText(text);
      if (mountedRef.current) {
        setCopySuccess(type);
        // Clear any existing timeout
        if (copySuccessTimeoutRef.current) {
          clearTimeout(copySuccessTimeoutRef.current);
        }
        copySuccessTimeoutRef.current = setTimeout(() => {
          if (mountedRef.current) {
            setCopySuccess(null);
          }
          copySuccessTimeoutRef.current = null;
        }, 2000);
      }
    } catch (err) {
      console.error('Failed to copy text:', err);
    }
  };

  return (
    <Portal>
      <Paper
        ref={popupRef}
        className='translation-popup'
        shadow='md'
        p='lg'
        style={{
          position: 'fixed',
          zIndex: 2500,
          width: resolvedPopupWidth,
          maxWidth: `calc(100vw - ${popupMargin * 2}px)`,
          maxHeight: `calc(100vh - ${popupMargin * 2}px)`,
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          ...position,
        }}
        withBorder
      >
        <Stack gap='xs' style={{ flex: 1, minHeight: 0 }}>
          {/* Header */}
          <Group justify='space-between' align='flex-start'>
            <Text size='md' fw={500} c='dimmed'>
              Translation
            </Text>
            <Button variant='subtle' size='sm' p={2} onClick={handleClose}>
              <IconX size={16} />
            </Button>
          </Group>

          <Box
            style={{
              flex: 1,
              minHeight: 0,
              overflowY: 'auto',
              paddingRight: 12,
            }}
          >
            <Stack gap='xs'>
              {/* Original text */}
              <Group gap='xs' wrap='wrap' align='center'>
                <TTSButton
                  getText={() => selection.text}
                  getVoice={() =>
                    getPreferredVoice(translation?.sourceLanguage)
                  }
                  getMetadata={() =>
                    getMetadata('original', translation?.sourceLanguage)
                  }
                  getId={() =>
                    `translation-popup::original::${selection.text ?? ''}`
                  }
                  size='sm'
                  ariaLabel='Listen to original text'
                />
                <Tooltip
                  label={
                    copySuccess === 'original'
                      ? 'Copied!'
                      : 'Copy original text'
                  }
                  withinPortal={false}
                >
                  <ActionIcon
                    size='sm'
                    variant='subtle'
                    color={copySuccess === 'original' ? 'green' : undefined}
                    onClick={() => handleCopy(selection.text, 'original')}
                  >
                    <IconCopy size={16} />
                  </ActionIcon>
                </Tooltip>
                <Text
                  size='md'
                  style={{ fontStyle: 'italic', wordBreak: 'break-word' }}
                >
                  {selection.text}
                </Text>
              </Group>

              {/* Language selector */}
              <Select
                data={languageOptions}
                value={targetLanguage}
                onChange={value => {
                  if (value) {
                    setTargetLanguage(value);
                    // Persist language selection to localStorage
                    localStorage.setItem('quiz-translation-target-lang', value);
                  }
                }}
                size='sm'
                placeholder={
                  languagesLoading ? 'Loading languages...' : 'Select language'
                }
                disabled={languagesLoading || languageOptions.length === 0}
                style={{ width: '100%' }}
                data-translation-select='true'
                comboboxProps={{
                  withinPortal: true,
                  zIndex: 100000,
                }}
                styles={{
                  dropdown: {
                    zIndex: 100000,
                  },
                }}
              />

              {/* Translation result */}
              <div style={{ minHeight: `${60 * fontScaleMap[fontSize]}px` }}>
                {translationLoading && (
                  <Group gap='xs'>
                    <Loader size='md' />
                    <Text size='md' c='dimmed'>
                      Translating...
                    </Text>
                  </Group>
                )}

                {translationError && (
                  <Text size='md' c='red'>
                    {translationError.includes('temporarily unavailable')
                      ? '🔄 Translation service is temporarily unavailable. Please wait a moment and try again.'
                      : translationError.includes('Rate limit exceeded')
                        ? '⏱️ Too many translation requests. Please wait a moment and try again.'
                        : `❌ ${translationError}`}
                  </Text>
                )}

                {translation && !translationLoading && !translationError && (
                  <Stack gap='xs'>
                    <Group gap='xs' wrap='wrap' align='center'>
                      <TTSButton
                        getText={() => translation.translatedText}
                        getVoice={() => getPreferredVoice(targetLanguage)}
                        getMetadata={() =>
                          getMetadata('translated', targetLanguage)
                        }
                        getId={() =>
                          `translation-popup::translated::${targetLanguage}::${selection.text ?? ''}`
                        }
                        size='sm'
                        ariaLabel='Listen to translation'
                      />
                      <Tooltip
                        label={
                          copySuccess === 'translated'
                            ? 'Copied!'
                            : 'Copy translated text'
                        }
                        withinPortal={false}
                      >
                        <ActionIcon
                          size='sm'
                          variant='subtle'
                          color={
                            copySuccess === 'translated' ? 'green' : undefined
                          }
                          onClick={() =>
                            handleCopy(translation.translatedText, 'translated')
                          }
                        >
                          <IconCopy size={16} />
                        </ActionIcon>
                      </Tooltip>
                      <Text
                        size='md'
                        style={{
                          wordBreak: 'break-word',
                          whiteSpace: 'normal',
                        }}
                      >
                        {translation.translatedText}
                      </Text>
                    </Group>
                    <Group gap='xs' wrap='wrap' justify='flex-end'>
                      <Tooltip
                        label={
                          requireQuestionId && !hasQuestionId
                            ? 'Waiting for question id…'
                            : 'Save to snippets'
                        }
                        withArrow
                        withinPortal={false}
                      >
                        <Button
                          variant={isSaved ? 'filled' : 'light'}
                          size='xs'
                          px={10}
                          leftSection={
                            isSaving ? (
                              <Loader size={14} data-testid='loader' />
                            ) : (
                              <IconBookmark size={14} />
                            )
                          }
                          onClick={handleSaveSnippet}
                          disabled={saveDisabled}
                          color={isSaved ? 'green' : undefined}
                        >
                          {isSaving ? 'Saving...' : isSaved ? 'Saved!' : 'Save'}
                        </Button>
                      </Tooltip>
                    </Group>
                    {saveError && (
                      <Text size='sm' c='red'>
                        {saveError}
                      </Text>
                    )}
                  </Stack>
                )}
              </div>

              {/* Footer */}
              <Text size='sm' c='dimmed' ta='center'>
                Powered by Google Translate
              </Text>
            </Stack>
          </Box>
        </Stack>
      </Paper>
    </Portal>
  );
};
