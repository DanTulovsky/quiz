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

  // Get the current question strictly based on active route to avoid stale IDs
  const currentQuestion = useMemo((): Question | StoryContext | null => {
    // Daily routes: prefer the globally-published question from QuestionContext
    // (kept in sync by DailyPage). Fall back to hook-based dailyQuestion only
    // if context is not yet populated.
    if (isDailyPage) {
      // Zero-risk mode: only trust the globally published question from
      // QuestionContext; do not fall back to a separate hook value.
      return quizQuestion || null;
    }

    // Story routes: provide story/section context, without an id
    if (isStoryPage && currentStory) {
      return {
        story_id: currentStory.id!,
        ...(viewMode === 'section' &&
          currentSection && {
            section_id: currentSection.id,
          }),
      };
    }

    // Quiz/Reading routes: use question context
    return quizQuestion || readingQuestion || null;
  }, [
    isDailyPage,
    dailyQuestion,
    isStoryPage,
    currentStory,
    currentSection,
    viewMode,
    quizQuestion,
    readingQuestion,
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
      requireQuestionId={isDailyPage}
    />
  );
};
