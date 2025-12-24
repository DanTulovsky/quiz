import React, { ReactNode, useCallback, useState, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useGetV1SettingsLevels } from '../api/api';
import { useTheme } from '../contexts/ThemeContext';
import { fontScaleMap } from '../theme/theme';
import { useMobileDetection } from '../hooks/useMobileDetection';
import { useQueryClient } from '@tanstack/react-query';
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
  IconBook,
  IconBook2,
  IconChartLine,
  IconAdjustments,
  IconLogout,
  IconBrain,
  IconGlobe,
  IconTrophy,
  IconSun,
  IconMoon,
  IconFile,
  IconShieldCheck,
  IconCalendar,
  IconAbc,
  IconPhone,
  IconHelp,
  IconLanguage,
  IconAlertCircle,
  IconStars,
} from '@tabler/icons-react';
import WorkerStatus from './WorkerStatus';
import VersionDisplay from './VersionDisplay';
import HelpModal from './HelpModal';
import FeedbackModal from './FeedbackModal';
import { getGetV1AiConversationsQueryKey } from '../api/api';

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
  const { colorScheme, setColorScheme, fontSize } = useTheme();
  // Expose current override to determine if we should offer switch back to mobile
  const { setMobileView, deviceView } = useMobileDetection();
  const queryClient = useQueryClient();

  // Detect if we're on a mobile device (by viewport or user agent)
  const isMobileViewport = window.innerWidth < 768;
  const isMobileUA =
    /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(
      navigator.userAgent
    );
  const shouldShowMobileButton =
    deviceView === 'desktop' || isMobileViewport || isMobileUA;
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

  // Help modal state
  const [helpModalOpened, setHelpModalOpened] = useState(false);

  // Feedback modal state
  const [feedbackModalOpened, setFeedbackModalOpened] = useState(false);

  // Check if user has seen help modal before (first login detection)
  useEffect(() => {
    if (user) {
      const hasSeenHelp = localStorage.getItem(`hasSeenHelp-${user.id}`);
      if (!hasSeenHelp) {
        // Show help modal on first login
        setHelpModalOpened(true);
        localStorage.setItem(`hasSeenHelp-${user.id}`, 'true');
      }
    }
  }, [user]);

  // Refresh AI Conversations when navigating to the page
  const refreshAiConversations = useCallback(() => {
    const conversationsListKey = getGetV1AiConversationsQueryKey({
      limit: 50,
      offset: 0,
    });
    queryClient.invalidateQueries({ queryKey: conversationsListKey });
    queryClient.refetchQueries({ queryKey: conversationsListKey });
  }, [queryClient]);

  // Refresh Bookmarked Messages when navigating to the page
  const refreshBookmarkedMessages = useCallback(() => {
    // Invalidate all bookmarked messages queries (including mobile with search params)
    queryClient.invalidateQueries({ queryKey: ['/v1/ai/bookmarks'] });
    queryClient.refetchQueries({ queryKey: ['/v1/ai/bookmarks'] });

    // Also try with different query key patterns that might be used
    queryClient.invalidateQueries({
      queryKey: ['/v1/ai/bookmarks'],
      exact: false,
    });
    queryClient.refetchQueries({
      queryKey: ['/v1/ai/bookmarks'],
      exact: false,
    });
  }, [queryClient]);

  useEffect(() => {
    if (location.pathname.startsWith('/conversations')) {
      refreshAiConversations();
    }
  }, [location.pathname, refreshAiConversations]);

  useEffect(() => {
    if (
      location.pathname.startsWith('/bookmarks') ||
      location.pathname.startsWith('/m/bookmarks')
    ) {
      refreshBookmarkedMessages();
    }
  }, [location.pathname, refreshBookmarkedMessages]);

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
        icon: IconFile,
        testId: 'nav-reading',
      },
      {
        name: 'Daily',
        href: '/daily',
        icon: IconCalendar,
        testId: 'nav-daily',
      },
      {
        name: 'Word of the Day',
        href: '/word-of-day',
        icon: IconStars,
        testId: 'nav-word-of-day',
      },
      {
        name: 'Story',
        href: '/story',
        icon: IconBook,
        testId: 'nav-story',
      },
      {
        name: 'Translation Practice',
        href: '/translation-practice',
        icon: IconLanguage,
        testId: 'nav-translation-practice',
      },
      {
        name: 'Saved AI Conversations',
        href: '/conversations',
        icon: IconBrain,
        testId: 'nav-conversations',
      },
      {
        name: 'Bookmarked AI Messages',
        href: '/bookmarks',
        icon: IconBook,
        testId: 'nav-bookmarks',
      },
      {
        name: 'Saved Snippets',
        href: '/snippets',
        icon: IconBook,
        testId: 'nav-snippets',
      },
      {
        name: 'Phrasebook',
        href: '/phrasebook',
        icon: IconLanguage,
        testId: 'nav-phrasebook',
      },
      {
        name: 'Verb Conjugations',
        href: '/verb-conjugation',
        icon: IconBook2,
        testId: 'nav-verb-conjugations',
      },
    ];

    const adminNav = isAdmin
      ? [
          {
            name: 'Admin',
            href: '/admin',
            icon: IconShieldCheck,
            testId: 'nav-admin',
          },
        ]
      : [];

    return { mainNav, adminNav };
  }, [isAdmin]);

  const { mainNav, adminNav } = navigation();

  // Helper function to map navigation index to shortcut key
  // Returns number string for indices 0-8 (1-9), letter string for indices 9+ (a, b, etc.)
  const getShortcutKey = (index: number): string => {
    if (index < 9) {
      return String(index + 1);
    }
    // Map 9 -> 'a', 10 -> 'b', etc.
    return String.fromCharCode(97 + (index - 9)); // 97 is 'a' in ASCII
  };

  // Navigation keyboard shortcuts (Shift + number/letter)
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
      if (mainNav[4] && mainNav[4].href !== location.pathname) {
        navigate(mainNav[4].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+6',
    e => {
      e.preventDefault();
      if (mainNav[5] && mainNav[5].href !== location.pathname) {
        navigate(mainNav[5].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+7',
    e => {
      e.preventDefault();
      if (mainNav[6] && mainNav[6].href !== location.pathname) {
        navigate(mainNav[6].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+8',
    e => {
      e.preventDefault();
      if (mainNav[7] && mainNav[7].href !== location.pathname) {
        navigate(mainNav[7].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+9',
    e => {
      e.preventDefault();
      if (mainNav[8] && mainNav[8].href !== location.pathname) {
        navigate(mainNav[8].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+a',
    e => {
      e.preventDefault();
      if (mainNav[9] && mainNav[9].href !== location.pathname) {
        navigate(mainNav[9].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+b',
    e => {
      e.preventDefault();
      if (mainNav[10] && mainNav[10].href !== location.pathname) {
        navigate(mainNav[10].href);
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  useHotkeys(
    'shift+0',
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
                <IconGlobe size={16} />
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
                    <IconChartLine
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
                      <IconPhone size={20} />
                    </ActionIcon>
                  </Tooltip>
                )}
                <Tooltip
                  label='Help and information about the system'
                  position='bottom'
                  withArrow
                >
                  <ActionIcon
                    onClick={() => setHelpModalOpened(true)}
                    variant='subtle'
                    size='lg'
                    aria-label='Help'
                  >
                    <IconHelp size={20} />
                  </ActionIcon>
                </Tooltip>
                <Tooltip
                  label='Report Issue or Give Feedback'
                  position='bottom'
                  withArrow
                >
                  <ActionIcon
                    onClick={() => setFeedbackModalOpened(true)}
                    variant='subtle'
                    size='lg'
                    aria-label='Feedback'
                    data-testid='feedback-button'
                  >
                    <IconAlertCircle size={20} />
                  </ActionIcon>
                </Tooltip>
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
                    <IconAdjustments
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
        <AppShell.Section grow style={{ overflow: 'auto' }}>
          <Stack gap='xs' mt='md'>
            {/* Main Navigation */}
            {mainNav.map((item, index) => (
              <React.Fragment key={item.name}>
                {item.name === 'Daily' && (
                  <Divider my='xs' label='Practice' labelPosition='center' />
                )}
                {item.name === 'Saved AI Conversations' && (
                  <Divider my='xs' label='History' labelPosition='center' />
                )}
                {item.name === 'Phrasebook' && (
                  <Divider my='xs' label='Reference' labelPosition='center' />
                )}
                <NavLink
                  component={Link}
                  to={item.href}
                  onClick={
                    item.name === 'Saved AI Conversations'
                      ? () => {
                          refreshAiConversations();
                        }
                      : undefined
                  }
                  label={
                    <Group
                      justify='space-between'
                      w='100%'
                      align='center'
                      gap='xs'
                    >
                      <Text size='sm' truncate style={{ flex: 1, minWidth: 0 }}>
                        {item.name}
                      </Text>
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
                            fontSize: `${11 * fontScaleMap[fontSize]}px`,
                            letterSpacing: '0.5px',
                            flexShrink: 0,
                          },
                        }}
                      >
                        ⇧{getShortcutKey(index)}
                      </Badge>
                    </Group>
                  }
                  leftSection={<item.icon size={20} />}
                  active={location.pathname.startsWith(item.href)}
                  data-testid={item.testId}
                  title={`${item.name} (Shift+${getShortcutKey(index)})`}
                />
              </React.Fragment>
            ))}

            {/* Admin Section */}
            {adminNav.length > 0 && (
              <>
                <Divider
                  my='md'
                  label='Administration'
                  labelPosition='center'
                />
                {adminNav.map(item => (
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
                              fontSize: `${11 * fontScaleMap[fontSize]}px`,
                              letterSpacing: '0.5px',
                            },
                          }}
                        >
                          ⇧0
                        </Badge>
                      </Group>
                    }
                    leftSection={<item.icon size={20} />}
                    active={location.pathname.startsWith(item.href)}
                    data-testid={item.testId}
                    title={`${item.name} (Shift+0)`}
                  />
                ))}
              </>
            )}
          </Stack>
        </AppShell.Section>
      </AppShell.Navbar>

      <AppShell.Main>{children}</AppShell.Main>
      <VersionDisplay />
      <HelpModal
        opened={helpModalOpened}
        onClose={() => setHelpModalOpened(false)}
      />
      <FeedbackModal
        opened={feedbackModalOpened}
        onClose={() => setFeedbackModalOpened(false)}
      />
    </AppShell>
  );
};

export default Layout;
