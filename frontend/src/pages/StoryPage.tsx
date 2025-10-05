import React, { useState } from 'react';
import {
  Container,
  Title,
  Text,
  Group,
  Button,
  Modal,
  Stack,
  Alert,
} from '@mantine/core';
import {
  IconBook,
  IconPlus,
  IconArchive,
  IconFileDownload,
  IconEye,
  IconList,
} from '@tabler/icons-react';

import { useStory } from '../hooks/useStory';
import CreateStoryForm from '../components/CreateStoryForm';
import ArchivedStoriesView from '../components/ArchivedStoriesView';
import StorySectionView from '../components/StorySectionView';
import StoryReadingView from '../components/StoryReadingView';

const StoryPage: React.FC = () => {
  const {
    currentStory,
    archivedStories,
    sections,
    currentSectionIndex,
    viewMode,
    isLoading,
    isLoadingArchivedStories,
    error,
    hasCurrentStory,
    currentSection,
    currentSectionWithQuestions,
    canGenerateToday,
    isGenerating,
    createStory,
    archiveStory,
    setCurrentStory,
    generateNextSection,
    exportStoryPDF,
    goToNextSection,
    goToPreviousSection,
    setViewMode,
  } = useStory();

  const [showCreateModal, setShowCreateModal] = useState(false);
  const [isCreatingStory, setIsCreatingStory] = useState(false);

  const handleCreateStory = async (data: {
    title: string;
    subject?: string;
    author_style?: string;
    time_period?: string;
    genre?: string;
    tone?: string;
    character_names?: string;
    custom_instructions?: string;
    section_length_override?: string | null;
  }) => {
    setIsCreatingStory(true);
    try {
      await createStory(data);
      setShowCreateModal(false);
    } finally {
      setIsCreatingStory(false);
    }
  };

  const handleArchiveStory = async () => {
    if (currentStory) {
      await archiveStory(currentStory.id!);
    }
  };

  const handleUnarchiveStory = async (storyId: number) => {
    await setCurrentStory(storyId);
  };

  const handleExportStory = async () => {
    if (currentStory) {
      await exportStoryPDF(currentStory.id!);
    }
  };

  const handleViewModeChange = () => {
    setViewMode(viewMode === 'section' ? 'reading' : 'section');
  };

  // Show archived stories if no current story but archived stories exist
  if (
    !hasCurrentStory &&
    !isLoading &&
    archivedStories &&
    archivedStories.length > 0
  ) {
    return (
      <ArchivedStoriesView
        archivedStories={archivedStories}
        isLoading={isLoadingArchivedStories}
        onUnarchive={handleUnarchiveStory}
        onCreateNew={() => setShowCreateModal(true)}
      />
    );
  }

  // Show create form if no current story and no archived stories
  if (
    !hasCurrentStory &&
    !isLoading &&
    (!archivedStories || archivedStories.length === 0)
  ) {
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

          <CreateStoryForm
            onSubmit={handleCreateStory}
            loading={isCreatingStory}
          />

          <Alert color='blue' variant='light'>
            <Text size='sm'>
              <strong>How it works:</strong> Create a story with custom
              parameters, then read new sections daily. Each section includes
              comprehension questions to help you learn and practice your target
              language.
            </Text>
          </Alert>
        </Stack>
      </Container>
    );
  }

  // Show loading state
  if (isLoading) {
    return (
      <Container size='lg' py='xl'>
        <Stack spacing='md' align='center'>
          <Title order={3}>Loading Story...</Title>
          <Text color='dimmed'>Please wait while we load your story.</Text>
        </Stack>
      </Container>
    );
  }

  // Show error state
  if (error) {
    return (
      <Container size='lg' py='xl'>
        <Alert color='red' variant='light' title='Error'>
          <Text>{error}</Text>
          <Button
            variant='light'
            onClick={() => window.location.reload()}
            mt='sm'
          >
            Try Again
          </Button>
        </Alert>
      </Container>
    );
  }

  // Show generating state (informational, not an error)
  if (isGenerating && currentStory && 'message' in currentStory) {
    return (
      <Container size='lg' py='xl'>
        <Alert color='blue' variant='light'>
          <Text>{currentStory.message}</Text>
        </Alert>
      </Container>
    );
  }

  // Show main story interface
  return (
    <Container size='lg' py='lg'>
      <Stack spacing='lg'>
        {/* Header */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <IconBook size={24} />
            <Title order={3}>{currentStory?.title}</Title>
          </div>

          <div style={{ display: 'flex', gap: '8px' }}>
            {/* Export Button */}
            {currentStory && (
              <Button
                variant='light'
                leftIcon={<IconFileDownload size={16} />}
                onClick={handleExportStory}
                size='sm'
              >
                Export PDF
              </Button>
            )}

            {/* Archive Button */}
            {currentStory && (
              <Button
                variant='outline'
                leftIcon={<IconArchive size={16} />}
                onClick={handleArchiveStory}
                size='sm'
                color='orange'
              >
                Archive
              </Button>
            )}

            {/* New Story Button */}
            <Button
              variant='outline'
              leftIcon={<IconPlus size={16} />}
              onClick={() => setShowCreateModal(true)}
              size='sm'
            >
              New Story
            </Button>
          </div>
        </div>

        {/* View Mode Toggle */}
        <Group position='center'>
          <Button
            variant={viewMode === 'section' ? 'filled' : 'light'}
            leftIcon={<IconList size={16} />}
            onClick={() => setViewMode('section')}
            size='sm'
          >
            Section View
          </Button>
          <Button
            variant={viewMode === 'reading' ? 'filled' : 'light'}
            leftIcon={<IconEye size={16} />}
            onClick={() => setViewMode('reading')}
            size='sm'
          >
            Reading View
          </Button>
        </Group>

        {/* Story Content */}
        {viewMode === 'section' ? (
          <StorySectionView
            section={currentSection}
            sectionWithQuestions={currentSectionWithQuestions}
            sectionIndex={currentSectionIndex}
            totalSections={sections.length}
            canGenerateToday={canGenerateToday}
            isGenerating={isGenerating}
            onGenerateNext={() =>
              currentStory && generateNextSection(currentStory.id!)
            }
            onPrevious={goToPreviousSection}
            onNext={goToNextSection}
            onViewModeChange={handleViewModeChange}
            viewMode={viewMode}
          />
        ) : (
          <StoryReadingView story={currentStory} isGenerating={isGenerating} />
        )}
      </Stack>

      {/* Create Story Modal - Available in all story states */}
      <Modal
        opened={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title='Create New Story'
        size='lg'
      >
        <CreateStoryForm
          onSubmit={handleCreateStory}
          loading={isCreatingStory}
        />
      </Modal>
    </Container>
  );
};

export default StoryPage;
