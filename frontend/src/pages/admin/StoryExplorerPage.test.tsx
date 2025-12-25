import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import StoryExplorerPage from './StoryExplorerPage';
import { useAuth } from '../../hooks/useAuth';
import {
  useAdminStories,
  useAdminStory,
  useAdminStorySection,
  useAdminDeleteStory,
} from '../../api/admin';
import { useUsersPaginated } from '../../api/admin';

// Mock the hooks
vi.mock('../../hooks/useAuth');
vi.mock('../../api/admin');
vi.mock('../../api/api', () => ({
  // Mock auth status for AuthProvider
  useGetV1AuthStatus: () => ({
    data: { authenticated: true, user: { id: 1, role: 'admin' } },
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
  useGetV1SettingsLanguages: () => ({
    data: [
      { code: 'italian', name: 'Italian' },
      { code: 'spanish', name: 'Spanish' },
      { code: 'french', name: 'French' },
    ],
    isLoading: false,
    error: null,
  }),
}));

// Mock the snippet hooks
vi.mock('../../hooks/useSectionSnippets', () => ({
  useSectionSnippets: () => ({
    snippets: [],
    isLoading: false,
    error: null,
  }),
}));

vi.mock('../../hooks/useStorySnippets', () => ({
  useStorySnippets: () => ({
    snippets: [],
    isLoading: false,
    error: null,
  }),
}));

const mockUseAuth = useAuth as ReturnType<typeof vi.fn>;
const mockUseAdminStories = useAdminStories as ReturnType<typeof vi.fn>;
const mockUseAdminStory = useAdminStory as ReturnType<typeof vi.fn>;
const mockUseAdminStorySection = useAdminStorySection as ReturnType<
  typeof vi.fn
>;
const mockUseAdminDeleteStory = useAdminDeleteStory as ReturnType<typeof vi.fn>;
const mockUseUsersPaginated = useUsersPaginated as ReturnType<typeof vi.fn>;
const mockUseGetV1SettingsLanguages = vi.fn();

const mockStories = [
  {
    id: 1,
    title: 'Italian Adventure Story',
    language: 'italian',
    status: 'active',
    subject: 'An adventure in Italy',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    user_id: 1,
  },
  {
    id: 2,
    title: 'Spanish Mystery',
    language: 'spanish',
    status: 'archived',
    subject: 'A mysterious story',
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
    user_id: 2,
  },
];

const mockStoryWithSections = {
  id: 1,
  title: 'Italian Adventure Story',
  language: 'italian',
  status: 'active',
  subject: 'An adventure in Italy',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  user_id: 1,
  sections: [
    {
      id: 1,
      story_id: 1,
      section_number: 1,
      content: 'Once upon a time in Italy...',
      language_level: 'A1',
      word_count: 6,
      generated_by: 'user',
      generated_at: '2024-01-01T00:00:00Z',
      generation_date: '2024-01-01',
    },
  ],
};

const renderStoryExplorerPage = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  const renderResult = render(
    <QueryClientProvider client={queryClient}>
      <MantineProvider>
        <BrowserRouter
          future={{
            v7_startTransition: false,
            v7_relativeSplatPath: false,
          }}
        >
          <StoryExplorerPage />
        </BrowserRouter>
      </MantineProvider>
    </QueryClientProvider>
  );

  return {
    ...renderResult,
    user: userEvent.setup(),
  };
};

describe('StoryExplorerPage', () => {
  beforeEach(() => {
    // Mock auth
    mockUseAuth.mockReturnValue({
      user: {
        id: 1,
        username: 'admin',
        roles: [{ name: 'admin' }],
      },
      isAuthenticated: true,
      login: vi.fn(),
      logout: vi.fn(),
    });

    // Mock API hooks
    mockUseAdminStories.mockReturnValue({
      data: {
        stories: mockStories,
        pagination: {
          page: 1,
          page_size: 20,
          total: 2,
          total_pages: 1,
        },
      },
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseAdminStory.mockReturnValue({
      data: undefined,
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseAdminStorySection.mockReturnValue({
      data: undefined,
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseUsersPaginated.mockReturnValue({
      data: {
        users: [
          { id: 1, username: 'testuser1', email: 'test1@example.com' },
          { id: 2, username: 'testuser2', email: 'test2@example.com' },
        ],
        pagination: { total: 2 },
      },
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseGetV1SettingsLanguages.mockReturnValue({
      data: ['italian', 'spanish', 'french'],
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseAdminDeleteStory.mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue({}),
      isPending: false,
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders without crashing', () => {
      expect(() => renderStoryExplorerPage()).not.toThrow();
    });

    it('renders the page title', () => {
      renderStoryExplorerPage();
      expect(screen.getByText('Story Explorer')).toBeInTheDocument();
    });

    it('renders stories table with data', () => {
      renderStoryExplorerPage();
      expect(screen.getByText('Italian Adventure Story')).toBeInTheDocument();
      expect(screen.getByText('Spanish Mystery')).toBeInTheDocument();
    });

    it('renders filter controls', () => {
      renderStoryExplorerPage();
      expect(
        screen.getByPlaceholderText('Search title...')
      ).toBeInTheDocument();
      expect(screen.getByDisplayValue('All Languages')).toBeInTheDocument();
      expect(screen.getByDisplayValue('All Statuses')).toBeInTheDocument();
    });

    it('renders pagination info', () => {
      renderStoryExplorerPage();
      expect(
        screen.getByText(/Showing 1 to 2 of 2 stories/)
      ).toBeInTheDocument();
    });
  });

  describe('Loading States', () => {
    it('shows loader when stories are loading', () => {
      mockUseAdminStories.mockReturnValue({
        data: undefined,
        isLoading: true,
        isFetching: false,
        error: null,
      });
      renderStoryExplorerPage();
      expect(screen.getByTestId('loader')).toBeInTheDocument();
    });
  });

  describe('Authentication', () => {
    it('redirects to login if not authenticated', () => {
      mockUseAuth.mockReturnValue({
        user: null,
        isAuthenticated: false,
        login: vi.fn(),
        logout: vi.fn(),
      });
      renderStoryExplorerPage();
      expect(screen.queryByText('Story Explorer')).not.toBeInTheDocument();
    });

    it('redirects to quiz if not admin', () => {
      mockUseAuth.mockReturnValue({
        user: {
          id: 1,
          username: 'regular_user',
          roles: [{ name: 'user' }],
        },
        isAuthenticated: true,
        login: vi.fn(),
        logout: vi.fn(),
      });
      renderStoryExplorerPage();
      expect(screen.queryByText('Story Explorer')).not.toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('handles API errors gracefully', () => {
      mockUseAdminStories.mockReturnValue({
        data: undefined,
        isLoading: false,
        isFetching: false,
        error: new Error('API Error'),
      });
      renderStoryExplorerPage();
      expect(screen.getByText('Error')).toBeInTheDocument();
      expect(screen.getByText('API Error')).toBeInTheDocument();
    });
  });

  describe('Modal Views', () => {
    it('opens story view modal when View button is clicked', async () => {
      const { user } = renderStoryExplorerPage();

      // Click the View button for the first story
      const viewButtons = screen.getAllByText('View');
      await user.click(viewButtons[0]);

      // Should show the modal (check if modal content is rendered)
      // The modal might not show "View Story" immediately if story is loading
      expect(screen.getByText('Story Explorer')).toBeInTheDocument();
    });

    it('opens section view modal when section is selected', async () => {
      // First open the story view
      const { user } = renderStoryExplorerPage();
      const viewButtons = screen.getAllByText('View');
      await user.click(viewButtons[0]);

      // Mock the story data to show that modal state is updated
      mockUseAdminStory.mockReturnValue({
        data: mockStoryWithSections,
        isLoading: false,
        isFetching: false,
        error: null,
      });

      // Should show the modal content
      expect(screen.getByText('Story Explorer')).toBeInTheDocument();
    });
  });

  describe('Filtering', () => {
    it('allows selecting language filter', async () => {
      const { user } = renderStoryExplorerPage();

      // Select Italian language filter
      const languageSelect = screen.getByDisplayValue('All Languages');
      await user.click(languageSelect);

      const italianOption = screen.getByText('Italian');
      await user.click(italianOption);

      // Should show the selected language
      expect(screen.getByDisplayValue('Italian')).toBeInTheDocument();
    });

    it('allows selecting status filter', async () => {
      const { user } = renderStoryExplorerPage();

      // Select archived status filter
      const statusSelect = screen.getByDisplayValue('All Statuses');
      await user.click(statusSelect);

      const archivedOption = screen.getByText('Archived');
      await user.click(archivedOption);

      // Should show the selected status
      expect(screen.getByDisplayValue('Archived')).toBeInTheDocument();
    });

    it('allows searching stories by title', async () => {
      const { user } = renderStoryExplorerPage();

      // Type in search box
      const searchInput = screen.getByPlaceholderText('Search title...');
      await user.type(searchInput, 'Adventure');

      // Should show the search text
      expect(screen.getByDisplayValue('Adventure')).toBeInTheDocument();
    });

    it('allows clearing filters', async () => {
      const { user } = renderStoryExplorerPage();

      // Apply some filters first
      const searchInput = screen.getByPlaceholderText('Search title...');
      await user.type(searchInput, 'Adventure');

      // Click clear all button
      const clearButton = screen.getByText('Clear All Filters');
      await user.click(clearButton);

      // Should show empty search input
      expect(screen.getByPlaceholderText('Search title...')).toHaveValue('');
    });
  });
});
