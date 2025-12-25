import React, { useState } from 'react';
import {
  Container,
  Title,
  Text,
  Group,
  Button,
  Stack,
  Badge,
  Alert,
  Loader,
  TextInput,
  ScrollArea,
  useMantineTheme,
} from '@mantine/core';
import {
  IconArchive,
  IconBook,
  IconCalendar,
  IconLanguage,
  IconSearch,
  IconCheck,
  IconEye,
} from '@tabler/icons-react';
import * as TablerIcons from '@tabler/icons-react';

const tablerIconMap = TablerIcons as unknown as Record<
  string,
  React.ComponentType<React.SVGProps<SVGSVGElement> & { size?: number }>
>;
const IconRotateClockwise: React.ComponentType<
  React.SVGProps<SVGSVGElement> & { size?: number }
> =
  tablerIconMap.IconRotateClockwise ||
  tablerIconMap.IconRefresh ||
  (() => null);
import { Story, StoryWithSections } from '../api/storyApi';

interface ArchivedStoriesViewProps {
  currentStory?: StoryWithSections | null;
  isGenerating?: boolean;
  archivedStories: Story[];
  isLoading: boolean;
  onUnarchive: (storyId: number) => Promise<void>;
  onViewCurrentStory?: () => void;
  onCreateNew: () => void;
  hideCreateButton?: boolean;
}

const ArchivedStoriesView: React.FC<ArchivedStoriesViewProps> = ({
  currentStory,
  isGenerating = false,
  archivedStories,
  isLoading,
  onUnarchive,
  onViewCurrentStory,
  onCreateNew,
  hideCreateButton = false,
}) => {
  const theme = useMantineTheme();
  const [searchQuery, setSearchQuery] = useState('');

  // Filter stories to only show archived ones, then apply search filter
  const filteredStories = archivedStories
    .filter(story => story.status === 'archived') // Only show archived stories
    .filter(
      story =>
        story.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        story.language?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        story.genre?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        story.subject?.toLowerCase().includes(searchQuery.toLowerCase())
    );

  if (isLoading) {
    return (
      <Container size='sm' py='xl'>
        <Stack gap='md' style={{ alignItems: 'center' }}>
          <Loader size='lg' />
          <Text color='dimmed'>Loading archived stories...</Text>
        </Stack>
      </Container>
    );
  }

  if (!archivedStories || archivedStories.length === 0) {
    return (
      <Container size='sm' py='xl'>
        <Stack gap='lg' style={{ alignItems: 'center' }}>
          <Group justify='center' gap='xs'>
            <IconBook size={32} />
            <Title order={2}>Story Mode</Title>
          </Group>

          <Text size='lg' color='dimmed' ta='center'>
            Create personalized stories in your target language at your
            proficiency level. Each story is generated daily with comprehension
            questions to test your understanding.
          </Text>

          <Alert color='blue' variant='light'>
            <Text size='sm'>
              <strong>How it works:</strong> Create a story with custom
              parameters, then read new sections daily. Each section includes
              comprehension questions to help you learn and practice your target
              language.
            </Text>
          </Alert>

          <Button
            leftSection={<IconBook size={16} />}
            size='lg'
            onClick={onCreateNew}
          >
            Create New Story
          </Button>
        </Stack>
      </Container>
    );
  }

  return (
    <Container size='md' py='xl'>
      <Stack gap='md'>
        {/* Create New Story Button - Moved to top */}
        {!hideCreateButton && (
          <Group justify='center'>
            <Button
              leftSection={<IconBook size={16} />}
              size='lg'
              onClick={onCreateNew}
            >
              Create New Story
            </Button>
          </Group>
        )}

        {/* Current Story Section */}
        {currentStory && (
          <>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <IconBook size={24} />
              <Title order={2}>Current Story</Title>
            </div>

            <Alert
              color={isGenerating ? 'blue' : 'green'}
              icon={
                isGenerating ? <Loader size={16} /> : <IconCheck size={16} />
              }
            >
              <Group justify='space-between' mb='xs'>
                <Title order={4} style={{ margin: 0 }}>
                  {currentStory.title}
                </Title>
                <Group gap='xs'>
                  <Badge
                    color={isGenerating ? 'blue' : 'green'}
                    variant='light'
                  >
                    {isGenerating ? 'Generating Section...' : 'Ready'}
                  </Badge>
                  {onViewCurrentStory && (
                    <Button
                      size='xs'
                      variant='light'
                      color='blue'
                      leftSection={<IconEye size={14} />}
                      onClick={onViewCurrentStory}
                    >
                      View
                    </Button>
                  )}
                </Group>
              </Group>

              <Group gap='lg'>
                <Group gap='xs'>
                  <IconLanguage size={14} />
                  <Text size='sm'>{currentStory.language}</Text>
                </Group>

                <Group gap='xs'>
                  <IconBook size={14} />
                  <Text size='sm'>
                    {currentStory.sections?.length || 0} section
                    {(currentStory.sections?.length || 0) !== 1 ? 's' : ''}
                  </Text>
                </Group>

                <Group gap='xs'>
                  <IconCalendar size={14} />
                  <Text size='sm'>
                    {currentStory.created_at
                      ? new Date(currentStory.created_at).toLocaleDateString()
                      : 'N/A'}
                  </Text>
                </Group>
              </Group>
            </Alert>
          </>
        )}

        {/* Archived Stories Section */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <IconArchive size={24} />
          <Title order={2}>Archived Stories</Title>
        </div>

        <Text color='dimmed' size='sm'>
          You have{' '}
          {archivedStories.filter(story => story.status === 'archived').length}{' '}
          archived{' '}
          {archivedStories.filter(story => story.status === 'archived')
            .length === 1
            ? 'story'
            : 'stories'}
          . You can restore any of these to continue reading or create a new
          story.
        </Text>

        {/* Search Input */}
        <TextInput
          placeholder='Search stories by title, language, genre, or subject...'
          leftSection={<IconSearch size={16} />}
          value={searchQuery}
          onChange={event => setSearchQuery(event.currentTarget.value)}
          size='md'
        />

        {/* Scrollable Stories Area */}
        <div style={{ height: '500px' }}>
          <ScrollArea h='100%' type='auto'>
            <Stack gap='xs'>
              {filteredStories.length === 0 ? (
                <Text color='dimmed' ta='center' py='xl'>
                  {searchQuery
                    ? 'No stories match your search.'
                    : 'No archived stories found.'}
                </Text>
              ) : (
                filteredStories.map((story, index) => (
                  <Group
                    key={story.id}
                    justify='space-between'
                    style={{
                      alignItems: 'center',
                      backgroundColor:
                        index % 2 === 0 ? theme.colors.gray[0] : 'transparent',
                    }}
                    p='sm'
                    onMouseEnter={e => {
                      e.currentTarget.style.backgroundColor =
                        theme.colors.gray[1];
                    }}
                    onMouseLeave={e => {
                      e.currentTarget.style.backgroundColor =
                        index % 2 === 0 ? theme.colors.gray[0] : 'transparent';
                    }}
                  >
                    <div style={{ flex: 1 }}>
                      <Group justify='space-between' mb='xs'>
                        <Title order={4} style={{ margin: 0 }}>
                          {story.title}
                        </Title>
                        <Badge color='gray' variant='light'>
                          {story.status}
                        </Badge>
                      </Group>

                      <Group gap='lg'>
                        <Group gap='xs'>
                          <IconLanguage size={14} />
                          <Text size='sm' color='dimmed'>
                            {story.language}
                          </Text>
                        </Group>

                        <Group gap='xs'>
                          <IconCalendar size={14} />
                          <Text size='sm' color='dimmed'>
                            {story.created_at
                              ? new Date(story.created_at).toLocaleDateString()
                              : 'N/A'}
                          </Text>
                        </Group>

                        {story.genre && (
                          <Text size='sm' color='dimmed'>
                            • {story.genre}
                          </Text>
                        )}

                        {story.subject && (
                          <Text size='sm' color='dimmed'>
                            • {story.subject}
                          </Text>
                        )}
                      </Group>
                    </div>

                    <Button
                      variant='light'
                      leftSection={<IconRotateClockwise size={16} />}
                      onClick={() => onUnarchive(story.id!)}
                      color='green'
                      size='sm'
                    >
                      Restore
                    </Button>
                  </Group>
                ))
              )}
            </Stack>
          </ScrollArea>
        </div>

        <Alert color='blue' variant='light'>
          <Text size='sm'>
            <strong>Tip:</strong> Restoring a story will make it your current
            active story. You can then continue reading from where you left off.
          </Text>
        </Alert>
      </Stack>
    </Container>
  );
};

export default ArchivedStoriesView;
