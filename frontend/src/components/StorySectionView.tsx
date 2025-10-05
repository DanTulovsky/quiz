import React from 'react';
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
} from '@mantine/core';
import {
  IconChevronLeft,
  IconChevronRight,
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
}) => {
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
        <Group position='center' mt='md'>
          <Button
            variant='light'
            leftIcon={<IconChevronLeft size={16} />}
            onClick={onPrevious}
            disabled={sectionIndex === 0}
            size='sm'
          >
            Previous
          </Button>

          <Text size='sm' color='dimmed'>
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
        </Group>
      </Paper>

      {/* Section Content */}
      <Paper p='lg' radius='md'>
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
        <Group position='apart' align='center'>
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
            <span>
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
            </span>
          </Tooltip>
        </Group>
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
