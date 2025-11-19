import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import userEvent from '@testing-library/user-event';
import { TranslationPopup } from '../TranslationPopup';
import { AuthProvider } from '../../contexts/AuthProvider';
import { TranslationProvider } from '../../contexts/TranslationContext';
import { ThemeProvider } from '../../contexts/ThemeContext';
import { defaultVoiceForLanguage } from '../../utils/tts';

// Mock dependencies
vi.mock('../../hooks/useAuth', () => ({
  useAuth: () => ({
    user: { id: 1 },
    isLoading: false,
  }),
}));

const mockPlayTTS = vi.fn();
vi.mock('../../hooks/useTTS', () => ({
  useTTS: () => ({
    isLoading: false,
    isPlaying: false,
    isPaused: false,
    playTTS: mockPlayTTS,
    pauseTTS: vi.fn(),
    resumeTTS: vi.fn(),
    restartTTS: vi.fn(),
    stopTTS: vi.fn(),
    currentText: null,
    currentKey: null,
  }),
}));

// Mock API to control user preferences
const mockPreferences = {
  tts_voice: 'es-ES-ElviraNeural', // User prefers Spanish voice
};

vi.mock('../../api/api', () => ({
  useGetV1AuthStatus: () => ({ data: { authenticated: true }, isLoading: false }),
  useGetV1SettingsLanguages: () => ({
    data: [
      { code: 'en', name: 'english' },
      { code: 'es', name: 'spanish' },
    ],
    isLoading: false,
  }),
  useGetV1PreferencesLearning: () => ({
    data: mockPreferences,
    isLoading: false,
  }),
  usePostV1Snippets: () => ({ mutateAsync: vi.fn() }),
  postV1Snippets: vi.fn(),
  useGetV1SettingsAiProviders: () => ({ data: { providers: [] } }),
  useGetV1SettingsLevels: () => ({ data: { levels: [] } }),
  useGetV1SettingsApiKeyAvailability: () => ({ data: { has_api_key: true } }),
  usePostV1AuthLogin: () => ({ mutateAsync: vi.fn() }),
  usePostV1AuthLogout: () => ({ mutateAsync: vi.fn() }),
  usePutV1Settings: () => ({ mutateAsync: vi.fn() }),
}));

// Mock Translation Context
vi.mock('../../contexts/TranslationContext', () => ({
  TranslationProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  useTranslation: () => ({
    translateText: vi.fn().mockResolvedValue({
      translatedText: 'Hello world',
      sourceLanguage: 'es',
      targetLanguage: 'en',
    }),
    translation: {
      translatedText: 'Hello world',
      sourceLanguage: 'es',
      targetLanguage: 'en',
    },
    isLoading: false,
    error: null,
  }),
}));

vi.mock('../../hooks/useTextSelection', () => ({
  useTextSelection: () => ({
    selection: { text: 'Hola mundo' },
    isVisible: true,
    clearSelection: vi.fn(),
  }),
}));

vi.mock('../../contexts/ThemeContext', () => ({
  useTheme: () => ({ fontSize: 'md' }),
  ThemeProvider: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

const TestWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MantineProvider>
      <Notifications />
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <ThemeProvider>
            <AuthProvider>
              <TranslationProvider>{children}</TranslationProvider>
            </AuthProvider>
          </ThemeProvider>
        </BrowserRouter>
      </QueryClientProvider>
    </MantineProvider>
  );
};

describe('TranslationPopup Voice Selection', () => {
  beforeEach(() => {
    mockPlayTTS.mockClear();
  });

  it('should use English voice for English translation even if user prefers Spanish voice', async () => {
    const user = userEvent.setup();

    render(
      <TestWrapper>
        <TranslationPopup
          selection={{ text: 'Hola mundo', x: 0, y: 0, width: 0, height: 0 }}
          onClose={vi.fn()}
        />
      </TestWrapper>
    );

    // Wait for translation to appear
    await waitFor(() => expect(screen.getByText('Hello world')).toBeInTheDocument());

    // Find the TTS button for the translation (English)
    // The label is "Listen to translation"
    const translatedButton = screen.getByLabelText(/Listen to translation/i);
    await user.click(translatedButton);

    await waitFor(() => {
      expect(mockPlayTTS).toHaveBeenCalled();
    });

    const [textArg, voiceArg] = mockPlayTTS.mock.calls[0];
    expect(textArg).toBe('Hello world');

    // Crucial check: Voice should be English (default), NOT the user's preferred Spanish voice
    const expectedEnglishVoice = defaultVoiceForLanguage('english');
    expect(voiceArg).toBe(expectedEnglishVoice);
    expect(voiceArg).not.toBe(mockPreferences.tts_voice);
  });
});
