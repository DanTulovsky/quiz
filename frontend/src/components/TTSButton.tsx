import React from 'react';
import { ActionIcon, Tooltip, Loader } from '@mantine/core';
import { Volume2, Pause, Play } from 'lucide-react';
import { useTTS } from '../hooks/useTTS';

interface TTSButtonProps {
  getText: () => string;
  getVoice?: () => string | undefined;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  color?: string;
  ariaLabel?: string;
}

const TTSButton: React.FC<TTSButtonProps> = ({
  getText,
  getVoice,
  size = 'md',
  color,
  ariaLabel,
}) => {
  const {
    isLoading: isTTSLoading,
    isPlaying: isTTSPlaying,
    isPaused,
    playTTS,
    pauseTTS,
    resumeTTS,
    restartTTS,
  } = useTTS();

  // Track if this button was responsible for starting the current playback
  const isOwnerRef = React.useRef(false);
  const lastPlayedTextRef = React.useRef<string>('');

  // Reset ownership when playback stops (but not during loading transitions)
  React.useEffect(() => {
    if (!isTTSPlaying && !isPaused && !isTTSLoading) {
      isOwnerRef.current = false;
      lastPlayedTextRef.current = '';
    }
  }, [isTTSLoading, isTTSPlaying, isPaused]);

  const handleClick: React.MouseEventHandler<HTMLButtonElement> = async e => {
    const text = getText();
    if (!text) return;

    // Alt+Click: restart from beginning
    if (e.altKey) {
      if (isTTSPlaying || isPaused) {
        restartTTS();
        return;
      }
    }

    // Normal click toggles play/pause; if not started, play
    if (isTTSPlaying) {
      // Only pause if this button owns the current playback
      if (isOwnerRef.current && lastPlayedTextRef.current === text) {
        pauseTTS();
      }
      return;
    }
    if (isPaused) {
      // Only resume if this button owns the current playback
      if (isOwnerRef.current && lastPlayedTextRef.current === text) {
        resumeTTS();
      }
      return;
    }
    const voice = getVoice ? getVoice() : undefined;
    isOwnerRef.current = true;
    lastPlayedTextRef.current = text;
    await playTTS(text, voice);
  };

  // Check if this button should show playing/paused state
  // Only show it if this button started the playback AND the text matches
  const isOwned = isOwnerRef.current && lastPlayedTextRef.current === getText();
  const showPlaying = isOwned && isTTSPlaying;
  const showPaused = isOwned && isPaused;
  const showLoading = isOwned && isTTSLoading;

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

  return (
    <Tooltip label={label}>
      <ActionIcon
        size={size}
        variant='subtle'
        color={computedColor}
        onClick={handleClick}
        aria-label={ariaLabel || label}
        disabled={showLoading}
      >
        {showLoading ? (
          <Loader size={16} color='orange' />
        ) : showPlaying ? (
          <Pause size={18} />
        ) : showPaused ? (
          <Play size={18} />
        ) : (
          <Volume2 size={18} />
        )}
      </ActionIcon>
    </Tooltip>
  );
};

export default TTSButton;
