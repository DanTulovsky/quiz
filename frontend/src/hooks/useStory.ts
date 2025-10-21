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
  toggleAutoGeneration as apiToggleAutoGeneration,
  StoryWithSections,
  StorySectionWithQuestions,
  StorySection,
  CreateStoryRequest,
  Story,
} from '../api/storyApi';
import { showNotificationWithClean } from '../notifications';

// Error type interfaces
interface ErrorWithResponse extends Error {
  response?: {
    data?: unknown;
    status?: number;
  };
}

interface AxiosError {
  response?: {
    data?: unknown;
    status?: number;
  };
  message?: string;
}

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
  generationType: 'story' | 'section' | null;

  // Actions
  createStory: (data: CreateStoryRequest) => Promise<void>;
  archiveStory: (storyId: number) => Promise<void>;
  completeStory: (storyId: number) => Promise<void>;
  setCurrentStory: (storyId: number) => Promise<void>;
  generateNextSection: (storyId: number) => Promise<void>;
  deleteStory: (storyId: number) => Promise<void>;
  exportStoryPDF: (storyId: number) => Promise<void>;
  toggleAutoGeneration: (storyId: number, paused: boolean) => Promise<void>;

  // Navigation
  goToSection: (index: number) => void;
  goToNextSection: () => void;
  goToPreviousSection: () => void;
  goToFirstSection: () => void;
  goToLastSection: () => void;
  setViewMode: (mode: ViewMode) => void;

  // Computed
  canGenerateToday: boolean;
  hasCurrentStory: boolean;
  currentSection: StorySection | null;
  currentSectionWithQuestions: StorySectionWithQuestions | null;
  generationDisabledReason: string;

  // Modal state
  generationErrorModal: {
    isOpen: boolean;
    errorMessage: string;
    errorDetails?: ErrorResponse;
  };
  closeGenerationErrorModal: () => void;
}

export const useStory = (): UseStoryReturn => {
  const { user } = useAuth();
  const queryClient = useQueryClient();

  // Helper to get localStorage key for section index
  const getSectionIndexKey = useCallback((storyId: number) => {
    return `story_section_index_${storyId}`;
  }, []);

  // Helper to load section index from localStorage
  const loadSectionIndex = useCallback((storyId: number): number | null => {
    try {
      const saved = localStorage.getItem(getSectionIndexKey(storyId));
      return saved !== null ? parseInt(saved, 10) : null;
    } catch {
      return null;
    }
  }, [getSectionIndexKey]);

  // Helper to save section index to localStorage
  const saveSectionIndex = useCallback((storyId: number, index: number) => {
    try {
      localStorage.setItem(getSectionIndexKey(storyId), String(index));
    } catch {
      // Ignore localStorage errors
    }
  }, [getSectionIndexKey]);

  // State
  const [currentSectionIndex, setCurrentSectionIndex] = useState(0);
  const [viewMode, setViewMode] = useState<ViewMode>('section');
  const [error, setError] = useState<string | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [generationType, setGenerationType] = useState<
    'story' | 'section' | null
  >(null);
  const [generationErrorModal, setGenerationErrorModal] = useState<{
    isOpen: boolean;
    errorMessage: string;
    errorDetails?: ErrorResponse;
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
      setGenerationType(null);
      stopPolling();
      // Don't clear error here - let the error handling useEffect manage error state
    }
    // If currentStory is null or undefined, don't change the generating state
    // (it might be in the process of being fetched after story creation)
  }, [currentStory]);

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
      setGenerationType('story');
      startPolling();
    },
    onError: (error: unknown) => {
      let errorMessage = 'Failed to create story. Please try again.' + error;

      if (typeof error === 'object' && error !== null) {
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
            // Handle case where response.data is the ErrorResponse structure
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'message' in axiosError.response.data
          ) {
            // Handle case where the error message is in the 'message' field
            errorMessage = (axiosError.response.data as { message: string })
              .message;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else if (error instanceof Error) {
          // If it's an Error but doesn't have response structure, use the message
          errorMessage = error.message;
        } else {
          // If error doesn't have response structure and isn't an Error, convert to string
          errorMessage = String(error);
        }
      } else if (error instanceof Error) {
        errorMessage = error.message;
      } else {
        errorMessage = String(error);
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
        type: 'error',
      });
      setGenerationType(null);
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
      let errorMessage = 'Failed to archive story. Please try again.' + error;

      if (typeof error === 'object' && error !== null) {
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
            // Handle case where response.data is the ErrorResponse structure
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'message' in axiosError.response.data
          ) {
            // Handle case where the error message is in the 'message' field
            errorMessage = (axiosError.response.data as { message: string })
              .message;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else if (error instanceof Error) {
          // If it's an Error but doesn't have response structure, use the message
          errorMessage = error.message;
        } else {
          // If error doesn't have response structure and isn't an Error, convert to string
          errorMessage = String(error);
        }
      } else if (error instanceof Error) {
        errorMessage = error.message;
      } else {
        errorMessage = String(error);
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
      let errorMessage = 'Failed to complete story. Please try again.' + error;

      if (typeof error === 'object' && error !== null) {
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
            // Handle case where response.data is the ErrorResponse structure
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'message' in axiosError.response.data
          ) {
            // Handle case where the error message is in the 'message' field
            errorMessage = (axiosError.response.data as { message: string })
              .message;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else if (error instanceof Error) {
          // If it's an Error but doesn't have response structure, use the message
          errorMessage = error.message;
        } else {
          // If error doesn't have response structure and isn't an Error, convert to string
          errorMessage = String(error);
        }
      } else if (error instanceof Error) {
        errorMessage = error.message;
      } else {
        errorMessage = String(error);
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
      let errorMessage =
        'Failed to set current story. Please try again.' + error;

      if (typeof error === 'object' && error !== null) {
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
            // Handle case where response.data is the ErrorResponse structure
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'message' in axiosError.response.data
          ) {
            // Handle case where the error message is in the 'message' field
            errorMessage = (axiosError.response.data as { message: string })
              .message;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else if (error instanceof Error) {
          // If it's an Error but doesn't have response structure, use the message
          errorMessage = error.message;
        } else {
          // If error doesn't have response structure and isn't an Error, convert to string
          errorMessage = String(error);
        }
      } else if (error instanceof Error) {
        errorMessage = error.message;
      } else {
        errorMessage = String(error);
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
        type: 'error',
      });
    },
  });

  const generateNextSectionMutation = useMutation({
    mutationFn: async (storyId: number) => {
      try {
        const result = await apiGenerateNextSection(storyId);

        // Check if the result indicates an error - this handles the case where
        // the backend returns a 200 response with an error message in the body
        if (result && typeof result === 'object') {
          if ('error' in result && result.error) {
            // Preserve the full error response object for error details extraction
            const errorWithDetails: ErrorWithResponse = new Error(result.error);
            errorWithDetails.response = { data: result };
            throw errorWithDetails;
          }
        }

        return result;
      } catch (error) {
        // Log the error for debugging
        throw error;
      }
    },
    onMutate: () => {
      // Set generating state when mutation starts
      setIsGenerating(true);
      setGenerationType('section');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });

      // Go to the new section first so currentSection is updated
      if (currentStory && currentStory.sections) {
        setCurrentSectionIndex(currentStory.sections.length);
      }

      // Invalidate the sectionWithQuestions query for the new section after state updates
      setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: ['sectionWithQuestions'],
        });
      }, 0);

      showNotificationWithClean({
        title: 'Section Generated',
        message: 'A new section has been added to your story!',
        type: 'success',
      });
      // Stop generating state on success
      setIsGenerating(false);
      setGenerationType(null);
    },
    onError: (error: unknown) => {
      let errorMessage = 'Failed to generate next section. Please try again.';
      let errorDetails: unknown | undefined;

      // First, try to parse the error itself as JSON if it's a string
      if (typeof error === 'string') {
        errorMessage = error;

        // Try to parse as JSON in case it contains structured error data
        try {
          const parsedError = JSON.parse(error);
          if (typeof parsedError === 'object' && parsedError !== null) {
            if (
              'code' in parsedError ||
              'message' in parsedError ||
              'error' in parsedError ||
              'details' in parsedError
            ) {
              errorDetails = parsedError as ErrorResponse;
              errorMessage =
                parsedError.message || parsedError.error || errorMessage;
            }
          }
        } catch {
          // Not JSON, use as plain error message
        }
      }

      // Handle different error formats
      if (error && typeof error === 'object' && 'response' in error) {
        // Handle axios error structure
        const axiosError = error as AxiosError;

        console.log('Axios error structure:', {
          hasResponse: !!axiosError.response,
          responseData: axiosError.response?.data,
          responseStatus: axiosError.response?.status,
          axiosMessage: axiosError.message,
        });

        if (axiosError.response?.data) {
          const responseData = axiosError.response.data;

          if (typeof responseData === 'string') {
            // If it's a string, it might be just an error message, or it might contain JSON
            errorMessage = responseData;

            // Try to parse as JSON in case it contains structured error data
            try {
              const parsedError = JSON.parse(responseData);
              if (typeof parsedError === 'object' && parsedError !== null) {
                console.log('Parsed JSON from string:', parsedError);
                if (
                  'code' in parsedError ||
                  'message' in parsedError ||
                  'error' in parsedError ||
                  'details' in parsedError
                ) {
                  errorDetails = parsedError as ErrorResponse;
                  errorMessage =
                    parsedError.message || parsedError.error || errorMessage;
                  console.log(
                    'Successfully extracted error details from JSON string'
                  );
                }
              }
            } catch (parseError) {
              console.log('Failed to parse response data as JSON:', parseError);
              // Not JSON, use as plain error message
            }
          } else if (
            typeof responseData === 'object' &&
            responseData !== null
          ) {
            console.log(
              'Response data is object with keys:',
              Object.keys(responseData)
            );

            // Check if it has the expected ErrorResponse structure
            if (
              'code' in responseData ||
              'message' in responseData ||
              'error' in responseData ||
              'details' in responseData
            ) {
              errorDetails = responseData as ErrorResponse;
              errorMessage =
                (responseData as ErrorResponse).message ||
                (responseData as ErrorResponse).error ||
                errorMessage;
              console.log('Successfully extracted error details from object');
            } else if ('error' in responseData) {
              // Handle case where response.data.error exists but it's not the full structure
              errorMessage = (responseData as { error: string }).error;
            } else {
              // Fallback: try to extract any meaningful error message
              errorMessage = String(responseData);
            }
          } else {
            errorMessage = String(responseData);
          }
        } else if (axiosError.message) {
          errorMessage = axiosError.message;
        }
      } else if (error instanceof Error) {
        // If it's an Error but doesn't have response structure, use the message
        errorMessage = error.message;

        // Try to parse the error message as JSON in case it contains structured error data
        try {
          const parsedError = JSON.parse(error.message);
          if (typeof parsedError === 'object' && parsedError !== null) {
            if (
              'code' in parsedError ||
              'message' in parsedError ||
              'error' in parsedError ||
              'details' in parsedError
            ) {
              errorDetails = parsedError as ErrorResponse;
              errorMessage =
                parsedError.message || parsedError.error || errorMessage;
              console.log(
                'Successfully extracted error details from Error message'
              );
            }
          }
        } catch {
          // Not JSON, use as plain error message
        }
      } else if (typeof error === 'string') {
        // Handle string errors directly
        errorMessage = error;

        // Try to parse as JSON in case it contains structured error data
        try {
          const parsedError = JSON.parse(error);
          if (typeof parsedError === 'object' && parsedError !== null) {
            console.log('Parsed JSON from error string:', parsedError);
            if (
              'code' in parsedError ||
              'message' in parsedError ||
              'error' in parsedError ||
              'details' in parsedError
            ) {
              errorDetails = parsedError as ErrorResponse;
              errorMessage =
                parsedError.message || parsedError.error || errorMessage;
              console.log('Successfully extracted error details from string');
            }
          }
        } catch {
          // Not JSON, use as plain error message
        }
      } else {
        // For any other error type, convert to string and try to parse
        const errorString = String(error);
        errorMessage = errorString;

        try {
          const parsedError = JSON.parse(errorString);
          if (typeof parsedError === 'object' && parsedError !== null) {
            if (
              'code' in parsedError ||
              'message' in parsedError ||
              'error' in parsedError ||
              'details' in parsedError
            ) {
              errorDetails = parsedError as ErrorResponse;
              errorMessage =
                parsedError.message || parsedError.error || errorMessage;
              console.log(
                'Successfully extracted error details from String(error)'
              );
            }
          }
        } catch {
          // Not JSON, use as plain error message
        }
      }

      // Show error modal for all generation errors
      console.log('Setting error modal with:', {
        errorMessage,
        errorDetails,
        errorDetailsType: typeof errorDetails,
        errorDetailsKeys: errorDetails ? Object.keys(errorDetails) : 'none',
      });

      setGenerationErrorModal({
        isOpen: true,
        errorMessage,
        errorDetails,
      });
      // Stop generating state on error
      setIsGenerating(false);
      setGenerationType(null);
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
      let errorMessage = 'Failed to delete story. Please try again. ' + error;

      if (typeof error === 'object' && error !== null) {
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
            // Handle case where response.data is the ErrorResponse structure
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'message' in axiosError.response.data
          ) {
            // Handle case where the error message is in the 'message' field
            errorMessage = (axiosError.response.data as { message: string })
              .message;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else if (error instanceof Error) {
          // If it's an Error but doesn't have response structure, use the message
          errorMessage = error.message;
        } else {
          // If error doesn't have response structure and isn't an Error, convert to string
          errorMessage = String(error);
        }
      } else if (error instanceof Error) {
        errorMessage = error.message;
      } else {
        errorMessage = String(error);
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
      let errorMessage = 'Failed to export story. Please try again.';

      if (typeof error === 'object' && error !== null) {
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
            // Handle case where response.data is the ErrorResponse structure
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'message' in axiosError.response.data
          ) {
            // Handle case where the error message is in the 'message' field
            errorMessage = (axiosError.response.data as { message: string })
              .message;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else if (error instanceof Error) {
          // If it's an Error but doesn't have response structure, use the message
          errorMessage = error.message;
        } else {
          // If error doesn't have response structure and isn't an Error, convert to string
          errorMessage = String(error);
        }
      } else if (error instanceof Error) {
        errorMessage = error.message;
      } else {
        errorMessage = String(error);
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
        type: 'error',
      });
    },
  });

  // Toggle auto-generation mutation
  const toggleAutoGenerationMutation = useMutation({
    mutationFn: ({ storyId, paused }: { storyId: number; paused: boolean }) =>
      apiToggleAutoGeneration(storyId, paused),
    onMutate: async ({ storyId, paused }) => {
      // Cancel any outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['currentStory'] });

      // Snapshot the previous value
      const previousStory = queryClient.getQueryData(['currentStory']);

      // Optimistically update the story
      queryClient.setQueryData(
        ['currentStory'],
        (old: StoryWithSections | undefined) => {
          if (old && old.id === storyId) {
            return { ...old, auto_generation_paused: paused };
          }
          return old;
        }
      );

      // Return a context object with the snapshotted value
      return { previousStory };
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['currentStory'] });
      queryClient.invalidateQueries({ queryKey: ['stories'] });

      const message = variables.paused
        ? 'Auto-generation paused. Manual generation still works.'
        : 'Auto-generation resumed.';

      showNotificationWithClean({
        title: 'Settings Updated',
        message,
        type: 'success',
      });
    },
    onError: (error: unknown, variables, context) => {
      // If the mutation fails, use the context returned from onMutate to roll back
      if (context?.previousStory) {
        queryClient.setQueryData(['currentStory'], context.previousStory);
      }

      let errorMessage = 'Failed to update auto-generation settings.';
      const err = error as AxiosError;
      if (err.response?.data && typeof err.response.data === 'object') {
        const data = err.response.data as { error?: string };
        if (data.error) {
          errorMessage = data.error;
        }
      }

      showNotificationWithClean({
        title: 'Update Failed',
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
        // Check if error has response structure (axios-like error)
        const hasResponse =
          'response' in currentStoryError || 'message' in currentStoryError;
        if (hasResponse) {
          const axiosError = currentStoryError as {
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
            // Handle case where response.data is the ErrorResponse structure
            errorMessage = (axiosError.response.data as { error: string })
              .error;
          } else if (
            axiosError.response?.data &&
            typeof axiosError.response.data === 'object' &&
            'message' in axiosError.response.data
          ) {
            // Handle case where the error message is in the 'message' field
            errorMessage = (axiosError.response.data as { message: string })
              .message;
          } else if (axiosError.message) {
            errorMessage = axiosError.message;
          }
        } else {
          // If error doesn't have response structure, convert to string
          errorMessage = String(currentStoryError);
        }
      } else if (typeof currentStoryError === 'string') {
        errorMessage = currentStoryError;
      }

      setError(errorMessage);
    } else {
      setError(null);
    }
  }, [currentStoryError]);

  // Load section index from localStorage when story changes, or default to last section
  useEffect(() => {
    if (
      currentStory &&
      currentStory.sections &&
      currentStory.sections.length > 0
    ) {
      const savedIndex = loadSectionIndex(currentStory.id!);
      if (savedIndex !== null && savedIndex >= 0 && savedIndex < currentStory.sections.length) {
        // Use saved index if valid
        setCurrentSectionIndex(savedIndex);
      } else {
        // Default to last section for new stories or if saved index is invalid
        setCurrentSectionIndex(currentStory.sections.length - 1);
      }
    }
  }, [currentStory?.id, loadSectionIndex]);

  // Save section index to localStorage whenever it changes
  useEffect(() => {
    if (currentStory?.id !== undefined) {
      saveSectionIndex(currentStory.id, currentSectionIndex);
    }
  }, [currentSectionIndex, currentStory?.id, saveSectionIndex]);

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
    setGenerationErrorModal({
      isOpen: false,
      errorMessage: '',
      errorDetails: undefined,
    });
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

  const toggleAutoGenerationAction = useCallback(
    async (storyId: number, paused: boolean) => {
      await toggleAutoGenerationMutation.mutateAsync({ storyId, paused });
    },
    [toggleAutoGenerationMutation]
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
    generationType,

    // Actions
    createStory,
    archiveStory,
    completeStory,
    setCurrentStory: setCurrentStoryAction,
    generateNextSection,
    deleteStory: deleteStoryAction,
    exportStoryPDF: exportStoryPDFAction,
    toggleAutoGeneration: toggleAutoGenerationAction,

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
      errorDetails: generationErrorModal.errorDetails,
    },
    closeGenerationErrorModal,
  };
};
