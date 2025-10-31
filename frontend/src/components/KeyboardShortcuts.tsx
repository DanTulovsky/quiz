import React, { useState, useRef, useEffect } from 'react';
import { useHotkeys } from 'react-hotkeys-hook';
import { useMediaQuery, useLocalStorage } from '@mantine/hooks';
import {
  Badge,
  Group,
  Text,
  Stack,
  Button,
  Box,
  Transition,
} from '@mantine/core';
import * as TablerIcons from '@tabler/icons-react';

// Some icon components from the tabler icons package have typings that compile
// oddly in our environment; cast them to `any` for JSX usage. Silence the
// explicit-any ESLint rule locally for these casts since they are safe UI
// components and fixing upstream typings is out of scope for this change.
/* eslint-disable @typescript-eslint/no-explicit-any */
const IconChevronRight = TablerIcons.IconChevronRight as unknown as any;
const IconChevronLeft = TablerIcons.IconChevronLeft as unknown as any;
const IconKeyboard = TablerIcons.IconKeyboard as unknown as any;
/* eslint-enable @typescript-eslint/no-explicit-any */

interface KeyboardShortcutsProps {
  onAnswerSelect: (index: number) => void;
  onSubmit: () => void;
  onNextQuestion: () => void;
  onNewQuestion: () => void;
  onPreviousQuestion?: () => void;
  onToggleExplanation?: () => void;
  onReportIssue?: () => void;
  onMarkKnown?: () => void;
  onClearChat?: () => void;
  onToggleMaximize?: () => void;
  onToggleTTS?: () => void;
  onShowHistory?: () => void;
  /** When true, history modal is currently open */
  isHistoryOpen?: boolean;
  /** Called to hide/close the history modal */
  onHideHistory?: () => void;
  isSubmitted: boolean;
  hasSelectedAnswer: boolean;
  maxOptions: number;
  explanationAvailable?: boolean;
  ttsAvailable?: boolean;
  isQuickSuggestionsOpen?: boolean;
  quickSuggestionsCount?: number;
  isInputFocused?: boolean;
  isMarkKnownModalOpen?: boolean;
  isReportModalOpen?: boolean;
  isReportTextareaFocused?: boolean;
  // When true, show and enable left/right arrow navigation (Daily review mode)
  enablePrevNextArrows?: boolean;
}

const KeyboardShortcuts: React.FC<KeyboardShortcutsProps> = ({
  onAnswerSelect,
  onSubmit,
  onNextQuestion,
  onNewQuestion,
  onPreviousQuestion,
  onToggleExplanation,
  onReportIssue,
  onMarkKnown,
  onClearChat,
  onToggleMaximize,
  onToggleTTS,
  onShowHistory,
  isHistoryOpen = false,
  onHideHistory,
  isSubmitted,
  hasSelectedAnswer,
  maxOptions,
  explanationAvailable,
  ttsAvailable,
  isQuickSuggestionsOpen = false,
  quickSuggestionsCount = 0,
  isInputFocused,
  isMarkKnownModalOpen = false,
  isReportModalOpen = false,
  isReportTextareaFocused = false,
  enablePrevNextArrows = false,
}) => {
  // Check if screen is small or if the shortcuts would overlap main content
  // Use a more conservative breakpoint to prevent overlap
  const isSmallScreen = useMediaQuery(
    '(max-width: 1200px) or (max-height: 700px)'
  );

  // Additional check for content overlap
  const [hasContentOverlap, setHasContentOverlap] = useState(false);
  const shortcutsRef = useRef<HTMLDivElement>(null);

  // Function to check if shortcuts would overlap with main content
  const checkForOverlap = () => {
    if (!shortcutsRef.current) return;

    const shortcutsElement = shortcutsRef.current;
    const shortcutsRect = shortcutsElement.getBoundingClientRect();

    // Check if shortcuts panel would overlap with main content area
    const viewportWidth = window.innerWidth;

    // Try to find actual main content elements first
    const mainContent = document.querySelector(
      '[data-testid="question-card"], main, .main-content, [role="main"]'
    );

    if (mainContent) {
      const mainRect = mainContent.getBoundingClientRect();
      const shortcutsLeft = shortcutsRect.left;
      const mainRight = mainRect.right;

      // Consider overlap if shortcuts panel extends into the main content area
      const buffer = 20; // 20px buffer
      const overlaps = shortcutsLeft < mainRight - buffer;

      setHasContentOverlap(overlaps);
    } else {
      // Fallback to viewport-based calculation
      // Define the main content area more conservatively
      // Main content typically takes up most of the left side of the screen
      const mainContentWidth = viewportWidth * 0.85; // More conservative estimate
      const mainContentRight = mainContentWidth;

      // Check if shortcuts panel overlaps with main content
      const shortcutsLeft = shortcutsRect.left;

      // Consider overlap if shortcuts panel extends into the main content area
      // Add some buffer to prevent edge cases
      const buffer = 20; // 20px buffer
      const overlaps = shortcutsLeft < mainContentRight - buffer;

      setHasContentOverlap(overlaps);
    }
  };

  // Check for overlap on mount and when screen size changes
  useEffect(() => {
    const checkOverlap = () => {
      // Small delay to ensure DOM is ready
      setTimeout(checkForOverlap, 100);
    };

    // Also check when the component mounts and after a short delay
    checkOverlap();

    // Check again after a longer delay to ensure all content is loaded
    const delayedCheck = setTimeout(checkOverlap, 500);

    window.addEventListener('resize', checkOverlap);

    return () => {
      window.removeEventListener('resize', checkOverlap);
      clearTimeout(delayedCheck);
    };
  }, []);

  // Start collapsed on smaller screens or when there's content overlap
  const shouldStartCollapsed = isSmallScreen || hasContentOverlap;

  // Persist user's preference so it remains consistent across navigation
  const [persistedExpanded, setPersistedExpanded] = useLocalStorage<
    boolean | null
  >({
    key: 'keyboard-shortcuts-expanded',
    defaultValue: null,
  });

  const [isExpanded, setIsExpanded] = useState<boolean>(
    // Use persisted preference if available, otherwise default based on screen size
    persistedExpanded !== null ? persistedExpanded : !shouldStartCollapsed
  );
  const [userManuallyExpanded, setUserManuallyExpanded] = useState(false);

  // Sync from persisted preference if present
  useEffect(() => {
    if (persistedExpanded !== null && persistedExpanded !== isExpanded) {
      setIsExpanded(persistedExpanded);
    }
  }, [persistedExpanded, isExpanded]);

  // Update expanded state when screen size or overlap changes
  // Only auto-collapse if the user hasn't manually expanded it
  useEffect(() => {
    // Only auto-collapse on overlap/small screens when the user has not set a preference yet
    if (persistedExpanded === null) {
      if (shouldStartCollapsed && isExpanded && !userManuallyExpanded) {
        setIsExpanded(false);
      }
    }
  }, [
    shouldStartCollapsed,
    isExpanded,
    userManuallyExpanded,
    persistedExpanded,
  ]);

  // Reset user manual expansion when screen size changes significantly
  useEffect(() => {
    if (persistedExpanded === null && isSmallScreen) {
      setUserManuallyExpanded(false);
    }
  }, [isSmallScreen, persistedExpanded]);

  // Ref to always have the latest value of isQuickSuggestionsOpen
  const isQuickSuggestionsOpenRef = useRef(isQuickSuggestionsOpen);
  useEffect(() => {
    isQuickSuggestionsOpenRef.current = isQuickSuggestionsOpen;
  }, [isQuickSuggestionsOpen]);

  // Only show 1-4 answer selection if quick suggestions are not open
  useHotkeys(
    ['1', '2', '3', '4'],
    e => {
      if (isQuickSuggestionsOpenRef.current) return;
      if (isSubmitted) return;
      if (isMarkKnownModalOpen) return; // Don't handle number keys when mark known modal is open
      if (isReportModalOpen) return; // Don't handle number keys when report modal is open
      if (maxOptions <= 0) return; // Don't handle if shuffling is not ready
      const index = parseInt(e.key) - 1;
      if (index >= 0 && index < maxOptions) {
        e.preventDefault();
        onAnswerSelect(index);
        // Try to simulate a click on the corresponding Radio element so the
        // controlled Radio.Group updates visually in all cases (helps in some
        // environments where context updates may not immediately reflect).
        try {
          const radioRoot = document.querySelector(
            `[data-testid="option-${index}"]`
          );
          const input = radioRoot?.querySelector(
            'input[type="radio"]'
          ) as HTMLElement | null;
          if (input) {
            input.click();
          }
        } catch {
          // ignore
        }
      }
    },
    {
      enableOnFormTags: false,
      preventDefault: true,
      enabled: !isQuickSuggestionsOpen && maxOptions > 0,
    },
    [
      isSubmitted,
      isMarkKnownModalOpen,
      isReportModalOpen,
      maxOptions,
      onAnswerSelect,
      isQuickSuggestionsOpen,
    ]
  );

  // Handle Enter key for submission or next question (only when quick suggestions are closed)
  useHotkeys(
    'enter',
    e => {
      // Check if the focused element is the chat input
      const activeElement = document.activeElement;
      const isChatInput = activeElement?.id === 'ai-chat-input';

      // If chat input is focused, let it handle Enter (for sending messages)
      if (isChatInput) {
        return;
      }

      // Don't handle Enter when modal is open
      if (isMarkKnownModalOpen || isReportModalOpen) {
        return;
      }

      if (!isSubmitted && hasSelectedAnswer) {
        e.preventDefault();
        // Find and click the QuestionCard's submit button instead of calling onSubmit directly
        const submitButton = document.querySelector(
          '[data-testid="submit-button"]'
        ) as HTMLElement;
        if (submitButton) {
          submitButton.click();
        } else {
          // Fallback to onSubmit if submit button not found
          onSubmit();
        }
      } else if (isSubmitted) {
        e.preventDefault();
        onNextQuestion();
      }
    },
    {
      enableOnFormTags: true,
      preventDefault: true,
      enabled: !isQuickSuggestionsOpen,
    },
    [
      isSubmitted,
      hasSelectedAnswer,
      isMarkKnownModalOpen,
      isReportModalOpen,
      onSubmit,
      onNextQuestion,
    ]
  );

  // Handle 't' key to scroll to top (same gating as Enter: ignore when chat input focused or modals open)
  useHotkeys(
    't',
    e => {
      // Only act when not typing in chat and no modals are open
      if (isInputFocused) return;
      if (isMarkKnownModalOpen || isReportModalOpen) return;
      e.preventDefault();
      window.scrollTo({ top: 0, behavior: 'smooth' });
    },
    {
      enableOnFormTags: true,
      preventDefault: true,
      enabled:
        !isQuickSuggestionsOpen &&
        !isInputFocused &&
        !isMarkKnownModalOpen &&
        !isReportModalOpen,
    },
    [
      isQuickSuggestionsOpen,
      isInputFocused,
      isMarkKnownModalOpen,
      isReportModalOpen,
    ]
  );

  // Handle 'n' key for next/new question
  useHotkeys(
    'n',
    e => {
      e.preventDefault();
      if (isSubmitted) {
        onNextQuestion();
      } else {
        onNewQuestion();
      }
    },
    { enableOnFormTags: false, preventDefault: true },
    [isSubmitted, onNextQuestion, onNewQuestion]
  );

  // Left/Right arrows for previous/next (enabled only in review mode)
  useHotkeys(
    'arrowleft',
    e => {
      if (enablePrevNextArrows && onPreviousQuestion) {
        e.preventDefault();
        onPreviousQuestion();
      }
    },
    {
      enableOnFormTags: false,
      preventDefault: true,
      enabled: enablePrevNextArrows,
    },
    [enablePrevNextArrows, onPreviousQuestion]
  );

  useHotkeys(
    'arrowright',
    e => {
      if (enablePrevNextArrows) {
        e.preventDefault();
        onNextQuestion();
      }
    },
    {
      enableOnFormTags: false,
      preventDefault: true,
      enabled: enablePrevNextArrows,
    },
    [enablePrevNextArrows, onNextQuestion]
  );

  // Handle 'e' key for explanation toggle
  useHotkeys(
    'e',
    e => {
      if (isSubmitted && explanationAvailable && onToggleExplanation) {
        e.preventDefault();
        onToggleExplanation();
      }
    },
    { enableOnFormTags: false, preventDefault: true },
    [isSubmitted, explanationAvailable, onToggleExplanation]
  );

  // Handle 'r' key for reporting issue
  useHotkeys(
    'r',
    e => {
      if (onReportIssue && !isReportModalOpen && !isMarkKnownModalOpen) {
        e.preventDefault();
        onReportIssue();
      }
    },
    { enableOnFormTags: false, preventDefault: true },
    [onReportIssue, isReportModalOpen, isMarkKnownModalOpen]
  );

  // Handle 'k' key for marking as known
  useHotkeys(
    'k',
    e => {
      if (onMarkKnown && !isMarkKnownModalOpen && !isReportModalOpen) {
        e.preventDefault();
        onMarkKnown();
      }
    },
    { enableOnFormTags: false, preventDefault: true },
    [onMarkKnown, isMarkKnownModalOpen, isReportModalOpen]
  );

  // Handle 'd' key for clearing chat
  useHotkeys(
    'd',
    e => {
      if (onClearChat) {
        e.preventDefault();
        onClearChat();
      }
    },
    { enableOnFormTags: false, preventDefault: true },
    [onClearChat]
  );

  // Handle 'm' key for maximizing/minimizing chat
  useHotkeys(
    'm',
    e => {
      if (onToggleMaximize) {
        e.preventDefault();
        onToggleMaximize();
      }
    },
    { enableOnFormTags: false, preventDefault: true },
    [onToggleMaximize]
  );

  // Handle 'p' key for play/stop audio (TTS)
  useHotkeys(
    'p',
    e => {
      // Only act when not typing in chat and no modals are open
      if (isInputFocused) return;
      if (isQuickSuggestionsOpen) return;
      if (isMarkKnownModalOpen || isReportModalOpen) return;

      if (onToggleTTS && ttsAvailable) {
        e.preventDefault();
        onToggleTTS();
      }
    },
    {
      enableOnFormTags: false,
      preventDefault: true,
      enabled:
        ttsAvailable &&
        !isQuickSuggestionsOpen &&
        !isInputFocused &&
        !isMarkKnownModalOpen &&
        !isReportModalOpen,
    },
    [
      onToggleTTS,
      ttsAvailable,
      isQuickSuggestionsOpen,
      isInputFocused,
      isReportModalOpen,
      isMarkKnownModalOpen,
    ]
  );

  // Handle '<' and '>' shortcuts to collapse/expand the shortcuts panel
  useHotkeys(
    'shift+comma',
    e => {
      e.preventDefault();
      setIsExpanded(true);
      setPersistedExpanded(true);
      setUserManuallyExpanded(true);
    },
    { enableOnFormTags: false, preventDefault: true },
    []
  );

  useHotkeys(
    'shift+period',
    e => {
      e.preventDefault();
      setIsExpanded(false);
      setPersistedExpanded(false);
    },
    { enableOnFormTags: false, preventDefault: true },
    []
  );

  // Handle 'h' key to open/close question history
  useHotkeys(
    'h',
    e => {
      // Only act when not typing in chat and no modals are open
      if (isInputFocused) return;
      if (isMarkKnownModalOpen || isReportModalOpen) return;

      // If history is currently open, prefer closing it
      if (isHistoryOpen && onHideHistory) {
        e.preventDefault();
        onHideHistory();
        return;
      }

      if (!onShowHistory) return;
      e.preventDefault();
      onShowHistory();
    },
    {
      enableOnFormTags: true,
      preventDefault: true,
      enabled:
        !isQuickSuggestionsOpen &&
        !isInputFocused &&
        !isMarkKnownModalOpen &&
        !isReportModalOpen,
    },
    [
      isQuickSuggestionsOpen,
      isInputFocused,
      isMarkKnownModalOpen,
      isReportModalOpen,
      onShowHistory,
      onHideHistory,
      isHistoryOpen,
    ]
  );

  return (
    <Box
      ref={shortcutsRef}
      data-testid='keyboard-shortcuts'
      style={{
        position: 'fixed',
        top: '50%',
        right: 0,
        transform: 'translateY(-50%)',
        zIndex: 1000,
        pointerEvents: 'none',
      }}
    >
      <Group gap={0} align='stretch' style={{ pointerEvents: 'auto' }}>
        {/* Collapse/Expand button */}
        <Button
          variant='subtle'
          size='sm'
          p={8}
          onClick={() => {
            const newExpandedState = !isExpanded;
            setIsExpanded(newExpandedState);
            setPersistedExpanded(newExpandedState);
            // Track user manual expansion
            if (newExpandedState) {
              setUserManuallyExpanded(true);
            } else {
              setUserManuallyExpanded(false);
            }
          }}
          style={{
            borderRadius: '8px 0 0 8px',
            border: '1px solid var(--mantine-color-default-border)',
            borderRight: 'none',
            background: 'var(--mantine-color-body)',
            minWidth: 'auto',
            height: 'auto',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
          title={
            isExpanded
              ? 'Collapse keyboard shortcuts'
              : 'Expand keyboard shortcuts'
          }
        >
          {isExpanded ? (
            <IconChevronRight size={16} />
          ) : (
            <IconChevronLeft size={16} />
          )}
        </Button>

        {/* Shortcuts panel */}
        <Transition
          mounted={isExpanded}
          transition='slide-left'
          duration={200}
          timingFunction='ease'
        >
          {styles => (
            <Box
              style={{
                ...styles,
                borderRadius: '0 8px 8px 0',
                border: '1px solid var(--mantine-color-default-border)',
                background: 'var(--mantine-color-body)',
                padding: 'var(--mantine-spacing-sm)',
                maxWidth: '280px',
                minWidth: '240px',
              }}
            >
              <Stack gap='xs'>
                <Group gap='xs' align='center'>
                  <IconKeyboard size={16} />
                  <Text size='sm' fw={500}>
                    Keyboard Shortcuts
                  </Text>
                </Group>

                {/* Expand/Collapse shortcuts info */}
                <Group gap='xs' align='center'>
                  <Badge size='sm' variant='light' style={{ minWidth: '32px' }}>
                    {'<'}
                  </Badge>
                  <Badge size='sm' variant='light' style={{ minWidth: '32px' }}>
                    {'>'}
                  </Badge>
                  <Text size='xs' c='dimmed'>
                    Expand / Collapse panel
                  </Text>
                </Group>

                <Stack gap='xs'>
                  {isMarkKnownModalOpen ? (
                    // Show only modal-specific shortcuts when mark known modal is open
                    <>
                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          1-5
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          Select confidence level
                        </Text>
                      </Group>
                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          Esc
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          Cancel
                        </Text>
                      </Group>
                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          ↵
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          Submit
                        </Text>
                      </Group>
                    </>
                  ) : isReportModalOpen ? (
                    // Show different shortcuts based on whether textarea is focused
                    isReportTextareaFocused ? (
                      // When textarea is focused, show only Esc
                      <>
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            Esc
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Cancel
                          </Text>
                        </Group>
                      </>
                    ) : (
                      // When textarea is not focused, show all modal shortcuts
                      <>
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            I
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Focus text area
                          </Text>
                        </Group>
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            Esc
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Cancel
                          </Text>
                        </Group>
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            ↵
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Submit
                          </Text>
                        </Group>
                      </>
                    )
                  ) : isInputFocused ? (
                    <>
                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          Esc
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          Reset focus / enable hotkeys
                        </Text>
                      </Group>
                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          ↑↓
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          Scroll chat
                        </Text>
                      </Group>
                    </>
                  ) : (
                    <>
                      {/* Navigation shortcuts */}
                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          ⇧1-3
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          Main Navigation
                        </Text>
                      </Group>

                      {/* Previous/Next with arrow keys in Daily review mode */}
                      {enablePrevNextArrows && (
                        <>
                          <Group gap='xs' align='center'>
                            <Badge
                              size='sm'
                              variant='light'
                              style={{ minWidth: '32px' }}
                            >
                              ←
                            </Badge>
                            <Text size='xs' c='dimmed'>
                              Previous question
                            </Text>
                          </Group>
                          <Group gap='xs' align='center'>
                            <Badge
                              size='sm'
                              variant='light'
                              style={{ minWidth: '32px' }}
                            >
                              →
                            </Badge>
                            <Text size='xs' c='dimmed'>
                              Next question
                            </Text>
                          </Group>
                        </>
                      )}

                      {/* Show 1-4 only if quick suggestions are not open */}
                      {!isQuickSuggestionsOpen && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            1-4
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Select answer
                          </Text>
                        </Group>
                      )}

                      {/* Scroll to top */}
                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          T
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          Scroll to top
                        </Text>
                      </Group>

                      {/* History shortcut */}
                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          H
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          Open question history
                        </Text>
                      </Group>

                      {/* Show 0-9 for quick suggestions if open */}
                      {isQuickSuggestionsOpen && quickSuggestionsCount > 0 && (
                        <>
                          <Group gap='xs' align='center'>
                            <Badge
                              size='sm'
                              variant='light'
                              style={{ minWidth: '32px' }}
                            >
                              {quickSuggestionsCount > 9
                                ? '0-9'
                                : `1-${quickSuggestionsCount}`}
                            </Badge>
                            <Text size='xs' c='dimmed'>
                              Select suggestion
                            </Text>
                          </Group>
                          <Group gap='xs' align='center'>
                            <Badge
                              size='sm'
                              variant='light'
                              style={{ minWidth: '32px' }}
                            >
                              ↑↓
                            </Badge>
                            <Text size='xs' c='dimmed'>
                              Navigate suggestions
                            </Text>
                          </Group>
                          <Group gap='xs' align='center'>
                            <Badge
                              size='sm'
                              variant='light'
                              style={{ minWidth: '32px' }}
                            >
                              ↵
                            </Badge>
                            <Text size='xs' c='dimmed'>
                              Execute selected
                            </Text>
                          </Group>
                        </>
                      )}

                      {/* Show Submit (↵) only when not submitted and an answer is selected */}
                      {!isSubmitted && hasSelectedAnswer && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            ↵
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Submit
                          </Text>
                        </Group>
                      )}
                      {/* Show Next question (↵) only when submitted */}
                      {isSubmitted && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            ↵
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Next question
                          </Text>
                        </Group>
                      )}

                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          N
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          {isSubmitted ? 'Next question' : 'New question'}
                        </Text>
                      </Group>

                      {/* onFocusChat && (
                        <Group gap="xs" align="center">
                          <Badge
                            size="sm"
                            variant="light"
                            style={{ minWidth: '32px' }}
                          >
                            C
                          </Badge>
                          <Text size="xs" c="dimmed">
                            Focus chat
                          </Text>
                        </Group>
                      ) */}

                      <Group gap='xs' align='center'>
                        <Badge
                          size='sm'
                          variant='light'
                          style={{ minWidth: '32px' }}
                        >
                          Q
                        </Badge>
                        <Text size='xs' c='dimmed'>
                          AI Chat: Quick suggestions
                        </Text>
                      </Group>

                      {isSubmitted && explanationAvailable && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            E
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Toggle explanation
                          </Text>
                        </Group>
                      )}

                      {ttsAvailable && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            P
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Play/Stop audio
                          </Text>
                        </Group>
                      )}

                      {onReportIssue && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            R
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Report issue
                          </Text>
                        </Group>
                      )}

                      {onMarkKnown && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            K
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            Rate question knowledge
                          </Text>
                        </Group>
                      )}

                      {onClearChat && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            D
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            AI Chat: Clear
                          </Text>
                        </Group>
                      )}

                      {onToggleMaximize && (
                        <Group gap='xs' align='center'>
                          <Badge
                            size='sm'
                            variant='light'
                            style={{ minWidth: '32px' }}
                          >
                            M
                          </Badge>
                          <Text size='xs' c='dimmed'>
                            AI Chat: Toggle maximize
                          </Text>
                        </Group>
                      )}
                    </>
                  )}
                </Stack>
              </Stack>
            </Box>
          )}
        </Transition>
      </Group>
    </Box>
  );
};

export default KeyboardShortcuts;
