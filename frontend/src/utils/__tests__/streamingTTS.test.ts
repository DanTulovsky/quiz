import { describe, expect, it } from 'vitest';

import {
  classifySafariPlaybackError,
  type SafariPlaybackErrorClassification,
} from '../streamingTTS';

const MEDIA_ERR_SRC_NOT_SUPPORTED_CODE = 4;

type PartialAudioShape = Pick<
  HTMLAudioElement,
  'paused' | 'ended' | 'currentTime' | 'readyState' | 'error'
>;

function createAudio(overrides: Partial<PartialAudioShape> = {}) {
  const base: PartialAudioShape = {
    paused: false,
    ended: false,
    currentTime: 5,
    readyState: 3,
    error: null,
  };

  return {
    ...base,
    ...overrides,
  } as unknown as HTMLAudioElement;
}

function createErrorEvent(message?: string): Event {
  if (typeof ErrorEvent !== 'undefined') {
    return new ErrorEvent('error', { message });
  }

  const fallback = new Event('error');
  if (message) {
    Object.defineProperty(fallback, 'message', {
      value: message,
      configurable: true,
    });
  }
  return fallback;
}

describe('classifySafariPlaybackError', () => {
  it('treats Safari format errors as recoverable when playback continues', () => {
    const audioError = {
      code: MEDIA_ERR_SRC_NOT_SUPPORTED_CODE,
      message: 'Audio format not supported by Safari',
    } as MediaError;

    const classification: SafariPlaybackErrorClassification =
      classifySafariPlaybackError({
        audio: createAudio({ error: audioError }),
        audioError,
        playbackStarted: true,
        event: createErrorEvent('Audio format not supported by Safari'),
      });

    expect(classification.recoverable).toBe(true);
    expect(classification.reason).toBe('safari-format-false-positive');
  });

  it('treats format errors as fatal when playback never progressed', () => {
    const audioError = {
      code: MEDIA_ERR_SRC_NOT_SUPPORTED_CODE,
      message: 'Audio format not supported by Safari',
    } as MediaError;

    const classification = classifySafariPlaybackError({
      audio: createAudio({ currentTime: 0, error: audioError }),
      audioError,
      playbackStarted: true,
      event: createErrorEvent('Audio format not supported by Safari'),
    });

    expect(classification.recoverable).toBe(false);
    expect(classification.reason).toBe('no-playback-progress');
  });

  it('allows paused playback with buffered data to recover', () => {
    const audioError = {
      code: MEDIA_ERR_SRC_NOT_SUPPORTED_CODE,
      message: 'Audio format not supported by Safari',
    } as MediaError;

    const classification = classifySafariPlaybackError({
      audio: createAudio({
        paused: true,
        readyState: 4,
        error: audioError,
      }),
      audioError,
      playbackStarted: true,
      event: createErrorEvent('Audio format not supported by Safari'),
    });

    expect(classification.recoverable).toBe(true);
    expect(classification.reason).toBe('safari-format-false-positive');
  });

  it('treats errors before playback starts as fatal', () => {
    const audioError = {
      code: MEDIA_ERR_SRC_NOT_SUPPORTED_CODE,
      message: 'Audio format not supported by Safari',
    } as MediaError;

    const classification = classifySafariPlaybackError({
      audio: createAudio({ error: audioError }),
      audioError,
      playbackStarted: false,
      event: createErrorEvent('Audio format not supported by Safari'),
    });

    expect(classification.recoverable).toBe(false);
    expect(classification.reason).toBe('playback-not-started');
  });

  it('treats intentional shutdowns as recoverable even without progress', () => {
    const classification = classifySafariPlaybackError({
      audio: createAudio({ currentTime: 0 }),
      audioError: {
        code: MEDIA_ERR_SRC_NOT_SUPPORTED_CODE,
        message: 'Audio format not supported by Safari',
      } as MediaError,
      playbackStarted: true,
      event: createErrorEvent('Audio format not supported by Safari'),
      intentionalShutdown: true,
      intentionalShutdownReason: 'stop',
    });

    expect(classification.recoverable).toBe(true);
    expect(classification.reason).toBe('intentional-shutdown:stop');
  });
});
