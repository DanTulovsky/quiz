import React from 'react';
import { ActionIcon, Tooltip, Box } from '@mantine/core';
import { Volume2, VolumeX } from 'lucide-react';
import { useTTS } from '../hooks/useTTS';
import { defaultVoiceForLanguage } from '../utils/tts';
import { useGetV1PreferencesLearning } from '../api/api';
import {
  Paper,
  Title,
  Text,
  Group,
  Button,
  Badge,
  Stack,
  Alert,
  Tooltip,
  Loader,
} from '@mantine/core';
import {
  IconChevronLeft,
  IconChevronRight,
  IconChevronsLeft,
  IconChevronsRight,
  IconPlus,
  IconBook,
  IconLanguage,
  IconFileText,
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
  generationDisabledReason?: string;
  onGenerateNext: () => void;
  onPrevious: () => void;
  onNext: () => void;
  onFirst: () => void;
  onLast: () => void;
}

const StorySectionView: React.FC<StorySectionViewProps> = ({
  section,
  sectionWithQuestions,
  sectionIndex,
  totalSections,
  canGenerateToday,
  isGenerating,
  generationDisabledReason,
  onGenerateNext,
  onPrevious,
  onNext,
  onFirst,
  onLast,
}) => {
  const {
    isLoading: isTTSLoading,
    isPlaying: isTTSPlaying,
    playTTS,
    stopTTS,
  } = useTTS();

  // Get user learning preferences for preferred voice
  const { data: userLearningPrefs } = useGetV1PreferencesLearning();

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
    <Stack spacing='md'>
      {/* Section Header */}
      <Paper p='md' radius='md'>
        <Group position='apart' align='center'>
          <Group spacing='xs'>
            <Badge variant='light' color='blue'>
              Section {section.section_number}
            </Badge>
            <Badge variant='outline'>
              <Group spacing={4}>
                <IconLanguage size={12} />
                {section.language_level}
              </Group>
            </Badge>
          </Group>

          <Group spacing='xs'>
            <Badge variant='outline'>
              <Group spacing={4}>
                <IconFileText size={12} />
                {section.word_count} words
              </Group>
            </Badge>
          </Group>
        </Group>

        {/* Section Navigation */}
        <Group position='center' mt='md' spacing='xs'>
          <Button
            variant='light'
            leftIcon={<IconChevronsLeft size={16} />}
            onClick={onFirst}
            disabled={sectionIndex === 0}
            size='sm'
          >
            First
          </Button>

          <Button
            variant='light'
            leftIcon={<IconChevronLeft size={16} />}
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
            rightIcon={<IconChevronRight size={16} />}
            onClick={onNext}
            disabled={sectionIndex >= totalSections - 1}
            size='sm'
          >
            Next
          </Button>

          <Button
            variant='light'
            rightIcon={<IconChevronsRight size={16} />}
            onClick={onLast}
            disabled={sectionIndex >= totalSections - 1}
            size='sm'
          >
            Last
          </Button>
        </Group>
      </Paper>

      {/* Section Content */}
      <Paper p='lg' radius='md' style={{ position: 'relative' }}>
        <Box style={{ position: 'absolute', top: 12, right: 12, zIndex: 10 }}>
          <Tooltip
            label={
              isTTSPlaying
                ? 'Stop audio'
                : isTTSLoading
                  ? 'Loading audio...'
                  : 'Listen to section'
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
                  // Determine preferred voice (user pref -> fallback -> 'echo')
                  let preferredVoice: string | undefined;
                  if (
                    userLearningPrefs?.tts_voice &&
                    userLearningPrefs.tts_voice.trim()
                  ) {
                    preferredVoice = userLearningPrefs.tts_voice.trim();
                  }
                  const finalVoice =
                    preferredVoice ??
                    defaultVoiceForLanguage(section.language_level) ??
                    'echo';
                  void playTTS(section.content || '', finalVoice);
                }
              }}
              aria-label={
                isTTSPlaying
                  ? 'Stop audio'
                  : isTTSLoading
                    ? 'Loading audio'
                    : 'Listen to section'
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

        <div
          style={{
            lineHeight: 1.6,
            fontSize: '16px',
            whiteSpace: 'pre-wrap',
          }}
        >
          {section.content}
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
              !canGenerateToday && !isGenerating
                ? generationDisabledReason || 'Unable to generate section'
                : 'Generate the next section of your story'
            }
            position='top'
            withArrow
          >
            <Button
              leftIcon={<IconPlus size={16} />}
              onClick={onGenerateNext}
              loading={isGenerating}
              disabled={!canGenerateToday || isGenerating}
              color='blue'
              variant={canGenerateToday ? 'filled' : 'light'}
            >
              {isGenerating ? 'Generating...' : 'Generate Next Section'}
            </Button>
          </Tooltip>
        </div>
      </Paper>
    </Stack>
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
      <Text size='sm' weight={500} mb='xs'>
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
            <label htmlFor={`option-${index}`} style={{ marginLeft: 8 }}>
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
          <Text size='xs'>{question.explanation}</Text>
        </Alert>
      )}
    </Paper>
  );
};

export default StorySectionView;
