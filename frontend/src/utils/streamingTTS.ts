import OpenAI from 'openai';

export interface StreamingTTSOptions {
  endpoint?: string;
  voice?: string;
  model?: string;
  speed?: number;
}

// Global OpenAI client instance
let openaiClient: OpenAI | null = null;
let currentAbortController: AbortController | null = null;
let globalAudioElement: HTMLAudioElement | null = null;
let currentBlobURL: string | null = null;
let finishedCallback: (() => void) | null = null;

/**
 * Clean up blob URL and audio element
 */
function cleanup(): void {
  if (currentBlobURL) {
    try {
      URL.revokeObjectURL(currentBlobURL);
    } catch {}
    currentBlobURL = null;
  }
  if (globalAudioElement) {
    globalAudioElement.pause();
    globalAudioElement.src = '';
    globalAudioElement.currentTime = 0;
  }
}

/**
 * Get or create OpenAI client configured for our backend
 */
function getOpenAIClient(endpoint: string): OpenAI {
  if (!openaiClient) {
    // Determine base URL from endpoint
    // The SDK appends /audio/speech to baseURL, so we need to set baseURL to /v1
    // to get /v1/audio/speech
    const url = new URL(endpoint, window.location.origin);
    const baseURL = `${url.origin}/v1`;

    openaiClient = new OpenAI({
      baseURL: baseURL,
      // No API key needed - our backend uses cookie-based auth
      apiKey: 'not-needed',
      dangerouslyAllowBrowser: true,
    });
  }
  return openaiClient;
}

/**
 * Detect if we're on iOS Safari
 */
function isIOSSafari(): boolean {
  if (typeof window === 'undefined') return false;

  // Check for iOS
  const isIOS =
    /iPad|iPhone|iPod/.test(navigator.userAgent) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);

  // Check for Safari (not Chrome/Firefox on iOS)
  const isSafari =
    /^((?!chrome|android).)*safari/i.test(navigator.userAgent) ||
    (isIOS && !window.MSStream);

  return isIOS && isSafari;
}

/**
 * Stream TTS audio and play it using OpenAI SDK or init/stream for iOS
 */
export async function streamAndPlayTTS(
  text: string,
  options: StreamingTTSOptions = {}
): Promise<void> {
  console.log('[Streaming TTS] Starting playback request');

  // Clean up any existing playback
  if (currentAbortController) {
    console.log('[Streaming TTS] Aborting previous request');
    currentAbortController.abort();
    currentAbortController = null;
  }
  cleanup();

  const endpoint = options.endpoint || '/v1/audio/speech';
  const voice = options.voice || 'alloy';
  const model = options.model || 'tts-1';
  const speed = options.speed || 1.0;

  const abortController = new AbortController();
  currentAbortController = abortController;

  // Create or reuse audio element
  if (!globalAudioElement) {
    globalAudioElement = document.createElement('audio');
    globalAudioElement.crossOrigin = 'anonymous';
    globalAudioElement.preload = 'auto';
    if (finishedCallback) {
      globalAudioElement.onended = finishedCallback;
    }
  }

  // Check if we're on iOS Safari - use init/stream approach (no HLS)
  if (isIOSSafari()) {
    console.log(
      '[Streaming TTS] iOS Safari detected - using init/stream approach'
    );
    try {
      const initUrl = `${endpoint.replace(/\/$/, '')}/init`;
      const initResponse = await fetch(initUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          input: text,
          voice: voice,
          model: model,
          speed: speed,
          response_format: 'aac',
        }),
        signal: abortController.signal,
      });

      if (!initResponse.ok) {
        const errText = await initResponse.text();
        throw new Error(
          errText || `Failed to initialize audio stream: ${initResponse.status}`
        );
      }

      const initData = await initResponse.json();
      const streamId =
        initData.stream_id ||
        initData.streamId ||
        initData.stream ||
        initData.id;
      const token =
        initData.token || initData.auth_token || initData.token_value || null;

      if (!streamId) {
        throw new Error('Server did not return stream_id for init');
      }

      // Create a fresh audio element for iOS playback
      const audio = document.createElement('audio');
      audio.crossOrigin = 'anonymous';
      audio.preload = 'auto';

      // Build stream URL
      const streamPath = token
        ? `${endpoint.replace(/\/$/, '')}/stream/${streamId}?token=${encodeURIComponent(token)}`
        : `${endpoint.replace(/\/$/, '')}/stream/${streamId}`;
      const absoluteStreamURL = new URL(streamPath, window.location.origin)
        .href;

      globalAudioElement = audio;
      currentBlobURL = null;
      globalAudioElement.src = absoluteStreamURL;

      // Set up event listeners before playing
      return new Promise((resolve, reject) => {
        if (abortController.signal.aborted) {
          reject(new Error('TTS playback aborted'));
          return;
        }

        const handleEnded = () => {
          globalAudioElement?.removeEventListener('ended', handleEnded);
          globalAudioElement?.removeEventListener('error', handleError);
          if (finishedCallback) finishedCallback();
          resolve();
        };

        const handleError = (event: Event) => {
          globalAudioElement?.removeEventListener('ended', handleEnded);
          globalAudioElement?.removeEventListener('error', handleError);
          const audioError = globalAudioElement?.error;
          const msg =
            audioError?.message ||
            (event instanceof ErrorEvent
              ? event.message
              : 'Audio playback error');
          reject(new Error(`Audio playback failed: ${msg}`));
        };

        globalAudioElement?.addEventListener('ended', handleEnded, {
          once: true,
        });
        globalAudioElement?.addEventListener('error', handleError, {
          once: true,
        });

        // Start playback
        globalAudioElement?.play().catch(err => {
          reject(new Error(`Failed to play audio: ${err}`));
        });
      });
    } catch (error) {
      // Cleanup if nothing is playing
      const isPlaying =
        globalAudioElement &&
        !globalAudioElement.paused &&
        globalAudioElement.currentTime > 0;
      if (!isPlaying) cleanup();

      if (error instanceof Error && error.name === 'AbortError') {
        if (isPlaying) return;
        throw new Error('TTS playback was cancelled');
      }
      throw error;
    } finally {
      if (currentAbortController === abortController)
        currentAbortController = null;
    }
  }

  // Desktop browsers: Use MediaSource API for streaming
  try {
    const client = getOpenAIClient(endpoint);

    // Create speech using OpenAI SDK
    // Server defaults to stream_format=audio_stream, so we don't need to specify it
    const response = await client.audio.speech.create(
      {
        model: model,
        voice: voice as
          | 'alloy'
          | 'echo'
          | 'fable'
          | 'onyx'
          | 'nova'
          | 'shimmer',
        input: text,
        response_format: 'mp3',
        speed: speed,
      },
      {
        signal: abortController.signal,
      }
    );

    // Use the response directly for progressive playback and pause/resume control
    // OpenAI SDK returns APIPromise<Response>, which resolves to a standard Response object
    // According to the TypeScript definitions: create() returns APIPromise<Response>
    if (!response.body) {
      throw new Error('Response has no body stream');
    }

    const responseStream = response.body;

    // Desktop browsers: Use MediaSource API for progressive streaming
    const reader = responseStream.getReader();

    if (
      typeof MediaSource !== 'undefined' &&
      MediaSource.isTypeSupported('audio/mpeg')
    ) {
      // Desktop browsers: Use MediaSource API for true streaming
      const mediaSource = new MediaSource();
      const mediaSourceURL = URL.createObjectURL(mediaSource);
      currentBlobURL = mediaSourceURL;
      globalAudioElement.src = mediaSourceURL;

      mediaSource.addEventListener('sourceopen', async () => {
        try {
          const sourceBuffer = mediaSource.addSourceBuffer('audio/mpeg');
          const queuedChunks: Uint8Array[] = [];

          // Function to append chunk when buffer is ready
          const appendChunk = async (chunk: Uint8Array) => {
            if (sourceBuffer.updating) {
              queuedChunks.push(chunk);
              return;
            }

            if (mediaSource.readyState === 'open') {
              try {
                sourceBuffer.appendBuffer(chunk);

                // Process queued chunks after this one
                if (queuedChunks.length > 0) {
                  sourceBuffer.addEventListener(
                    'updateend',
                    function processNext() {
                      sourceBuffer.removeEventListener(
                        'updateend',
                        processNext
                      );
                      const nextChunk = queuedChunks.shift();
                      if (nextChunk) {
                        appendChunk(nextChunk);
                      }
                    },
                    { once: true }
                  );
                }
              } catch (err) {
                console.error('[Streaming TTS] Error appending buffer:', err);
              }
            }
          };

          // Read and append chunks as they arrive
          while (true) {
            if (abortController.signal.aborted) {
              reader.cancel();
              break;
            }

            const { done, value } = await reader.read();
            if (done) {
              // Append any queued chunks before ending
              while (
                queuedChunks.length > 0 &&
                !sourceBuffer.updating &&
                mediaSource.readyState === 'open'
              ) {
                const chunk = queuedChunks.shift();
                if (chunk) {
                  try {
                    sourceBuffer.appendBuffer(chunk);
                    await new Promise(resolve => {
                      sourceBuffer.addEventListener('updateend', resolve, {
                        once: true,
                      });
                    });
                  } catch (err) {
                    console.error(
                      '[Streaming TTS] Error appending queued chunk:',
                      err
                    );
                  }
                }
              }
              if (mediaSource.readyState === 'open') {
                mediaSource.endOfStream();
              }
              break;
            }

            if (value) {
              await appendChunk(new Uint8Array(value));
            }
          }
        } catch (streamError) {
          console.error(
            '[Streaming TTS] Error in MediaSource stream:',
            streamError
          );
          if (mediaSource.readyState === 'open') {
            try {
              mediaSource.endOfStream();
            } catch {}
          }
        }
      });

      globalAudioElement.addEventListener('loadedmetadata', () => {
        globalAudioElement?.play().catch(err => {
          console.debug('[Streaming TTS] Auto-play prevented:', err);
        });
      });

      globalAudioElement.load();
    } else {
      // Desktop browser without MediaSource support - fallback to blob
      console.warn(
        '[Streaming TTS] MediaSource not supported, falling back to blob'
      );
      const chunks: BlobPart[] = [];

      while (true) {
        if (abortController.signal.aborted) {
          throw new DOMException('Request aborted', 'AbortError');
        }

        const { done, value } = await reader.read();
        if (done) break;

        if (value) {
          chunks.push(new Uint8Array(value));
        }
      }

      if (chunks.length === 0) {
        throw new Error('No audio data received');
      }

      const blob = new Blob(chunks as BlobPart[], { type: 'audio/mpeg' });
      const blobURL = URL.createObjectURL(blob);
      currentBlobURL = blobURL;
      globalAudioElement.src = blobURL;
      globalAudioElement.load();
    }

    // Wait for playback to complete
    return new Promise((resolve, reject) => {
      if (abortController.signal.aborted) {
        reject(new Error('TTS playback aborted'));
        return;
      }

      const handleEnded = () => {
        globalAudioElement?.removeEventListener('ended', handleEnded);
        globalAudioElement?.removeEventListener('error', handleError);
        if (finishedCallback) {
          finishedCallback();
        }
        resolve();
      };

      const handleError = () => {
        globalAudioElement?.removeEventListener('ended', handleEnded);
        globalAudioElement?.removeEventListener('error', handleError);
        reject(new Error('Audio playback error'));
      };

      globalAudioElement?.addEventListener('ended', handleEnded, {
        once: true,
      });
      globalAudioElement?.addEventListener('error', handleError, {
        once: true,
      });

      // Start playback
      globalAudioElement?.play().catch(err => {
        reject(new Error(`Failed to play audio: ${err}`));
      });
    });
  } catch (error) {
    // Don't cleanup if audio is playing - let it finish
    const isPlaying =
      globalAudioElement &&
      !globalAudioElement.paused &&
      globalAudioElement.currentTime > 0;
    const hasAudioElement = !!globalAudioElement;
    const audioPaused = globalAudioElement?.paused ?? true;
    const audioCurrentTime = globalAudioElement?.currentTime ?? 0;

    console.log('[Streaming TTS] Error caught:', {
      errorName: error instanceof Error ? error.name : 'unknown',
      errorMessage: error instanceof Error ? error.message : String(error),
      isPlaying,
      hasAudioElement,
      audioPaused,
      audioCurrentTime,
      abortControllerAborted: abortController.signal.aborted,
    });

    if (!isPlaying) {
      cleanup();
    }

    if (error instanceof Error && error.name === 'AbortError') {
      // If aborted but audio is playing, don't show error - just return silently
      if (isPlaying) {
        console.log(
          '[Streaming TTS] Request aborted, but audio continues playing'
        );
        return; // Return silently if audio is playing
      }
      // Check if abort happened before we could start playback
      console.warn(
        '[Streaming TTS] Request was cancelled before playback could start'
      );
      throw new Error('TTS playback was cancelled');
    }
    throw error;
  } finally {
    if (currentAbortController === abortController) {
      currentAbortController = null;
    }
  }
}

/**
 * Stop current TTS playback
 */
export function stopStreamingTTS(): void {
  if (currentAbortController) {
    currentAbortController.abort();
    currentAbortController = null;
  }
  if (globalAudioElement) {
    globalAudioElement.pause();
    globalAudioElement.currentTime = 0;
  }
  cleanup();
}

/**
 * Pause current TTS playback
 */
export function pauseStreamingTTS(): void {
  if (globalAudioElement && !globalAudioElement.paused) {
    globalAudioElement.pause();
  }
}

/**
 * Resume paused TTS playback
 */
export function resumeStreamingTTS(): void {
  if (globalAudioElement && globalAudioElement.paused) {
    globalAudioElement.play().catch(err => {
      console.warn('Failed to resume playback:', err);
    });
  }
}

/**
 * Get the current audio element (for event listeners, etc.)
 */
export function getAudioElement(): HTMLAudioElement | null {
  return globalAudioElement;
}

/**
 * Set callback for when audio finishes
 */
export function setFinishedCallback(callback: () => void): void {
  finishedCallback = callback;
  if (globalAudioElement) {
    globalAudioElement.onended = callback;
  }
}
