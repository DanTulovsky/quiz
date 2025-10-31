import React, { useState, useRef, useCallback, useEffect } from 'react';
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
  ActionIcon,
  Menu,
  Modal,
  Box,
  Divider,
} from '@mantine/core';
import {
  Search,
  Edit,
  Trash2,
  MessageCircle,
  Calendar,
  Bookmark,
} from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { useAuth } from '../hooks/useAuth';
import { usePagination } from '../hooks/usePagination';
import { PaginationControls } from '../components/PaginationControls';
import {
  useGetV1AiConversationsId,
  useDeleteV1AiConversationsId,
  usePutV1AiConversationsId,
  usePutV1AiConversationsBookmark,
  getGetV1AiConversationsIdQueryKey,
  Conversation,
  ChatMessage,
} from '../api/api';
import { customInstance } from '../api/axios';
import { useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';

interface ConversationCardProps {
  conversation: Conversation;
  onEdit: (conversation: Conversation) => void;
  onDelete: (conversationId: string) => void;
  onView: (conversation: Conversation) => void;
}

const ConversationCard: React.FC<ConversationCardProps> = ({
  conversation,
  onEdit,
  onDelete,
  onView,
}) => {
  // Derive count from optional fields to avoid fetching messages list
  const messageCount =
    (conversation as unknown as { message_count?: number }).message_count ??
    conversation.messages?.length ??
    0;

  // Create a preview snippet based on the conversation title
  const createTitlePreview = (title: string, maxLength: number = 100) => {
    if (!title || title === 'Untitled Conversation') {
      return 'Click to view conversation';
    }

    if (title.length <= maxLength) return title;

    // Try to truncate at a reasonable boundary
    let truncated = title.substring(0, maxLength);

    // If we're in the middle of a word, try to find a better break point
    const lastSpace = truncated.lastIndexOf(' ');

    if (lastSpace > maxLength * 0.7) {
      truncated = truncated.substring(0, lastSpace);
    }

    // Add ellipsis if we truncated
    if (truncated.length < title.length) {
      truncated += '...';
    }

    return truncated;
  };

  const previewText = createTitlePreview(
    conversation.title || 'Untitled Conversation'
  );

  return (
    <Group
      justify='space-between'
      align='center'
      py='sm'
      px='xs'
      style={{
        borderBottom: '1px solid var(--mantine-color-gray-2)',
        cursor: 'pointer',
        '&:hover': {
          backgroundColor: 'var(--mantine-color-gray-0)',
        },
      }}
      onClick={() => onView(conversation)}
    >
      <Group align='center' gap='md' style={{ flex: 1 }}>
        <Box style={{ flex: 1, minWidth: 0 }}>
          <Text
            size='sm'
            fw={500}
            mb={2}
            style={{
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {conversation.title || 'Untitled Conversation'}
          </Text>
          <Text
            size='xs'
            c='dimmed'
            style={{
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {previewText}
          </Text>
        </Box>

        <Badge
          variant='light'
          color='blue'
          size='xs'
          leftSection={<Calendar size={10} />}
        >
          {format(new Date(conversation.created_at), 'MMM d, h:mm a')}
        </Badge>

        <Badge variant='light' color='green' size='xs'>
          {messageCount} {messageCount === 1 ? 'msg' : 'msgs'}
        </Badge>
      </Group>

      <Menu shadow='md' width={120}>
        <Menu.Target>
          <ActionIcon
            aria-label='Conversation actions'
            variant='subtle'
            color='gray'
            size='sm'
            onClick={e => e.stopPropagation()}
          >
            <Edit size={14} />
          </ActionIcon>
        </Menu.Target>
        <Menu.Dropdown>
          <Menu.Item
            leftSection={<MessageCircle size={16} />}
            onClick={e => {
              e.stopPropagation();
              onView(conversation);
            }}
          >
            View
          </Menu.Item>
          <Menu.Item
            leftSection={<Edit size={16} />}
            onClick={e => {
              e.stopPropagation();
              onEdit(conversation);
            }}
          >
            Edit Title
          </Menu.Item>
          <Menu.Item
            leftSection={<Trash2 size={16} />}
            color='red'
            onClick={e => {
              e.stopPropagation();
              onDelete(conversation.id);
            }}
          >
            Delete
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>
    </Group>
  );
};

interface ConversationDetailModalProps {
  conversation: Conversation | null;
  opened: boolean;
  onClose: () => void;
  messages: ChatMessage[];
  onBookmarkToggle: (messageId: string) => void;
  isBookmarking: boolean;
}

const ConversationDetailModal: React.FC<ConversationDetailModalProps> = ({
  conversation,
  opened,
  onClose,
  messages,
  onBookmarkToggle,
  isBookmarking,
}) => {
  if (!conversation) return null;

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={conversation.title || 'Untitled Conversation'}
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
        <Stack gap='sm'>
          {messages.map((message, index) => {
            const messageText =
              typeof message.content === 'string'
                ? message.content
                : message.content?.text || '';

            return (
              <Card
                key={message.id || index}
                padding='md'
                radius='sm'
                withBorder
                style={{
                  backgroundColor:
                    message.role === 'user'
                      ? 'var(--mantine-primary-color-1)'
                      : 'var(--mantine-color-body)',
                }}
              >
                <Group mb='xs' justify='space-between'>
                  <Group>
                    <Badge
                      color={message.role === 'user' ? 'blue' : 'green'}
                      variant='filled'
                    >
                      {message.role === 'user' ? 'You' : 'AI'}
                    </Badge>
                    <Text size='xs' c='dimmed'>
                      {format(new Date(message.created_at), 'MMM d, h:mm a')}
                    </Text>
                  </Group>
                  {message.role === 'assistant' && (
                    <Button
                      variant='light'
                      size='xs'
                      leftSection={<Bookmark size={14} />}
                      onClick={() => onBookmarkToggle(message.id)}
                      disabled={isBookmarking}
                      color={message.bookmarked ? 'blue' : undefined}
                      style={{
                        opacity: message.bookmarked ? 1 : 0.7,
                      }}
                    >
                      {message.bookmarked ? 'Bookmarked' : 'Bookmark'}
                    </Button>
                  )}
                </Group>
                <Box size='sm' style={{ maxWidth: 'none' }} data-allow-translate='true'>
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
              </Card>
            );
          })}
        </Stack>
      </div>
    </Modal>
  );
};

export const SavedConversationsPage: React.FC = () => {
  const {} = useAuth();
  const [searchQuery, setSearchQuery] = useState('');
  const [activeSearchQuery, setActiveSearchQuery] = useState('');
  const [selectedConversation, setSelectedConversation] =
    useState<Conversation | null>(null);
  const [conversationMessages, setConversationMessages] = useState<
    ChatMessage[]
  >([]);
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingConversation, setEditingConversation] =
    useState<Conversation | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [conversationToDelete, setConversationToDelete] = useState<
    string | null
  >(null);

  const queryClient = useQueryClient();
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Use pagination hook for conversations
  const {
    data: conversations,
    isLoading: conversationsLoading,
    isFetching: conversationsFetching,
    pagination: conversationsPagination,
    goToPage: goToConversationsPage,
    goToNextPage: goToNextConversationsPage,
    goToPreviousPage: goToPreviousConversationsPage,
    reset: resetConversations,
  } = usePagination<Conversation>(
    ['/v1/ai/conversations', activeSearchQuery],
    async ({ limit, offset }) => {
      if (activeSearchQuery.trim()) {
        // Use search API
        const params: { q: string; limit: number; offset: number } = {
          q: activeSearchQuery.trim(),
          limit,
          offset,
        };
        const responseData = await customInstance({
          url: '/v1/ai/search',
          method: 'GET',
          params,
        });
        return {
          items: responseData.conversations || [],
          total: responseData.total || 0,
        };
      } else {
        // Use conversations API
        const responseData = await customInstance({
          url: '/v1/ai/conversations',
          method: 'GET',
          params: { limit, offset },
        });
        return {
          items: responseData.conversations || [],
          total: responseData.total || 0,
        };
      }
    },
    {
      initialLimit: 20,
      enableInfiniteScroll: false,
    }
  );

  const isLoading = conversationsLoading;
  const isFetching = conversationsFetching;

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
      if (e.key === 'Enter' && searchQuery.trim()) {
        setActiveSearchQuery(searchQuery);
        resetConversations(); // Reset pagination when searching
      }
    },
    [searchQuery, resetConversations]
  );

  // Clear search
  const handleClearSearch = () => {
    setSearchQuery('');
    setActiveSearchQuery('');
    resetConversations(); // Reset pagination when clearing search
    // Focus back to search input
    setTimeout(() => {
      searchInputRef.current?.focus();
    }, 0);
  };

  // Handle search button click
  const handleSearch = () => {
    setActiveSearchQuery(searchQuery);
    resetConversations(); // Reset pagination when searching
  };

  // Mutations (use generated hooks directly; avoid creating hooks inside callbacks)
  const deleteConversationMutation = useDeleteV1AiConversationsId(
    {
      mutation: {
        onSuccess: () => {
          resetConversations();
        },
      },
    },
    queryClient
  );

  const updateConversationMutation = usePutV1AiConversationsId(
    {
      mutation: {
        onSuccess: () => {
          resetConversations();
          setEditModalOpen(false);
          setEditingConversation(null);
          setEditTitle('');
        },
      },
    },
    queryClient
  );

  const bookmarkMessageMutation = usePutV1AiConversationsBookmark(
    {
      mutation: {
        onSuccess: () => {
          // Invalidate the conversation query to refresh the messages
          if (selectedConversation?.id) {
            queryClient.invalidateQueries({
              queryKey: getGetV1AiConversationsIdQueryKey(
                selectedConversation.id
              ),
            });
          }
        },
      },
    },
    queryClient
  );

  const totalCount = conversationsPagination.totalItems;

  // Hook to fetch conversation with messages
  const { data: conversationWithMessages } = useGetV1AiConversationsId(
    selectedConversation?.id || '',
    {
      query: {
        enabled: !!selectedConversation?.id && detailModalOpen,
      },
    }
  );

  const handleViewConversation = (conversation: Conversation) => {
    setSelectedConversation(conversation);
    setConversationMessages([]);
    setDetailModalOpen(true);
  };

  // Update messages when conversation data is loaded
  useEffect(() => {
    if (conversationWithMessages) {
      setConversationMessages(conversationWithMessages.messages || []);
    }
  }, [conversationWithMessages]);

  const handleEditConversation = (conversation: Conversation) => {
    setEditingConversation(conversation);
    setEditTitle(conversation.title || '');
    setEditModalOpen(true);
  };

  const handleDeleteConversation = (conversationId: string) => {
    setConversationToDelete(conversationId);
    setDeleteModalOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (!conversationToDelete) return;

    try {
      await deleteConversationMutation.mutateAsync({
        id: conversationToDelete,
      });
      setDeleteModalOpen(false);
      setConversationToDelete(null);
    } catch {}
  };

  const handleUpdateConversation = async () => {
    if (!editingConversation) return;

    try {
      await updateConversationMutation.mutateAsync({
        id: editingConversation.id,
        data: { title: editTitle },
      });
    } catch {}
  };

  const handleBookmarkToggle = async (messageId: string) => {
    if (!selectedConversation?.id) return;

    try {
      await bookmarkMessageMutation.mutateAsync({
        data: {
          conversation_id: selectedConversation.id,
          message_id: messageId,
        },
      });
    } catch {}
  };

  return (
    <Container size='xl' py='xl'>
      <Stack gap='xl'>
        <Group justify='space-between' align='center'>
          <div>
            <Title order={1}>Saved AI Conversations</Title>
            <Text c='dimmed' mt='xs'>
              View and manage your saved AI conversations
            </Text>
          </div>
          <Badge variant='light' color='blue' size='lg'>
            {totalCount} conversations
          </Badge>
        </Group>

        <Card padding='lg' radius='md' withBorder>
          <Group gap='md' mb='lg'>
            <TextInput
              ref={searchInputRef}
              placeholder='Type to prepare search query...'
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
                disabled={!searchQuery.trim() || isLoading || isFetching}
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
              Loading conversations...
            </Text>
          ) : conversations.length === 0 ? (
            <Text ta='center' py='xl' c='dimmed'>
              {activeSearchQuery
                ? 'No conversations found matching your search.'
                : 'No saved conversations yet.'}
            </Text>
          ) : (
            <>
              <Stack gap='md'>
                {conversations.map(conversation => (
                  <ConversationCard
                    key={conversation.id}
                    conversation={conversation}
                    onEdit={handleEditConversation}
                    onDelete={handleDeleteConversation}
                    onView={handleViewConversation}
                  />
                ))}
              </Stack>

              <Divider my='md' />

              <PaginationControls
                pagination={conversationsPagination}
                onPageChange={goToConversationsPage}
                onNext={goToNextConversationsPage}
                onPrevious={goToPreviousConversationsPage}
                isLoading={isLoading || isFetching}
                variant='desktop'
              />
            </>
          )}
        </Card>
      </Stack>

      {/* Conversation Detail Modal */}
      <ConversationDetailModal
        conversation={selectedConversation}
        opened={detailModalOpen}
        onClose={() => {
          setDetailModalOpen(false);
          setSelectedConversation(null);
          setConversationMessages([]);
        }}
        messages={conversationMessages}
        onBookmarkToggle={handleBookmarkToggle}
        isBookmarking={bookmarkMessageMutation.isPending}
      />

      {/* Edit Title Modal */}
      <Modal
        opened={editModalOpen}
        onClose={() => {
          setEditModalOpen(false);
          setEditingConversation(null);
          setEditTitle('');
        }}
        title='Edit Conversation Title'
      >
        <Stack gap='md'>
          <TextInput
            label='Title'
            value={editTitle}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setEditTitle(e.target.value)
            }
            placeholder='Enter conversation title...'
          />
          <Group justify='flex-end' gap='sm'>
            <Button variant='light' onClick={() => setEditModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleUpdateConversation}
              loading={updateConversationMutation.isPending}
            >
              Save
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        opened={deleteModalOpen}
        onClose={() => {
          setDeleteModalOpen(false);
          setConversationToDelete(null);
        }}
        title='Delete Conversation'
        size='sm'
      >
        <Stack gap='md'>
          <Text>
            Are you sure you want to delete this conversation? This action
            cannot be undone.
          </Text>
          <Group justify='flex-end' gap='sm'>
            <Button
              variant='light'
              onClick={() => {
                setDeleteModalOpen(false);
                setConversationToDelete(null);
              }}
            >
              Cancel
            </Button>
            <Button
              color='red'
              onClick={handleConfirmDelete}
              loading={deleteConversationMutation.isPending}
            >
              Delete
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
};

export default SavedConversationsPage;
