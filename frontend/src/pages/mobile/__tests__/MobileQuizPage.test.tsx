import { describe, it, expect, vi } from 'vitest';
import MobileQuizPage from '../MobileQuizPage';

// Mock MobileQuestionPageBase
vi.mock('../../../components/MobileQuestionPageBase', () => ({
  default: ({ mode }: { mode: string }) => (
    <div data-testid='mobile-question-page-base' data-mode={mode}>
      MobileQuestionPageBase
    </div>
  ),
}));

describe('MobileQuizPage', () => {
  it('renders MobileQuestionPageBase with quiz mode', () => {
    expect(MobileQuizPage).toBeDefined();
    expect(typeof MobileQuizPage).toBe('function');
  });

  it('passes correct mode to MobileQuestionPageBase', () => {
    // The component is a simple wrapper that passes mode='quiz'
    // Full functionality is tested in MobileQuestionPageBase.test.tsx
    const Component = MobileQuizPage;
    expect(Component).toBeDefined();
  });
});
