// Type definitions for TTS.Rocks library (from https://tts.rocks/tts.js)

declare global {
  interface Window {
    TTS: {
      // Provider settings
      TTSProvider: string;
      rate: number;
      pitch: number;
      speech: boolean;
      useKokoroTTS?: boolean;
      usePiper?: boolean;
      useEspeak?: boolean;
      useKitten?: boolean;
      voices?: SpeechSynthesisVoice[] | null;

      // OpenAI settings
      openAISettings: {
        apiKey: string | false;
        endpoint: string;
        model: string;
        voice: string;
        responseFormat: string;
        speed: number;
      };
      OpenAIAPIKey: string | false;

      // Audio element
      audio: HTMLAudioElement | null;

      // State
      isPlaying: boolean;
      isPaused: boolean;
      isLoading: boolean;
      premiumQueueActive: boolean;

      // Methods
      speak: (text: string, autoplay?: boolean) => void | Promise<void>;
      openAITTS?: (text: string) => void;
      // Note: TTS.Rocks doesn't provide stop/pause/resume methods,
      // we control the audio element directly

      // Initialization
      initOpenAI?: () => Promise<void>;
      finishedAudio?: () => void;
    };
    // Internal TTS state management (used by useTTS hook)
    __ttsStateCheckInterval?: NodeJS.Timeout;
    __ttsCleanup?: () => void;
    TTS_LOADED?: boolean;
  }
}

export {};
