import React from 'react';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import DataExplorerPage from './DataExplorerPage';
import { ThemeProvider } from '../../contexts/ThemeContext';
import { useAuth } from '../../hooks/useAuth';
import {
  useAllQuestions,
  useReportedQuestions,
  useUsersPaginated,
  useUpdateQuestion,
  useAssignUsersToQuestion,
  useUnassignUsersFromQuestion,
  useMarkQuestionAsFixed,
  useFixQuestionWithAI,
  useClearUserDataForUser,
  useClearDatabase,
  useClearUserData,
  useUsersForQuestion,
} from '../../api/admin';

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
    data: ['english', 'spanish', 'french', 'german', 'italian'],
    isLoading: false,
    error: null,
  }),
  useGetV1SettingsLevels: () => ({
    data: {
      levels: ['A1', 'A2', 'B1', 'B2'],
      level_descriptions: {
        A1: 'Beginner',
        A2: 'Elementary',
        B1: 'Intermediate',
        B2: 'Upper intermediate',
      },
    },
    isLoading: false,
    error: null,
  }),
}));

const mockUseAuth = useAuth as ReturnType<typeof vi.fn>;
const mockUseAllQuestions = useAllQuestions as ReturnType<typeof vi.fn>;
const mockUseReportedQuestions = useReportedQuestions as ReturnType<
  typeof vi.fn
>;
const mockUseUsersPaginated = useUsersPaginated as ReturnType<typeof vi.fn>;
const mockUseUpdateQuestion = useUpdateQuestion as ReturnType<typeof vi.fn>;
const mockUseAssignUsersToQuestion = useAssignUsersToQuestion as ReturnType<
  typeof vi.fn
>;
const mockUseUnassignUsersFromQuestion =
  useUnassignUsersFromQuestion as ReturnType<typeof vi.fn>;
const mockUseMarkQuestionAsFixed = useMarkQuestionAsFixed as ReturnType<
  typeof vi.fn
>;
const mockUseFixQuestionWithAI = useFixQuestionWithAI as ReturnType<
  typeof vi.fn
>;
const mockUseClearUserDataForUser = useClearUserDataForUser as ReturnType<
  typeof vi.fn
>;
const mockUseClearDatabase = useClearDatabase as ReturnType<typeof vi.fn>;
const mockUseClearUserData = useClearUserData as ReturnType<typeof vi.fn>;
const mockUseUsersForQuestion = useUsersForQuestion as ReturnType<typeof vi.fn>;
const mockUseGetV1SettingsLanguages = vi.fn();
const mockUseGetV1SettingsLevels = vi.fn();

const mockQuestions = [
  {
    id: 1,
    type: 'multiple_choice',
    content: {
      question: 'What is the Italian word for "hello"?',
      options: ['Ciao', 'Grazie', 'Prego', 'Arrivederci'],
    },
    language: 'italian',
    level: 'A1',
    status: 'active',
    is_reported: false,
    user_count: 2,
  },
  {
    id: 2,
    type: 'fill_blank',
    content: {
      sentence: 'The cat ___ on the mat.',
      options: ['sits', 'sit', 'sitting', 'sat'],
    },
    language: 'english',
    level: 'A2',
    status: 'reported',
    is_reported: true,
    user_count: 1,
  },
];

const mockReportedQuestions = [
  {
    id: 2,
    type: 'fill_blank',
    content: {
      sentence: 'The cat ___ on the mat.',
      options: ['sits', 'sit', 'sitting', 'sat'],
    },
    language: 'english',
    level: 'A2',
    status: 'reported',
    is_reported: true,
    user_count: 1,
  },
];

const mockUsers = [
  {
    id: 1,
    username: 'testuser1',
    email: 'test1@example.com',
    language: 'italian',
    level: 'A1',
  },
  {
    id: 2,
    username: 'testuser2',
    email: 'test2@example.com',
    language: 'english',
    level: 'A2',
  },
];

const renderDataExplorerPage = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <MantineProvider>
          <BrowserRouter
            future={{
              v7_startTransition: false,
              v7_relativeSplatPath: false,
            }}
          >
            <DataExplorerPage />
          </BrowserRouter>
        </MantineProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
};

describe('DataExplorerPage', () => {
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
    mockUseAllQuestions.mockReturnValue({
      data: mockQuestions,
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseReportedQuestions.mockReturnValue({
      data: mockReportedQuestions,
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseUsersPaginated.mockReturnValue({
      data: { users: mockUsers, total: 2 },
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseUpdateQuestion.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });

    mockUseAssignUsersToQuestion.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });

    mockUseUnassignUsersFromQuestion.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });

    mockUseMarkQuestionAsFixed.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });

    mockUseFixQuestionWithAI.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });

    mockUseClearUserDataForUser.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });

    mockUseClearDatabase.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });

    mockUseClearUserData.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    });

    mockUseUsersForQuestion.mockReturnValue({
      data: mockUsers,
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseGetV1SettingsLanguages.mockReturnValue({
      data: ['english', 'italian', 'spanish'],
      isLoading: false,
      isFetching: false,
      error: null,
    });

    mockUseGetV1SettingsLevels.mockReturnValue({
      data: {
        levels: ['A1', 'A2', 'B1', 'B2'],
        level_descriptions: {
          A1: 'Beginner',
          A2: 'Elementary',
          B1: 'Intermediate',
          B2: 'Upper Intermediate',
        },
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
      expect(() => renderDataExplorerPage()).not.toThrow();
    });

    it('renders the page title', () => {
      renderDataExplorerPage();
      expect(screen.getByText('Data Explorer')).toBeInTheDocument();
    });

    it('renders action buttons', () => {
      renderDataExplorerPage();
      expect(screen.getByText('Clear User Data')).toBeInTheDocument();
      expect(screen.getByText('Clear Database')).toBeInTheDocument();
    });
  });

  describe('Loading States', () => {
    it('shows loader when questions are loading', () => {
      mockUseAllQuestions.mockReturnValue({
        data: undefined,
        isLoading: true,
        isFetching: false,
        error: null,
      });
      renderDataExplorerPage();
      expect(screen.getByTestId('loader')).toBeInTheDocument();
    });

    it('shows loader when reported questions are loading', () => {
      mockUseReportedQuestions.mockReturnValue({
        data: undefined,
        isLoading: true,
        isFetching: false,
        error: null,
      });
      renderDataExplorerPage();
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
      renderDataExplorerPage();
      expect(screen.queryByText('Data Explorer')).not.toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('handles API errors gracefully', () => {
      mockUseAllQuestions.mockReturnValue({
        data: undefined,
        isLoading: false,
        isFetching: false,
        error: new Error('API Error'),
      });
      renderDataExplorerPage();
      expect(screen.getByText('Error')).toBeInTheDocument();
      expect(
        screen.getByText(/Failed to load data explorer data/)
      ).toBeInTheDocument();
    });
  });
});
