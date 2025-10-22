import React from 'react';
import { useLocation } from 'react-router-dom';
import { useTextSelection } from '../hooks/useTextSelection';
import { TranslationPopup } from './TranslationPopup';
import { useQuestion } from '../contexts/useQuestion';
import { useDailyQuestions } from '../hooks/useDailyQuestions';

export const TranslationOverlay: React.FC = () => {
  const { selection, isVisible, clearSelection } = useTextSelection();
  const { quizQuestion, readingQuestion } = useQuestion();
  const location = useLocation();
  
  // Get daily question if we're on the daily page
  const isDailyPage = location.pathname.startsWith('/daily');
  const { currentQuestion: dailyQuestion } = useDailyQuestions();

  // Get the current question from either quiz, reading, or daily context
  // For daily questions, we need to create a question object with the correct ID
  const currentQuestion = quizQuestion || readingQuestion || (isDailyPage && dailyQuestion ? {
    ...dailyQuestion.question,
    id: dailyQuestion.question_id
  } : null);

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
