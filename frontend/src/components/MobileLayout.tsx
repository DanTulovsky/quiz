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
  IconLanguage,
  IconNotes,
  IconLanguageHiragana,
  IconSettings,
  IconBug,
  IconSparkles,
} from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';
import FeedbackModal from './FeedbackModal';
import { useMobileDetection } from '../hooks/useMobileDetection';
import { useTheme } from '../contexts/ThemeContext';
import { useNavigate, useLocation } from 'react-router-dom';
import VersionDisplay from './VersionDisplay';
import { TranslationOverlay } from './TranslationOverlay';

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
  const [
    feedbackModalOpened,
    { open: openFeedbackModal, close: closeFeedbackModal },
  ] = useDisclosure(false);

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
    if (path.startsWith('/m/translation-practice'))
      return 'translation-practice';
    if (path.startsWith('/m/conversations')) return 'conversations';
    if (path.startsWith('/m/bookmarks')) return 'bookmarks';
    if (path.startsWith('/m/snippets')) return 'snippets';
    if (path.startsWith('/m/phrasebook')) return 'phrasebook';
    if (path.startsWith('/m/verb-conjugation')) return 'verb-conjugation';
    if (path.startsWith('/m/settings')) return 'settings';
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
      key: 'word-of-day',
      label: 'Word of the Day',
      icon: IconSparkles,
      path: '/m/word-of-day',
    },
    {
      key: 'translation-practice',
      label: 'Translation Practice',
      icon: IconLanguage,
      path: '/m/translation-practice',
    },
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
    {
      key: 'snippets',
      label: 'Snippets',
      icon: IconNotes,
      path: '/m/snippets',
    },
    {
      key: 'phrasebook',
      label: 'Phrasebook',
      icon: IconLanguage,
      path: '/m/phrasebook',
    },
    {
      key: 'verb-conjugation',
      label: 'Verb Conjugations',
      icon: IconLanguageHiragana,
      path: '/m/verb-conjugation',
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
        p='sm'
        style={{ display: 'flex', flexDirection: 'column' }}
      >
        <AppShell.Section grow style={{ overflow: 'auto' }}>
          <Stack gap='xs'>
            <Text fw={500}>Menu</Text>
            {/* Primary navigation moved from footer */}
            {navItems.map(item => {
              const Icon = item.icon;
              return (
                <React.Fragment key={item.key}>
                  {item.key === 'daily' && (
                    <Divider my='xs' label='Practice' labelPosition='center' />
                  )}
                  {item.key === 'conversations' && (
                    <Divider
                      my='xs'
                      label='AI History'
                      labelPosition='center'
                    />
                  )}
                  {item.key === 'snippets' && (
                    <Divider my='xs' label='Reference' labelPosition='center' />
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
            <Divider my='xs' />
            <Button
              variant={activeTab === 'settings' ? 'light' : 'subtle'}
              leftSection={<IconSettings size={16} />}
              onClick={() => {
                handleNavigation('/m/settings');
                if (opened) toggle();
              }}
              fullWidth
              justify='flex-start'
            >
              Settings
            </Button>
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
            <Divider my='xs' />
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
              variant='subtle'
              onClick={openFeedbackModal}
              title='Feedback'
            >
              <IconBug size={18} />
            </ActionIcon>
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
      <TranslationOverlay />
      <FeedbackModal
        opened={feedbackModalOpened}
        onClose={closeFeedbackModal}
      />
    </AppShell>
  );
};

export default MobileLayout;
