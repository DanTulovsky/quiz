import { useState, useEffect, useCallback, useRef } from 'react';
import { extract } from 'sentence-extractor';

export interface TextSelection {
  text: string;
  x: number;
  y: number;
  width: number;
  height: number;
  sentence?: string; // The full sentence containing the selected text
}

/**
 * Extracts the full sentence containing the selected text from the parent element's text content.
 * @param selectedText The text that was selected by the user
 * @param parentElement The parent element containing the selected text
 * @returns The full sentence containing the selected text, or the selected text itself if no sentence can be extracted
 */
const extractSentence = (
  selectedText: string,
  parentElement: HTMLElement | null,
  selectionRange?: Range
): string => {
  if (!parentElement || !selectedText) {
    return selectedText;
  }

  // Get the text content from the parent element
  let fullText = parentElement.textContent || '';

  if (!fullText || !fullText.includes(selectedText)) {
    return selectedText;
  }

  // Walk up the DOM tree to find a container with more context
  // Keep going up until we find a container with substantially more text
  let textContainer = parentElement;
  while (textContainer && textContainer.parentElement) {
    const parentText = textContainer.parentElement.textContent || '';
    // If the parent has more text and includes our selected text, use it
    if (
      parentText.includes(selectedText) &&
      parentText.length > fullText.length
    ) {
      textContainer = textContainer.parentElement;
      fullText = parentText;
    } else {
      break;
    }
  }

  // Use the sentence-extractor library for sentence detection
  // Provide common abbreviations so it doesn't treat short quoted text as complete sentences
  const abbreviations = [
    'Dr',
    'Mr',
    'Mrs',
    'Ms',
    'Prof',
    'etc',
    'vs',
    'Inc',
    'Ltd',
    'Co',
  ];
  const sentences = extract(fullText, abbreviations);

  // Find the sentence that contains the selected text
  // Use the actual selection range to get the correct position
  let selectedIndex = fullText.indexOf(selectedText);

  // If we have a selection range, try to get the actual position
  if (selectionRange) {
    try {
      // Get the text before the selection in the full text
      const range = selectionRange.cloneRange();
      range.setStart(textContainer, 0);
      range.setEnd(selectionRange.startContainer, selectionRange.startOffset);
      const textBeforeSelection = range.toString();
      selectedIndex = textBeforeSelection.length;
    } catch {
      // Fallback to indexOf if range calculation fails
    }
  }

  let bestSentence = null;
  let bestDistance = Infinity;

  for (const sentence of sentences) {
    if (sentence.includes(selectedText)) {
      // Find the position of the selected text within this sentence
      const sentenceStart = fullText.indexOf(sentence);
      const sentenceEnd = sentenceStart + sentence.length;

      // Check if the selected text position falls within this sentence
      if (selectedIndex >= sentenceStart && selectedIndex < sentenceEnd) {
        return sentence.trim();
      }

      // If not, calculate distance from selected text to sentence
      const distance = Math.min(
        Math.abs(selectedIndex - sentenceStart),
        Math.abs(selectedIndex - sentenceEnd)
      );

      if (distance < bestDistance) {
        bestSentence = sentence;
        bestDistance = distance;
      }
    }
  }

  if (bestSentence) {
    return bestSentence.trim();
  }

  // Fallback: return the selected text if we can't extract a sentence
  return selectedText;
};

export const useTextSelection = () => {
  const [selection, setSelection] = useState<TextSelection | null>(null);
  const [isVisible, setIsVisible] = useState(false);
  const visibilityTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const savedSelectionRef = useRef<{
    text: string;
    x: number;
    y: number;
    width: number;
    height: number;
    sentence?: string;
  } | null>(null);
  // Track mouse and touch interaction state to prevent premature popup display
  const isMouseDownRef = useRef<boolean>(false);
  const isTouchActiveRef = useRef<boolean>(false);

  const handleSelectionChange = useCallback(() => {
    // Clear any pending visibility timeout
    if (visibilityTimeoutRef.current) {
      clearTimeout(visibilityTimeoutRef.current);
      visibilityTimeoutRef.current = null;
    }

    // Check if we're in the annotation modal - if so, don't show translation
    const annotationModal = document.querySelector(
      '[data-no-translate="true"]'
    );
    if (annotationModal) {
      setIsVisible(false);
      return;
    }

    const sel = window.getSelection();

    if (!sel || sel.rangeCount === 0) {
      // If selection is cleared but we have a saved one, use it
      if (savedSelectionRef.current) {
        setSelection(savedSelectionRef.current);
        visibilityTimeoutRef.current = setTimeout(() => {
          setIsVisible(true);
          visibilityTimeoutRef.current = null;
        }, 50);
        return;
      }

      // Don't hide popup if translation popup is visible
      // Once the popup is visible, only it should control when it closes
      const isTranslationPopupVisible =
        document.querySelector('.translation-popup');
      if (!isTranslationPopupVisible) {
        setIsVisible(false);
      }
      return;
    }

    const range = sel.getRangeAt(0);
    const selectedText = sel.toString().trim();

    // Only show popup for meaningful text selections (more than 1 character)
    if (selectedText.length > 1) {
      const rect = range.getBoundingClientRect();

      // Mark the selected element as translation-enabled for click detection
      const selectedElement = range.commonAncestorContainer.parentElement;

      // Check if element or any parent has data-no-translate attribute
      if (selectedElement?.closest('[data-no-translate="true"]')) {
        setIsVisible(false);
        return;
      }

      // Allowlist: only show popup within approved content areas
      // - Explicit allow attribute
      // - Reading passage text
      // - Selectable text regions used in content pages
      const allowedContainer = selectedElement?.closest(
        '[data-allow-translate="true"], .reading-passage-text, [data-selectable-text]'
      );
      if (!allowedContainer) {
        setIsVisible(false);
        return;
      }

      // Check if selection is within a Fabric.js canvas (annotation tool)
      // This includes the canvas elements and any Fabric.js text editing elements
      if (
        selectedElement?.closest('.canvas-container') ||
        selectedElement?.closest('.upper-canvas') ||
        selectedElement?.closest('.lower-canvas') ||
        selectedElement?.classList.contains('canvas-container') ||
        selectedElement?.tagName === 'TEXTAREA' ||
        // Check if the modal title contains "Annotate Screenshot"
        document
          .querySelector('[class*="mantine-Modal-title"]')
          ?.textContent?.includes('Annotate Screenshot')
      ) {
        setIsVisible(false);
        return;
      }

      if (selectedElement) {
        selectedElement.setAttribute('data-translation-enabled', 'true');
      }

      // Extract the full sentence containing the selected text
      const sentence = extractSentence(selectedText, selectedElement, range);

      const x = rect.left + rect.width / 2; // Center of selection
      const y = rect.top - 10; // Slightly above selection

      const selectionData = {
        text: selectedText,
        x: x,
        y: y,
        width: rect.width,
        height: rect.height,
        sentence: sentence,
      };

      setSelection(selectionData);

      // Save selection in case iOS clears it
      savedSelectionRef.current = selectionData;

      // Clear text selection to prevent native browser popup
      window.getSelection()?.removeAllRanges();

      // Add a slight delay before showing the popup to make it feel less jarring
      visibilityTimeoutRef.current = setTimeout(() => {
        setIsVisible(true);
        visibilityTimeoutRef.current = null;
      }, 50);
    } else {
      // Don't hide popup if translation popup is visible
      // Once the popup is visible, only it should control when it closes
      const isTranslationPopupVisible =
        document.querySelector('.translation-popup');
      if (!isTranslationPopupVisible) {
        setIsVisible(false);
      }
    }
  }, []);

  // Wait for mouse/touch release before showing popup
  useEffect(() => {
    let debounceTimer: NodeJS.Timeout;

    const debouncedHandler = () => {
      clearTimeout(debounceTimer);
      // Small debounce to ensure selection is stable after mouse/touch release
      debounceTimer = setTimeout(handleSelectionChange, 50);
    };

    // Track mouse down events
    const handleMouseDown = () => {
      isMouseDownRef.current = true;
    };

    // Track mouse up events
    const handleMouseUp = () => {
      isMouseDownRef.current = false;
      debouncedHandler();
    };

    // Track touch start events
    const handleTouchStart = () => {
      isTouchActiveRef.current = true;
    };

    // End touch interaction and trigger selection handling
    const handleTouchEnd = () => {
      isTouchActiveRef.current = false;
      debouncedHandler();
    };

    // Also listen to selectionchange for more reliable selection detection
    // BUT: only trigger if mouse/touch is not currently active
    const selectionChangeHandler = () => {
      // Don't trigger if mouse button is pressed or touch is active
      if (isMouseDownRef.current || isTouchActiveRef.current) {
        return;
      }

      clearTimeout(debounceTimer);
      // Check if there's actually a selection before handling
      const sel = window.getSelection();
      if (sel && sel.rangeCount > 0 && sel.toString().trim().length > 1) {
        debounceTimer = setTimeout(handleSelectionChange, 50);
      }
    };

    // Removed preventNativeMenuOnTouch: allow platform-native touch callout.

    // Track mouse and touch state changes
    document.addEventListener('mousedown', handleMouseDown);
    document.addEventListener('touchstart', handleTouchStart, {
      passive: true,
    });

    // Listen to mouseup and touchend events so popup appears after
    // mouse button is released (desktop) or finger is lifted (mobile)
    document.addEventListener('mouseup', handleMouseUp);
    document.addEventListener('touchend', handleTouchEnd, { passive: true });
    // Also listen to selectionchange for better iOS support (but only when not actively selecting)
    document.addEventListener('selectionchange', selectionChangeHandler);

    return () => {
      clearTimeout(debounceTimer);
      document.removeEventListener('mousedown', handleMouseDown);
      document.removeEventListener('touchstart', handleTouchStart);
      document.removeEventListener('mouseup', handleMouseUp);
      document.removeEventListener('touchend', handleTouchEnd);
      document.removeEventListener('selectionchange', selectionChangeHandler);
    };
  }, [handleSelectionChange]);

  const clearSelection = useCallback(() => {
    // Clear any pending visibility timeout
    if (visibilityTimeoutRef.current) {
      clearTimeout(visibilityTimeoutRef.current);
      visibilityTimeoutRef.current = null;
    }

    // Clear saved selection
    savedSelectionRef.current = null;

    // Clean up data attributes from previously selected elements
    const elementsWithTranslation = document.querySelectorAll(
      '[data-translation-enabled]'
    );
    elementsWithTranslation.forEach(el =>
      el.removeAttribute('data-translation-enabled')
    );

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
