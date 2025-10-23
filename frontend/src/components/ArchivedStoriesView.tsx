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
} from '@mantine/core';
import {
  IconArchive,
  IconBook,
  IconRestore,
  IconCalendar,
  IconLanguage,
  IconSearch,
} from '@tabler/icons-react';
import { Story } from '../api/storyApi';

interface ArchivedStoriesViewProps {
  archivedStories: Story[];
  isLoading: boolean;
  onUnarchive: (storyId: number) => Promise<void>;
  onCreateNew: () => void;
  hideCreateButton?: boolean;
}

const ArchivedStoriesView: React.FC<ArchivedStoriesViewProps> = ({
  archivedStories,
  isLoading,
  onUnarchive,
  onCreateNew,
  hideCreateButton = false,
}) => {
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
        <Stack spacing='md' align='center'>
          <Loader size='lg' />
          <Text color='dimmed'>Loading archived stories...</Text>
        </Stack>
      </Container>
    );
  }

  if (!archivedStories || archivedStories.length === 0) {
    return (
      <Container size='sm' py='xl'>
        <Stack spacing='lg' align='center'>
          <Group position='center' spacing='xs'>
            <IconBook size={32} />
            <Title order={2}>Story Mode</Title>
          </Group>

          <Text size='lg' color='dimmed' align='center'>
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
      <Stack spacing='md'>
        {/* Create New Story Button - Moved to top */}
        {!hideCreateButton && (
          <Group position='center'>
            <Button
              leftSection={<IconBook size={16} />}
              size='lg'
              onClick={onCreateNew}
            >
              Create New Story
            </Button>
          </Group>
        )}

        {/* Archived Stories Section */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <IconArchive size={24} />
          <Title order={2}>Archived Stories</Title>
        </div>

        <Text color='dimmed' size='sm'>
          You have {archivedStories.filter(story => story.status === 'archived').length} archived{' '}
          {archivedStories.filter(story => story.status === 'archived').length === 1 ? 'story' : 'stories'}. You can restore
          any of these to continue reading or create a new story.
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
            <Stack spacing='xs'>
              {filteredStories.length === 0 ? (
                <Text color='dimmed' align='center' py='xl'>
                  {searchQuery
                    ? 'No stories match your search.'
                    : 'No archived stories found.'}
                </Text>
              ) : (
                filteredStories.map((story, index) => (
                  <Group
                    key={story.id}
                    position='apart'
                    align='center'
                    p='sm'
                    sx={theme => ({
                      backgroundColor:
                        index % 2 === 0 ? theme.colors.gray[0] : 'transparent',
                      '&:hover': {
                        backgroundColor: theme.colors.gray[1],
                      },
                    })}
                  >
                    <div style={{ flex: 1 }}>
                      <Group position='apart' mb='xs'>
                        <Title order={4} style={{ margin: 0 }}>
                          {story.title}
                        </Title>
                        <Badge color='gray' variant='light'>
                          {story.status}
                        </Badge>
                      </Group>

                      <Group spacing='lg'>
                        <Group spacing='xs'>
                          <IconLanguage size={14} />
                          <Text size='sm' color='dimmed'>
                            {story.language}
                          </Text>
                        </Group>

                        <Group spacing='xs'>
                          <IconCalendar size={14} />
                          <Text size='sm' color='dimmed'>
                            {new Date(story.created_at).toLocaleDateString()}
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
                      leftSection={<IconRestore size={16} />}
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
