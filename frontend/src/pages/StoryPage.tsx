import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
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
import { getGeneratingMessage } from '../utils/storyMessages';

const StoryPage: React.FC = () => {
  const { id: storyIdParam, sectionId: sectionIdParam } = useParams<{
    id?: string;
    sectionId?: string;
  }>();

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
    goToSection,
    goToNextSection,
    goToPreviousSection,
    goToFirstSection,
    goToLastSection,
    setViewMode,
    generationErrorModal,
    closeGenerationErrorModal,
    toggleAutoGeneration,
  } = useStory({ skipLocalStorage: !!sectionIdParam });

  const [showCreateModal, setShowCreateModal] = useState(false);
  const [isCreatingStory, setIsCreatingStory] = useState(false);

  // Handle URL parameters for story and section navigation
  useEffect(() => {
    if (storyIdParam && !isLoading) {
      const storyId = parseInt(storyIdParam, 10);
      if (!isNaN(storyId) && (!currentStory || currentStory.id !== storyId)) {
        setCurrentStory(storyId);
      }
    }
  }, [storyIdParam, isLoading, currentStory, setCurrentStory]);

  // Handle section ID parameter - prioritize URL over localStorage
  // This must run AFTER sections are loaded but BEFORE localStorage restoration
  useEffect(() => {
    if (sectionIdParam && currentStory && currentStory.sections) {
      const sectionId = parseInt(sectionIdParam, 10);
      if (!isNaN(sectionId)) {
        const sectionIndex = currentStory.sections.findIndex(
          section => section.id === sectionId
        );
        // Navigate if we found the section AND it's different from current
        if (sectionIndex !== -1 && sectionIndex !== currentSectionIndex) {
          goToSection(sectionIndex);
        }
      }
    }
  }, [
    sectionIdParam,
    currentStory?.sections,
    currentSectionIndex,
    goToSection,
  ]);

  const navigate = useNavigate();

  // Wrapper functions that update URL when navigating sections
  const handleGoToPreviousSection = () => {
    const targetIndex = currentSectionIndex - 1;
    if (targetIndex >= 0) {
      const targetSection = sections[targetIndex];
      goToPreviousSection();
      if (targetSection?.id !== undefined && currentStory?.id) {
        navigate(`/story/${currentStory.id}/section/${targetSection.id}`);
      }
    }
  };

  const handleGoToNextSection = () => {
    const targetIndex = currentSectionIndex + 1;
    if (targetIndex < sections.length) {
      const targetSection = sections[targetIndex];
      goToNextSection();
      if (targetSection?.id !== undefined && currentStory?.id) {
        navigate(`/story/${currentStory.id}/section/${targetSection.id}`);
      }
    }
  };

  const handleGoToFirstSection = () => {
    const firstSection = sections[0];
    goToFirstSection();
    if (firstSection?.id !== undefined && currentStory?.id) {
      navigate(`/story/${currentStory.id}/section/${firstSection.id}`);
    }
  };

  const handleGoToLastSection = () => {
    const lastSection = sections[sections.length - 1];
    goToLastSection();
    if (lastSection?.id !== undefined && currentStory?.id) {
      navigate(`/story/${currentStory.id}/section/${lastSection.id}`);
    }
  };

  // Update URL when story loads or section changes without a section parameter in URL
  useEffect(() => {
    if (currentStory && !sectionIdParam && currentSection?.id) {
      navigate(`/story/${currentStory.id}/section/${currentSection.id}`, {
        replace: true,
      });
    }
  }, [currentStory?.id, currentSection?.id, navigate, sectionIdParam]);

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
          hideCreateButton={false}
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
    const message = getGeneratingMessage(generationType, currentStory);

    // If we have archived stories, show them below the generating message
    if (archivedStories && archivedStories.length > 0) {
      return (
        <>
          <Container size='lg' py='md'>
            <Alert color='blue' variant='light'>
              <Text>{message}</Text>
            </Alert>
          </Container>
          <ArchivedStoriesView
            archivedStories={archivedStories}
            isLoading={isLoadingArchivedStories}
            onUnarchive={handleUnarchiveStory}
            onCreateNew={() => setShowCreateModal(true)}
            hideCreateButton={true}
          />
          {/* Create Story Modal - Available during generation */}
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

    // Otherwise show just the generating message
    return (
      <Container size='lg' py='xl'>
        <Alert color='blue' variant='light'>
          <Text>{message}</Text>
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

          <div style={{ display: 'flex', gap: '4px' }}>
            {/* View Mode Toggle */}
            <Tooltip label='Section View' position='bottom' withArrow>
              <Button
                variant={viewMode === 'section' ? 'filled' : 'light'}
                onClick={() => setViewMode('section')}
                size='sm'
                px='xs'
              >
                <IconLayoutList size={18} />
              </Button>
            </Tooltip>
            <Tooltip label='Reading View' position='bottom' withArrow>
              <Button
                variant={viewMode === 'reading' ? 'filled' : 'light'}
                onClick={() => setViewMode('reading')}
                size='sm'
                px='xs'
              >
                <IconEye size={18} />
              </Button>
            </Tooltip>

            {/* Export Button */}
            {currentStory && (
              <Tooltip label='Export PDF' position='bottom' withArrow>
                <Button
                  variant='light'
                  onClick={handleExportStory}
                  size='sm'
                  px='xs'
                >
                  <IconDownload size={18} />
                </Button>
              </Tooltip>
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
                  onClick={handleToggleAutoGeneration}
                  size='sm'
                  px='xs'
                  color={currentStory.auto_generation_paused ? 'green' : 'blue'}
                >
                  {currentStory.auto_generation_paused ? (
                    <IconPlayerPlay size={18} />
                  ) : (
                    <IconPlayerPause size={18} />
                  )}
                </Button>
              </Tooltip>
            )}

            {/* Archive Button */}
            {currentStory && (
              <Tooltip
                label='Archive Story (can always be restored)'
                position='bottom'
                withArrow
              >
                <Button
                  variant='outline'
                  onClick={handleArchiveStory}
                  size='sm'
                  px='xs'
                  color='orange'
                >
                  <IconArchive size={18} />
                </Button>
              </Tooltip>
            )}

            {/* New Story Button */}
            <Tooltip label='New Story' position='bottom' withArrow>
              <Button
                variant='outline'
                onClick={() => setShowCreateModal(true)}
                size='sm'
                px='xs'
                aria-label='New Story'
              >
                <IconPlus size={18} />
              </Button>
            </Tooltip>
          </div>
        </div>

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
            onPrevious={handleGoToPreviousSection}
            onNext={handleGoToNextSection}
            onFirst={handleGoToFirstSection}
            onLast={handleGoToLastSection}
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
