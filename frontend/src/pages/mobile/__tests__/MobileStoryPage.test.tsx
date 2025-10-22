import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import MobileStoryPage from '../MobileStoryPage';
import { StoryWithSections, StorySection } from '../../../api/storyApi';

// Mock the story hook
const mockUseStory = {
  currentStory: null,
  archivedStories: [],
  sections: [],
  currentSectionIndex: 0,
  viewMode: 'section' as const,
  isLoading: false,
  isLoadingArchivedStories: false,
  error: null,
  hasCurrentStory: false,
  currentSection: null,
  currentSectionWithQuestions: null,
  canGenerateToday: true,
  isGenerating: false,
  generationDisabledReason: '',
  createStory: vi.fn(),
  archiveStory: vi.fn(),
  setCurrentStory: vi.fn(),
  generateNextSection: vi.fn(),
  goToNextSection: vi.fn(),
  goToPreviousSection: vi.fn(),
  goToFirstSection: vi.fn(),
  goToLastSection: vi.fn(),
  setViewMode: vi.fn(),
  generationErrorModal: { isOpen: false, errorMessage: '' },
  closeGenerationErrorModal: vi.fn(),
};

vi.mock('../../../hooks/useStory', () => ({
  useStory: () => mockUseStory,
}));

// Mock the TTS hook
const mockTTS = {
  isLoading: false,
  isPlaying: false,
  playTTS: vi.fn(),
  stopTTS: vi.fn(),
  isBuffering: false,
  bufferingProgress: 0,
};

vi.mock('../../../hooks/useTTS', () => ({
  useTTS: () => mockTTS,
}));

// Reset mock state before each test
beforeEach(() => {
  vi.clearAllMocks();
  mockTTS.isLoading = false;
  mockTTS.isPlaying = false;
  mockTTS.playTTS.mockClear();
  mockTTS.stopTTS.mockClear();
});

// Mock the TTS utils
vi.mock('../../../utils/tts', () => ({
  defaultVoiceForLanguage: vi.fn(() => 'en-US-Default'),
}));

// Mock the snippet hooks
vi.mock('../../../hooks/useSectionSnippets', () => ({
  useSectionSnippets: () => ({
    snippets: [],
    isLoading: false,
    error: null,
  }),
}));

vi.mock('../../../hooks/useStorySnippets', () => ({
  useStorySnippets: () => ({
    snippets: [],
    isLoading: false,
    error: null,
  }),
}));

// Mock SnippetHighlighter component
vi.mock('../../../components/SnippetHighlighter', () => ({
  SnippetHighlighter: ({
    text,
    component: Component,
    componentProps,
  }: {
    text: string;
    component?: React.ElementType;
    componentProps?: Record<string, unknown>;
  }) => {
    const ComponentToRender = Component || 'span';
    return <ComponentToRender {...componentProps}>{text}</ComponentToRender>;
  },
}));

// Mock the API hooks for learning preferences
vi.mock('../../../api/api', async () => {
  const actual = await vi.importActual('../../../api/api');
  return {
    ...actual,
    useGetV1PreferencesLearning: vi.fn(() => ({
      data: { tts_voice: 'en-US-TestVoice' },
    })),
  };
});

describe('MobileStoryPage', () => {
  const mockSection: StorySection = {
    id: 1,
    story_id: 1,
    section_number: 1,
    content: 'This is the first section of the story.',
    language_level: 'intermediate',
    word_count: 8,
    generated_at: '2025-01-15T10:30:00Z',
    generation_date: '2025-01-15',
  };

  const mockStory: StoryWithSections = {
    id: 1,
    title: 'Test Story',
    language: 'en',
    status: 'active',
    is_current: true,
    sections: [mockSection],
  };

  beforeEach(() => {
    mockUseStory.currentStory = mockStory;
    mockUseStory.hasCurrentStory = true;
    mockUseStory.currentSection = mockSection;
    mockUseStory.sections = [mockSection];
    mockUseStory.currentSectionIndex = 0;
  });

  const renderComponent = (props = {}) => {
    return render(
      <MantineProvider>
        <MobileStoryPage {...props} />
      </MantineProvider>
    );
  };

  describe('MobileStorySectionView TTS Functionality', () => {
    beforeEach(() => {
      mockUseStory.viewMode = 'section';
    });

    it('displays TTS button for mobile story sections', () => {
      renderComponent();

      const ttsButton = screen.getByLabelText('Listen to section');
      expect(ttsButton).toBeInTheDocument();
    });

    it('shows loading state when TTS is loading', () => {
      mockTTS.isLoading = true;
      renderComponent();

      const ttsButton = screen.getByLabelText('Loading audio');
      expect(ttsButton).toBeInTheDocument();
      expect(ttsButton).toBeDisabled();
    });

    it('shows playing state when TTS is playing', () => {
      mockTTS.isPlaying = true;
      renderComponent();

      const ttsButton = screen.getByLabelText('Stop audio');
      expect(ttsButton).toBeInTheDocument();
    });

    it('calls playTTS when TTS button is clicked', () => {
      renderComponent();

      const ttsButton = screen.getByLabelText('Listen to section');
      fireEvent.click(ttsButton);

      expect(mockTTS.playTTS).toHaveBeenCalledWith(
        mockSection.content,
        expect.any(String)
      );
    });

    it('calls stopTTS when TTS button is clicked while playing', () => {
      mockTTS.isPlaying = true;
      renderComponent();

      const ttsButton = screen.getByLabelText('Stop audio');
      fireEvent.click(ttsButton);

      expect(mockTTS.stopTTS).toHaveBeenCalled();
    });
  });

  describe('MobileStoryReadingView TTS Functionality', () => {
    beforeEach(() => {
      mockUseStory.viewMode = 'reading';
    });

    it('displays TTS button in header for mobile story reading view', () => {
      renderComponent();

      const ttsButton = screen.getByLabelText('Listen to story');
      expect(ttsButton).toBeInTheDocument();
    });

    it('shows loading state when TTS is loading', () => {
      mockTTS.isLoading = true;
      renderComponent();

      const ttsButton = screen.getByLabelText('Loading audio');
      expect(ttsButton).toBeInTheDocument();
      expect(ttsButton).toBeDisabled();
    });

    it('shows playing state when TTS is playing', () => {
      mockTTS.isPlaying = true;
      renderComponent();

      const ttsButton = screen.getByLabelText('Stop audio');
      expect(ttsButton).toBeInTheDocument();
    });

    it('calls playTTS with combined story content when TTS button is clicked', () => {
      renderComponent();

      const ttsButton = screen.getByLabelText('Listen to story');
      fireEvent.click(ttsButton);

      expect(mockTTS.playTTS).toHaveBeenCalledWith(
        mockSection.content,
        expect.any(String)
      );
    });

    it('calls stopTTS when TTS button is clicked while playing', () => {
      mockTTS.isPlaying = true;
      renderComponent();

      const ttsButton = screen.getByLabelText('Stop audio');
      fireEvent.click(ttsButton);

      expect(mockTTS.stopTTS).toHaveBeenCalled();
    });
  });
});
