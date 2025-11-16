import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import TranslationPracticePage from './TranslationPracticePage';

vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({
    user: { preferred_language: 'ru', current_level: 'B1' },
    isAuthenticated: true,
  }),
}));

const mockSentence = {
  id: 1,
  sentence_text: 'Это не просто слова',
  source_language: 'ru',
  target_language: 'en',
  language_level: 'B1',
  source_type: 'ai',
  source_id: null,
  topic: null,
  created_at: new Date().toISOString(),
};

const mockSession = {
  id: 10,
  sentence_id: mockSentence.id,
  original_sentence: mockSentence.sentence_text,
  user_translation: 'These are not just words',
  translation_direction: 'learning_to_en',
  ai_feedback:
    'Good job! The meaning is accurate. Consider alternatives for style.\n\n- Word choice: “not just words” is idiomatic.\n',
  ai_score: 4.5,
  created_at: new Date().toISOString(),
};

vi.mock('../api/translationPracticeApi', () => {
  return {
    useGeneratePracticeSentence: () => ({
      mutateAsync: async () => mockSentence,
      isPending: false,
    }),
    useGetPracticeSentence: () => ({
      refetch: async () => ({ data: mockSentence, error: null }),
    }),
    useSubmitTranslation: () => ({
      mutateAsync: async () => mockSession,
      isPending: false,
    }),
    usePracticeHistory: () => ({ data: { sessions: [] } }),
    usePracticeStats: () => ({ data: {} }),
  };
});

describe('TranslationPracticePage - inline AI feedback', () => {
  it('renders AI feedback inline after submit', async () => {
    render(<TranslationPracticePage />);

    // Load sentence
    const generateBtn = await screen.findByRole('button', { name: /generate with ai/i });
    fireEvent.click(generateBtn);

    // Should show the sentence text
    await screen.findByText(mockSentence.sentence_text);

    // Type translation
    const textarea = screen.getByLabelText(/your translation/i);
    fireEvent.change(textarea, { target: { value: 'These are not just words' } });

    // Submit
    const submitBtn = screen.getByRole('button', { name: /submit for feedback/i });
    fireEvent.click(submitBtn);

    // Expect inline feedback card
    await waitFor(() => {
      expect(screen.getByText(/AI Feedback/i)).toBeInTheDocument();
      expect(screen.getByText(/Score: 4\.5 \/ 5/i)).toBeInTheDocument();
      expect(screen.getByText(/Good job!/i)).toBeInTheDocument();
      // A markdown list should render as text containing "Word choice"
      expect(screen.getByText(/Word choice/i)).toBeInTheDocument();
    });
  });
});


