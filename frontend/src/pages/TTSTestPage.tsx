import React, { useState, useEffect } from 'react';
import {
  Container,
  Title,
  Textarea,
  Button,
  Stack,
  Select,
  Card,
  Text,
  Group,
  Loader,
  Alert,
} from '@mantine/core';
import { useTTS } from '../hooks/useTTS';
import {
  defaultVoiceForLanguage,
  extractVoiceName,
  sampleTextForLanguage,
} from '../utils/tts';
import type { EdgeTTSVoiceInfo } from '../utils/tts';
import { Volume2, Play, Pause, Square } from 'lucide-react';

const TTSTestPage: React.FC = () => {
  const defaultText =
    'Hello! This is a test of the text-to-speech functionality. You can type any text here and it will be spoken aloud.';
  const [text, setText] = useState(defaultText);
  const [selectedLanguage, setSelectedLanguage] = useState<string | null>(
    'English'
  );
  const [selectedVoice, setSelectedVoice] = useState<string | null>(null);
  const [availableVoices, setAvailableVoices] = useState<string[]>([]);
  const [loadingVoices, setLoadingVoices] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    isLoading,
    isPlaying,
    isPaused,
    playTTS,
    stopTTS,
    pauseTTS,
    resumeTTS,
    restartTTS,
  } = useTTS();

  // Language options
  const languages = [
    { value: 'English', label: 'English' },
    { value: 'Spanish', label: 'Spanish' },
    { value: 'French', label: 'French' },
    { value: 'German', label: 'German' },
    { value: 'Italian', label: 'Italian' },
    { value: 'Portuguese', label: 'Portuguese' },
    { value: 'Russian', label: 'Russian' },
    { value: 'Japanese', label: 'Japanese' },
    { value: 'Korean', label: 'Korean' },
    { value: 'Chinese', label: 'Chinese' },
    { value: 'Hindi', label: 'Hindi' },
  ];

  // Fetch voices when language changes and update sample text
  useEffect(() => {
    if (!selectedLanguage) {
      setAvailableVoices([]);
      setSelectedVoice(null);
      setText(defaultText);
      return;
    }

    // Update text to language-specific sample
    const sampleText = sampleTextForLanguage(selectedLanguage);
    if (sampleText) {
      setText(sampleText);
    }

    const fetchVoices = async () => {
      setLoadingVoices(true);
      setError(null);
      try {
        // Get locale from language name
        const localeMap: Record<string, string> = {
          English: 'en-US',
          Spanish: 'es-ES',
          French: 'fr-FR',
          German: 'de-DE',
          Italian: 'it-IT',
          Portuguese: 'pt-PT',
          Russian: 'ru-RU',
          Japanese: 'ja-JP',
          Korean: 'ko-KR',
          Chinese: 'zh-CN',
          Hindi: 'hi-IN',
        };

        const locale = localeMap[selectedLanguage];
        if (!locale) {
          setAvailableVoices([]);
          return;
        }

        const res = await fetch(
          `/v1/voices?language=${encodeURIComponent(locale)}`
        );
        if (!res.ok) throw new Error('Failed to fetch voices');
        const json: unknown = await res.json();
        const rawVoices: EdgeTTSVoiceInfo[] = Array.isArray(json)
          ? (json as EdgeTTSVoiceInfo[])
          : ((json as { voices?: EdgeTTSVoiceInfo[] })?.voices ?? []);
        const voices = (rawVoices || [])
          .map(extractVoiceName)
          .filter((v): v is string => !!v);

        setAvailableVoices(voices);

        // Set default voice if available
        if (voices.length > 0 && !selectedVoice) {
          const defaultVoice = defaultVoiceForLanguage(selectedLanguage);
          if (defaultVoice && voices.includes(defaultVoice)) {
            setSelectedVoice(defaultVoice);
          } else {
            setSelectedVoice(voices[0]);
          }
        }
      } catch (err) {
        console.error('Failed to fetch voices:', err);
        setError('Failed to load voices for this language');
        setAvailableVoices([]);
      } finally {
        setLoadingVoices(false);
      }
    };

    fetchVoices();
  }, [selectedLanguage]);

  const handlePlay = async () => {
    if (!text.trim()) {
      setError('Please enter some text to speak');
      return;
    }
    if (!selectedVoice) {
      setError('Please select a voice');
      return;
    }
    setError(null);
    try {
      await playTTS(text, selectedVoice);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to play audio');
    }
  };

  return (
    <Container size='lg' py='xl'>
      <Stack gap='lg'>
        <div>
          <Title order={1} mb='xs'>
            TTS Test Page
          </Title>
          <Text c='dimmed'>
            Test the text-to-speech functionality. Enter text and select a
            language/voice.
          </Text>
        </div>

        {error && (
          <Alert
            color='red'
            title='Error'
            onClose={() => setError(null)}
            withCloseButton
          >
            {error}
          </Alert>
        )}

        <Card shadow='sm' padding='lg' radius='md' withBorder>
          <Stack gap='md'>
            <Select
              label='Language'
              placeholder='Select a language'
              data={languages}
              value={selectedLanguage}
              onChange={setSelectedLanguage}
              required
            />

            {loadingVoices ? (
              <Group>
                <Loader size='sm' />
                <Text size='sm' c='dimmed'>
                  Loading voices...
                </Text>
              </Group>
            ) : (
              <Select
                label='Voice'
                placeholder='Select a voice'
                data={availableVoices.map(v => ({ value: v, label: v }))}
                value={selectedVoice}
                onChange={setSelectedVoice}
                required
                disabled={availableVoices.length === 0}
                description={
                  availableVoices.length === 0
                    ? 'No voices available for this language'
                    : `${availableVoices.length} voice(s) available`
                }
              />
            )}

            <Textarea
              label='Text to Speak'
              placeholder='Enter text here...'
              value={text}
              onChange={e => setText(e.target.value)}
              minRows={8}
              required
              styles={{
                input: {
                  fontSize: '1.1rem',
                  lineHeight: 1.6,
                },
              }}
            />

            <Group>
              <Button
                leftSection={
                  isLoading ? (
                    <Loader size={16} />
                  ) : isPlaying ? (
                    <Pause size={18} />
                  ) : isPaused ? (
                    <Play size={18} />
                  ) : (
                    <Volume2 size={18} />
                  )
                }
                onClick={() => {
                  if (isPlaying) {
                    pauseTTS();
                  } else if (isPaused) {
                    resumeTTS();
                  } else {
                    handlePlay();
                  }
                }}
                disabled={isLoading || !text.trim() || !selectedVoice}
                loading={isLoading}
                color={isPaused ? 'green' : undefined}
              >
                {isLoading
                  ? 'Generating...'
                  : isPlaying
                    ? 'Pause'
                    : isPaused
                      ? 'Resume'
                      : 'Play'}
              </Button>

              {(isPlaying || isPaused) && (
                <>
                  <Button
                    leftSection={<Square size={18} />}
                    onClick={restartTTS}
                    variant='light'
                  >
                    Restart
                  </Button>
                  <Button
                    leftSection={<Square size={18} />}
                    onClick={stopTTS}
                    variant='light'
                  >
                    Stop
                  </Button>
                </>
              )}
            </Group>

            <Text size='sm' c='dimmed'>
              Debug: isLoading={String(isLoading)}, isPlaying=
              {String(isPlaying)}, isPaused={String(isPaused)}
            </Text>
            {(isPlaying || isPaused) && (
              <Text size='sm' c='dimmed'>
                Status:{' '}
                {isPlaying ? 'Playing' : isPaused ? 'Paused' : 'Stopped'}
              </Text>
            )}
          </Stack>
        </Card>

        <Card shadow='sm' padding='lg' radius='md' withBorder>
          <Stack gap='xs'>
            <Text fw={600}>Instructions:</Text>
            <Text size='sm' c='dimmed'>
              1. Select a language from the dropdown
            </Text>
            <Text size='sm' c='dimmed'>
              2. Wait for voices to load, then select a voice
            </Text>
            <Text size='sm' c='dimmed'>
              3. Enter or modify the text in the text area
            </Text>
            <Text size='sm' c='dimmed'>
              4. Click Play to hear the text spoken
            </Text>
          </Stack>
        </Card>
      </Stack>
    </Container>
  );
};

export default TTSTestPage;
