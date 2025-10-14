import React, { useState, useRef, useMemo, useCallback } from 'react';
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
} from '@mantine/core';
import { Search, ExternalLink, BookmarkX } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { useAuth } from '../hooks/useAuth';
import {
  usePutV1AiConversationsBookmark,
  ChatMessage,
} from '../api/api';
import { useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

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
  const navigate = useNavigate();
  const messageText =
    typeof message.content === 'string'
      ? message.content
      : message.content?.text || '';

  // Get first ~200 characters for preview
  const preview = messageText.substring(0, 200) + (messageText.length > 200 ? '...' : '');

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
            <Tooltip label='View full conversation'>
              <ActionIcon
                variant='light'
                color='blue'
                size='sm'
                onClick={(e) => {
                  e.stopPropagation();
                  navigate(`/conversations`);
                }}
              >
                <ExternalLink size={14} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label='Remove bookmark'>
              <ActionIcon
                variant='light'
                color='red'
                size='sm'
                onClick={(e) => {
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
  const [selectedMessage, setSelectedMessage] = useState<ChatMessage | null>(null);
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const queryClient = useQueryClient();
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Fetch bookmarked messages
  const { data: bookmarksData, isLoading } = useQuery({
    queryKey: ['bookmarkedMessages', activeSearchQuery],
    queryFn: async () => {
      const params = new URLSearchParams({
        limit: '50',
        offset: '0',
      });
      if (activeSearchQuery.trim()) {
        params.append('q', activeSearchQuery);
      }
      
      const response = await fetch(`/api/v1/ai/bookmarks?${params.toString()}`, {
        credentials: 'include',
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch bookmarked messages');
      }
      
      return response.json();
    },
  });

  const bookmarkMessageMutation = usePutV1AiConversationsBookmark(
    {
      mutation: {
        onSuccess: () => {
          // Refresh bookmarked messages
          queryClient.invalidateQueries({ queryKey: ['bookmarkedMessages'] });
          setDetailModalOpen(false);
          setSelectedMessage(null);
        },
      },
    },
    queryClient
  );

  const messages: ChatMessage[] = bookmarksData?.messages || [];
  const totalCount = bookmarksData?.total || messages.length;

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
      }
    },
    [searchQuery]
  );

  // Clear search
  const handleClearSearch = () => {
    setSearchQuery('');
    setActiveSearchQuery('');
    setTimeout(() => {
      searchInputRef.current?.focus();
    }, 0);
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
              disabled={isLoading}
            />
            <Group gap='xs'>
              <Button
                variant='filled'
                leftSection={<Search size={16} />}
                onClick={() => setActiveSearchQuery(searchQuery)}
                disabled={isLoading}
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
            <Stack gap='md'>
              {messages.map((message) => (
                <MessageCard
                  key={message.id}
                  message={message}
                  onView={handleViewMessage}
                  onRemoveBookmark={handleRemoveBookmark}
                />
              ))}
            </Stack>
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
