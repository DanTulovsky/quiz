import React from 'react';
import { useTextSelection } from '../hooks/useTextSelection';
import { TranslationPopup } from './TranslationPopup';

export const TranslationOverlay: React.FC = () => {
  const { selection, isVisible, clearSelection } = useTextSelection();

  if (!isVisible || !selection) {
    return null;
  }

  return <TranslationPopup selection={selection} onClose={clearSelection} />;
};
