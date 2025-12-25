import {
  render,
  screen,
  cleanup,
  act,
  fireEvent,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MantineProvider } from '@mantine/core';
import * as apiModule from '../api/api';
import * as axiosModule from '../api/axios';
import SavedConversationsPage from './SavedConversationsPage';

// Mock return values for each hook
const mockConversationsData = {
  conversations: [
    {
      id: 'conv-1',
      title: 'Grammar Questions',
      created_at: '2025-01-15T10:30:00Z',
      updated_at: '2025-01-15T10:30:00Z',
      message_count: 3,
      preview_message: 'Explain the grammar...',
      user_id: 1,
    },
    {
      id: 'conv-2',
      title: 'Vocabulary Help',
      created_at: '2025-01-14T15:45:00Z',
      updated_at: '2025-01-14T15:45:00Z',
      message_count: 2,
      preview_message: 'What does this word mean...',
      user_id: 1,
    },
  ],
  total: 2,
};

const mockConversationDetailData = {
  id: 'conv-1',
  title: 'Grammar Questions',
  created_at: '2025-01-15T10:30:00Z',
  updated_at: '2025-01-15T10:30:00Z',
  messages: [
    {
      id: 'msg-1',
      role: 'user',
      content: 'Explain this grammar rule',
      created_at: '2025-01-15T10:30:00Z',
    },
    {
      id: 'msg-2',
      role: 'assistant',
      content: 'This grammar rule explains...',
      created_at: '2025-01-15T10:30:01Z',
    },
  ],
};

// Mock all the API hooks and axios
vi.mock('../api/api', () => ({
  useGetV1AiConversationsId: vi.fn(),
  useDeleteV1AiConversationsId: vi.fn(),
  usePutV1AiConversationsId: vi.fn(),
  usePutV1AiConversationsBookmark: vi.fn(),
  // Mock auth status for AuthProvider
  useGetV1AuthStatus: () => ({
    data: { authenticated: true, user: { id: 1, role: 'user' } },
    isLoading: false,
    refetch: vi.fn(),
  }),
  usePostV1AuthLogin: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePostV1AuthLogout: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePutV1Settings: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}));

// Mock the usePagination hook
vi.mock('../hooks/usePagination', () => ({
  usePagination: vi.fn().mockReturnValue({
    data: [
      {
        id: '1',
        title: 'Grammar Questions',
        created_at: '2025-01-15T10:30:01Z',
        updated_at: '2025-01-15T10:30:01Z',
        message_count: 5,
        is_bookmarked: false,
      },
      {
        id: '2',
        title: 'Vocabulary Help for Academic Writing',
        created_at: '2025-01-14T14:20:00Z',
        updated_at: '2025-01-14T14:20:00Z',
        message_count: 3,
        is_bookmarked: true,
      },
    ],
    isLoading: false,
    isFetching: false,
    pagination: {
      currentPage: 1,
      totalPages: 2,
      totalItems: 50,
      hasNextPage: true,
      hasPreviousPage: false,
    },
    goToPage: vi.fn(),
    goToNextPage: vi.fn(),
    goToPreviousPage: vi.fn(),
    reset: vi.fn(),
  }),
}));

vi.mock('../api/axios', () => ({
  customInstance: vi.fn().mockImplementation(config => {
    // Mock conversations API response
    if (config.url === '/v1/ai/conversations') {
      return Promise.resolve({
        conversations: [
          {
            id: '1',
            title: 'Grammar Questions and Common Mistakes',
            created_at: '2025-01-15T10:30:01Z',
            updated_at: '2025-01-15T10:30:01Z',
            message_count: 5,
            is_bookmarked: false,
          },
          {
            id: '2',
            title: 'Vocabulary Help for Academic Writing',
            created_at: '2025-01-14T14:20:00Z',
            updated_at: '2025-01-14T14:20:00Z',
            message_count: 3,
            is_bookmarked: true,
          },
        ],
        total: 50,
      });
    }
    // Mock search API response
    if (config.url === '/v1/ai/search') {
      return Promise.resolve({
        conversations: [
          {
            id: '1',
            title: 'Grammar Questions and Common Mistakes',
            created_at: '2025-01-15T10:30:01Z',
            updated_at: '2025-01-15T10:30:01Z',
            message_count: 5,
            is_bookmarked: false,
          },
        ],
        total: 1,
      });
    }
    // Default mock response
    return Promise.resolve({});
  }),
}));

// Mock the useAuth hook
vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({
    user: { id: 1 },
    isAuthenticated: true,
    logout: vi.fn(),
    refreshUser: vi.fn(),
  }),
}));

// Mock the logger
vi.mock('../utils/logger', () => ({
  default: {
    error: vi.fn(),
  },
}));

// Create a test-specific QueryClient configuration
const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        staleTime: 0,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

describe('SavedConversationsPage', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = createTestQueryClient();
    // Clear all mocks before each test
    vi.clearAllMocks();

    // Mock axios responses for pagination hook
    vi.mocked(axiosModule.customInstance).mockImplementation(
      <T,>(config: unknown) => {
        let promise: Promise<T>;
        const configTyped = config as { url?: string };
        if (configTyped.url === '/v1/ai/conversations') {
          promise = Promise.resolve({
            data: mockConversationsData,
          } as T);
        } else if (configTyped.url === '/v1/ai/search') {
          promise = Promise.resolve({
            data: { conversations: [], total: 0 },
          } as T);
        } else {
          promise = Promise.resolve({ data: {} } as T);
        }
        const cancelablePromise = promise as Promise<T> & {
          cancel: () => void;
        };
        cancelablePromise.cancel = vi.fn();
        return cancelablePromise;
      }
    );

    // Setup mock return values for each hook
    vi.mocked(apiModule.useGetV1AiConversationsId).mockReturnValue({
      data: mockConversationDetailData,
      isLoading: false,
      isError: false,
      error: null,
      isSuccess: true,
      isPending: false,
      status: 'success',
      dataUpdatedAt: Date.now(),
      errorUpdatedAt: 0,
      failureCount: 0,
      failureReason: null,
      errorUpdateCount: 0,
      isFetched: true,
      isFetchedAfterMount: true,
      isFetching: false,
      isInitialLoading: false,
      isLoadingError: false,
      isPaused: false,
      isPlaceholderData: false,
      isRefetching: false,
      isRefetchError: false,
      isStale: false,
      refetch: vi.fn(),
      fetchStatus: 'idle',
    } as unknown as ReturnType<typeof apiModule.useGetV1AiConversationsId>);

    vi.mocked(apiModule.useDeleteV1AiConversationsId).mockReturnValue({
      mutateAsync: vi.fn(),
      mutate: vi.fn(),
      reset: vi.fn(),
      status: 'idle',
      isIdle: true,
      isPending: false,
      isError: false,
      isSuccess: false,
      data: undefined,
      error: null,
      failureCount: 0,
      failureReason: null,
      submittedAt: 0,
      variables: undefined,
    } as unknown as ReturnType<typeof apiModule.useDeleteV1AiConversationsId>);

    vi.mocked(apiModule.usePutV1AiConversationsId).mockReturnValue({
      mutateAsync: vi.fn(),
      mutate: vi.fn(),
      reset: vi.fn(),
      status: 'idle',
      isIdle: true,
      isPending: false,
      isError: false,
      isSuccess: false,
      data: undefined,
      error: null,
      failureCount: 0,
      failureReason: null,
      submittedAt: 0,
      variables: undefined,
    } as unknown as ReturnType<typeof apiModule.usePutV1AiConversationsId>);

    vi.mocked(apiModule.usePutV1AiConversationsBookmark).mockReturnValue({
      mutateAsync: vi.fn(),
      mutate: vi.fn(),
      reset: vi.fn(),
      status: 'idle',
      isIdle: true,
      isPending: false,
      isError: false,
      isSuccess: false,
      data: undefined,
      error: null,
      failureCount: 0,
      failureReason: null,
      submittedAt: 0,
      variables: undefined,
    } as unknown as ReturnType<
      typeof apiModule.usePutV1AiConversationsBookmark
    >);
  });

  afterEach(() => {
    cleanup();
    queryClient.clear();
  });
  it('should render the page title and description', () => {
    act(() => {
      render(
        <QueryClientProvider client={queryClient}>
          <MantineProvider>
            <BrowserRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <SavedConversationsPage />
            </BrowserRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    expect(screen.getByText('Saved AI Conversations')).toBeInTheDocument();
    expect(
      screen.getByText('View and manage your saved AI conversations')
    ).toBeInTheDocument();
  });

  it('should render search input', () => {
    act(() => {
      render(
        <QueryClientProvider client={queryClient}>
          <MantineProvider>
            <BrowserRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <SavedConversationsPage />
            </BrowserRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    const searchInput = screen.getByPlaceholderText(
      'Type to prepare search query...'
    );
    expect(searchInput).toBeInTheDocument();
  });

  it('should render conversation cards', () => {
    act(() => {
      render(
        <QueryClientProvider client={queryClient}>
          <MantineProvider>
            <BrowserRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <SavedConversationsPage />
            </BrowserRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Check that conversation cards are rendered - look for titles specifically
    expect(screen.getAllByText('Grammar Questions').length).toBeGreaterThan(0);
    expect(
      screen.getAllByText('Vocabulary Help for Academic Writing').length
    ).toBeGreaterThan(0);
  });

  it('allows editing a conversation title via Save', async () => {
    const mutateAsync = vi.fn().mockResolvedValue({});
    vi.mocked(apiModule.usePutV1AiConversationsId).mockReturnValue({
      mutateAsync,
    } as unknown as ReturnType<typeof apiModule.usePutV1AiConversationsId>);

    await act(async () => {
      render(
        <QueryClientProvider client={queryClient}>
          <MantineProvider>
            <BrowserRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <SavedConversationsPage />
            </BrowserRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Open actions menu for the first conversation
    const actionButtons = screen.getAllByRole('button', {
      name: 'Conversation actions',
    });
    expect(actionButtons.length).toBeGreaterThan(0);
    await act(async () => {
      actionButtons[0].click();
    });

    // Click Edit Title
    const editItem = await screen.findByText('Edit Title');
    await act(async () => {
      editItem.click();
    });

    // Change input value
    const titleInput = await screen.findByLabelText('Title');
    await act(async () => {
      fireEvent.change(titleInput, { target: { value: 'New Title' } });
    });

    // Click Save
    const saveButton = await screen.findByText('Save');
    await act(async () => {
      saveButton.click();
    });

    expect(mutateAsync).toHaveBeenCalled();
  });

  it('displays pagination controls when there are multiple pages', () => {
    // Mock a response with more items than the page limit
    const largeMockData = {
      conversations: Array.from({ length: 25 }, (_, i) => ({
        id: `conv-${i + 1}`,
        title: `Conversation ${i + 1}`,
        created_at: '2025-01-15T10:30:00Z',
        updated_at: '2025-01-15T10:30:00Z',
        message_count: 3,
        preview_message: 'Explain the grammar...',
        user_id: 1,
      })),
      total: 50, // More than one page
    };

    vi.mocked(axiosModule.customInstance).mockImplementation(
      <T,>(config: unknown) => {
        let promise: Promise<T>;
        const configTyped = config as { url?: string };
        if (configTyped.url === '/v1/ai/conversations') {
          promise = Promise.resolve({ data: largeMockData } as T);
        } else {
          promise = Promise.resolve({ data: {} } as T);
        }
        const cancelablePromise = promise as Promise<T> & {
          cancel: () => void;
        };
        cancelablePromise.cancel = vi.fn();
        return cancelablePromise;
      }
    );

    act(() => {
      render(
        <QueryClientProvider client={queryClient}>
          <MantineProvider>
            <BrowserRouter
              future={{
                v7_startTransition: false,
                v7_relativeSplatPath: false,
              }}
            >
              <SavedConversationsPage />
            </BrowserRouter>
          </MantineProvider>
        </QueryClientProvider>
      );
    });

    // Should display pagination info
    expect(screen.getByText(/Showing 50 items/)).toBeInTheDocument();
    expect(screen.getByText(/Page 1 of 2/)).toBeInTheDocument();
  });
});
