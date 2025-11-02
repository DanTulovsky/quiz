// Reference to clear function - set by useTTS module
let clearSharedTTSStateFn: (() => void) | null = null;

/**
 * Register the clear function from useTTS module
 * This allows finishedAudio to clear shared state directly
 */
export function registerClearTTSState(fn: () => void) {
  clearSharedTTSStateFn = fn;
}

/**
 * Initialize TTS.Rocks library with our backend configuration
 * This should be called once when the app starts
 */
export function initializeTTSRocks(): void {
  if (typeof window === 'undefined') {
    console.warn('TTS.Rocks: window is undefined');
    return;
  }

  if (!window.TTS) {
    console.warn('TTS.Rocks library not loaded yet');
    return;
  }

  // Configure TTS.Rocks to use our backend endpoint
  window.TTS.TTSProvider = 'openai';

  // Set a dummy API key so TTS.Rocks doesn't skip OpenAI mode
  // (it checks for OpenAIAPIKey before calling openAITTS)
  // Our backend doesn't actually need it since auth is via cookies
  window.TTS.OpenAIAPIKey = 'dummy-key-for-custom-endpoint';

  window.TTS.openAISettings = {
    apiKey: false, // Backend handles auth via cookies, but we set OpenAIAPIKey above
    endpoint: '/v1/audio/speech', // Our backend endpoint
    model: 'tts-1',
    voice: 'alloy', // Default, can be overridden
    responseFormat: 'mp3', // Our backend returns mp3 (not SSE for early playback)
    speed: 1.0,
  };

  // Ensure we request mp3 format (not SSE) for early playback
  // The browser audio element will start playing as soon as enough data is buffered
  // SSE would require custom streaming logic which TTS.Rocks doesn't support
  // Patch TTS.Rocks's openAITTS function to include stream_format in the request
  if (window.TTS.openAITTS) {
    const originalOpenAITTS = window.TTS.openAITTS;
    window.TTS.openAITTS = function (text: string) {
      // Call original, but patch the fetch call if possible
      // Note: TTS.Rocks constructs the fetch internally, so we'll need to
      // intercept at the fetch level or modify the request body post-construction
      return originalOpenAITTS.call(window.TTS, text);
    };
  }

  // Intercept fetch calls to /v1/audio/speech to add stream_format parameter
  // This ensures we get mp3 format (not SSE) for early playback
  // The browser audio element will start playing as soon as enough data is buffered
  const originalFetch = window.fetch;
  window.fetch = function (
    input: RequestInfo | URL,
    init?: RequestInit
  ): Promise<Response> {
    // Extract URL - TTS.Rocks typically uses string URLs
    const url =
      typeof input === 'string'
        ? input
        : input instanceof URL
          ? input.toString()
          : input instanceof Request
            ? input.url
            : '';

    // Check if this is a TTS request
    const isTTSRequest =
      url.includes('/v1/audio/speech') &&
      (init?.method === 'POST' ||
        (input instanceof Request && input.method === 'POST'));

    if (isTTSRequest && init?.body && typeof init.body === 'string') {
      try {
        // Parse and modify the request body to include stream_format
        const bodyObj = JSON.parse(init.body);
        // Add stream_format: 'mp3' to get direct mp3 response (not SSE)
        // This allows the browser audio element to start playing as soon as enough data is buffered
        bodyObj.stream_format = 'mp3';

        // Create new init with modified body
        const modifiedInit: RequestInit = {
          ...init,
          body: JSON.stringify(bodyObj),
          headers: {
            ...init.headers,
            'Content-Type': 'application/json',
          },
        };

        return originalFetch.call(window, input, modifiedInit);
      } catch (e) {
        console.warn('Failed to patch TTS request body:', e);
        // Fall through to original fetch
      }
    }

    // For Request objects, we'd need async handling which is complex
    // TTS.Rocks should use string URLs, so this should work
    return originalFetch.call(window, input, init);
  };

  // Disable other TTS providers to ensure we use OpenAI
  window.TTS.useKokoroTTS = false;
  window.TTS.usePiper = false;
  window.TTS.useEspeak = false;
  window.TTS.useKitten = false;

  // Ensure we don't fall back to system TTS
  // Clear voices so system TTS isn't used
  window.TTS.voices = null;

  // Enable speech by default
  window.TTS.speech = true;

  // Set up audio element if not already created
  // Note: TTS.Rocks will create its own audio element in openAITTS,
  // but we ensure one exists for initialization
  if (!window.TTS.audio) {
    window.TTS.audio = document.createElement('audio');
  }

  // Store the original finishedAudio if it exists (before we overwrite it)
  const originalFinishedAudio = window.TTS.finishedAudio;

  // Set up finishedAudio callback that triggers our state updates
  // This will be called when audio.onended fires (set by TTS.Rocks)
  window.TTS.finishedAudio = function () {
    // Call original callback if it exists
    if (
      originalFinishedAudio &&
      typeof originalFinishedAudio === 'function' &&
      originalFinishedAudio !== window.TTS.finishedAudio
    ) {
      try {
        originalFinishedAudio();
      } catch (e) {
        console.warn('Error in original finishedAudio:', e);
      }
    }

    window.TTS.isPlaying = false;
    window.TTS.isPaused = false;
    window.TTS.isLoading = false;
    if (window.TTS.audio) {
      window.TTS.audio.pause();
      if (window.TTS.audio.src && window.TTS.audio.src.startsWith('blob:')) {
        try {
          URL.revokeObjectURL(window.TTS.audio.src);
        } catch {}
      }
      window.TTS.audio.src = '';
    }

    // Clear shared state directly - this ensures currentPlayingText becomes null
    // This is a backup in case the event listener isn't attached or component unmounted
    if (clearSharedTTSStateFn) {
      clearSharedTTSStateFn();
    }

    // Trigger custom event so our hook can listen
    if (typeof window !== 'undefined' && window.dispatchEvent) {
      window.dispatchEvent(new CustomEvent('tts-finished'));
    }
  };

  // Ensure the audio element has onended handler set
  if (window.TTS.audio) {
    window.TTS.audio.onended = window.TTS.finishedAudio;
  }

  // Initialize state
  window.TTS.isPlaying = false;
  window.TTS.isPaused = false;
  window.TTS.isLoading = false;
}
