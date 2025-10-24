import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import StoryPage from '../StoryPage';
import { ThemeProvider } from '../../contexts/ThemeContext';

// Mock the story hook
const mockUseStory = {
  currentStory: {
    id: 1,
    title: 'Existing Story',
    sections: [],
    status: 'active',
    language: 'en',
    is_current: true,
  },
  archivedStories: [],
  sections: [],
  currentSectionIndex: 0,
  viewMode: 'section' as const,
  isLoading: false,
  isLoadingArchivedStories: false,
  error: null,
  hasCurrentStory: true,
  currentSection: null,
  currentSectionWithQuestions: null,
  canGenerateToday: true,
  isGenerating: false,
  generationType: 'story' as const,
  generationDisabledReason: '',
  createStory: vi.fn(),
  archiveStory: vi.fn(),
  setCurrentStory: vi.fn(),
  generateNextSection: vi.fn(),
  exportStoryPDF: vi.fn(),
  goToNextSection: vi.fn(),
  goToPreviousSection: vi.fn(),
  goToFirstSection: vi.fn(),
  goToLastSection: vi.fn(),
  setViewMode: vi.fn(),
  generationErrorModal: { isOpen: false, errorMessage: '', errorDetails: '' },
  closeGenerationErrorModal: vi.fn(),
};

vi.mock('../../hooks/useStory', () => ({
  useStory: () => mockUseStory,
}));

// Silence TTS hooks used by children
vi.mock('../../hooks/useTTS', () => ({
  useTTS: () => ({
    isLoading: false,
    isPlaying: false,
    playTTS: vi.fn(),
    stopTTS: vi.fn(),
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

// Preferences hook used by children
vi.mock('../../api/api', async () => {
  const actual = await vi.importActual('../../api/api');
  return {
    ...actual,
    useGetV1PreferencesLearning: vi.fn(() => ({ data: {} })),
  };
});

const renderPage = () =>
  render(
    <ThemeProvider>
      <MantineProvider>
        <StoryPage />
      </MantineProvider>
    </ThemeProvider>
  );

describe('StoryPage - New Story button', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('opens Create Story modal when New Story is clicked', () => {
    renderPage();

    const btn = screen.getByRole('button', { name: /new story/i });
    fireEvent.click(btn);

    const dialog = screen.getByRole('dialog');
    expect(dialog).toBeInTheDocument();
    // Ensure the form rendered by checking for a labeled input
    expect(screen.getByLabelText(/Story Title/i)).toBeInTheDocument();
  });
});
