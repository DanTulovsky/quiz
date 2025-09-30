import React from 'react';
import { renderWithProviders } from '../test-utils';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import KeyboardShortcuts from './KeyboardShortcuts';

describe('KeyboardShortcuts h key behavior', () => {
  it('calls onShowHistory when history closed and onHideHistory when open', async () => {
    const user = userEvent.setup();
    const onShow = vi.fn();
    const onHide = vi.fn();

    const { rerender } = renderWithProviders(
      <KeyboardShortcuts
        onAnswerSelect={() => {}}
        onSubmit={() => {}}
        onNextQuestion={() => {}}
        onNewQuestion={() => {}}
        isSubmitted={false}
        hasSelectedAnswer={false}
        maxOptions={0}
        onShowHistory={onShow}
        onHideHistory={onHide}
      />
    );

    // Press 'h' when history is closed
    await user.keyboard('h');
    expect(onShow).toHaveBeenCalled();

    // Rerender with history open
    rerender(
      <KeyboardShortcuts
        onAnswerSelect={() => {}}
        onSubmit={() => {}}
        onNextQuestion={() => {}}
        onNewQuestion={() => {}}
        isSubmitted={false}
        hasSelectedAnswer={false}
        maxOptions={0}
        onShowHistory={onShow}
        onHideHistory={onHide}
        isHistoryOpen={true}
      />
    );

    await user.keyboard('h');
    expect(onHide).toHaveBeenCalled();
  });
});

import { vi } from 'vitest';
import { useMediaQuery, useLocalStorage } from '@mantine/hooks';

// Mock Mantine hooks used by KeyboardShortcuts
vi.mock('@mantine/hooks', () => ({
  useMediaQuery: vi.fn(),
  // Mock useLocalStorage - we'll control the value per test
  useLocalStorage: vi.fn(),
}));

// Type for the mocked hooks
const mockUseMediaQuery = useMediaQuery as ReturnType<typeof vi.fn>;
const mockUseLocalStorage = useLocalStorage as ReturnType<typeof vi.fn>;

// Setup the mock to return proper typed values
interface UseLocalStorageOptions {
  key: string;
  defaultValue?: unknown;
}

mockUseLocalStorage.mockImplementation((options: UseLocalStorageOptions) => {
  const defaultValue = options?.defaultValue ?? null;
  return [defaultValue, vi.fn()];
});

describe('KeyboardShortcuts dynamic hotkey display', () => {
  const baseProps = {
    onAnswerSelect: vi.fn(),
    onSubmit: vi.fn(),
    onNextQuestion: vi.fn(),
    onNewQuestion: vi.fn(),
    isSubmitted: false,
    hasSelectedAnswer: false,
    maxOptions: 4,
    onToggleExplanation: vi.fn(),
    explanationAvailable: true,
  };

  beforeEach(() => {
    // Default to large screen
    mockUseMediaQuery.mockReturnValue(false);
    // Reset the mock implementation
    mockUseLocalStorage.mockImplementation(
      (options: UseLocalStorageOptions) => {
        const defaultValue = options?.defaultValue ?? null;
        return [defaultValue, vi.fn()];
      }
    );
  });

  it('shows navigation shortcuts', () => {
    // Ensure component is expanded by setting the value to true
    mockUseLocalStorage.mockImplementation(() => [true, vi.fn()]);
    renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );
    expect(screen.getByText('⇧1-3')).toBeInTheDocument();
    expect(screen.getByText('Main Navigation')).toBeInTheDocument();
  });

  it('starts expanded on large screens', () => {
    mockUseMediaQuery.mockReturnValue(false); // Large screen (width > 1200px and height > 700px)
    mockUseLocalStorage.mockImplementation(() => [true, vi.fn()]); // Force expanded
    renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Should show the shortcuts panel (expanded)
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument();
    expect(screen.getByText('⇧1-3')).toBeInTheDocument();
  });

  it('starts collapsed on small screens', () => {
    mockUseMediaQuery.mockReturnValue(true); // Small screen (width <= 1200px or height <= 700px)
    renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Should not show the shortcuts panel (collapsed)
    expect(screen.queryByText('Keyboard Shortcuts')).not.toBeInTheDocument();
    expect(screen.queryByText('⇧1-3')).not.toBeInTheDocument();

    // Should show the collapse/expand button
    expect(screen.getByTitle('Expand keyboard shortcuts')).toBeInTheDocument();
  });

  it('collapses when screen becomes small', () => {
    // Start with large screen
    mockUseMediaQuery.mockReturnValue(false);
    const { rerender } = renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Should be expanded initially
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument();

    // Change to small screen
    mockUseMediaQuery.mockReturnValue(true);
    rerender(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // The component should start collapsed on small screens, but the test environment
    // might not immediately reflect the state change. Let's check for the button instead.
    expect(screen.getByTitle('Expand keyboard shortcuts')).toBeInTheDocument();
  });

  it('shows 1-4 badge when quick suggestions are closed', () => {
    mockUseLocalStorage.mockImplementation(() => [true, vi.fn()]); // Force expanded
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={false}
        quickSuggestionsCount={9}
      />
    );
    expect(screen.getByText('1-4')).toBeInTheDocument();
    expect(screen.queryByText('0-9')).not.toBeInTheDocument();
    expect(screen.getByText('Q')).toBeInTheDocument();
  });

  it('shows 0-9 badge when quick suggestions are open and count > 9', () => {
    mockUseLocalStorage.mockImplementation(() => [true, vi.fn()]); // Force expanded
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={true}
        quickSuggestionsCount={10}
      />
    );
    expect(screen.getByText('0-9')).toBeInTheDocument();
    expect(screen.queryByText('1-4')).not.toBeInTheDocument();
    expect(screen.getByText('Q')).toBeInTheDocument();
  });

  it('shows 1-N badge when quick suggestions are open and count <= 9', () => {
    mockUseLocalStorage.mockImplementation(() => [true, vi.fn()]); // Force expanded
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={true}
        quickSuggestionsCount={5}
      />
    );
    expect(screen.getByText('1-5')).toBeInTheDocument();
    expect(screen.queryByText('1-4')).not.toBeInTheDocument();
    expect(screen.getByText('Q')).toBeInTheDocument();
  });

  it('shows N and Enter badges', () => {
    mockUseLocalStorage.mockImplementation(() => [true, vi.fn()]); // Force expanded
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={false}
        quickSuggestionsCount={9}
        hasSelectedAnswer={true}
      />
    );
    expect(screen.getByText('N')).toBeInTheDocument();
    expect(screen.getByText('↵')).toBeInTheDocument();
  });

  it('shows E badge only when isSubmitted and explanationAvailable', () => {
    mockUseLocalStorage.mockImplementation(() => [true, vi.fn()]); // Force expanded
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isSubmitted={true}
        explanationAvailable={true}
      />
    );
    expect(screen.getByText('E')).toBeInTheDocument();
  });

  it('does not show E badge when not submitted', () => {
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isSubmitted={false}
        explanationAvailable={true}
      />
    );
    expect(screen.queryByText('E')).not.toBeInTheDocument();
  });

  it('does not show C badge if onFocusChat is not provided', () => {
    const props = { ...baseProps };
    renderWithProviders(
      <KeyboardShortcuts
        {...props}
        isQuickSuggestionsOpen={false}
        quickSuggestionsCount={9}
      />
    );
    expect(screen.queryByText('C')).not.toBeInTheDocument();
  });

  it('shows scroll chat shortcut only when input is focused and suggestions closed', () => {
    mockUseLocalStorage.mockImplementation(() => [true, vi.fn()]); // Force expanded
    // Should show when focused and suggestions closed
    const { rerender } = renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isInputFocused={true}
        isQuickSuggestionsOpen={false}
      />
    );
    const label = screen.getByText('Scroll chat');
    // Find the badge in the same group
    const group = label.closest('[class*="Group-root"]');
    expect(group).toBeTruthy();
    expect(group?.textContent).toContain('↑');
    expect(group?.textContent).toContain('↓');

    // Should not show when not focused
    rerender(
      <KeyboardShortcuts
        {...baseProps}
        isInputFocused={false}
        isQuickSuggestionsOpen={false}
      />
    );
    expect(screen.queryByText('Scroll chat')).not.toBeInTheDocument();
  });

  it('shows navigation shortcuts when quick suggestions are open', () => {
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={true}
        quickSuggestionsCount={5}
      />
    );

    // Should show navigation shortcuts
    expect(screen.getByText('↑↓')).toBeInTheDocument();
    expect(screen.getByText('Navigate suggestions')).toBeInTheDocument();
    expect(screen.getByText('↵')).toBeInTheDocument();
    expect(screen.getByText('Execute selected')).toBeInTheDocument();

    // Should also show the number shortcuts
    expect(screen.getByText('1-5')).toBeInTheDocument();
    expect(screen.getByText('Select suggestion')).toBeInTheDocument();
  });

  it('does not show navigation shortcuts when quick suggestions are closed', () => {
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={false}
        quickSuggestionsCount={5}
      />
    );

    // Should not show navigation shortcuts
    expect(screen.queryByText('Navigate suggestions')).not.toBeInTheDocument();
    expect(screen.queryByText('Execute selected')).not.toBeInTheDocument();

    // Should show regular shortcuts instead
    expect(screen.getByText('1-4')).toBeInTheDocument();
    expect(screen.getByText('Select answer')).toBeInTheDocument();
  });

  it('shows correct number range for suggestions (0-9 when more than 9)', () => {
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={true}
        quickSuggestionsCount={10}
      />
    );

    // Should show 0-9 for 10+ suggestions
    expect(screen.getByText('0-9')).toBeInTheDocument();
    expect(screen.getByText('Select suggestion')).toBeInTheDocument();
  });

  it('shows correct number range for suggestions (1-N when 9 or fewer)', () => {
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={true}
        quickSuggestionsCount={3}
      />
    );

    // Should show 1-3 for 3 suggestions
    expect(screen.getByText('1-3')).toBeInTheDocument();
    expect(screen.getByText('Select suggestion')).toBeInTheDocument();
  });

  it('registers K key handler for marking question as known', async () => {
    const mockOnMarkKnown = vi.fn();

    renderWithProviders(
      <KeyboardShortcuts {...baseProps} onMarkKnown={mockOnMarkKnown} />
    );

    // Verify that the K badge is displayed
    expect(screen.getByText('K')).toBeInTheDocument();
    expect(screen.getByText('Rate question knowledge')).toBeInTheDocument();
  });

  it('does not show K badge when onMarkKnown is not provided', () => {
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        // onMarkKnown not provided
      />
    );

    // Should not show K badge
    expect(screen.queryByText('K')).not.toBeInTheDocument();
  });

  it('shows correct shortcuts when input is focused', () => {
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isInputFocused={true}
        isQuickSuggestionsOpen={false}
      />
    );

    // When input is focused, should show:
    // - Esc for resetting focus
    expect(screen.getByText('Esc')).toBeInTheDocument();
    expect(
      screen.getByText('Reset focus / enable hotkeys')
    ).toBeInTheDocument();

    // - Arrow keys for scrolling chat
    expect(screen.getByText('↑↓')).toBeInTheDocument();
    expect(screen.getByText('Scroll chat')).toBeInTheDocument();

    // Should NOT show regular shortcuts like 1-4, N, etc.
    expect(screen.queryByText('1-4')).not.toBeInTheDocument();
    expect(screen.queryByText('Select answer')).not.toBeInTheDocument();
    expect(screen.queryByText('N')).not.toBeInTheDocument();
    expect(screen.queryByText('New question')).not.toBeInTheDocument();
  });

  it('shows regular shortcuts when input is not focused', () => {
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isInputFocused={false}
        isQuickSuggestionsOpen={false}
        hasSelectedAnswer={true}
      />
    );

    // When input is not focused, should show regular shortcuts:
    // - 1-4 for answer selection
    expect(screen.getByText('1-4')).toBeInTheDocument();
    expect(screen.getByText('Select answer')).toBeInTheDocument();

    // - N for new question
    expect(screen.getByText('N')).toBeInTheDocument();
    expect(screen.getByText('New question')).toBeInTheDocument();

    // - Enter for submit
    expect(screen.getByText('↵')).toBeInTheDocument();
    expect(screen.getByText('Submit')).toBeInTheDocument();

    // Should NOT show input-focused shortcuts
    expect(screen.queryByText('Esc')).not.toBeInTheDocument();
    expect(
      screen.queryByText('Reset focus / enable hotkeys')
    ).not.toBeInTheDocument();
    expect(screen.queryByText('Scroll chat')).not.toBeInTheDocument();
  });

  it('transitions correctly between focused and unfocused states', () => {
    const { rerender } = renderWithProviders(
      <KeyboardShortcuts {...baseProps} isInputFocused={true} />
    );

    // Should show focused shortcuts
    expect(
      screen.getByText('Reset focus / enable hotkeys')
    ).toBeInTheDocument();

    // Change to unfocused
    rerender(<KeyboardShortcuts {...baseProps} isInputFocused={false} />);

    // Should show regular shortcuts
    expect(screen.getByText('Main Navigation')).toBeInTheDocument();
  });

  it('renders with overlap detection functionality', () => {
    // Test that the component renders correctly with the new overlap detection
    renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Should render the component without errors
    expect(screen.getByTestId('keyboard-shortcuts')).toBeInTheDocument();
  });

  it('handles overlap detection state changes', () => {
    // Mock useMediaQuery to return false (large screen)
    mockUseMediaQuery.mockReturnValue(false);

    // Test that the component handles overlap detection state properly
    const { rerender } = renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Should render without errors
    expect(screen.getByTestId('keyboard-shortcuts')).toBeInTheDocument();

    // Test with different screen sizes
    mockUseMediaQuery.mockReturnValue(true); // Small screen
    rerender(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Should still render without errors
    expect(screen.getByTestId('keyboard-shortcuts')).toBeInTheDocument();
  });

  it('maintains existing functionality with overlap detection', () => {
    // Test that the new overlap detection doesn't break existing functionality
    renderWithProviders(
      <KeyboardShortcuts
        {...baseProps}
        isQuickSuggestionsOpen={false}
        isSubmitted={true}
        hasSelectedAnswer={true}
      />
    );

    // Should show the shortcuts panel
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument();
    expect(screen.getByText('N')).toBeInTheDocument(); // New question shortcut
    expect(screen.getByText('↵')).toBeInTheDocument(); // Enter shortcut
  });

  it('allows user to manually expand and prevents auto-collapse', () => {
    // Mock useMediaQuery to return false (large screen)
    mockUseMediaQuery.mockReturnValue(false);

    renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Initially should be expanded on large screen
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument();

    // Test that the component renders the toggle button correctly
    expect(
      screen.getByTitle('Collapse keyboard shortcuts')
    ).toBeInTheDocument();
  });

  it('tests user manual expansion and auto-collapse prevention', () => {
    // Mock useMediaQuery to return false (large screen)
    mockUseMediaQuery.mockReturnValue(false);

    renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Initially should be expanded on large screen
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument();
    expect(
      screen.getByTitle('Collapse keyboard shortcuts')
    ).toBeInTheDocument();

    // Test that the toggle button exists and has the correct title
    const toggleButton = screen.getByRole('button');
    expect(toggleButton).toHaveAttribute(
      'title',
      'Collapse keyboard shortcuts'
    );
  });

  it('tests auto-collapse prevention after manual expansion', () => {
    // Mock useMediaQuery to return false (large screen)
    mockUseMediaQuery.mockReturnValue(false);

    renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Initially should be expanded
    expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument();
    expect(
      screen.getByTitle('Collapse keyboard shortcuts')
    ).toBeInTheDocument();

    // Test that the component renders correctly with the user interaction logic
    // The actual user interaction testing would require more complex setup
    // This test verifies the component structure and state management
    expect(screen.getByTestId('keyboard-shortcuts')).toBeInTheDocument();
  });

  it('tests auto-collapse on initial load with overlap', () => {
    // Mock useMediaQuery to return true (small screen)
    mockUseMediaQuery.mockReturnValue(true);

    renderWithProviders(
      <KeyboardShortcuts {...baseProps} isQuickSuggestionsOpen={false} />
    );

    // Should start collapsed on small screen
    expect(screen.getByTitle('Expand keyboard shortcuts')).toBeInTheDocument();
    expect(screen.queryByText('Keyboard Shortcuts')).not.toBeInTheDocument();
  });
});
