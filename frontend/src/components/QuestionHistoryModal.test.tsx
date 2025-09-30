import { screen } from '@testing-library/react';
import { renderWithProviders } from '../test-utils';
import { QuestionHistoryModal } from './QuestionHistoryModal';
import type { DailyQuestionHistory as ApiDailyQuestionHistory } from '../api/api';

describe('QuestionHistoryModal', () => {
  it('displays correct/incorrect/not attempted based on is_correct and is_completed', () => {
    const mockHistory: ApiDailyQuestionHistory[] = [
      {
        assignment_date: '2025-08-11T00:00:00Z',
        is_completed: true,
        is_correct: true,
        submitted_at: '2025-08-11T12:00:00Z',
      },
      {
        assignment_date: '2025-08-14T00:00:00Z',
        is_completed: true,
        is_correct: false,
        submitted_at: '2025-08-14T12:00:00Z',
      },
      {
        assignment_date: '2025-08-15T00:00:00Z',
        is_completed: false,
        is_correct: null,
        submitted_at: null,
      },
    ];

    renderWithProviders(
      <QuestionHistoryModal
        opened={true}
        onClose={() => {}}
        history={mockHistory}
        isLoading={false}
        questionText={'Test question'}
      />
    );

    // Expect labels to be present (use flexible matchers in case date text is split)
    expect(
      screen.getByText(
        content => typeof content === 'string' && content.includes('Aug 11')
      )
    ).toBeInTheDocument();
    expect(screen.getByText('Correct')).toBeInTheDocument();

    expect(
      screen.getByText(
        content => typeof content === 'string' && content.includes('Aug 14')
      )
    ).toBeInTheDocument();
    expect(screen.getByText('Incorrect')).toBeInTheDocument();

    expect(
      screen.getByText(
        content => typeof content === 'string' && content.includes('Aug 15')
      )
    ).toBeInTheDocument();
    expect(screen.getAllByText('Not attempted').length).toBeGreaterThan(0);

    // Ensure the most recent date appears first (Aug 15 should be above Aug 14 and Aug 11)
    const allDateNodes = screen.getAllByText(
      content =>
        typeof content === 'string' &&
        /(Aug\s+11|Aug\s+14|Aug\s+15)/.test(content)
    );
    const renderedDates = allDateNodes.map(n => n.textContent || '');
    // The first rendered date should include Aug 15
    expect(renderedDates[0]).toMatch(/Aug\s+15/);
  });
});
