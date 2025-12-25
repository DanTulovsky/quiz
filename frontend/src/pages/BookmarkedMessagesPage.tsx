import React, { useState, useRef, useCallback } from 'react';
import {
  Container,
  Title,
  Text,
  Card,
  Group,
  Badge,
  Button,
  TextInput,
  Stack,
  Modal,
  Box,
  ActionIcon,
  Tooltip,
  Divider,
} from '@mantine/core';
import { Search, BookmarkX } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { useAuth } from '../hooks/useAuth';
import { usePagination } from '../hooks/usePagination';
import { useTheme } from '../contexts/ThemeContext';
import { fontScaleMap } from '../theme/theme';
import { PaginationControls } from '../components/PaginationControls';
import { usePutV1AiConversationsBookmark, ChatMessage } from '../api/api';
import { customInstance } from '../api/axios';
import { useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';

interface MessageCardProps {
  message: ChatMessage;
  onView: (message: ChatMessage) => void;
  onRemoveBookmark: (message: ChatMessage) => void;
}

const MessageCard: React.FC<MessageCardProps> = ({
  message,
  onView,
  onRemoveBookmark,
}) => {
  const { fontSize } = useTheme();
  const messageText =
    typeof message.content === 'string'
      ? message.content
      : message.content?.text || '';

  // Function to create a markdown preview with proper truncation
  const createMarkdownPreview = (text: string, maxLength: number = 200) => {
    if (!text) return '';

    // For very short content, return as is
    if (text.length <= maxLength) return text;

    // Try to truncate at a reasonable markdown boundary
    let truncated = text.substring(0, maxLength);

    // If we're in the middle of markdown syntax, try to find a better break point
    const lastSpace = truncated.lastIndexOf(' ');
    const lastNewline = truncated.lastIndexOf('\n');

    // Prefer breaking at newlines, then spaces
    if (lastNewline > maxLength * 0.7) {
      truncated = truncated.substring(0, lastNewline);
    } else if (lastSpace > maxLength * 0.7) {
      truncated = truncated.substring(0, lastSpace);
    }

    // Add ellipsis if we truncated
    if (truncated.length < text.length) {
      truncated += '...';
    }

    return truncated;
  };

  const preview = createMarkdownPreview(messageText, 200);

  return (
    <Card
      padding='md'
      radius='sm'
      withBorder
      style={{
        cursor: 'pointer',
        transition: 'all 0.2s',
      }}
      onClick={() => onView(message)}
    >
      <Stack gap='xs'>
        <Group justify='space-between' align='center'>
          <Group>
            <Badge color='blue' variant='filled'>
              AI Response
            </Badge>
            <Text size='xs' c='dimmed'>
              {format(new Date(message.created_at), 'MMM d, yyyy h:mm a')}
            </Text>
            {message.conversation_title && (
              <Badge variant='light' color='gray'>
                {message.conversation_title}
              </Badge>
            )}
          </Group>
          <Group gap='xs'>
            <Tooltip label='Remove bookmark'>
              <ActionIcon
                variant='light'
                color='red'
                size='sm'
                onClick={e => {
                  e.stopPropagation();
                  onRemoveBookmark(message);
                }}
              >
                <BookmarkX size={14} />
              </ActionIcon>
            </Tooltip>
          </Group>
        </Group>
        <Box size='sm' data-allow-translate='true'>
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              code({ children, ...props }: any) {
                // For preview, just show code blocks as simple code tags
                return (
                  <code
                    style={{
                      backgroundColor: 'var(--mantine-color-gray-1)',
                      padding: '2px 4px',
                      borderRadius: '3px',
                      fontSize: `${0.85 * fontScaleMap[fontSize]}em`,
                    }}
                    {...props}
                  >
                    {String(children).replace(/\n$/, '')}
                  </code>
                );
              },
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              p: ({ children }: any) => (
                <Box component='p' mb='xs'>
                  {children}
                </Box>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              h1: ({ children }: any) => (
                <Text component='h1' size='lg' fw={600} mb='xs'>
                  {children}
                </Text>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              h2: ({ children }: any) => (
                <Text component='h2' size='md' fw={600} mb='xs'>
                  {children}
                </Text>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              h3: ({ children }: any) => (
                <Text component='h3' size='sm' fw={600} mb='xs'>
                  {children}
                </Text>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              ul: ({ children }: any) => (
                <Box component='ul' mb='xs' ml='md'>
                  {children}
                </Box>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              ol: ({ children }: any) => (
                <Box component='ol' mb='xs' ml='md'>
                  {children}
                </Box>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              li: ({ children }: any) => (
                <Box component='li' mb='2px'>
                  {children}
                </Box>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              blockquote: ({ children }: any) => (
                <Box
                  component='blockquote'
                  mb='xs'
                  style={{
                    borderLeft: '3px solid var(--mantine-color-blue-5)',
                    paddingLeft: '8px',
                    backgroundColor: 'var(--mantine-color-gray-0)',
                  }}
                >
                  {children}
                </Box>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              table: ({ children }: any) => (
                <Box
                  component='table'
                  mb='xs'
                  style={{
                    borderCollapse: 'collapse',
                    width: '100%',
                  }}
                >
                  {children}
                </Box>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              th: ({ children }: any) => (
                <Box
                  component='th'
                  style={{
                    border: '1px solid var(--mantine-color-gray-3)',
                    padding: '4px 8px',
                    textAlign: 'left',
                    fontWeight: 600,
                    backgroundColor: 'var(--mantine-color-gray-1)',
                  }}
                >
                  {children}
                </Box>
              ),
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              td: ({ children }: any) => (
                <Box
                  component='td'
                  style={{
                    border: '1px solid var(--mantine-color-gray-3)',
                    padding: '4px 8px',
                  }}
                >
                  {children}
                </Box>
              ),
            }}
          >
            {preview}
          </ReactMarkdown>
        </Box>
      </Stack>
    </Card>
  );
};

interface MessageDetailModalProps {
  message: ChatMessage | null;
  opened: boolean;
  onClose: () => void;
  onRemoveBookmark: (message: ChatMessage) => void;
  isBookmarking: boolean;
}

const MessageDetailModal: React.FC<MessageDetailModalProps> = ({
  message,
  opened,
  onClose,
  onRemoveBookmark,
  isBookmarking,
}) => {
  if (!message) return null;

  const messageText =
    typeof message.content === 'string'
      ? message.content
      : message.content?.text || '';

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title='Bookmarked Message'
      size='90%'
      styles={{
        content: {
          maxHeight: '80vh',
          display: 'flex',
          flexDirection: 'column',
        },
      }}
    >
      <div style={{ flex: 1, overflow: 'auto', maxHeight: '60vh' }}>
        <Stack gap='md'>
          <Group justify='space-between' align='center'>
            <Group>
              <Badge color='blue' variant='filled'>
                AI Response
              </Badge>
              <Text size='xs' c='dimmed'>
                {format(new Date(message.created_at), 'MMM d, yyyy h:mm a')}
              </Text>
              {message.conversation_title && (
                <Badge variant='light' color='gray'>
                  {message.conversation_title}
                </Badge>
              )}
            </Group>
            <Button
              variant='light'
              color='red'
              size='xs'
              leftSection={<BookmarkX size={14} />}
              onClick={() => onRemoveBookmark(message)}
              disabled={isBookmarking}
            >
              Remove Bookmark
            </Button>
          </Group>
          <Box
            size='sm'
            style={{ maxWidth: 'none' }}
            data-allow-translate='true'
          >
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                code({ children, className, ...props }: any) {
                  const match = /language-(\w+)/.exec(className || '');
                  return match ? (
                    <SyntaxHighlighter
                      className='syntax-highlighter-vsc-dark'
                      language={match[1]}
                      PreTag='div'
                      {...props}
                    >
                      {String(children).replace(/\n$/, '')}
                    </SyntaxHighlighter>
                  ) : (
                    <code className={className} {...props}>
                      {children}
                    </code>
                  );
                },
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                p: ({ children }: any) => (
                  <Box mb='md' component='p'>
                    {children}
                  </Box>
                ),
              }}
            >
              {messageText}
            </ReactMarkdown>
          </Box>
        </Stack>
      </div>
    </Modal>
  );
};

export const BookmarkedMessagesPage: React.FC = () => {
  const {} = useAuth();
  const [searchQuery, setSearchQuery] = useState('');

  const [activeSearchQuery, setActiveSearchQuery] = useState('');
  const [selectedMessage, setSelectedMessage] = useState<ChatMessage | null>(
    null
  );
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const queryClient = useQueryClient();
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Use the pagination hook for better performance and UX
  const {
    data: messages,
    isLoading,
    isFetching,
    pagination,
    goToPage,
    goToNextPage,
    goToPreviousPage,
    reset,
  } = usePagination<ChatMessage>(
    ['/v1/ai/bookmarks', activeSearchQuery],
    async ({ limit, offset }) => {
      const params: { limit: number; offset: number; q?: string } = {
        limit,
        offset,
      };
      if (activeSearchQuery.trim()) {
        params.q = activeSearchQuery.trim();
      }

      const responseData = (await customInstance({
        url: '/v1/ai/bookmarks',
        method: 'GET',
        params,
      })) as { messages?: ChatMessage[]; total?: number };

      return {
        items: responseData.messages || [],
        total: responseData.total || 0,
      };
    },
    {
      initialLimit: 20,
      enableInfiniteScroll: false, // We'll use pagination controls instead
    }
  );

  const bookmarkMessageMutation = usePutV1AiConversationsBookmark(
    {
      mutation: {
        onSuccess: () => {
          // Refresh bookmarked messages
          reset();
          setDetailModalOpen(false);
          setSelectedMessage(null);
        },
      },
    },
    queryClient
  );

  const totalCount = pagination.totalItems;

  // Handle search input change
  const handleSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setSearchQuery(e.target.value);
    },
    []
  );

  // Handle Enter key press to trigger search
  const handleKeyPress = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        setActiveSearchQuery(searchQuery);
        reset(); // Reset pagination when searching
      }
    },
    [searchQuery, reset]
  );

  // Clear search
  const handleClearSearch = () => {
    setSearchQuery('');
    setActiveSearchQuery('');
    reset(); // Reset pagination when clearing search
    setTimeout(() => {
      searchInputRef.current?.focus();
    }, 0);
  };

  // Handle search button click
  const handleSearch = () => {
    setActiveSearchQuery(searchQuery);
    reset(); // Reset pagination when searching
  };

  const handleViewMessage = (message: ChatMessage) => {
    setSelectedMessage(message);
    setDetailModalOpen(true);
  };

  const handleRemoveBookmark = async (message: ChatMessage) => {
    if (!message.conversation_id) return;

    try {
      await bookmarkMessageMutation.mutateAsync({
        data: {
          conversation_id: message.conversation_id,
          message_id: message.id,
        },
      });
    } catch (error) {
      console.error('Failed to remove bookmark:', error);
    }
  };

  return (
    <Container size='xl' py='xl'>
      <Stack gap='xl'>
        <Group justify='space-between' align='center'>
          <div>
            <Title order={1}>Bookmarked Messages</Title>
            <Text c='dimmed' mt='xs'>
              View and manage your bookmarked AI responses
            </Text>
          </div>
          <Badge variant='light' color='blue' size='lg'>
            {totalCount} bookmarked
          </Badge>
        </Group>

        <Card padding='lg' radius='md' withBorder>
          <Group gap='md' mb='lg'>
            <TextInput
              ref={searchInputRef}
              placeholder='Search bookmarked messages...'
              value={searchQuery}
              onChange={handleSearchChange}
              onKeyDown={handleKeyPress}
              leftSection={<Search size={16} />}
              style={{ flex: 1 }}
              disabled={isLoading || isFetching}
            />
            <Group gap='xs'>
              <Button
                variant='filled'
                leftSection={<Search size={16} />}
                onClick={handleSearch}
                disabled={isLoading || isFetching}
              >
                Search
              </Button>
              {(searchQuery || activeSearchQuery) && (
                <Button variant='subtle' onClick={handleClearSearch}>
                  Clear
                </Button>
              )}
            </Group>
          </Group>

          {isLoading ? (
            <Text ta='center' py='xl' c='dimmed'>
              Loading bookmarked messages...
            </Text>
          ) : !messages || messages.length === 0 ? (
            <Text ta='center' py='xl' c='dimmed'>
              {activeSearchQuery
                ? 'No bookmarked messages found matching your search.'
                : 'No bookmarked messages yet. Bookmark messages from conversations to see them here.'}
            </Text>
          ) : (
            <>
              <Stack gap='md'>
                {messages.map(message => (
                  <MessageCard
                    key={message.id}
                    message={message}
                    onView={handleViewMessage}
                    onRemoveBookmark={handleRemoveBookmark}
                  />
                ))}
              </Stack>

              <Divider my='md' />

              <PaginationControls
                pagination={pagination}
                onPageChange={goToPage}
                onNext={goToNextPage}
                onPrevious={goToPreviousPage}
                isLoading={isLoading || isFetching}
                variant='desktop'
              />
            </>
          )}
        </Card>
      </Stack>

      {/* Message Detail Modal */}
      <MessageDetailModal
        message={selectedMessage}
        opened={detailModalOpen}
        onClose={() => {
          setDetailModalOpen(false);
          setSelectedMessage(null);
        }}
        onRemoveBookmark={handleRemoveBookmark}
        isBookmarking={bookmarkMessageMutation.isPending}
      />
    </Container>
  );
};

export default BookmarkedMessagesPage;
