// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore - library ships without types
import { TTSRequest } from '../api/api';

export interface CachedAudio {
  channelData: Float32Array[];
  sampleRate: number;
  rawBytes?: Uint8Array | null;
}

const MIN_DECODE_BYTES = 48_000;

/**
 * Parse SSE stream and extract audio chunks from base64-encoded delta events.
 * Handles partial lines across chunk boundaries.
 */
export async function parseSSEAudioChunks(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  onProgress?: (progress: number) => void
): Promise<{ chunks: Uint8Array[]; bytes: number }> {
  const decoder = new TextDecoder();
  const chunks: Uint8Array[] = [];
  let bytes = 0;
  let carry = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    const textPart = decoder.decode(value, { stream: true });
    const combined = carry + textPart;
    const lines = combined.split(/\r?\n/);
    carry = lines.pop() ?? '';

    for (const line of lines) {
      if (!line.startsWith('data: ')) continue;
      try {
        const rawParsed: unknown = JSON.parse(line.slice(6));
        if (
          rawParsed &&
          typeof rawParsed === 'object' &&
          !Array.isArray(rawParsed)
        ) {
          const obj = rawParsed as Record<string, unknown>;
          const type = typeof obj.type === 'string' ? obj.type : undefined;
          if (type === 'error') {
            const errorMsg =
              typeof obj.error === 'string' ? obj.error : 'Unknown TTS error';
            throw new Error(`TTS server error: ${errorMsg}`);
          }
          if (type === 'audio' || type === 'speech.audio.delta') {
            const b64 = typeof obj.audio === 'string' ? obj.audio : undefined;
            if (b64) {
              const binary = atob(b64);
              const bytesArr = new Uint8Array(binary.length);
              for (let i = 0; i < binary.length; i++)
                bytesArr[i] = binary.charCodeAt(i);
              chunks.push(bytesArr);
              bytes += bytesArr.byteLength;
              if (onProgress) {
                try {
                  const p = Math.min(bytes / MIN_DECODE_BYTES, 1);
                  onProgress(p);
                } catch {}
              }
            }
          }
        }
      } catch (e) {
        // Rethrow error events, ignore parse errors for other data
        if (e instanceof Error && e.message.startsWith('TTS server error:')) {
          throw e;
        }
        // ignore other parse errors
      }
    }
  }

  return { chunks, bytes };
}

/**
 * Fetch TTS audio via SSE and return raw audio chunks.
 */
export async function fetchSSEAudioChunks(
  text: string,
  voice: string | undefined,
  signal?: AbortSignal,
  onProgress?: (progress: number) => void
): Promise<{ chunks: Uint8Array[]; bytes: number }> {
  const response = await fetch('/v1/audio/speech', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      input: text,
      voice: voice || undefined,
      model: 'tts-1',
      stream_format: 'sse',
    } as TTSRequest),
    signal,
  });

  if (!response.ok) throw new Error(`TTS request failed: ${response.status}`);
  const reader = response.body?.getReader();
  if (!reader) throw new Error('No response body reader available');

  return parseSSEAudioChunks(reader, onProgress);
}

/**
 * Decode audio chunks into an AudioBuffer and extract channel data.
 */
export async function decodeAudioChunks(
  chunks: Uint8Array[],
  audioContext: AudioContext
): Promise<CachedAudio> {
  if (chunks.length === 0) {
    throw new Error('No audio chunks to decode');
  }

  const merged = new Uint8Array(
    chunks.reduce((sum, chunk) => sum + chunk.byteLength, 0)
  );
  let off = 0;
  for (const c of chunks) {
    merged.set(c, off);
    off += c.byteLength;
  }

  const decoded: AudioBuffer = await audioContext.decodeAudioData(
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

  return {
    channelData,
    sampleRate: decoded.sampleRate,
    rawBytes: merged,
  };
}

/**
 * Create an AudioBufferSourceNode from cached audio data.
 */
export function createAudioSource(
  cached: CachedAudio,
  audioContext: AudioContext
): AudioBufferSourceNode {
  const audioBuffer = audioContext.createBuffer(
    cached.channelData.length,
    cached.channelData[0].length,
    cached.sampleRate
  );
  for (let ch = 0; ch < cached.channelData.length; ch++) {
    const channel = cached.channelData[ch];
    const floatChannel = new Float32Array(channel.buffer as ArrayBuffer);
    audioBuffer.copyToChannel(floatChannel, ch, 0);
  }

  const source = audioContext.createBufferSource();
  source.buffer = audioBuffer;
  source.connect(audioContext.destination);
  return source;
}

/**
 * Generate cache key for TTS text/voice combination.
 */
export function getCacheKey(text: string, voice?: string): string {
  return `${voice ?? ''}::${text}`;
}
