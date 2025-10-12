import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { AXIOS_INSTANCE } from './axios';
import { Role, Story, StoryWithSections, StorySectionWithQuestions } from './api';

// Types for admin data
export interface UserWithProgress {
  user: {
    id: number;
    username: string;
    email?: string;
    timezone?: string;
    preferred_language?: string;
    current_level?: string;
    ai_provider?: string;
    ai_model?: string;
    ai_enabled?: boolean;
    is_paused?: boolean;
    last_active?: string;
    created_at: string;
    updated_at?: string;
    roles?: Role[];
  };
  progress?: {
    current_level: string;
    suggested_level: string;
    accuracy_rate: number;
    total_questions: number;
    correct_answers: number;
  };
  question_stats?: {
    user_id: number;
    total_answered: number;
    answered_by_type?: Record<string, number>;
    answered_by_level?: Record<string, number>;
    accuracy_by_type?: Record<string, number>;
    accuracy_by_level?: Record<string, number>;
    available_by_type?: Record<string, number>;
    available_by_level?: Record<string, number>;
  };
  user_question_counts?: Record<string, unknown>;
}

// API hooks for admin functionality
export const useBackendAdminData = () => {
  return useQuery({
    queryKey: ['backend-admin-data'],
    queryFn: async () => {
      const response = await AXIOS_INSTANCE.get('/v1/admin/backend/dashboard', {
        headers: {
          Accept: 'application/json',
        },
      });
      return response.data;
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });
};

export const useRoles = () => {
  return useQuery({
    queryKey: ['admin-roles'],
    queryFn: async () => {
      const response = await AXIOS_INSTANCE.get('/v1/admin/backend/roles', {
        headers: {
          Accept: 'application/json',
        },
      });
      return response.data.roles;
    },
  });
};

export const useCreateUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (userData: {
      username: string;
      email?: string;
      timezone?: string;
      password: string;
      preferred_language?: string;
      current_level?: string;
    }) => {
      const response = await AXIOS_INSTANCE.post(
        '/v1/admin/backend/userz',
        userData,
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
      queryClient.invalidateQueries({ queryKey: ['users-paginated'] });
    },
  });
};

export const useUpdateUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      userId,
      userData,
    }: {
      userId: number;
      userData: {
        username?: string;
        email?: string;
        timezone?: string;
        preferred_language?: string;
        current_level?: string;
        ai_enabled?: boolean;
        ai_provider?: string;
        ai_model?: string;
        api_key?: string;
        selectedRoles?: string[];
      };
    }) => {
      const response = await AXIOS_INSTANCE.put(
        `/v1/admin/backend/userz/${userId}`,
        userData,
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
      queryClient.invalidateQueries({ queryKey: ['users-paginated'] });
    },
  });
};

export const useDeleteUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (userId: number) => {
      const response = await AXIOS_INSTANCE.delete(
        `/v1/admin/backend/userz/${userId}`,
        {
          headers: {
            Accept: 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
      queryClient.invalidateQueries({ queryKey: ['users-paginated'] });
    },
  });
};

export const useResetUserPassword = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      userId,
      newPassword,
    }: {
      userId: number;
      newPassword: string;
    }) => {
      const response = await AXIOS_INSTANCE.post(
        `/v1/admin/backend/userz/${userId}/reset-password`,
        {
          new_password: newPassword,
        },
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      queryClient.invalidateQueries({ queryKey: ['users-paginated'] });
    },
  });
};

export const useClearUserData = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const response = await AXIOS_INSTANCE.post(
        '/v1/admin/backend/clear-user-data',
        {},
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
    },
  });
};

// Data Explorer specific types and hooks

export interface QuestionWithStats {
  id: number;
  type: string;
  language: string;
  level: string;
  content: {
    question?: string;
    sentence?: string;
    passage?: string;
    options?: string[];
  };
  correct_answer: string;
  explanation: string;
  topic_category?: string;
  grammar_focus?: string;
  vocabulary_domain?: string;
  scenario?: string;
  style_modifier?: string;
  difficulty_modifier?: string;
  time_context?: string;
  usage_count: number;
  user_count: number;
  created_at: string;
  status: string;
  is_reported: boolean;
  reporters?: string;
  report_reasons?: string;
  response_stats?: {
    total: number;
    correct: number;
    accuracy: number;
  };
}

export const useAllQuestions = (
  page: number = 1,
  pageSize: number = 20,
  search?: string,
  typeFilter?: string,
  statusFilter?: string,
  languageFilter?: string,
  levelFilter?: string,
  userId?: number
) => {
  return useQuery({
    queryKey: [
      'all-questions',
      page,
      pageSize,
      search,
      typeFilter,
      statusFilter,
      languageFilter,
      levelFilter,
      userId,
    ],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
      });

      if (search) params.append('search', search);
      if (typeFilter) params.append('type', typeFilter);
      if (statusFilter) params.append('status', statusFilter);
      if (languageFilter) params.append('language', languageFilter);
      if (levelFilter) params.append('level', levelFilter);
      if (userId) params.append('user_id', userId.toString());

      const response = await AXIOS_INSTANCE.get(
        `/v1/admin/backend/questions?${params.toString()}`,
        {
          headers: {
            Accept: 'application/json',
          },
        }
      );
      return response.data;
    },
    placeholderData: previousData => previousData,
    staleTime: 30000, // 30 seconds
  });
};

export const useReportedQuestions = (
  page: number = 1,
  pageSize: number = 20,
  search?: string,
  typeFilter?: string,
  languageFilter?: string,
  levelFilter?: string
) => {
  return useQuery({
    queryKey: [
      'reported-questions',
      page,
      pageSize,
      search,
      typeFilter,
      languageFilter,
      levelFilter,
    ],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
      });

      if (search) params.append('search', search);
      if (typeFilter) params.append('type', typeFilter);
      if (languageFilter) params.append('language', languageFilter);
      if (levelFilter) params.append('level', levelFilter);

      const response = await AXIOS_INSTANCE.get(
        `/v1/admin/backend/reported-questions?${params.toString()}`,
        {
          headers: {
            Accept: 'application/json',
          },
        }
      );
      return response.data;
    },
    placeholderData: previousData => previousData,
    staleTime: 30000, // 30 seconds
  });
};

export const useUpdateQuestion = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      questionId,
      data,
    }: {
      questionId: number;
      data: {
        content?: Record<string, unknown>;
        correct_answer?: number;
        explanation?: string;
      };
    }) => {
      const response = await AXIOS_INSTANCE.put(
        `/v1/admin/backend/questions/${questionId}`,
        data,
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
    },
  });
};

export const useMarkQuestionAsFixed = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (questionId: number) => {
      const response = await AXIOS_INSTANCE.post(
        `/v1/admin/backend/questions/${questionId}/fix`,
        {},
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-explorer'] });
      queryClient.invalidateQueries({ queryKey: ['reported-questions'] });
    },
  });
};

export const useFixQuestionWithAI = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({
      questionId,
      additionalContext,
    }: {
      questionId: number;
      additionalContext?: string;
    }) => {
      const body: Record<string, unknown> = {};
      if (additionalContext) body.additional_context = additionalContext;
      const response = await AXIOS_INSTANCE.post(
        `/v1/admin/backend/questions/${questionId}/ai-fix`,
        body,
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-explorer'] });
      queryClient.invalidateQueries({ queryKey: ['reported-questions'] });
    },
  });
};

export const useClearUserDataForUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (userId: number) => {
      const response = await AXIOS_INSTANCE.post(
        `/v1/admin/backend/userz/${userId}/clear`,
        {},
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-explorer'] });
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      queryClient.invalidateQueries({ queryKey: ['users-paginated'] });
    },
  });
};

export const useClearDatabase = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const response = await AXIOS_INSTANCE.post(
        '/v1/admin/backend/clear-database',
        {},
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['data-explorer'] });
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
    },
  });
};

export const useUsersForQuestion = (questionId: number) => {
  return useQuery({
    queryKey: ['question-users', questionId],
    queryFn: async () => {
      const response = await AXIOS_INSTANCE.get(
        `/v1/admin/backend/questions/${questionId}/users`,
        {
          headers: {
            Accept: 'application/json',
          },
        }
      );
      return response.data;
    },
    enabled: !!questionId,
    staleTime: 30000, // Cache for 30 seconds
    retry: 2, // Only retry twice to avoid overwhelming the server
    retryDelay: 1000, // Wait 1 second between retries
  });
};

export const useAssignUsersToQuestion = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      questionId,
      userIDs,
    }: {
      questionId: number;
      userIDs: number[];
    }) => {
      if (!Array.isArray(userIDs) || userIDs.length === 0) {
        throw new Error('user_ids cannot be empty');
      }
      const response = await AXIOS_INSTANCE.post(
        `/v1/admin/backend/questions/${questionId}/assign-users`,
        { user_ids: userIDs },
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: (_, { questionId }) => {
      // Invalidate and refetch users for this question
      queryClient.invalidateQueries({
        queryKey: ['question-users', questionId],
      });
      // Also invalidate the questions list to refresh user counts
      queryClient.invalidateQueries({ queryKey: ['all-questions'] });
    },
  });
};

export const useUnassignUsersFromQuestion = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      questionId,
      userIDs,
    }: {
      questionId: number;
      userIDs: number[];
    }) => {
      if (!Array.isArray(userIDs) || userIDs.length === 0) {
        throw new Error('user_ids cannot be empty');
      }
      const response = await AXIOS_INSTANCE.post(
        `/v1/admin/backend/questions/${questionId}/unassign-users`,
        { user_ids: userIDs },
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: (_, { questionId }) => {
      // Invalidate and refetch users for this question
      queryClient.invalidateQueries({
        queryKey: ['question-users', questionId],
      });
      // Also invalidate the questions list to refresh user counts
      queryClient.invalidateQueries({ queryKey: ['all-questions'] });
    },
  });
};

export const useDeleteQuestion = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (questionId: number) => {
      const response = await AXIOS_INSTANCE.delete(
        `/v1/admin/backend/questions/${questionId}`,
        {
          headers: {
            Accept: 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['all-questions'] });
      queryClient.invalidateQueries({ queryKey: ['reported-questions'] });
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
    },
  });
};

// User pause/unpause functionality
export const usePauseUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (userId: number) => {
      const response = await AXIOS_INSTANCE.post(
        '/v1/admin/worker/users/pause',
        { user_id: userId },
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users-paginated'] });
      queryClient.invalidateQueries({
        queryKey: ['users-paginated'],
        exact: false,
      });
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
    },
  });
};

export const useResumeUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (userId: number) => {
      const response = await AXIOS_INSTANCE.post(
        '/v1/admin/worker/users/resume',
        { user_id: userId },
        {
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
          },
        }
      );
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users-paginated'] });
      queryClient.invalidateQueries({
        queryKey: ['users-paginated'],
        exact: false,
      });
      queryClient.invalidateQueries({ queryKey: ['admin-users'] });
      queryClient.invalidateQueries({ queryKey: ['backend-admin-data'] });
    },
  });
};

export const useUsersPaginated = ({
  page = 1,
  pageSize = 20,
  search,
  language,
  level,
  aiProvider,
  aiModel,
  aiEnabled,
  active,
}: {
  page?: number;
  pageSize?: number;
  search?: string;
  language?: string;
  level?: string;
  aiProvider?: string;
  aiModel?: string;
  aiEnabled?: string;
  active?: string;
}) => {
  const queryClient = useQueryClient();

  return useQuery({
    queryKey: [
      'users-paginated',
      page,
      pageSize,
      search,
      language,
      level,
      aiProvider,
      aiModel,
      aiEnabled,
      active,
    ],
    queryFn: async () => {
      const params = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
      });

      if (search) params.append('search', search);
      if (language) params.append('language', language);
      if (level) params.append('level', level);
      if (aiProvider) params.append('ai_provider', aiProvider);
      if (aiModel) params.append('ai_model', aiModel);
      if (aiEnabled) params.append('ai_enabled', aiEnabled);
      if (active) params.append('active', active);

      const response = await AXIOS_INSTANCE.get(
        `/v1/admin/backend/userz/paginated?${params.toString()}`,
        {
          headers: {
            Accept: 'application/json',
          },
        }
      );
      return response.data;
    },
    staleTime: 30000, // 30 seconds
    retry: 2,
    retryDelay: 1000,
  });
};

// --- Story Explorer (Admin) ---

type StoriesPaginatedResponse = {
  stories: Story[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
};

export const useAdminStories = (
  page: number = 1,
  pageSize: number = 20,
  search?: string,
  language?: string,
  status?: string,
  userId?: number
) => {
  return useQuery({
    queryKey: ['admin-stories', page, pageSize, search, language, status, userId],
    queryFn: async (): Promise<StoriesPaginatedResponse> => {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
      });
      if (search) params.append('search', search);
      if (language) params.append('language', language);
      if (status) params.append('status', status);
      if (userId) params.append('user_id', String(userId));
      const resp = await AXIOS_INSTANCE.get(`/v1/admin/backend/stories?${params.toString()}`, {
        headers: { Accept: 'application/json' },
      });
      return resp.data as StoriesPaginatedResponse;
    },
    placeholderData: previous => previous,
    staleTime: 30000,
  });
};

export const useAdminStory = (storyId: number | null) => {
  return useQuery({
    queryKey: ['admin-story', storyId],
    enabled: !!storyId,
    queryFn: async (): Promise<StoryWithSections> => {
      const resp = await AXIOS_INSTANCE.get(`/v1/admin/backend/stories/${storyId}`, {
        headers: { Accept: 'application/json' },
      });
      return resp.data as StoryWithSections;
    },
    staleTime: 30000,
  });
};

export const useAdminStorySection = (sectionId: number | null) => {
  return useQuery({
    queryKey: ['admin-story-section', sectionId],
    enabled: !!sectionId,
    queryFn: async (): Promise<StorySectionWithQuestions> => {
      const resp = await AXIOS_INSTANCE.get(`/v1/admin/backend/story-sections/${sectionId}`, {
        headers: { Accept: 'application/json' },
      });
      return resp.data as StorySectionWithQuestions;
    },
    staleTime: 30000,
  });
};
