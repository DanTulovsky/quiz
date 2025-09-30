import type { Question, AnswerResponse } from '../api/api';
import type { ReactNode } from 'react';

export interface QuestionContextType {
  quizQuestion: Question | null;
  setQuizQuestion: (question: Question | null) => void;
  readingQuestion: Question | null;
  setReadingQuestion: (question: Question | null) => void;
  quizFeedback: AnswerResponse | null;
  setQuizFeedback: (feedback: AnswerResponse | null) => void;
  readingFeedback: AnswerResponse | null;
  setReadingFeedback: (feedback: AnswerResponse | null) => void;
  selectedAnswer: number | null;
  setSelectedAnswer: (answer: number | null) => void;
  isSubmitted: boolean;
  setIsSubmitted: (submitted: boolean) => void;
  showExplanation: boolean;
  setShowExplanation: React.Dispatch<React.SetStateAction<boolean>>;
}

export interface QuestionProviderProps {
  children: ReactNode;
}
