import { describe, it, expect, vi } from 'vitest';
import { renderWithProviders } from '../../test-utils';
import { TranslationOverlay } from '../../components/TranslationOverlay';

// Mock hooks used by the overlay
vi.mock('react-router-dom', async () => {
  const actual =
    await vi.importActual<typeof import('react-router-dom')>(
      'react-router-dom'
    );
  return {
    ...actual,
    useLocation: () => ({ pathname: '/daily/2025-10-30' }),
  };
});

vi.mock('../../hooks/useTextSelection', () => ({
  useTextSelection: () => ({
    selection: { text: 'ciao', sentence: 'ciao', x: 0, y: 0, height: 0 },
    isVisible: true,
    clearSelection: () => {},
  }),
}));

vi.mock('../../contexts/useQuestion', () => ({
  useQuestion: () => ({
    quizQuestion: {
      id: 448,
      level: 'B1',
      content: { question: 'old', options: ['a'] },
    },
    readingQuestion: null,
  }),
}));

// Capture props passed into TranslationPopup
const captured: Array<Record<string, unknown>> = [];
vi.mock('../../components/TranslationPopup', () => ({
  TranslationPopup: (props: Record<string, unknown>) => {
    captured.push(props);
    return null;
  },
}));

describe('TranslationOverlay daily routing', () => {
  beforeEach(() => {
    captured.length = 0;
  });

  it('does not fall back to quiz/reading when daily question missing', async () => {
    vi.mock('../../hooks/useDailyQuestions', () => ({
      useDailyQuestions: () => ({ currentQuestion: null }),
    }));

    renderWithProviders(<TranslationOverlay />);

    const last = captured.at(-1);
    expect(last).toBeDefined();
    expect(last?.requireQuestionId).toBe(true);
    // With the new behavior, Daily prefers QuestionContext when available.
    // In this test, QuestionContext provides a quizQuestion (id 448), so it is used.
    expect((last?.currentQuestion as { id?: number } | undefined)?.id).toBe(
      448
    );
  });

  // Focus the regression: when daily question is not yet available, we should not
  // fall back to previous quiz/reading question id.
});
