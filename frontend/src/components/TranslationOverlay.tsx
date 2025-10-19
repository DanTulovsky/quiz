import React from 'react';
import { useTextSelection } from '../hooks/useTextSelection';
import { TranslationPopup } from './TranslationPopup';
import { useQuestion } from '../contexts/useQuestion';

export const TranslationOverlay: React.FC = () => {
  const { selection, isVisible, clearSelection } = useTextSelection();
  const { quizQuestion, readingQuestion } = useQuestion();

  // Get the current question from either quiz or reading context
  const currentQuestion = quizQuestion || readingQuestion;

  if (!isVisible || !selection) {
    return null;
  }

  return (
    <TranslationPopup
      selection={selection}
      onClose={clearSelection}
      currentQuestion={currentQuestion}
    />
  );
};
