import { useContext } from 'react';
import { QuestionContext } from './QuestionContextContext';
import type { QuestionContextType } from './QuestionContext.types';

export const useQuestion = (): QuestionContextType => {
  const context = useContext(QuestionContext);
  if (context === undefined) {
    throw new Error('useQuestion must be used within a QuestionProvider');
  }
  return context;
};
