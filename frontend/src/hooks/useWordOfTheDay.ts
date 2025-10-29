import { useState, useCallback } from 'react';
import { useGetV1WordOfDayDate, WordOfTheDayDisplay } from '../api/api';
import { useAuth } from './useAuth';

export interface UseWordOfTheDayReturn {
  // State
  selectedDate: string;
  setSelectedDate: (date: string) => void;
  word: WordOfTheDayDisplay | undefined;

  // Loading states
  isLoading: boolean;

  // Actions
  goToPreviousDay: () => void;
  goToNextDay: () => void;
  goToToday: () => void;

  // Computed
  canGoPrevious: boolean;
  canGoNext: boolean;
}

// Format a date as YYYY-MM-DD
const formatDateLocal = (d: Date): string => {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

const getCurrentDateString = (): string => {
  return formatDateLocal(new Date());
};

export const useWordOfTheDay = (
  initialDate?: string
): UseWordOfTheDayReturn => {
  const { user } = useAuth();
  const today = getCurrentDateString();

  const [selectedDate, setSelectedDate] = useState<string>(
    initialDate || today
  );

  // Fetch word of the day for selected date
  const { data: word, isLoading } = useGetV1WordOfDayDate(selectedDate, {
    query: {
      enabled: !!user,
      refetchOnWindowFocus: false,
    },
  });

  // Helper to add/subtract days
  const addDays = useCallback((date: string, days: number): string => {
    // Parse the date string as local date to avoid timezone issues
    const [year, month, day] = date.split('-').map(Number);
    const d = new Date(year, month - 1, day);
    d.setDate(d.getDate() + days);
    return formatDateLocal(d);
  }, []);

  const goToPreviousDay = useCallback(() => {
    setSelectedDate(prev => addDays(prev, -1));
  }, [addDays]);

  const goToNextDay = useCallback(() => {
    setSelectedDate(prev => addDays(prev, 1));
  }, [addDays]);

  const goToToday = useCallback(() => {
    setSelectedDate(today);
  }, [today]);

  // Can navigate if not before a reasonable limit (e.g., 1 year ago)
  const oneYearAgo = addDays(today, -365);
  const canGoPrevious = selectedDate > oneYearAgo;
  // Can go next if selected date is before today (allow navigation up to but not including today)
  // Use explicit string comparison since YYYY-MM-DD format compares correctly as strings
  const canGoNext = selectedDate < today && selectedDate !== today;

  return {
    selectedDate,
    setSelectedDate,
    word,
    isLoading,
    goToPreviousDay,
    goToNextDay,
    goToToday,
    canGoPrevious,
    canGoNext,
  };
};
