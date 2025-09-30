import React from 'react';
import {
  Container,
  Title,
  Text,
  Card,
  Group,
  Button,
  Stack,
} from '@mantine/core';
import {
  IconUsers,
  IconChartBar,
  IconSettings,
  IconServer,
} from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';
import { Navigate } from 'react-router-dom';

const AdminPage: React.FC = () => {
  const { user, isAuthenticated } = useAuth();

  // Redirect if not authenticated or not admin
  if (!isAuthenticated || !user) {
    return <Navigate to='/login' />;
  }

  // Check if user has admin role (this will need to be updated when roles are added to the User type)
  const isAdmin = user.roles?.some(role => role.name === 'admin') || false;

  if (!isAdmin) {
    return <Navigate to='/quiz' />;
  }

  const adminSections = [
    {
      title: 'User Management',
      description: 'Manage users, roles, and permissions',
      icon: <IconUsers size={24} />,
      path: '/admin/backend/userz',
      color: 'blue',
    },

    {
      title: 'Backend Overview',
      description: 'Backend service administration and monitoring',
      icon: <IconServer size={24} />,
      path: '/admin/backend/adminz',
      color: 'orange',
    },
    {
      title: 'Worker Admin',
      description: 'Worker service administration and controls',
      icon: <IconSettings size={24} />,
      path: '/admin/worker/adminz',
      color: 'purple',
    },
    {
      title: 'Analytics',
      description: 'Advanced analytics and reporting',
      icon: <IconChartBar size={24} />,
      path: '/admin/worker/analyticsz',
      color: 'teal',
    },
  ];

  return (
    <Container size='lg' py='xl'>
      <Stack gap='xl'>
        <div>
          <Title order={1} mb='xs'>
            Admin Dashboard
          </Title>
          <Text color='dimmed' size='lg'>
            Welcome to the admin interface. Select a section to manage.
          </Text>
        </div>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
            gap: '1rem',
          }}
        >
          {adminSections.map(section => (
            <Card
              key={section.path}
              shadow='sm'
              padding='lg'
              radius='md'
              withBorder
            >
              <Group justify='space-between' mb='xs'>
                <div
                  style={{ color: `var(--mantine-color-${section.color}-6)` }}
                >
                  {section.icon}
                </div>
              </Group>

              <Text fw={500} size='lg' mb='xs'>
                {section.title}
              </Text>

              <Text size='sm' c='dimmed' mb='md'>
                {section.description}
              </Text>

              <Button
                variant='light'
                color={section.color}
                fullWidth
                onClick={() => {
                  // Navigate to the admin section (will be handled by AdminLayout)
                  window.location.href = section.path;
                }}
              >
                Open {section.title}
              </Button>
            </Card>
          ))}
        </div>
      </Stack>
    </Container>
  );
};

export default AdminPage;
