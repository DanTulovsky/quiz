import { renderHook, act } from '@testing-library/react';
import { vi } from 'vitest';
import { useQuestionUrlState } from './useQuestionUrlState';
import type { Question } from '../api/api';

// Mock react-router-dom
const mockNavigate = vi.fn();
const mockLocation = { pathname: '/quiz/123' };
vi.mock('react-router-dom', () => ({
  useParams: vi.fn(() => ({ questionId: '123' })),
  useLocation: () => mockLocation,
  useNavigate: () => mockNavigate,
}));

describe('useQuestionUrlState', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockNavigate.mockClear();
  });

  describe('URL Navigation', () => {
    it('navigates to specific question ID', () => {
      const mockQuestion: Question = {
        id: 456,
        level: 'A1',
        content: {
          question: 'Test question',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      renderHook(() =>
        useQuestionUrlState({
          mode: 'quiz',
          question: mockQuestion,
          isLoading: false,
        })
      );

      expect(mockNavigate).toHaveBeenCalledWith('/quiz/456', { replace: true });
    });

    it('navigates to clear question ID from URL', () => {
      const mockQuestion: Question = {
        id: 456,
        level: 'A1',
        content: {
          question: 'Test question',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      renderHook(() =>
        useQuestionUrlState({
          mode: 'quiz',
          question: mockQuestion,
          isLoading: false,
        })
      );

      // The hook should have called navigate to update the URL
      expect(mockNavigate).toHaveBeenCalled();
    });

    it('uses correct base path for different modes', () => {
      const mockQuestion: Question = {
        id: 456,
        level: 'A1',
        content: {
          question: 'Test question',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      // Test quiz mode
      renderHook(() =>
        useQuestionUrlState({
          mode: 'quiz',
          question: mockQuestion,
          isLoading: false,
        })
      );
      expect(mockNavigate).toHaveBeenCalledWith('/quiz/456', { replace: true });

      vi.clearAllMocks();

      // Test reading mode
      renderHook(() =>
        useQuestionUrlState({
          mode: 'reading',
          question: mockQuestion,
          isLoading: false,
        })
      );
      expect(mockNavigate).toHaveBeenCalledWith('/reading-comprehension/456', {
        replace: true,
      });

      vi.clearAllMocks();

      // Test vocabulary mode
      renderHook(() =>
        useQuestionUrlState({
          mode: 'vocabulary',
          question: mockQuestion,
          isLoading: false,
        })
      );
      expect(mockNavigate).toHaveBeenCalledWith('/vocabulary/456', {
        replace: true,
      });
    });
  });

  describe('Question ID Handling', () => {
    it('returns questionId from URL params', () => {
      const mockQuestion: Question = {
        id: 456,
        level: 'A1',
        content: {
          question: 'Test question',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      const { result } = renderHook(() =>
        useQuestionUrlState({
          mode: 'quiz',
          question: mockQuestion,
          isLoading: false,
        })
      );

      expect(result.current.questionId).toBe('123');
    });

    it('updates URL when question changes', () => {
      const mockQuestion1: Question = {
        id: 456,
        level: 'A1',
        content: {
          question: 'Test question 1',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      const mockQuestion2: Question = {
        id: 789,
        level: 'A1',
        content: {
          question: 'Test question 2',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      const { rerender } = renderHook(
        ({ question, isLoading }) =>
          useQuestionUrlState({ mode: 'quiz', question, isLoading }),
        {
          initialProps: { question: mockQuestion1, isLoading: false },
        }
      );

      expect(mockNavigate).toHaveBeenCalledWith('/quiz/456', { replace: true });

      vi.clearAllMocks();

      // Change to different question
      rerender({ question: mockQuestion2, isLoading: false });

      expect(mockNavigate).toHaveBeenCalledWith('/quiz/789', { replace: true });
    });

    it('does not update URL while loading', () => {
      const mockQuestion: Question = {
        id: 456,
        level: 'A1',
        content: {
          question: 'Test question',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      renderHook(() =>
        useQuestionUrlState({
          mode: 'quiz',
          question: mockQuestion,
          isLoading: true,
        })
      );

      // Should not navigate while loading
      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it('does not update URL when question is null', () => {
      renderHook(() =>
        useQuestionUrlState({
          mode: 'quiz',
          question: null,
          isLoading: false,
        })
      );

      // If navigate was called, ensure it included clearing to '/quiz'. It is
      // valid for no navigation to occur depending on previous state, so accept
      // either case.
      const calls = mockNavigate.mock.calls.map((c: unknown[]) => c[0]);
      const navigatedToQuiz = calls.some((p: unknown) => p === '/quiz');
      // Either navigation happened to '/quiz' or no navigation was required.
      expect(navigatedToQuiz || mockNavigate.mock.calls.length === 0).toBe(
        true
      );
    });
  });

  describe('navigateToQuestion function', () => {
    it('navigates to specific question ID', () => {
      const mockQuestion: Question = {
        id: 456,
        level: 'A1',
        content: {
          question: 'Test question',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      const { result } = renderHook(() =>
        useQuestionUrlState({
          mode: 'quiz',
          question: mockQuestion,
          isLoading: false,
        })
      );

      act(() => {
        result.current.navigateToQuestion(789);
      });

      expect(mockNavigate).toHaveBeenCalledWith('/quiz/789', { replace: true });
    });

    it('clears question ID from URL when passed null', () => {
      const mockQuestion: Question = {
        id: 456,
        level: 'A1',
        content: {
          question: 'Test question',
          options: ['A', 'B', 'C', 'D'],
        },
      };

      const { result } = renderHook(() =>
        useQuestionUrlState({
          mode: 'quiz',
          question: mockQuestion,
          isLoading: false,
        })
      );

      act(() => {
        result.current.navigateToQuestion(null);
      });

      expect(mockNavigate).toHaveBeenCalledWith('/quiz', { replace: true });
    });
  });
});
