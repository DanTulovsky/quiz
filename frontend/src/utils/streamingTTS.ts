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
function ensureAudioElementAttached(audio: HTMLAudioElement): void {
  if (!audio.hasAttribute('data-quiz-tts')) {
    audio.setAttribute('data-quiz-tts', '');
  }

  // Keep the element in the DOM (and not display:none) so iOS can continue playback in background.
  const style = audio.style;
  style.position = 'fixed';
  style.bottom = '0';
  style.left = '0';
  style.width = '1px';
  style.height = '1px';
  style.opacity = '0';
  style.pointerEvents = 'none';
  style.zIndex = '-1';
  audio.muted = false;
  audio.defaultMuted = false;
  audio.volume = 1;
  audio.setAttribute('playsinline', 'true');
  audio.setAttribute('webkit-playsinline', 'true');
  audio.setAttribute('aria-hidden', 'true');
  audio.tabIndex = -1;

  if (typeof document !== 'undefined' && audio.parentNode !== document.body) {
    // Remove from previous parent to avoid duplicates
    if (audio.parentNode) {
      audio.parentNode.removeChild(audio);
    }
    document.body.appendChild(audio);
  }
}

/**
 * Clean up blob URL and audio element
 */
const INTENTIONAL_SHUTDOWN_ATTR = 'data-quiz-tts-intentional-shutdown';

function markIntentionalShutdown(
  audio: HTMLAudioElement | null,
  reason: string
): void {
  if (!audio) return;
  audio.setAttribute(INTENTIONAL_SHUTDOWN_ATTR, reason || 'true');
}

function clearIntentionalShutdown(audio: HTMLAudioElement | null): void {
  audio?.removeAttribute(INTENTIONAL_SHUTDOWN_ATTR);
}

function getIntentionalShutdownReason(
  audio: HTMLAudioElement | null
): string | null {
  if (!audio) return null;
  return audio.getAttribute(INTENTIONAL_SHUTDOWN_ATTR);
}

function cleanup(): void {
  if (currentBlobURL) {
    try {
      URL.revokeObjectURL(currentBlobURL);
    } catch {}
    currentBlobURL = null;
  }
  if (globalAudioElement) {
    // Pause first, then wait a bit before clearing src to avoid triggering errors
    // that might be caught by active error handlers
    globalAudioElement.pause();
    globalAudioElement.currentTime = 0;
    // Use a small timeout to allow any pending error handlers to complete
    // before clearing src, which can trigger new error events
    setTimeout(() => {
      if (globalAudioElement) {
        globalAudioElement.src = '';
      }
    }, 0);
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
 * Detect if we're on Safari (iOS or desktop) - use init/stream approach
 */
function useSafariTTS(): boolean {
  if (typeof window === 'undefined') return false;

  // Check for iOS
  const isIOS =
    /iPad|iPhone|iPod/.test(navigator.userAgent) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);

  // Check for Safari (not Chrome/Firefox on iOS, and not Chrome/Edge on desktop)
  // MSStream is an old IE property that doesn't exist in TypeScript types
  const hasMSStream = 'MSStream' in window;
  const isSafari =
    /^((?!chrome|android).)*safari/i.test(navigator.userAgent) ||
    (isIOS && !hasMSStream);

  return isSafari;
}

/**
 * Check if we're on desktop Safari (not iOS)
 */
function isDesktopSafari(): boolean {
  if (typeof window === 'undefined') return false;

  const isIOS =
    /iPad|iPhone|iPod/.test(navigator.userAgent) ||
    (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);

  const hasMSStream = 'MSStream' in window;
  const isSafari =
    /^((?!chrome|android).)*safari/i.test(navigator.userAgent) ||
    (isIOS && !hasMSStream);

  return isSafari && !isIOS;
}

const MEDIA_ERR_SRC_NOT_SUPPORTED_CODE =
  typeof MediaError !== 'undefined'
    ? MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED
    : 4;

const HAVE_CURRENT_DATA =
  typeof HTMLMediaElement !== 'undefined'
    ? HTMLMediaElement.HAVE_CURRENT_DATA
    : 2;

const HAVE_FUTURE_DATA =
  typeof HTMLMediaElement !== 'undefined'
    ? HTMLMediaElement.HAVE_FUTURE_DATA
    : 3;

export interface SafariPlaybackErrorContext {
  audio: HTMLAudioElement | null;
  audioError: MediaError | null;
  playbackStarted: boolean;
  event: Event;
  intentionalShutdown?: boolean;
  intentionalShutdownReason?: string | null;
}

export interface SafariPlaybackErrorClassification {
  recoverable: boolean;
  reason: string;
}

export function classifySafariPlaybackError({
  audio,
  audioError,
  playbackStarted,
  event,
  intentionalShutdown = false,
  intentionalShutdownReason = null,
}: SafariPlaybackErrorContext): SafariPlaybackErrorClassification {
  if (intentionalShutdown) {
    return {
      recoverable: true,
      reason: intentionalShutdownReason
        ? `intentional-shutdown:${intentionalShutdownReason}`
        : 'intentional-shutdown',
    };
  }

  if (!playbackStarted) {
    return { recoverable: false, reason: 'playback-not-started' };
  }

  if (!audio) {
    return { recoverable: false, reason: 'no-audio-element' };
  }

  const currentTime = audio.currentTime ?? 0;
  const hasProgress = currentTime > 0;
  const ended = !!audio.ended;
  const paused = !!audio.paused;
  const readyState = audio.readyState ?? 0;
  const hasBufferedData = readyState >= HAVE_CURRENT_DATA;
  const hasFutureData = readyState >= HAVE_FUTURE_DATA;
  const isActuallyPlaying = !paused && !ended && hasProgress && hasBufferedData;
  const isBufferedWhilePaused =
    paused && !ended && hasProgress && hasFutureData;

  const errorMessage = event instanceof ErrorEvent ? event.message || '' : '';
  const normalizedMessage = errorMessage.toLowerCase();

  const indicatesUnsupportedFormat =
    (audioError && audioError.code === MEDIA_ERR_SRC_NOT_SUPPORTED_CODE) ||
    normalizedMessage.includes('not supported by safari') ||
    normalizedMessage.includes('format not supported') ||
    normalizedMessage.includes('audio format not supported');

  if (!hasProgress) {
    return { recoverable: false, reason: 'no-playback-progress' };
  }

  if (
    indicatesUnsupportedFormat &&
    (isActuallyPlaying || isBufferedWhilePaused)
  ) {
    return {
      recoverable: true,
      reason: 'safari-format-false-positive',
    };
  }

  if (!audioError && (isActuallyPlaying || isBufferedWhilePaused)) {
    return {
      recoverable: true,
      reason: 'playback-continues-without-error',
    };
  }

  return { recoverable: false, reason: 'fatal-error' };
}

/**
 * Stream TTS audio and play it using OpenAI SDK or init/stream for iOS
 */
export async function streamAndPlayTTS(
  text: string,
  options: StreamingTTSOptions = {}
): Promise<void> {
  // Clean up any existing playback
  if (currentAbortController) {
    currentAbortController.abort();
    currentAbortController = null;
  }
  markIntentionalShutdown(globalAudioElement, 'restart');
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
    ensureAudioElementAttached(globalAudioElement);
    if (finishedCallback) {
      globalAudioElement.onended = finishedCallback;
    }
  } else {
    ensureAudioElementAttached(globalAudioElement);
  }
  clearIntentionalShutdown(globalAudioElement);

  // Check if we're on Safari (iOS or desktop) - use init/stream approach (no HLS)
  if (useSafariTTS()) {
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
          // Use mp3 for all Safari targets; modern iOS Safari can decode mp3 reliably
          response_format: 'mp3',
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

      // Create a fresh audio element for Safari playback
      const audio = document.createElement('audio');
      audio.crossOrigin = 'anonymous';
      audio.preload = 'auto';
      ensureAudioElementAttached(audio);

      // Build stream URL
      const streamPath = token
        ? `${endpoint.replace(/\/$/, '')}/stream/${streamId}?token=${encodeURIComponent(token)}`
        : `${endpoint.replace(/\/$/, '')}/stream/${streamId}`;
      const absoluteStreamURL = new URL(streamPath, window.location.origin)
        .href;

      globalAudioElement = audio;
      clearIntentionalShutdown(globalAudioElement);
      currentBlobURL = null;
      globalAudioElement.src = absoluteStreamURL;

      // Explicitly call load() to trigger Safari to start loading the stream
      // This ensures Safari makes its probe requests (Range, Icy-Metadata) immediately
      globalAudioElement.load();

      // Set up event listeners before playing
      return new Promise((resolve, reject) => {
        if (abortController.signal.aborted) {
          reject(new Error('TTS playback aborted'));
          return;
        }

        let playbackStarted = false;
        let readinessTimeout: ReturnType<typeof setTimeout> | null = null;
        let stateCheckInterval: ReturnType<typeof setInterval> | null = null;
        let playTimeout: ReturnType<typeof setTimeout> | null = null;
        const READINESS_TIMEOUT_MS = 10000; // 10 seconds timeout for audio to become ready

        const logAudioState = (event: string) => {
          void event; // Parameter kept for API consistency with call sites
          const audio = globalAudioElement;
          if (!audio) return;

          const buffered = [];
          for (let i = 0; i < audio.buffered.length; i++) {
            buffered.push({
              start: audio.buffered.start(i),
              end: audio.buffered.end(i),
            });
          }
        };

        const clearReadinessMonitors = () => {
          if (readinessTimeout) {
            clearTimeout(readinessTimeout);
            readinessTimeout = null;
          }
          if (stateCheckInterval) {
            clearInterval(stateCheckInterval);
            stateCheckInterval = null;
          }
        };

        const cleanupPlayTimeout = () => {
          if (playTimeout) {
            clearTimeout(playTimeout);
            playTimeout = null;
          }
        };

        const removeReadinessListeners = () => {
          globalAudioElement?.removeEventListener('canplay', handleCanPlay);
          globalAudioElement?.removeEventListener(
            'canplaythrough',
            handleCanPlayThrough
          );
          globalAudioElement?.removeEventListener('loadstart', handleLoadStart);
          globalAudioElement?.removeEventListener('stalled', handleStalled);
          globalAudioElement?.removeEventListener('progress', handleProgress);
          globalAudioElement?.removeEventListener(
            'loadedmetadata',
            handleLoadedMetadata
          );
          globalAudioElement?.removeEventListener(
            'loadeddata',
            handleLoadedData
          );
          globalAudioElement?.removeEventListener('waiting', handleWaiting);
          clearReadinessMonitors();
        };

        const cleanupPlaybackListeners = () => {
          globalAudioElement?.removeEventListener('ended', handleEnded);
          globalAudioElement?.removeEventListener('error', handleError);
          globalAudioElement?.removeEventListener('playing', handlePlaying);
          globalAudioElement?.removeEventListener('pause', handlePause);
        };

        const cleanupAllListeners = () => {
          removeReadinessListeners();
          cleanupPlaybackListeners();
          cleanupPlayTimeout();
        };

        const handleEnded = () => {
          cleanupAllListeners();
          if (finishedCallback) finishedCallback();
          resolve();
        };

        const handleError = (event: Event) => {
          logAudioState('error');

          const audioError = globalAudioElement?.error;
          const intentionalShutdownReason = getIntentionalShutdownReason(
            globalAudioElement ?? null
          );
          let msg =
            audioError?.message ||
            (event instanceof ErrorEvent
              ? event.message
              : 'Audio playback error');

          const classification = classifySafariPlaybackError({
            audio: globalAudioElement ?? null,
            audioError: audioError ?? null,
            playbackStarted,
            event,
            intentionalShutdown: !!intentionalShutdownReason,
            intentionalShutdownReason,
          });

          if (classification.recoverable) {
            cleanupAllListeners();
            clearIntentionalShutdown(globalAudioElement ?? null);
            logAudioState('error-recoverable');
            return;
          }

          clearIntentionalShutdown(globalAudioElement ?? null);
          cleanupAllListeners();

          // Provide more helpful error messages for iOS-specific issues
          if (audioError) {
            switch (audioError.code) {
              case MediaError.MEDIA_ERR_ABORTED:
                msg = 'Audio loading was aborted';
                break;
              case MediaError.MEDIA_ERR_NETWORK:
                msg = 'Network error while loading audio stream';
                break;
              case MediaError.MEDIA_ERR_DECODE:
                msg = 'Audio decoding error - stream may be corrupted';
                break;
              case MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED:
                msg = 'Audio format not supported by Safari';
                break;
            }
          }

          reject(new Error(`Audio playback failed: ${msg}`));
        };

        const handleLoadStart = () => {
          logAudioState('loadstart');
        };

        const handleStalled = () => {
          logAudioState('stalled');
        };

        const handleProgress = () => {
          logAudioState('progress');
        };

        const handleLoadedMetadata = () => {
          logAudioState('loadedmetadata');
          // On desktop Safari, try playing immediately after metadata loads
          // Desktop Safari may not fire loadeddata or canplay events reliably
          if (
            !playbackStarted &&
            !abortController.signal.aborted &&
            isDesktopSafari()
          ) {
            // Small delay to let Safari finish probe requests
            setTimeout(() => {
              if (!playbackStarted && !abortController.signal.aborted) {
                attemptPlay();
              }
            }, 500);
          }
        };

        const handleLoadedData = () => {
          logAudioState('loadeddata');
          // On desktop Safari, loadeddata may fire before canplay - try playing early
          if (!playbackStarted && !abortController.signal.aborted) {
            attemptPlay();
          }
        };

        const handleWaiting = () => {
          logAudioState('waiting');
        };

        const handlePlaying = () => {
          logAudioState('playing');
          if (import.meta.env?.DEV) {
          }
          // Playback has started successfully - clear any pending timeouts
          if (readinessTimeout) {
            clearTimeout(readinessTimeout);
            readinessTimeout = null;
          }
        };

        const handlePause = () => {
          logAudioState('pause');
        };

        const attemptPlay = async () => {
          if (playbackStarted || abortController.signal.aborted) {
            return;
          }

          logAudioState('attemptPlay');

          try {
            playbackStarted = true;
            removeReadinessListeners();

            const playPromise = globalAudioElement?.play();
            if (!playPromise) {
              throw new Error('play() returned undefined');
            }

            // Add a timeout for the play promise to prevent hanging
            // For desktop Safari, we may need to wait longer due to probe requests
            const playTimeoutMs = isDesktopSafari() ? 15000 : 5000;
            let playPromiseResolved = false;

            playTimeout = setTimeout(async () => {
              if (playPromiseResolved) {
                return; // Promise already resolved, just cleanup
              }

              logAudioState('play-timeout-check');

              // Check if audio is actually playing despite promise not resolving
              if (globalAudioElement) {
                const isActuallyPlaying =
                  !globalAudioElement.paused &&
                  globalAudioElement.currentTime > 0 &&
                  globalAudioElement.readyState >= 2; // HAVE_CURRENT_DATA or better

                if (isActuallyPlaying) {
                  logAudioState('play-timeout-success');
                  return; // Audio is playing, don't reject
                } else {
                  // Try to manually trigger play again
                  try {
                    await globalAudioElement.play();
                    logAudioState('play-timeout-retry-success');
                  } catch (err) {
                    logAudioState('play-timeout-retry-failed');
                    cleanupAllListeners();
                    reject(
                      new Error(
                        `Play promise timed out and manual play failed: ${err instanceof Error ? err.message : String(err)}`
                      )
                    );
                  }
                }
              } else {
                cleanupAllListeners();
                reject(
                  new Error(
                    'Play promise timed out and audio element is missing'
                  )
                );
              }
            }, playTimeoutMs);

            try {
              await playPromise;
              playPromiseResolved = true;
              cleanupPlayTimeout();
            } catch (playErr) {
              playPromiseResolved = true;
              cleanupPlayTimeout();

              // For desktop Safari, check if audio actually started playing despite the error
              if (isDesktopSafari() && globalAudioElement) {
                logAudioState('play-error-recovery');

                // Wait a moment to see if playback actually started
                await new Promise(resolve => setTimeout(resolve, 500));

                if (
                  globalAudioElement &&
                  !globalAudioElement.paused &&
                  globalAudioElement.currentTime > 0
                ) {
                  logAudioState('play-recovered');
                  // Don't throw - audio is playing
                } else {
                  throw playErr;
                }
              } else {
                throw playErr;
              }
            }

            logAudioState('play-success');
            // Resolve the outer promise when playback actually starts
            // Note: We don't resolve here - we wait for the 'playing' event or 'ended' event
          } catch (err) {
            logAudioState('play-error');
            const errorMsg =
              err instanceof Error
                ? err.message
                : 'Failed to start audio playback';
            cleanupAllListeners();
            reject(
              new Error(
                `Failed to play audio: ${errorMsg}. This may occur if Safari is still processing probe requests.`
              )
            );
          }
        };

        const handleCanPlay = () => {
          logAudioState('canplay');
          if (readinessTimeout) {
            clearTimeout(readinessTimeout);
            readinessTimeout = null;
          }
          attemptPlay();
        };

        const handleCanPlayThrough = () => {
          logAudioState('canplaythrough');
          if (readinessTimeout) {
            clearTimeout(readinessTimeout);
            readinessTimeout = null;
          }
          attemptPlay();
        };

        // Set up event listeners

        globalAudioElement?.addEventListener('ended', handleEnded, {
          once: true,
        });
        globalAudioElement?.addEventListener('error', handleError, {
          once: true,
        });
        globalAudioElement?.addEventListener('loadstart', handleLoadStart, {
          once: true,
        });
        globalAudioElement?.addEventListener('stalled', handleStalled);
        globalAudioElement?.addEventListener('progress', handleProgress);
        globalAudioElement?.addEventListener(
          'loadedmetadata',
          handleLoadedMetadata
        );
        globalAudioElement?.addEventListener('loadeddata', handleLoadedData);
        globalAudioElement?.addEventListener('waiting', handleWaiting);
        globalAudioElement?.addEventListener('playing', handlePlaying);
        globalAudioElement?.addEventListener('pause', handlePause);

        // Wait for canplay or canplaythrough - Safari needs time to complete probe requests
        // canplay is fired when enough data is loaded to start playback
        // canplaythrough is fired when the entire stream is loaded (may not fire for streams)
        globalAudioElement?.addEventListener('canplay', handleCanPlay, {
          once: true,
        });
        globalAudioElement?.addEventListener(
          'canplaythrough',
          handleCanPlayThrough,
          {
            once: true,
          }
        );

        // Periodic state checking for debugging
        stateCheckInterval = setInterval(() => {
          if (!playbackStarted && !abortController.signal.aborted) {
            logAudioState('periodic-check');
          }
        }, 1000);

        // Timeout fallback: if Safari doesn't signal readiness within 10 seconds,
        // try playing anyway (some streams may work without explicit readiness events)
        readinessTimeout = setTimeout(() => {
          if (!playbackStarted && !abortController.signal.aborted) {
            logAudioState('timeout');
            attemptPlay();
          }
        }, READINESS_TIMEOUT_MS);
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
    if (!response.ok) {
      // If response is not ok, throw an error with status code
      throw new Error(`TTS request failed: ${response.status}`);
    }
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
        const audioElement = globalAudioElement;
        const MIN_BUFFERED_SECONDS = 0.35;
        const PLAYBACK_READY_TIMEOUT_MS = 2500;

        let playbackRequested = false;
        let playbackReadyTimeout: ReturnType<typeof setTimeout> | null = null;

        function clearPlaybackReadyTimeout(): void {
          if (playbackReadyTimeout) {
            clearTimeout(playbackReadyTimeout);
            playbackReadyTimeout = null;
          }
        }

        function detachPlaybackGuards(): void {
          clearPlaybackReadyTimeout();
          if (!audioElement) return;
          audioElement.removeEventListener('canplay', handleCanPlay);
          audioElement.removeEventListener('loadeddata', handleLoadedData);
          audioElement.removeEventListener(
            'loadedmetadata',
            handleLoadedMetadata
          );
        }

        function startPlayback(): boolean {
          if (
            !audioElement ||
            playbackRequested ||
            abortController.signal.aborted
          ) {
            if (abortController.signal.aborted) {
              detachPlaybackGuards();
            }
            return playbackRequested;
          }

          playbackRequested = true;
          detachPlaybackGuards();

          try {
            const buffered = audioElement.buffered;
            if (buffered.length > 0) {
              const firstRangeStart = buffered.start(0);
              if (!Number.isNaN(firstRangeStart) && firstRangeStart >= 0) {
                audioElement.currentTime = firstRangeStart;
              }
            }
          } catch {
            if (import.meta.env?.DEV) {
            }
          }

          audioElement.play().catch(() => {});
          return true;
        }

        function checkAndStartPlayback(reason: string): boolean {
          void reason; // Parameter kept for API consistency with call sites
          if (!audioElement || playbackRequested) {
            return playbackRequested;
          }
          if (abortController.signal.aborted) {
            detachPlaybackGuards();
            return playbackRequested;
          }

          if (audioElement.readyState >= 3) {
            return startPlayback();
          }

          try {
            const buffered = audioElement.buffered;
            if (buffered.length === 0) {
              return false;
            }

            const lastIndex = buffered.length - 1;
            const start = buffered.start(0);
            const end = buffered.end(lastIndex);
            const current = audioElement.currentTime;

            if (
              !Number.isNaN(start) &&
              start > current &&
              start - current < 0.5
            ) {
              audioElement.currentTime = start;
            }

            const bufferedAhead = end - audioElement.currentTime;
            if (bufferedAhead >= MIN_BUFFERED_SECONDS) {
              return startPlayback();
            }
          } catch {
            if (import.meta.env?.DEV) {
            }
          }

          return false;
        }

        function scheduleBufferedCheck(reason: string): void {
          if (
            !audioElement ||
            playbackRequested ||
            abortController.signal.aborted
          ) {
            return;
          }
          if (typeof requestAnimationFrame === 'function') {
            requestAnimationFrame(() => {
              checkAndStartPlayback(reason);
            });
          } else {
            setTimeout(() => {
              checkAndStartPlayback(reason);
            }, 0);
          }
        }

        function handleCanPlay(): void {
          checkAndStartPlayback('canplay');
        }

        function handleLoadedData(): void {
          checkAndStartPlayback('loadeddata');
        }

        function handleLoadedMetadata(): void {
          checkAndStartPlayback('loadedmetadata');
        }

        function attachPlaybackGuards(): void {
          if (!audioElement || abortController.signal.aborted) return;
          audioElement.addEventListener('canplay', handleCanPlay);
          audioElement.addEventListener('loadeddata', handleLoadedData);
          audioElement.addEventListener('loadedmetadata', handleLoadedMetadata);

          clearPlaybackReadyTimeout();
          playbackReadyTimeout = setTimeout(() => {
            if (!checkAndStartPlayback('timeout')) {
              startPlayback();
            }
          }, PLAYBACK_READY_TIMEOUT_MS);
        }

        try {
          const sourceBuffer = mediaSource.addSourceBuffer('audio/mpeg');
          const queuedChunks: Uint8Array[] = [];

          attachPlaybackGuards();

          // Function to append chunk when buffer is ready
          const appendChunk = async (chunk: Uint8Array): Promise<void> => {
            // If buffer is updating, queue the chunk
            if (sourceBuffer.updating) {
              queuedChunks.push(chunk);
              return;
            }

            // Check if we should abort
            if (
              abortController.signal.aborted ||
              mediaSource.readyState !== 'open'
            ) {
              detachPlaybackGuards();
              return;
            }

            try {
              // @ts-expect-error - MediaSource API accepts Uint8Array but TypeScript types are overly strict
              sourceBuffer.appendBuffer(chunk);

              // Wait for the append to complete before processing next chunk
              await new Promise<void>((resolve, reject) => {
                const handleUpdateEnd = () => {
                  sourceBuffer.removeEventListener(
                    'updateend',
                    handleUpdateEnd
                  );
                  sourceBuffer.removeEventListener('error', handleError);
                  resolve();
                };

                const handleError = () => {
                  sourceBuffer.removeEventListener(
                    'updateend',
                    handleUpdateEnd
                  );
                  sourceBuffer.removeEventListener('error', handleError);
                  reject(new Error('SourceBuffer append failed'));
                };

                sourceBuffer.addEventListener('updateend', handleUpdateEnd, {
                  once: true,
                });
                sourceBuffer.addEventListener('error', handleError, {
                  once: true,
                });
              });

              scheduleBufferedCheck('append');

              // Process queued chunks after this one completes
              if (queuedChunks.length > 0 && !abortController.signal.aborted) {
                const nextChunk = queuedChunks.shift();
                if (nextChunk) {
                  await appendChunk(nextChunk);
                }
              }
            } catch {
              scheduleBufferedCheck('append-error');
              // If append failed, try to continue with next chunk
              if (queuedChunks.length > 0 && !abortController.signal.aborted) {
                const nextChunk = queuedChunks.shift();
                if (nextChunk) {
                  await appendChunk(nextChunk);
                }
              }
            }
          };

          // Read and append chunks as they arrive
          while (true) {
            if (abortController.signal.aborted) {
              detachPlaybackGuards();
              reader.cancel();
              break;
            }

            const { done, value } = await reader.read();
            if (done) {
              // Wait for any in-progress append to complete
              while (
                sourceBuffer.updating &&
                mediaSource.readyState === 'open'
              ) {
                await new Promise<void>(resolve => {
                  sourceBuffer.addEventListener('updateend', () => resolve(), {
                    once: true,
                  });
                  sourceBuffer.addEventListener('error', () => resolve(), {
                    once: true,
                  });
                });
              }

              // Append any remaining queued chunks before ending
              while (
                queuedChunks.length > 0 &&
                !sourceBuffer.updating &&
                mediaSource.readyState === 'open' &&
                !abortController.signal.aborted
              ) {
                const chunk = queuedChunks.shift();
                if (chunk) {
                  await appendChunk(chunk);
                }
              }

              // Final check - wait for any final append to complete
              while (
                sourceBuffer.updating &&
                mediaSource.readyState === 'open'
              ) {
                await new Promise<void>(resolve => {
                  sourceBuffer.addEventListener('updateend', () => resolve(), {
                    once: true,
                  });
                  sourceBuffer.addEventListener('error', () => resolve(), {
                    once: true,
                  });
                });
              }

              if (mediaSource.readyState === 'open') {
                try {
                  mediaSource.endOfStream();
                } catch {}
              }

              scheduleBufferedCheck('end-of-stream');
              break;
            }

            if (value) {
              await appendChunk(new Uint8Array(value));
            }
          }
        } catch {
          detachPlaybackGuards();
          if (mediaSource.readyState === 'open') {
            try {
              mediaSource.endOfStream();
            } catch {}
          }
        }
      });

      globalAudioElement.load();
    } else {
      // Desktop browser without MediaSource support - fallback to blob
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

      const handleError = (e: Event) => {
        globalAudioElement?.removeEventListener('ended', handleEnded);
        globalAudioElement?.removeEventListener('error', handleError);

        // Check if this is a cleanup-related error (empty src)
        const audioElement = e.target as HTMLAudioElement;
        const errorCode = audioElement?.error?.code;
        const errorMessage = audioElement?.error?.message || 'Unknown error';
        const src = audioElement?.src || '';

        // IGNORE: Empty src errors are expected during cleanup when we clear the src
        const isEmptySrcError =
          errorMessage.includes('Empty src') ||
          errorMessage.includes('empty src') ||
          errorMessage.includes('Empty src attribute') ||
          errorMessage.includes('empty src attribute') ||
          errorMessage.includes('MEDIA_ELEMENT_ERROR');

        const hasEmptySrc =
          !src ||
          src === '' ||
          src === window.location.href ||
          src === window.location.origin + '/';

        // If this is a cleanup-related error, resolve silently (don't reject)
        if (
          isEmptySrcError ||
          ((errorCode === 4 || errorCode === undefined) && hasEmptySrc)
        ) {
          resolve(); // Resolve instead of reject for cleanup errors
          return;
        }

        // Also check if the request was aborted
        if (abortController.signal.aborted) {
          resolve(); // Resolve silently for aborted requests
          return;
        }

        // Only reject for real errors
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

    if (!isPlaying) {
      cleanup();
    }

    if (error instanceof Error && error.name === 'AbortError') {
      // If aborted but audio is playing, don't show error - just return silently
      if (isPlaying) {
        return; // Return silently if audio is playing
      }
      // Check if abort happened before we could start playback
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
  markIntentionalShutdown(globalAudioElement, 'stop');
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
    globalAudioElement.play().catch(() => {});
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
