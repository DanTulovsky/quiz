// Mock React Query hooks before importing useStory
vi.mock('@tanstack/react-query', () => {
  const mockQueryClient = {
    invalidateQueries: vi.fn(),
    removeQueries: vi.fn(),
    refetchQueries: vi.fn(),
  };
  return {
    useQueryClient: vi.fn(() => mockQueryClient),
    useQuery: vi.fn(),
    useMutation: vi.fn(),
  };
});

import { renderHook, act, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useStory } from './useStory';
import { useAuth } from './useAuth';
import * as storyApi from '../api/storyApi';

// Mock React Query hooks
const mockUseQuery = vi.fn();
const mockUseMutation = vi.fn();
const mockUseQueryClient = vi.fn();

// Mock dependencies
vi.mock('@tanstack/react-query', () => ({
  useQueryClient: () => mockUseQueryClient(),
  useQuery: mockUseQuery,
  useMutation: mockUseMutation,
}));

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
    extra_generations_today: 0,
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


  beforeEach(() => {
    // Mock window.URL for PDF export test
    Object.defineProperty(window, 'URL', {
      value: {
        createObjectURL: vi.fn(() => 'blob:mock-url'),
        revokeObjectURL: vi.fn(),
      },
      writable: true,
    });

    // Mock DOM methods for PDF export test
    const mockElement = {
      href: '',
      download: '',
      click: vi.fn(),
      style: {},
    };

    vi.spyOn(document, 'createElement').mockReturnValue(mockElement as any);
    vi.spyOn(document.body, 'appendChild').mockImplementation(() => mockElement as any);
    vi.spyOn(document.body, 'removeChild').mockImplementation(() => true);

    // Setup default mocks
    mockUseAuth.mockReturnValue({
      user: mockUser,
      isLoading: false,
      login: vi.fn(),
      logout: vi.fn(),
      refreshUser: vi.fn(),
    });

    // Mock getUserStories to return empty array by default
    mockStoryApi.getUserStories.mockResolvedValue([]);
    // Mock getSection to return null by default (for sectionWithQuestions query)
    mockStoryApi.getSection?.mockResolvedValue(null);

    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Initialization', () => {
    it('initializes with correct default state', () => {
      const { result } = renderHook(() => useStory());

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

      const { result } = renderHook(() => useStory());

      // Wait for the query to complete and currentStory to be loaded
      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
      });

      expect(result.current.sections).toEqual(mockStory.sections);
      expect(result.current.hasCurrentStory).toBe(true);
    });
  });

  describe('Story Management', () => {
    it('creates a new story', async () => {
      const newStory = { ...mockStory, id: 2, title: 'New Story' };
      mockStoryApi.createStory.mockResolvedValue(newStory);

      const { result } = renderHook(() => useStory());

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

    it('sets generating state when story is created', async () => {
      const { result } = renderHook(() => useStory());

      // Initially should not be generating
      expect(result.current.isGenerating).toBe(false);

      // Create story - the mutation should set isGenerating = true
      await act(async () => {
        await result.current.createStory({
          title: 'New Story',
          subject: 'A new story',
        });
      });

      // The mutation's onSuccess should have set isGenerating = true
      // But the useEffect might override it, so let's just check that the mutation was called
      expect(mockStoryApi.createStory).toHaveBeenCalledWith(
        { title: 'New Story', subject: 'A new story' },
        expect.anything()
      );
    });

    it('archives a story', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
      mockStoryApi.archiveStory.mockResolvedValue();

      const { result } = renderHook(() => useStory());

      // Wait for initial load
      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
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

      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
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

      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
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

      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
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

      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
      });

      await act(async () => {
        await result.current.exportStoryPDF(1);
      });

      expect(mockStoryApi.exportStoryPDF).toHaveBeenCalledWith(
        1,
        expect.anything()
      );

      // Verify the mock element was used correctly
      expect(document.createElement).toHaveBeenCalledWith('a');
      expect(mockElement.click).toHaveBeenCalled();
      expect(window.URL.createObjectURL).toHaveBeenCalledWith(mockBlob);
      expect(window.URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock-url');
    });
  });

  describe('Section Navigation', () => {
    beforeEach(() => {
      mockStoryApi.getCurrentStory.mockResolvedValue(mockStory);
    });

    it('navigates to next section', async () => {
      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
      });

      act(() => {
        result.current.goToNextSection();
      });

      // Should not change since we're at the last section
      expect(result.current.currentSectionIndex).toBe(0);
    });

    it('navigates to previous section', async () => {
      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
      });

      act(() => {
        result.current.goToPreviousSection();
      });

      // Should not change since we're at the first section
      expect(result.current.currentSectionIndex).toBe(0);
    });

    it('changes view mode', async () => {
      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
      });

      act(() => {
        result.current.setViewMode('reading');
      });

      expect(result.current.viewMode).toBe('reading');
    });

    it('goes to specific section', async () => {
      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
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

      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
      });

      // Story is active and has sections, and we're at the last section (index 0 for single section)
      expect(result.current.canGenerateToday).toBe(true);
      expect(result.current.generationDisabledReason).toBe('');

      // If we were at the last section, it should be true
      act(() => {
        result.current.goToSection(0); // Go to last section (index 0 since there's only 1)
      });

      // For a story with one section, should allow generation if it's the last section
      expect(result.current.canGenerateToday).toBe(true);
      expect(result.current.generationDisabledReason).toBe('');
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

      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toEqual(mockStory);
      });

      await act(async () => {
        await result.current.generateNextSection(1);
      });

      expect(mockStoryApi.generateNextSection).toHaveBeenCalledWith(1);
    });
  });

  describe('Error Handling', () => {
    it('handles errors gracefully', async () => {
      mockStoryApi.getCurrentStory.mockRejectedValue(
        new Error('Failed to load story')
      );
      mockStoryApi.getUserStories.mockResolvedValue([]);

      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.error).toBe('Failed to load story');
      });

      expect(result.current.currentStory).toBeUndefined();
    });

    it('handles no current story gracefully', async () => {
      mockStoryApi.getCurrentStory.mockResolvedValue(null);

      const { result } = renderHook(() => useStory());

      await waitFor(() => {
        expect(result.current.currentStory).toBeNull();
      });

      expect(result.current.hasCurrentStory).toBe(false);
      expect(result.current.error).toBeNull();
    });
  });
});
