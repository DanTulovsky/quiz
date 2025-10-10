import React, {
  useState,
  useRef,
  useMemo,
  useCallback,
  useEffect,
} from 'react';
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
  Divider,
  ActionIcon,
  Menu,
  Modal,
} from '@mantine/core';
import { Search, Edit, Trash2, MessageCircle, Calendar } from 'lucide-react';
import { useAuth } from '../hooks/useAuth';
import {
  useGetV1AiConversations,
  useGetV1AiConversationsId,
  useGetV1AiSearch,
  useDeleteV1AiConversationsId,
  usePutV1AiConversationsId,
  Conversation,
  ChatMessage,
} from '../api/api';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import logger from '../utils/logger';
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
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <Card shadow='sm' padding='lg' radius='md' withBorder>
      <Group justify='space-between' mb='xs'>
        <Title
          order={4}
          style={{ cursor: 'pointer' }}
          onClick={() => onView(conversation)}
        >
          {conversation.title || 'Untitled Conversation'}
        </Title>
        <Menu shadow='md' width={120}>
          <Menu.Target>
            <ActionIcon variant='subtle' color='gray'>
              <Edit size={16} />
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Item
              leftSection={<MessageCircle size={16} />}
              onClick={() => onView(conversation)}
            >
              View
            </Menu.Item>
            <Menu.Item
              leftSection={<Edit size={16} />}
              onClick={() => onEdit(conversation)}
            >
              Edit Title
            </Menu.Item>
            <Menu.Item
              leftSection={<Trash2 size={16} />}
              color='red'
              onClick={() => onDelete(conversation.id)}
            >
              Delete
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>

      <Group mb='sm'>
        <Badge
          variant='light'
          color='blue'
          leftSection={<Calendar size={12} />}
        >
          {format(new Date(conversation.created_at), 'MMM d, h:mm a')}
        </Badge>
        <Badge variant='light' color='green'>
          {conversation.messages?.length || 0} messages
        </Badge>
      </Group>

      {conversation.messages && conversation.messages.length > 0 && (
        <Text size='sm' c='dimmed' lineClamp={isExpanded ? undefined : 2}>
          {typeof conversation.messages[0]?.content === 'string'
            ? conversation.messages[0].content.substring(0, 200)
            : 'No preview available'}
        </Text>
      )}

      {conversation.messages &&
        conversation.messages.length > 0 &&
        typeof conversation.messages[0]?.content === 'string' &&
        conversation.messages[0].content.length > 100 && (
          <Button
            variant='subtle'
            size='xs'
            mt='xs'
            onClick={() => setIsExpanded(!isExpanded)}
          >
            {isExpanded ? 'Show Less' : 'Show More'}
          </Button>
        )}

      <Divider my='sm' />

      <Group justify='flex-end'>
        <Button variant='light' size='xs' onClick={() => onView(conversation)}>
          View Conversation
        </Button>
      </Group>
    </Card>
  );
};

interface ConversationDetailModalProps {
  conversation: Conversation | null;
  opened: boolean;
  onClose: () => void;
  messages: ChatMessage[];
}

const ConversationDetailModal: React.FC<ConversationDetailModalProps> = ({
  conversation,
  opened,
  onClose,
  messages,
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
          {messages.map((message, index) => (
            <Card key={message.id || index} padding='md' radius='sm' withBorder>
              <Group mb='xs'>
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
              <Text size='sm' style={{ whiteSpace: 'pre-wrap' }}>
                {typeof message.content === 'string'
                  ? message.content
                  : JSON.stringify(message.content, null, 2)}
              </Text>
            </Card>
          ))}
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

  const queryClient = useQueryClient();
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Fetch conversations or search results
  const { data: conversationsData, isLoading: conversationsLoading } =
    useGetV1AiConversations(
      {
        limit: 50,
        offset: 0,
      },
      {
        query: {
          enabled: !activeSearchQuery.trim(),
        },
      }
    );

  const { data: searchData, isLoading: searchLoading } = useGetV1AiSearch(
    {
      q: activeSearchQuery,
      limit: 50,
      offset: 0,
    },
    {
      query: {
        enabled: !!activeSearchQuery.trim(),
      },
    }
  );

  const isLoading = conversationsLoading || searchLoading;

  // Search is now only triggered manually via the search button

  // Get conversations or search results
  const filteredConversations = useMemo(() => {
    if (activeSearchQuery.trim()) {
      // When searching, use search results directly
      return searchData?.conversations || [];
    } else {
      return conversationsData?.conversations || [];
    }
  }, [
    conversationsData?.conversations,
    searchData?.conversations,
    activeSearchQuery,
  ]);

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
      }
    },
    [searchQuery]
  );

  // Clear search
  const handleClearSearch = () => {
    setSearchQuery('');
    setActiveSearchQuery('');
    // Focus back to search input
    setTimeout(() => {
      searchInputRef.current?.focus();
    }, 0);
  };

  // Mutations
  const deleteConversationMutation = useMutation({
    mutationFn: ({ conversationId }: { conversationId: string }) =>
      useDeleteV1AiConversationsId().mutateAsync({ id: conversationId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['aiConversations'] });
    },
  });

  const updateConversationMutation = useMutation({
    mutationFn: ({
      conversationId,
      data,
    }: {
      conversationId: string;
      data: { title: string };
    }) => usePutV1AiConversationsId().mutateAsync({ id: conversationId, data }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['aiConversations'] });
      setEditModalOpen(false);
      setEditingConversation(null);
      setEditTitle('');
    },
  });

  const totalCount = activeSearchQuery.trim()
    ? searchData?.total || filteredConversations.length
    : conversationsData?.total || filteredConversations.length;

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

  const handleDeleteConversation = async (conversationId: string) => {
    if (window.confirm('Are you sure you want to delete this conversation?')) {
      try {
        await deleteConversationMutation.mutateAsync({ conversationId });
      } catch (error) {
        logger.error('Failed to delete conversation:', error);
      }
    }
  };

  const handleUpdateConversation = async () => {
    if (!editingConversation) return;

    try {
      await updateConversationMutation.mutateAsync({
        conversationId: editingConversation.id,
        data: { title: editTitle },
      });
    } catch (error) {
      logger.error('Failed to update conversation:', error);
    }
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
              disabled={isLoading}
            />
            <Group gap='xs'>
              <Button
                variant='filled'
                leftSection={<Search size={16} />}
                onClick={() => {
                  // Trigger search immediately by setting active search query
                  setActiveSearchQuery(searchQuery);
                }}
                disabled={!searchQuery.trim() || isLoading}
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
          ) : filteredConversations.length === 0 ? (
            <Text ta='center' py='xl' c='dimmed'>
              {activeSearchQuery
                ? 'No conversations found matching your search.'
                : 'No saved conversations yet.'}
            </Text>
          ) : (
            <Stack gap='md'>
              {filteredConversations.map(conversation => (
                <ConversationCard
                  key={conversation.id}
                  conversation={conversation}
                  onEdit={handleEditConversation}
                  onDelete={handleDeleteConversation}
                  onView={handleViewConversation}
                />
              ))}
            </Stack>
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
    </Container>
  );
};

export default SavedConversationsPage;
