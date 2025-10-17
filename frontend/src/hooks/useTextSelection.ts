import { useState, useEffect, useCallback } from 'react';

export interface TextSelection {
  text: string;
  x: number;
  y: number;
  width: number;
  height: number;
}

export const useTextSelection = () => {
  const [selection, setSelection] = useState<TextSelection | null>(null);
  const [isVisible, setIsVisible] = useState(false);

  const handleSelectionChange = useCallback(() => {
    const sel = window.getSelection();

    if (!sel || sel.rangeCount === 0) {
      // Don't hide popup if user is interacting with translation interface
      const isInteractingWithTranslation =
        document.querySelector('.translation-popup') &&
        document.activeElement?.closest('.translation-popup');
      if (!isInteractingWithTranslation) {
        setIsVisible(false);
      }
      return;
    }

    const range = sel.getRangeAt(0);
    const selectedText = sel.toString().trim();

    // Only show popup for meaningful text selections (more than 1 character)
    if (selectedText.length > 1) {
      const rect = range.getBoundingClientRect();

      setSelection({
        text: selectedText,
        x: rect.left + rect.width / 2, // Center of selection
        y: rect.top - 10, // Slightly above selection
        width: rect.width,
        height: rect.height,
      });
      setIsVisible(true);
    } else {
      // Don't hide popup if user is interacting with translation interface
      const isInteractingWithTranslation =
        document.querySelector('.translation-popup') &&
        document.activeElement?.closest('.translation-popup');
      if (!isInteractingWithTranslation) {
        setIsVisible(false);
      }
    }
  }, []);

  // Increased debounce delay to wait for selection completion
  useEffect(() => {
    let debounceTimer: NodeJS.Timeout;

    const debouncedHandler = () => {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(handleSelectionChange, 400); // Increased from 150ms to 400ms
    };

    document.addEventListener('selectionchange', debouncedHandler);
    document.addEventListener('mouseup', debouncedHandler);

    return () => {
      clearTimeout(debounceTimer);
      document.removeEventListener('selectionchange', debouncedHandler);
      document.removeEventListener('mouseup', debouncedHandler);
    };
  }, [handleSelectionChange]);

  const clearSelection = useCallback(() => {
    setIsVisible(false);
    setSelection(null);
    window.getSelection()?.removeAllRanges();
  }, []);

  // Note: Click outside handling is now managed by TranslationPopup component
  // to properly handle Select dropdown interactions

  return {
    selection,
    isVisible,
    clearSelection,
  };
};
