// React import not required directly in this test
import { renderWithProviders } from '../test-utils';
import { Question, AnswerResponse } from '../api/api';
import { screen } from '@testing-library/react';
import QuestionCard from './QuestionCard';
import { vi } from 'vitest';

// Mock useAuth so QuestionCard can render in isolation
vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({ isAuthenticated: true }),
}));

const makeQuestion = (id: number, options: string[]) => ({
  id,
  type: 'quiz',
  content: { options },
  correct_answer: 0,
  total_responses: 0,
  correct_count: 0,
  incorrect_count: 0,
  level: 1,
});

describe('QuestionCard selection guard', () => {
  it('does not show parent selectedAnswer when question id does not match', () => {
    const q = makeQuestion(123, ['A', 'B', 'C']);
    renderWithProviders(
      <QuestionCard
        question={q as unknown as Question}
        onAnswer={async () => ({ is_correct: false }) as AnswerResponse}
        onNext={() => {}}
        feedback={null}
        selectedAnswer={1}
        selectedAnswerQuestionId={999}
        onAnswerSelect={() => {}}
        showExplanation={false}
        setShowExplanation={() => {}}
      />
    );

    // No "Your answer" badge should be present because parent-selected id doesn't match
    expect(screen.queryByText('Your answer')).not.toBeInTheDocument();
  });

  it('shows parent selectedAnswer when question id matches', () => {
    const q = makeQuestion(200, ['X', 'Y', 'Z']);
    renderWithProviders(
      <QuestionCard
        question={q as unknown as Question}
        onAnswer={async () => ({ is_correct: false }) as AnswerResponse}
        onNext={() => {}}
        feedback={null}
        selectedAnswer={2}
        selectedAnswerQuestionId={200}
        onAnswerSelect={() => {}}
        showExplanation={false}
        setShowExplanation={() => {}}
      />
    );

    // When parent selection is trusted, the corresponding option should display as selected
    // The badge text appears only after submission, so rely on the radio input value instead
    const radios = screen.getAllByRole('radio');
    // selectedAnswer 2 -> should have the 3rd radio checked (index 2)
    expect(radios[2] as HTMLInputElement).toBeChecked();
  });
});
