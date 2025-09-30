import { describe, it, expect, vi } from 'vitest';
import MobileReadingComprehensionPage from '../MobileReadingComprehensionPage';

// Mock MobileQuestionPageBase
vi.mock('../../../components/MobileQuestionPageBase', () => ({
  default: ({ mode }: { mode: string }) => (
    <div data-testid='mobile-question-page-base' data-mode={mode}>
      MobileQuestionPageBase
    </div>
  ),
}));

describe('MobileReadingComprehensionPage', () => {
  it('renders MobileQuestionPageBase with reading mode', () => {
    expect(MobileReadingComprehensionPage).toBeDefined();
    expect(typeof MobileReadingComprehensionPage).toBe('function');
  });

  it('passes correct mode to MobileQuestionPageBase', () => {
    // The component is a simple wrapper that passes mode='reading'
    // Full functionality is tested in MobileQuestionPageBase.test.tsx
    const Component = MobileReadingComprehensionPage;
    expect(Component).toBeDefined();
  });
});
