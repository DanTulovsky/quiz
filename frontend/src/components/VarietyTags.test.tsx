import { screen } from '@testing-library/react';
import VarietyTags from './VarietyTags';
import { Question } from '../api/api';
import { renderWithProviders } from '../test-utils';

describe('VarietyTags', () => {
  const mockQuestionWithAllVarietyElements: Question = {
    id: 1,
    type: 'vocabulary',
    content: {
      question: 'What is the Italian word for "hello"?',
      options: ['Ciao', 'Buongiorno', 'Arrivederci', 'Grazie'],
    },
    level: 'A1',
    created_at: '2023-01-01T00:00:00Z',
    // All variety elements populated
    topic_category: 'daily_life',
    grammar_focus: 'present_simple',
    vocabulary_domain: 'greetings',
    scenario: 'meeting_people',
    style_modifier: 'conversational',
    difficulty_modifier: 'basic',
    time_context: 'morning_routine',
  };

  const mockQuestionWithPartialVarietyElements: Question = {
    id: 2,
    type: 'fill_blank',
    content: {
      question: 'Complete the sentence: Io _____ italiano.',
      options: ['parlo', 'parli', 'parla', 'parliamo'],
    },
    level: 'A2',
    created_at: '2023-01-01T00:00:00Z',
    // Only some variety elements populated
    topic_category: 'education',
    grammar_focus: 'present_tense',
    vocabulary_domain: 'languages',
    // Others are undefined/empty
  };

  const mockQuestionWithNoVarietyElements: Question = {
    id: 3,
    type: 'reading_comprehension',
    content: {
      question: 'What is the main topic?',
      passage: 'Some passage text here.',
      options: ['Option A', 'Option B', 'Option C', 'Option D'],
    },
    level: 'B1',
    created_at: '2023-01-01T00:00:00Z',
    // No variety elements
  };

  it('renders all variety elements when all are present', () => {
    renderWithProviders(
      <VarietyTags question={mockQuestionWithAllVarietyElements} />
    );

    // Check that all variety values are rendered (no label, just value)
    expect(screen.getByText('Daily Life')).toBeInTheDocument();
    expect(screen.getByText('Present Simple')).toBeInTheDocument();
    expect(screen.getByText('Greetings')).toBeInTheDocument();
    expect(screen.getByText('Meeting People')).toBeInTheDocument();
    expect(screen.getByText('Conversational')).toBeInTheDocument();
    expect(screen.getByText('Basic')).toBeInTheDocument();
    expect(screen.getByText('Morning Routine')).toBeInTheDocument();
  });

  it('renders only populated variety elements when some are missing', () => {
    renderWithProviders(
      <VarietyTags question={mockQuestionWithPartialVarietyElements} />
    );

    // Check that only populated values are rendered (no label, just value)
    expect(screen.getByText('Education')).toBeInTheDocument();
    expect(screen.getByText('Present Tense')).toBeInTheDocument();
    expect(screen.getByText('Languages')).toBeInTheDocument();

    // Check that empty/undefined elements are not rendered
    expect(screen.queryByText(/Scenario/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Style/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Difficulty/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Context/)).not.toBeInTheDocument();
  });

  it('renders nothing when no variety elements are present', () => {
    renderWithProviders(
      <VarietyTags question={mockQuestionWithNoVarietyElements} />
    );

    // The component should not render the variety-tags test-id when no elements are present
    expect(screen.queryByTestId('variety-tags')).not.toBeInTheDocument();
  });

  it('renders with different sizes', () => {
    renderWithProviders(
      <VarietyTags question={mockQuestionWithAllVarietyElements} size='lg' />
    );

    // Check that tags are rendered (we can't easily test the size prop directly)
    expect(screen.getByText('Daily Life')).toBeInTheDocument();
    expect(screen.getByText('Present Simple')).toBeInTheDocument();
  });

  it('renders in compact mode', () => {
    renderWithProviders(
      <VarietyTags
        question={mockQuestionWithAllVarietyElements}
        compact={true}
      />
    );

    // Check that tags are rendered in compact mode (formatted but without labels)
    expect(screen.getByText('Daily Life')).toBeInTheDocument();
    expect(screen.getByText('Present Simple')).toBeInTheDocument();
    expect(screen.getByText('Greetings')).toBeInTheDocument();
    expect(screen.getByText('Meeting People')).toBeInTheDocument();
    expect(screen.getByText('Conversational')).toBeInTheDocument();
    expect(screen.getByText('Basic')).toBeInTheDocument();
    expect(screen.getByText('Morning Routine')).toBeInTheDocument();
  });

  it('formats variety element names correctly', () => {
    const questionWithUnderscores: Question = {
      id: 4,
      type: 'vocabulary',
      content: {
        question: 'Test question',
        options: ['A', 'B', 'C', 'D'],
      },
      level: 'B2',
      created_at: '2023-01-01T00:00:00Z',
      topic_category: 'business_and_work',
      grammar_focus: 'past_perfect_continuous',
      vocabulary_domain: 'office_equipment',
      scenario: 'job_interview',
      style_modifier: 'formal_business',
      difficulty_modifier: 'intermediate_advanced',
      time_context: 'work_day_morning',
    };

    renderWithProviders(<VarietyTags question={questionWithUnderscores} />);

    // Check that underscores are replaced with spaces and properly formatted (value only)
    expect(screen.getByText('Business And Work')).toBeInTheDocument();
    expect(screen.getByText('Past Perfect Continuous')).toBeInTheDocument();
    expect(screen.getByText('Office Equipment')).toBeInTheDocument();
    expect(screen.getByText('Job Interview')).toBeInTheDocument();
    expect(screen.getByText('Formal Business')).toBeInTheDocument();
    expect(screen.getByText('Intermediate Advanced')).toBeInTheDocument();
    expect(screen.getByText('Work Day Morning')).toBeInTheDocument();
  });

  it('handles variety elements with single words', () => {
    const questionWithSingleWords: Question = {
      id: 5,
      type: 'vocabulary',
      content: {
        question: 'Test question',
        options: ['A', 'B', 'C', 'D'],
      },
      level: 'A1',
      created_at: '2023-01-01T00:00:00Z',
      topic_category: 'food',
      grammar_focus: 'nouns',
      vocabulary_domain: 'cooking',
      scenario: 'restaurant',
      style_modifier: 'casual',
      difficulty_modifier: 'easy',
      time_context: 'lunch',
    };

    renderWithProviders(<VarietyTags question={questionWithSingleWords} />);

    // Check that single words are rendered correctly (value only)
    expect(screen.getByText('Food')).toBeInTheDocument();
    expect(screen.getByText('Nouns')).toBeInTheDocument();
    expect(screen.getByText('Cooking')).toBeInTheDocument();
    expect(screen.getByText('Restaurant')).toBeInTheDocument();
    expect(screen.getByText('Casual')).toBeInTheDocument();
    expect(screen.getByText('Easy')).toBeInTheDocument();
    expect(screen.getByText('Lunch')).toBeInTheDocument();
  });

  it('handles empty strings correctly', () => {
    const questionWithEmptyStrings: Question = {
      id: 6,
      type: 'vocabulary',
      content: {
        question: 'Test question',
        options: ['A', 'B', 'C', 'D'],
      },
      level: 'A1',
      created_at: '2023-01-01T00:00:00Z',
      topic_category: 'travel',
      grammar_focus: '',
      vocabulary_domain: 'transportation',
      scenario: '',
      style_modifier: 'informal',
      difficulty_modifier: '',
      time_context: 'evening',
    };

    renderWithProviders(<VarietyTags question={questionWithEmptyStrings} />);

    // Check that only non-empty values are rendered (value only)
    expect(screen.getByText('Travel')).toBeInTheDocument();
    expect(screen.getByText('Transportation')).toBeInTheDocument();
    expect(screen.getByText('Informal')).toBeInTheDocument();
    expect(screen.getByText('Evening')).toBeInTheDocument();

    // Check that empty strings are not rendered
    expect(screen.queryByText(/Grammar/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Scenario/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Difficulty/)).not.toBeInTheDocument();
  });

  it('uses correct badge colors for different variety types', () => {
    renderWithProviders(
      <VarietyTags question={mockQuestionWithAllVarietyElements} />
    );

    // We can't easily test the specific colors, but we can verify the tags are rendered (value only)
    expect(screen.getByText('Daily Life')).toBeInTheDocument();
    expect(screen.getByText('Present Simple')).toBeInTheDocument();
    expect(screen.getByText('Greetings')).toBeInTheDocument();
    expect(screen.getByText('Meeting People')).toBeInTheDocument();
    expect(screen.getByText('Conversational')).toBeInTheDocument();
    expect(screen.getByText('Basic')).toBeInTheDocument();
    expect(screen.getByText('Morning Routine')).toBeInTheDocument();
  });

  it('renders with correct test ids for accessibility', () => {
    renderWithProviders(
      <VarietyTags question={mockQuestionWithAllVarietyElements} />
    );

    // Check that the main container has a test id
    expect(screen.getByTestId('variety-tags')).toBeInTheDocument();
  });
});
