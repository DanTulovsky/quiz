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

  // Track what text this button is responsible for
  const [ownedText, setOwnedText] = React.useState<string | null>(null);
  const isStartingRef = React.useRef(false);

  // Reset ownership when playback truly ends (not during transitions)
  React.useEffect(() => {
    // Clear starting flag only when playback actually starts (not just when loading)
    // This prevents reset during the gap between loading ending and playing starting
    if (isTTSPlaying) {
      isStartingRef.current = false;

      // Auto-claim ownership if audio is playing and we don't have ownership yet
      // BUT only if the currently playing text matches our text
      // This handles cases where audio was started via hotkey or other means
      if (!ownedText && currentPlayingText) {
        const ourText = getText()?.trim();
        // Only claim ownership if the text matches (handles hotkey scenario)
        if (ourText && ourText === currentPlayingText) {
          setOwnedText(ourText);
        }
      }
    }

    // Auto-claim ownership if audio is paused and we don't have ownership yet
    // BUT only if the currently playing/paused text matches our text
    // This handles cases where audio was paused via hotkey
    if (isPaused && !ownedText && currentPlayingText) {
      const ourText = getText()?.trim();
      // Only claim ownership if the text matches
      if (ourText && ourText === currentPlayingText) {
        setOwnedText(ourText);
      }
    }

    // Don't reset ownership while loading (playback is starting)
    if (isTTSLoading) {
      return;
    }

    // Only reset ownership if playback truly ended AND we're not starting new playback
    if (!isTTSPlaying && !isPaused && !isTTSLoading && !isStartingRef.current) {
      setOwnedText(null);
    }
  }, [
    isTTSLoading,
    isTTSPlaying,
    isPaused,
    ownedText,
    getText,
    currentPlayingText,
  ]);

  const handleClick: React.MouseEventHandler<HTMLButtonElement> = async e => {
    const text = getText();
    if (!text) return;
    const trimmedText = text.trim();
    if (!trimmedText) return; // Don't play empty text

    // Alt+Click: restart from beginning
    if (e.altKey) {
      if (isTTSPlaying || isPaused) {
        restartTTS();
        return;
      }
    }

    // Normal click toggles play/pause; if not started, play
    if (isTTSPlaying) {
      // Always allow pause if audio is playing - ownership check is just for UI state
      pauseTTS();
      return;
    }
    if (isPaused) {
      // Always allow resume if audio is paused - ownership check is just for UI state
      resumeTTS();
      return;
    }
    // Not playing and not paused - start playing
    const voice = getVoice ? getVoice() : undefined;
    const metadata = getMetadata ? getMetadata() : undefined;

    // Set ownership and starting flag BEFORE calling playTTS
    // This ensures ownership is set before any state changes from playTTS
    isStartingRef.current = true;
    setOwnedText(trimmedText);

    // Play the text - this is async and calls stopTTS() first, but isStartingRef prevents reset
    try {
      await playTTS(text, voice, metadata);
    } catch (error) {
      // If playback fails, reset ownership
      setOwnedText(null);
      isStartingRef.current = false;
      throw error;
    }
  };

  // Show state based on ownership and actual audio state
  // If we own any text (setOwnedText was called), show state when audio is in that state
  const isOwned = ownedText !== null;

  // Show playing state if we own it AND audio is playing
  // Show paused state if we own it AND audio is paused
  // Show loading state if we own it AND audio is loading
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
