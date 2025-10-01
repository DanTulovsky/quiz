import { useState, useRef, useCallback, useEffect } from 'react';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore - library ships without types
import logger from '../utils/logger';
import { TTSRequest } from '../api/api';
import { showNotificationWithClean } from '../notifications';

// Shared decoded audio cache so any hook/component instance can prebuffer and
// later playback without relying on the same hook instance.
const sharedDecodedCache = new Map<
  string,
  { channelData: Float32Array[]; sampleRate: number }
>();
// Shared inflight map to dedupe concurrent prebuffer requests across instances.
// key -> { promise, controller }
const sharedInflight = new Map<
  string,
  { promise: Promise<void>; controller: AbortController }
>();

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
}

type PrebufferSource = 'page' | 'play' | 'other';

interface TTSHookReturn extends TTSState {
  playTTS: (text: string, voice?: string) => Promise<void>;
  stopTTS: () => void;
  prebufferTTS: (
    text: string,
    voice?: string,
    source?: PrebufferSource
  ) => Promise<void>;
  cancelPrebuffer: (text: string, voice?: string) => void;
  isBuffering: boolean;
  bufferingProgress: number; // 0..1
}

export const useTTS = (): TTSHookReturn => {
  const [isLoading, setIsLoading] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const audioContextRef = useRef<AudioContext | null>(null);
  const currentAudioRef = useRef<HTMLAudioElement | null>(null);
  const bufferSourceRef = useRef<AudioBufferSourceNode | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const scheduledTimeRef = useRef<number>(0);
  const accumChunksRef = useRef<Uint8Array[]>([]);
  const accumBytesRef = useRef<number>(0);
  const MIN_DECODE_BYTES = 48_000; // accumulate ~0.5s to reduce artifacts
  // per-instance refs kept for streaming decode state; actual decoded cache is
  // module-scoped above (`sharedDecodedCache`).
  const prebufferAbortRef = useRef<AbortController | null>(null);
  const mountedRef = useRef<boolean>(true);
  const [isBuffering, setIsBuffering] = useState(false);
  const [bufferingProgress, setBufferingProgress] = useState<number>(0);

  const stopTTS = useCallback(() => {
    // Stop the current audio if it's playing
    if (currentAudioRef.current) {
      currentAudioRef.current.pause();
      currentAudioRef.current.currentTime = 0;
      currentAudioRef.current = null;
    }

    // Abort any in-flight streaming request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }

    // Stop any active AudioBufferSourceNode
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

    setIsPlaying(false);
    setIsLoading(false);
  }, []);

  const prebufferTTS = useCallback(
    async (text: string, voice?: string, source: PrebufferSource = 'other') => {
      if (!text) return;
      const key = `${voice ?? ''}::${text}`;
      if (sharedDecodedCache.has(key)) return;

      setIsBuffering(true);

      // If another component already started prebuffering the same key, wait for it.
      const existing = sharedInflight.get(key);
      if (existing) {
        try {
          await existing.promise;
        } finally {
          if (mountedRef.current) setIsBuffering(false);
        }
        return;
      }

      const controller = new AbortController();
      const inflight = (async () => {
        let completedLocal = false;
        try {
          const response = await fetch('/v1/audio/speech', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              input: text,
              voice: voice || 'echo',
              model: 'tts-1',
              stream_format: 'sse',
            } as TTSRequest),
            signal: controller.signal,
          });

          if (!response.ok)
            throw new Error(`TTS request failed: ${response.status}`);

          const reader = response.body?.getReader();
          if (!reader) throw new Error('No response body reader available');
          const decoder = new TextDecoder();

          const chunks: Uint8Array[] = [];
          let bytes = 0;

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            const chunk = decoder.decode(value);
            const lines = chunk.split('\n');
            for (const line of lines) {
              if (line.startsWith('data: ')) {
                try {
                  const rawParsed: unknown = JSON.parse(line.slice(6));
                  if (
                    rawParsed &&
                    typeof rawParsed === 'object' &&
                    !Array.isArray(rawParsed)
                  ) {
                    const obj = rawParsed as Record<string, unknown>;
                    const type =
                      typeof obj.type === 'string' ? obj.type : undefined;
                    if (type === 'audio' || type === 'speech.audio.delta') {
                      const b64 =
                        typeof obj.audio === 'string' ? obj.audio : undefined;
                      if (b64) {
                        const binary = atob(b64);
                        const bytesArr = new Uint8Array(binary.length);
                        for (let i = 0; i < binary.length; i++)
                          bytesArr[i] = binary.charCodeAt(i);
                        chunks.push(bytesArr);
                        bytes += bytesArr.byteLength;
                        // update progress while buffering, capped to MIN_DECODE_BYTES
                        try {
                          const p = Math.min(bytes / MIN_DECODE_BYTES, 1);
                          if (mountedRef.current) setBufferingProgress(p);
                        } catch {
                          // ignore
                        }
                      }
                    }
                  }
                } catch {
                  // ignore parse errors
                }
              }
            }
          }

          if (bytes > 0) {
            const merged = new Uint8Array(bytes);
            let off = 0;
            for (const c of chunks) {
              merged.set(c, off);
              off += c.byteLength;
            }
            const ctx =
              audioContextRef.current ||
              new (window.AudioContext ||
                (
                  window as unknown as {
                    webkitAudioContext: typeof AudioContext;
                  }
                ).webkitAudioContext)();
            try {
              if (!audioContextRef.current) audioContextRef.current = ctx;
              await ctx.resume();
            } catch {}

            const decoded: AudioBuffer = await ctx.decodeAudioData(
              merged.buffer.slice(0)
            );
            const ch = decoded.numberOfChannels;
            const channelData: Float32Array[] = new Array(ch);
            for (let i = 0; i < ch; i++) {
              const src = decoded.getChannelData(i);
              const copy = new Float32Array(src.length);
              copy.set(src);
              channelData[i] = copy;
            }
            sharedDecodedCache.set(key, {
              channelData,
              sampleRate: decoded.sampleRate,
            });
            if (mountedRef.current) setBufferingProgress(1);
            completedLocal = true;
          }
        } catch (e) {
          const name = (e as { name?: string })?.name || '';
          const message = (e as { message?: string })?.message || '';
          const isAbort =
            name === 'AbortError' || /aborted|abort(ed)?/i.test(message || '');
          if (!isAbort) {
            logger.error('Prebuffer TTS error:', e);
            throw e;
          }
        } finally {
          // Clean up inflight entry
          if (sharedInflight.get(key)?.controller === controller)
            sharedInflight.delete(key);
          if (mountedRef.current) setIsBuffering(false);
          if (!completedLocal && mountedRef.current) setBufferingProgress(0);
        }
      })();

      sharedInflight.set(key, { promise: inflight, controller });
      return inflight;
    },
    []
  );

  const playTTS = useCallback(
    async (text: string, voice?: string) => {
      if (!text || isPlaying) return;

      try {
        setIsLoading(true);
        setIsPlaying(false);

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

        const key = `${voice ?? ''}::${text}`;
        // Require a cached decoded buffer. If missing, attempt to prebuffer and wait.
        let cached = sharedDecodedCache.get(key);
        let prebufferError: unknown = undefined;
        if (!cached) {
          try {
            // start prebuffering and wait for it to complete
            await prebufferTTS(text, voice);
            cached = sharedDecodedCache.get(key);
          } catch (e) {
            prebufferError = e;
            logger.error('Prebuffer attempt failed before playback', e);
          }
        }

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
            // Build an AudioBuffer from the cached channel data
            const audioBuffer = ctx.createBuffer(
              cached.channelData.length,
              cached.channelData[0].length,
              cached.sampleRate
            );
            for (let ch = 0; ch < cached.channelData.length; ch++) {
              // copyToChannel expects Float32Array; ensure correct type
              const channel = cached.channelData[ch];
              const floatChannel = new Float32Array(
                channel.buffer as ArrayBuffer
              );
              audioBuffer.copyToChannel(floatChannel, ch, 0);
            }

            // Stop any previous source
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

            const source = ctx.createBufferSource();
            source.buffer = audioBuffer;
            source.connect(ctx.destination);
            source.onended = () => {
              setIsPlaying(false);
            };
            source.start();
            bufferSourceRef.current = source;
            setIsPlaying(true);
            return;
          } catch (e) {
            logger.error('Failed to play cached TTS buffer', e);
          }
        }

        // No cached audio available — prefer to surface original prebuffer error
        if (prebufferError instanceof Error) throw prebufferError;
        throw new Error('TTS playback unavailable: audio not buffered');
      } catch (error) {
        // Suppress user-initiated aborts (Stop button or ESC)
        const name = (error as { name?: string })?.name || '';
        const message = (error as { message?: string })?.message || '';
        const isAbort =
          name === 'AbortError' || /aborted|abort(ed)?/i.test(message || '');
        if (!isAbort) {
          logger.error('TTS error:', error);
          showNotificationWithClean({
            title: 'TTS Error',
            message:
              error instanceof Error
                ? error.message
                : 'Failed to generate speech',
            color: 'red',
          });
        }
      } finally {
        setIsLoading(false);
      }
    },
    [isPlaying]
  );

  // Cleanup on unmount or route change: stop audio and abort streaming
  useEffect(() => {
    return () => {
      mountedRef.current = false;
      // Abort any prebuffering in progress
      if (prebufferAbortRef.current) {
        try {
          prebufferAbortRef.current.abort();
        } catch {
          // ignore
        }
        prebufferAbortRef.current = null;
      }
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

  const cancelPrebuffer = useCallback((text: string, voice?: string) => {
    if (!text) return;
    const key = `${voice ?? ''}::${text}`;
    const entry = sharedInflight.get(key);
    if (entry) {
      try {
        entry.controller.abort();
      } catch {
        // ignore
      }
      sharedInflight.delete(key);
      // Also clear any cached progress state for UI consumers
      if (mountedRef.current) setIsBuffering(false);
      if (mountedRef.current) setBufferingProgress(0);
    }
  }, []);

  return {
    isLoading,
    isPlaying,
    playTTS,
    stopTTS,
    prebufferTTS,
    cancelPrebuffer,
    isBuffering,
    bufferingProgress,
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
      // Fetch SSE stream and accumulate audio chunks
      const response = await fetch('/v1/audio/speech', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          input: text,
          voice: voice || 'echo',
          model: 'tts-1',
          stream_format: 'sse',
        }),
      });

      if (!response.ok)
        throw new Error(`TTS request failed: ${response.status}`);
      const reader = response.body?.getReader();
      if (!reader) throw new Error('No response body reader available');
      const decoder = new TextDecoder();

      const MIN_DECODE_BYTES = 48_000;
      const chunks: Uint8Array[] = [];
      let bytes = 0;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        const chunk = decoder.decode(value);
        const lines = chunk.split('\n');
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const rawParsed: unknown = JSON.parse(line.slice(6));
              if (
                rawParsed &&
                typeof rawParsed === 'object' &&
                !Array.isArray(rawParsed)
              ) {
                const obj = rawParsed as Record<string, unknown>;
                const type =
                  typeof obj.type === 'string' ? obj.type : undefined;
                if (type === 'audio' || type === 'speech.audio.delta') {
                  const b64 =
                    typeof obj.audio === 'string' ? obj.audio : undefined;
                  if (b64) {
                    const binary = atob(b64);
                    const bytesArr = new Uint8Array(binary.length);
                    for (let i = 0; i < binary.length; i++)
                      bytesArr[i] = binary.charCodeAt(i);
                    chunks.push(bytesArr);
                    bytes += bytesArr.byteLength;
                    // update progress while buffering, capped to MIN_DECODE_BYTES
                    try {
                      const p = Math.min(bytes / MIN_DECODE_BYTES, 1);
                      if (callbacks?.onBuffering) callbacks.onBuffering(p);
                    } catch {
                      // ignore
                    }
                  }
                }
              }
            } catch {
              // ignore parse errors
            }
          }
        }
      }

      if (bytes > 0) {
        const merged = new Uint8Array(bytes);
        let off = 0;
        for (const c of chunks) {
          merged.set(c, off);
          off += c.byteLength;
        }

        const decoded: AudioBuffer = await ctx.decodeAudioData(
          merged.buffer.slice(0)
        );
        const ch = decoded.numberOfChannels;
        const channelData: Float32Array[] = new Array(ch);
        for (let i = 0; i < ch; i++) {
          const src = decoded.getChannelData(i);
          const copy = new Float32Array(src.length);
          copy.set(src);
          channelData[i] = copy;
        }
        sharedDecodedCache.set(key, {
          channelData,
          sampleRate: decoded.sampleRate,
        });
        cached = sharedDecodedCache.get(key)!;
      }
    }

    if (cached) {
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
      source.connect(ctx.destination);

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
