import React, { useState } from 'react';
import { Question, AnswerResponse } from '../api/api';
import type {
  QuestionContextType,
  QuestionProviderProps,
} from './QuestionContext.types';
import { QuestionContext } from './QuestionContextContext';

export const QuestionProvider: React.FC<QuestionProviderProps> = ({
  children,
}) => {
  const [quizQuestion, setQuizQuestion] = useState<Question | null>(null);
  const [readingQuestion, setReadingQuestion] = useState<Question | null>(null);
  const [quizFeedback, setQuizFeedback] = useState<AnswerResponse | null>(null);
  const [readingFeedback, setReadingFeedback] = useState<AnswerResponse | null>(
    null
  );
  const [selectedAnswer, setSelectedAnswer] = useState<number | null>(null);
  const [isSubmitted, setIsSubmitted] = useState(false);
  const [showExplanation, setShowExplanation] = useState(false);

  const value: QuestionContextType = {
    quizQuestion,
    setQuizQuestion,
    readingQuestion,
    setReadingQuestion,
    quizFeedback,
    setQuizFeedback,
    readingFeedback,
    setReadingFeedback,
    selectedAnswer,
    setSelectedAnswer,
    isSubmitted,
    setIsSubmitted,
    showExplanation,
    setShowExplanation,
  };

  return (
    <QuestionContext.Provider value={value}>
      {children}
    </QuestionContext.Provider>
  );
};
