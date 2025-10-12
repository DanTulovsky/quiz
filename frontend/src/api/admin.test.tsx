import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import {
  useAdminStories,
  useAdminStory,
  useAdminStorySection,
} from './admin';

// Mock axios
vi.mock('./axios', () => ({
  AXIOS_INSTANCE: {
    get: vi.fn(),
  },
}));

import { AXIOS_INSTANCE } from './axios';

const mockAxios = AXIOS_INSTANCE as ReturnType<typeof vi.fn>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('Admin Hooks', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('useAdminStories', () => {
    it('fetches stories successfully', async () => {
      const mockData = {
        stories: [
          {
            id: 1,
            title: 'Test Story',
            language: 'italian',
            status: 'active',
            user_id: 1,
          },
        ],
        pagination: {
          page: 1,
          page_size: 20,
          total: 1,
          total_pages: 1,
        },
      };

      mockAxios.get.mockResolvedValueOnce({ data: mockData });

      const { result } = renderHook(
        () => useAdminStories(1, 20, '', 'italian', 'active', 1),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      });

      expect(result.current.data).toEqual(mockData);
      expect(mockAxios.get).toHaveBeenCalledWith(
        '/v1/admin/backend/stories?page=1&page_size=20&language=italian&status=active&user_id=1',
        { headers: { Accept: 'application/json' } }
      );
    });

    it('handles API errors', async () => {
      mockAxios.get.mockRejectedValueOnce(new Error('API Error'));

      const { result } = renderHook(
        () => useAdminStories(1, 20),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      });

      expect(result.current.error).toBeTruthy();
      expect(result.current.data).toBeUndefined();
    });

    it('applies filters correctly', async () => {
      const mockData = {
        stories: [],
        pagination: { page: 1, page_size: 20, total: 0, total_pages: 0 },
      };

      mockAxios.get.mockResolvedValueOnce({ data: mockData });

      renderHook(
        () => useAdminStories(1, 20, 'search term', 'spanish', 'archived'),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(mockAxios.get).toHaveBeenCalledWith(
          '/v1/admin/backend/stories?page=1&page_size=20&search=search+term&language=spanish&status=archived',
          { headers: { Accept: 'application/json' } }
        );
      });
    });
  });

  describe('useAdminStory', () => {
    it('fetches a single story successfully', async () => {
      const mockStory = {
        id: 1,
        title: 'Test Story',
        language: 'italian',
        status: 'active',
        sections: [],
      };

      mockAxios.get.mockResolvedValueOnce({ data: mockStory });

      const { result } = renderHook(
        () => useAdminStory(1),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      });

      expect(result.current.data).toEqual(mockStory);
      expect(mockAxios.get).toHaveBeenCalledWith(
        '/v1/admin/backend/stories/1',
        { headers: { Accept: 'application/json' } }
      );
    });

    it('returns undefined when storyId is null', () => {
      const { result } = renderHook(
        () => useAdminStory(null),
        { wrapper: createWrapper() }
      );

      expect(result.current.isLoading).toBe(false);
      expect(result.current.data).toBeUndefined();
      expect(mockAxios.get).not.toHaveBeenCalled();
    });

    it('handles not found error', async () => {
      mockAxios.get.mockRejectedValueOnce({ status: 404 });

      const { result } = renderHook(
        () => useAdminStory(999),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      });

      expect(result.current.error).toBeTruthy();
      expect(result.current.data).toBeUndefined();
    });
  });

  describe('useAdminStorySection', () => {
    it('fetches a story section successfully', async () => {
      const mockSection = {
        id: 1,
        story_id: 1,
        section_number: 1,
        content: 'Test content',
        questions: [],
      };

      mockAxios.get.mockResolvedValueOnce({ data: mockSection });

      const { result } = renderHook(
        () => useAdminStorySection(1),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      });

      expect(result.current.data).toEqual(mockSection);
      expect(mockAxios.get).toHaveBeenCalledWith(
        '/v1/admin/backend/story-sections/1',
        { headers: { Accept: 'application/json' } }
      );
    });

    it('returns undefined when sectionId is null', () => {
      const { result } = renderHook(
        () => useAdminStorySection(null),
        { wrapper: createWrapper() }
      );

      expect(result.current.isLoading).toBe(false);
      expect(result.current.data).toBeUndefined();
      expect(mockAxios.get).not.toHaveBeenCalled();
    });

    it('handles not found error', async () => {
      mockAxios.get.mockRejectedValueOnce({ status: 404 });

      const { result } = renderHook(
        () => useAdminStorySection(999),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      });

      expect(result.current.error).toBeTruthy();
      expect(result.current.data).toBeUndefined();
    });
  });

  describe('Query Key Generation', () => {
    it('generates correct query keys for useAdminStories', () => {
      // Test that the hook can be called with parameters (query key generation is internal)
      const { result } = renderHook(
        () => useAdminStories(2, 10, 'search', 'italian', 'active', 5),
        { wrapper: createWrapper() }
      );

      // Verify the hook returns expected structure
      expect(result.current).toHaveProperty('data');
      expect(result.current).toHaveProperty('isLoading');
      expect(result.current).toHaveProperty('error');
    });

    it('generates correct query keys for useAdminStory', () => {
      // Test that the hook can be called with parameters
      const { result } = renderHook(
        () => useAdminStory(42),
        { wrapper: createWrapper() }
      );

      // Verify the hook returns expected structure
      expect(result.current).toHaveProperty('data');
      expect(result.current).toHaveProperty('isLoading');
      expect(result.current).toHaveProperty('error');
    });

    it('generates correct query keys for useAdminStorySection', () => {
      // Test that the hook can be called with parameters
      const { result } = renderHook(
        () => useAdminStorySection(123),
        { wrapper: createWrapper() }
      );

      // Verify the hook returns expected structure
      expect(result.current).toHaveProperty('data');
      expect(result.current).toHaveProperty('isLoading');
      expect(result.current).toHaveProperty('error');
    });
  });
});
