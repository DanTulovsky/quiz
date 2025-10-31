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
      pauseTTS();
      return;
    }
    if (isPaused) {
      resumeTTS();
      return;
    }
    const voice = getVoice ? getVoice() : undefined;
    await playTTS(text, voice);
  };

  const baseLabel = isTTSLoading
    ? 'Loading audio'
    : isTTSPlaying
      ? 'Pause audio'
      : isPaused
        ? 'Resume audio'
        : 'Play audio';
  const label = `${baseLabel} — Alt+Click to restart`;

  const computedColor =
    color || (isTTSPlaying ? 'blue' : isTTSLoading ? 'orange' : 'blue');

  return (
    <Tooltip label={label}>
      <ActionIcon
        size={size}
        variant='subtle'
        color={computedColor}
        onClick={handleClick}
        aria-label={ariaLabel || label}
        disabled={isTTSLoading}
      >
        {isTTSLoading ? (
          <Loader size={16} color='orange' />
        ) : isTTSPlaying ? (
          <Pause size={18} />
        ) : isPaused ? (
          <Play size={18} />
        ) : (
          <Volume2 size={18} />
        )}
      </ActionIcon>
    </Tooltip>
  );
};

export default TTSButton;
