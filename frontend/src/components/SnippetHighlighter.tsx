import React, { useMemo, useState } from 'react';
import {
  Popover,
  Stack,
  Text,
  Group,
  Badge,
  Divider,
  ActionIcon,
  Modal,
  Button,
} from '@mantine/core';
import { IconTrash, IconExternalLink } from '@tabler/icons-react';
import { useDisclosure } from '@mantine/hooks';
import { useNavigate } from 'react-router-dom';
import { Snippet, deleteV1SnippetsId } from '../api/api';
import { useQueryClient } from '@tanstack/react-query';
import { useTheme } from '../contexts/ThemeContext';
import { fontScaleMap } from '../theme/theme';

interface SnippetHighlighterProps {
  text: string;
  snippets: Snippet[];
  /** Optional component to wrap the entire text */
  component?: React.ElementType;
  /** Props to pass to the wrapper component */
  componentProps?: Record<string, unknown>;
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
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { fontSize } = useTheme();
  const [
    deleteModalOpened,
    { open: openDeleteModal, close: closeDeleteModal },
  ] = useDisclosure(false);
  const [snippetToDelete, setSnippetToDelete] = useState<number | null>(null);

  const handleDeleteClick = (snippetId: number) => {
    setSnippetToDelete(snippetId);
    openDeleteModal();
  };

  const handleConfirmDelete = async () => {
    if (snippetToDelete) {
      try {
        await deleteV1SnippetsId(snippetToDelete);

        // Invalidate all snippet queries to refresh the UI
        queryClient.invalidateQueries({
          queryKey: ['/v1/snippets'],
        });
        queryClient.invalidateQueries({
          predicate: query => {
            return query.queryKey[0]?.toString().includes('/v1/snippets/');
          },
        });
      } catch (error) {
        console.error('Failed to delete snippet:', error);
      }
    }
    closeDeleteModal();
    setSnippetToDelete(null);
  };

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

          const isWordBoundaryBefore = /[\s.,!?;:()\[\]{}"'«»]/.test(
            beforeChar
          );
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
        <strong key={index} style={{ color: '#1976d2', fontWeight: 700 }}>
          {segment.text}
        </strong>
      );
    }

    // Snippet highlighting (dashed underline with tooltip)
    if (segment.isSnippet) {
      const snippet = segment.snippet!;

      // Create rich tooltip content
      const tooltipContent = (
        <Stack gap='xs'>
          {/* Header with translation and action buttons */}
          <Group justify='space-between' align='flex-start'>
            <Text size='md' fw={500} style={{ flex: 1 }}>
              {snippet.translated_text || 'No translation available'}
            </Text>
            <Group gap='xs'>
              <ActionIcon
                size='md'
                variant='subtle'
                color='blue'
                onClick={e => {
                  e.stopPropagation();
                  // Navigate to snippets page with search for this specific snippet
                  const searchQuery = encodeURIComponent(
                    snippet.original_text || ''
                  );
                  navigate(`/snippets?q=${searchQuery}`);
                }}
                title='View this snippet in snippets page'
              >
                <IconExternalLink size={16} />
              </ActionIcon>
              <ActionIcon
                size='md'
                variant='subtle'
                color='red'
                onClick={e => {
                  e.stopPropagation();
                  handleDeleteClick(snippet.id);
                }}
                title='Delete snippet'
              >
                <IconTrash size={16} />
              </ActionIcon>
            </Group>
          </Group>

          {/* Language pair and difficulty */}
          <Group gap='xs' wrap='wrap'>
            <Badge size='sm' variant='outline' color='blue'>
              {snippet.source_language} → {snippet.target_language}
            </Badge>
            {snippet.difficulty_level && (
              <Badge size='sm' variant='outline' color='green'>
                {snippet.difficulty_level}
              </Badge>
            )}
          </Group>

          {/* Context if available */}
          {snippet.context && (
            <>
              <Divider size='xs' />
              <Text
                size='sm'
                c='dimmed'
                fs='italic'
                style={{
                  wordWrap: 'break-word',
                  overflowWrap: 'break-word',
                  whiteSpace: 'normal',
                  maxWidth: `${280 * fontScaleMap[fontSize]}px`,
                  width: '100%',
                }}
              >
                "{snippet.context}"
              </Text>
            </>
          )}
        </Stack>
      );

      return (
        <Popover
          key={index}
          position='top'
          withArrow
          shadow='md'
          radius='md'
          trigger='click'
          width={300 * fontScaleMap[fontSize]}
          offset={8}
          middlewares={{ flip: true, shift: true, inline: false }}
        >
          <Popover.Target>
            <span
              style={{
                cursor: 'pointer',
                textDecoration: 'none',
                position: 'relative',
              }}
            >
              {segment.text.split(/(\s+)/).map((part, i) => {
                if (/\s+/.test(part)) {
                  return <React.Fragment key={i}>{part}</React.Fragment>;
                }
                return (
                  <span
                    key={i}
                    style={{
                      borderBottom: '1px dashed var(--mantine-color-blue-6)',
                      paddingBottom: '2px',
                    }}
                  >
                    {part}
                  </span>
                );
              })}
            </span>
          </Popover.Target>
          <Popover.Dropdown>{tooltipContent}</Popover.Dropdown>
        </Popover>
      );
    }

    // Fallback for any other case
    return <React.Fragment key={index}>{segment.text}</React.Fragment>;
  });

  // If a wrapper component is specified, use it
  if (Component) {
    return (
      <>
        <Component
          {...componentProps}
          className={`selectable-text ${componentProps.className || ''}`}
          data-selectable-text='true'
        >
          {content}
        </Component>
        <Modal
          opened={deleteModalOpened}
          onClose={closeDeleteModal}
          title='Delete Snippet'
          centered
          zIndex={2000}
        >
          <Stack gap='md'>
            <Text>
              Are you sure you want to delete this snippet? This action cannot
              be undone.
            </Text>
            <Group justify='flex-end'>
              <Button variant='light' onClick={closeDeleteModal}>
                Cancel
              </Button>
              <Button color='red' onClick={handleConfirmDelete}>
                Delete
              </Button>
            </Group>
          </Stack>
        </Modal>
      </>
    );
  }

  // Otherwise, return the content directly with a wrapper to enable translation
  return (
    <>
      <span className='selectable-text' data-selectable-text='true'>
        {content}
      </span>
      <Modal
        opened={deleteModalOpened}
        onClose={closeDeleteModal}
        title='Delete Snippet'
        centered
        zIndex={2000}
      >
        <Stack gap='md'>
          <Text>
            Are you sure you want to delete this snippet? This action cannot be
            undone.
          </Text>
          <Group justify='flex-end'>
            <Button variant='light' onClick={closeDeleteModal}>
              Cancel
            </Button>
            <Button color='red' onClick={handleConfirmDelete}>
              Delete
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
};
