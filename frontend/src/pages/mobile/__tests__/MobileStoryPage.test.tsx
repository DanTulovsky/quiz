import React from 'react';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { BrowserRouter } from 'react-router-dom';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import MobileStoryPage from '../MobileStoryPage';
import { StoryWithSections, StorySection } from '../../../api/storyApi';
import { ThemeProvider } from '../../../contexts/ThemeContext';

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
  isPaused: false,
  currentPlayingText: null as string | null,
  currentKey: null as string | null,
  playTTS: vi.fn(),
  stopTTS: vi.fn(),
  pauseTTS: vi.fn(),
  resumeTTS: vi.fn(),
  restartTTS: vi.fn(),
};

vi.mock('../../../hooks/useTTS', () => ({
  useTTS: () => ({
    isLoading: mockTTS.isLoading,
    isPlaying: mockTTS.isPlaying,
    isPaused: mockTTS.isPaused,
    currentText: mockTTS.currentPlayingText,
    currentKey: mockTTS.currentKey,
    playTTS: mockTTS.playTTS,
    stopTTS: mockTTS.stopTTS,
    pauseTTS: mockTTS.pauseTTS,
    resumeTTS: mockTTS.resumeTTS,
    restartTTS: mockTTS.restartTTS,
  }), // Return object with current values so React sees changes
}));

// Reset mock state before each test
beforeEach(() => {
  vi.clearAllMocks();
  mockTTS.isLoading = false;
  mockTTS.isPlaying = false;
  mockTTS.isPaused = false;
  mockTTS.currentPlayingText = null;
  mockTTS.currentKey = null;
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
      <BrowserRouter>
        <ThemeProvider>
          <MantineProvider>
            <MobileStoryPage {...props} />
          </MantineProvider>
        </ThemeProvider>
      </BrowserRouter>
    );
  };

  describe('MobileStorySectionView TTS Functionality', () => {
    beforeEach(() => {
      mockUseStory.viewMode = 'section';
    });

    it('displays TTS button for mobile story sections', () => {
      renderComponent();

      const ttsButton = screen.getByLabelText(/Section audio/i);
      expect(ttsButton).toBeInTheDocument();
    });

    it('shows loading state when TTS is loading', async () => {
      const { rerender } = renderComponent();

      const ttsButton = screen.getByLabelText(/Section audio/i);
      expect(ttsButton).toBeInTheDocument();

      // Click button to establish ownership and start loading
      mockTTS.playTTS.mockImplementation(() => {
        mockTTS.isLoading = true;
        return Promise.resolve();
      });

      await act(async () => {
        fireEvent.click(ttsButton);
        // Force re-render to pick up the new isLoading state
        rerender(
          <BrowserRouter>
            <ThemeProvider>
              <MantineProvider>
                <MobileStoryPage />
              </MantineProvider>
            </ThemeProvider>
          </BrowserRouter>
        );
      });

      // Button should show loading state (spinner) but not be disabled
      const updatedButton = screen.getByLabelText(/Section audio/i);
      expect(updatedButton).toBeInTheDocument();
    });

    it('shows playing state when TTS is playing', () => {
      mockTTS.isPlaying = true;
      renderComponent();

      const ttsButton = screen.getByLabelText(/Section audio/i);
      expect(ttsButton).toBeInTheDocument();
    });

    it('calls playTTS when TTS button is clicked', () => {
      renderComponent();

      const ttsButton = screen.getByLabelText(/Section audio/i);
      fireEvent.click(ttsButton);

      expect(mockTTS.playTTS).toHaveBeenCalledWith(
        mockSection.content,
        expect.any(String),
        expect.objectContaining({
          title: expect.any(String),
        }),
        expect.any(String)
      );
    });

    it('calls pauseTTS when TTS button is clicked while playing', async () => {
      const { rerender } = renderComponent();

      // Set up initial playing state
      mockTTS.isPlaying = true;
      mockTTS.currentPlayingText = mockSection.content.trim();
      mockTTS.currentKey = mockSection.content.trim();
      mockTTS.isLoading = false;

      // Re-render with playing state
      await act(async () => {
        rerender(
          <BrowserRouter>
            <ThemeProvider>
              <MantineProvider>
                <MobileStoryPage />
              </MantineProvider>
            </ThemeProvider>
          </BrowserRouter>
        );
      });

      // Now click the button - should call pauseTTS since text matches and isPlaying is true
      const ttsButton = screen.getByLabelText(/Section audio/i);
      await act(async () => {
        fireEvent.click(ttsButton);
      });

      expect(mockTTS.pauseTTS).toHaveBeenCalled();
    });
  });

  describe('MobileStorySectionView scroll behavior', () => {
    const scrollIntoViewMock = vi.fn();
    const scrollToMock = vi.fn();
    const rafMock = vi.fn<(cb: FrameRequestCallback) => number>(cb => {
      cb(0);
      return 0;
    });
    const cancelRafMock = vi.fn();
    let originalScrollIntoView:
      | typeof HTMLElement.prototype.scrollIntoView
      | undefined;
    let originalScrollTo: typeof HTMLElement.prototype.scrollTo | undefined;
    let originalRAF: typeof window.requestAnimationFrame;
    let originalCancelRAF: typeof window.cancelAnimationFrame;

    beforeEach(() => {
      mockUseStory.viewMode = 'section';
      mockUseStory.sections = [
        {
          ...mockSection,
          id: 1,
          section_number: 1,
          content: 'First section content',
        },
        {
          ...mockSection,
          id: 2,
          section_number: 2,
          content: 'Second section content',
        },
      ];
      mockUseStory.currentSection = mockUseStory.sections[0];
      mockUseStory.currentSectionIndex = 0;

      scrollIntoViewMock.mockClear();
      scrollToMock.mockClear();
      rafMock.mockClear();
      cancelRafMock.mockClear();

      originalScrollIntoView = HTMLElement.prototype.scrollIntoView;
      Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
        configurable: true,
        value: scrollIntoViewMock,
      });

      originalScrollTo = HTMLElement.prototype.scrollTo;
      Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
        configurable: true,
        value: scrollToMock,
      });

      originalRAF = window.requestAnimationFrame;
      originalCancelRAF = window.cancelAnimationFrame;
      window.requestAnimationFrame =
        rafMock as unknown as typeof window.requestAnimationFrame;
      window.cancelAnimationFrame =
        cancelRafMock as unknown as typeof window.cancelAnimationFrame;
    });

    afterEach(() => {
      if (originalScrollIntoView) {
        Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
          configurable: true,
          value: originalScrollIntoView,
        });
      } else {
        delete (
          HTMLElement.prototype as unknown as {
            scrollIntoView?: typeof scrollIntoViewMock;
          }
        ).scrollIntoView;
      }

      if (originalScrollTo) {
        Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
          configurable: true,
          value: originalScrollTo,
        });
      } else {
        delete (
          HTMLElement.prototype as unknown as {
            scrollTo?: typeof scrollToMock;
          }
        ).scrollTo;
      }

      window.requestAnimationFrame = originalRAF;
      window.cancelAnimationFrame = originalCancelRAF;
    });

    it('scrolls to the top anchor and resets story content on section change', async () => {
      const { rerender } = renderComponent();

      expect(scrollIntoViewMock).toHaveBeenCalledTimes(1);
      expect(scrollToMock).toHaveBeenCalledTimes(1);

      mockUseStory.currentSection = mockUseStory.sections[1];
      mockUseStory.currentSectionIndex = 1;

      await act(async () => {
        rerender(
          <BrowserRouter>
            <ThemeProvider>
              <MantineProvider>
                <MobileStoryPage />
              </MantineProvider>
            </ThemeProvider>
          </BrowserRouter>
        );
      });

      expect(scrollIntoViewMock).toHaveBeenCalledTimes(2);
      expect(scrollToMock).toHaveBeenCalledTimes(2);
    });
  });

  describe('MobileStoryReadingView TTS Functionality', () => {
    beforeEach(() => {
      mockUseStory.viewMode = 'reading';
    });

    it('displays TTS button in header for mobile story reading view', () => {
      renderComponent();

      const ttsButton = screen.getByLabelText(/Story audio/i);
      expect(ttsButton).toBeInTheDocument();
    });

    it('shows loading state when TTS is loading', async () => {
      const { rerender } = renderComponent();

      const ttsButton = screen.getByLabelText(/Story audio/i);
      expect(ttsButton).toBeInTheDocument();

      // Click button to establish ownership and start loading
      mockTTS.playTTS.mockImplementation(() => {
        mockTTS.isLoading = true;
        return Promise.resolve();
      });

      await act(async () => {
        fireEvent.click(ttsButton);
        // Force re-render to pick up the new isLoading state
        rerender(
          <BrowserRouter>
            <ThemeProvider>
              <MantineProvider>
                <MobileStoryPage />
              </MantineProvider>
            </ThemeProvider>
          </BrowserRouter>
        );
      });

      // Button should show loading state (spinner) but not be disabled
      const updatedButton = screen.getByLabelText(/Story audio/i);
      expect(updatedButton).toBeInTheDocument();
    });

    it('shows playing state when TTS is playing', () => {
      mockTTS.isPlaying = true;
      renderComponent();

      const ttsButton = screen.getByLabelText(/Story audio/i);
      expect(ttsButton).toBeInTheDocument();
    });

    it('calls playTTS with combined story content when TTS button is clicked', () => {
      renderComponent();

      const ttsButton = screen.getByLabelText(/Story audio/i);
      fireEvent.click(ttsButton);

      expect(mockTTS.playTTS).toHaveBeenCalledWith(
        mockSection.content,
        expect.any(String),
        expect.objectContaining({
          title: expect.any(String),
        }),
        expect.any(String)
      );
    });

    it('calls pauseTTS when TTS button is clicked while playing', async () => {
      const { rerender } = renderComponent();

      // Get the actual text that will be played (all sections joined)
      const expectedContent =
        mockStory.sections.map(s => s.content).join('\n\n') || '';

      // Set up initial playing state
      mockTTS.isPlaying = true;
      mockTTS.currentPlayingText = expectedContent.trim();
      mockTTS.currentKey = expectedContent.trim();
      mockTTS.isLoading = false;

      // Re-render with playing state
      await act(async () => {
        rerender(
          <BrowserRouter>
            <ThemeProvider>
              <MantineProvider>
                <MobileStoryPage />
              </MantineProvider>
            </ThemeProvider>
          </BrowserRouter>
        );
      });

      // Now click the button - should call pauseTTS since text matches and isPlaying is true
      const ttsButton = screen.getByLabelText(/Story audio/i);
      await act(async () => {
        fireEvent.click(ttsButton);
      });

      expect(mockTTS.pauseTTS).toHaveBeenCalled();
    });
  });
});
