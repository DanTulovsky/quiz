import React, { ReactNode, useCallback } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useGetV1SettingsLevels } from '../api/api';
import { useTheme } from '../contexts/ThemeContext';
import { useMobileDetection } from '../hooks/useMobileDetection';
import { useHotkeys } from 'react-hotkeys-hook';
import {
  AppShell,
  Text,
  ThemeIcon,
  NavLink,
  ActionIcon,
  Badge,
  Container,
  Stack,
  Group,
  Tooltip,
  useMantineTheme,
  Divider,
} from '@mantine/core';
import {
  IconBook2,
  IconChartBar,
  IconSettings,
  IconLogout,
  IconBrain,
  IconWorld,
  IconTrophy,
  IconSun,
  IconMoon,
  IconFileText,
  IconShield,
  IconCalendar,
  IconAbc,
  IconDeviceMobile,
} from '@tabler/icons-react';
import WorkerStatus from './WorkerStatus';
import VersionDisplay from './VersionDisplay';

interface LayoutProps {
  children: ReactNode;
}

// Add this type for the levels API response
interface LevelsApiResponse {
  levels: string[];
  level_descriptions: Record<string, string>;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  const { user, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();
  const theme = useMantineTheme();
  const { colorScheme, setColorScheme } = useTheme();
  // Expose current override to determine if we should offer switch back to mobile
  const { setMobileView, deviceView } = useMobileDetection();

  // Detect if we're on a mobile device (by viewport or user agent)
  const isMobileViewport = window.innerWidth < 768;
  const isMobileUA =
    /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
      navigator.userAgent
    );
  const shouldShowMobileButton = deviceView === 'desktop' || isMobileViewport || isMobileUA;
  // Only call the hook when we have a valid language parameter
  const language = user?.preferred_language;
  const hasValidLanguage = Boolean(language && language.trim() !== '');
  const { data: levelsData } = useGetV1SettingsLevels<LevelsApiResponse>(
    hasValidLanguage ? { language: language! } : undefined,
    {
      query: {
        enabled: hasValidLanguage,
      },
    }
  );
  const levelDescriptions = levelsData?.level_descriptions || {};

  // Check if user has admin role
  const isAdmin = user?.roles?.some(role => role.name === 'admin') || false;

  // Use useCallback to prevent recreation of navigation array
  const navigation = useCallback(() => {
    const mainNav = [
      { name: 'Quiz', href: '/quiz', icon: IconBook2, testId: 'nav-quiz' },
      {
        name: 'Vocabulary',
        href: '/vocabulary',
        icon: IconAbc,
        testId: 'nav-vocab',
      },
      {
        name: 'Reading Comprehension',
        href: '/reading-comprehension',
        icon: IconFileText,
        testId: 'nav-reading',
      },
      {
        name: 'Daily',
        href: '/daily',
        icon: IconCalendar,
        testId: 'nav-daily',
      },
    ];

    const adminNav = isAdmin
      ? [
          {
            name: 'Admin',
            href: '/admin',
            icon: IconShield,
            testId: 'nav-admin',
          },
        ]
      : [];

    return { mainNav, adminNav };
  }, [isAdmin]);

  const { mainNav, adminNav } = navigation();

  // Navigation keyboard shortcuts (Shift + number)
  // Register individual hotkeys for better compatibility
  useHotkeys(
    'shift+1',
    e => {
      e.preventDefault();
      if (mainNav[0] && mainNav[0].href !== location.pathname) {
        navigate(mainNav[0].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+2',
    e => {
      e.preventDefault();
      if (mainNav[1] && mainNav[1].href !== location.pathname) {
        navigate(mainNav[1].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+3',
    e => {
      e.preventDefault();
      if (mainNav[2] && mainNav[2].href !== location.pathname) {
        navigate(mainNav[2].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+4',
    e => {
      e.preventDefault();
      if (mainNav[3] && mainNav[3].href !== location.pathname) {
        navigate(mainNav[3].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+5',
    e => {
      e.preventDefault();
      if (adminNav[0] && adminNav[0].href !== location.pathname) {
        navigate(adminNav[0].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  const handleLogout = async () => {
    await logout();
  };

  const toggleColorScheme = () => {
    setColorScheme(colorScheme === 'dark' ? 'light' : 'dark');
  };

  const handleSwitchToMobile = () => {
    setMobileView();
    // Convert current path to mobile path
    const mobilePath = '/m' + location.pathname + location.search;
    navigate(mobilePath);
  };

  return (
    <AppShell
      header={{ height: 70 }}
      navbar={{ width: 280, breakpoint: 'sm' }}
      padding='md'
    >
      <AppShell.Header>
        <Container size='lg' px={0} h='100%'>
          <Group h='100%' px='md' py='xs' justify='space-between' wrap='nowrap'>
            {/* Logo */}
            <Group gap='sm'>
              <ThemeIcon size='lg' radius='md'>
                <IconBrain size={20} />
              </ThemeIcon>
              <div>
                <Text fw={700} size='lg' data-testid='quiz-title'>
                  Quiz
                </Text>
                <Text size='xs' c='dimmed'>
                  Adaptive Learning
                </Text>
              </div>
            </Group>

            {/* User Info */}
            <Group gap='md'>
              <WorkerStatus />
              <Group gap='xs'>
                <IconWorld size={16} />
                <Text size='sm' fw={500} data-testid='quiz-title'>
                  {user?.preferred_language || 'italian'}
                </Text>
              </Group>
              <Group gap='xs'>
                <IconTrophy size={16} />
                <Tooltip
                  label={
                    levelDescriptions[user?.current_level || 'A1'] ||
                    'Level description not available'
                  }
                  position='bottom'
                  withArrow
                >
                  <Badge variant='light' size='sm' data-testid='quiz-level'>
                    {user?.current_level || 'A1'}
                  </Badge>
                </Tooltip>
              </Group>
              {/* Progress and Settings icons in header */}
              <Group gap={4}>
                <Tooltip
                  label='View your learning progress and statistics'
                  position='bottom'
                  withArrow
                >
                  <ActionIcon
                    component={Link}
                    to='/progress'
                    variant={
                      location.pathname.startsWith('/progress')
                        ? 'light'
                        : 'subtle'
                    }
                    color={
                      location.pathname.startsWith('/progress')
                        ? 'primary'
                        : undefined
                    }
                    size='lg'
                    aria-label='Progress'
                  >
                    <IconChartBar
                      size={20}
                      color={
                        location.pathname.startsWith('/progress')
                          ? theme.colors[theme.primaryColor][6]
                          : undefined
                      }
                    />
                  </ActionIcon>
                </Tooltip>
                <Tooltip
                  label={
                    colorScheme === 'dark'
                      ? 'Switch to light mode'
                      : 'Switch to dark mode'
                  }
                  position='bottom'
                  withArrow
                >
                  <ActionIcon
                    onClick={toggleColorScheme}
                    variant='subtle'
                    size='lg'
                    aria-label='Toggle color scheme'
                  >
                    {colorScheme === 'dark' ? (
                      <IconSun size={20} />
                    ) : (
                      <IconMoon size={20} />
                    )}
                  </ActionIcon>
                </Tooltip>
                {shouldShowMobileButton && (
                  <Tooltip
                    label='Switch to mobile-optimized view'
                    position='bottom'
                    withArrow
                  >
                    <ActionIcon
                      onClick={handleSwitchToMobile}
                      variant='subtle'
                      size='lg'
                      aria-label='Switch to mobile view'
                    >
                      <IconDeviceMobile size={20} />
                    </ActionIcon>
                  </Tooltip>
                )}
                <Tooltip
                  label='Manage your account settings and preferences'
                  position='bottom'
                  withArrow
                >
                  <ActionIcon
                    component={Link}
                    to='/settings'
                    variant={
                      location.pathname.startsWith('/settings')
                        ? 'light'
                        : 'subtle'
                    }
                    color={
                      location.pathname.startsWith('/settings')
                        ? 'primary'
                        : undefined
                    }
                    size='lg'
                    aria-label='Settings'
                  >
                    <IconSettings
                      size={20}
                      color={
                        location.pathname.startsWith('/settings')
                          ? theme.colors[theme.primaryColor][6]
                          : undefined
                      }
                    />
                  </ActionIcon>
                </Tooltip>
              </Group>
              <Group gap='sm'>
                <Text size='sm' fw={500}>
                  {user?.username}
                </Text>
                <ActionIcon
                  variant='subtle'
                  onClick={handleLogout}
                  title='Logout'
                >
                  <IconLogout size={16} />
                </ActionIcon>
              </Group>
            </Group>
          </Group>
        </Container>
      </AppShell.Header>

      <AppShell.Navbar>
        <Stack gap='xs' mt='md'>
          {/* Main Navigation */}
          {mainNav.map((item, index) => (
            <React.Fragment key={item.name}>
              {item.name === 'Daily' && (
                <Divider my='xs' label='Practice' labelPosition='center' />
              )}
              <NavLink
                component={Link}
                to={item.href}
                label={
                  <Group justify='space-between' w='100%' align='center'>
                    <span>{item.name}</span>
                    <Badge
                      size='xs'
                      variant='light'
                      color={theme.colors.gray[6]}
                      styles={{
                        root: {
                          backgroundColor:
                            colorScheme === 'dark'
                              ? theme.colors.gray[7]
                              : theme.white,
                          color:
                            colorScheme === 'dark'
                              ? theme.colors.gray[2]
                              : theme.colors.gray[8],
                          border: `1px solid ${colorScheme === 'dark' ? theme.colors.gray[6] : theme.colors.gray[3]}`,
                          fontWeight: 600,
                          minWidth: '32px',
                          height: '22px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: '11px',
                          letterSpacing: '0.5px',
                        },
                      }}
                    >
                      ⇧{index + 1}
                    </Badge>
                  </Group>
                }
                leftSection={<item.icon size={20} />}
                active={
                  item.name === 'Daily'
                    ? location.pathname.startsWith(item.href)
                    : location.pathname === item.href
                }
                data-testid={item.testId}
                title={`${item.name} (Shift+${index + 1})`}
              />
            </React.Fragment>
          ))}

          {/* Admin Section */}
          {adminNav.length > 0 && (
            <>
              <Divider my='md' label='Administration' labelPosition='center' />
              {adminNav.map((item, index) => (
                <NavLink
                  key={item.name}
                  component={Link}
                  to={item.href}
                  label={
                    <Group justify='space-between' w='100%' align='center'>
                      <span>{item.name}</span>
                      <Badge
                        size='xs'
                        variant='light'
                        color={theme.colors.gray[6]}
                        styles={{
                          root: {
                            backgroundColor:
                              colorScheme === 'dark'
                                ? theme.colors.gray[7]
                                : theme.white,
                            color:
                              colorScheme === 'dark'
                                ? theme.colors.gray[2]
                                : theme.colors.gray[8],
                            border: `1px solid ${colorScheme === 'dark' ? theme.colors.gray[6] : theme.colors.gray[3]}`,
                            fontWeight: 600,
                            minWidth: '32px',
                            height: '22px',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            fontSize: '11px',
                            letterSpacing: '0.5px',
                          },
                        }}
                      >
                        ⇧{index + 5}
                      </Badge>
                    </Group>
                  }
                  leftSection={<item.icon size={20} />}
                  active={
                    item.name === 'Admin'
                      ? location.pathname.startsWith(item.href)
                      : location.pathname === item.href
                  }
                  data-testid={item.testId}
                  title={`${item.name} (Shift+${index + 4})`}
                />
              ))}
            </>
          )}
        </Stack>
      </AppShell.Navbar>

      <AppShell.Main>{children}</AppShell.Main>
      <VersionDisplay />
    </AppShell>
  );
};

export default Layout;
