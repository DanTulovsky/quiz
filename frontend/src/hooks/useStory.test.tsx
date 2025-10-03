import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { useStory } from './useStory';
import { useAuth } from './useAuth';
import * as storyApi from '../api/storyApi';

// Mock dependencies
vi.mock('./useAuth');
vi.mock('../api/storyApi');
vi.mock('../notifications', () => ({
  showNotificationWithClean: vi.fn(),
}));
vi.mock('../utils/logger', () => ({
  default: {
    error: vi.fn(),
  },
  error: vi.fn(),
}));

const mockUseAuth = useAuth as vi.MockedFunction<typeof useAuth>;
const mockStoryApi = storyApi as vi.Mocked<typeof storyApi>;

describe('useStory', () => {
  let queryClient: QueryClient;

  const mockUser = {
    id: 1,
    username: 'testuser',
    language: 'en',
  };

  const mockStory = {
    id: 1,
    title: 'Test Story',
    language: 'en',
    status: 'active' as const,
    is_current: true,
    sections: [
      {
        id: 1,
        section_number: 1,
        content: 'This is the first section.',
        language_level: 'intermediate',
        word_count: 6,
        generated_at: '2025-01-15T10:30:00Z',
        generation_date: '2025-01-15',
      },
    ],
  };

  const wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });

    // Mock window.URL for PDF export test
    Object.defineProperty(window, 'URL', {
      value: {
        createObjectURL: vi.fn(() => 'blob:mock-url'),
        revokeObjectURL: vi.fn(),
      },
      writable: true,
    });

    vi.clearAllMocks();

    mockUseAuth.mockReturnValue({
      user: mockUser,
      isLoading: false,
      login: vi.fn(),
      logout: vi.fn(),
      refreshUser: vi.fn(),
    });
  });

  describe('Initialization', () => {
    it('initializes with correct default state', () => {
      const { result } = renderHook(() => useStory(), { wrapper });

      expect(result.current.currentStory).toBeUndefined();
      expect(result.current.sections).toEqual([]);
      expect(result.current.currentSectionIndex).toBe(0);
      expect(result.current.viewMode).toBe('section');
      expect(result.current.isLoading).toBe(true);
      expect(result.current.error).toBeNull();
      expect(result.current.hasCurrentStory).toBe(false);
    });

    it('loads current story when user is authenticated', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);

      const { result } = renderHook(() => useStory(), { wrapper });

      // Wait for the query to complete and currentStory to be loaded
      await new Promise(resolve => setTimeout(resolve, 100));

      expect(result.current.currentStory).toEqual(mockStory);
      expect(result.current.sections).toEqual(mockStory.sections);
      expect(result.current.hasCurrentStory).toBe(true);
    });
  });

  describe('Story Management', () => {
    it('creates a new story', async () => {
      const newStory = { ...mockStory, id: 2, title: 'New Story' };
      mockStoryApi.createStory.mockResolvedValue(newStory);

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await result.current.createStory({
          title: 'New Story',
          subject: 'A new story',
        });
      });

      expect(mockStoryApi.createStory).toHaveBeenCalledWith(
        { title: 'New Story', subject: 'A new story' },
        expect.anything()
      );
    });

    it('archives a story', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
      mockStoryApi.archiveStory.mockResolvedValue();

      const { result } = renderHook(() => useStory(), { wrapper });

      // Wait for initial load
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      await act(async () => {
        await result.current.archiveStory(1);
      });

      expect(mockStoryApi.archiveStory).toHaveBeenCalledWith(
        1,
        expect.anything()
      );
    });

    it('completes a story', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
      mockStoryApi.completeStory.mockResolvedValue();

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      await act(async () => {
        await result.current.completeStory(1);
      });

      expect(mockStoryApi.completeStory).toHaveBeenCalledWith(
        1,
        expect.anything()
      );
    });

    it('sets current story', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
      mockStoryApi.setCurrentStory.mockResolvedValue();

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      await act(async () => {
        await result.current.setCurrentStory(1);
      });

      expect(mockStoryApi.setCurrentStory).toHaveBeenCalledWith(
        1,
        expect.anything()
      );
    });

    it('deletes a story', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
      mockStoryApi.deleteStory.mockResolvedValue();

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      await act(async () => {
        await result.current.deleteStory(1);
      });

      expect(mockStoryApi.deleteStory).toHaveBeenCalledWith(
        1,
        expect.anything()
      );
    });

    it('exports story as PDF', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
      const mockBlob = new Blob(['pdf content'], { type: 'application/pdf' });
      mockStoryApi.exportStoryPDF.mockResolvedValue(mockBlob);

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      await act(async () => {
        await result.current.exportStoryPDF(1);
      });

      expect(mockStoryApi.exportStoryPDF).toHaveBeenCalledWith(
        1,
        expect.anything()
      );
    });
  });

  describe('Section Navigation', () => {
    beforeEach(() => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
    });

    it('navigates to next section', async () => {
      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      act(() => {
        result.current.goToNextSection();
      });

      // Should not change since we're at the last section
      expect(result.current.currentSectionIndex).toBe(0);
    });

    it('navigates to previous section', async () => {
      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      act(() => {
        result.current.goToPreviousSection();
      });

      // Should not change since we're at the first section
      expect(result.current.currentSectionIndex).toBe(0);
    });

    it('changes view mode', async () => {
      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      act(() => {
        result.current.setViewMode('reading');
      });

      expect(result.current.viewMode).toBe('reading');
    });

    it('goes to specific section', async () => {
      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      act(() => {
        result.current.goToSection(0);
      });

      expect(result.current.currentSectionIndex).toBe(0);
    });
  });

  describe('Generation Logic', () => {
    it('determines if generation is allowed today', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      // Story is active and has sections, and we're at the last section (index 0 for single section)
      expect(result.current.canGenerateToday).toBe(true);

      // If we were at the last section, it should be true
      act(() => {
        result.current.goToSection(0); // Go to last section (index 0 since there's only 1)
      });

      // For a story with one section, should allow generation if it's the last section
      expect(result.current.canGenerateToday).toBe(true);
    });

    it('generates next section', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
      mockStoryApi.generateNextSection.mockResolvedValue({
        id: 2,
        section_number: 2,
        content: 'This is the second section.',
        language_level: 'intermediate',
        word_count: 6,
        generated_at: '2025-01-15T11:30:00Z',
        generation_date: '2025-01-15',
      });

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      await act(async () => {
        await result.current.generateNextSection(1);
      });

      expect(mockStoryApi.generateNextSection).toHaveBeenCalledWith(
        1,
        expect.anything()
      );
    });
  });

  describe('Error Handling', () => {
    it('handles errors gracefully', async () => {
      mockStoryApi.getCurrentStory.mockRejectedValue(
        new Error('Failed to load story')
      );

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      expect(result.current.error).toBe('Failed to load story');
      expect(result.current.currentStory).toBeUndefined();
    });

    it('handles no current story gracefully', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(null);

      const { result } = renderHook(() => useStory(), { wrapper });

      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      expect(result.current.currentStory).toBeNull();
      expect(result.current.hasCurrentStory).toBe(false);
      expect(result.current.error).toBeNull();
    });
  });
});
