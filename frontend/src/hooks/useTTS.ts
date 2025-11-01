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
let sharedAudioContext: AudioContext | null = null;
let sharedCachedAudio: CachedAudio | null = null;
let sharedStartTime: number = 0; // When playback started (AudioContext time)
let sharedPauseTime: number = 0; // How much we've played before pause
let sharedAbortController: AbortController | null = null;
// Shared state so all instances know if audio is playing/paused
let sharedIsPlaying = false;
let sharedIsPaused = false;
// Track the currently playing text so buttons can match against it
let sharedCurrentText: string | null = null;
// Track MediaStream destination for Web Audio to <audio> routing (for iOS background support)
let sharedMediaStreamDestination: MediaStreamAudioDestinationNode | null = null;
// Track metadata for current playback (for Media Session API)
let sharedMetadata: TTSMetadata | null = null;

// Notify all hook instances of state changes (simple listener pattern)
const stateListeners = new Set<() => void>();
function notifyStateListeners() {
  stateListeners.forEach(listener => {
    try {
      listener();
    } catch {}
  });
}

// Helper function to set up Media Session API metadata
function setupMediaSessionMetadata(metadata?: TTSMetadata | null) {
  if ('mediaSession' in navigator && navigator.mediaSession) {
    const title = metadata?.title || 'Text-to-Speech';
    const artist = [metadata?.language, metadata?.level]
      .filter(Boolean)
      .join(' ? ') || 'Language Learning Quiz';
    const album = 'TTS Audio';

    navigator.mediaSession.metadata = new MediaMetadata({
      title,
      artist,
      album,
    });
  }
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

export interface TTSMetadata {
  title?: string;
  language?: string;
  level?: string;
}

interface TTSHookReturn extends TTSState {
  playTTS: (text: string, voice?: string, metadata?: TTSMetadata) => Promise<void>;
  stopTTS: () => void;
  pauseTTS: () => void;
  resumeTTS: () => void;
  restartTTS: () => void;
  currentText: string | null; // The text currently being played/paused
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
        sharedCurrentAudio.srcObject = null;
      } catch {}
      sharedCurrentAudio = null;
    }
    if (currentAudioRef.current) {
      try {
        currentAudioRef.current.pause();
        currentAudioRef.current.currentTime = 0;
        currentAudioRef.current.srcObject = null;
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

    // Clear Web Audio API pause/resume tracking
    sharedStartTime = 0;
    sharedPauseTime = 0;
    sharedCachedAudio = null;
    sharedAudioContext = null;
    sharedMediaStreamDestination = null;
    sharedMetadata = null;

    sharedIsPlaying = false;
    sharedIsPaused = false;
    sharedCurrentText = null; // Clear current text when stopping
    setIsPlaying(false);
    setIsPaused(false);
    setIsLoading(false);
    notifyStateListeners();
  }, [setIsPlaying, setIsPaused]);

  const pauseTTS = useCallback(() => {
    // Handle HTMLAudioElement (MediaSource streaming path)
    const el = sharedCurrentAudio || currentAudioRef.current;
    if (el) {
      try {
        el.pause();
        setIsPaused(true);
        setIsPlaying(false);
        return;
      } catch {}
    }

    // Handle Web Audio API (mobile fallback and cached audio)
    const source = sharedBufferSource || bufferSourceRef.current;
    if (source && sharedAudioContext) {
      try {
        // Calculate how much has played
        const currentTime = sharedAudioContext.currentTime;
        sharedPauseTime += currentTime - sharedStartTime;

        // Stop the current source
        source.onended = null;
        source.stop();
        source.disconnect();

        sharedBufferSource = null;
        if (bufferSourceRef.current === source) {
          bufferSourceRef.current = null;
        }

        setIsPaused(true);
        setIsPlaying(false);
      } catch (e) {
        console.warn('Error pausing Web Audio API playback:', e);
      }
    }
  }, [setIsPlaying, setIsPaused]);

  const resumeTTS = useCallback(() => {
    // Handle HTMLAudioElement (both MediaSource streaming and Web Audio via MediaStream)
    const el = sharedCurrentAudio || currentAudioRef.current;
    if (el) {
      try {
        void el.play();
        setIsPaused(false);
        setIsPlaying(true);
        return;
      } catch {}
    }

    // Fallback: Handle Web Audio API without audio element (legacy path)
    if (sharedCachedAudio && sharedAudioContext) {
      console.log(
        '[TTS Resume] Starting resume, isPaused:',
        sharedIsPaused,
        'pauseTime:',
        sharedPauseTime
      );

      // Resume needs to be async to handle AudioContext resume
      (async () => {
        try {
          console.log(
            '[TTS Resume] AudioContext state:',
            sharedAudioContext!.state
          );

          // Resume the AudioContext (required on iOS after pause)
          if (sharedAudioContext!.state === 'suspended') {
            await sharedAudioContext!.resume();
            console.log(
              '[TTS Resume] AudioContext resumed, new state:',
              sharedAudioContext!.state
            );
          }

          // Recreate MediaStream destination if needed
          let destination = sharedMediaStreamDestination;
          if (!destination) {
            destination = sharedAudioContext!.createMediaStreamDestination();
            sharedMediaStreamDestination = destination;
          }

          // Create new source from cached audio
          const audioBuffer = sharedAudioContext!.createBuffer(
            sharedCachedAudio!.channelData.length,
            sharedCachedAudio!.channelData[0].length,
            sharedCachedAudio!.sampleRate
          );
          for (let ch = 0; ch < sharedCachedAudio!.channelData.length; ch++) {
            const channel = sharedCachedAudio!.channelData[ch];
            const floatChannel = new Float32Array(channel.buffer as ArrayBuffer);
            audioBuffer.copyToChannel(floatChannel, ch, 0);
          }

          const source = sharedAudioContext!.createBufferSource();
          source.buffer = audioBuffer;
          source.connect(destination);

          // Create or reuse audio element
          let audioEl = sharedCurrentAudio || currentAudioRef.current;
          if (!audioEl) {
            audioEl = new Audio();
            audioEl.autoplay = false;
            currentAudioRef.current = audioEl;
            sharedCurrentAudio = audioEl;
          }
          audioEl.srcObject = destination.stream;

          // Set up onended handler
          source.onended = () => {
            console.log('[TTS Resume] Playback ended');
            setIsPlaying(false);
            setIsPaused(false);
            sharedBufferSource = null;
            sharedStartTime = 0;
            sharedPauseTime = 0;
          };

          audioEl.onended = () => {
            setIsPlaying(false);
            setIsPaused(false);
            if (sharedCurrentAudio === audioEl) {
              sharedCurrentAudio = null;
            }
          };

          // Start from the paused position
          const offset = sharedPauseTime;
          const channelLength = sharedCachedAudio!.channelData[0].length;
          const duration = channelLength / sharedCachedAudio!.sampleRate;
          const remaining = Math.max(0, duration - offset);

          console.log(
            '[TTS Resume] Duration:',
            duration,
            'Offset:',
            offset,
            'Remaining:',
            remaining
          );

          if (remaining > 0) {
            // Update state BEFORE starting playback to ensure UI is in sync
            setIsPaused(false);
            setIsPlaying(true);

            source.start(0, offset, remaining);
            sharedStartTime = sharedAudioContext!.currentTime;
            sharedBufferSource = source;
            bufferSourceRef.current = source;

            await audioEl.play();

            console.log('[TTS Resume] Playback started from offset', offset);
          } else {
            // Already at the end, reset to beginning and restart
            console.log(
              '[TTS Resume] At end of audio, restarting from beginning'
            );
            sharedPauseTime = 0;
            sharedStartTime = 0;

            setIsPaused(false);
            setIsPlaying(true);

            source.start();
            sharedStartTime = sharedAudioContext!.currentTime;
            sharedBufferSource = source;
            bufferSourceRef.current = source;

            await audioEl.play();
          }
        } catch (e) {
          console.error(
            '[TTS Resume] Error resuming Web Audio API playback:',
            e
          );
          // Reset state on error
          setIsPlaying(false);
          setIsPaused(false);
        }
      })();
    } else {
      console.warn(
        '[TTS Resume] Cannot resume: missing cached audio or context',
        {
          hasCachedAudio: !!sharedCachedAudio,
          hasContext: !!sharedAudioContext,
          isPaused: sharedIsPaused,
        }
      );
    }
  }, [setIsPlaying, setIsPaused]);

  const restartTTS = useCallback(() => {
    // Handle HTMLAudioElement (both MediaSource streaming and Web Audio via MediaStream)
    const el = sharedCurrentAudio || currentAudioRef.current;
    if (el) {
      try {
        el.currentTime = 0;
        void el.play();
        setIsPaused(false);
        setIsPlaying(true);
        return;
      } catch {}
    }

    // Fallback: Handle Web Audio API without audio element (legacy path)
    if (sharedCachedAudio && sharedAudioContext) {
      (async () => {
        try {
          // Stop current source if playing
          if (sharedBufferSource) {
            try {
              sharedBufferSource.onended = null;
              sharedBufferSource.stop();
              sharedBufferSource.disconnect();
            } catch {}
            sharedBufferSource = null;
          }

          // Reset position
          sharedPauseTime = 0;

          // Resume AudioContext if suspended
          if (sharedAudioContext.state === 'suspended') {
            await sharedAudioContext.resume();
          }

          // Recreate MediaStream destination if needed
          let destination = sharedMediaStreamDestination;
          if (!destination) {
            destination = sharedAudioContext.createMediaStreamDestination();
            sharedMediaStreamDestination = destination;
          }

          // Create new source from cached audio
          const audioBuffer = sharedAudioContext.createBuffer(
            sharedCachedAudio.channelData.length,
            sharedCachedAudio.channelData[0].length,
            sharedCachedAudio.sampleRate
          );
          for (let ch = 0; ch < sharedCachedAudio.channelData.length; ch++) {
            const channel = sharedCachedAudio.channelData[ch];
            const floatChannel = new Float32Array(channel.buffer as ArrayBuffer);
            audioBuffer.copyToChannel(floatChannel, ch, 0);
          }

          const source = sharedAudioContext.createBufferSource();
          source.buffer = audioBuffer;
          source.connect(destination);

          // Create or reuse audio element
          let audioEl = sharedCurrentAudio || currentAudioRef.current;
          if (!audioEl) {
            audioEl = new Audio();
            audioEl.autoplay = false;
            currentAudioRef.current = audioEl;
            sharedCurrentAudio = audioEl;
          }
          audioEl.srcObject = destination.stream;

          source.onended = () => {
            setIsPlaying(false);
            setIsPaused(false);
            sharedBufferSource = null;
            sharedStartTime = 0;
            sharedPauseTime = 0;
          };

          audioEl.onended = () => {
            setIsPlaying(false);
            setIsPaused(false);
            if (sharedCurrentAudio === audioEl) {
              sharedCurrentAudio = null;
            }
          };

          source.start();
          sharedStartTime = sharedAudioContext.currentTime;
          sharedBufferSource = source;
          bufferSourceRef.current = source;

          await audioEl.play();

          setIsPaused(false);
          setIsPlaying(true);
        } catch (e) {
          console.warn('Error restarting Web Audio API playback:', e);
        }
      })();
    }
  }, [setIsPlaying, setIsPaused]);

  const playTTS = useCallback(
    async (text: string, voice?: string, metadata?: TTSMetadata) => {
      if (!text) return;

      // Always stop any existing playback before starting new one to prevent artifacts
      stopTTS();

      // Store the text we're about to play and metadata
      sharedCurrentText = text.trim();
      sharedMetadata = metadata || null;

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

        // If we have cached audio, play it now (using MediaStream for iOS background support)
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
            await ctx.resume();

            // Create an <audio> element to ensure background playback on iOS
            if (sharedCurrentAudio) {
              try {
                sharedCurrentAudio.pause();
                sharedCurrentAudio.srcObject = null;
              } catch {}
              sharedCurrentAudio = null;
            }
            if (currentAudioRef.current) {
              try {
                currentAudioRef.current.pause();
                currentAudioRef.current.srcObject = null;
              } catch {}
              currentAudioRef.current = null;
            }

            const audioEl = new Audio();
            audioEl.autoplay = false;
            currentAudioRef.current = audioEl;
            sharedCurrentAudio = audioEl;

            // Create MediaStreamDestination to route Web Audio to <audio> element
            const destination = ctx.createMediaStreamDestination();
            sharedMediaStreamDestination = destination;

            // Create audio source and connect to MediaStream
            const audioBuffer = ctx.createBuffer(
              cached.channelData.length,
              cached.channelData[0].length,
              cached.sampleRate
            );
            for (let ch = 0; ch < cached.channelData.length; ch++) {
              const channel = cached.channelData[ch];
              const floatChannel = new Float32Array(channel.buffer as ArrayBuffer);
              audioBuffer.copyToChannel(floatChannel, ch, 0);
            }

            const source = ctx.createBufferSource();
            source.buffer = audioBuffer;
            source.connect(destination);

            // Set the MediaStream as the audio element's source
            audioEl.srcObject = destination.stream;

            // Store for pause/resume
            sharedCachedAudio = cached;
            sharedAudioContext = ctx;
            sharedPauseTime = 0;
            bufferSourceRef.current = source;
            sharedBufferSource = source;

            // Set up Media Session API
            setupMediaSessionMetadata(sharedMetadata);
            if ('mediaSession' in navigator && navigator.mediaSession) {
              navigator.mediaSession.setActionHandler('play', () => {
                if (sharedCurrentAudio) {
                  void sharedCurrentAudio.play();
                }
              });

              navigator.mediaSession.setActionHandler('pause', () => {
                if (sharedCurrentAudio) {
                  sharedCurrentAudio.pause();
                }
              });

              navigator.mediaSession.setActionHandler('stop', () => {
                stopTTS();
              });

              const updatePositionState = () => {
                if (
                  navigator.mediaSession &&
                  audioEl.duration &&
                  isFinite(audioEl.duration)
                ) {
                  try {
                    navigator.mediaSession.setPositionState({
                      duration: audioEl.duration || 0,
                      playbackRate: audioEl.playbackRate || 1,
                      position: audioEl.currentTime || 0,
                    });
                  } catch {}
                }
              };

              audioEl.addEventListener('loadedmetadata', updatePositionState);
              audioEl.addEventListener('timeupdate', updatePositionState);
            }

            // Set up event handlers
            source.onended = () => {
              setIsPlaying(false);
              setIsPaused(false);
              sharedBufferSource = null;
              sharedStartTime = 0;
              sharedPauseTime = 0;
              if (sharedCurrentAudio === audioEl) {
                sharedCurrentAudio = null;
              }
              if (currentAudioRef.current === audioEl) {
                currentAudioRef.current = null;
              }
            };

            audioEl.onended = () => {
              setIsPlaying(false);
              setIsPaused(false);
              if (sharedCurrentAudio === audioEl) {
                sharedCurrentAudio = null;
              }
              if (currentAudioRef.current === audioEl) {
                currentAudioRef.current = null;
              }
            };

            // Start playback from user gesture
            source.start();
            sharedStartTime = ctx.currentTime;
            
            await audioEl.play();
            
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

        // MediaSource not available - use Web Audio API routed to <audio> element for mobile
        // This approach downloads all audio data via SSE (preventing timeouts by actively reading the stream),
        // then decodes and plays using Web Audio API routed to an <audio> element via MediaStream.
        // This ensures background playback on iOS Safari, which may suspend pure Web Audio when backgrounded.
        try {
          const controller = new AbortController();
          abortControllerRef.current = controller;
          sharedAbortController = controller;

          // Initialize AudioContext for Web Audio API playback
          if (!audioContextRef.current) {
            audioContextRef.current = new (window.AudioContext ||
              (
                window as unknown as {
                  webkitAudioContext: typeof AudioContext;
                }
              ).webkitAudioContext)();
          }
          const ctx = audioContextRef.current;
          await ctx.resume();

          // Fetch audio via SSE - by continuously reading the stream, we prevent timeout
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
          });

          if (!resp || !resp.ok || !resp.body) {
            throw new Error(`TTS request failed: ${resp?.status || 'unknown'}`);
          }

          const reader = resp.body.getReader();
          const decoder = new TextDecoder();
          const chunks: Uint8Array[] = [];
          let streamError: string | null = null;
          let carry = '';

          // Process SSE stream continuously to prevent timeout
          try {
            while (true) {
              const { done, value } = await reader.read();
              if (done) break;

              const textChunk = decoder.decode(value, { stream: true });
              const combined = carry + textChunk;
              const lines = combined.split(/\r?\n/);
              carry = lines.pop() ?? '';

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
                      chunks.push(bytes);
                    }
                  }
                } catch {}
              }
              if (streamError) break;
            }
          } catch (readError) {
            const name = (readError as { name?: string })?.name || '';
            if (name !== 'AbortError') {
              streamError = 'Stream read error';
            }
          }

          if (streamError) {
            throw new Error(streamError);
          }

          if (chunks.length === 0) {
            throw new Error('No audio data received');
          }

          // Decode all chunks into an AudioBuffer
          const cached = await decodeAudioChunks(chunks, ctx);
          sharedDecodedCache.set(key, cached);

          // Create an <audio> element to ensure background playback on iOS
          // Route Web Audio through MediaStream to the audio element
          if (sharedCurrentAudio) {
            try {
              sharedCurrentAudio.pause();
              sharedCurrentAudio.srcObject = null;
            } catch {}
            sharedCurrentAudio = null;
          }
          if (currentAudioRef.current) {
            try {
              currentAudioRef.current.pause();
              currentAudioRef.current.srcObject = null;
            } catch {}
            currentAudioRef.current = null;
          }

          const audioEl = new Audio();
          audioEl.autoplay = false; // We'll call play() from user gesture
          currentAudioRef.current = audioEl;
          sharedCurrentAudio = audioEl;

          // Create MediaStreamDestination to route Web Audio to <audio> element
          const destination = ctx.createMediaStreamDestination();
          sharedMediaStreamDestination = destination;
          
          // Create audio source and connect to MediaStream
          const audioBuffer = ctx.createBuffer(
            cached.channelData.length,
            cached.channelData[0].length,
            cached.sampleRate
          );
          for (let ch = 0; ch < cached.channelData.length; ch++) {
            const channel = cached.channelData[ch];
            const floatChannel = new Float32Array(channel.buffer as ArrayBuffer);
            audioBuffer.copyToChannel(floatChannel, ch, 0);
          }

          const source = ctx.createBufferSource();
          source.buffer = audioBuffer;
          source.connect(destination); // Connect to MediaStream instead of speakers
          
          // Set the MediaStream as the audio element's source
          audioEl.srcObject = destination.stream;

          // Store for pause/resume
          sharedCachedAudio = cached;
          sharedAudioContext = ctx;
          sharedPauseTime = 0;
          bufferSourceRef.current = source;
          sharedBufferSource = source;

          // Set up Media Session API for lock screen controls
          setupMediaSessionMetadata(sharedMetadata);
          if ('mediaSession' in navigator && navigator.mediaSession) {
            navigator.mediaSession.setActionHandler('play', () => {
              if (sharedCurrentAudio) {
                void sharedCurrentAudio.play();
              }
            });

            navigator.mediaSession.setActionHandler('pause', () => {
              if (sharedCurrentAudio) {
                sharedCurrentAudio.pause();
              }
            });

            navigator.mediaSession.setActionHandler('stop', () => {
              stopTTS();
            });

            // Update position state when duration is available
            const updatePositionState = () => {
              if (
                navigator.mediaSession &&
                audioEl.duration &&
                isFinite(audioEl.duration)
              ) {
                try {
                  navigator.mediaSession.setPositionState({
                    duration: audioEl.duration || 0,
                    playbackRate: audioEl.playbackRate || 1,
                    position: audioEl.currentTime || 0,
                  });
                } catch {
                  // Position state may fail on some browsers
                }
              }
            };

            audioEl.addEventListener('loadedmetadata', updatePositionState);
            audioEl.addEventListener('timeupdate', updatePositionState);
          }

          // Set up event handlers
          source.onended = () => {
            setIsPlaying(false);
            setIsPaused(false);
            sharedBufferSource = null;
            sharedStartTime = 0;
            sharedPauseTime = 0;
            if (sharedAbortController === controller) {
              sharedAbortController = null;
            }
            if (sharedCurrentAudio === audioEl) {
              sharedCurrentAudio = null;
            }
            if (currentAudioRef.current === audioEl) {
              currentAudioRef.current = null;
            }
          };

          audioEl.onended = () => {
            setIsPlaying(false);
            setIsPaused(false);
            if (sharedCurrentAudio === audioEl) {
              sharedCurrentAudio = null;
            }
            if (currentAudioRef.current === audioEl) {
              currentAudioRef.current = null;
            }
          };

          // Start playback from user gesture (critical for iOS)
          source.start();
          sharedStartTime = ctx.currentTime;
          
          // Play the audio element (this must be from user gesture on iOS)
          await audioEl.play();
          
          setIsPlaying(true);
          setIsPaused(false);
          setIsLoading(false);

          return;
        } catch (fallbackError) {
          console.error('Mobile TTS playback failed:', fallbackError);
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
    currentText: sharedCurrentText,
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
