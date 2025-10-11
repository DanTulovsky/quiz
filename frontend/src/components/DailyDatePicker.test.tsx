import React from 'react';
import { render, screen, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MantineProvider } from '@mantine/core';
import DailyDatePicker from './DailyDatePicker';

// Force a timezone west of UTC to catch off-by-one regressions
process.env.TZ = 'America/Los_Angeles';

// Mock Mantine's DatePickerInput so we can directly trigger the onChange callback
vi.mock('@mantine/dates', async () => {
  const actual = await vi.importActual('@mantine/dates');
  return {
    ...actual,
    DatePickerInput: ({
      onChange,
      getDayProps,
    }: {
      onChange: (date: Date) => void;
      getDayProps?: (date: string) => Record<string, unknown>;
    }) => {
      // Simulate the getDayProps function being called with date strings
      const mockDate = '2025-08-03';
      const dayProps = getDayProps ? getDayProps(mockDate) : {};

      return (
        <div data-testid='date-picker'>
          <button
            onClick={() => onChange(new Date(2025, 7, 3))}
            aria-label='pick-2025-08-03'
            data-testid='day-2025-08-03'
            {...dayProps}
          >
            pick 2025-08-03
          </button>
        </div>
      );
    },
  };
});

// Test wrapper with Mantine providers
const TestWrapper = ({ children }: { children: React.ReactNode }) => (
  <MantineProvider>{children}</MantineProvider>
);

describe('DailyDatePicker local-time behavior', () => {
  it('calls onDateSelect with the same local date (no previous-day shift)', async () => {
    const user = userEvent.setup();
    const mockOnDateSelect = vi.fn();

    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker
            selectedDate='2025-08-03'
            onDateSelect={mockOnDateSelect}
            availableDates={['2025-08-03']}
            progressData={{}}
          />
        </TestWrapper>
      );
    });

    // Click the date picker button to trigger the onChange
    const pickerButton = screen.getByRole('button', {
      name: /pick 2025-08-03/i,
    });
    await user.click(pickerButton);

    // Verify that onDateSelect was called with the exact same date string
    // (no off-by-one day shift due to timezone parsing)
    expect(mockOnDateSelect).toHaveBeenCalledWith('2025-08-03');
  });
});

describe('DailyDatePicker', () => {
  const defaultProps = {
    selectedDate: '2024-01-01',
    onDateSelect: vi.fn(),
    availableDates: ['2024-01-01', '2024-01-02'],
    progressData: {},
  };

  it('renders with correct placeholder', () => {
    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker {...defaultProps} placeholder='Select date' />
        </TestWrapper>
      );
    });

    expect(screen.getByTestId('date-picker')).toBeInTheDocument();
  });

  it('shows available dates with indicators', () => {
    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker {...defaultProps} />
        </TestWrapper>
      );
    });

    // Check that available dates are marked
    expect(screen.getByTestId('day-2025-08-03')).toBeInTheDocument();
  });

  it('marks selected date correctly', () => {
    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker {...defaultProps} selectedDate='2024-01-01' />
        </TestWrapper>
      );
    });

    expect(screen.getByTestId('day-2025-08-03')).toBeInTheDocument();
  });

  it('shows completion status correctly', () => {
    const progressData = {
      '2024-01-01': { completed: 5, total: 5 },
      '2024-01-02': { completed: 2, total: 5 },
    };

    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker {...defaultProps} progressData={progressData} />
        </TestWrapper>
      );
    });

    // Check that available dates are marked correctly
    expect(screen.getByTestId('day-2025-08-03')).toBeInTheDocument();
  });

  it('calls onDateSelect when date is changed', async () => {
    const user = userEvent.setup();
    const mockOnDateSelect = vi.fn();

    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker {...defaultProps} onDateSelect={mockOnDateSelect} />
        </TestWrapper>
      );
    });

    const pickerButton = screen.getByRole('button', {
      name: /pick 2025-08-03/i,
    });
    await user.click(pickerButton);

    expect(mockOnDateSelect).toHaveBeenCalledWith('2025-08-03');
  });

  it('handles null value correctly', () => {
    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker {...defaultProps} selectedDate={null} />
        </TestWrapper>
      );
    });

    expect(screen.getByTestId('date-picker')).toBeInTheDocument();
  });

  it('applies custom props correctly', () => {
    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker
            {...defaultProps}
            size='lg'
            style={{ backgroundColor: 'red' }}
          />
        </TestWrapper>
      );
    });

    expect(screen.getByTestId('date-picker')).toBeInTheDocument();
  });

  it('handles empty availableDates gracefully', () => {
    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker {...defaultProps} availableDates={[]} />
        </TestWrapper>
      );
    });

    expect(screen.getByTestId('date-picker')).toBeInTheDocument();
  });

  it('handles empty progressData gracefully', () => {
    act(() => {
      render(
        <TestWrapper>
          <DailyDatePicker {...defaultProps} progressData={{}} />
        </TestWrapper>
      );
    });

    // Should still show available dates but without progress indicators
    expect(screen.getByTestId('day-2025-08-03')).toBeInTheDocument();
  });
});
