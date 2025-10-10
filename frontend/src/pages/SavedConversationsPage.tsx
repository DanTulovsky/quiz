import React, { useState } from 'react';
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
  Box,
  Divider,
  ActionIcon,
  Menu,
  Modal,
} from '@mantine/core';
import {
  Search,
  Edit,
  Trash2,
  MessageCircle,
  Calendar,
  Filter,
  SortDesc,
} from 'lucide-react';
import { useAuth } from '../hooks/useAuth';
import {
  useGetV1AiConversations,
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
          {format(new Date(conversation.created_at), 'MMM d, yyyy')}
        </Badge>
        <Badge variant='light' color='green'>
          {conversation.message_count || 0} messages
        </Badge>
      </Group>

      {conversation.preview_message && (
        <Text size='sm' c='dimmed' lineClamp={isExpanded ? undefined : 2}>
          {conversation.preview_message}
        </Text>
      )}

      {conversation.preview_message &&
        conversation.preview_message.length > 100 && (
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
      <Box style={{ flex: 1, overflow: 'auto', maxHeight: '60vh' }}>
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
      </Box>
    </Modal>
  );
};

export const SavedConversationsPage: React.FC = () => {
  const {} = useAuth();
  const [searchQuery, setSearchQuery] = useState('');
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

  // Fetch conversations
  const { data: conversationsData, isLoading } = useGetV1AiConversations({
    limit: 50,
    offset: 0,
  });

  // Fetch search results - only when there's a search query
  const { data: searchData, isLoading: isSearching } = useGetV1AiSearch(
    {
      q: searchQuery,
      limit: 50,
      offset: 0,
    },
    {
      enabled: searchQuery.length > 0,
    }
  );

  // Mutations
  const deleteConversationMutation = useMutation({
    mutationFn: ({ conversationId }: { conversationId: string }) =>
      useDeleteV1AiConversationsId().mutateAsync({ id: conversationId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['aiConversations'] });
      queryClient.invalidateQueries({ queryKey: ['aiSearch'] });
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
      queryClient.invalidateQueries({ queryKey: ['aiSearch'] });
      setEditModalOpen(false);
      setEditingConversation(null);
      setEditTitle('');
    },
  });

  const conversations = searchQuery
    ? searchData?.conversations || []
    : conversationsData?.conversations || [];
  const totalCount = searchQuery
    ? searchData?.total_count || 0
    : conversationsData?.total_count || 0;

  const handleViewConversation = async (conversation: Conversation) => {
    // For now, we'll just show the conversation in a modal with existing data
    // In a real implementation, you'd fetch the full conversation with messages
    setSelectedConversation(conversation);
    setConversationMessages([]); // Would fetch messages here
    setDetailModalOpen(true);
  };

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
              placeholder='Search conversations...'
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              leftSection={<Search size={16} />}
              style={{ flex: 1 }}
              disabled={isLoading || isSearching}
            />
            <Group gap='xs'>
              <Button variant='outline' leftSection={<Filter size={16} />}>
                Filter
              </Button>
              <Button variant='outline' leftSection={<SortDesc size={16} />}>
                Sort
              </Button>
            </Group>
          </Group>

          {isLoading ? (
            <Text ta='center' py='xl' c='dimmed'>
              Loading conversations...
            </Text>
          ) : conversations.length === 0 ? (
            <Text ta='center' py='xl' c='dimmed'>
              {searchQuery
                ? 'No conversations found matching your search.'
                : 'No saved conversations yet.'}
            </Text>
          ) : (
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
            onChange={e => setEditTitle(e.target.value)}
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
