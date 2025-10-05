import { useState, useEffect, useCallback } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

import { useAuth } from './useAuth';
import {
  createStory as apiCreateStory,
  getCurrentStory as apiGetCurrentStory,
  getUserStories as apiGetUserStories,
  getSection as apiGetSection,
  generateNextSection as apiGenerateNextSection,
  archiveStory as apiArchiveStory,
  completeStory as apiCompleteStory,
  setCurrentStory as apiSetCurrentStory,
  deleteStory as apiDeleteStory,
  exportStoryPDF as apiExportStoryPDF,
  StoryWithSections,
  StorySectionWithQuestions,
  StorySection,
  CreateStoryRequest,
  Story,
} from '../api/storyApi';
import { showNotificationWithClean } from '../notifications';
import logger from '../utils/logger';

export type ViewMode = 'section' | 'reading';

export interface UseStoryReturn {
  // State
  currentStory: StoryWithSections | null;
  archivedStories: Story[] | undefined;
  sections: StorySection[];
  currentSectionIndex: number;
  viewMode: ViewMode;
  isLoading: boolean;
  isLoadingArchivedStories: boolean;
  error: string | null;

  // Actions
  createStory: (data: CreateStoryRequest) => Promise<void>;
  archiveStory: (storyId: number) => Promise<void>;
  completeStory: (storyId: number) => Promise<void>;
  setCurrentStory: (storyId: number) => Promise<void>;
  generateNextSection: (storyId: number) => Promise<void>;
  deleteStory: (storyId: number) => Promise<void>;
  exportStoryPDF: (storyId: number) => Promise<void>;

  // Navigation
  goToSection: (index: number) => void;
  goToNextSection: () => void;
  goToPreviousSection: () => void;
  setViewMode: (mode: ViewMode) => void;

  // Computed
  canGenerateToday: boolean;
  hasCurrentStory: boolean;
  currentSection: StorySection | null;
  currentSectionWithQuestions: StorySectionWithQuestions | null;
  isGenerating: boolean;
}

export const useStory = (): UseStoryReturn => {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  // State
  const [currentSectionIndex, setCurrentSectionIndex] = useState(0);
  const [viewMode, setViewMode] = useState<ViewMode>('section');
  const [error, setError] = useState<string | null>(null);

  // Queries
  const {
    data: currentStory,
    isLoading: isLoadingCurrentStory,
    error: currentStoryError,
  } = useQuery({
    queryKey: ['currentStory', user?.id, user?.preferred_language],
    queryFn: apiGetCurrentStory,
    enabled: !!user?.id,
    retry: false, // Don't retry 404s
  });

  const { data: archivedStories, isLoading: isLoadingArchivedStories } =
    useQuery({
      queryKey: ['archivedStories', user?.id, user?.preferred_language],
      queryFn: () => apiGetUserStories(true), // includeArchived = true
      enabled: !!user?.id && !currentStory, // Only fetch if no current story
    });

  // Mutations
  const createStoryMutation = useMutation({
    mutationFn: apiCreateStory,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({ queryKey: ['userStories'] });
      showNotificationWithClean({
        title: 'Story Created',
        message: 'Your story has been created successfully!',
        type: 'success',
      });
    },
    onError: (error: Error) => {
      logger.error('Failed to create story', error);
      showNotificationWithClean({
        title: 'Error',
        message: error?.message || 'Failed to create story. Please try again.',
        type: 'error',
      });
    },
  });

  const archiveStoryMutation = useMutation({
    mutationFn: apiArchiveStory,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({
        queryKey: ['archivedStories', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({ queryKey: ['userStories'] });
      setCurrentSectionIndex(0);
      setViewMode('section');
      showNotificationWithClean({
        title: 'Story Archived',
        message: 'Your story has been archived successfully.',
        type: 'success',
      });
    },
    onError: (error: Error) => {
      logger.error('Failed to archive story', error);
      showNotificationWithClean({
        title: 'Error',
        message: error?.message || 'Failed to archive story. Please try again.',
        type: 'error',
      });
    },
  });

  const completeStoryMutation = useMutation({
    mutationFn: apiCompleteStory,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({ queryKey: ['userStories'] });
      showNotificationWithClean({
        title: 'Story Completed',
        message: 'Your story has been marked as completed!',
        type: 'success',
      });
    },
    onError: (error: Error) => {
      logger.error('Failed to complete story', error);
      showNotificationWithClean({
        title: 'Error',
        message:
          error?.message || 'Failed to complete story. Please try again.',
        type: 'error',
      });
    },
  });

  const setCurrentStoryMutation = useMutation({
    mutationFn: apiSetCurrentStory,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id],
      });
      queryClient.invalidateQueries({ queryKey: ['userStories'] });
      setCurrentSectionIndex(0);
      setViewMode('section');
      showNotificationWithClean({
        title: 'Story Activated',
        message: 'Story has been set as your current active story.',
        type: 'success',
      });
    },
    onError: (error: Error) => {
      logger.error('Failed to set current story', error);
      showNotificationWithClean({
        title: 'Error',
        message:
          error?.message || 'Failed to set current story. Please try again.',
        type: 'error',
      });
    },
  });

  const generateNextSectionMutation = useMutation({
    mutationFn: apiGenerateNextSection,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({
        queryKey: ['sectionWithQuestions', currentSection?.id],
      });
      // Go to the new section
      if (currentStory) {
        setCurrentSectionIndex(currentStory.sections.length);
      }
      showNotificationWithClean({
        title: 'Section Generated',
        message: 'A new section has been added to your story!',
        type: 'success',
      });
    },
    onError: (error: Error) => {
      logger.error('Failed to generate next section', error);
      showNotificationWithClean({
        title: 'Error',
        message:
          error?.message ||
          'Failed to generate next section. Please try again.',
        type: 'error',
      });
    },
  });

  const deleteStoryMutation = useMutation({
    mutationFn: apiDeleteStory,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({ queryKey: ['userStories'] });
      showNotificationWithClean({
        title: 'Story Deleted',
        message: 'Story has been deleted successfully.',
        type: 'success',
      });
    },
    onError: (error: Error) => {
      logger.error('Failed to delete story', error);
      showNotificationWithClean({
        title: 'Error',
        message: error?.message || 'Failed to delete story. Please try again.',
        type: 'error',
      });
    },
  });

  const exportStoryPDFMutation = useMutation({
    mutationFn: apiExportStoryPDF,
    onSuccess: blob => {
      // Create download link
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `story_${currentStory?.title || 'export'}.pdf`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      showNotificationWithClean({
        title: 'Export Complete',
        message: 'Your story has been exported as PDF.',
        type: 'success',
      });
    },
    onError: (error: Error) => {
      logger.error('Failed to export story PDF', error);
      showNotificationWithClean({
        title: 'Error',
        message: error?.message || 'Failed to export story. Please try again.',
        type: 'error',
      });
    },
  });

  // Computed values
  const sections = currentStory?.sections || [];
  const hasCurrentStory = !!currentStory;
  const currentSection = sections[currentSectionIndex] || null;

  // Check if generation is allowed today (simplified logic)
  const canGenerateToday =
    hasCurrentStory &&
    currentStory.status === 'active' &&
    (sections.length === 0 || currentSectionIndex === sections.length - 1);

  // Query for current section with questions
  const { data: currentSectionWithQuestions } = useQuery({
    queryKey: ['sectionWithQuestions', currentSection?.id],
    queryFn: () => {
      if (!currentSection?.id) return null;
      return apiGetSection(currentSection.id);
    },
    enabled: !!currentSection?.id && !!currentStory,
  });

  // Error handling
  useEffect(() => {
    if (currentStoryError) {
      const errorMessage =
        currentStoryError instanceof Error
          ? currentStoryError.message
          : 'Failed to load current story';
      setError(errorMessage);
    } else {
      setError(null);
    }
  }, [currentStoryError]);

  // Reset section index when story changes
  useEffect(() => {
    if (currentStory) {
      setCurrentSectionIndex(0);
    }
  }, [currentStory?.id]);

  // Action handlers
  const createStory = useCallback(
    async (data: CreateStoryRequest) => {
      await createStoryMutation.mutateAsync(data);
    },
    [createStoryMutation]
  );

  const archiveStory = useCallback(
    async (storyId: number) => {
      await archiveStoryMutation.mutateAsync(storyId);
    },
    [archiveStoryMutation]
  );

  const completeStory = useCallback(
    async (storyId: number) => {
      await completeStoryMutation.mutateAsync(storyId);
    },
    [completeStoryMutation]
  );

  const setCurrentStoryAction = useCallback(
    async (storyId: number) => {
      await setCurrentStoryMutation.mutateAsync(storyId);
    },
    [setCurrentStoryMutation]
  );

  const generateNextSection = useCallback(
    async (storyId: number) => {
      await generateNextSectionMutation.mutateAsync(storyId);
    },
    [generateNextSectionMutation]
  );

  const deleteStoryAction = useCallback(
    async (storyId: number) => {
      await deleteStoryMutation.mutateAsync(storyId);
    },
    [deleteStoryMutation]
  );

  const exportStoryPDFAction = useCallback(
    async (storyId: number) => {
      await exportStoryPDFMutation.mutateAsync(storyId);
    },
    [exportStoryPDFMutation]
  );

  const goToSection = useCallback(
    (index: number) => {
      if (index >= 0 && index < sections.length) {
        setCurrentSectionIndex(index);
      }
    },
    [sections.length]
  );

  const goToNextSection = useCallback(() => {
    if (currentSectionIndex < sections.length - 1) {
      setCurrentSectionIndex(currentSectionIndex + 1);
    }
  }, [currentSectionIndex, sections.length]);

  const goToPreviousSection = useCallback(() => {
    if (currentSectionIndex > 0) {
      setCurrentSectionIndex(currentSectionIndex - 1);
    }
  }, [currentSectionIndex]);

  const setViewModeAction = useCallback((mode: ViewMode) => {
    setViewMode(mode);
  }, []);

  return {
    // State
    currentStory,
    archivedStories,
    sections,
    currentSectionIndex,
    viewMode,
    isLoading: isLoadingCurrentStory || isLoadingArchivedStories,
    isLoadingArchivedStories,
    error,

    // Actions
    createStory,
    archiveStory,
    completeStory,
    setCurrentStory: setCurrentStoryAction,
    generateNextSection,
    deleteStory: deleteStoryAction,
    exportStoryPDF: exportStoryPDFAction,

    // Navigation
    goToSection,
    goToNextSection,
    goToPreviousSection,
    setViewMode: setViewModeAction,

    // Computed
    canGenerateToday,
    hasCurrentStory,
    currentSection,
    currentSectionWithQuestions,
    isGenerating: generateNextSectionMutation.isPending,
  };
};
