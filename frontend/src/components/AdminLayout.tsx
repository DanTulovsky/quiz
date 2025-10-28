import React, { useState } from 'react';
import {
  AppShell,
  Text,
  Group,
  Button,
  Stack,
  NavLink,
  Divider,
} from '@mantine/core';
import {
  IconUsers,
  IconChartBar,
  IconSettings,
  IconServer,
  IconHome,
  IconLogout,
  IconDatabase,
  IconArrowLeft,
  IconBell,
  IconCalendar,
  IconBook,
  IconTrendingUp,
} from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';
import { useNavigate, useLocation, Link } from 'react-router-dom';
import FeedbackModal from './FeedbackModal';
import { IconBug } from '@tabler/icons-react';

interface AdminLayoutProps {
  children: React.ReactNode;
}

const AdminLayout: React.FC<AdminLayoutProps> = ({ children }) => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [feedbackModalOpened, setFeedbackModalOpened] = useState(false);

  const backendNavItems = [
    {
      label: 'Backend Overview',
      icon: <IconServer size={16} />,
      path: '/admin/backend/adminz',
    },
    {
      label: 'User Management',
      icon: <IconUsers size={16} />,
      path: '/admin/backend/userz',
    },
    {
      label: 'Data Explorer',
      icon: <IconDatabase size={16} />,
      path: '/admin/backend/data-explorer',
    },
    {
      label: 'Story Explorer',
      icon: <IconBook size={16} />,
      path: '/admin/backend/story-explorer',
    },
  ];

  const workerNavItems = [
    {
      label: 'Worker Admin',
      icon: <IconSettings size={16} />,
      path: '/admin/worker/adminz',
    },
    {
      label: 'Analytics',
      icon: <IconChartBar size={16} />,
      path: '/admin/worker/analyticsz',
    },
    {
      label: 'Daily',
      icon: <IconCalendar size={16} />,
      path: '/admin/worker/daily',
    },
    {
      label: 'Notifications',
      icon: <IconBell size={16} />,
      path: '/admin/worker/notifications',
    },
  ];

  const statsNavItems = [
    {
      label: 'Translation Usage',
      icon: <IconTrendingUp size={16} />,
      path: '/admin/stats/translation',
    },
    {
      label: 'Feedback Reports',
      icon: <IconBug size={16} />,
      path: '/admin/feedback',
    },
  ];

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const handleNavClick = (path: string) => {
    // Navigate within the app for all pages
    navigate(path);
  };

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{ width: 300, breakpoint: 'sm' }}
      padding='md'
    >
      <AppShell.Header>
        <Group h='100%' px='md' justify='space-between'>
          <Text fw={700} size='lg'>
            Quiz Admin
          </Text>
          <Group>
            <Button
              variant='subtle'
              leftSection={<IconBug size={16} />}
              onClick={() => setFeedbackModalOpened(true)}
              data-testid='feedback-button'
            >
              Feedback
            </Button>
            <Text size='sm' c='dimmed'>
              {user?.username}
            </Text>
            <Button
              variant='subtle'
              leftSection={<IconLogout size={16} />}
              onClick={handleLogout}
            >
              Logout
            </Button>
          </Group>
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p='md'>
        <Stack gap='xs'>
          {/* Dashboard */}
          <NavLink
            label='Dashboard'
            leftSection={<IconHome size={16} />}
            active={location.pathname === '/admin'}
            onClick={() => handleNavClick('/admin')}
            variant='filled'
          />

          {/* Back to Main Site */}
          <Button
            component={Link}
            to='/quiz'
            variant='light'
            color='gray'
            leftSection={<IconArrowLeft size={16} />}
            fullWidth
            size='sm'
            mt='xs'
            mb='xs'
          >
            Back to Main Site
          </Button>

          <Divider my='xs' />

          {/* Backend Group */}
          <Text size='xs' fw={500} c='dimmed' tt='uppercase' mb='xs'>
            Backend
          </Text>
          {backendNavItems.map(item => (
            <NavLink
              key={item.path}
              label={item.label}
              leftSection={item.icon}
              active={
                location.pathname === item.path ||
                (item.path !== '/admin' &&
                  location.pathname.startsWith(item.path))
              }
              onClick={() => handleNavClick(item.path)}
              variant='filled'
            />
          ))}

          <Divider my='xs' />

          {/* Worker Group */}
          <Text size='xs' fw={500} c='dimmed' tt='uppercase' mb='xs'>
            Worker
          </Text>
          {workerNavItems.map(item => (
            <NavLink
              key={item.path}
              label={item.label}
              leftSection={item.icon}
              active={
                location.pathname === item.path ||
                (item.path !== '/admin' &&
                  location.pathname.startsWith(item.path))
              }
              onClick={() => handleNavClick(item.path)}
              variant='filled'
            />
          ))}

          <Divider my='xs' />

          {/* Stats Group */}
          <Text size='xs' fw={500} c='dimmed' tt='uppercase' mb='xs'>
            Stats
          </Text>
          {statsNavItems.map(item => (
            <NavLink
              key={item.path}
              label={item.label}
              leftSection={item.icon}
              active={
                location.pathname === item.path ||
                (item.path !== '/admin' &&
                  location.pathname.startsWith(item.path))
              }
              onClick={() => handleNavClick(item.path)}
              variant='filled'
            />
          ))}
        </Stack>
      </AppShell.Navbar>

      <AppShell.Main>{children}</AppShell.Main>
      <FeedbackModal
        opened={feedbackModalOpened}
        onClose={() => setFeedbackModalOpened(false)}
      />
    </AppShell>
  );
};

export default AdminLayout;
