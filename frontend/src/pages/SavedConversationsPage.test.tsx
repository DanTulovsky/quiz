import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MantineProvider } from '@mantine/core';
import SavedConversationsPage from './SavedConversationsPage';

// Mock all the API hooks
vi.mock('../api/api', () => ({
  useGetV1AiConversations: vi.fn(() => ({
    data: {
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
    },
    isLoading: false,
  })),
  useGetV1AiSearch: vi.fn(() => ({
    data: { conversations: [], total_count: 0 },
    isLoading: false,
  })),
  useGetV1AiConversationsId: vi.fn(() => ({
    data: {
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
    },
    isLoading: false,
  })),
  useDeleteV1AiConversationsId: vi.fn(() => ({
    mutateAsync: vi.fn(),
  })),
  usePutV1AiConversationsId: vi.fn(() => ({
    mutateAsync: vi.fn(),
  })),
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

describe('SavedConversationsPage', () => {
  it('should render the page title and description', () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MantineProvider>
          <BrowserRouter>
            <SavedConversationsPage />
          </BrowserRouter>
        </MantineProvider>
      </QueryClientProvider>
    );

    expect(screen.getByText('Saved AI Conversations')).toBeInTheDocument();
    expect(
      screen.getByText('View and manage your saved AI conversations')
    ).toBeInTheDocument();
  });

  it('should render search input', () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MantineProvider>
          <BrowserRouter>
            <SavedConversationsPage />
          </BrowserRouter>
        </MantineProvider>
      </QueryClientProvider>
    );

    const searchInput = screen.getByPlaceholderText(
      'Type to prepare search query...'
    );
    expect(searchInput).toBeInTheDocument();
  });

  it('should render conversation cards', () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MantineProvider>
          <BrowserRouter>
            <SavedConversationsPage />
          </BrowserRouter>
        </MantineProvider>
      </QueryClientProvider>
    );

    expect(screen.getByText('Grammar Questions')).toBeInTheDocument();
    expect(screen.getByText('Vocabulary Help')).toBeInTheDocument();
  });
});
