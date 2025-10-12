import { render, screen, cleanup, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MantineProvider } from '@mantine/core';
import * as apiModule from '../api/api';
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
  total_count: 2,
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

// Mock all the API hooks
vi.mock('../api/api', () => ({
  useGetV1AiConversations: vi.fn(),
  useGetV1AiSearch: vi.fn(),
  useGetV1AiConversationsId: vi.fn(),
  useDeleteV1AiConversationsId: vi.fn(),
  usePutV1AiConversationsId: vi.fn(),
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

    // Setup mock return values for each hook
    vi.mocked(apiModule.useGetV1AiConversations).mockReturnValue({
      data: mockConversationsData,
      isLoading: false,
    });

    vi.mocked(apiModule.useGetV1AiSearch).mockReturnValue({
      data: { conversations: [], total_count: 0 },
      isLoading: false,
    });

    vi.mocked(apiModule.useGetV1AiConversationsId).mockReturnValue({
      data: mockConversationDetailData,
      isLoading: false,
    });

    vi.mocked(apiModule.useDeleteV1AiConversationsId).mockReturnValue({
      mutateAsync: vi.fn(),
    });

    vi.mocked(apiModule.usePutV1AiConversationsId).mockReturnValue({
      mutateAsync: vi.fn(),
    });
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

    expect(screen.getByText('Grammar Questions')).toBeInTheDocument();
    expect(screen.getByText('Vocabulary Help')).toBeInTheDocument();
  });
});
