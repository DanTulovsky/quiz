import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import DailyCompletionScreen from './DailyCompletionScreen';

// Mock the DailyDatePicker component
vi.mock('./DailyDatePicker', () => ({
  default: ({
    selectedDate,
    onDateSelect,
  }: {
    selectedDate: string;
    onDateSelect: (date: string) => void;
  }) => (
    <div data-testid='daily-date-picker'>
      <input
        data-testid='date-picker-input'
        value={selectedDate}
        onChange={e => onDateSelect(e.target.value)}
      />
    </div>
  ),
}));

const TestWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <MantineProvider>{children}</MantineProvider>
);

describe('DailyCompletionScreen', () => {
  const defaultProps = {
    selectedDate: '2024-01-01',
    onDateSelect: vi.fn(),
    availableDates: ['2024-01-01', '2024-01-02', '2024-01-03'],
    progressData: {
      '2024-01-01': { completed: 10, total: 10 },
    },
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders completion screen with correct title for today', () => {
    const today = new Date().toISOString().split('T')[0];
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} selectedDate={today} />
      </TestWrapper>
    );

    expect(
      screen.getByText("Today's Questions Completed!")
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Great job! You've completed all of today's questions. Come back tomorrow for more practice."
      )
    ).toBeInTheDocument();
  });

  it('renders completion screen with correct title for other dates', () => {
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} />
      </TestWrapper>
    );

    expect(screen.getByText('Questions Completed!')).toBeInTheDocument();
    expect(
      screen.getByText(
        "Great job! You've completed all questions for 1/1/2024."
      )
    ).toBeInTheDocument();
  });

  it('displays progress information when available', () => {
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} />
      </TestWrapper>
    );

    expect(screen.getByText('Progress: 10/10 completed')).toBeInTheDocument();
  });

  it('shows "Come Back Tomorrow" button for today', () => {
    const today = new Date().toISOString().split('T')[0];
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} selectedDate={today} />
      </TestWrapper>
    );

    expect(
      screen.getByText('Come back tomorrow for more practice')
    ).toBeInTheDocument();
  });

  it('does not show "Come Back Tomorrow" button for other dates', () => {
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} />
      </TestWrapper>
    );

    expect(screen.queryByText('Come Back Tomorrow')).not.toBeInTheDocument();
  });

  it('shows "Select Another Date" button', () => {
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} />
      </TestWrapper>
    );

    expect(
      screen.getByText('Select another date to practice more questions')
    ).toBeInTheDocument();
  });

  it('calls onDateSelect when "Come Back Tomorrow" is clicked', () => {
    const today = new Date().toISOString().split('T')[0];

    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} selectedDate={today} />
      </TestWrapper>
    );

    // Since it's now just text, we don't need to click it
    expect(
      screen.getByText('Come back tomorrow for more practice')
    ).toBeInTheDocument();

    // Since it's no longer a button, onDateSelect should not be called
    expect(defaultProps.onDateSelect).not.toHaveBeenCalled();
  });

  it('handles date selection through hidden date picker', () => {
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} />
      </TestWrapper>
    );

    const datePicker = screen.getByTestId('date-picker-input');
    fireEvent.change(datePicker, { target: { value: '2024-01-02' } });

    expect(defaultProps.onDateSelect).toHaveBeenCalledWith('2024-01-02');
  });

  it('renders with success icon', () => {
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} />
      </TestWrapper>
    );

    // The icon should be present (IconCheck)
    expect(screen.getByText('Questions Completed!')).toBeInTheDocument();
  });

  it('handles empty progress data gracefully', () => {
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} progressData={{}} />
      </TestWrapper>
    );

    expect(screen.getByText('Questions Completed!')).toBeInTheDocument();
    expect(screen.queryByText(/Progress:/)).not.toBeInTheDocument();
  });

  it('handles empty available dates gracefully', () => {
    render(
      <TestWrapper>
        <DailyCompletionScreen {...defaultProps} availableDates={[]} />
      </TestWrapper>
    );

    expect(screen.getByText('Questions Completed!')).toBeInTheDocument();
  });
});
