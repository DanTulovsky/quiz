import React, { useState } from 'react';
import {
  Container,
  Title,
  Text,
  Card,
  Grid,
  Group,
  Badge,
  Stack,
  Button,
  Modal,
  TextInput,
  PasswordInput,
  Select,
  Switch,
  Alert,
  Loader,
  Center,
  Checkbox,
  Table,
  Pagination,
  ActionIcon,
  Menu,
  Paper,
  SimpleGrid,
  ThemeIcon,
} from '@mantine/core';
import {
  IconTrash,
  IconEdit,
  IconKey,
  IconRefresh,
  IconPlus,
  IconAlertTriangle,
  IconCheck,
  IconX,
  IconSearch,
  IconFilter,
  IconDotsVertical,
  IconUsers,
  IconBrain,
  IconActivity,
  IconPlayerPause,
  IconPlayerPlay,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import {
  useCreateUser,
  useUpdateUser,
  useDeleteUser,
  useResetUserPassword,
  useClearUserDataForUser,
  useRoles,
  useUsersPaginated,
  usePauseUser,
  useResumeUser,
} from '../../api/admin';
import { User, Role } from '../../api/api';
import {
  useGetV1SettingsLanguages,
  useGetV1SettingsLevels,
} from '../../api/api';

// Add this type for the levels API response
interface LevelsApiResponse {
  levels: string[];
  level_descriptions: Record<string, string>;
}
import { useAuth } from '../../hooks/useAuth';
import { Navigate } from 'react-router-dom';

const UserManagementPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isPasswordModalOpen, setIsPasswordModalOpen] = useState(false);
  const [isClearDataModalOpen, setIsClearDataModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [userToDelete, setUserToDelete] = useState<{
    id: number;
    username: string;
  } | null>(null);

  // Pagination and filtering state
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(20);
  const [filters, setFilters] = useState({
    search: '',
    language: '',
    level: '',
    aiProvider: '',
    aiModel: '',
    aiEnabled: '',
    active: '',
  });

  // API hooks with pagination
  const {
    data: usersData,
    isLoading,
    error,
  } = useUsersPaginated({
    page: currentPage,
    pageSize,
    search: filters.search || undefined,
    language: filters.language || undefined,
    level: filters.level || undefined,
    aiProvider: filters.aiProvider || undefined,
    aiModel: filters.aiModel || undefined,
    aiEnabled: filters.aiEnabled || undefined,
    active: filters.active || undefined,
  });

  // API hooks for languages and levels
  const { data: languagesData } = useGetV1SettingsLanguages();
  const { data: levelsData } = useGetV1SettingsLevels<LevelsApiResponse>({
    language: filters.language || undefined,
  });

  const { data: allRoles } = useRoles();

  // Mutations
  const createUserMutation = useCreateUser();
  const updateUserMutation = useUpdateUser();
  const deleteUserMutation = useDeleteUser();
  const resetPasswordMutation = useResetUserPassword();
  const clearUserDataMutation = useClearUserDataForUser();
  const pauseUserMutation = usePauseUser();
  const resumeUserMutation = useResumeUser();

  // Form states
  const [createForm, setCreateForm] = useState({
    username: '',
    email: '',
    timezone: 'UTC',
    password: '',
    preferred_language: 'italian',
    current_level: 'A1',
    selectedRoles: [] as string[],
  });

  const [editForm, setEditForm] = useState({
    username: '',
    email: '',
    timezone: '',
    preferred_language: '',
    current_level: '',
    ai_enabled: false,
    ai_provider: '',
    ai_model: '',
    api_key: '',
    selectedRoles: [] as string[],
  });

  const [passwordForm, setPasswordForm] = useState({
    new_password: '',
  });

  // Check if user is admin
  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }

  const isAdmin = user.roles?.some(role => role.name === 'admin') || false;
  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  const handleCreateUser = async () => {
    try {
      const result = await createUserMutation.mutateAsync(createForm);

      // Assign roles to the newly created user
      if (result.user?.id && allRoles && createForm.selectedRoles.length > 0) {
        for (const roleName of createForm.selectedRoles) {
          const role = allRoles.find((r: Role) => r.name === roleName);
          if (role) {
            await fetch(`/v1/admin/backend/userz/${result.user.id}/roles`, {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
              },
              body: JSON.stringify({ role_id: role.id }),
            });
          }
        }
      }

      notifications.show({
        title: 'Success',
        message: 'User created successfully',
        color: 'green',
        icon: <IconCheck size={16} />,
      });
      setIsCreateModalOpen(false);
      setCreateForm({
        username: '',
        email: '',
        timezone: 'UTC',
        password: '',
        preferred_language: 'italian',
        current_level: 'A1',
        selectedRoles: [],
      });
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to create user',
        color: 'red',
        icon: <IconX size={16} />,
      });
    }
  };

  const handleUpdateUser = async () => {
    if (!selectedUser) return;
    try {
      // Update user data including roles
      await updateUserMutation.mutateAsync({
        userId: selectedUser.id || 0,
        userData: editForm,
      });

      notifications.show({
        title: 'Success',
        message: 'User updated successfully',
        color: 'green',
        icon: <IconCheck size={16} />,
      });
      setIsEditModalOpen(false);
      setSelectedUser(null);
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to update user',
        color: 'red',
        icon: <IconX size={16} />,
      });
    }
  };

  const handleDeleteUser = async (userId: number) => {
    try {
      await deleteUserMutation.mutateAsync(userId);
      notifications.show({
        title: 'Success',
        message: 'User deleted successfully',
        color: 'green',
        icon: <IconCheck size={16} />,
      });
      setIsDeleteModalOpen(false);
      setUserToDelete(null);
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to delete user',
        color: 'red',
        icon: <IconX size={16} />,
      });
    }
  };

  const handleResetPassword = async () => {
    if (!selectedUser) return;
    try {
      await resetPasswordMutation.mutateAsync({
        userId: selectedUser.id || 0,
        newPassword: passwordForm.new_password,
      });
      notifications.show({
        title: 'Success',
        message: 'Password reset successfully',
        color: 'green',
        icon: <IconCheck size={16} />,
      });
      setIsPasswordModalOpen(false);
      setSelectedUser(null);
      setPasswordForm({ new_password: '' });
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to reset password',
        color: 'red',
        icon: <IconX size={16} />,
      });
    }
  };

  const handleClearUserData = async () => {
    if (!selectedUser) return;
    try {
      await clearUserDataMutation.mutateAsync(selectedUser.id || 0);
      notifications.show({
        title: 'Success',
        message: 'User data cleared successfully',
        color: 'green',
        icon: <IconCheck size={16} />,
      });
      setIsClearDataModalOpen(false);
      setSelectedUser(null);
    } catch {
      notifications.show({
        title: 'Error',
        message: 'Failed to clear user data',
        color: 'red',
        icon: <IconX size={16} />,
      });
    }
  };

  const totalUsers = usersData?.pagination?.total || 0;
  const totalPages = usersData?.pagination?.total_pages || 1;

  return (
    <Container size='xl' py='xl'>
      <Stack gap='xl'>
        {/* Header */}
        <Group justify='space-between' align='center'>
          <div>
            <Title order={1}>User Management</Title>
            <Text color='dimmed' size='lg'>
              Create, edit, and manage user accounts
            </Text>
          </div>
          <Button
            leftSection={<IconPlus size={16} />}
            onClick={() => setIsCreateModalOpen(true)}
          >
            Create User
          </Button>
        </Group>

        {/* Statistics */}
        <SimpleGrid cols={{ base: 1, sm: 3 }}>
          <Paper p='md' withBorder>
            <Group>
              <ThemeIcon size='lg' color='blue'>
                <IconUsers size={20} />
              </ThemeIcon>
              <div>
                <Text size='xs' color='dimmed' tt='uppercase' fw={700}>
                  Total Users
                </Text>
                <Text size='xl' fw={700}>
                  {totalUsers}
                </Text>
              </div>
            </Group>
          </Paper>

          <Paper p='md' withBorder>
            <Group>
              <ThemeIcon size='lg' color='green'>
                <IconBrain size={20} />
              </ThemeIcon>
              <div>
                <Text size='xs' color='dimmed' tt='uppercase' fw={700}>
                  AI Enabled
                </Text>
                <Text size='xl' fw={700}>
                  {usersData?.users?.filter(
                    (u: { ai_enabled?: boolean }) => u?.ai_enabled
                  ).length || 0}
                </Text>
              </div>
            </Group>
          </Paper>

          <Paper p='md' withBorder>
            <Group>
              <ThemeIcon size='lg' color='orange'>
                <IconActivity size={20} />
              </ThemeIcon>
              <div>
                <Text size='xs' color='dimmed' tt='uppercase' fw={700}>
                  Active Users
                </Text>
                <Text size='xl' fw={700}>
                  {usersData?.users?.filter(
                    (u: { last_active?: string }) =>
                      u?.last_active &&
                      new Date(u.last_active) >
                        new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
                  ).length || 0}
                </Text>
              </div>
            </Group>
          </Paper>
        </SimpleGrid>

        {/* Filters */}
        <Card shadow='sm' padding='lg' radius='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Title order={3}>Filters</Title>
            <Button
              variant='light'
              size='sm'
              leftSection={<IconFilter size={16} />}
              onClick={() =>
                setFilters({
                  search: '',
                  language: '',
                  level: '',
                  aiProvider: '',
                  aiModel: '',
                  aiEnabled: '',
                  active: '',
                })
              }
            >
              Clear Filters
            </Button>
          </Group>

          <Grid>
            <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
              <TextInput
                label='Search'
                placeholder='Username or email...'
                value={filters.search}
                onChange={e =>
                  setFilters({ ...filters, search: e.target.value })
                }
                leftSection={<IconSearch size={16} />}
              />
            </Grid.Col>
            <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
              <Select
                label='Language'
                placeholder='All languages'
                value={filters.language}
                onChange={value =>
                  setFilters({ ...filters, language: value || '' })
                }
                data={[
                  { value: '', label: 'All Languages' },
                  ...(languagesData?.map(lang => ({
                    value: lang,
                    label: lang.charAt(0).toUpperCase() + lang.slice(1),
                  })) || []),
                ]}
                clearable
              />
            </Grid.Col>
            <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
              <Select
                label='Level'
                placeholder='All levels'
                value={filters.level}
                onChange={value =>
                  setFilters({ ...filters, level: value || '' })
                }
                data={[
                  { value: '', label: 'All Levels' },
                  ...(levelsData?.levels?.map(level => ({
                    value: level,
                    label:
                      `${level} - ${levelsData.level_descriptions?.[level] || ''}`.trim(),
                  })) || []),
                ]}
                clearable
              />
            </Grid.Col>
            <Grid.Col span={{ base: 12, sm: 6, md: 3 }}>
              <Select
                label='AI Status'
                placeholder='All users'
                value={filters.aiEnabled}
                onChange={value =>
                  setFilters({ ...filters, aiEnabled: value || '' })
                }
                data={[
                  { value: 'true', label: 'AI Enabled' },
                  { value: 'false', label: 'AI Disabled' },
                ]}
                clearable
              />
            </Grid.Col>
          </Grid>
        </Card>

        {/* Users Table */}
        <Card shadow='sm' padding='lg' radius='md' withBorder>
          <Group justify='space-between' mb='md'>
            <Title order={3}>Users</Title>
            <Text size='sm' color='dimmed'>
              Showing {usersData?.pagination?.total || 0} users
            </Text>
          </Group>

          {isLoading ? (
            <Center py='xl'>
              <Loader size='lg' />
            </Center>
          ) : error ? (
            <Alert
              icon={<IconAlertTriangle size={16} />}
              title='Error'
              color='red'
            >
              Failed to load users: {error?.message || 'Unknown error'}
            </Alert>
          ) : usersData?.users && usersData.users.length > 0 ? (
            <>
              <Table striped highlightOnHover>
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>User</Table.Th>
                    <Table.Th>Level</Table.Th>
                    <Table.Th>Language</Table.Th>
                    <Table.Th>AI Status</Table.Th>
                    <Table.Th>Pause Status</Table.Th>
                    <Table.Th>Last Active</Table.Th>
                    <Table.Th>Progress</Table.Th>
                    <Table.Th>Actions</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {usersData.users
                    .filter(userData => userData?.id)
                    .map(userData => (
                      <Table.Tr key={userData.id}>
                        <Table.Td>
                          <Stack gap='xs'>
                            <Text fw={500}>{userData?.username}</Text>
                            <Text size='sm' c='dimmed'>
                              {userData?.email}
                            </Text>
                            <Group gap='xs'>
                              {userData?.roles?.map((role: Role) => (
                                <Badge key={role.id} color='green' size='xs'>
                                  {role.name}
                                </Badge>
                              ))}
                            </Group>
                          </Stack>
                        </Table.Td>
                        <Table.Td>
                          <Badge color='gray' size='sm'>
                            {userData?.current_level || 'A1'}
                          </Badge>
                        </Table.Td>
                        <Table.Td>
                          <Text size='sm'>
                            {userData?.preferred_language || 'N/A'}
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          {userData?.ai_enabled ? (
                            <Badge color='blue' size='sm'>
                              AI Enabled
                            </Badge>
                          ) : (
                            <Badge color='gray' size='sm'>
                              AI Disabled
                            </Badge>
                          )}
                        </Table.Td>
                        <Table.Td>
                          {userData?.is_paused ? (
                            <Badge color='red' size='sm'>
                              Paused
                            </Badge>
                          ) : (
                            <Badge color='green' size='sm'>
                              Active
                            </Badge>
                          )}
                        </Table.Td>
                        <Table.Td>
                          <Text size='sm'>
                            {userData?.last_active
                              ? new Date(
                                  userData.last_active
                                ).toLocaleDateString()
                              : 'Never'}
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          <Text size='sm' c='dimmed'>
                            Progress data not available
                          </Text>
                        </Table.Td>
                        <Table.Td>
                          <Menu>
                            <Menu.Target>
                              <ActionIcon variant='subtle' size='sm'>
                                <IconDotsVertical size={16} />
                              </ActionIcon>
                            </Menu.Target>
                            <Menu.Dropdown>
                              <Menu.Item
                                leftSection={<IconEdit size={16} />}
                                onClick={() => {
                                  setSelectedUser(userData || null);
                                  setEditForm({
                                    username: userData?.username || '',
                                    email: userData?.email || '',
                                    timezone: userData?.timezone || '',
                                    preferred_language:
                                      userData?.preferred_language || '',
                                    current_level:
                                      userData?.current_level || '',
                                    ai_enabled: userData?.ai_enabled || false,
                                    ai_provider: userData?.ai_provider || '',
                                    ai_model: userData?.ai_model || '',
                                    api_key: '',
                                    selectedRoles:
                                      userData?.roles?.map(
                                        (role: Role) => role.name
                                      ) || [],
                                  });
                                  setIsEditModalOpen(true);
                                }}
                              >
                                Edit
                              </Menu.Item>
                              <Menu.Item
                                leftSection={<IconKey size={16} />}
                                onClick={() => {
                                  setSelectedUser(userData || null);
                                  setIsPasswordModalOpen(true);
                                }}
                              >
                                Reset Password
                              </Menu.Item>
                              {userData?.is_paused ? (
                                <Menu.Item
                                  leftSection={<IconPlayerPlay size={16} />}
                                  onClick={() => {
                                    if (userData?.id) {
                                      resumeUserMutation.mutate(userData.id, {
                                        onSuccess: () => {
                                          notifications.show({
                                            title: 'Success',
                                            message: `User ${userData.username} resumed successfully`,
                                            color: 'green',
                                            icon: <IconCheck size={16} />,
                                          });
                                        },
                                        onError: () => {
                                          notifications.show({
                                            title: 'Error',
                                            message: `Failed to resume user ${userData.username}`,
                                            color: 'red',
                                            icon: <IconX size={16} />,
                                          });
                                        },
                                      });
                                    }
                                  }}
                                >
                                  Resume User
                                </Menu.Item>
                              ) : (
                                <Menu.Item
                                  leftSection={<IconPlayerPause size={16} />}
                                  onClick={() => {
                                    if (userData?.id) {
                                      pauseUserMutation.mutate(userData.id, {
                                        onSuccess: () => {
                                          notifications.show({
                                            title: 'Success',
                                            message: `User ${userData.username} paused successfully`,
                                            color: 'green',
                                            icon: <IconCheck size={16} />,
                                          });
                                        },
                                        onError: () => {
                                          notifications.show({
                                            title: 'Error',
                                            message: `Failed to pause user ${userData.username}`,
                                            color: 'red',
                                            icon: <IconX size={16} />,
                                          });
                                        },
                                      });
                                    }
                                  }}
                                >
                                  Pause User
                                </Menu.Item>
                              )}
                              <Menu.Item
                                leftSection={<IconRefresh size={16} />}
                                onClick={() => {
                                  setSelectedUser(userData || null);
                                  setIsClearDataModalOpen(true);
                                }}
                              >
                                Clear Data
                              </Menu.Item>
                              <Menu.Divider />
                              <Menu.Item
                                leftSection={<IconTrash size={16} />}
                                color='red'
                                onClick={() => {
                                  setUserToDelete({
                                    id: userData?.id || 0,
                                    username: userData?.username || '',
                                  });
                                  setIsDeleteModalOpen(true);
                                }}
                              >
                                Delete
                              </Menu.Item>
                            </Menu.Dropdown>
                          </Menu>
                        </Table.Td>
                      </Table.Tr>
                    ))}
                </Table.Tbody>
              </Table>

              {/* Pagination */}
              <Group justify='center' mt='lg'>
                <Pagination
                  total={totalPages}
                  value={currentPage}
                  onChange={setCurrentPage}
                  size='sm'
                />
              </Group>
            </>
          ) : (
            <Center py='xl'>
              <Stack align='center' gap='md'>
                <IconUsers size={48} color='gray' />
                <Text size='lg' fw={500}>
                  No users found
                </Text>
                <Text size='sm' color='dimmed'>
                  Try adjusting your filters or create a new user
                </Text>
              </Stack>
            </Center>
          )}
        </Card>

        {/* Create User Modal */}
        <Modal
          opened={isCreateModalOpen}
          onClose={() => setIsCreateModalOpen(false)}
          title='Create New User'
          size='md'
        >
          <Stack gap='md'>
            <TextInput
              label='Username'
              required
              value={createForm.username}
              onChange={e =>
                setCreateForm({ ...createForm, username: e.target.value })
              }
            />
            <TextInput
              label='Email'
              type='email'
              value={createForm.email}
              onChange={e =>
                setCreateForm({ ...createForm, email: e.target.value })
              }
            />
            <TextInput
              label='Timezone'
              value={createForm.timezone}
              onChange={e =>
                setCreateForm({ ...createForm, timezone: e.target.value })
              }
            />
            <PasswordInput
              label='Password'
              required
              value={createForm.password}
              onChange={e =>
                setCreateForm({ ...createForm, password: e.target.value })
              }
            />
            <Select
              label='Preferred Language'
              data={[
                { value: 'italian', label: 'Italian' },
                { value: 'russian', label: 'Russian' },
                { value: 'french', label: 'French' },
                { value: 'japanese', label: 'Japanese' },
                { value: 'chinese', label: 'Chinese' },
                { value: 'german', label: 'German' },
              ]}
              value={createForm.preferred_language}
              onChange={value =>
                setCreateForm({
                  ...createForm,
                  preferred_language: value || 'italian',
                })
              }
            />
            <Select
              label='Current Level'
              data={[
                { value: 'A1', label: 'A1 - Beginner' },
                { value: 'A2', label: 'A2 - Elementary' },
                { value: 'B1', label: 'B1 - Intermediate' },
                { value: 'B1+', label: 'B1+ - Upper Intermediate' },
                { value: 'B1++', label: 'B1++ - Advanced Intermediate' },
                { value: 'B2', label: 'B2 - Upper Intermediate' },
                { value: 'C1', label: 'C1 - Advanced' },
                { value: 'C2', label: 'C2 - Mastery' },
              ]}
              value={createForm.current_level}
              onChange={value =>
                setCreateForm({ ...createForm, current_level: value || 'A1' })
              }
            />

            {/* Role Management */}
            {allRoles && (
              <Stack gap='xs'>
                <Text size='sm' fw={500}>
                  User Roles
                </Text>
                {allRoles.map((role: Role) => (
                  <Checkbox
                    key={role.id}
                    label={`${role.name} - ${role.description}`}
                    checked={createForm.selectedRoles.includes(role.name)}
                    onChange={e => {
                      if (e.currentTarget.checked) {
                        setCreateForm({
                          ...createForm,
                          selectedRoles: [
                            ...createForm.selectedRoles,
                            role.name,
                          ],
                        });
                      } else {
                        setCreateForm({
                          ...createForm,
                          selectedRoles: createForm.selectedRoles.filter(
                            r => r !== role.name
                          ),
                        });
                      }
                    }}
                  />
                ))}
              </Stack>
            )}

            <Group justify='flex-end' gap='xs'>
              <Button
                variant='light'
                onClick={() => setIsCreateModalOpen(false)}
              >
                Cancel
              </Button>
              <Button
                onClick={handleCreateUser}
                loading={createUserMutation.isPending}
                disabled={!createForm.username || !createForm.password}
              >
                Create User
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Edit User Modal */}
        <Modal
          opened={isEditModalOpen}
          onClose={() => setIsEditModalOpen(false)}
          title='Edit User'
          size='md'
        >
          <Stack gap='md'>
            <TextInput
              label='Username'
              value={editForm.username}
              onChange={e =>
                setEditForm({ ...editForm, username: e.target.value })
              }
            />
            <TextInput
              label='Email'
              type='email'
              value={editForm.email}
              onChange={e =>
                setEditForm({ ...editForm, email: e.target.value })
              }
            />
            <TextInput
              label='Timezone'
              value={editForm.timezone}
              onChange={e =>
                setEditForm({ ...editForm, timezone: e.target.value })
              }
            />
            <Select
              label='Preferred Language'
              data={[
                { value: 'italian', label: 'Italian' },
                { value: 'russian', label: 'Russian' },
                { value: 'french', label: 'French' },
                { value: 'japanese', label: 'Japanese' },
                { value: 'chinese', label: 'Chinese' },
                { value: 'german', label: 'German' },
              ]}
              value={editForm.preferred_language}
              onChange={value =>
                setEditForm({ ...editForm, preferred_language: value || '' })
              }
            />
            <Select
              label='Current Level'
              data={[
                { value: 'A1', label: 'A1 - Beginner' },
                { value: 'A2', label: 'A2 - Elementary' },
                { value: 'B1', label: 'B1 - Intermediate' },
                { value: 'B1+', label: 'B1+ - Upper Intermediate' },
                { value: 'B1++', label: 'B1++ - Advanced Intermediate' },
                { value: 'B2', label: 'B2 - Upper Intermediate' },
                { value: 'C1', label: 'C1 - Advanced' },
                { value: 'C2', label: 'C2 - Mastery' },
              ]}
              value={editForm.current_level}
              onChange={value =>
                setEditForm({ ...editForm, current_level: value || '' })
              }
            />
            <Switch
              label='AI Enabled'
              checked={editForm.ai_enabled}
              onChange={e =>
                setEditForm({
                  ...editForm,
                  ai_enabled: e.currentTarget.checked,
                })
              }
            />
            <TextInput
              label='AI Provider'
              value={editForm.ai_provider}
              onChange={e =>
                setEditForm({ ...editForm, ai_provider: e.target.value })
              }
            />
            <TextInput
              label='AI Model'
              value={editForm.ai_model}
              onChange={e =>
                setEditForm({ ...editForm, ai_model: e.target.value })
              }
            />
            <PasswordInput
              label='API Key (leave empty to keep existing)'
              value={editForm.api_key}
              onChange={e =>
                setEditForm({ ...editForm, api_key: e.target.value })
              }
            />

            {/* Role Management */}
            {allRoles && (
              <Stack gap='xs'>
                <Text size='sm' fw={500}>
                  User Roles
                </Text>
                {allRoles.map((role: Role) => (
                  <Checkbox
                    key={role.id}
                    label={`${role.name} - ${role.description}`}
                    checked={editForm.selectedRoles.includes(role.name)}
                    onChange={e => {
                      if (e.currentTarget.checked) {
                        setEditForm({
                          ...editForm,
                          selectedRoles: [...editForm.selectedRoles, role.name],
                        });
                      } else {
                        setEditForm({
                          ...editForm,
                          selectedRoles: editForm.selectedRoles.filter(
                            r => r !== role.name
                          ),
                        });
                      }
                    }}
                  />
                ))}
              </Stack>
            )}

            <Group justify='flex-end' gap='xs'>
              <Button variant='light' onClick={() => setIsEditModalOpen(false)}>
                Cancel
              </Button>
              <Button
                onClick={handleUpdateUser}
                loading={updateUserMutation.isPending}
              >
                Update User
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Reset Password Modal */}
        <Modal
          opened={isPasswordModalOpen}
          onClose={() => setIsPasswordModalOpen(false)}
          title='Reset Password'
          size='sm'
        >
          <Stack gap='md'>
            <PasswordInput
              label='New Password'
              required
              value={passwordForm.new_password}
              onChange={e =>
                setPasswordForm({
                  ...passwordForm,
                  new_password: e.target.value,
                })
              }
            />
            <Group justify='flex-end' gap='xs'>
              <Button
                variant='light'
                onClick={() => setIsPasswordModalOpen(false)}
              >
                Cancel
              </Button>
              <Button
                onClick={handleResetPassword}
                loading={resetPasswordMutation.isPending}
                disabled={!passwordForm.new_password}
              >
                Reset Password
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Clear User Data Modal */}
        <Modal
          opened={isClearDataModalOpen}
          onClose={() => setIsClearDataModalOpen(false)}
          title='Clear User Data'
          size='sm'
        >
          <Stack gap='md'>
            <Alert icon={<IconAlertTriangle size={16} />} color='orange'>
              This will clear all user activity data (progress, responses, etc.)
              but keep the user account. This action cannot be undone.
            </Alert>
            <Group justify='flex-end' gap='xs'>
              <Button
                variant='light'
                onClick={() => setIsClearDataModalOpen(false)}
              >
                Cancel
              </Button>
              <Button
                color='orange'
                onClick={handleClearUserData}
                loading={clearUserDataMutation.isPending}
              >
                Clear Data
              </Button>
            </Group>
          </Stack>
        </Modal>

        {/* Delete User Confirmation Modal */}
        <Modal
          opened={isDeleteModalOpen}
          onClose={() => setIsDeleteModalOpen(false)}
          title='Confirm Deletion'
          size='sm'
        >
          <Stack gap='md'>
            <Alert icon={<IconAlertTriangle size={16} />} color='red'>
              Are you sure you want to delete user "{userToDelete?.username}"?
              This action cannot be undone.
            </Alert>
            <Group justify='flex-end' gap='xs'>
              <Button
                variant='light'
                onClick={() => setIsDeleteModalOpen(false)}
              >
                Cancel
              </Button>
              <Button
                color='red'
                onClick={() => {
                  if (userToDelete?.id) {
                    handleDeleteUser(userToDelete.id);
                  }
                  setIsDeleteModalOpen(false);
                  setUserToDelete(null);
                }}
                loading={deleteUserMutation.isPending}
              >
                Delete User
              </Button>
            </Group>
          </Stack>
        </Modal>
      </Stack>
    </Container>
  );
};

export default UserManagementPage;
