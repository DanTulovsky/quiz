import React, { useMemo } from 'react';
import { Tooltip } from '@mantine/core';
import { Snippet } from '../api/api';

interface SnippetHighlighterProps {
  text: string;
  snippets: Snippet[];
  /** Optional component to wrap the entire text */
  component?: React.ElementType;
  /** Props to pass to the wrapper component */
  componentProps?: Record<string, any>;
  /** Optional target word to highlight (for vocabulary questions) */
  targetWord?: string;
}

interface HighlightSegment {
  text: string;
  isSnippet: boolean;
  isTargetWord: boolean;
  snippet?: Snippet;
}

/**
 * SnippetHighlighter component highlights saved snippets in text with subtle
 * dashed underlines and shows translations on hover.
 *
 * Performance optimizations:
 * - Uses efficient string matching algorithm
 * - Memoizes processing to avoid re-computation
 * - Only re-renders when text or snippets change
 */
export const SnippetHighlighter: React.FC<SnippetHighlighterProps> = ({
  text,
  snippets,
  component: Component,
  componentProps = {},
  targetWord,
}) => {
  const segments = useMemo(() => {
    // If no snippets and no target word, return the original text as a single segment
    if ((!snippets || snippets.length === 0) && !targetWord) {
      return [{ text, isSnippet: false, isTargetWord: false }];
    }

    // Build a map of all highlights (snippets and target word) in the text
    const matches: Array<{
      start: number;
      end: number;
      snippet?: Snippet;
      isTargetWord: boolean;
    }> = [];

    // Add snippet matches
    if (snippets && snippets.length > 0) {
      snippets.forEach(snippet => {
        if (!snippet.original_text) return;

        const searchText = text.toLowerCase();
        const snippetText = snippet.original_text.toLowerCase();
        let startIndex = 0;

        // Find all occurrences of this snippet in the text
        while (startIndex < searchText.length) {
          const index = searchText.indexOf(snippetText, startIndex);
          if (index === -1) break;

          // Check for word boundaries (optional: helps avoid partial matches)
          const beforeChar = index > 0 ? text[index - 1] : ' ';
          const afterChar =
            index + snippetText.length < text.length
              ? text[index + snippetText.length]
              : ' ';

          const isWordBoundaryBefore = /[\s.,!?;:()\[\]{}"'«»]/.test(beforeChar);
          const isWordBoundaryAfter = /[\s.,!?;:()\[\]{}"'«»]/.test(afterChar);

          // Only match if it's a whole word (or at start/end of text)
          if (isWordBoundaryBefore && isWordBoundaryAfter) {
            matches.push({
              start: index,
              end: index + snippet.original_text.length,
              snippet,
              isTargetWord: false,
            });
          }

          startIndex = index + 1; // Move forward to find next occurrence
        }
      });
    }

    // Add target word matches
    if (targetWord) {
      const searchText = text.toLowerCase();
      const targetText = targetWord.toLowerCase();
      let startIndex = 0;

      while (startIndex < searchText.length) {
        const index = searchText.indexOf(targetText, startIndex);
        if (index === -1) break;

        // Check for word boundaries
        const beforeChar = index > 0 ? text[index - 1] : ' ';
        const afterChar =
          index + targetWord.length < text.length
            ? text[index + targetWord.length]
            : ' ';

        const isWordBoundaryBefore = /[\s.,!?;:()\[\]{}"'«»]/.test(beforeChar);
        const isWordBoundaryAfter = /[\s.,!?;:()\[\]{}"'«»]/.test(afterChar);

        if (isWordBoundaryBefore && isWordBoundaryAfter) {
          matches.push({
            start: index,
            end: index + targetWord.length,
            isTargetWord: true,
          });
        }

        startIndex = index + 1;
      }
    }

    // Sort matches by start position
    matches.sort((a, b) => a.start - b.start);

    // Resolve overlapping matches (prioritize snippets over target word)
    const nonOverlapping: typeof matches = [];
    let lastEnd = -1;
    for (const match of matches) {
      if (match.start >= lastEnd) {
        nonOverlapping.push(match);
        lastEnd = match.end;
      }
    }

    // Build segments from matches
    const result: HighlightSegment[] = [];
    let currentPos = 0;

    for (const match of nonOverlapping) {
      // Add non-highlighted text before this match
      if (match.start > currentPos) {
        result.push({
          text: text.slice(currentPos, match.start),
          isSnippet: false,
          isTargetWord: false,
        });
      }

      // Add highlighted text
      result.push({
        text: text.slice(match.start, match.end),
        isSnippet: !!match.snippet,
        isTargetWord: match.isTargetWord,
        snippet: match.snippet,
      });

      currentPos = match.end;
    }

    // Add remaining non-highlighted text
    if (currentPos < text.length) {
      result.push({
        text: text.slice(currentPos),
        isSnippet: false,
        isTargetWord: false,
      });
    }

    return result;
  }, [text, snippets, targetWord]);

  const content = segments.map((segment, index) => {
    // Regular text - no highlighting
    if (!segment.isSnippet && !segment.isTargetWord) {
      return <React.Fragment key={index}>{segment.text}</React.Fragment>;
    }

    // Target word highlighting (blue and bold)
    if (segment.isTargetWord && !segment.isSnippet) {
      return (
        <strong
          key={index}
          style={{ color: '#1976d2', fontWeight: 700 }}
        >
          {segment.text}
        </strong>
      );
    }

    // Snippet highlighting (dashed underline with tooltip)
    if (segment.isSnippet) {
      const snippet = segment.snippet!;
      const tooltipLabel = snippet.translated_text || 'No translation available';

      return (
        <Tooltip
          key={index}
          label={tooltipLabel}
          position="top"
          withArrow
          multiline
          maw={300}
          transitionProps={{ duration: 150 }}
        >
          <span
            style={{
              borderBottom: '1px dashed #228be6',
              cursor: 'help',
              textDecoration: 'none',
            }}
          >
            {segment.text}
          </span>
        </Tooltip>
      );
    }

    // Fallback for any other case
    return <React.Fragment key={index}>{segment.text}</React.Fragment>;
  });

  // If a wrapper component is specified, use it
  if (Component) {
    return <Component {...componentProps}>{content}</Component>;
  }

  // Otherwise, return the content directly
  return <>{content}</>;
};

