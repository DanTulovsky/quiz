import React from 'react';
import { ActionIcon, Button, Tooltip, Loader } from '@mantine/core';
import { Volume2, Pause, Play } from 'lucide-react';
import { useTTS, TTSMetadata } from '../hooks/useTTS';

interface TTSButtonProps {
  getText: () => string;
  getVoice?: () => string | undefined;
  getMetadata?: () => TTSMetadata | undefined;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  color?: string;
  ariaLabel?: string;
  textLabel?: string;
}

const TTSButton: React.FC<TTSButtonProps> = ({
  getText,
  getVoice,
  getMetadata,
  size = 'md',
  color,
  ariaLabel,
  textLabel,
}) => {
  const {
    isLoading: isTTSLoading,
    isPlaying: isTTSPlaying,
    isPaused,
    playTTS,
    pauseTTS,
    resumeTTS,
    restartTTS,
    currentText: currentPlayingText,
  } = useTTS();

  // Simple flag to track if we initiated the current loading state
  // This is only needed to show spinner when currentPlayingText is temporarily null during transition
  const initiatedLoadingRef = React.useRef(false);

  // Clear the flag when playback starts or when our text becomes current
  React.useEffect(() => {
    const ourText = getText()?.trim();
    if (isTTSPlaying && currentPlayingText === ourText) {
      // Our audio started playing - clear the flag
      initiatedLoadingRef.current = false;
    } else if (currentPlayingText === ourText) {
      // Our text is now current (playing or paused) - clear the flag
      initiatedLoadingRef.current = false;
    } else if (currentPlayingText === null && !isTTSLoading) {
      // Audio ended and not loading - clear the flag
      initiatedLoadingRef.current = false;
    }
  }, [isTTSPlaying, currentPlayingText, isTTSLoading, getText]);

  const handleClick: React.MouseEventHandler<HTMLButtonElement> = async e => {
    const text = getText();
    if (!text) return;
    const trimmedText = text.trim();
    if (!trimmedText) return; // Don't play empty text

    const voice = getVoice ? getVoice() : undefined;
    const metadata = getMetadata ? getMetadata() : undefined;

    // Alt+Click: restart from beginning
    if (e.altKey) {
      // Only restart if the currently playing/paused text matches this button's text
      if ((isTTSPlaying || isPaused) && currentPlayingText === trimmedText) {
        const restartSuccess = restartTTS();
        if (restartSuccess) {
          return;
        }
      }
      // Fall back to playing from beginning
      initiatedLoadingRef.current = true;
      try {
        await playTTS(text, voice, metadata);
      } catch (error) {
        initiatedLoadingRef.current = false;
        throw error;
      }
      return;
    }

    // Normal click: toggle play/pause for our text, or start new playback
    if (isTTSPlaying && currentPlayingText === trimmedText) {
      // Our text is playing - pause it
      pauseTTS();
      return;
    }

    if (isPaused && currentPlayingText === trimmedText) {
      // Our text is paused - resume it
      resumeTTS();
      return;
    }

    // Our text is not playing/paused - start new playback
    // Mark that we initiated this loading state
    initiatedLoadingRef.current = true;
    try {
      await playTTS(text, voice, metadata);
    } catch (error) {
      initiatedLoadingRef.current = false;
      throw error;
    }
  };

  // SIMPLIFIED: Each button decides its icon independently based on shared TTS state
  // No ownership needed - just compare our text to what's currently playing
  const ourText = React.useMemo(() => getText()?.trim() || null, [getText]);
  const isOurTextPlaying =
    currentPlayingText !== null && currentPlayingText === ourText;

  // Show loading spinner if:
  // 1. TTS is loading AND
  // 2. Either: our text is current, OR we initiated the load (currentPlayingText might be null during transition)
  const showLoading =
    isTTSLoading &&
    (isOurTextPlaying ||
      (initiatedLoadingRef.current && currentPlayingText === null));

  // Show pause icon if our text is playing
  const showPlaying = isOurTextPlaying && isTTSPlaying && !isTTSLoading;

  // Show play/resume icon if our text is paused
  const showPaused = isOurTextPlaying && isPaused && !isTTSLoading;

  const baseLabel = showLoading
    ? 'Loading audio'
    : showPlaying
      ? 'Pause audio'
      : showPaused
        ? 'Resume audio'
        : 'Play audio';
  const label = `${baseLabel} — Alt+Click to restart`;

  const computedColor =
    color || (showPlaying ? 'blue' : showLoading ? 'orange' : 'blue');

  const IconComponent = showLoading ? (
    <Loader size={16} color='orange' />
  ) : showPlaying ? (
    <Pause size={18} />
  ) : showPaused ? (
    <Play size={18} />
  ) : (
    <Volume2 size={18} />
  );

  return (
    <Tooltip label={label} withinPortal={false}>
      {textLabel ? (
        <Button
          size={size}
          variant='light'
          color={computedColor}
          onClick={handleClick}
          aria-label={ariaLabel || label}
          disabled={showLoading}
          leftSection={IconComponent}
          px={textLabel.length <= 6 ? 6 : 8}
          style={{ whiteSpace: 'nowrap', minWidth: 'auto' }}
        >
          {textLabel}
        </Button>
      ) : (
        <ActionIcon
          size={size}
          variant='subtle'
          color={computedColor}
          onClick={handleClick}
          aria-label={ariaLabel || label}
          disabled={showLoading}
        >
          {IconComponent}
        </ActionIcon>
      )}
    </Tooltip>
  );
};

export default TTSButton;
