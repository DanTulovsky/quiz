import React from 'react';
import {
  AppShell,
  Text,
  Burger,
  useMantineTheme,
  Group,
  Button,
  ActionIcon,
  Stack,
  Divider,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconSun,
  IconMoon,
  IconDeviceDesktop,
  IconBook2,
  IconAbc,
  IconMessage,
  IconCalendar,
  IconLogout,
  IconBookmark,
  IconBook,
  IconBrain,
} from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';
import { useMobileDetection } from '../hooks/useMobileDetection';
import { useTheme } from '../contexts/ThemeContext';
import { useNavigate, useLocation } from 'react-router-dom';
import VersionDisplay from './VersionDisplay';

interface MobileLayoutProps {
  children: React.ReactNode;
}

const MobileLayout: React.FC<MobileLayoutProps> = ({ children }) => {
  const theme = useMantineTheme();
  const [opened, { toggle }] = useDisclosure(false);
  const { user, logout } = useAuth();
  const { setDesktopView } = useMobileDetection();
  const { colorScheme, setColorScheme } = useTheme();
  const navigate = useNavigate();
  const location = useLocation();

  const isDark = colorScheme === 'dark';
  const toggleTheme = () => {
    setColorScheme(isDark ? 'light' : 'dark');
  };

  const handleLogout = () => {
    logout();
    navigate('/m/login');
  };

  const handleNavigation = (path: string) => {
    navigate(path);
  };

  const getActiveTab = () => {
    const path = location.pathname;
    if (path.startsWith('/m/quiz')) return 'quiz';
    if (path.startsWith('/m/vocabulary')) return 'vocabulary';
    if (path.startsWith('/m/reading-comprehension')) return 'reading';
    if (path.startsWith('/m/story')) return 'story';
    if (path.startsWith('/m/daily')) return 'daily';
    if (path.startsWith('/m/conversations')) return 'conversations';
    if (path.startsWith('/m/bookmarks')) return 'bookmarks';
    return 'quiz';
  };

  const activeTab = getActiveTab();

  const navItems = [
    { key: 'quiz', label: 'Quiz', icon: IconBook2, path: '/m/quiz' },
    {
      key: 'vocabulary',
      label: 'Vocabulary',
      icon: IconAbc,
      path: '/m/vocabulary',
    },
    {
      key: 'reading',
      label: 'Reading',
      icon: IconMessage,
      path: '/m/reading-comprehension',
    },
    { key: 'story', label: 'Story', icon: IconBook, path: '/m/story' },
    { key: 'daily', label: 'Daily', icon: IconCalendar, path: '/m/daily' },
    {
      key: 'conversations',
      label: 'AI Conversations',
      icon: IconBrain,
      path: '/m/conversations',
    },
    {
      key: 'bookmarks',
      label: 'Bookmarked Messages',
      icon: IconBookmark,
      path: '/m/bookmarks',
    },
  ];

  return (
    <AppShell
      styles={{
        root: {
          height: '100vh',
        },
      }}
      padding='md'
      navbar={{
        width: { sm: 200, lg: 300 },
        breakpoint: 'sm',
        collapsed: { mobile: !opened },
      }}
      header={{ height: { base: 50, md: 70 } }}
    >
      <AppShell.Navbar
        p='md'
        style={{ display: 'flex', flexDirection: 'column' }}
      >
        <AppShell.Section grow>
          <Stack gap='md'>
            <Text fw={500}>Menu</Text>
            {/* Primary navigation moved from footer */}
            {navItems.map(item => {
              const Icon = item.icon;
              return (
                <React.Fragment key={item.key}>
                  {item.key === 'daily' && (
                    <Divider my='sm' label='Practice' labelPosition='center' />
                  )}
                  {item.key === 'conversations' && (
                    <Divider
                      my='sm'
                      label='AI History'
                      labelPosition='center'
                    />
                  )}
                  <Button
                    variant={activeTab === item.key ? 'light' : 'subtle'}
                    leftSection={<Icon size={16} />}
                    onClick={() => {
                      handleNavigation(item.path);
                      if (opened) toggle(); // close menu after navigation
                    }}
                    fullWidth
                    justify='flex-start'
                  >
                    {item.label}
                  </Button>
                </React.Fragment>
              );
            })}
            <Divider my='sm' />
            <Button
              variant='subtle'
              leftSection={<IconDeviceDesktop size={16} />}
              onClick={() => {
                setDesktopView();
                navigate('/quiz');
              }}
              fullWidth
              justify='flex-start'
            >
              Desktop View
            </Button>
            <Divider my='sm' />
            <Button
              variant='subtle'
              color='red'
              leftSection={<IconLogout size={16} />}
              onClick={handleLogout}
              fullWidth
              justify='flex-start'
            >
              Logout
            </Button>
          </Stack>
        </AppShell.Section>
        {/* Spacer pushes version to bottom */}
        <div style={{ flexGrow: 1 }} />
        <VersionDisplay copyOnClick={false} position='static' />
      </AppShell.Navbar>

      <AppShell.Header p='md' className='mobile-safe-header'>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            height: '100%',
            justifyContent: 'space-between',
          }}
        >
          <Burger
            opened={opened}
            onClick={toggle}
            size='sm'
            color={theme.colors.gray[6]}
          />

          <Group>
            <ActionIcon
              variant='outline'
              color={isDark ? 'yellow' : 'blue'}
              onClick={() => toggleTheme()}
              title='Toggle color scheme'
            >
              {isDark ? <IconSun size={18} /> : <IconMoon size={18} />}
            </ActionIcon>
            <Text size='sm' c='dimmed'>
              {user?.email}
            </Text>
          </Group>
        </div>
      </AppShell.Header>

      {/* Removed footer displaying active section label */}

      <AppShell.Main>{children}</AppShell.Main>
    </AppShell>
  );
};

export default MobileLayout;
