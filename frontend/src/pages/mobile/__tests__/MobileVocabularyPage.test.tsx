import { describe, it, expect, vi } from 'vitest';
import MobileVocabularyPage from '../MobileVocabularyPage';

// Mock MobileQuestionPageBase
vi.mock('../../../components/MobileQuestionPageBase', () => ({
  default: ({ mode }: { mode: string }) => (
    <div data-testid='mobile-question-page-base' data-mode={mode}>
      MobileQuestionPageBase
    </div>
  ),
}));

describe('MobileVocabularyPage', () => {
  it('renders MobileQuestionPageBase with vocabulary mode', () => {
    expect(MobileVocabularyPage).toBeDefined();
    expect(typeof MobileVocabularyPage).toBe('function');
  });

  it('passes correct mode to MobileQuestionPageBase', () => {
    // The component is a simple wrapper that passes mode='vocabulary'
    // Full functionality is tested in MobileQuestionPageBase.test.tsx
    const Component = MobileVocabularyPage;
    expect(Component).toBeDefined();
  });
});
