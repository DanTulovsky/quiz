import React, { useState, useEffect, useRef } from 'react';
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
} from '@mantine/core';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from '../contexts/TranslationContext';
import { TextSelection } from '../hooks/useTextSelection';
import { IconX, IconBookmark } from '@tabler/icons-react';
import {
  postV1Snippets,
  Question,
  useGetV1PreferencesLearning,
} from '../api/api';
import { useTheme } from '../contexts/ThemeContext';
import { fontScaleMap } from '../theme/theme';
import TTSButton from './TTSButton';
import { defaultVoiceForLanguage } from '../utils/tts';

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
  const [targetLanguage, setTargetLanguage] = useState(() => {
    const saved = localStorage.getItem('quiz-translation-target-lang');
    if (
      saved &&
      ['en', 'es', 'fr', 'de', 'it', 'pt', 'ru', 'ja', 'ko', 'zh'].includes(
        saved
      )
    ) {
      return saved;
    }
    // Try to detect user's preferred language from browser
    const browserLang = navigator.language.split('-')[0];
    if (
      ['en', 'es', 'fr', 'de', 'it', 'pt', 'ru', 'ja', 'ko', 'zh'].includes(
        browserLang
      )
    ) {
      return browserLang;
    }
    return 'en';
  });
  const [isSelectFocused, setIsSelectFocused] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [isSaved, setIsSaved] = useState(false);
  const {
    translateText,
    translation,
    isLoading: translationLoading,
    error: translationError,
  } = useTranslation();
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();
  const popupRef = useRef<HTMLDivElement>(null);

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
    };
    return mapping[code] || code;
  };

  // Language options for the dropdown
  const languageOptions = [
    { value: 'en', label: 'English' },
    { value: 'es', label: 'Spanish' },
    { value: 'fr', label: 'French' },
    { value: 'de', label: 'German' },
    { value: 'it', label: 'Italian' },
    { value: 'pt', label: 'Portuguese' },
    { value: 'ru', label: 'Russian' },
    { value: 'ja', label: 'Japanese' },
    { value: 'ko', label: 'Korean' },
    { value: 'zh', label: 'Chinese' },
  ];

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
      // Don't close if the Select is focused (user is interacting with it)
      if (isSelectFocused) {
        return;
      }

      const target = event.target as Element;

      // Don't close if clicking inside the popup or on Select elements
      const isInsidePopup = popupRef.current?.contains(target);
      const isSelectElement =
        target.closest('.mantine-Select-dropdown') ||
        target.closest('.mantine-Popover-dropdown') ||
        target.closest('[role="option"]') ||
        target.closest('.mantine-Select-item') ||
        target.closest('.mantine-Select-input') ||
        target.closest('.mantine-Select-root');

      // Don't close if clicking on the selected text itself
      const isSelectedText =
        selection &&
        target.textContent?.includes(selection.text) &&
        target.closest('[data-translation-enabled]');

      if (isInsidePopup || isSelectElement || isSelectedText) {
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
  }, [onClose, isSelectFocused, selection]);

  // Calculate popup position to stay within viewport
  const getPopupPosition = () => {
    const popupWidth = 320;
    const popupHeight = 200;
    const margin = 10;

    let x = selection.x - popupWidth / 2;
    let y = selection.y - popupHeight - margin;

    // Adjust if popup goes off screen horizontally
    if (x < margin) {
      x = margin;
    } else if (x + popupWidth > window.innerWidth - margin) {
      x = window.innerWidth - popupWidth - margin;
    }

    // Adjust if popup goes off screen vertically
    if (y < margin) {
      y = selection.y + selection.height + margin;
    }

    return { left: x, top: y };
  };

  const position = getPopupPosition();

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

      // Invalidate relevant snippet queries to refresh highlights
      if (currentQuestion && 'id' in currentQuestion) {
        // Invalidate question snippets
        queryClient.invalidateQueries({
          queryKey: [
            `/v1/snippets/by-question/${(currentQuestion as Question).id}`,
          ],
        });
      }
      if (currentQuestion && 'section_id' in currentQuestion) {
        // Invalidate section snippets
        queryClient.invalidateQueries({
          queryKey: [
            `/v1/snippets/by-section/${(currentQuestion as StoryContext).section_id}`,
          ],
        });
      }
      if (currentQuestion && 'story_id' in currentQuestion) {
        // Invalidate story snippets
        queryClient.invalidateQueries({
          queryKey: [
            `/v1/snippets/by-story/${(currentQuestion as StoryContext).story_id}`,
          ],
        });
      }

      // Also invalidate general snippets list in case any components show all snippets
      queryClient.invalidateQueries({
        queryKey: ['/v1/snippets'],
      });

      setIsSaved(true);
      // Reset saved state after 3 seconds
      setTimeout(() => setIsSaved(false), 3000);
    } catch (error) {
      setSaveError(
        error instanceof Error ? error.message : 'Failed to save snippet'
      );
    } finally {
      setIsSaving(false);
    }
  };

  // Enhanced close handler that clears all interaction states
  const handleClose = () => {
    setIsSelectFocused(false);
    onClose();
  };

  return (
    <Portal zIndex={2500}>
      <Paper
        ref={popupRef}
        className='translation-popup'
        shadow='md'
        p='lg'
        style={{
          position: 'fixed',
          zIndex: 99999,
          width: `${320 * fontScaleMap[fontSize]}px`,
          maxWidth: '90vw',
          ...position,
        }}
        withBorder
      >
        <Stack gap='xs'>
          {/* Header */}
          <Group justify='space-between' align='flex-start'>
            <Text size='md' fw={500} c='dimmed'>
              Translation
            </Text>
            <Button variant='subtle' size='sm' p={2} onClick={handleClose}>
              <IconX size={16} />
            </Button>
          </Group>

          {/* Original text */}
          <Group gap='xs' wrap='nowrap' align='center'>
            <TTSButton
              getText={() => selection.text}
              getVoice={() => {
                const saved = (userLearningPrefs?.tts_voice || '').trim();
                if (saved) return saved;
                // Use detected source language if available
                if (translation?.sourceLanguage) {
                  const languageName = codeToLanguageName(
                    translation.sourceLanguage
                  );
                  return defaultVoiceForLanguage(languageName) || undefined;
                }
                return undefined;
              }}
              size='sm'
              ariaLabel='Listen to original text'
            />
            <Text size='md' style={{ fontStyle: 'italic' }}>
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
            placeholder='Select language'
            style={{ width: '100%' }}
            onFocus={() => {
              setIsSelectFocused(true);
            }}
            onBlur={() => {
              setIsSelectFocused(false);
            }}
            styles={{
              dropdown: {
                zIndex: 1600,
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
                <Group gap='xs' wrap='nowrap' align='center'>
                  <TTSButton
                    getText={() => translation.translatedText}
                    getVoice={() => {
                      // For translations, always use the target language from dropdown
                      const languageName = codeToLanguageName(targetLanguage);
                      return defaultVoiceForLanguage(languageName) || undefined;
                    }}
                    size='sm'
                    ariaLabel='Listen to translation'
                  />
                  <Text size='md'>{translation.translatedText}</Text>
                </Group>
                <Group gap='xs' wrap='nowrap' justify='flex-end'>
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
      </Paper>
    </Portal>
  );
};
