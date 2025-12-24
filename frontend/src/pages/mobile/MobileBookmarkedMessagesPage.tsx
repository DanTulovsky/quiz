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
  Paper,
  Divider,
} from '@mantine/core';
import { Search, BookmarkX, Maximize2 } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { useAuth } from '../../hooks/useAuth';
import { usePagination } from '../../hooks/usePagination';
import { PaginationControls } from '../../components/PaginationControls';
import { usePutV1AiConversationsBookmark, ChatMessage } from '../../api/api';
import { customInstance } from '../../api/axios';
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

  // Get first ~150 characters for preview on mobile
  const preview =
    messageText.substring(0, 150) + (messageText.length > 150 ? '...' : '');

  return (
    <Paper
      radius='sm'
      withBorder
      style={{
        cursor: 'pointer',
        transition: 'all 0.2s',
        padding: '28px', // Generous internal padding for more space between text and border
      }}
      onClick={() => onView(message)}
    >
      <Stack gap='xs'>
        <Group justify='space-between' align='flex-start'>
          <Stack gap={4} style={{ flex: 1 }}>
            <Group gap='xs' wrap='wrap'>
              <Badge color='blue' variant='filled' size='xs'>
                AI
              </Badge>
              <Text size='xs' c='dimmed'>
                {format(new Date(message.created_at), 'MMM d, h:mm a')}
              </Text>
            </Group>
            {message.conversation_title && (
              <Badge variant='light' color='gray' size='xs'>
                {message.conversation_title}
              </Badge>
            )}
          </Stack>
          <Group gap={4}>
            <Tooltip label='Remove'>
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
        <Text size='sm' style={{ whiteSpace: 'pre-wrap' }} lineClamp={3}>
          {preview}
        </Text>
        <Group justify='flex-end'>
          <ActionIcon variant='subtle' color='blue' size='sm'>
            <Maximize2 size={14} />
          </ActionIcon>
        </Group>
      </Stack>
    </Paper>
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
      size='100%'
      fullScreen
      styles={{
        content: {
          height: '100vh',
          display: 'flex',
          flexDirection: 'column',
        },
        body: {
          flex: 1,
          overflow: 'auto',
          padding: '12px',
        },
      }}
    >
      <Stack gap='md'>
        <Group justify='space-between' align='flex-start' wrap='wrap'>
          <Group gap='xs' wrap='wrap'>
            <Badge color='blue' variant='filled' size='sm'>
              AI Response
            </Badge>
            <Text size='xs' c='dimmed'>
              {format(new Date(message.created_at), 'MMM d, yyyy h:mm a')}
            </Text>
            {message.conversation_title && (
              <Badge variant='light' color='gray' size='sm'>
                {message.conversation_title}
              </Badge>
            )}
          </Group>
        </Group>

        <Button
          variant='light'
          color='red'
          size='sm'
          leftSection={<BookmarkX size={16} />}
          onClick={() => onRemoveBookmark(message)}
          disabled={isBookmarking}
          fullWidth
        >
          Remove Bookmark
        </Button>

        <Box
          style={{ maxWidth: '100%', overflow: 'auto' }}
          data-allow-translate='true'
        >
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
    </Modal>
  );
};

export const MobileBookmarkedMessagesPage: React.FC = () => {
  const {} = useAuth();
  const [searchQuery, setSearchQuery] = useState('');
  const [activeSearchQuery, setActiveSearchQuery] = useState('');
  const [selectedMessage, setSelectedMessage] = useState<ChatMessage | null>(
    null
  );
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const queryClient = useQueryClient();
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Force refresh when component mounts or search query changes
  React.useEffect(() => {
    const queryKey = ['/v1/ai/bookmarks', activeSearchQuery];
    queryClient.invalidateQueries({ queryKey });
    queryClient.refetchQueries({ queryKey });
  }, [activeSearchQuery, queryClient]);

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
      initialLimit: 15, // Smaller limit for mobile
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
    <Container size='lg' py='md' px='xs'>
      <Stack gap='md'>
        <Group justify='space-between' align='center'>
          <div>
            <Title order={2} size='h3'>
              Bookmarked Messages
            </Title>
            <Text c='dimmed' size='sm' mt={4}>
              Your saved AI responses
            </Text>
          </div>
          <Badge variant='light' color='blue' size='md'>
            {totalCount}
          </Badge>
        </Group>

        <Card p='sm' radius='md' withBorder>
          <Stack gap='sm'>
            <TextInput
              ref={searchInputRef}
              placeholder='Search bookmarks...'
              value={searchQuery}
              onChange={handleSearchChange}
              onKeyDown={handleKeyPress}
              leftSection={<Search size={16} />}
              disabled={isLoading || isFetching}
              size='sm'
            />
            <Group gap='xs'>
              <Button
                variant='filled'
                size='sm'
                leftSection={<Search size={16} />}
                onClick={handleSearch}
                disabled={isLoading || isFetching}
                style={{ flex: 1 }}
              >
                Search
              </Button>
              {(searchQuery || activeSearchQuery) && (
                <Button variant='subtle' size='sm' onClick={handleClearSearch}>
                  Clear
                </Button>
              )}
            </Group>
          </Stack>
        </Card>

        {isLoading ? (
          <Text ta='center' py='xl' c='dimmed' size='sm'>
            Loading bookmarked messages...
          </Text>
        ) : messages.length === 0 ? (
          <Paper p='xl' radius='md' withBorder>
            <Text ta='center' c='dimmed' size='sm'>
              {activeSearchQuery
                ? 'No bookmarked messages found matching your search.'
                : 'No bookmarked messages yet. Bookmark messages from conversations to see them here.'}
            </Text>
          </Paper>
        ) : (
          <>
            <Stack gap='sm'>
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
              variant='mobile'
            />
          </>
        )}
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

export default MobileBookmarkedMessagesPage;
