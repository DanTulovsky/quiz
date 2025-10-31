import { useState, useRef, useCallback, useEffect } from 'react';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore - library ships without types
import { TTSRequest } from '../api/api';
import { showNotificationWithClean } from '../notifications';
import {
  fetchSSEAudioChunks,
  decodeAudioChunks,
  createAudioSource,
  getCacheKey,
  type CachedAudio,
} from '../utils/ttsCore';

// Shared decoded audio cache so any hook/component instance can use cached audio
// without relying on the same hook instance.
const sharedDecodedCache = new Map<string, CachedAudio>();
// Shared inflight map to dedupe concurrent streaming requests across instances.
// key -> { promise, controller }
const sharedInflight = new Map<
  string,
  { promise: Promise<void>; controller: AbortController }
>();

// Global current audio element so all hook instances can pause/resume the same playback
let sharedCurrentAudio: HTMLAudioElement | null = null;
let sharedBufferSource: AudioBufferSourceNode | null = null;
let sharedAbortController: AbortController | null = null;
// Shared state so all instances know if audio is playing/paused
let sharedIsPlaying = false;
let sharedIsPaused = false;

// Notify all hook instances of state changes (simple listener pattern)
const stateListeners = new Set<() => void>();
function notifyStateListeners() {
  stateListeners.forEach(listener => {
    try {
      listener();
    } catch {}
  });
}

// Global current playback handle so non-hook callers can stop playback.
let currentPlayback: {
  source: AudioBufferSourceNode;
  ctx: AudioContext;
} | null = null;

export function stopTTSOnce(): void {
  if (!currentPlayback) return;
  try {
    currentPlayback.source.onended = null;
    try {
      currentPlayback.source.stop();
    } catch {}
  } catch {}
  try {
    currentPlayback.ctx.close();
  } catch {}
  currentPlayback = null;
}

interface TTSState {
  isLoading: boolean;
  isPlaying: boolean;
  isPaused: boolean;
}

interface TTSHookReturn extends TTSState {
  playTTS: (text: string, voice?: string) => Promise<void>;
  stopTTS: () => void;
  pauseTTS: () => void;
  resumeTTS: () => void;
  restartTTS: () => void;
}

export const useTTS = (): TTSHookReturn => {
  const [isLoading, setIsLoading] = useState(false);
  const [isPlayingLocal, setIsPlayingLocal] = useState(sharedIsPlaying);
  const [isPausedLocal, setIsPausedLocal] = useState(sharedIsPaused);

  // Sync local state with shared state
  useEffect(() => {
    const updateState = () => {
      setIsPlayingLocal(sharedIsPlaying);
      setIsPausedLocal(sharedIsPaused);
    };
    stateListeners.add(updateState);
    return () => {
      stateListeners.delete(updateState);
    };
  }, []);

  // Helper functions to update both local and shared state
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

  const isPlaying = isPlayingLocal;
  const isPaused = isPausedLocal;
  const audioContextRef = useRef<AudioContext | null>(null);
  const currentAudioRef = useRef<HTMLAudioElement | null>(null);
  const bufferSourceRef = useRef<AudioBufferSourceNode | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const scheduledTimeRef = useRef<number>(0);
  const accumChunksRef = useRef<Uint8Array[]>([]);
  const accumBytesRef = useRef<number>(0);
  // per-instance refs kept for streaming decode state; actual decoded cache is
  // module-scoped above (`sharedDecodedCache`).
  const mountedRef = useRef<boolean>(true);

  const stopTTS = useCallback(() => {
    // Stop the current audio if it's playing (check shared reference)
    if (sharedCurrentAudio) {
      try {
        sharedCurrentAudio.pause();
        sharedCurrentAudio.currentTime = 0;
      } catch {}
      sharedCurrentAudio = null;
    }
    if (currentAudioRef.current) {
      try {
        currentAudioRef.current.pause();
        currentAudioRef.current.currentTime = 0;
      } catch {}
      currentAudioRef.current = null;
    }

    // Abort any in-flight streaming request (check shared reference)
    if (sharedAbortController) {
      try {
        sharedAbortController.abort();
      } catch {
        // Ignore abort errors during cleanup
      }
      sharedAbortController = null;
    }
    if (abortControllerRef.current) {
      try {
        abortControllerRef.current.abort();
      } catch {
        // Ignore abort errors during cleanup
      }
      abortControllerRef.current = null;
    }

    // Stop any active AudioBufferSourceNode (check shared reference)
    if (sharedBufferSource) {
      try {
        sharedBufferSource.onended = null;
        sharedBufferSource.stop();
      } catch {
        // ignore
      }
      try {
        sharedBufferSource.disconnect();
      } catch {
        // ignore
      }
      sharedBufferSource = null;
    }
    if (bufferSourceRef.current) {
      try {
        bufferSourceRef.current.onended = null;
        bufferSourceRef.current.stop();
      } catch {
        // ignore
      }
      try {
        bufferSourceRef.current.disconnect();
      } catch {
        // ignore
      }
      bufferSourceRef.current = null;
    }
    scheduledTimeRef.current = 0;
    accumChunksRef.current = [];
    accumBytesRef.current = 0;

    sharedIsPlaying = false;
    sharedIsPaused = false;
    setIsPlaying(false);
    setIsPaused(false);
    setIsLoading(false);
    notifyStateListeners();
  }, [setIsPlaying, setIsPaused]);

  const pauseTTS = useCallback(() => {
    // Check shared reference first, then instance reference
    const el = sharedCurrentAudio || currentAudioRef.current;
    if (!el) return;
    try {
      el.pause();
      setIsPaused(true);
      setIsPlaying(false);
    } catch {}
  }, []);

  const resumeTTS = useCallback(() => {
    // Check shared reference first, then instance reference
    const el = sharedCurrentAudio || currentAudioRef.current;
    if (!el) return;
    try {
      void el.play();
      setIsPaused(false);
      setIsPlaying(true);
    } catch {}
  }, []);

  const restartTTS = useCallback(() => {
    // Check shared reference first, then instance reference
    const el = sharedCurrentAudio || currentAudioRef.current;
    if (!el) return;
    try {
      el.currentTime = 0;
      void el.play();
      setIsPaused(false);
      setIsPlaying(true);
    } catch {}
  }, []);

  const playTTS = useCallback(
    async (text: string, voice?: string) => {
      if (!text) return;

      // Always stop any existing playback before starting new one to prevent artifacts
      stopTTS();

      try {
        setIsLoading(true);
        setIsPlaying(false);
        setIsPaused(false);

        // Initialize AudioContext to satisfy user-gesture policies; feeder will create its own as needed
        if (!audioContextRef.current) {
          const newAudioContext = new (window.AudioContext ||
            (window as unknown as { webkitAudioContext: typeof AudioContext })
              .webkitAudioContext)();
          audioContextRef.current = newAudioContext;
        }
        try {
          await audioContextRef.current.resume();
        } catch {
          // ignore
        }

        // Check for cached audio (from previous streaming) or inflight streaming request
        const key = getCacheKey(text, voice);
        let cached = sharedDecodedCache.get(key);

        // If there's an inflight streaming request for the same key, wait for it
        if (!cached) {
          const existing = sharedInflight.get(key);
          if (existing) {
            try {
              await existing.promise;
              cached = sharedDecodedCache.get(key);
            } catch {
              // Continue with new streaming if inflight failed
            }
          }
        }

        // If we have cached audio, play it now
        if (cached) {
          try {
            if (!audioContextRef.current) {
              audioContextRef.current = new (window.AudioContext ||
                (
                  window as unknown as {
                    webkitAudioContext: typeof AudioContext;
                  }
                ).webkitAudioContext)();
            }
            const ctx = audioContextRef.current;
            if (bufferSourceRef.current) {
              try {
                bufferSourceRef.current.onended = null;
                bufferSourceRef.current.stop();
              } catch {}
              try {
                bufferSourceRef.current.disconnect();
              } catch {}
              bufferSourceRef.current = null;
            }
            const source = createAudioSource(cached, ctx);
            source.onended = () => {
              setIsPlaying(false);
              setIsPaused(false);
              sharedBufferSource = null;
            };
            source.start();
            bufferSourceRef.current = source;
            sharedBufferSource = source; // Share across instances
            setIsPlaying(true);
            setIsPaused(false);
            setIsLoading(false);
            return; // Use cached audio
          } catch (playError) {
            console.warn('Cached audio playback failed:', playError);
            setIsLoading(false);
            throw playError instanceof Error
              ? playError
              : new Error('Cached audio playback failed');
          }
        }

        const MediaSourceCtor = (
          window as unknown as { MediaSource?: typeof MediaSource }
        ).MediaSource;
        const useStreaming =
          MediaSourceCtor &&
          (() => {
            try {
              if (typeof MediaSourceCtor.isTypeSupported === 'function') {
                return MediaSourceCtor.isTypeSupported('audio/mpeg');
              }
              return true;
            } catch {
              return false;
            }
          })();

        // Use streaming if MediaSource is available and we don't have cached audio
        if (useStreaming) {
          // Progressive streaming: commit to MediaSource, no fallback
          const mediaSource = new MediaSourceCtor!();
          const objectUrl = URL.createObjectURL(mediaSource);

          if (sharedCurrentAudio) {
            try {
              sharedCurrentAudio.pause();
            } catch {}
            sharedCurrentAudio.src = '';
            sharedCurrentAudio = null;
          }
          if (currentAudioRef.current) {
            try {
              currentAudioRef.current.pause();
            } catch {}
            currentAudioRef.current.src = '';
            currentAudioRef.current = null;
          }
          const audioEl = new Audio();
          audioEl.preload = 'auto';
          audioEl.src = objectUrl;
          audioEl.crossOrigin = 'anonymous';
          currentAudioRef.current = audioEl;
          sharedCurrentAudio = audioEl; // Share across instances

          // Create promise for streaming completion (used for deduplication)
          let resolveStreamPromise: (() => void) | undefined;
          const streamPromise = new Promise<void>(resolve => {
            resolveStreamPromise = resolve;
          });

          // Mark as inflight before starting the stream
          const controller = new AbortController();
          abortControllerRef.current = controller;
          sharedAbortController = controller; // Share across instances
          sharedInflight.set(key, { promise: streamPromise, controller });

          let sourceBuffer: SourceBuffer | null = null;
          const pending: Uint8Array[] = [];
          const streamedChunks: Uint8Array[] = []; // Collect chunks for caching
          let ended = false;

          const flush = () => {
            if (!sourceBuffer || sourceBuffer.updating) return;
            const next = pending.shift();
            if (!next) {
              if (ended) {
                try {
                  mediaSource.endOfStream();
                } catch {}
              }
              return;
            }
            try {
              // Ensure we have an ArrayBuffer, not SharedArrayBuffer
              const buffer =
                next.buffer instanceof ArrayBuffer
                  ? next.buffer.slice(
                      next.byteOffset,
                      next.byteOffset + next.byteLength
                    )
                  : new Uint8Array(next).buffer;
              sourceBuffer.appendBuffer(buffer);
            } catch {}
          };

          const onOpen = async () => {
            try {
              sourceBuffer = mediaSource.addSourceBuffer('audio/mpeg');
            } catch {
              try {
                URL.revokeObjectURL(objectUrl);
              } catch {}
              if (currentAudioRef.current) {
                try {
                  currentAudioRef.current.pause();
                } catch {}
                currentAudioRef.current.src = '';
                currentAudioRef.current = null;
              }
              // Gracefully abort without throwing to avoid unhandled rejection on first click
              return;
            }
            sourceBuffer.addEventListener('updateend', flush);

            const resp = await fetch('/v1/audio/speech', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                input: text,
                voice: voice,
                model: 'tts-1',
                stream_format: 'sse',
              } as TTSRequest),
              signal: controller.signal,
            }).catch(() => undefined);
            if (!resp || !resp.ok || !resp.body) {
              try {
                URL.revokeObjectURL(objectUrl);
              } catch {}
              if (currentAudioRef.current) {
                try {
                  currentAudioRef.current.pause();
                } catch {}
                currentAudioRef.current.src = '';
                currentAudioRef.current = null;
              }
              // Avoid throwing; just exit this attempt so user can click again
              return;
            }

            const reader = resp.body.getReader();
            const decoder = new TextDecoder();

            // Start playback on first chunk
            let hasStarted = false;
            const startPlayback = () => {
              if (hasStarted) return;
              hasStarted = true;
              void audioEl
                .play()
                .then(() => {
                  setIsPlaying(true);
                  setIsPaused(false);
                })
                .catch(() => {});
            };

            let streamError: string | null = null;
            let carry = ''; // Buffer for partial SSE lines
            try {
              while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                const textChunk = decoder.decode(value, { stream: true });
                const combined = carry + textChunk;
                const lines = combined.split(/\r?\n/);
                carry = lines.pop() ?? ''; // Keep the last (potentially partial) line
                for (const line of lines) {
                  if (!line.startsWith('data: ')) continue;
                  try {
                    const obj = JSON.parse(line.slice(6));
                    const type =
                      typeof obj?.type === 'string' ? obj.type : undefined;
                    if (type === 'error') {
                      streamError =
                        typeof obj?.error === 'string'
                          ? obj.error
                          : 'Unknown TTS error';
                      try {
                        reader.cancel();
                      } catch {}
                      break;
                    }
                    if (type === 'audio' || type === 'speech.audio.delta') {
                      const b64 =
                        typeof obj?.audio === 'string' ? obj.audio : undefined;
                      if (b64) {
                        const bin = atob(b64);
                        const bytes = new Uint8Array(bin.length);
                        for (let i = 0; i < bin.length; i++)
                          bytes[i] = bin.charCodeAt(i);
                        pending.push(bytes);
                        streamedChunks.push(bytes); // Collect for caching
                        flush();
                        startPlayback();
                      }
                    } else if (type === 'speech.audio.done') {
                      ended = true;
                      flush();
                    }
                  } catch {}
                }
                if (streamError) break;
              }
            } catch (readError) {
              // Handle abort errors gracefully during cleanup
              const name = (readError as { name?: string })?.name || '';
              if (name !== 'AbortError') {
                streamError = 'Stream read error';
              }
            }
            ended = true;
            flush();

            // Cache the streamed audio for future use (if streaming succeeded)
            if (!streamError && streamedChunks.length > 0) {
              try {
                const ctx =
                  audioContextRef.current ||
                  new (window.AudioContext ||
                    (
                      window as unknown as {
                        webkitAudioContext: typeof AudioContext;
                      }
                    ).webkitAudioContext)();
                if (!audioContextRef.current) audioContextRef.current = ctx;
                const cached = await decodeAudioChunks(streamedChunks, ctx);
                sharedDecodedCache.set(key, cached);
                // Clean up inflight entry now that we're cached
                const inflightEntry = sharedInflight.get(key);
                if (inflightEntry?.controller === controller) {
                  sharedInflight.delete(key);
                }
              } catch (cacheError) {
                // Ignore cache errors - streaming playback still works
                console.warn('Failed to cache streamed audio:', cacheError);
              }
            }

            // Resolve the stream promise and clean up inflight entry
            const inflightEntry = sharedInflight.get(key);
            if (inflightEntry?.controller === controller) {
              sharedInflight.delete(key);
            }
            if (resolveStreamPromise) {
              resolveStreamPromise();
            }

            // Keep audio element alive until playback completes
            audioEl.addEventListener(
              'ended',
              () => {
                setIsPlaying(false);
                setIsPaused(false);
                if (sharedCurrentAudio === audioEl) {
                  sharedCurrentAudio = null;
                }
                if (sharedAbortController === controller) {
                  sharedAbortController = null;
                }
                try {
                  URL.revokeObjectURL(objectUrl);
                } catch {}
              },
              { once: true }
            );
            if (streamError) {
              showNotificationWithClean({
                title: 'TTS Error',
                message: streamError,
                color: 'red',
              });
            }
          };

          mediaSource.addEventListener('sourceopen', onOpen, { once: true });
          audioEl.load();
          setIsLoading(false);
          return;
        }

        // MediaSource not available - fallback to blob-based playback for mobile
        try {
          const controller = new AbortController();
          abortControllerRef.current = controller;
          sharedAbortController = controller;

          // Fetch all audio chunks
          const result = await fetchSSEAudioChunks(
            text,
            voice,
            controller.signal,
            undefined
          );

          if (!result.chunks || result.chunks.length === 0) {
            throw new Error('No audio data received');
          }

          // Concatenate all chunks into a single buffer
          const totalLength = result.chunks.reduce(
            (sum, chunk) => sum + chunk.length,
            0
          );
          const combined = new Uint8Array(totalLength);
          let offset = 0;
          for (const chunk of result.chunks) {
            combined.set(chunk, offset);
            offset += chunk.length;
          }

          // Create a blob from the combined audio data
          const blob = new Blob([combined], { type: 'audio/mpeg' });
          const blobUrl = URL.createObjectURL(blob);

          // Clean up any existing audio
          if (sharedCurrentAudio) {
            try {
              sharedCurrentAudio.pause();
            } catch {}
            if (sharedCurrentAudio.src) {
              try {
                URL.revokeObjectURL(sharedCurrentAudio.src);
              } catch {}
            }
            sharedCurrentAudio = null;
          }
          if (currentAudioRef.current) {
            try {
              currentAudioRef.current.pause();
            } catch {}
            if (currentAudioRef.current.src) {
              try {
                URL.revokeObjectURL(currentAudioRef.current.src);
              } catch {}
            }
            currentAudioRef.current = null;
          }

          // Create and play audio element
          const audioEl = new Audio();
          audioEl.preload = 'auto';
          audioEl.src = blobUrl;
          audioEl.crossOrigin = 'anonymous';
          currentAudioRef.current = audioEl;
          sharedCurrentAudio = audioEl;

          // Set up event listeners
          audioEl.addEventListener(
            'ended',
            () => {
              setIsPlaying(false);
              setIsPaused(false);
              if (sharedCurrentAudio === audioEl) {
                sharedCurrentAudio = null;
              }
              if (sharedAbortController === controller) {
                sharedAbortController = null;
              }
              try {
                URL.revokeObjectURL(blobUrl);
              } catch {}
            },
            { once: true }
          );

          audioEl.addEventListener(
            'error',
            () => {
              setIsPlaying(false);
              setIsPaused(false);
              setIsLoading(false);
              if (sharedCurrentAudio === audioEl) {
                sharedCurrentAudio = null;
              }
              try {
                URL.revokeObjectURL(blobUrl);
              } catch {}
              showNotificationWithClean({
                title: 'TTS Error',
                message: 'Failed to play audio',
                color: 'red',
              });
            },
            { once: true }
          );

          // Start playback
          await audioEl.play();
          setIsPlaying(true);
          setIsPaused(false);
          setIsLoading(false);

          // Cache the audio for future use
          try {
            const ctx =
              audioContextRef.current ||
              new (window.AudioContext ||
                (
                  window as unknown as {
                    webkitAudioContext: typeof AudioContext;
                  }
                ).webkitAudioContext)();
            if (!audioContextRef.current) audioContextRef.current = ctx;
            const cached = await decodeAudioChunks(result.chunks, ctx);
            sharedDecodedCache.set(key, cached);
          } catch (cacheError) {
            console.warn('Failed to cache audio:', cacheError);
          }

          return;
        } catch (fallbackError) {
          console.error('Fallback TTS playback failed:', fallbackError);
          throw fallbackError;
        }
      } catch (e) {
        const name = (e as { name?: string })?.name || '';
        const message = (e as { message?: string })?.message || '';
        const isAbort =
          name === 'AbortError' || /aborted|abort(ed)?/i.test(message || '');
        // Suppress abort errors (user-initiated stops or component unmounts)
        if (!isAbort) {
          showNotificationWithClean({
            title: 'TTS Error',
            message: message || 'Failed to generate speech',
            color: 'red',
          });
        }
      } finally {
        setIsLoading(false);
      }
    },
    [isPlaying, isPaused, stopTTS]
  );

  // Cleanup on unmount or route change: stop audio and abort streaming
  useEffect(() => {
    return () => {
      mountedRef.current = false;
      stopTTS();
      if (audioContextRef.current) {
        try {
          audioContextRef.current.close();
        } catch {
          // ignore
        }
        audioContextRef.current = null;
      }
    };
  }, [stopTTS]);

  return {
    isLoading,
    isPlaying,
    isPaused,
    playTTS,
    stopTTS,
    pauseTTS,
    resumeTTS,
    restartTTS,
  };
};

// Convenience helper to play a single TTS sample without using the React hook.
// This is intended to be called directly from a user gesture (e.g. click)
// so it performs its own fetching/decoding and playback using the shared
// caches. It mirrors the hook's logic but does not depend on React lifecycle.
export async function playTTSOnce(
  text: string,
  voice?: string,
  callbacks?: {
    onBuffering?: (p: number) => void;
    onPlayStart?: () => void;
    onPlayEnd?: () => void;
  }
): Promise<void> {
  if (!text) return;
  const key = `${voice ?? ''}::${text}`;

  try {
    // Check cache first
    let cached = sharedDecodedCache.get(key);

    // Ensure an AudioContext exists and is resumed within the user gesture
    const ctx = new (window.AudioContext ||
      (window as unknown as { webkitAudioContext: typeof AudioContext })
        .webkitAudioContext)();
    try {
      // Resume may return a promise; attempt to resume immediately so browsers
      // treat this as a user-initiated gesture where possible.
      void ctx.resume();
    } catch {}

    if (!cached) {
      const chunks = await fetchSSEAudioChunks(
        text,
        voice || undefined,
        undefined,
        p => {
          if (callbacks?.onBuffering) callbacks.onBuffering(p);
        }
      );

      if (chunks.chunks.length > 0) {
        const decoded = await decodeAudioChunks(chunks.chunks, ctx);
        sharedDecodedCache.set(key, decoded);
        cached = sharedDecodedCache.get(key)!;
      }
    }

    if (cached) {
      const source = createAudioSource(cached, ctx);

      // If another playback is active, stop it to avoid overlap
      try {
        if (currentPlayback) stopTTSOnce();
      } catch {}

      // expose as current playback so callers can stop it
      currentPlayback = { source, ctx };

      try {
        if (callbacks?.onPlayStart) callbacks.onPlayStart();
      } catch {}

      await new Promise<void>(resolve => {
        source.onended = () => {
          try {
            if (callbacks?.onPlayEnd) callbacks.onPlayEnd();
          } catch {}
          // clear current playback handle
          try {
            if (currentPlayback && currentPlayback.source === source) {
              currentPlayback = null;
            }
          } catch {}
          resolve();
        };
        source.start();
      });

      // ensure currentPlayback cleared
      try {
        if (currentPlayback && currentPlayback.source === source)
          currentPlayback = null;
      } catch {}

      return;
    }

    throw new Error('TTS playback unavailable: audio not buffered');
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    showNotificationWithClean({ title: 'TTS Error', message, color: 'red' });
    throw error;
  }
}
