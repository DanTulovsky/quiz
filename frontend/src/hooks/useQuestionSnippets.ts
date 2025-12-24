import { useGetV1SnippetsByQuestionQuestionId } from '../api/api';

/**
 * Custom hook for loading snippets for a specific question.
 *
 * This hook implements async loading to ensure question display is not blocked
 * by snippet fetching. The snippets are loaded out-of-band after the question
 * is displayed.
 *
 * Performance characteristics:
 * - Automatically disabled when questionId is null/undefined
 * - Uses React Query for caching and deduplication
 * - Stale time set to prevent unnecessary refetches
 *
 * @param questionId - The ID of the question to fetch snippets for
 * @returns Object containing snippets array, loading state, and error state
 */
export const useQuestionSnippets = (questionId: number | null | undefined) => {
  const { data, isLoading, error } = useGetV1SnippetsByQuestionQuestionId(
    questionId || 0,
    {
      query: {
        // Only enable the query if we have a valid question ID
        enabled: !!questionId && questionId > 0,
        // Cache snippets for 5 minutes to avoid unnecessary refetches
        staleTime: 5 * 60 * 1000,
        // Keep data in cache for 10 minutes
        gcTime: 10 * 60 * 1000,
        // Don't retry on error (snippet highlighting is a nice-to-have feature)
        retry: false,
        // Don't refetch on window focus (not critical data)
        refetchOnWindowFocus: false,
      },
    }
  );

  return {
    snippets: data?.snippets || [],
    isLoading,
    error,
  };
};
