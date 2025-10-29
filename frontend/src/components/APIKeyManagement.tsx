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
import ConfirmationModal from './ConfirmationModal';

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
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [keyToDelete, setKeyToDelete] = useState<APIKey | null>(null);
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [keyToTest, setKeyToTest] = useState<APIKey | null>(null);
  const [testFullKey, setTestFullKey] = useState('');
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<null | {
    ok: boolean;
    status: number;
    body?: unknown;
  }>(null);

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

  const handleDeleteKey = (key: APIKey) => {
    setKeyToDelete(key);
    setDeleteModalOpen(true);
  };

  const confirmDeleteKey = async () => {
    if (!keyToDelete) return;
    try {
      const response = await fetch(`/v1/api-keys/${keyToDelete.id}`, {
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
    } finally {
      setDeleteModalOpen(false);
      setKeyToDelete(null);
    }
  };

  const prefillFullKeyIfAvailable = (k: APIKey) => {
    if (!createdKey) return '';
    // If the created key matches this key's prefix, prefill
    const prefix = createdKey.substring(0, Math.min(createdKey.length, 12));
    if (k.key_prefix && prefix === k.key_prefix) return createdKey;
    return '';
  };

  const runTest = async (mode: 'read' | 'write') => {
    setTesting(true);
    setTestResult(null);
    try {
      const origin = window.location.origin;
      const endpoint =
        mode === 'read' ? '/v1/api-keys/test-read' : '/v1/api-keys/test-write';
      const url = `${origin}${endpoint}`;
      const headers: Record<string, string> = {
        Authorization: `Bearer ${testFullKey.trim()}`,
      };
      if (mode === 'write') headers['Content-Type'] = 'application/json';
      const resp = await fetch(url, {
        method: mode === 'read' ? 'GET' : 'POST',
        headers,
        credentials: 'omit',
        body: mode === 'write' ? JSON.stringify({}) : undefined,
      });
      let body: unknown = undefined;
      try {
        body = await resp.json();
      } catch {
        body = await resp.text();
      }
      setTestResult({ ok: resp.ok, status: resp.status, body });
    } catch (e) {
      setTestResult({ ok: false, status: 0, body: String(e) });
    } finally {
      setTesting(false);
    }
  };

  const curlSnippet = (mode: 'read' | 'write') => {
    const origin = window.location.origin;
    const ep =
      mode === 'read' ? '/v1/api-keys/test-read' : '/v1/api-keys/test-write';
    const key = testFullKey.trim() || '<YOUR_API_KEY>';
    if (mode === 'read') {
      return `curl -sS -X GET "${origin}${ep}" -H "Authorization: Bearer ${key}"`;
    }
    return `curl -sS -X POST "${origin}${ep}" -H "Authorization: Bearer ${key}" -H "Content-Type: application/json" -d '{}'`;
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
                    <Group gap='xs'>
                      <Tooltip label='Test this key'>
                        <Button
                          size='xs'
                          variant='light'
                          onClick={() => {
                            setKeyToTest(key);
                            const prefill = prefillFullKeyIfAvailable(key);
                            setTestFullKey(prefill);
                            setTestResult(null);
                            setTestModalOpen(true);
                          }}
                        >
                          Test
                        </Button>
                      </Tooltip>
                      <Tooltip label='Delete key'>
                        <ActionIcon
                          color='red'
                          variant='subtle'
                          onClick={() => handleDeleteKey(key)}
                        >
                          <IconTrash size={16} />
                        </ActionIcon>
                      </Tooltip>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Stack>

      {/* Delete API Key Confirmation Modal */}
      <ConfirmationModal
        isOpen={deleteModalOpen}
        onClose={() => {
          setDeleteModalOpen(false);
          setKeyToDelete(null);
        }}
        onConfirm={confirmDeleteKey}
        title='Delete API key'
        message={`Are you sure you want to delete the API key "${keyToDelete?.key_name ?? ''}"? This action cannot be undone.`}
        confirmText='Delete'
        cancelText='Cancel'
      />

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

      {/* Test API Key Modal */}
      <Modal
        opened={testModalOpen}
        onClose={() => setTestModalOpen(false)}
        title={`Test API Key${keyToTest ? `: ${keyToTest.key_name}` : ''}`}
        size='lg'
      >
        <Stack gap='md'>
          <Text size='sm'>
            Paste the full API key to test. Requests ignore cookies and use
            Authorization: Bearer.
          </Text>
          <TextInput
            label='Full API Key'
            placeholder='qapp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
            value={testFullKey}
            onChange={e => setTestFullKey(e.currentTarget.value)}
          />
          <Group>
            <Button
              onClick={() => runTest('read')}
              loading={testing}
              variant='default'
            >
              Test Read (GET)
            </Button>
            <Button onClick={() => runTest('write')} loading={testing}>
              Test Write (POST)
            </Button>
          </Group>
          {testResult && (
            <Alert
              color={testResult.ok ? 'green' : 'red'}
              title={`HTTP ${testResult.status}`}
            >
              <Code block style={{ whiteSpace: 'pre-wrap' }}>
                {typeof testResult.body === 'string'
                  ? testResult.body
                  : JSON.stringify(testResult.body, null, 2)}
              </Code>
            </Alert>
          )}
          <Stack gap='xs'>
            <Text size='sm' fw={500}>
              curl (read):
            </Text>
            <Group gap='xs'>
              <Code style={{ flex: 1, wordBreak: 'break-all', padding: '8px' }}>
                {curlSnippet('read')}
              </Code>
              <CopyButton value={curlSnippet('read')}>
                {({ copied, copy }) => (
                  <ActionIcon
                    color={copied ? 'teal' : 'gray'}
                    variant='subtle'
                    onClick={copy}
                  >
                    {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                  </ActionIcon>
                )}
              </CopyButton>
            </Group>
            <Text size='sm' fw={500} mt='sm'>
              curl (write):
            </Text>
            <Group gap='xs'>
              <Code style={{ flex: 1, wordBreak: 'break-all', padding: '8px' }}>
                {curlSnippet('write')}
              </Code>
              <CopyButton value={curlSnippet('write')}>
                {({ copied, copy }) => (
                  <ActionIcon
                    color={copied ? 'teal' : 'gray'}
                    variant='subtle'
                    onClick={copy}
                  >
                    {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                  </ActionIcon>
                )}
              </CopyButton>
            </Group>
          </Stack>
        </Stack>
      </Modal>
    </Card>
  );
};
