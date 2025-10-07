import { useState, useEffect, useCallback, useRef } from 'react';
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
  isGenerating: boolean;

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
  const [isGenerating, setIsGenerating] = useState(false);
  const [generationErrorModal, setGenerationErrorModal] = useState<{
    isOpen: boolean;
    errorMessage: string;
  }>({ isOpen: false, errorMessage: '' });

  // Polling
  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const currentStoryRef = useRef<StoryWithSections | null>(null);

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

  // Handle generating status and polling
  useEffect(() => {
    currentStoryRef.current = currentStory;

    // Check if we received a generating response or if story has no sections yet
    const isGeneratingState =
      (currentStory &&
        typeof currentStory === 'object' &&
        'status' in currentStory &&
        currentStory.status === 'generating') ||
      (currentStory &&
        typeof currentStory === 'object' &&
        'sections' in currentStory &&
        (!currentStory.sections || currentStory.sections.length === 0));

    if (isGeneratingState) {
      setIsGenerating(true);
      // Clear any previous errors since we're in generating state
      setError(null);
      // Don't set as error - this is informational
      startPolling();
    } else if (
      currentStory &&
      'sections' in currentStory &&
      currentStory.sections &&
      currentStory.sections.length > 0
    ) {
      // Only stop generating if we have a story with actual sections
      setIsGenerating(false);
      stopPolling();
      setError(null);
    }
    // If currentStory is null or undefined, don't change the generating state
    // (it might be in the process of being fetched after story creation)
  }, [currentStory, error]);

  // Polling functions
  const stopPolling = useCallback(() => {
    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
  }, []);

  const startPolling = useCallback(() => {
    stopPolling(); // Clear any existing interval

    pollingIntervalRef.current = setInterval(async () => {
      try {
        // Only poll if we have a user
        if (user) {
          queryClient.invalidateQueries({
            queryKey: ['currentStory', user.id, user.preferred_language],
          });
        }
      } catch (error) {
        console.error('Polling error:', error);
        // swallow; next tick will retry
      }
    }, 3000); // Poll every 3 seconds
  }, [user, queryClient]);

  // Cleanup polling on unmount
  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  const { data: archivedStories, isLoading: isLoadingArchivedStories } =
    useQuery({
      queryKey: ['archivedStories', user?.id, user?.preferred_language],
      queryFn: () => apiGetUserStories(true), // includeArchived = true
      enabled: !!user?.id, // Always fetch if user exists
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
      // Start polling for the first section
      setIsGenerating(true);
      startPolling();
    },
    onError: (error: unknown) => {
      logger.error('Failed to create story', error);
      let errorMessage = 'Failed to create story. Please try again.';

      if (error instanceof Error) {
        errorMessage = error.message;
      } else if (typeof error === 'object' && error !== null) {
        // Check if error has response structure (axios-like error)
        const hasResponse = 'response' in error || 'message' in error;
        if (hasResponse) {
          const axiosError = error as {
            response?: { data?: { error?: string }; status?: number };
            message?: string;
          };

          if (axiosError.response?.data?.error) {
            errorMessage = axiosError.response.data.error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'error' in axiosError.response.data
          ) {
            // Handle case where response.data is the error object itself
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else {
          // If error doesn't have response structure, convert to string
          errorMessage = String(error);
        }
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
        type: 'error',
      });
    },
  });

  const archiveStoryMutation = useMutation({
    mutationFn: apiArchiveStory,
    onSuccess: () => {
      // Remove current story from cache to force immediate UI update
      queryClient.removeQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({
        queryKey: ['archivedStories', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({ queryKey: ['userStories'] });

      // Also refetch archived stories immediately
      queryClient.refetchQueries({
        queryKey: ['archivedStories', user?.id, user?.preferred_language],
      });
      setCurrentSectionIndex(0);
      setViewMode('section');
      showNotificationWithClean({
        title: 'Story Archived',
        message: 'Your story has been archived successfully.',
        type: 'success',
      });
    },
    onError: (error: unknown) => {
      logger.error('Failed to archive story', error);
      let errorMessage = 'Failed to archive story. Please try again.';

      if (error instanceof Error) {
        errorMessage = error.message;
      } else if (typeof error === 'object' && error !== null) {
        // Check if error has response structure (axios-like error)
        const hasResponse = 'response' in error || 'message' in error;
        if (hasResponse) {
          const axiosError = error as {
            response?: { data?: { error?: string }; status?: number };
            message?: string;
          };

          if (axiosError.response?.data?.error) {
            errorMessage = axiosError.response.data.error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'error' in axiosError.response.data
          ) {
            // Handle case where response.data is the error object itself
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else {
          // If error doesn't have response structure, convert to string
          errorMessage = String(error);
        }
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
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
    onError: (error: unknown) => {
      logger.error('Failed to complete story', error);
      let errorMessage = 'Failed to complete story. Please try again.';

      if (error instanceof Error) {
        errorMessage = error.message;
      } else if (typeof error === 'object' && error !== null) {
        // Check if error has response structure (axios-like error)
        const hasResponse = 'response' in error || 'message' in error;
        if (hasResponse) {
          const axiosError = error as {
            response?: { data?: { error?: string }; status?: number };
            message?: string;
          };

          if (axiosError.response?.data?.error) {
            errorMessage = axiosError.response.data.error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'error' in axiosError.response.data
          ) {
            // Handle case where response.data is the error object itself
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else {
          // If error doesn't have response structure, convert to string
          errorMessage = String(error);
        }
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
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
    onError: (error: unknown) => {
      logger.error('Failed to set current story', error);
      let errorMessage = 'Failed to set current story. Please try again.';

      if (error instanceof Error) {
        errorMessage = error.message;
      } else if (typeof error === 'object' && error !== null) {
        // Check if error has response structure (axios-like error)
        const hasResponse = 'response' in error || 'message' in error;
        if (hasResponse) {
          const axiosError = error as {
            response?: { data?: { error?: string }; status?: number };
            message?: string;
          };

          if (axiosError.response?.data?.error) {
            errorMessage = axiosError.response.data.error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'error' in axiosError.response.data
          ) {
            // Handle case where response.data is the error object itself
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else {
          // If error doesn't have response structure, convert to string
          errorMessage = String(error);
        }
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
        type: 'error',
      });
    },
  });

  const generateNextSectionMutation = useMutation({
    mutationFn: apiGenerateNextSection,
    onMutate: () => {
      // Set generating state when mutation starts
      setIsGenerating(true);
    },
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
      // Stop generating state on success
      setIsGenerating(false);
    },
    onError: (error: unknown) => {
      logger.error('Failed to generate next section', error);

      let errorMessage = 'Failed to generate next section. Please try again.';

      if (error instanceof Error) {
        errorMessage = error.message;
      } else if (typeof error === 'object' && error !== null) {
        // Check if error has response structure (axios-like error)
        const hasResponse = 'response' in error || 'message' in error;
        if (hasResponse) {
          const axiosError = error as {
            response?: { data?: { error?: string }; status?: number };
            message?: string;
          };

          if (axiosError.response?.data?.error) {
            errorMessage = axiosError.response.data.error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'error' in axiosError.response.data
          ) {
            // Handle case where response.data is the error object itself
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else {
          // If error doesn't have response structure, convert to string
          errorMessage = String(error);
        }
      }

      // Show error modal for all generation errors
      setGenerationErrorModal({
        isOpen: true,
        errorMessage,
      });
      // Stop generating state on error
      setIsGenerating(false);
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
    onError: (error: unknown) => {
      logger.error('Failed to delete story', error);
      let errorMessage = 'Failed to delete story. Please try again.';

      if (error instanceof Error) {
        errorMessage = error.message;
      } else if (typeof error === 'object' && error !== null) {
        // Check if error has response structure (axios-like error)
        const hasResponse = 'response' in error || 'message' in error;
        if (hasResponse) {
          const axiosError = error as {
            response?: { data?: { error?: string }; status?: number };
            message?: string;
          };

          if (axiosError.response?.data?.error) {
            errorMessage = axiosError.response.data.error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'error' in axiosError.response.data
          ) {
            // Handle case where response.data is the error object itself
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else {
          // If error doesn't have response structure, convert to string
          errorMessage = String(error);
        }
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
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
    onError: (error: unknown) => {
      logger.error('Failed to export story PDF', error);
      let errorMessage = 'Failed to export story. Please try again.';

      if (error instanceof Error) {
        errorMessage = error.message;
      } else if (typeof error === 'object' && error !== null) {
        // Check if error has response structure (axios-like error)
        const hasResponse = 'response' in error || 'message' in error;
        if (hasResponse) {
          const axiosError = error as {
            response?: { data?: { error?: string }; status?: number };
            message?: string;
          };

          if (axiosError.response?.data?.error) {
            errorMessage = axiosError.response.data.error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'error' in axiosError.response.data
          ) {
            // Handle case where response.data is the error object itself
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else {
          // If error doesn't have response structure, convert to string
          errorMessage = String(error);
        }
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
        type: 'error',
      });
    },
  });

  // Computed values
  const sections = currentStory?.sections || [];
  const hasCurrentStory = !!currentStory;
  const currentSection = sections[currentSectionIndex] || null;

  // Check if generation is allowed today (basic client-side checks)
  // The backend will do the final validation and return appropriate errors
  const canGenerateToday =
    hasCurrentStory &&
    currentStory.status === 'active' &&
    (sections.length === 0 || currentSectionIndex === sections.length - 1);

  // Get reason why generation might be disabled
  const getGenerationDisabledReason = (): string => {
    if (!hasCurrentStory) {
      return 'No active story';
    }
    if (currentStory.status !== 'active') {
      return 'Story is not active';
    }
    if (sections.length === 0) {
      return 'Ready to generate first section';
    }
    if (currentSectionIndex !== sections.length - 1) {
      return 'Navigate to the latest section to generate the next part';
    }
    // If we reach here, generation should be allowed (backend will validate)
    return '';
  };

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
      let errorMessage = 'Failed to load current story';

      if (currentStoryError instanceof Error) {
        errorMessage = currentStoryError.message;
      } else if (
        typeof currentStoryError === 'object' &&
        currentStoryError !== null
      ) {
        // Handle axios-like error objects
        const axiosError = currentStoryError as {
          response?: { data?: { error?: string } };
          message?: string;
        };
        if (axiosError.response?.data?.error) {
          errorMessage = axiosError.response.data.error;
        } else if (axiosError.message) {
          errorMessage = axiosError.message;
        }
      } else if (typeof currentStoryError === 'string') {
        errorMessage = currentStoryError;
      }

      setError(errorMessage);
    } else {
      setError(null);
    }
  }, [currentStoryError]);

  // Reset section index when story changes - start at last section
  useEffect(() => {
    if (currentStory && currentStory.sections.length > 0) {
      setCurrentSectionIndex(currentStory.sections.length - 1);
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

  const closeGenerationErrorModal = useCallback(() => {
    setGenerationErrorModal({ isOpen: false, errorMessage: '' });
  }, []);

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

  const goToFirstSection = useCallback(() => {
    setCurrentSectionIndex(0);
  }, []);

  const goToLastSection = useCallback(() => {
    if (sections.length > 0) {
      setCurrentSectionIndex(sections.length - 1);
    }
  }, [sections.length]);

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
    isGenerating,

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
    goToFirstSection,
    goToLastSection,
    setViewMode: setViewModeAction,

    // Computed
    canGenerateToday,
    hasCurrentStory,
    currentSection,
    currentSectionWithQuestions,
    isGeneratingNextSection: generateNextSectionMutation.isPending,
    generationDisabledReason: getGenerationDisabledReason(),

    // Modal state
    generationErrorModal: {
      isOpen: generationErrorModal.isOpen,
      errorMessage: generationErrorModal.errorMessage,
    },
    closeGenerationErrorModal,
  };
};
