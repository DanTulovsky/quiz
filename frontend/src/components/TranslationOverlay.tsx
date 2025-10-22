import React from 'react';
import { useLocation } from 'react-router-dom';
import { useTextSelection } from '../hooks/useTextSelection';
import { TranslationPopup } from './TranslationPopup';
import { useQuestion } from '../contexts/useQuestion';
import { useDailyQuestions } from '../hooks/useDailyQuestions';
import { useStory } from '../hooks/useStory';

export const TranslationOverlay: React.FC = () => {
  const { selection, isVisible, clearSelection } = useTextSelection();
  const { quizQuestion, readingQuestion } = useQuestion();
  const location = useLocation();

  // Get daily question if we're on the daily page (both desktop and mobile)
  const isDailyPage = location.pathname.startsWith('/daily') || location.pathname.startsWith('/m/daily');
  const { currentQuestion: dailyQuestion } = useDailyQuestions();

  // Get story context if we're on the story page (both desktop and mobile)
  const isStoryPage = location.pathname.startsWith('/story') || location.pathname.startsWith('/m/story');
  const { currentStory, currentSection, viewMode } = useStory();

  // Mobile pages now update the QuestionContext just like desktop pages,
  // so we don't need special handling for mobile question pages

  // Get the current question from either quiz, reading, daily, story, or mobile context
  let currentQuestion = quizQuestion || readingQuestion;

  if (isDailyPage && dailyQuestion) {
    // For daily questions, we need to create a question object with the correct ID
    currentQuestion = {
      ...dailyQuestion.question,
      id: dailyQuestion.question_id,
    };
  } else if (isStoryPage && currentStory) {
    // For stories, we need to create a question object with story/section context
    // We'll use the story ID as the "question" ID for snippet context
    currentQuestion = {
      id: currentStory.id!,
      story_id: currentStory.id!, // Always set story_id to the story ID
      // Add section context if we're in section view and have a current section
      ...(viewMode === 'section' && currentSection && { section_id: currentSection.id }),
    } as { id: number; section_id?: number; story_id?: number }; // Type assertion since we're creating a custom object
  }

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
