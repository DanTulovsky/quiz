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
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconSun,
  IconMoon,
  IconDeviceDesktop,
  IconBook,
  IconVocabulary,
  IconMessage,
  IconCalendar,
  IconLogout,
} from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';
import { useMobileDetection } from '../hooks/useMobileDetection';
import { useTheme } from '../contexts/ThemeContext';
import { useNavigate, useLocation } from 'react-router-dom';

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
    if (path.startsWith('/m/daily')) return 'daily';
    return 'quiz';
  };

  const activeTab = getActiveTab();

  const navItems = [
    { key: 'quiz', label: 'Quiz', icon: IconBook, path: '/m/quiz' },
    {
      key: 'vocabulary',
      label: 'Vocabulary',
      icon: IconVocabulary,
      path: '/m/vocabulary',
    },
    {
      key: 'reading',
      label: 'Reading',
      icon: IconMessage,
      path: '/m/reading-comprehension',
    },
    { key: 'daily', label: 'Daily', icon: IconCalendar, path: '/m/daily' },
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
      footer={{ height: 90 }}
    >
      <AppShell.Navbar p='md'>
        <AppShell.Section grow>
          <Stack gap='md'>
            <Text fw={500}>Menu</Text>
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
          <Group>
            <Burger
              opened={opened}
              onClick={toggle}
              size='sm'
              color={theme.colors.gray[6]}
            />
            <Text>WetSnow Quiz</Text>
          </Group>

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

      <AppShell.Footer
        p='xs'
        className='mobile-safe-footer'
        style={{
          backgroundColor:
            colorScheme === 'dark'
              ? theme.colors.dark[7]
              : theme.colors.gray[1],
        }}
      >
        <Group
          justify='space-around'
          gap={4}
          style={{ height: '100%', width: '100%' }}
        >
          {navItems.map(item => {
            const Icon = item.icon;
            return (
              <Button
                key={item.key}
                variant={activeTab === item.key ? 'filled' : 'subtle'}
                size='xs'
                px={2}
                onClick={() => handleNavigation(item.path)}
                style={{
                  flex: 1,
                  maxWidth: '90px',
                  height: '75px',
                  borderRadius: theme.radius.md,
                  padding: '8px 4px',
                }}
              >
                <Stack gap={4} align='center' justify='center'>
                  <Icon size={24} />
                  <Text size='xs' tt='capitalize' fw={500} lh={1.2}>
                    {item.label}
                  </Text>
                </Stack>
              </Button>
            );
          })}
        </Group>
      </AppShell.Footer>

      <AppShell.Main>{children}</AppShell.Main>
    </AppShell>
  );
};

export default MobileLayout;
