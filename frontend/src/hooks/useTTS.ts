import { useState, useRef, useCallback, useEffect } from 'react';
import { showNotificationWithClean } from '../notifications';
import {
  streamAndPlayTTS,
  stopStreamingTTS,
  pauseStreamingTTS,
  resumeStreamingTTS,
  getAudioElement,
  setFinishedCallback,
} from '../utils/streamingTTS';

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

// Set up finished callback to clear shared state
if (typeof window !== 'undefined') {
  setFinishedCallback(() => {
    clearSharedTTSState();
    window.dispatchEvent(new CustomEvent('tts-finished'));
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
      resumeStreamingTTS();
    });

    navigator.mediaSession.setActionHandler('pause', () => {
      pauseStreamingTTS();
    });

    navigator.mediaSession.setActionHandler('stop', () => {
      stopStreamingTTS();
    });
  }
}

// Global playback handle for non-hook callers
let currentPlayback: { text: string } | null = null;

export function stopTTSOnce(): void {
  stopStreamingTTS();
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

    stopStreamingTTS();
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
    pauseStreamingTTS();
    setIsPaused(true);
    setIsPlaying(false);
  }, [setIsPlaying, setIsPaused]);

  const resumeTTS = useCallback(() => {
    resumeStreamingTTS();
    setIsPaused(false);
    setIsPlaying(true);
  }, [setIsPlaying, setIsPaused]);

  const restartTTS = useCallback((): boolean => {
    const audio = getAudioElement();
    if (!audio || !sharedCurrentText) {
      console.warn(
        'restartTTS: Cannot restart - missing audio element or current text'
      );
      return false;
    }
    try {
      audio.currentTime = 0;
      void audio
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

      // Stop any existing playback
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

        // Set up Media Session API
        setupMediaSessionMetadata(sharedMetadata);

        // Listen for TTS finished event
        const handleTTSFinished = () => {
          if (mountedRef.current) {
            setIsPlaying(false);
            setIsPaused(false);
            setIsLoadingState(false);
            sharedCurrentText = null;
            sharedMetadata = null;
            notifyStateListeners();
          }
        };

        window.addEventListener('tts-finished', handleTTSFinished);

        // Set up audio event handlers on the audio element
        const setupAudioListeners = () => {
          const audio = getAudioElement();
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
            handleTTSFinished();
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
        const audio = getAudioElement();
        if (audio) {
          setupAudioListeners();
        }

        // Store playback handle
        currentPlayback = { text: trimmedText };

        // Clear any existing state check interval
        if (window.__ttsStateCheckInterval) {
          clearInterval(window.__ttsStateCheckInterval);
          delete window.__ttsStateCheckInterval;
        }

        // Set up listeners after audio element is created (poll briefly)
        let setupAttempts = 0;
        const maxSetupAttempts = 20; // 1 second max (50ms * 20)
        const setupListenersAfterStart = setInterval(() => {
          const currentAudio = getAudioElement();
          if (currentAudio && currentAudio.src) {
            setupAudioListeners();
            clearInterval(setupListenersAfterStart);
          } else {
            setupAttempts++;
            if (setupAttempts >= maxSetupAttempts) {
              clearInterval(setupListenersAfterStart);
            }
          }
        }, 50);

        // Also check periodically for audio state changes (simpler than before)
        const checkAudioStateInterval = setInterval(() => {
          const currentAudio = getAudioElement();
          if (currentAudio) {
            const isActuallyPlaying =
              !currentAudio.paused &&
              currentAudio.currentTime > 0 &&
              !currentAudio.ended;
            const isActuallyPaused =
              currentAudio.paused &&
              currentAudio.currentTime > 0 &&
              !currentAudio.ended;
            const isActuallyEnded = currentAudio.ended;

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
              handleTTSFinished();
            }
          }
        }, 100);

        window.__ttsStateCheckInterval = checkAudioStateInterval;

        // Start streaming and playing TTS
        await streamAndPlayTTS(trimmedText, {
          voice: voice,
          endpoint: '/v1/audio/speech',
          model: 'tts-1',
          speed: 1.0,
        });

        // Set up listeners now that audio should be ready
        const finalAudio = getAudioElement();
        if (finalAudio) {
          setupAudioListeners();
        }

        // Loading timeout fallback
        const loadingTimeout = setTimeout(() => {
          if (mountedRef.current && sharedIsLoading) {
            console.warn('TTS loading timeout - clearing loading state');
            setIsLoadingState(false);
          }
        }, 30000);

        // Cleanup function
        const cleanup = () => {
          clearTimeout(loadingTimeout);
          clearInterval(setupListenersAfterStart);
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
        let message =
          error instanceof Error ? error.message : 'Failed to generate speech';

        // Format HTTP error responses to match expected format
        // Extract status code from error message if present (e.g., "API request failed with status 500")
        // Also check for OpenAI SDK error format which may include status in the error object
        const statusMatch = message.match(/status\s+(\d+)/i);
        let statusCode: string | null = null;
        if (statusMatch) {
          statusCode = statusMatch[1];
        } else if (error && typeof error === 'object' && 'status' in error) {
          // OpenAI SDK errors may have status as a property
          statusCode = String(error.status);
        } else if (error && typeof error === 'object' && 'response' in error) {
          // Some errors wrap the response
          const response = (error as { response?: { status?: number } })
            .response;
          if (response?.status) {
            statusCode = String(response.status);
          }
        }

        if (statusCode) {
          message = `TTS request failed: ${statusCode}`;
        } else if (
          message.includes('library not loaded') ||
          message.includes('TTS library')
        ) {
          message = `TTS library not loaded`;
        } else if (
          message.includes('500') ||
          message.includes('Internal Server Error')
        ) {
          // Fallback: if message mentions 500, format it
          message = `TTS request failed: 500`;
        }

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

  try {
    // Stop any existing playback
    if (currentPlayback) {
      stopTTSOnce();
    }

    // Set up callbacks
    const audio = getAudioElement();
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

    // Use streaming TTS to play
    await streamAndPlayTTS(text.trim(), {
      voice: voice,
      endpoint: '/v1/audio/speech',
      model: 'tts-1',
      speed: 1.0,
    });

    // Wait for playback to complete
    const finalAudio = getAudioElement();
    if (finalAudio) {
      await new Promise<void>(resolve => {
        if (finalAudio.ended) {
          resolve();
        } else {
          const handleEnded = () => {
            finalAudio.removeEventListener('ended', handleEnded);
            resolve();
          };
          finalAudio.addEventListener('ended', handleEnded, { once: true });
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
