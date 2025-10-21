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
  Tooltip,
} from '@mantine/core';
import {
  IconBook,
  IconArchive,
  IconEye,
  IconLayoutList,
  IconDownload,
  IconPlus,
  IconPlayerPause,
  IconPlayerPlay,
} from '@tabler/icons-react';

import { useStory } from '../hooks/useStory';
import CreateStoryForm from '../components/CreateStoryForm';
import ArchivedStoriesView from '../components/ArchivedStoriesView';
import StorySectionView from '../components/StorySectionView';
import StoryReadingView from '../components/StoryReadingView';
import StoryGenerationErrorModal from '../components/StoryGenerationErrorModal';
import { CreateStoryRequest } from '../api/storyApi';

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
    generationType,
    generationDisabledReason,
    createStory,
    archiveStory,
    setCurrentStory,
    generateNextSection,
    exportStoryPDF,
    goToNextSection,
    goToPreviousSection,
    goToFirstSection,
    goToLastSection,
    setViewMode,
    generationErrorModal,
    closeGenerationErrorModal,
    toggleAutoGeneration,
  } = useStory();

  const [showCreateModal, setShowCreateModal] = useState(false);
  const [isCreatingStory, setIsCreatingStory] = useState(false);

  const handleCreateStory = async (data: CreateStoryRequest) => {
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

  const handleToggleAutoGeneration = async () => {
    if (currentStory) {
      await toggleAutoGeneration(
        currentStory.id!,
        !currentStory.auto_generation_paused
      );
    }
  };

  // Show archived stories if no current story but archived stories exist
  if (
    !hasCurrentStory &&
    !isLoading &&
    archivedStories &&
    archivedStories.length > 0
  ) {
    return (
      <>
        <ArchivedStoriesView
          archivedStories={archivedStories}
          isLoading={isLoadingArchivedStories}
          onUnarchive={handleUnarchiveStory}
          onCreateNew={() => setShowCreateModal(true)}
        />
        {/* Create Story Modal - Available when showing archived stories */}
        {showCreateModal && (
          <Modal
            opened={true}
            onClose={() => setShowCreateModal(false)}
            title='Create New Story'
            size='xl'
            centered
          >
            <CreateStoryForm
              onSubmit={handleCreateStory}
              loading={isCreatingStory}
            />
          </Modal>
        )}

        {/* Generation Error Modal */}
        <StoryGenerationErrorModal
          isOpen={generationErrorModal.isOpen}
          onClose={closeGenerationErrorModal}
          errorMessage={generationErrorModal.errorMessage}
          errorDetails={generationErrorModal.errorDetails}
        />
      </>
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
        <Stack gap='lg' align='center'>
          <Group justify='center' gap='xs'>
            <IconBook size={32} />
            <Title order={2}>Story Mode</Title>
          </Group>

          <Text size='lg' c='dimmed' ta='center'>
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
        <Stack gap='md'>
          <Title order={3}>Loading Story...</Title>
          <Text c='dimmed'>Please wait while we load your story.</Text>
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

  // Show generating state (informational, not an error) - check this before main interface
  if (isGenerating) {
    const getGeneratingMessage = (): string => {
      if (currentStory && 'message' in currentStory && currentStory.message) {
        return currentStory.message as string;
      }

      switch (generationType) {
        case 'story':
          return 'Story created successfully. The first section is being generated. Please check back shortly.';
        case 'section':
          return 'Generating the next section of your story. Please check back shortly.';
        default:
          return 'Generating content. Please check back shortly.';
      }
    };

    return (
      <Container size='lg' py='xl'>
        <Alert color='blue' variant='light'>
          <Text>{getGeneratingMessage()}</Text>
        </Alert>
      </Container>
    );
  }

  // Show main story interface
  return (
    <Container size='lg' py='lg'>
      <Stack gap='lg'>
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
                leftSection={<IconDownload size={16} />}
                onClick={handleExportStory}
                size='sm'
              >
                Export PDF
              </Button>
            )}

            {/* Pause/Resume Auto-Generation Button */}
            {currentStory && (
              <Tooltip
                label={
                  currentStory.auto_generation_paused
                    ? 'Resume automatic story generation. New sections will be generated daily.'
                    : 'Pause automatic story generation. No new sections will be generated until resumed.'
                }
                position='bottom'
                withArrow
              >
                <Button
                  variant='light'
                  leftSection={
                    currentStory.auto_generation_paused ? (
                      <IconPlayerPlay size={16} />
                    ) : (
                      <IconPlayerPause size={16} />
                    )
                  }
                  onClick={handleToggleAutoGeneration}
                  size='sm'
                  color={currentStory.auto_generation_paused ? 'green' : 'blue'}
                >
                  {currentStory.auto_generation_paused ? 'Resume' : 'Pause'}{' '}
                  Auto
                </Button>
              </Tooltip>
            )}

            {/* Archive Button */}
            {currentStory && (
              <Button
                variant='outline'
                leftSection={<IconArchive size={16} />}
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
              leftSection={<IconPlus size={16} />}
              onClick={() => setShowCreateModal(true)}
              size='sm'
            >
              New Story
            </Button>
          </div>
        </div>

        {/* View Mode Toggle */}
        <Group justify='center'>
          <Button
            variant={viewMode === 'section' ? 'filled' : 'light'}
            leftSection={<IconLayoutList size={16} />}
            onClick={() => setViewMode('section')}
            size='sm'
          >
            Section View
          </Button>
          <Button
            variant={viewMode === 'reading' ? 'filled' : 'light'}
            leftSection={<IconEye size={16} />}
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
            generationDisabledReason={generationDisabledReason}
            onGenerateNext={() =>
              currentStory && generateNextSection(currentStory.id!)
            }
            onPrevious={goToPreviousSection}
            onNext={goToNextSection}
            onFirst={goToFirstSection}
            onLast={goToLastSection}
          />
        ) : (
          <StoryReadingView story={currentStory} isGenerating={isGenerating} />
        )}
      </Stack>

      {/* Generation Error Modal - Main story interface */}
      <StoryGenerationErrorModal
        isOpen={generationErrorModal.isOpen}
        onClose={closeGenerationErrorModal}
        errorMessage={generationErrorModal.errorMessage}
        errorDetails={generationErrorModal.errorDetails}
      />

      {/* Create Story Modal - Main story interface */}
      {showCreateModal && (
        <Modal
          opened={true}
          onClose={() => setShowCreateModal(false)}
          title='Create New Story'
          size='xl'
          centered
        >
          <CreateStoryForm
            onSubmit={handleCreateStory}
            loading={isCreatingStory}
          />
        </Modal>
      )}
    </Container>
  );
};

export default StoryPage;
