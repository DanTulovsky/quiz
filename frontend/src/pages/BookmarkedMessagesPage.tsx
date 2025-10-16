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
  const messageText =
    typeof message.content === 'string'
      ? message.content
      : message.content?.text || '';

  // Get first ~200 characters for preview
  const preview =
    messageText.substring(0, 200) + (messageText.length > 200 ? '...' : '');

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
        <Text size='sm' style={{ whiteSpace: 'pre-wrap' }}>
          {preview}
        </Text>
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
          <Box size='sm' style={{ maxWidth: 'none' }}>
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                code({ className, children, ...props }: any) {
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

      const responseData = await customInstance({
        url: '/v1/ai/bookmarks',
        method: 'GET',
        params,
      });

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
          ) : messages.length === 0 ? (
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
