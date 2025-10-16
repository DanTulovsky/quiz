import { useState, useCallback, useMemo } from 'react';
import {
  useInfiniteQuery,
  UseInfiniteQueryOptions,
} from '@tanstack/react-query';

export interface PaginationOptions {
  initialLimit?: number;
  maxLimit?: number;
  enableInfiniteScroll?: boolean;
}

export interface PaginationState {
  limit: number;
  offset: number;
  currentPage: number;
  totalPages: number;
  totalItems: number;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
}

export interface UsePaginationReturn<TData> {
  // Data
  data: TData[] | undefined;
  allData: TData[];
  isLoading: boolean;
  isFetching: boolean;
  isError: boolean;
  error: Error | null;

  // Pagination state
  pagination: PaginationState;

  // Actions
  loadMore: () => void;
  goToPage: (page: number) => void;
  goToNextPage: () => void;
  goToPreviousPage: () => void;
  reset: () => void;

  // Infinite scroll helpers
  loadMoreRef: (node?: Element | null) => void;
}

export function usePagination<TData = unknown>(
  queryKey: string[],
  queryFn: (params: { limit: number; offset: number }) => Promise<{
    items: TData[];
    total: number;
  }>,
  options: PaginationOptions = {}
): UsePaginationReturn<TData> {
  const { initialLimit = 20, enableInfiniteScroll = false } = options;

  const [limit] = useState(initialLimit);
  const [currentPage, setCurrentPage] = useState(1);
  const [allData, setAllData] = useState<TData[]>([]);

  const infiniteQueryOptions: UseInfiniteQueryOptions = {
    queryKey: [...queryKey, 'pagination'],
    queryFn: ({ pageParam = 0 }) =>
      queryFn({
        limit,
        offset: pageParam * limit,
      }),
    getNextPageParam: (lastPage, pages) => {
      const totalLoaded = pages.reduce((acc, page) => {
        if (page?.items && Array.isArray(page.items)) {
          return acc + page.items.length;
        } else if (page?.conversations && Array.isArray(page.conversations)) {
          return acc + page.conversations.length;
        }
        return acc;
      }, 0);

      const total = lastPage?.total || 0;

      return totalLoaded < total ? pages.length : undefined;
    },
    getPreviousPageParam: (firstPage, pages) => {
      return pages.length > 1 ? pages.length - 2 : undefined;
    },
  };

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isLoading,
    isError,
    error,
    refetch,
  } = useInfiniteQuery(infiniteQueryOptions);

  // Flatten all pages into a single array
  const flattenedData = useMemo(() => {
    if (!data?.pages) {
      return [];
    }

    return (
      data.pages.flatMap(page => {
        // React Query stores the return value from queryFn directly in page
        // My query function returns {items: [...], total: 1}
        if (page && Array.isArray(page.items)) {
          return page.items;
        } else if (page && page.conversations) {
          return page.conversations;
        } else {
          return [];
        }
      }) || []
    );
  }, [data]);

  // Update allData when flattenedData changes
  useMemo(() => {
    setAllData(flattenedData);
  }, [flattenedData]);

  const totalItems = useMemo(() => {
    if (!data?.pages?.[0]) {
      return 0;
    }

    const firstPage = data.pages[0];

    let total = 0;
    if (firstPage?.total !== undefined) {
      total = firstPage.total;
    } else if (firstPage?.conversations !== undefined) {
      // For conversations API, total is separate from conversations array
      total = firstPage.total || 0;
    }

    return total;
  }, [data]);

  const totalPages = useMemo(() => {
    return Math.ceil(totalItems / limit);
  }, [totalItems, limit]);

  const pagination: PaginationState = {
    limit,
    offset: (currentPage - 1) * limit,
    currentPage,
    totalPages,
    totalItems,
    hasNextPage: currentPage < totalPages,
    hasPreviousPage: currentPage > 1,
  };

  const loadMore = useCallback(() => {
    if (hasNextPage && !isFetching) {
      fetchNextPage();
      setCurrentPage(prev => Math.min(prev + 1, totalPages));
    }
  }, [hasNextPage, isFetching, fetchNextPage, totalPages]);

  const goToPage = useCallback(
    (page: number) => {
      if (page >= 1 && page <= totalPages && page !== currentPage) {
        setCurrentPage(page);
        // For now, we'll refetch with new offset
        // In a more sophisticated implementation, we might want to cache pages
        refetch();
      }
    },
    [totalPages, currentPage, refetch]
  );

  const goToNextPage = useCallback(() => {
    if (pagination.hasNextPage) {
      goToPage(currentPage + 1);
    }
  }, [pagination.hasNextPage, goToPage, currentPage]);

  const goToPreviousPage = useCallback(() => {
    if (pagination.hasPreviousPage) {
      goToPage(currentPage - 1);
    }
  }, [pagination.hasPreviousPage, goToPage, currentPage]);

  const reset = useCallback(() => {
    setCurrentPage(1);
    setAllData([]);
    refetch();
  }, [refetch]);

  // Intersection Observer for infinite scroll
  const loadMoreRef = useCallback(
    (node: Element | null) => {
      if (!node || !enableInfiniteScroll) return;

      const observer = new IntersectionObserver(
        entries => {
          if (entries[0].isIntersecting && hasNextPage && !isFetching) {
            loadMore();
          }
        },
        { threshold: 0.1 }
      );

      observer.observe(node);

      return () => {
        observer.disconnect();
      };
    },
    [enableInfiniteScroll, hasNextPage, isFetching, loadMore]
  );

  return {
    data: flattenedData,
    allData,
    isLoading,
    isFetching,
    isError,
    error: error as Error | null,
    pagination,
    loadMore,
    goToPage,
    goToNextPage,
    goToPreviousPage,
    reset,
    loadMoreRef,
    hasNextPage,
  };
}
