import React from 'react';
import {
  Container,
  Title,
  Text,
  Card,
  Group,
  Badge,
  Stack,
  Loader,
  Center,
  SimpleGrid,
  Box,
} from '@mantine/core';
import {
  IconUsers,
  IconBrain,
  IconDatabase,
  IconActivity,
} from '@tabler/icons-react';
import { useBackendAdminData } from '../../api/admin';
import { useAuth } from '../../hooks/useAuth';
import { Navigate } from 'react-router-dom';

const BackendAdminPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();

  // API hooks
  const { data: adminData, isLoading } = useBackendAdminData();

  // Check if user is admin
  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }

  const isAdmin = user.roles?.some(role => role.name === 'admin') || false;
  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  if (isLoading) {
    return (
      <Center h='50vh'>
        <Stack align='center' gap='md'>
          <Loader size='lg' />
          <Text>Loading admin data...</Text>
        </Stack>
      </Center>
    );
  }

  return (
    <Container size='xl' py='xl'>
      <Stack gap='xl'>
        <Group justify='space-between' align='center'>
          <div>
            <Title order={1}>Backend Overview</Title>
            <Text color='dimmed' size='lg'>
              System overview and user statistics
            </Text>
          </div>
        </Group>

        {/* Statistics Cards */}
        <SimpleGrid cols={{ base: 1, md: 4 }} spacing='lg'>
          <Card shadow='sm' padding='lg' radius='md' withBorder>
            <Group justify='space-between' mb='xs'>
              <IconUsers size={24} color='var(--mantine-color-blue-6)' />
            </Group>
            <Text fw={500} size='lg'>
              Total Users
            </Text>
            <Text size='2rem' fw={700} c='blue'>
              {adminData?.user_stats?.total_users || 0}
            </Text>
          </Card>

          <Card shadow='sm' padding='lg' radius='md' withBorder>
            <Group justify='space-between' mb='xs'>
              <IconDatabase size={24} color='var(--mantine-color-orange-6)' />
            </Group>
            <Text fw={500} size='lg'>
              Total Questions
            </Text>
            <Text size='2rem' fw={700} c='orange'>
              {adminData?.question_stats?.total_questions || 0}
            </Text>
          </Card>

          <Card shadow='sm' padding='lg' radius='md' withBorder>
            <Group justify='space-between' mb='xs'>
              <IconBrain size={24} color='var(--mantine-color-purple-6)' />
            </Group>
            <Text fw={500} size='lg'>
              AI Requests
            </Text>
            <Text size='2rem' fw={700} c='purple'>
              {adminData?.ai_concurrency_stats?.active_requests || 0}
            </Text>
            <Text size='sm' c='dimmed'>
              / {adminData?.ai_concurrency_stats?.max_concurrent || 0} max
            </Text>
          </Card>

          <Card shadow='sm' padding='lg' radius='md' withBorder>
            <Group justify='space-between' mb='xs'>
              <IconActivity size={24} color='var(--mantine-color-teal-6)' />
            </Group>
            <Text fw={500} size='lg'>
              Active Sessions
            </Text>
            <Text size='2rem' fw={700} c='teal'>
              {adminData?.session_stats?.active_sessions || 0}
            </Text>
            <Text size='sm' c='dimmed'>
              / {adminData?.session_stats?.max_sessions || 0} max
            </Text>
          </Card>
        </SimpleGrid>

        {/* AI Concurrency Stats */}
        {adminData?.ai_concurrency_stats && (
          <Card shadow='sm' padding='lg' radius='md' withBorder>
            <Title order={3} mb='md'>
              AI Service Concurrency
            </Title>
            <SimpleGrid cols={{ base: 1, md: 2 }} spacing='sm'>
              <Box>
                <Group justify='space-between'>
                  <Text>Active Requests:</Text>
                  <Text fw={500}>
                    {adminData.ai_concurrency_stats.active_requests}/
                    {adminData.ai_concurrency_stats.max_concurrent}
                  </Text>
                </Group>
                <Group justify='space-between'>
                  <Text>Queued Requests:</Text>
                  <Text fw={500}>
                    {adminData.ai_concurrency_stats.queued_requests}
                  </Text>
                </Group>
                <Group justify='space-between'>
                  <Text>Total Requests:</Text>
                  <Text fw={500}>
                    {adminData.ai_concurrency_stats.total_requests}
                  </Text>
                </Group>
              </Box>
              <Box>
                <Group justify='space-between'>
                  <Text>Max Per User:</Text>
                  <Text fw={500}>
                    {adminData.ai_concurrency_stats.max_per_user}
                  </Text>
                </Group>
                <Group justify='space-between'>
                  <Text>Active Users:</Text>
                  <Text fw={500}>
                    {
                      Object.keys(
                        adminData.ai_concurrency_stats.user_active_count || {}
                      ).length
                    }
                  </Text>
                </Group>
              </Box>
            </SimpleGrid>
          </Card>
        )}

        {/* User Statistics */}
        <Card shadow='sm' padding='lg' radius='md' withBorder>
          <Title order={3} mb='md'>
            User Statistics
          </Title>

          {adminData?.user_stats ? (
            <Stack gap='lg'>
              {/* Summary Stats */}
              <SimpleGrid cols={{ base: 1, md: 3 }} spacing='md'>
                <Card shadow='xs' padding='md'>
                  <Text fw={500} size='lg' mb='xs'>
                    AI Usage
                  </Text>
                  <Group justify='space-between'>
                    <Text size='sm' c='dimmed'>
                      Enabled:
                    </Text>
                    <Badge color='blue'>
                      {adminData.user_stats.ai_enabled}
                    </Badge>
                  </Group>
                  <Group justify='space-between'>
                    <Text size='sm' c='dimmed'>
                      Disabled:
                    </Text>
                    <Badge color='gray'>
                      {adminData.user_stats.ai_disabled}
                    </Badge>
                  </Group>
                </Card>

                <Card shadow='xs' padding='md'>
                  <Text fw={500} size='lg' mb='xs'>
                    Activity
                  </Text>
                  <Group justify='space-between'>
                    <Text size='sm' c='dimmed'>
                      Active:
                    </Text>
                    <Badge color='green'>
                      {adminData.user_stats.active_users}
                    </Badge>
                  </Group>
                  <Group justify='space-between'>
                    <Text size='sm' c='dimmed'>
                      Inactive:
                    </Text>
                    <Badge color='red'>
                      {adminData.user_stats.inactive_users}
                    </Badge>
                  </Group>
                </Card>

                <Card shadow='xs' padding='md'>
                  <Text fw={500} size='lg' mb='xs'>
                    Performance
                  </Text>
                  <Group justify='space-between'>
                    <Text size='sm' c='dimmed'>
                      Questions:
                    </Text>
                    <Text fw={500}>
                      {adminData.user_stats.total_questions_answered}
                    </Text>
                  </Group>
                  <Group justify='space-between'>
                    <Text size='sm' c='dimmed'>
                      Accuracy:
                    </Text>
                    <Text fw={500}>
                      {adminData.user_stats.average_accuracy.toFixed(1)}%
                    </Text>
                  </Group>
                </Card>
              </SimpleGrid>

              {/* Detailed Breakdowns */}
              <SimpleGrid cols={{ base: 1, md: 2 }} spacing='md'>
                <Card shadow='xs' padding='md'>
                  <Text fw={500} size='lg' mb='xs'>
                    By Language
                  </Text>
                  <Stack gap='xs'>
                    {Object.entries(adminData.user_stats.by_language).map(
                      ([lang, count]) => (
                        <Group key={lang} justify='space-between'>
                          <Text size='sm' c='dimmed'>
                            {lang}:
                          </Text>
                          <Badge size='sm'>{count}</Badge>
                        </Group>
                      )
                    )}
                  </Stack>
                </Card>

                <Card shadow='xs' padding='md'>
                  <Text fw={500} size='lg' mb='xs'>
                    By Level
                  </Text>
                  <Stack gap='xs'>
                    {Object.entries(adminData.user_stats.by_level).map(
                      ([level, count]) => (
                        <Group key={level} justify='space-between'>
                          <Text size='sm' c='dimmed'>
                            {level}:
                          </Text>
                          <Badge size='sm'>{count}</Badge>
                        </Group>
                      )
                    )}
                  </Stack>
                </Card>
              </SimpleGrid>

              {/* AI Provider and Model Stats */}
              <SimpleGrid cols={{ base: 1, md: 2 }} spacing='md'>
                <Card shadow='xs' padding='md'>
                  <Text fw={500} size='lg' mb='xs'>
                    AI Providers
                  </Text>
                  <Stack gap='xs'>
                    {Object.entries(adminData.user_stats.by_ai_provider).map(
                      ([provider, count]) => (
                        <Group key={provider} justify='space-between'>
                          <Text size='sm' c='dimmed'>
                            {provider}:
                          </Text>
                          <Badge size='sm'>{count}</Badge>
                        </Group>
                      )
                    )}
                  </Stack>
                </Card>

                <Card shadow='xs' padding='md'>
                  <Text fw={500} size='lg' mb='xs'>
                    AI Models
                  </Text>
                  <Stack gap='xs'>
                    {Object.entries(adminData.user_stats.by_ai_model).map(
                      ([model, count]) => (
                        <Group key={model} justify='space-between'>
                          <Text size='sm' c='dimmed'>
                            {model}:
                          </Text>
                          <Badge size='sm'>{count}</Badge>
                        </Group>
                      )
                    )}
                  </Stack>
                </Card>
              </SimpleGrid>
            </Stack>
          ) : (
            <Text c='dimmed' ta='center' py='xl'>
              No user statistics available.
            </Text>
          )}
        </Card>

        {/* System Information */}
        <Card shadow='sm' padding='lg' radius='md' withBorder>
          <Title order={3} mb='md'>
            System Information
          </Title>
          <SimpleGrid cols={{ base: 1, md: 2 }} spacing='sm'>
            <Box>
              <Group justify='space-between'>
                <Text size='sm' c='dimmed'>
                  Service:
                </Text>
                <Text size='sm'>Backend</Text>
              </Group>
              <Group justify='space-between'>
                <Text size='sm' c='dimmed'>
                  Status:
                </Text>
                <Badge color='green' size='sm'>
                  Running
                </Badge>
              </Group>
              <Group justify='space-between'>
                <Text size='sm' c='dimmed'>
                  Port:
                </Text>
                <Text size='sm'>8080</Text>
              </Group>
            </Box>
            <Box>
              <Group justify='space-between'>
                <Text size='sm' c='dimmed'>
                  Database:
                </Text>
                <Badge color='green' size='sm'>
                  Connected
                </Badge>
              </Group>
              <Group justify='space-between'>
                <Text size='sm' c='dimmed'>
                  Worker Service:
                </Text>
                <Badge color='green' size='sm'>
                  Connected
                </Badge>
              </Group>
              <Group justify='space-between'>
                <Text size='sm' c='dimmed'>
                  Schema Validation:
                </Text>
                <Badge color='green' size='sm'>
                  Active
                </Badge>
              </Group>
            </Box>
          </SimpleGrid>
        </Card>
      </Stack>
    </Container>
  );
};

export default BackendAdminPage;
