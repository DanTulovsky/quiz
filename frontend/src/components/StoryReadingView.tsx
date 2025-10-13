import React from 'react';
import {
  Paper,
  Title,
  Text,
  Group,
  Stack,
  Divider,
  ScrollArea,
  Loader,
  Center,
  ActionIcon,
  Tooltip,
  Box,
} from '@mantine/core';
import { IconBook, IconRefresh } from '@tabler/icons-react';
import { StoryWithSections } from '../api/storyApi';
import { useTTS } from '../hooks/useTTS';
import { Volume2, VolumeX } from 'lucide-react';
import { defaultVoiceForLanguage } from '../utils/tts';
import { useGetV1PreferencesLearning } from '../api/api';

interface StoryReadingViewProps {
  story: StoryWithSections | null;
  isGenerating?: boolean;
}

const StoryReadingView: React.FC<StoryReadingViewProps> = ({
  story,
  isGenerating = false,
}) => {
  const {
    isLoading: isTTSLoading,
    isPlaying: isTTSPlaying,
    playTTS,
    stopTTS,
  } = useTTS();

  // Get user learning preferences for preferred voice
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();

  if (!story) {
    return (
      <Paper p='xl' radius='md' style={{ textAlign: 'center' }}>
        <IconBook size={48} style={{ opacity: 0.5, marginBottom: 16 }} />
        <Title order={4}>No story to display</Title>
        <Text color='dimmed'>Create a new story to start reading.</Text>
      </Paper>
    );
  }

  if (story.sections.length === 0) {
    if (isGenerating) {
      return (
        <Paper p='xl' radius='md' style={{ textAlign: 'center' }}>
          <Center>
            <Stack spacing='md' align='center'>
              <Loader size='lg' />
              <IconBook size={48} style={{ opacity: 0.5 }} />
              <Title order={4}>Generating Your Story</Title>
              <Text color='dimmed' align='center'>
                We're creating the first section of your story.
                <br />
                This should only take a few moments...
              </Text>
              <Group spacing='xs'>
                <IconRefresh size={16} />
                <Text size='sm' color='dimmed'>
                  Checking for updates...
                </Text>
              </Group>
            </Stack>
          </Center>
        </Paper>
      );
    } else {
      return (
        <Paper p='xl' radius='md' style={{ textAlign: 'center' }}>
          <IconBook size={48} style={{ opacity: 0.5, marginBottom: 16 }} />
          <Title order={4}>Story in Progress</Title>
          <Text color='dimmed'>
            Your story is being prepared. Check back soon for the first section!
          </Text>
        </Paper>
      );
    }
  }

  return (
    <Stack spacing='md'>
      {/* Story Content */}
      <Paper p='lg' radius='md' style={{ position: 'relative' }}>
        <Box style={{ position: 'absolute', top: 12, right: 12, zIndex: 10 }}>
          <Tooltip
            label={
              isTTSPlaying
                ? 'Stop audio'
                : isTTSLoading
                  ? 'Loading audio...'
                  : 'Listen to story'
            }
          >
            <ActionIcon
              size='md'
              variant='subtle'
              color={isTTSPlaying ? 'red' : isTTSLoading ? 'orange' : 'blue'}
              onClick={() => {
                if (isTTSPlaying || isTTSLoading) {
                  stopTTS();
                } else {
                  // Combine the sections into one text blob
                  const full = story.sections.map(s => s.content).join('\n\n');
                  let preferredVoice: string | undefined;
                  if (
                    userLearningPrefs?.tts_voice &&
                    userLearningPrefs.tts_voice.trim()
                  ) {
                    preferredVoice = userLearningPrefs.tts_voice.trim();
                  }
                  const finalVoice =
                    preferredVoice ??
                    defaultVoiceForLanguage(story.language) ??
                    'echo';
                  void playTTS(full, finalVoice);
                }
              }}
              aria-label={
                isTTSPlaying
                  ? 'Stop audio'
                  : isTTSLoading
                    ? 'Loading audio'
                    : 'Listen to story'
              }
              disabled={isTTSLoading}
            >
              {isTTSLoading ? (
                <Loader size={16} color='orange' />
              ) : isTTSPlaying ? (
                <VolumeX size={18} />
              ) : (
                <Volume2 size={18} />
              )}
            </ActionIcon>
          </Tooltip>
        </Box>

        <ScrollArea style={{ height: '60vh' }}>
          <div style={{ paddingRight: '56px' }}>
            <Stack spacing='lg'>
              {/* Story Metadata */}
              {(story.subject || story.author_style || story.genre) && (
                <div>
                  <Title order={5} mb='xs'>
                    Story Details
                  </Title>
                  <Group gap='md'>
                    {story.subject && (
                      <Text size='sm'>
                        <strong>Subject:</strong> {story.subject}
                      </Text>
                    )}
                    {story.author_style && (
                      <Text size='sm'>
                        <strong>Style:</strong> {story.author_style}
                      </Text>
                    )}
                    {story.genre && (
                      <Text size='sm'>
                        <strong>Genre:</strong> {story.genre}
                      </Text>
                    )}
                  </Group>
                  <Divider my='md' />
                </div>
              )}

              {/* Story Sections */}
              {story.sections.map((section, index) => (
                <div key={section.id || index}>
                  {/* Section Content */}
                  <div
                    style={{
                      lineHeight: 1.7,
                      fontSize: '16px',
                      whiteSpace: 'pre-wrap',
                      // Space for the TTS icon so text doesn't overlap
                      paddingRight: '4px',
                      marginBottom:
                        index < story.sections.length - 1 ? '1.5rem' : '1rem',
                    }}
                  >
                    {section.content}
                  </div>
                </div>
              ))}

              {/* Story End Notice */}
              <Paper
                p='md'
                radius='sm'
                style={{
                  backgroundColor: '#f0f9ff',
                  border: '1px solid #e0f2fe',
                }}
              >
                <Text size='sm' color='blue' align='center'>
                  {story.status === 'active'
                    ? 'This story is ongoing. New sections will be added daily!'
                    : story.status === 'completed'
                      ? 'This story has been completed.'
                      : 'This story has been archived.'}
                </Text>
              </Paper>
            </Stack>
          </div>
        </ScrollArea>
      </Paper>
    </Stack>
  );
};

export default StoryReadingView;
