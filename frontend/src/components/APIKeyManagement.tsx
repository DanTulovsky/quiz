import React, { useState, useEffect } from 'react';
import {
  Card,
  Stack,
  Group,
  Title,
  Button,
  Table,
  Modal,
  TextInput,
  Select,
  Text,
  Badge,
  ActionIcon,
  Code,
  Alert,
  CopyButton,
  Tooltip,
  Loader,
} from '@mantine/core';
import {
  IconKey,
  IconPlus,
  IconTrash,
  IconCopy,
  IconCheck,
  IconAlertCircle,
} from '@tabler/icons-react';
import { showNotification } from '@mantine/notifications';

interface APIKey {
  id: number;
  key_name: string;
  key_prefix: string;
  permission_level: 'readonly' | 'full';
  last_used_at: string | null;
  created_at: string;
  updated_at: string;
}

interface CreateAPIKeyResponse {
  id: number;
  key_name: string;
  key: string;
  key_prefix: string;
  permission_level: string;
  created_at: string;
  message: string;
}

export const APIKeyManagement: React.FC = () => {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyPermission, setNewKeyPermission] = useState<'readonly' | 'full'>(
    'full'
  );
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [showKeyModal, setShowKeyModal] = useState(false);
  const [creating, setCreating] = useState(false);

  const fetchAPIKeys = async () => {
    try {
      setLoading(true);
      const response = await fetch('/v1/api-keys', {
        credentials: 'include',
      });
      if (!response.ok) throw new Error('Failed to fetch API keys');
      const data = await response.json();
      setApiKeys(data.api_keys || []);
    } catch (error) {
      console.error('Failed to fetch API keys:', error);
      showNotification({
        title: 'Error',
        message: 'Failed to load API keys',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAPIKeys();
  }, []);

  const handleCreateKey = async () => {
    if (!newKeyName.trim()) {
      showNotification({
        title: 'Error',
        message: 'Please enter a name for the API key',
        color: 'red',
      });
      return;
    }

    try {
      setCreating(true);
      const response = await fetch('/v1/api-keys', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({
          key_name: newKeyName,
          permission_level: newKeyPermission,
        }),
      });

      if (!response.ok) throw new Error('Failed to create API key');

      const data: CreateAPIKeyResponse = await response.json();
      setCreatedKey(data.key);
      setShowKeyModal(true);
      setCreateModalOpen(false);
      setNewKeyName('');
      setNewKeyPermission('full');

      // Refresh the list
      await fetchAPIKeys();

      showNotification({
        title: 'Success',
        message: 'API key created successfully',
        color: 'green',
      });
    } catch (error) {
      console.error('Failed to create API key:', error);
      showNotification({
        title: 'Error',
        message: 'Failed to create API key',
        color: 'red',
      });
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteKey = async (id: number, name: string) => {
    if (
      !window.confirm(
        `Are you sure you want to delete the API key "${name}"? This action cannot be undone.`
      )
    ) {
      return;
    }

    try {
      const response = await fetch(`/v1/api-keys/${id}`, {
        method: 'DELETE',
        credentials: 'include',
      });

      if (!response.ok) throw new Error('Failed to delete API key');

      await fetchAPIKeys();

      showNotification({
        title: 'Success',
        message: 'API key deleted successfully',
        color: 'green',
      });
    } catch (error) {
      console.error('Failed to delete API key:', error);
      showNotification({
        title: 'Error',
        message: 'Failed to delete API key',
        color: 'red',
      });
    }
  };

  const formatDate = (dateString: string | null) => {
    if (!dateString) return 'Never';
    return new Date(dateString).toLocaleString();
  };

  return (
    <Card shadow='sm' padding='lg' radius='md' withBorder>
      <Stack gap='lg'>
        <Group justify='space-between'>
          <Group>
            <IconKey size={20} />
            <Title order={2}>API Keys</Title>
          </Group>
          <Button
            leftSection={<IconPlus size={16} />}
            onClick={() => setCreateModalOpen(true)}
          >
            Create New Key
          </Button>
        </Group>

        <Text size='sm' c='dimmed'>
          API keys allow you to access the Quiz App API programmatically. Keep
          your keys secure and never share them publicly.
        </Text>

        {loading ? (
          <Group justify='center' p='xl'>
            <Loader />
          </Group>
        ) : apiKeys.length === 0 ? (
          <Alert
            icon={<IconAlertCircle size={16} />}
            title='No API keys yet'
            color='blue'
          >
            You haven't created any API keys. Create one to start using the API
            programmatically.
          </Alert>
        ) : (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Name</Table.Th>
                <Table.Th>Key Prefix</Table.Th>
                <Table.Th>Permission</Table.Th>
                <Table.Th>Last Used</Table.Th>
                <Table.Th>Created</Table.Th>
                <Table.Th>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {apiKeys.map(key => (
                <Table.Tr key={key.id}>
                  <Table.Td>{key.key_name}</Table.Td>
                  <Table.Td>
                    <Code>{key.key_prefix}...</Code>
                  </Table.Td>
                  <Table.Td>
                    <Badge
                      color={key.permission_level === 'full' ? 'green' : 'blue'}
                    >
                      {key.permission_level}
                    </Badge>
                  </Table.Td>
                  <Table.Td>{formatDate(key.last_used_at)}</Table.Td>
                  <Table.Td>{formatDate(key.created_at)}</Table.Td>
                  <Table.Td>
                    <Tooltip label='Delete key'>
                      <ActionIcon
                        color='red'
                        variant='subtle'
                        onClick={() => handleDeleteKey(key.id, key.key_name)}
                      >
                        <IconTrash size={16} />
                      </ActionIcon>
                    </Tooltip>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Stack>

      {/* Create API Key Modal */}
      <Modal
        opened={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        title='Create New API Key'
        size='md'
      >
        <Stack gap='md'>
          <TextInput
            label='Key Name'
            placeholder='e.g., Production App Key'
            value={newKeyName}
            onChange={e => setNewKeyName(e.currentTarget.value)}
            required
          />

          <Select
            label='Permission Level'
            value={newKeyPermission}
            onChange={value =>
              setNewKeyPermission(value as 'readonly' | 'full')
            }
            data={[
              { value: 'readonly', label: 'Read Only - GET requests only' },
              { value: 'full', label: 'Full Access - All operations' },
            ]}
            required
          />

          <Group justify='flex-end' mt='md'>
            <Button variant='subtle' onClick={() => setCreateModalOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreateKey} loading={creating}>
              Create Key
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Show Created Key Modal */}
      <Modal
        opened={showKeyModal}
        onClose={() => {
          setShowKeyModal(false);
          setCreatedKey(null);
        }}
        title='API Key Created'
        size='lg'
        closeOnClickOutside={false}
        closeOnEscape={false}
      >
        <Stack gap='md'>
          <Alert
            icon={<IconAlertCircle size={16} />}
            title='Important!'
            color='yellow'
          >
            This is the only time you'll see this key. Copy it now and store it
            securely. You won't be able to see it again!
          </Alert>

          <Stack gap='xs'>
            <Text size='sm' fw={500}>
              Your API Key:
            </Text>
            <Group gap='xs'>
              <Code
                style={{
                  flex: 1,
                  padding: '8px',
                  fontSize: '12px',
                  wordBreak: 'break-all',
                }}
              >
                {createdKey}
              </Code>
              <CopyButton value={createdKey || ''}>
                {({ copied, copy }) => (
                  <Tooltip label={copied ? 'Copied!' : 'Copy to clipboard'}>
                    <ActionIcon
                      color={copied ? 'teal' : 'gray'}
                      variant='subtle'
                      onClick={copy}
                    >
                      {copied ? (
                        <IconCheck size={16} />
                      ) : (
                        <IconCopy size={16} />
                      )}
                    </ActionIcon>
                  </Tooltip>
                )}
              </CopyButton>
            </Group>
          </Stack>

          <Text size='sm' c='dimmed'>
            Use this key in your API requests by adding the Authorization
            header:
          </Text>
          <Code block>Authorization: Bearer {createdKey}</Code>

          <Group justify='flex-end' mt='md'>
            <Button
              onClick={() => {
                setShowKeyModal(false);
                setCreatedKey(null);
              }}
            >
              I've Saved My Key
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Card>
  );
};
