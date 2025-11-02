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
  Paper,
  Badge,
  Divider,
  ScrollArea,
  Tooltip,
  Box,
} from '@mantine/core';
import {
  IconBook,
  IconBook2,
  IconMessage,
  IconPlayerPlay,
  IconPlayerPause,
  IconChevronLeft,
  IconChevronRight,
  IconChevronsLeft,
  IconChevronsRight,
} from '@tabler/icons-react';
import TTSButton from '../../components/TTSButton';
import { useGetV1PreferencesLearning } from '../../api/api';
import { defaultVoiceForLanguage } from '../../utils/tts';
import { useTheme } from '../../contexts/ThemeContext';
import { fontScaleMap } from '../../theme/theme';

import { useStory } from '../../hooks/useStory';
import { splitIntoParagraphs } from '../../utils/passage';
import CreateStoryForm from '../../components/CreateStoryForm';
import ArchivedStoriesView from '../../components/ArchivedStoriesView';
import StoryGenerationErrorModal from '../../components/StoryGenerationErrorModal';
import { SnippetHighlighter } from '../../components/SnippetHighlighter';
import { useSectionSnippets } from '../../hooks/useSectionSnippets';
import { useStorySnippets } from '../../hooks/useStorySnippets';
import {
  CreateStoryRequest,
  StoryWithSections,
  StorySection,
  StorySectionQuestion,
  StorySectionWithQuestions,
} from '../../api/storyApi';
import { getGeneratingMessage } from '../../utils/storyMessages';

const MobileStoryPage: React.FC = () => {
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

  const navigate = useNavigate();

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

  // Wrapper functions that update URL when navigating sections
  const handleGoToPreviousSection = () => {
    const targetIndex = currentSectionIndex - 1;
    if (targetIndex >= 0) {
      const targetSection = sections[targetIndex];
      goToPreviousSection();
      if (targetSection?.id !== undefined && currentStory?.id) {
        navigate(`/m/story/${currentStory.id}/section/${targetSection.id}`);
      }
    }
  };

  const handleGoToNextSection = () => {
    const targetIndex = currentSectionIndex + 1;
    if (targetIndex < sections.length) {
      const targetSection = sections[targetIndex];
      goToNextSection();
      if (targetSection?.id !== undefined && currentStory?.id) {
        navigate(`/m/story/${currentStory.id}/section/${targetSection.id}`);
      }
    }
  };

  const handleGoToFirstSection = () => {
    const firstSection = sections[0];
    goToFirstSection();
    if (firstSection?.id !== undefined && currentStory?.id) {
      navigate(`/m/story/${currentStory.id}/section/${firstSection.id}`);
    }
  };

  const handleGoToLastSection = () => {
    const lastSection = sections[sections.length - 1];
    goToLastSection();
    if (lastSection?.id !== undefined && currentStory?.id) {
      navigate(`/m/story/${currentStory.id}/section/${lastSection.id}`);
    }
  };

  // Update URL when story loads or section changes without a section parameter in URL
  useEffect(() => {
    if (currentStory && !sectionIdParam && currentSection?.id) {
      navigate(`/m/story/${currentStory.id}/section/${currentSection.id}`, {
        replace: true,
      });
    }
  }, [currentStory?.id, currentSection?.id, navigate, sectionIdParam]);

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
    <Container
      size='lg'
      py='lg'
      style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}
    >
      <Stack
        gap='lg'
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
      >
        {/* Header */}
        <Paper p='sm' radius='md'>
          <Group justify='space-between' align='center'>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <IconBook size={20} />
              <Title order={5}>{currentStory?.title}</Title>
            </div>

            <Group gap={2}>
              {/* View Mode Toggle */}
              <Button
                variant={viewMode === 'section' ? 'filled' : 'light'}
                size='xs'
                onClick={() => setViewMode('section')}
                leftSection={<IconBook2 size={12} />}
                styles={{
                  root: { padding: '1px 8px' },
                }}
              >
                Section
              </Button>
              <Button
                variant={viewMode === 'reading' ? 'filled' : 'light'}
                size='xs'
                onClick={() => setViewMode('reading')}
                leftSection={<IconMessage size={12} />}
                styles={{
                  root: { padding: '1px 8px' },
                }}
              >
                Reading
              </Button>
            </Group>
          </Group>
        </Paper>

        {/* Story Content */}
        {viewMode === 'section' ? (
          <MobileStorySectionView
            section={currentSection}
            sectionWithQuestions={currentSectionWithQuestions}
            sectionIndex={currentSectionIndex}
            totalSections={sections.length}
            canGenerateToday={canGenerateToday}
            isGenerating={isGenerating}
            generationDisabledReason={generationDisabledReason}
            story={currentStory}
            onGenerateNext={() =>
              currentStory && generateNextSection(currentStory.id!)
            }
            onToggleAutoGeneration={() =>
              currentStory &&
              toggleAutoGeneration(
                currentStory.id!,
                !currentStory.auto_generation_paused
              )
            }
            onPrevious={handleGoToPreviousSection}
            onNext={handleGoToNextSection}
            onFirst={handleGoToFirstSection}
            onLast={handleGoToLastSection}
          />
        ) : (
          <MobileStoryReadingView
            story={currentStory}
            isGenerating={isGenerating}
          />
        )}

        {/* Archive Button */}
        {currentStory && (
          <Paper p='md' radius='md'>
            <Group justify='center'>
              <Button
                variant='outline'
                color='orange'
                onClick={handleArchiveStory}
                size='md'
              >
                Archive Story
              </Button>
              <Button
                variant='outline'
                onClick={() => setShowCreateModal(true)}
                size='md'
              >
                New Story
              </Button>
            </Group>
          </Paper>
        )}
      </Stack>

      {/* Generation Error Modal - Main story interface */}
      <StoryGenerationErrorModal
        isOpen={generationErrorModal.isOpen}
        onClose={closeGenerationErrorModal}
        errorMessage={generationErrorModal.errorMessage}
      />

      {/* Create Story Modal */}
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

interface MobileStorySectionViewProps {
  section: StorySection | null;
  sectionWithQuestions: StorySectionWithQuestions | null;
  sectionIndex: number;
  totalSections: number;
  canGenerateToday: boolean;
  isGenerating: boolean;
  generationDisabledReason?: string;
  story: StoryWithSections | null;
  onGenerateNext: () => void;
  onToggleAutoGeneration: () => void;
  onPrevious: () => void;
  onNext: () => void;
  onFirst: () => void;
  onLast: () => void;
}

const MobileStorySectionView: React.FC<MobileStorySectionViewProps> = ({
  section,
  sectionWithQuestions,
  sectionIndex,
  totalSections,
  canGenerateToday,
  isGenerating,
  generationDisabledReason,
  story,
  onGenerateNext,
  onToggleAutoGeneration,
  onPrevious,
  onNext,
  onFirst,
  onLast,
}) => {
  // Get user learning preferences for preferred voice
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();

  // Get font size from theme context
  const { fontSize } = useTheme();

  // Fetch snippets for the current section
  const { snippets } = useSectionSnippets(section?.id);

  // State to track if we should hide double arrows due to overflow
  const [shouldHideDoubleArrows, setShouldHideDoubleArrows] =
    React.useState(false);
  const navContainerRef = React.useRef<HTMLDivElement>(null);

  // Check for overflow on mount and when content changes
  React.useEffect(() => {
    const checkOverflow = () => {
      if (navContainerRef.current) {
        const container = navContainerRef.current;
        const leftGroup = container.querySelector(
          '[data-nav-left-group]'
        ) as HTMLElement;
        const rightGroup = container.querySelector(
          '[data-nav-right-group]'
        ) as HTMLElement;

        if (leftGroup && rightGroup) {
          // Check if any child in leftGroup wraps to a different line
          const leftChildren = leftGroup.children;
          let previousBottom = 0;
          let hasWrapped = false;

          for (let i = 0; i < leftChildren.length; i++) {
            const rect = (
              leftChildren[i] as HTMLElement
            ).getBoundingClientRect();
            if (i > 0 && rect.top > previousBottom + 1) {
              // This element is on a different line (with 1px tolerance)
              hasWrapped = true;
              break;
            }
            previousBottom = rect.bottom;
          }

          // If the left group has wrapped, hide the double arrows
          if (!hasWrapped) {
            // Also check if right group wraps (though it typically won't)
            const rightChildren = rightGroup.children;
            previousBottom = 0;
            for (let i = 0; i < rightChildren.length; i++) {
              const rect = (
                rightChildren[i] as HTMLElement
              ).getBoundingClientRect();
              if (i > 0 && rect.top > previousBottom + 1) {
                hasWrapped = true;
                break;
              }
              previousBottom = rect.bottom;
            }
          }

          setShouldHideDoubleArrows(hasWrapped);
        }
      }
    };

    // Delay the check to ensure the DOM has updated
    const timeoutId = setTimeout(checkOverflow, 100);

    // Also check on window resize
    window.addEventListener('resize', checkOverflow);

    return () => {
      clearTimeout(timeoutId);
      window.removeEventListener('resize', checkOverflow);
    };
  }, [sectionIndex, totalSections]);
  if (!section) {
    return (
      <Paper p='xl' radius='md' style={{ textAlign: 'center' }}>
        <IconBook size={48} style={{ opacity: 0.5, marginBottom: 16 }} />
        <Title order={4}>No section to display</Title>
        <Text color='dimmed'>
          Create a new story or select a section to view.
        </Text>
      </Paper>
    );
  }

  return (
    <Stack gap='md'>
      {/* Section Header */}
      <Paper p='sm' radius='md'>
        {/* Section Navigation */}
        <div
          ref={navContainerRef}
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            width: '100%',
          }}
        >
          <Group gap={4} data-nav-left-group>
            <Button
              variant='light'
              size='xs'
              onClick={onFirst}
              disabled={sectionIndex === 0}
              styles={{
                root: {
                  padding: '2px 6px',
                  minHeight: '28px',
                  display: shouldHideDoubleArrows ? 'none' : undefined,
                },
              }}
            >
              <IconChevronsLeft size={16} />
            </Button>

            <Button
              variant='light'
              size='xs'
              onClick={onPrevious}
              disabled={sectionIndex === 0}
              styles={{
                root: {
                  padding: '2px 6px',
                  minHeight: '28px',
                },
              }}
            >
              <IconChevronLeft size={16} />
            </Button>

            <Text
              size='xs'
              color='dimmed'
              style={{ minWidth: '50px', textAlign: 'center' }}
            >
              {sectionIndex + 1} / {totalSections}
            </Text>

            <Button
              variant='light'
              size='xs'
              onClick={onNext}
              disabled={sectionIndex >= totalSections - 1}
              styles={{
                root: {
                  padding: '2px 6px',
                  minHeight: '28px',
                },
              }}
            >
              <IconChevronRight size={16} />
            </Button>

            <Button
              variant='light'
              size='xs'
              onClick={onLast}
              disabled={sectionIndex >= totalSections - 1}
              styles={{
                root: {
                  padding: '2px 6px',
                  minHeight: '28px',
                  display: shouldHideDoubleArrows ? 'none' : undefined,
                },
              }}
            >
              <IconChevronsRight size={16} />
            </Button>
          </Group>

          <Group gap={4} data-nav-right-group>
            {/* Pause/Resume Auto-Generation Button */}
            {story && (
              <Tooltip
                label={
                  story.auto_generation_paused
                    ? 'Resume automatic story generation'
                    : 'Pause automatic story generation'
                }
                position='bottom'
                withArrow
              >
                <Button
                  variant='light'
                  onClick={onToggleAutoGeneration}
                  size='xs'
                  px='xs'
                  color={story.auto_generation_paused ? 'green' : 'blue'}
                  styles={{
                    root: {
                      padding: '2px 6px',
                      minHeight: '28px',
                    },
                  }}
                >
                  {story.auto_generation_paused ? (
                    <IconPlayerPlay size={16} />
                  ) : (
                    <IconPlayerPause size={16} />
                  )}
                </Button>
              </Tooltip>
            )}

            <Badge variant='outline' size='sm'>
              {section.language_level}
            </Badge>
          </Group>
        </div>
      </Paper>

      {/* Section Content */}
      <Paper
        p='lg'
        radius='md'
        style={{ flex: 1, overflow: 'hidden', position: 'relative' }}
      >
        <Box style={{ position: 'absolute', top: 12, right: 12, zIndex: 10 }}>
          <TTSButton
            getText={() => section.content || ''}
            getVoice={() => {
              const saved = (userLearningPrefs?.tts_voice || '').trim();
              if (saved) return saved;
              return (
                defaultVoiceForLanguage(section.language_level) || undefined
              );
            }}
            getMetadata={() => ({
              title: story?.title || 'Story',
              language: story?.language,
              level: section.language_level,
            })}
            size='md'
            ariaLabel='Section audio'
          />
        </Box>
        <ScrollArea style={{ height: '100%' }}>
          <div
            className='selectable-text'
            data-allow-translate='true'
            style={{ padding: '1rem 56px 1rem 0' }}
          >
            {(() => {
              const paragraphs = splitIntoParagraphs(section.content, 2);
              return (
                <div>
                  {paragraphs.map((paragraph, index) => (
                    <div key={index}>
                      <SnippetHighlighter
                        text={paragraph}
                        snippets={snippets}
                        component={Text}
                        componentProps={{
                          style: {
                            lineHeight: 1.6,
                            fontSize: `${16 * fontScaleMap[fontSize]}px`,
                            whiteSpace: 'pre-wrap',
                            paddingRight: '4px',
                            marginBottom:
                              index < paragraphs.length - 1 ? '1rem' : 0,
                          },
                        }}
                      />
                    </div>
                  ))}
                </div>
              );
            })()}
          </div>
        </ScrollArea>
      </Paper>

      {/* Comprehension Questions */}
      {sectionWithQuestions?.questions &&
      sectionWithQuestions.questions.length > 0 ? (
        <Paper p='md' radius='md'>
          <Title order={5} mb='sm'>
            Comprehension Questions
          </Title>
          <Stack gap='sm'>
            {sectionWithQuestions.questions.map(
              (question: StorySectionQuestion, index: number) => (
                <MobileStoryQuestionCard
                  key={question.id || index}
                  question={question}
                />
              )
            )}
          </Stack>
        </Paper>
      ) : (
        <Alert color='gray' variant='light'>
          No questions available for this section yet.
        </Alert>
      )}

      {/* Bottom Section Navigation */}
      <Paper p='sm' radius='md'>
        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-start',
            alignItems: 'center',
            width: '100%',
          }}
        >
          <Group gap={4}>
            <Button
              variant='light'
              size='xs'
              onClick={onFirst}
              disabled={sectionIndex === 0}
              styles={{
                root: {
                  padding: '2px 6px',
                  minHeight: '28px',
                },
              }}
            >
              ?
            </Button>

            <Button
              variant='light'
              size='xs'
              onClick={onPrevious}
              disabled={sectionIndex === 0}
              styles={{
                root: {
                  padding: '2px 6px',
                  minHeight: '28px',
                },
              }}
            >
              ?
            </Button>

            <Text
              size='xs'
              color='dimmed'
              style={{ minWidth: '50px', textAlign: 'center' }}
            >
              {sectionIndex + 1} / {totalSections}
            </Text>

            <Button
              variant='light'
              size='xs'
              onClick={onNext}
              disabled={sectionIndex >= totalSections - 1}
              styles={{
                root: {
                  padding: '2px 6px',
                  minHeight: '28px',
                },
              }}
            >
              ?
            </Button>

            <Button
              variant='light'
              size='xs'
              onClick={onLast}
              disabled={sectionIndex >= totalSections - 1}
              styles={{
                root: {
                  padding: '2px 6px',
                  minHeight: '28px',
                },
              }}
            >
              ?
            </Button>
          </Group>
        </div>
      </Paper>

      {/* Generate Next Section */}
      <Paper p='md' radius='md'>
        <Group justify='space-between' align='center'>
          <div>
            <Title order={5}>Continue the Story</Title>
            <Text size='sm' color='dimmed'>
              Generate the next section of your story
            </Text>
          </div>
          <Button
            size='sm'
            onClick={onGenerateNext}
            loading={isGenerating}
            disabled={!canGenerateToday || isGenerating}
            color='blue'
            leftSection={<IconBook size={14} />}
          >
            {isGenerating ? 'Generating...' : 'Generate Next Section'}
          </Button>
        </Group>
        {!canGenerateToday && generationDisabledReason && (
          <Text size='xs' color='dimmed' mt='xs'>
            {generationDisabledReason}
          </Text>
        )}
      </Paper>
    </Stack>
  );
};

// Simplified question component for mobile
interface MobileStoryQuestionCardProps {
  question: StorySectionQuestion;
}

const MobileStoryQuestionCard: React.FC<MobileStoryQuestionCardProps> = ({
  question,
}) => {
  const [selectedAnswer, setSelectedAnswer] = React.useState<number | null>(
    null
  );
  const [showResult, setShowResult] = React.useState(false);

  const handleSubmit = () => {
    setShowResult(true);
  };

  const handleReset = () => {
    setSelectedAnswer(null);
    setShowResult(false);
  };

  return (
    <Paper p='sm' radius='sm' style={{ backgroundColor: '#f8f9fa' }}>
      <Text size='sm' fw={500} mb='xs'>
        {question.question_text}
      </Text>

      <Stack gap='xs'>
        {question.options?.map((option: string, index: number) => (
          <div key={index}>
            <input
              type='radio'
              id={`option-${index}`}
              name={`question-${question.id}`}
              value={index}
              checked={selectedAnswer === index}
              onChange={() => setSelectedAnswer(index)}
              disabled={showResult}
            />
            <label htmlFor={`option-${index}`} style={{ marginLeft: 8 }}>
              {option}
            </label>
          </div>
        ))}
      </Stack>

      <Group justify='space-between' mt='xs'>
        {!showResult ? (
          <Button
            size='xs'
            onClick={handleSubmit}
            disabled={selectedAnswer === null}
          >
            Submit Answer
          </Button>
        ) : (
          <>
            <Text
              size='xs'
              color={
                selectedAnswer === question.correct_answer_index
                  ? 'green'
                  : 'red'
              }
            >
              {selectedAnswer === question.correct_answer_index
                ? '? Correct!'
                : '? Incorrect'}
            </Text>
            <Button size='xs' variant='light' onClick={handleReset}>
              Try Again
            </Button>
          </>
        )}
      </Group>

      {showResult && question.explanation && (
        <Alert color='blue' variant='light' mt='xs'>
          <Text size='xs'>{question.explanation}</Text>
        </Alert>
      )}
    </Paper>
  );
};

interface MobileStoryReadingViewProps {
  story: StoryWithSections | null;
  isGenerating?: boolean;
}

const MobileStoryReadingView: React.FC<MobileStoryReadingViewProps> = ({
  story,
  isGenerating = false,
}) => {
  // Get user learning preferences for preferred voice
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();

  // Get font size from theme context
  const { fontSize } = useTheme();

  // Fetch snippets for the entire story
  const { snippets } = useStorySnippets(story?.id);
  if (!story) {
    return (
      <Paper p='xl' radius='md' style={{ textAlign: 'center' }}>
        <IconBook size={48} style={{ opacity: 0.5, marginBottom: 16 }} />
        <Title order={4}>No story to display</Title>
        <Text color='dimmed'>Create a new story to start reading.</Text>
      </Paper>
    );
  }

  if (!story.sections || story.sections.length === 0) {
    if (isGenerating) {
      return (
        <Paper p='xl' radius='md' style={{ textAlign: 'center' }}>
          <Stack gap='md' align='center'>
            <IconBook size={48} style={{ opacity: 0.5 }} />
            <Title order={4}>Generating Your Story</Title>
            <Text color='dimmed' ta='center'>
              We're creating the first section of your story.
            </Text>
          </Stack>
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
    <Stack gap='md'>
      {/* Story Content */}
      <Paper
        p='lg'
        radius='md'
        style={{ flex: 1, overflow: 'hidden', position: 'relative' }}
      >
        <Box style={{ position: 'absolute', top: 12, right: 12, zIndex: 10 }}>
          <TTSButton
            getText={() =>
              story.sections?.map(s => s.content).join('\n\n') || ''
            }
            getVoice={() => {
              const saved = (userLearningPrefs?.tts_voice || '').trim();
              if (saved) return saved;
              return defaultVoiceForLanguage(story.language) || undefined;
            }}
            getMetadata={() => ({
              title: story.title || 'Story',
              language: story.language,
              level: story.sections?.[0]?.language_level,
            })}
            size='md'
            ariaLabel='Story audio'
          />
        </Box>
        <ScrollArea style={{ height: '100%' }}>
          <div
            className='selectable-text'
            data-allow-translate='true'
            style={{ padding: '1rem 56px 1rem 20px' }}
          >
            <Stack gap='lg'>
              {/* Story Sections */}
              {story.sections?.map((section: StorySection, index: number) => (
                <div key={section.id || index}>
                  <Divider my='md' />
                  {(() => {
                    const paragraphs = splitIntoParagraphs(section.content, 3);
                    return (
                      <div>
                        {paragraphs.map((paragraph, paraIndex) => (
                          <SnippetHighlighter
                            key={paraIndex}
                            text={paragraph}
                            snippets={snippets}
                            component={Text}
                            componentProps={{
                              style: {
                                lineHeight: 1.7,
                                fontSize: `${16 * fontScaleMap[fontSize]}px`,
                                whiteSpace: 'pre-wrap',
                                marginBottom:
                                  paraIndex < paragraphs.length - 1
                                    ? '1.5rem'
                                    : index < (story.sections?.length || 0) - 1
                                      ? '1.5rem'
                                      : '1rem',
                              },
                            }}
                          />
                        ))}
                      </div>
                    );
                  })()}
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
                <Text size='sm' color='blue' ta='center'>
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

export default MobileStoryPage;
