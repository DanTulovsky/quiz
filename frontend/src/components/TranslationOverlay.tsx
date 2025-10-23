import React, { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { useTextSelection } from '../hooks/useTextSelection';
import { TranslationPopup } from './TranslationPopup';
import { useQuestion } from '../contexts/useQuestion';
import { useDailyQuestions } from '../hooks/useDailyQuestions';
import { useStory } from '../hooks/useStory';
import { Question } from '../api/api';

// Type for story context when no question is available
interface StoryContext {
  story_id: number;
  section_id?: number;
}

export const TranslationOverlay: React.FC = () => {
  const { selection, isVisible, clearSelection } = useTextSelection();
  const { quizQuestion, readingQuestion } = useQuestion();
  const location = useLocation();

  // Get daily question if we're on the daily page (both desktop and mobile)
  const isDailyPage =
    location.pathname.startsWith('/daily') ||
    location.pathname.startsWith('/m/daily');
  const { currentQuestion: dailyQuestion } = useDailyQuestions();

  // Get story context if we're on the story page (both desktop and mobile)
  const isStoryPage =
    location.pathname.startsWith('/story') ||
    location.pathname.startsWith('/m/story');
  const { currentStory, currentSection, viewMode, currentSectionIndex } =
    useStory();

  // Mobile pages now update the QuestionContext just like desktop pages,
  // so we don't need special handling for mobile question pages

  // Get the current question from either quiz, reading, daily, story, or mobile context
  // Use useMemo to ensure it updates when story context changes
  const currentQuestion = useMemo((): Question | StoryContext | null => {
    let question: Question | StoryContext | null =
      quizQuestion || readingQuestion;

    if (isDailyPage && dailyQuestion) {
      // For daily questions, we need to create a question object with the correct ID
      question = {
        ...dailyQuestion.question,
        id: dailyQuestion.question_id,
      };
    } else if (isStoryPage && currentStory) {
      // For stories, create a context object with story/section info
      // NOTE: Do NOT set 'id' field here - stories don't have questions, so question_id should not be set
      question = {
        story_id: currentStory.id!, // Always set story_id to the story ID
        // Add section context if we're in section view and have a current section
        ...(viewMode === 'section' &&
          currentSection && { section_id: currentSection.id }),
      };
    }

    return question;
  }, [
    quizQuestion,
    readingQuestion,
    isDailyPage,
    dailyQuestion,
    isStoryPage,
    currentStory,
    currentSection,
    viewMode,
  ]);

  if (!isVisible || !selection) {
    return null;
  }

  return (
    <TranslationPopup
      key={`${currentStory?.id}-${currentSection?.id}-${currentSectionIndex}`}
      selection={selection}
      onClose={clearSelection}
      currentQuestion={currentQuestion}
    />
  );
};
