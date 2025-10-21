import { useState } from 'react';
import { Paper, Text, Box } from '@mantine/core';
import type { PhrasebookWord } from '../utils/phrasebook';

interface WordCardProps {
  word: PhrasebookWord;
}

export function WordCard({ word }: WordCardProps) {
  const [showTranslation, setShowTranslation] = useState(false);

  const handleClick = () => {
    setShowTranslation(prev => !prev);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      setShowTranslation(prev => !prev);
    }
  };

  return (
    <Paper
      p='md'
      withBorder
      style={{
        cursor: 'pointer',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
      }}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role='button'
      aria-label={`${word.term}. Click to ${showTranslation ? 'hide' : 'show'} translation`}
      aria-pressed={showTranslation}
    >
      <Box style={{ flex: 1 }}>
        <Text
          size='lg'
          fw={600}
          mb='xs'
          style={{
            wordWrap: 'break-word',
            overflowWrap: 'break-word',
            hyphens: 'auto',
          }}
        >
          {word.term}
        </Text>
        {word.note && (
          <Text
            size='sm'
            c='dimmed'
            mb='xs'
            fs='italic'
            style={{
              wordWrap: 'break-word',
              overflowWrap: 'break-word',
            }}
          >
            {word.note}
          </Text>
        )}
        {showTranslation && (
          <Text
            size='md'
            c='blue'
            mt='sm'
            style={{
              wordWrap: 'break-word',
              overflowWrap: 'break-word',
            }}
          >
            → {word.translation}
          </Text>
        )}
      </Box>
    </Paper>
  );
}
