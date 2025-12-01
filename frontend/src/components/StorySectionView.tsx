import React, { useEffect, useRef } from 'react';
import { Tooltip, Box } from '@mantine/core';
import { defaultVoiceForLanguage } from '../utils/tts';
import { useGetV1PreferencesLearning } from '../api/api';
import { useTheme } from '../contexts/ThemeContext';
import { fontScaleMap } from '../theme/theme';
import { SnippetHighlighter } from './SnippetHighlighter';
import { useSectionSnippets } from '../hooks/useSectionSnippets';
import { useTTS } from '../hooks/useTTS';
import TTSButton from './TTSButton';
import {
  Paper,
  Title,
  Text,
  Group,
  Button,
  Badge,
  Stack,
  Alert,
} from '@mantine/core';
import {
  IconChevronLeft,
  IconChevronRight,
  IconChevronsLeft,
  IconChevronsRight,
  IconPlus,
  IconBook,
  IconLanguage,
} from '@tabler/icons-react';
import {
  StorySection,
  StorySectionQuestion,
  StorySectionWithQuestions,
} from '../api/storyApi';

interface StorySectionViewProps {
  section: StorySection | null;
  sectionWithQuestions: StorySectionWithQuestions | null;
  sectionIndex: number;
  totalSections: number;
  canGenerateToday: boolean;
  isGenerating: boolean;
  isGeneratingNextSection?: boolean;
  generationDisabledReason?: string;
  onGenerateNext: () => void;
  onPrevious: () => void;
  onNext: () => void;
  onFirst: () => void;
  onLast: () => void;
  storyTitle?: string;
  storyLanguage?: string;
}

const StorySectionView: React.FC<StorySectionViewProps> = ({
  section,
  sectionWithQuestions,
  sectionIndex,
  totalSections,
  canGenerateToday,
  isGenerating,
  isGeneratingNextSection = false,
  generationDisabledReason,
  onGenerateNext,
  onPrevious,
  onNext,
  onFirst,
  onLast,
  storyTitle,
  storyLanguage,
}) => {
  // Get user learning preferences for preferred voice
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();

  // Get font size from theme context
  const { fontSize } = useTheme();

  // Get snippets for this specific section
  const { snippets } = useSectionSnippets(section?.id);

  // Stop TTS audio when switching sections
  const { stopTTS } = useTTS();
  const sectionTopRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    return () => {
      stopTTS();
    };
  }, [section?.id, stopTTS]);

  useEffect(() => {
    if (!section || !sectionTopRef.current) {
      return;
    }

    // Wait a frame so the new section renders before scrolling
    const frameId = requestAnimationFrame(() => {
      sectionTopRef.current?.scrollIntoView({
        behavior: 'smooth',
        block: 'start',
        inline: 'nearest',
      });
    });

    return () => cancelAnimationFrame(frameId);
  }, [section?.id, section?.section_number, section?.content, sectionIndex]);

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
    <>
      <div
        ref={sectionTopRef}
        aria-hidden='true'
        style={{ height: 0, margin: 0, padding: 0 }}
      />
      <Stack spacing='md'>
        {/* Generating Alert */}
        {isGenerating && (
          <Alert color='blue' icon={<IconBook size={16} />}>
            <Text fw={500}>Generating next section...</Text>
            <Text size='sm' mt='xs'>
              Your story is being continued. This may take a moment.
            </Text>
          </Alert>
        )}

        {/* Section Header */}
        <Paper p='md' radius='md'>
          {/* Section Navigation */}
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              width: '100%',
            }}
          >
            <Group spacing='xs'>
              <Button
                variant='light'
                leftSection={<IconChevronsLeft size={16} />}
                onClick={onFirst}
                disabled={sectionIndex === 0}
                size='sm'
              >
                First
              </Button>

              <Button
                variant='light'
                leftSection={<IconChevronLeft size={16} />}
                onClick={onPrevious}
                disabled={sectionIndex === 0}
                size='sm'
              >
                Previous
              </Button>

              <Text
                size='sm'
                color='dimmed'
                style={{ minWidth: '80px', textAlign: 'center' }}
              >
                {sectionIndex + 1} of {totalSections}
              </Text>

              <Button
                variant='light'
                rightSection={<IconChevronRight size={16} />}
                onClick={onNext}
                disabled={sectionIndex >= totalSections - 1}
                size='sm'
              >
                Next
              </Button>

              <Button
                variant='light'
                rightSection={<IconChevronsRight size={16} />}
                onClick={onLast}
                disabled={sectionIndex >= totalSections - 1}
                size='sm'
              >
                Last
              </Button>
            </Group>

            <Badge variant='outline'>
              <Group spacing={4}>
                <IconLanguage size={12} />
                {section.language_level}
              </Group>
            </Badge>
          </div>
        </Paper>

        {/* Section Content */}
        <Paper p='lg' radius='md' style={{ position: 'relative' }}>
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
                title: storyTitle || 'Story',
                language: storyLanguage,
                level: section.language_level,
              })}
              size='md'
              ariaLabel='Section audio'
            />
          </Box>

          <div data-allow-translate='true'>
            <SnippetHighlighter
              text={section.content || ''}
              snippets={snippets}
              component={Text}
              componentProps={{
                style: {
                  lineHeight: 1.6,
                  fontSize: `${16 * fontScaleMap[fontSize]}px`,
                  whiteSpace: 'pre-wrap',
                  // Ensure text never overlaps the TTS icon in the top-right
                  paddingRight: '56px',
                },
              }}
            />
          </div>
        </Paper>

        {/* Comprehension Questions */}
        {sectionWithQuestions?.questions &&
        sectionWithQuestions.questions.length > 0 ? (
          <Paper p='md' radius='md'>
            <Title order={5} mb='sm'>
              Comprehension Questions
            </Title>
            <Stack spacing='sm'>
              {sectionWithQuestions.questions.map((question, index) => (
                <StoryQuestionCard
                  key={question.id || index}
                  question={question}
                />
              ))}
            </Stack>
          </Paper>
        ) : (
          <Alert color='gray' variant='light'>
            No questions available for this section yet.
          </Alert>
        )}

        {/* Bottom Section Navigation */}
        <Paper p='md' radius='md'>
          <div
            style={{
              display: 'flex',
              justifyContent: 'flex-start',
              alignItems: 'center',
              width: '100%',
            }}
          >
            <Group spacing='xs'>
              <Button
                variant='light'
                leftSection={<IconChevronsLeft size={16} />}
                onClick={onFirst}
                disabled={sectionIndex === 0}
                size='sm'
              >
                First
              </Button>

              <Button
                variant='light'
                leftSection={<IconChevronLeft size={16} />}
                onClick={onPrevious}
                disabled={sectionIndex === 0}
                size='sm'
              >
                Previous
              </Button>

              <Text
                size='sm'
                color='dimmed'
                style={{ minWidth: '80px', textAlign: 'center' }}
              >
                {sectionIndex + 1} of {totalSections}
              </Text>

              <Button
                variant='light'
                rightSection={<IconChevronRight size={16} />}
                onClick={onNext}
                disabled={sectionIndex >= totalSections - 1}
                size='sm'
              >
                Next
              </Button>

              <Button
                variant='light'
                rightSection={<IconChevronsRight size={16} />}
                onClick={onLast}
                disabled={sectionIndex >= totalSections - 1}
                size='sm'
              >
                Last
              </Button>
            </Group>
          </div>
        </Paper>

        {/* Generate Next Section */}
        <Paper p='md' radius='md'>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <div>
              <Title order={5}>Continue the Story</Title>
              <Text size='sm' color='dimmed'>
                Generate the next section of your story
              </Text>
            </div>
            <Tooltip
              label={
                !canGenerateToday && !isGeneratingNextSection
                  ? generationDisabledReason || 'Unable to generate section'
                  : 'Generate the next section of your story'
              }
              position='top'
              withArrow
            >
              <Button
                leftSection={<IconPlus size={16} />}
                onClick={onGenerateNext}
                loading={isGeneratingNextSection}
                disabled={!canGenerateToday || isGeneratingNextSection}
                color='blue'
                variant={canGenerateToday ? 'filled' : 'light'}
              >
                {isGeneratingNextSection
                  ? 'Generating...'
                  : 'Generate Next Section'}
              </Button>
            </Tooltip>
          </div>
        </Paper>
      </Stack>
    </>
  );
};

// Simple question component for story sections
interface StoryQuestionCardProps {
  question: StorySectionQuestion;
}

const StoryQuestionCard: React.FC<StoryQuestionCardProps> = ({ question }) => {
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
      <Text
        size='sm'
        weight={500}
        mb='xs'
        data-allow-translate='true'
        data-selectable-text='true'
      >
        {question.question_text}
      </Text>

      <Stack spacing='xs'>
        {question.options.map((option, index) => (
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
            <label
              htmlFor={`option-${index}`}
              style={{ marginLeft: 8 }}
              data-allow-translate='true'
              data-selectable-text='true'
            >
              {option}
            </label>
          </div>
        ))}
      </Stack>

      <Group position='apart' mt='xs'>
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
                ? '✓ Correct!'
                : '✗ Incorrect'}
            </Text>
            <Button size='xs' variant='light' onClick={handleReset}>
              Try Again
            </Button>
          </>
        )}
      </Group>

      {showResult && question.explanation && (
        <Alert color='blue' variant='light' mt='xs'>
          <Text
            size='xs'
            data-allow-translate='true'
            data-selectable-text='true'
          >
            {question.explanation}
          </Text>
        </Alert>
      )}
    </Paper>
  );
};

export default StorySectionView;
