import React, { useState, useEffect, useRef } from 'react';
import {
  Paper,
  Text,
  Loader,
  Button,
  Group,
  Select,
  Stack,
} from '@mantine/core';
import { useTranslation } from '../contexts/TranslationContext';
import { TextSelection } from '../hooks/useTextSelection';
import { IconX, IconVolume, IconBookmark } from '@tabler/icons-react';
import { postV1Snippets, Question } from '../api/api';

interface TranslationPopupProps {
  selection: TextSelection;
  onClose: () => void;
  currentQuestion?: Question | null;
}

export const TranslationPopup: React.FC<TranslationPopupProps> = ({
  selection,
  onClose,
  currentQuestion,
}) => {
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
  const popupRef = useRef<HTMLDivElement>(null);

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

      if (isInsidePopup || isSelectElement) {
        return;
      }

      onClose();
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose, isSelectFocused]);

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

  const speakText = (text: string, lang: string) => {
    if ('speechSynthesis' in window) {
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.lang = lang;
      utterance.rate = 0.8;
      speechSynthesis.speak(utterance);
    }
  };

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
        ...(currentQuestion?.id && { question_id: currentQuestion.id }),
      };

      await postV1Snippets(payload);

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

  return (
    <Paper
      ref={popupRef}
      className='translation-popup'
      shadow='md'
      p='md'
      style={{
        position: 'fixed',
        zIndex: 1500, // Increased z-index to ensure dropdown appears on top
        width: 320,
        maxWidth: '90vw',
        ...position,
      }}
      withBorder
    >
      <Stack gap='xs'>
        {/* Header */}
        <Group justify='space-between' align='flex-start'>
          <Text size='sm' fw={500} c='dimmed'>
            Translation
          </Text>
          <Button variant='subtle' size='xs' p={2} onClick={onClose}>
            <IconX size={14} />
          </Button>
        </Group>

        {/* Original text */}
        <Text size='sm' style={{ fontStyle: 'italic' }}>
          "{selection.text}"
        </Text>

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
          size='xs'
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
              zIndex: 2000,
            },
          }}
        />

        {/* Translation result */}
        <div style={{ minHeight: 60 }}>
          {translationLoading && (
            <Group gap='xs'>
              <Loader size='sm' />
              <Text size='sm' c='dimmed'>
                Translating...
              </Text>
            </Group>
          )}

          {translationError && (
            <Text size='sm' c='red'>
              {translationError.includes('temporarily unavailable')
                ? '🔄 Translation service is temporarily unavailable. Please wait a moment and try again.'
                : translationError.includes('Rate limit exceeded')
                  ? '⏱️ Too many translation requests. Please wait a moment and try again.'
                  : `❌ ${translationError}`}
            </Text>
          )}

          {translation && !translationLoading && !translationError && (
            <Stack gap='xs'>
              <Text size='sm'>{translation.translatedText}</Text>
              <Group gap='xs'>
                <Button
                  variant='light'
                  size='xs'
                  leftSection={<IconVolume size={14} />}
                  onClick={() =>
                    speakText(translation.translatedText, targetLanguage)
                  }
                >
                  Listen
                </Button>
                <Button
                  variant='light'
                  size='xs'
                  leftSection={<IconVolume size={14} />}
                  onClick={() =>
                    speakText(selection.text, translation.sourceLanguage)
                  }
                >
                  Original
                </Button>
                <Button
                  variant={isSaved ? 'filled' : 'light'}
                  size='xs'
                  leftSection={
                    isSaving ? (
                      <Loader size={14} data-testid='loader' />
                    ) : isSaved ? (
                      <IconBookmark size={14} />
                    ) : (
                      <IconBookmark size={14} />
                    )
                  }
                  onClick={handleSaveSnippet}
                  disabled={isSaving || isSaved}
                  color={isSaved ? 'green' : undefined}
                >
                  {isSaving ? 'Saving...' : isSaved ? 'Saved!' : 'Save'}
                </Button>
              </Group>
              {saveError && (
                <Text size='xs' c='red'>
                  {saveError}
                </Text>
              )}
            </Stack>
          )}
        </div>

        {/* Footer */}
        <Text size='xs' c='dimmed' ta='center'>
          Powered by Google Translate
        </Text>
      </Stack>
    </Paper>
  );
};
