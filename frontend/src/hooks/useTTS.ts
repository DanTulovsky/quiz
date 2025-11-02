import { useState, useRef, useCallback, useEffect } from 'react';
import { showNotificationWithClean } from '../notifications';

// Shared state so all hook instances can track the same playback
let sharedIsPlaying = false;
let sharedIsPaused = false;
let sharedIsLoading = false;
let sharedCurrentText: string | null = null;
let sharedMetadata: TTSMetadata | null = null;

// Notify all hook instances of state changes
const stateListeners = new Set<() => void>();
function notifyStateListeners() {
  stateListeners.forEach(listener => {
    try {
      listener();
    } catch {}
  });
}

// Export function to clear shared state - can be called from finishedAudio callback
export function clearSharedTTSState() {
  sharedIsPlaying = false;
  sharedIsPaused = false;
  sharedIsLoading = false;
  sharedCurrentText = null;
  sharedMetadata = null;
  notifyStateListeners();
}

// Register the clear function with ttsRocksInit so finishedAudio can call it
if (typeof window !== 'undefined') {
  import('../utils/ttsRocksInit')
    .then(({ registerClearTTSState }) => {
      registerClearTTSState(clearSharedTTSState);
    })
    .catch(() => {
      // Silent fail - event-based clearing will still work
    });
}

// Helper function to set up Media Session API metadata
function setupMediaSessionMetadata(metadata?: TTSMetadata | null) {
  if ('mediaSession' in navigator && navigator.mediaSession) {
    const title = metadata?.title || 'Text-to-Speech';
    const artist =
      [metadata?.language, metadata?.level].filter(Boolean).join(' • ') ||
      'Language Learning Quiz';
    const album = 'TTS Audio';

    navigator.mediaSession.metadata = new MediaMetadata({
      title,
      artist,
      album,
    });

    // Set up action handlers
    navigator.mediaSession.setActionHandler('play', () => {
      if (window.TTS?.audio) {
        void window.TTS.audio.play();
      }
    });

    navigator.mediaSession.setActionHandler('pause', () => {
      if (window.TTS?.audio) {
        window.TTS.audio.pause();
      }
    });

    navigator.mediaSession.setActionHandler('stop', () => {
      if (window.TTS?.audio) {
        window.TTS.audio.pause();
        window.TTS.audio.currentTime = 0;
      }
    });
  }
}

// Global playback handle for non-hook callers
let currentPlayback: { text: string } | null = null;

export function stopTTSOnce(): void {
  // TTS.Rocks doesn't have a stop() method, so we control the audio element directly
  if (window.TTS && window.TTS.audio) {
    window.TTS.audio.pause();
    window.TTS.audio.currentTime = 0;
    if (window.TTS.audio.src && window.TTS.audio.src.startsWith('blob:')) {
      try {
        URL.revokeObjectURL(window.TTS.audio.src);
      } catch {}
    }
    window.TTS.audio.src = '';
  }
  if (window.TTS && window.TTS.finishedAudio) {
    window.TTS.finishedAudio();
  }
  currentPlayback = null;
}

interface TTSState {
  isLoading: boolean;
  isPlaying: boolean;
  isPaused: boolean;
}

export interface TTSMetadata {
  title?: string;
  language?: string;
  level?: string;
}

interface TTSHookReturn extends TTSState {
  playTTS: (
    text: string,
    voice?: string,
    metadata?: TTSMetadata
  ) => Promise<void>;
  stopTTS: () => void;
  pauseTTS: () => void;
  resumeTTS: () => void;
  restartTTS: () => boolean;
  currentText: string | null;
}

export const useTTS = (): TTSHookReturn => {
  const [isLoading, setIsLoading] = useState(sharedIsLoading);
  const [isPlayingLocal, setIsPlayingLocal] = useState(sharedIsPlaying);
  const [isPausedLocal, setIsPausedLocal] = useState(sharedIsPaused);
  const [currentTextLocal, setCurrentTextLocal] = useState(sharedCurrentText);

  // Sync local state with shared state
  useEffect(() => {
    const updateState = () => {
      setIsPlayingLocal(sharedIsPlaying);
      setIsPausedLocal(sharedIsPaused);
      setIsLoading(sharedIsLoading);
      setCurrentTextLocal(sharedCurrentText);
    };
    stateListeners.add(updateState);
    // Also update immediately to ensure we have latest state
    updateState();
    return () => {
      stateListeners.delete(updateState);
    };
  }, []);

  const setIsPlaying = useCallback((value: boolean) => {
    sharedIsPlaying = value;
    setIsPlayingLocal(value);
    notifyStateListeners();
  }, []);

  const setIsPaused = useCallback((value: boolean) => {
    sharedIsPaused = value;
    setIsPausedLocal(value);
    notifyStateListeners();
  }, []);

  const setIsLoadingState = useCallback((value: boolean) => {
    sharedIsLoading = value;
    setIsLoading(value);
    notifyStateListeners();
  }, []);

  const isPlaying = isPlayingLocal;
  const isPaused = isPausedLocal;
  const mountedRef = useRef<boolean>(true);

  const stopTTS = useCallback(() => {
    // Clear state check interval if it exists
    if (window.__ttsStateCheckInterval) {
      clearInterval(window.__ttsStateCheckInterval);
      delete window.__ttsStateCheckInterval;
    }

    // TTS.Rocks doesn't have a stop() method, so we control the audio element directly
    if (window.TTS && window.TTS.audio) {
      window.TTS.audio.pause();
      window.TTS.audio.currentTime = 0;
      // Clean up blob URL if possible
      if (window.TTS.audio.src && window.TTS.audio.src.startsWith('blob:')) {
        try {
          URL.revokeObjectURL(window.TTS.audio.src);
        } catch {}
      }
      window.TTS.audio.src = '';
    }
    // Also call finishedAudio if it exists to clean up TTS.Rocks state
    if (window.TTS && window.TTS.finishedAudio) {
      window.TTS.finishedAudio();
    }
    sharedIsPlaying = false;
    sharedIsPaused = false;
    sharedIsLoading = false;
    sharedCurrentText = null;
    sharedMetadata = null;
    currentPlayback = null;
    setIsPlaying(false);
    setIsPaused(false);
    setIsLoadingState(false);
    notifyStateListeners();
  }, [setIsPlaying, setIsPaused, setIsLoadingState]);

  const pauseTTS = useCallback(() => {
    if (window.TTS && window.TTS.audio) {
      window.TTS.audio.pause();
      setIsPaused(true);
      setIsPlaying(false);
    }
  }, [setIsPlaying, setIsPaused]);

  const resumeTTS = useCallback(() => {
    if (window.TTS && window.TTS.audio) {
      void window.TTS.audio.play().then(() => {
        setIsPaused(false);
        setIsPlaying(true);
      });
    }
  }, [setIsPlaying, setIsPaused]);

  const restartTTS = useCallback((): boolean => {
    if (!window.TTS || !window.TTS.audio || !sharedCurrentText) {
      console.warn(
        'restartTTS: Cannot restart - missing TTS, audio element, or current text'
      );
      return false;
    }
    try {
      window.TTS.audio.currentTime = 0;
      void window.TTS.audio
        .play()
        .then(() => {
          setIsPaused(false);
          setIsPlaying(true);
        })
        .catch(error => {
          console.error(
            'restartTTS: Failed to play audio after restart:',
            error
          );
          // Reset state on error
          setIsPaused(false);
          setIsPlaying(false);
        });
      return true;
    } catch (error) {
      console.error('restartTTS: Error during restart:', error);
      return false;
    }
  }, [setIsPlaying, setIsPaused]);

  const playTTS = useCallback(
    async (text: string, voice?: string, metadata?: TTSMetadata) => {
      if (!text || !text.trim()) return;

      // Ensure TTS.Rocks is loaded
      if (!window.TTS) {
        showNotificationWithClean({
          title: 'TTS Error',
          message: 'TTS library not loaded',
          color: 'red',
        });
        return;
      }

      // Stop any existing playback and clear any intervals
      stopTTS();

      // Wait a tiny bit to ensure cleanup completes
      await new Promise(resolve => setTimeout(resolve, 50));

      // Store current text and metadata
      const trimmedText = text.trim();
      sharedCurrentText = trimmedText;
      sharedMetadata = metadata || null;

      try {
        setIsLoadingState(true);
        setIsPlaying(false);
        setIsPaused(false);

        // Configure voice if provided
        if (voice && window.TTS.openAISettings) {
          window.TTS.openAISettings.voice = voice;
        }

        // Set up Media Session API
        setupMediaSessionMetadata(sharedMetadata);

        // Listen for TTS finished event (dispatched from finishedAudio callback)
        // This is more reliable than listening to audio events since TTS.Rocks
        // may create/reassign the audio element
        const handleTTSFinished = () => {
          if (mountedRef.current) {
            setIsPlaying(false);
            setIsPaused(false);
            setIsLoadingState(false);
            // Always clear sharedCurrentText when audio ends - don't require text match
            // This ensures currentPlayingText becomes null in all hook instances
            sharedCurrentText = null;
            sharedMetadata = null;
            notifyStateListeners();
          }
        };

        window.addEventListener('tts-finished', handleTTSFinished);

        // Set up audio event handlers on the audio element
        // We'll set these up after speak() is called since TTS.Rocks may create a new element
        const setupAudioListeners = () => {
          const audio = window.TTS?.audio;
          if (!audio) return;

          const handlePlay = () => {
            if (mountedRef.current) {
              setIsPlaying(true);
              setIsPaused(false);
              setIsLoadingState(false);
            }
          };

          const handlePause = () => {
            if (mountedRef.current) {
              setIsPaused(true);
              setIsPlaying(false);
            }
          };

          const handleEnded = () => {
            // Call finishedAudio to ensure cleanup
            if (window.TTS?.finishedAudio) {
              window.TTS.finishedAudio();
            }
            handleTTSFinished(); // Use the same handler
          };

          const handleError = (e: Event) => {
            const audioElement = e.target as HTMLAudioElement;
            const errorCode = audioElement?.error?.code;
            const errorMessage =
              audioElement?.error?.message || 'Unknown error';
            const src = audioElement?.src || '';

            // IGNORE: Empty src errors are expected during cleanup when we clear the src
            // The error message contains "Empty src" or "MEDIA_ELEMENT_ERROR" which indicates
            // the audio element tried to play but src was cleared/empty
            const isEmptySrcError =
              errorMessage.includes('Empty src') ||
              errorMessage.includes('empty src') ||
              errorMessage.includes('Empty src attribute') ||
              errorMessage.includes('empty src attribute') ||
              errorMessage.includes('MEDIA_ELEMENT_ERROR');

            // If error code is 4 and message indicates empty src, ignore it
            // Also check if src is empty (fallback check)
            const hasEmptySrc =
              !src ||
              src === '' ||
              src === window.location.href ||
              src === window.location.origin + '/';

            // PRIMARY CHECK: If error message says "Empty src" or "MEDIA_ELEMENT_ERROR", ignore it
            // This is the most reliable indicator that it's a cleanup-related error
            if (isEmptySrcError) {
              // Expected during cleanup - silently ignore
              return;
            }

            // SECONDARY CHECK: If error code is 4 and src is empty, also ignore
            if ((errorCode === 4 || errorCode === undefined) && hasEmptySrc) {
              // Expected during cleanup
              return;
            }

            // MEDIA_ERR_ABORTED (1) - User aborted, MEDIA_ERR_DECODE (3) - Decode error
            // MEDIA_ERR_SRC_NOT_SUPPORTED (4) - Format not supported (but not empty src)
            // These are usually real errors that should be reported
            // MEDIA_ERR_NETWORK (2) - Network error - might be transient, only log if persistent
            // IMPORTANT: Don't log error code 4 if it has empty src (checked above)
            if (
              errorCode === 1 ||
              errorCode === 3 ||
              (errorCode === 4 && !isEmptySrcError && !hasEmptySrc)
            ) {
              console.error('[TTS] Audio error:', {
                code: errorCode,
                message: errorMessage,
                src: audioElement?.src?.substring(0, 50),
                currentTime: audioElement?.currentTime,
              });

              // Only show notification if audio actually failed and hasn't played
              if (
                mountedRef.current &&
                audio &&
                (!audio.src || !audio.currentTime || audio.currentTime === 0)
              ) {
                setIsPlaying(false);
                setIsPaused(false);
                setIsLoadingState(false);
                showNotificationWithClean({
                  title: 'TTS Error',
                  message: `Failed to play audio: ${errorMessage}`,
                  color: 'red',
                });
              }
            } else if (errorCode === 2) {
              // Network error - might be transient, only log at debug level
              console.warn(
                '[TTS] Network error (might be transient):',
                errorMessage
              );
            } else {
              // Unknown error - log for debugging but don't spam
              console.warn(
                '[TTS] Audio error (non-fatal):',
                errorCode,
                errorMessage
              );
            }
          };

          const handleLoadedData = () => {
            // Update Media Session position state when metadata is available
            if (
              'mediaSession' in navigator &&
              navigator.mediaSession &&
              audio.duration &&
              isFinite(audio.duration)
            ) {
              try {
                navigator.mediaSession.setPositionState({
                  duration: audio.duration || 0,
                  playbackRate: audio.playbackRate || 1,
                  position: audio.currentTime || 0,
                });
              } catch {}
            }
          };

          const handleTimeUpdate = () => {
            // Update Media Session position state periodically
            if (
              'mediaSession' in navigator &&
              navigator.mediaSession &&
              audio.duration &&
              isFinite(audio.duration)
            ) {
              try {
                navigator.mediaSession.setPositionState({
                  duration: audio.duration || 0,
                  playbackRate: audio.playbackRate || 1,
                  position: audio.currentTime || 0,
                });
              } catch {}
            }
          };

          const handleCanPlay = () => {
            // Clear loading state when audio is ready to play
            if (mountedRef.current && sharedIsLoading) {
              setIsLoadingState(false);
            }
            // If audio is already playing or can play, set playing state
            if (mountedRef.current && !audio.paused && audio.currentTime > 0) {
              setIsPlaying(true);
              setIsPaused(false);
              setIsLoadingState(false);
            }
          };

          const handleCanPlayThrough = () => {
            // Also clear loading on canplaythrough (audio is fully buffered)
            if (mountedRef.current && sharedIsLoading) {
              setIsLoadingState(false);
            }
          };

          // Remove old listeners to prevent duplicates
          // Note: removeEventListener only works if function references match, but we try anyway
          const listenersToRemove = [
            ['play', handlePlay],
            ['pause', handlePause],
            ['ended', handleEnded],
            ['error', handleError],
            ['loadeddata', handleLoadedData],
            ['timeupdate', handleTimeUpdate],
            ['canplay', handleCanPlay],
            ['canplaythrough', handleCanPlayThrough],
          ] as const;

          // Remove any existing listeners (may fail silently if not attached)
          listenersToRemove.forEach(([event, handler]) => {
            audio.removeEventListener(event, handler);
          });

          // Add listeners
          audio.addEventListener('play', handlePlay, { once: false });
          audio.addEventListener('pause', handlePause, { once: false });
          audio.addEventListener('ended', handleEnded, { once: false });
          audio.addEventListener('error', handleError, { once: false });
          audio.addEventListener('loadeddata', handleLoadedData, {
            once: false,
          });
          audio.addEventListener('timeupdate', handleTimeUpdate, {
            once: false,
          });
          audio.addEventListener('canplay', handleCanPlay, { once: false });
          audio.addEventListener('canplaythrough', handleCanPlayThrough, {
            once: false,
          });

          // Immediately check if audio is already playing (in case it started before listeners were attached)
          if (!audio.paused && audio.currentTime > 0 && !audio.ended) {
            if (mountedRef.current) {
              setIsPlaying(true);
              setIsPaused(false);
              setIsLoadingState(false);
            }
          }
        };

        // Set up listeners on current audio element if it exists
        if (window.TTS?.audio) {
          setupAudioListeners();
        }

        // Poll for audio element after speak() is called (TTS.Rocks may create new element)
        // We'll set this up after calling speak() below

        // Verify TTS.Rocks is configured correctly before calling
        // Re-initialize if needed (in case TTS.Rocks script loaded after our init or reset the provider)
        if (window.TTS.TTSProvider !== 'openai' || !window.TTS.OpenAIAPIKey) {
          console.warn('TTS.Rocks not configured, re-initializing...', {
            provider: window.TTS.TTSProvider,
            hasApiKey: !!window.TTS.OpenAIAPIKey,
          });
          // Re-initialize TTS.Rocks
          const { initializeTTSRocks } = await import('../utils/ttsRocksInit');
          initializeTTSRocks();

          // Check again after re-initialization
          if (window.TTS.TTSProvider !== 'openai') {
            throw new Error(
              `TTS.Rocks not configured for OpenAI mode. Provider: ${window.TTS.TTSProvider}`
            );
          }

          if (!window.TTS.OpenAIAPIKey) {
            throw new Error(
              'TTS.Rocks OpenAI API key not set after initialization'
            );
          }
        }

        // Use TTS.Rocks to speak the text
        // TTS.Rocks will handle the API call to /v1/audio/speech
        // Note: speak() doesn't return a Promise, but sets up async fetch/playback
        window.TTS.speak(trimmedText, true);

        // Store playback handle
        currentPlayback = { text: trimmedText };

        // Clear any existing state check interval before starting new playback
        if (window.__ttsStateCheckInterval) {
          clearInterval(window.__ttsStateCheckInterval);
          delete window.__ttsStateCheckInterval;
        }

        // Set up listeners on the audio element after speak() is called
        // TTS.Rocks creates/reassigns the audio element in openAITTS after fetch completes
        // We need to poll for it and ensure onended is set
        let setupAttempts = 0;
        const maxSetupAttempts = 100; // 5 seconds max (50ms * 100)
        let lastAudioElement: HTMLAudioElement | null = null;
        const setupListenersAfterSpeak = setInterval(() => {
          if (window.TTS?.audio) {
            const audio = window.TTS.audio;

            // If this is a new audio element (different from last time), set up listeners immediately
            const isNewAudioElement = audio !== lastAudioElement;
            if (isNewAudioElement) {
              setupAudioListeners();
              lastAudioElement = audio;
            }

            // Check if audio has started playing (it might have already started)
            // Also check if audio has a src (meaning it's loaded)
            const hasSrc = audio.src && audio.src.length > 0;
            const isActuallyPlayingNow =
              !audio.paused && audio.currentTime > 0 && !audio.ended && hasSrc;

            // CRITICAL: Always update state if audio is playing, regardless of current state
            // This handles the case where play event fired before listeners were attached
            let shouldClearInterval = false;

            if (isActuallyPlayingNow) {
              // Always update state when audio is actually playing - don't check current state
              setIsPlaying(true);
              setIsPaused(false);
              setIsLoadingState(false);
              shouldClearInterval = true; // Clear interval once we've detected playing state
            } else if (hasSrc && sharedIsLoading && audio.readyState >= 2) {
              // Audio is loaded and ready, clear loading even if not playing yet
              setIsLoadingState(false);
            }

            // Also check if audio might have just started playing (very recent currentTime)
            if (
              hasSrc &&
              audio.currentTime > 0 &&
              audio.currentTime < 0.5 &&
              !audio.paused &&
              !audio.ended
            ) {
              // Audio just started - ensure state is correct
              setIsPlaying(true);
              setIsPaused(false);
              setIsLoadingState(false);
              shouldClearInterval = true;
            }

            // Ensure onended handler is set (TTS.Rocks sets this, but ensure our version is active)
            if (window.TTS.finishedAudio) {
              // Always set it to ensure our callback is active
              if (audio.onended !== window.TTS.finishedAudio) {
                audio.onended = window.TTS.finishedAudio;
              }
            }

            // Set up event listeners on the current audio element (if not already set up above)
            if (!isNewAudioElement) {
              setupAudioListeners();
            }

            // Clear interval if audio is playing OR if audio is ready with src
            if (shouldClearInterval || (hasSrc && audio.readyState >= 2)) {
              clearInterval(setupListenersAfterSpeak);
            }
          } else {
            setupAttempts++;
            if (setupAttempts >= maxSetupAttempts) {
              console.warn(
                'Audio element not created after speak(), attempts:',
                setupAttempts
              );
              clearInterval(setupListenersAfterSpeak);
              // Clear loading state if audio never appeared
              if (sharedIsLoading) {
                console.warn(
                  'Audio element never appeared, clearing loading state'
                );
                setIsLoadingState(false);
              }
            }
          }
        }, 50); // Check every 50ms

        // Also check periodically if audio has started (fallback for missed events)
        // Make this more aggressive to catch state issues immediately
        const checkAudioStateInterval = setInterval(() => {
          if (window.TTS?.audio) {
            const audio = window.TTS.audio;
            const isActuallyPlaying =
              !audio.paused && audio.currentTime > 0 && !audio.ended;
            const isActuallyEnded = audio.ended;
            const isActuallyPaused =
              audio.paused && audio.currentTime > 0 && !audio.ended;

            // Force state sync based on actual audio element state
            if (isActuallyPlaying && (!sharedIsPlaying || sharedIsLoading)) {
              setIsPlaying(true);
              setIsPaused(false);
              setIsLoadingState(false);
            } else if (
              isActuallyPaused &&
              (!sharedIsPaused || sharedIsPlaying)
            ) {
              setIsPaused(true);
              setIsPlaying(false);
              setIsLoadingState(false);
            } else if (
              isActuallyEnded &&
              (sharedIsPlaying || sharedIsPaused || sharedIsLoading)
            ) {
              if (window.TTS?.finishedAudio) {
                window.TTS.finishedAudio();
              }
            }
          }
        }, 100); // Check every 100ms - catch quick starts

        // Store interval to clear later
        window.__ttsStateCheckInterval = checkAudioStateInterval;

        // Loading will be set to false when audio starts playing (via event handler)
        // Also set a timeout fallback in case events don't fire
        const loadingTimeout = setTimeout(() => {
          if (mountedRef.current) {
            console.warn('TTS loading timeout - clearing loading state');
            setIsLoadingState(false);
          }
        }, 30000); // 30 second timeout

        // Cleanup function
        const cleanup = () => {
          clearTimeout(loadingTimeout);
          clearInterval(setupListenersAfterSpeak);
          if (window.__ttsStateCheckInterval) {
            clearInterval(window.__ttsStateCheckInterval);
            delete window.__ttsStateCheckInterval;
          }
          window.removeEventListener('tts-finished', handleTTSFinished);
        };

        // Store cleanup to call on next playTTS or unmount
        if (window.__ttsCleanup) {
          try {
            window.__ttsCleanup();
          } catch (e) {
            console.error('Error calling previous cleanup:', e);
          }
        }
        window.__ttsCleanup = cleanup;
      } catch (error) {
        const message =
          error instanceof Error ? error.message : 'Failed to generate speech';
        showNotificationWithClean({
          title: 'TTS Error',
          message,
          color: 'red',
        });
        setIsLoadingState(false);
        setIsPlaying(false);
        setIsPaused(false);
        if (sharedCurrentText === trimmedText) {
          sharedCurrentText = null;
        }
      }
    },
    [stopTTS, setIsPlaying, setIsPaused, setIsLoadingState]
  );

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  return {
    isLoading,
    isPlaying,
    isPaused,
    playTTS,
    stopTTS,
    pauseTTS,
    resumeTTS,
    restartTTS,
    currentText: currentTextLocal,
  };
};

// Convenience helper to play a single TTS sample without using the React hook.
export async function playTTSOnce(
  text: string,
  voice?: string,
  callbacks?: {
    onBuffering?: (p: number) => void;
    onPlayStart?: () => void;
    onPlayEnd?: () => void;
  }
): Promise<void> {
  if (!text || !text.trim()) return;

  if (!window.TTS) {
    const message = 'TTS library not loaded';
    showNotificationWithClean({ title: 'TTS Error', message, color: 'red' });
    throw new Error(message);
  }

  try {
    // Stop any existing playback
    if (currentPlayback) {
      stopTTSOnce();
    }

    // Configure voice if provided
    if (voice && window.TTS.openAISettings) {
      window.TTS.openAISettings.voice = voice;
    }

    // Set up callbacks
    const audio = window.TTS.audio;
    if (audio && callbacks) {
      const handleCanPlay = () => {
        if (callbacks.onBuffering) {
          callbacks.onBuffering(1);
        }
      };

      const handlePlay = () => {
        if (callbacks.onPlayStart) {
          callbacks.onPlayStart();
        }
      };

      const handleEnded = () => {
        if (callbacks.onPlayEnd) {
          callbacks.onPlayEnd();
        }
        audio.removeEventListener('canplay', handleCanPlay);
        audio.removeEventListener('play', handlePlay);
        audio.removeEventListener('ended', handleEnded);
      };

      audio.addEventListener('canplay', handleCanPlay, { once: true });
      audio.addEventListener('play', handlePlay, { once: true });
      audio.addEventListener('ended', handleEnded, { once: true });
    }

    // Call onBuffering start
    if (callbacks?.onBuffering) {
      callbacks.onBuffering(0);
    }

    // Use TTS.Rocks to speak
    await window.TTS.speak(text.trim(), true);

    // Wait for playback to complete
    if (audio) {
      await new Promise<void>(resolve => {
        if (audio.ended) {
          resolve();
        } else {
          const handleEnded = () => {
            audio.removeEventListener('ended', handleEnded);
            resolve();
          };
          audio.addEventListener('ended', handleEnded, { once: true });
        }
      });
    }
  } catch (error) {
    const message =
      error instanceof Error ? error.message : 'Failed to play audio';
    showNotificationWithClean({ title: 'TTS Error', message, color: 'red' });
    throw error;
  }
}
