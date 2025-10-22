import { useGetV1SnippetsBySectionSectionId } from '../api/api';

export const useSectionSnippets = (sectionId: number | null | undefined) => {
  const { data, isLoading, error } = useGetV1SnippetsBySectionSectionId(
    sectionId || 0,
    {
      query: {
        // Only enable the query if we have a valid section ID
        enabled: !!sectionId && sectionId > 0,
        // Cache snippets for 5 minutes to avoid unnecessary refetches
        staleTime: 5 * 60 * 1000,
        // Keep data in cache for 10 minutes
        cacheTime: 10 * 60 * 1000,
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
