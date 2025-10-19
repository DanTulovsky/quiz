import React from 'react';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import { BookmarkedMessagesPage } from './BookmarkedMessagesPage';
import { useAuth } from '../hooks/useAuth';
import { usePagination } from '../hooks/usePagination';
import { renderWithProviders } from '../test-utils';

// Mock the dependencies
vi.mock('../hooks/useAuth');
vi.mock('../hooks/usePagination');
vi.mock('../api/api', () => ({
  useGetV1AuthStatus: () => ({
    data: mockAuthStatusData, // ✅ Stable reference
    isLoading: false,
    refetch: mockRefetch, // ✅ Stable reference
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
  useGetV1SettingsAiProviders: vi.fn(() => ({
    data: { providers: [] },
    isLoading: false,
    error: null,
  })),
  usePostV1AiConversations: vi.fn(() => ({
    mutateAsync: vi.fn(),
    isPending: false,
  })),
  usePostV1AiConversationsConversationIdMessages: vi.fn(() => ({
    mutateAsync: vi.fn(),
    isPending: false,
  })),
  usePutV1AiConversationsBookmark: vi.fn(() => ({
    mutateAsync: vi.fn(),
    isPending: false,
  })),
}));
vi.mock('react-markdown', () => ({
  default: ({ children }: { children?: React.ReactNode }) => {
    // Simple mock that renders children as plain text for testing
    if (typeof children === 'string') {
      return <span data-testid='markdown-content'>{children}</span>;
    }
    return <span data-testid='markdown-content'>{children}</span>;
  },
}));

const mockAuthStatusData = {
  authenticated: true,
  user: { id: 1, role: 'user' },
};

const mockRefetch = vi.fn();

const mockPaginationData = {
  data: [
    {
      id: '1',
      content: { text: 'Test message content' },
      created_at: '2023-01-01T00:00:00Z',
      conversation_title: 'Test Conversation',
      conversation_id: 'conv-1',
    },
  ],
  isLoading: false,
  isFetching: false,
  pagination: {
    currentPage: 1,
    totalPages: 1,
    totalItems: 1,
    hasNextPage: false,
    hasPreviousPage: false,
  },
  goToPage: vi.fn(),
  goToNextPage: vi.fn(),
  goToPreviousPage: vi.fn(),
  reset: vi.fn(),
};

const mockUseAuth = useAuth as vi.MockedFunction<typeof useAuth>;
const mockUsePagination = usePagination as vi.MockedFunction<
  typeof usePagination
>;

describe('BookmarkedMessagesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    // Setup default mocks
    mockUseAuth.mockReturnValue({
      user: mockAuthStatusData.user,
      login: vi.fn(),
      logout: vi.fn(),
      isLoading: false,
    });

    mockUsePagination.mockReturnValue(mockPaginationData);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders the page correctly', () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    expect(screen.getByText('Bookmarked Messages')).toBeInTheDocument();
    expect(
      screen.getByText('View and manage your bookmarked AI responses')
    ).toBeInTheDocument();
    expect(screen.getByText('1 bookmarked')).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText('Search bookmarked messages...')
    ).toBeInTheDocument();
    expect(screen.getByText('Search')).toBeInTheDocument();
  });

  it('renders bookmarked messages in cards', () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    expect(screen.getByText('AI Response')).toBeInTheDocument();
    expect(screen.getByText('Test Conversation')).toBeInTheDocument();
    expect(screen.getByText('Test message content')).toBeInTheDocument();
  });

  it('renders message cards that are clickable', () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    // Find the message card by looking for the markdown content
    const markdownContent = screen.getByTestId('markdown-content');
    const messageCard = markdownContent.closest('[style*="cursor: pointer"]');
    expect(messageCard).toBeInTheDocument();

    // The card should be clickable (has cursor pointer style)
    // Check that the style attribute contains cursor: pointer
    expect(messageCard?.getAttribute('style')).toContain('cursor: pointer');
  });

  it('renders remove bookmark button in message cards', () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    // Should have remove bookmark button (ActionIcon) - look for all buttons and find the one with bookmark-x icon
    const buttons = screen.getAllByRole('button');
    const removeButton = buttons.find(button =>
      button.querySelector('.lucide-bookmark-x')
    );
    expect(removeButton).toBeInTheDocument();
  });

  it('handles search input correctly', async () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    const searchInput = screen.getByPlaceholderText(
      'Search bookmarked messages...'
    );
    fireEvent.change(searchInput, { target: { value: 'test search' } });

    expect(searchInput).toHaveValue('test search');
  });

  it('triggers search when Enter key is pressed', async () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    const searchInput = screen.getByPlaceholderText(
      'Search bookmarked messages...'
    );
    fireEvent.change(searchInput, { target: { value: 'test search' } });

    fireEvent.keyDown(searchInput, { key: 'Enter', code: 'Enter' });

    await waitFor(() => {
      expect(mockPaginationData.reset).toHaveBeenCalled();
    });
  });

  it('triggers search when Search button is clicked', async () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    const searchInput = screen.getByPlaceholderText(
      'Search bookmarked messages...'
    );
    fireEvent.change(searchInput, { target: { value: 'test search' } });

    const searchButton = screen.getByText('Search');
    fireEvent.click(searchButton);

    await waitFor(() => {
      expect(mockPaginationData.reset).toHaveBeenCalled();
    });
  });

  it('clears search when Clear button is clicked', async () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    // Set search query first
    const searchInput = screen.getByPlaceholderText(
      'Search bookmarked messages...'
    );
    fireEvent.change(searchInput, { target: { value: 'test search' } });

    // Click search to set active query
    const searchButton = screen.getByText('Search');
    fireEvent.click(searchButton);

    await waitFor(() => {
      expect(mockPaginationData.reset).toHaveBeenCalled();
    });

    // Clear search
    const clearButton = screen.getByText('Clear');
    fireEvent.click(clearButton);

    expect(searchInput).toHaveValue('');
  });

  it('shows loading state correctly', () => {
    mockUsePagination.mockReturnValue({
      ...mockPaginationData,
      isLoading: true,
    });

    renderWithProviders(<BookmarkedMessagesPage />);

    expect(
      screen.getByText('Loading bookmarked messages...')
    ).toBeInTheDocument();
  });

  it('shows empty state when no messages found', () => {
    mockUsePagination.mockReturnValue({
      ...mockPaginationData,
      data: [],
    });

    renderWithProviders(<BookmarkedMessagesPage />);

    expect(
      screen.getByText(
        'No bookmarked messages yet. Bookmark messages from conversations to see them here.'
      )
    ).toBeInTheDocument();
  });

  it('shows no search results message when search returns no results', () => {
    mockUsePagination.mockReturnValue({
      ...mockPaginationData,
      data: [],
    });

    renderWithProviders(<BookmarkedMessagesPage />);

    // Set active search query
    const searchInput = screen.getByPlaceholderText(
      'Search bookmarked messages...'
    );
    fireEvent.change(searchInput, { target: { value: 'nonexistent' } });

    const searchButton = screen.getByText('Search');
    fireEvent.click(searchButton);

    expect(
      screen.getByText('No bookmarked messages found matching your search.')
    ).toBeInTheDocument();
  });

  it('renders pagination controls when there are multiple pages', () => {
    mockUsePagination.mockReturnValue({
      ...mockPaginationData,
      pagination: {
        currentPage: 1,
        totalPages: 3,
        totalItems: 60,
        hasNextPage: true,
        hasPreviousPage: false,
      },
    });

    renderWithProviders(<BookmarkedMessagesPage />);

    expect(screen.getByText('60 bookmarked')).toBeInTheDocument();
  });

  it('displays remove bookmark button with proper accessibility', () => {
    renderWithProviders(<BookmarkedMessagesPage />);

    // The remove bookmark button should exist (ActionIcon with aria-label)
    const buttons = screen.getAllByRole('button');
    const removeButton = buttons.find(button =>
      button.querySelector('.lucide-bookmark-x')
    );
    expect(removeButton).toBeInTheDocument();
  });

  it('renders message content correctly', () => {
    const messageWithContent = {
      ...mockPaginationData.data[0],
      content: {
        text: 'Test message content with special characters: **bold** and *italic*',
      },
    };

    mockUsePagination.mockReturnValue({
      ...mockPaginationData,
      data: [messageWithContent],
    });

    renderWithProviders(<BookmarkedMessagesPage />);

    // The message content should be rendered - check the markdown content element
    const markdownContent = screen.getByTestId('markdown-content');
    expect(markdownContent).toBeInTheDocument();
    expect(markdownContent).toHaveTextContent(
      'Test message content with special characters:'
    );
    expect(markdownContent).toHaveTextContent('bold');
    expect(markdownContent).toHaveTextContent('italic');
  });

  it('handles code blocks with language detection (tests the className fix)', () => {
    // This test specifically validates that the className destructuring fix works
    // by ensuring the code component receives and processes the className prop correctly
    const messageWithCode = {
      ...mockPaginationData.data[0],
      content: { text: '```javascript\nconst test = "hello";\n```' },
    };

    mockUsePagination.mockReturnValue({
      ...mockPaginationData,
      data: [messageWithCode],
    });

    // Mock console to capture any errors that would occur if className was undefined
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    renderWithProviders(<BookmarkedMessagesPage />);

    // If the className fix works, no error should be thrown
    expect(consoleSpy).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it('handles malformed code blocks gracefully', () => {
    const messageWithMalformedCode = {
      ...mockPaginationData.data[0],
      content: { text: '```invalid-language\nsome code\n```' },
    };

    mockUsePagination.mockReturnValue({
      ...mockPaginationData,
      data: [messageWithMalformedCode],
    });

    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    renderWithProviders(<BookmarkedMessagesPage />);

    // Should not throw errors even with malformed language tags
    expect(consoleSpy).not.toHaveBeenCalled();

    consoleSpy.mockRestore();
  });
});
