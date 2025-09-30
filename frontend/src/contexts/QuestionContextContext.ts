import { createContext } from 'react';
import type { QuestionContextType } from './QuestionContext.types';

export const QuestionContext = createContext<QuestionContextType | undefined>(
  undefined
);
