import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import MobileVerbConjugationPage from '../MobileVerbConjugationPage';
import { ThemeProvider } from '../../../contexts/ThemeContext';
import * as verbConjugationsUtils from '../../../utils/verbConjugations';

// Mock fetch globally
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Mock the useAuth hook
vi.mock('../../../hooks/useAuth', () => ({
  useAuth: () => ({
    user: {
      id: '1',
      email: 'test@example.com',
      preferred_language: 'italian',
    },
    isAuthenticated: true,
  }),
}));

// Mock the verb conjugations utilities
vi.mock('../../../utils/verbConjugations', () => ({
  loadVerbConjugations: vi.fn(),
  loadVerbConjugation: vi.fn(),
}));

// Mock the API hooks
vi.mock('../../../api/api', () => ({
  useGetV1SettingsLanguages: vi.fn(() => ({
    data: [
      { name: 'Italian', code: 'it' },
      { name: 'Spanish', code: 'es' },
    ],
    isLoading: false,
    error: null,
  })),
  useGetV1PreferencesLearning: vi.fn(() => ({
    data: { tts_voice: 'it-IT-TestVoice' },
    isLoading: false,
    error: null,
  })),
}));

// Mock the TTS utilities
vi.mock('../../../utils/tts', () => ({
  defaultVoiceForLanguage: vi.fn(() => 'it-IT-DefaultVoice'),
}));

// Mock the HoverTranslation component
vi.mock('../../../components/HoverTranslation', () => ({
  HoverTranslation: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

// Mock the TTSButton component
vi.mock('../../../components/TTSButton', () => ({
  default: ({
    'aria-label': ariaLabel,
  }: {
    'aria-label'?: string;
  }) => <button aria-label={ariaLabel}>TTS</button>,
}));

const mockVerbConjugations = {
  language: 'it',
  languageName: 'Italian',
  verbs: [
    {
      infinitive: 'essere',
      infinitiveEn: 'to be',
      category: 'irregular',
      tenses: [],
    },
    {
      infinitive: 'avere',
      infinitiveEn: 'to have',
      category: 'irregular',
      tenses: [],
    },
  ],
};

const mockVerbData = {
  infinitive: 'essere',
  infinitiveEn: 'to be',
  category: 'irregular',
  tenses: [
    {
      tenseId: 'presente',
      tenseName: 'Presente',
      tenseNameEn: 'Present',
      description: 'Present tense description',
      conjugations: [
        {
          pronoun: 'io',
          form: 'sono',
          exampleSentence: 'Io sono felice.',
          exampleSentenceEn: 'I am happy.',
        },
      ],
    },
  ],
};

const renderComponent = () => {
  return render(
    <BrowserRouter>
      <ThemeProvider>
        <MantineProvider>
          <MobileVerbConjugationPage />
        </MantineProvider>
      </ThemeProvider>
    </BrowserRouter>
  );
};

describe('MobileVerbConjugationPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(verbConjugationsUtils.loadVerbConjugations).mockClear();
    vi.mocked(verbConjugationsUtils.loadVerbConjugation).mockClear();
  });

  it('renders the page title', async () => {
    vi.mocked(verbConjugationsUtils.loadVerbConjugations).mockResolvedValue(mockVerbConjugations);
    vi.mocked(verbConjugationsUtils.loadVerbConjugation).mockResolvedValue(mockVerbData);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByText('Verb Conjugations')).toBeInTheDocument();
    });
  });

  it('loads and displays available verbs', async () => {
    vi.mocked(verbConjugationsUtils.loadVerbConjugations).mockResolvedValue(mockVerbConjugations);
    vi.mocked(verbConjugationsUtils.loadVerbConjugation).mockResolvedValue(mockVerbData);

    renderComponent();

    await waitFor(() => {
      expect(verbConjugationsUtils.loadVerbConjugations).toHaveBeenCalledWith('it');
    });

    await waitFor(() => {
      expect(screen.getByText(/2 VERBS/i)).toBeInTheDocument();
    });
  });
});

