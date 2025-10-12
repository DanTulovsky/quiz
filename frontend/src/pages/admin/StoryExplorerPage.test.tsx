import React from 'react';
import { render, screen } from '@testing-library/react';
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
} from '../../api/admin';
import { useUsersPaginated } from '../../api/admin';

// Mock the hooks
vi.mock('../../hooks/useAuth');
vi.mock('../../api/admin');

const mockUseAuth = useAuth as ReturnType<typeof vi.fn>;
const mockUseAdminStories = useAdminStories as ReturnType<typeof vi.fn>;
const mockUseAdminStory = useAdminStory as ReturnType<typeof vi.fn>;
const mockUseAdminStorySection = useAdminStorySection as ReturnType<
  typeof vi.fn
>;
const mockUseUsersPaginated = useUsersPaginated as ReturnType<typeof vi.fn>;

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

const mockSectionWithQuestions = {
  id: 1,
  story_id: 1,
  section_number: 1,
  content: 'Once upon a time in Italy...',
  language_level: 'A1',
  word_count: 6,
  generated_by: 'user',
  generated_at: '2024-01-01T00:00:00Z',
  generation_date: '2024-01-01',
  questions: [
    {
      id: 1,
      section_id: 1,
      question_text: 'What happened in Italy?',
      options: ['Adventure', 'Mystery', 'Romance'],
      correct_answer_index: 0,
      explanation: 'It was an adventure story',
      created_at: '2024-01-01T00:00:00Z',
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

  return render(
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
        total: 2,
      },
      isLoading: false,
      isFetching: false,
      error: null,
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
        screen.getByPlaceholderText('Search stories...')
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
      expect(
        screen.getByText(/Failed to load story explorer data/)
      ).toBeInTheDocument();
    });
  });

  describe('Modal Views', () => {
    it('opens story view modal when View button is clicked', async () => {
      const { user } = renderStoryExplorerPage();

      // Click the View button for the first story
      const viewButtons = screen.getAllByText('View');
      await user.click(viewButtons[0]);

      // Should show the story reading view
      expect(screen.getByText('Story Details')).toBeInTheDocument();
      expect(screen.getByText('An adventure in Italy')).toBeInTheDocument();
    });

    it('opens section view modal when section is selected', async () => {
      // First open the story view
      const { user } = renderStoryExplorerPage();
      const viewButtons = screen.getAllByText('View');
      await user.click(viewButtons[0]);

      // Mock the story data
      mockUseAdminStory.mockReturnValue({
        data: mockStoryWithSections,
        isLoading: false,
        isFetching: false,
        error: null,
      });

      // Click on a section
      const sectionButton = screen.getByText('Section 1');
      await user.click(sectionButton);

      // Should show the section view
      expect(screen.getByText('Section 1')).toBeInTheDocument();
      expect(
        screen.getByText('Once upon a time in Italy...')
      ).toBeInTheDocument();
    });

    it('displays questions in section view', async () => {
      // First open the story view
      const { user } = renderStoryExplorerPage();
      const viewButtons = screen.getAllByText('View');
      await user.click(viewButtons[0]);

      // Mock the story data
      mockUseAdminStory.mockReturnValue({
        data: mockStoryWithSections,
        isLoading: false,
        isFetching: false,
        error: null,
      });

      // Mock section data with questions
      mockUseAdminStorySection.mockReturnValue({
        data: mockSectionWithQuestions,
        isLoading: false,
        isFetching: false,
        error: null,
      });

      // Click on a section
      const sectionButton = screen.getByText('Section 1');
      await user.click(sectionButton);

      // Should show questions
      expect(screen.getByText('Comprehension Questions')).toBeInTheDocument();
      expect(screen.getByText('What happened in Italy?')).toBeInTheDocument();
    });
  });

  describe('Filtering', () => {
    it('filters stories by language', async () => {
      const { user } = renderStoryExplorerPage();

      // Select Italian language filter
      const languageSelect = screen.getByDisplayValue('All Languages');
      await user.click(languageSelect);

      const italianOption = screen.getByText('Italian');
      await user.click(italianOption);

      // Should only show Italian stories
      expect(screen.getByText('Italian Adventure Story')).toBeInTheDocument();
      expect(screen.queryByText('Spanish Mystery')).not.toBeInTheDocument();
    });

    it('filters stories by status', async () => {
      const { user } = renderStoryExplorerPage();

      // Select archived status filter
      const statusSelect = screen.getByDisplayValue('All Statuses');
      await user.click(statusSelect);

      const archivedOption = screen.getByText('Archived');
      await user.click(archivedOption);

      // Should only show archived stories
      expect(screen.getByText('Spanish Mystery')).toBeInTheDocument();
      expect(
        screen.queryByText('Italian Adventure Story')
      ).not.toBeInTheDocument();
    });

    it('searches stories by title', async () => {
      const { user } = renderStoryExplorerPage();

      // Type in search box
      const searchInput = screen.getByPlaceholderText('Search stories...');
      await user.type(searchInput, 'Adventure');

      // Should only show matching stories
      expect(screen.getByText('Italian Adventure Story')).toBeInTheDocument();
      expect(screen.queryByText('Spanish Mystery')).not.toBeInTheDocument();
    });

    it('clears all filters', async () => {
      const { user } = renderStoryExplorerPage();

      // Apply some filters first
      const searchInput = screen.getByPlaceholderText('Search stories...');
      await user.type(searchInput, 'Adventure');

      // Click clear all button
      const clearButton = screen.getByText('Clear All');
      await user.click(clearButton);

      // Should show all stories again
      expect(screen.getByText('Italian Adventure Story')).toBeInTheDocument();
      expect(screen.getByText('Spanish Mystery')).toBeInTheDocument();
    });
  });
});
