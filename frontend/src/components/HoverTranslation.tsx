import React, { useState } from 'react';
import { HoverCard, Text, Loader, Alert } from '@mantine/core';
import { useTranslation } from '../contexts/TranslationContext';

interface HoverTranslationProps {
  text: string;
  children: React.ReactNode;
  targetLanguage?: string;
}

export const HoverTranslation: React.FC<HoverTranslationProps> = ({
  text,
  children,
  targetLanguage = 'en',
}) => {
  const [isHovered, setIsHovered] = useState(false);
  const { translateText, translation, isLoading, error } = useTranslation();

  const handleMouseEnter = async () => {
    if (!isHovered && text.trim()) {
      setIsHovered(true);
      try {
        await translateText(text, targetLanguage);
      } catch (err) {
        // Error is handled by the useTranslation hook
        console.error('Translation failed:', err);
      }
    }
  };

  const handleMouseLeave = () => {
    setIsHovered(false);
  };

  return (
    <HoverCard
      width={280}
      shadow='md'
      radius='md'
      position='top'
      withArrow
      openDelay={300}
      closeDelay={100}
      onOpen={handleMouseEnter}
      onClose={handleMouseLeave}
    >
      <HoverCard.Target>
        <span
          style={{
            cursor: 'pointer',
            textDecoration: 'underline',
            textDecorationStyle: 'dotted',
            textDecorationColor: '#868e96',
          }}
        >
          {children}
        </span>
      </HoverCard.Target>
      <HoverCard.Dropdown>
        <div style={{ padding: '8px' }}>
          {isLoading ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Loader size='xs' />
              <Text size='sm' c='dimmed'>
                Translating...
              </Text>
            </div>
          ) : error ? (
            <Alert color='red' size='sm'>
              <Text size='xs'>Translation failed</Text>
            </Alert>
          ) : translation ? (
            <Text size='sm'>{translation.translatedText}</Text>
          ) : (
            <Text size='sm' c='dimmed'>
              Hover to translate
            </Text>
          )}
        </div>
      </HoverCard.Dropdown>
    </HoverCard>
  );
};
