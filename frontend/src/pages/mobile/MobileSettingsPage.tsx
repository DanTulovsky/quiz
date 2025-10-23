import { useAuth } from '../../hooks/useAuth';
import {
  useGetV1SettingsAiProviders,
  usePostV1SettingsTestAi,
  useGetV1SettingsApiKeyProvider,
  usePutV1UserzProfile,
  useGetV1SettingsLanguages,
  useGetV1SettingsLevels,
  useGetV1PreferencesLearning,
  usePutV1PreferencesLearning,
  usePostV1SettingsTestEmail,
  UserLearningPreferences,
  UserUpdateRequest,
} from '../../api/api';
import { Save, Lightbulb } from 'lucide-react';
import React, { useEffect, useState, useRef } from 'react';
import { showNotificationWithClean } from '../../notifications';
import ErrorModal from '../../components/ErrorModal';
import ConfirmationModal from '../../components/ConfirmationModal';
import { useQueryClient } from '@tanstack/react-query';
import { getGetV1PreferencesLearningQueryKey } from '../../api/api';
import TimezoneSelector from '../../components/TimezoneSelector';
import { useTheme } from '../../contexts/ThemeContext';
import {
  defaultVoiceForLanguage,
  extractVoiceName,
  languageToLocale,
  EdgeTTSVoiceInfo,
  sampleTextForLanguage,
} from '../../utils/tts';
import {
  Container,
  Title,
  Text,
  Card,
  Stack,
  TextInput,
  Select,
  Switch,
  Button,
  Group,
  Center,
  Loader,
  Alert,
  Box,
  NumberInput,
  Slider,
  Tooltip,
  SegmentedControl,
  Accordion,
} from '@mantine/core';
import {
  clearAllStories,
  resetAccount,
  clearAllAIChats,
  clearAllSnippets,
} from '../../api/settingsApi';
import {
  IconUser,
  IconBrain,
  IconTarget,
  IconPalette,
  IconBell,
  IconChevronDown,
} from '@tabler/icons-react';

// Add this type for the levels API response
interface LevelsApiResponse {
  levels: string[];
  level_descriptions: Record<string, string>;
}

interface ApiError {
  response?: {
    data?: {
      error?: string;
      details?: string;
    };
  };
}

const MobileSettingsPage: React.FC = () => {
  const { user, refreshUser } = useAuth();
  const {
    currentTheme,
    setTheme,
    themeNames,
    colorScheme,
    setColorScheme,
    fontSize,
    setFontSize,
  } = useTheme();
  const [language, setLanguage] = useState('');
  const [level, setLevel] = useState('');

  // Account information state
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [timezone, setTimezone] = useState('');

  const {
    data: providersData,
    isLoading: providersLoading,
    error: providersError,
  } = useGetV1SettingsAiProviders();
  const providers = providersData?.providers;

  const hasValidLanguage = Boolean(language && language.trim() !== '');
  const { data: levelsData, refetch: refetchLevels } =
    useGetV1SettingsLevels<LevelsApiResponse>(
      hasValidLanguage ? { language } : undefined,
      {
        query: {
          enabled: hasValidLanguage,
        },
      }
    );
  const levels = levelsData?.levels;
  const levelDescriptions = levelsData?.level_descriptions || {};

  const { data: languagesData } = useGetV1SettingsLanguages();
  const languages = languagesData;

  // Find the current language object for TTS configuration
  const currentLanguageObj = languages?.find(lang => lang.name === language);

  const [aiProvider, setAiProvider] = useState('');
  const [aiModel, setAiModel] = useState('');
  const [apiKey, setApiKey] = useState(''); // API key is write-only
  const [aiEnabled, setAiEnabled] = useState(false); // AI enable/disable toggle
  const [isInitialized, setIsInitialized] = useState(false); // Track if we've finished initializing from user data

  const [learningPrefs, setLearningPrefs] =
    useState<UserLearningPreferences | null>(null);
  const learningPrefsRef = useRef<UserLearningPreferences | null>(null);
  const [availableVoices, setAvailableVoices] = useState<string[]>([]);
  // Local UI state for TTS sample button
  const [ttsBufferingLocal, setTtsBufferingLocal] = useState(false);
  const [ttsBufferProgress, setTtsBufferProgress] = useState(0);
  const [ttsPlayingLocal, setTtsPlayingLocal] = useState(false);
  const queryClient = useQueryClient();
  const {
    data: fetchedPrefs,
    isLoading: prefsLoading,
    error: prefsError,
  } = useGetV1PreferencesLearning();
  const prefsMutation = usePutV1PreferencesLearning();

  // Error modal state
  const [errorModal, setErrorModal] = useState({
    isOpen: false,
    title: '',
    message: '',
  });

  // Confirmation modal states for dangerous actions
  const [deleteAllStoriesModal, setDeleteAllStoriesModal] = useState(false);
  const [deleteAllAiChatsModal, setDeleteAllAiChatsModal] = useState(false);
  const [deleteAllSnippetsModal, setDeleteAllSnippetsModal] = useState(false);
  const [resetAccountModal, setResetAccountModal] = useState(false);

  const testConnectionMutation = usePostV1SettingsTestAi();
  const profileUpdateMutation = usePutV1UserzProfile();
  const testEmailMutation = usePostV1SettingsTestEmail();

  // Check API key availability for the currently selected provider (no special case for ollama)
  const { data: apiKeyAvailable, refetch: refetchApiKeyAvailability } =
    useGetV1SettingsApiKeyProvider(aiProvider, {
      query: { enabled: !!aiProvider },
    });

  // Refetch API key availability when AI provider changes
  useEffect(() => {
    if (aiProvider && refetchApiKeyAvailability) {
      refetchApiKeyAvailability();
    }
  }, [aiProvider, refetchApiKeyAvailability]);

  // Refetch levels when language changes
  useEffect(() => {
    if (language && language.trim() !== '') {
      refetchLevels();
    }
    // It's intentional to not include learningPrefs as a dependency to avoid loops
  }, [language, learningPrefs]);

  useEffect(() => {
    if (fetchedPrefs) {
      setLearningPrefs(fetchedPrefs);
    }
  }, [fetchedPrefs]);

  useEffect(() => {
    learningPrefsRef.current = learningPrefs;
  }, [learningPrefs]);

  // Fetch available TTS voices for the selected language
  useEffect(() => {
    let isCancelled = false;
    const fetchVoices = async () => {
      try {
        const locale = languageToLocale(currentLanguageObj);
        if (!locale) {
          setAvailableVoices([]);
          return;
        }
        const res = await fetch(
          `/v1/voices?language=${encodeURIComponent(locale)}`
        );
        if (!res.ok) throw new Error('failed');
        const json: unknown = await res.json();
        const rawVoices: EdgeTTSVoiceInfo[] = Array.isArray(json)
          ? (json as EdgeTTSVoiceInfo[])
          : ((json as { voices?: EdgeTTSVoiceInfo[] })?.voices ?? []);
        const voices = (rawVoices || [])
          .map(extractVoiceName)
          .filter((v): v is string => !!v);
        if (!isCancelled) {
          // Only keep voices for the selected language; do not merge previous-language voice
          setAvailableVoices(voices);
        }
      } catch (error) {
        console.warn('Failed to fetch TTS voices:', error);
        setAvailableVoices([]);
      }
    };
    fetchVoices();
    return () => {
      isCancelled = true;
    };
  }, [language, languages]); // Also depend on languages being loaded

  // Ensure TTS voice is properly selected when learning preferences or voices change
  useEffect(() => {
    if (learningPrefs && availableVoices.length > 0) {
      const saved = (learningPrefs as unknown as { tts_voice?: string })
        .tts_voice;
      const preferred = defaultVoiceForLanguage(currentLanguageObj);
      const chosen =
        (saved && availableVoices.includes(saved) && saved) ||
        (preferred && availableVoices.includes(preferred) && preferred) ||
        availableVoices[0] ||
        '';
      if (chosen && chosen !== saved) {
        setLearningPrefs(prev =>
          prev
            ? ({ ...prev, tts_voice: chosen } as UserLearningPreferences)
            : prev
        );
      }
    }
  }, [learningPrefs, availableVoices, currentLanguageObj]);

  // Initialize state from user data
  useEffect(() => {
    if (
      user &&
      !isInitialized &&
      user.preferred_language !== undefined &&
      !language
    ) {
      // Account information
      setUsername(user.username || '');
      setEmail(user.email || '');
      setTimezone(user.timezone || '');

      // Learning settings
      // Only set language if it's not already set or if it's different from current
      const newLanguage = user.preferred_language || '';
      // Prevent setting language if it's already set to the same value to avoid race conditions
      // Also prevent setting if language is already set to the user's preferred language
      if (
        language !== newLanguage &&
        (!language || language === '') &&
        newLanguage !== '' &&
        language !== user.preferred_language
      ) {
        setLanguage(newLanguage);
      }

      // Set level from user.current_level - preserve user's actual level
      if (user.current_level) {
        setLevel(user.current_level);
      }

      // AI settings
      setAiProvider(user.ai_provider || '');
      setAiModel(user.ai_model || '');
      setAiEnabled(user.ai_enabled || false);

      // Auto-detect timezone if not set
      if (!user.timezone) {
        try {
          const detectedTimezone =
            Intl.DateTimeFormat().resolvedOptions().timeZone;
          setTimezone(detectedTimezone);
        } catch {
          setTimezone('UTC');
        }
      }

      setIsInitialized(true);
    }
  }, [user, isInitialized, language]);

  // When levels change, ensure level is always valid for the selected language
  useEffect(() => {
    // Only run validation if we have levels data and a level is set
    if (levels && levels.length > 0 && level) {
      // Only change level if current level is not valid for the selected language
      if (!levels.includes(level)) {
        // Try to find a similar level or default to first available
        const newLevel = levels[0];
        setLevel(newLevel);
      }
    }
    // Don't clear the level when levels are loading - this prevents race conditions
    // where the user's level gets cleared before levels are loaded
  }, [levels, level, language]);

  // When the AI provider changes, update the model to its default ONLY if switching providers
  useEffect(() => {
    if (providers && aiProvider) {
      const selectedProvider = providers.find(p => p.code === aiProvider);
      if (selectedProvider) {
        // Only update model if we don't have a valid model for this provider
        // This preserves user's saved model when the page loads
        if (!aiModel) {
          const firstModel = selectedProvider.models?.[0]?.code || '';
          setAiModel(firstModel);
        } else if (
          selectedProvider.models &&
          selectedProvider.models.length > 0
        ) {
          // Check if current model exists in this provider's models
          const currentModelExistsInProvider = selectedProvider.models.some(
            m => m.code === aiModel
          );

          // Only change model if current model is not valid for this provider
          // This happens when user switches from one provider to another
          if (!currentModelExistsInProvider) {
            const firstModel = selectedProvider.models[0]?.code || '';
            setAiModel(firstModel);
          }
        }
      }
    }
  }, [aiProvider, providers, aiModel, isInitialized]);

  const handlePrefsChange = (
    field: keyof UserLearningPreferences,
    value: string | number | boolean
  ) => {
    setLearningPrefs(prev => (prev ? { ...prev, [field]: value } : prev));
  };

  // Function to update all user settings using the unified profile endpoint
  const updateAllSettings = async (settingsData: UserUpdateRequest) => {
    try {
      const result = await profileUpdateMutation.mutateAsync({
        data: settingsData,
      });
      return result;
    } catch (error) {
      throw error;
    }
  };

  const handleTestConnection = async () => {
    // Validate required fields
    if (!aiProvider) {
      setErrorModal({
        isOpen: true,
        title: 'Test Connection Failed',
        message: 'Please select an AI provider before testing the connection.',
      });
      return;
    }

    // For all providers, we need either a new API key or a saved one (no ollama special case)
    if (!apiKey.trim() && !apiKeyAvailable?.has_api_key) {
      setErrorModal({
        isOpen: true,
        title: 'Test Connection Failed',
        message: 'Please enter an API key before testing the connection.',
      });
      return;
    }

    // For custom providers, validate URL
    const selectedProvider = providers?.find(p => p.code === aiProvider);
    if (!selectedProvider?.url) {
      setErrorModal({
        isOpen: true,
        title: 'Test Connection Failed',
        message: `No endpoint URL configured for provider '${aiProvider}'. Please check config.yaml.`,
      });
      return;
    }

    try {
      const response = await testConnectionMutation.mutateAsync({
        data: {
          provider: aiProvider,
          model: aiModel,
          api_key: apiKey || undefined,
        },
      });

      if (response.success) {
        showNotificationWithClean({
          title: 'Success',
          message: 'AI connection test successful!',
          color: 'green',
        });
      } else {
        setErrorModal({
          isOpen: true,
          title: 'Test Connection Failed',
          message: 'Connection test failed',
        });
      }
    } catch (error: unknown) {
      const message =
        (error as ApiError)?.response?.data?.error ||
        (error as ApiError)?.response?.data?.details ||
        'Test connection failed';
      setErrorModal({
        isOpen: true,
        title: 'Test Connection Failed',
        message: message,
      });
    }
  };

  const handleTestEmail = async () => {
    try {
      const response = await testEmailMutation.mutateAsync();

      if (response.success) {
        showNotificationWithClean({
          title: 'Success',
          message: 'Test email sent successfully!',
          color: 'green',
        });
      } else {
        setErrorModal({
          isOpen: true,
          title: 'Test Email Failed',
          message: 'Failed to send test email',
        });
      }
    } catch (error: unknown) {
      const message =
        (error as ApiError)?.response?.data?.error ||
        (error as ApiError)?.response?.data?.details ||
        'Test email failed';
      setErrorModal({
        isOpen: true,
        title: 'Test Email Failed',
        message: message,
      });
    }
  };

  // Handler functions for dangerous actions
  const handleDeleteAllStories = async () => {
    setDeleteAllStoriesModal(false);
    try {
      await clearAllStories();

      // Invalidate story queries to ensure UI updates immediately
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({
        queryKey: ['archivedStories', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({
        queryKey: ['userStories'],
      });

      showNotificationWithClean({
        title: 'Success',
        message: 'All stories deleted',
        color: 'green',
      });
      await refreshUser();
    } catch (e) {
      showNotificationWithClean({
        title: 'Error',
        message: String(e),
        color: 'red',
      });
    }
  };

  const handleDeleteAllAiChats = async () => {
    setDeleteAllAiChatsModal(false);
    try {
      await clearAllAIChats();

      // Invalidate AI conversation queries to ensure UI updates immediately
      queryClient.invalidateQueries({
        queryKey: ['aiConversations'],
      });
      queryClient.invalidateQueries({
        queryKey: ['aiConversations', user?.id],
      });

      showNotificationWithClean({
        title: 'Success',
        message: 'All AI chats deleted',
        color: 'green',
      });
      await refreshUser();
    } catch (e) {
      showNotificationWithClean({
        title: 'Error',
        message: String(e),
        color: 'red',
      });
    }
  };

  const handleDeleteAllSnippets = async () => {
    setDeleteAllSnippetsModal(false);
    try {
      await clearAllSnippets();

      // Invalidate snippet queries to ensure UI updates immediately
      // The SnippetsPage uses these query keys with pagination
      queryClient.invalidateQueries({
        queryKey: ['/v1/snippets'],
      });
      queryClient.invalidateQueries({
        queryKey: ['/v1/snippets/search'],
      });

      showNotificationWithClean({
        title: 'Success',
        message: 'All snippets deleted',
        color: 'green',
      });
      await refreshUser();
    } catch (e) {
      showNotificationWithClean({
        title: 'Error',
        message: String(e),
        color: 'red',
      });
    }
  };

  const handleResetAccount = async () => {
    setResetAccountModal(false);
    try {
      await resetAccount();

      // Invalidate story queries to ensure UI updates immediately
      queryClient.invalidateQueries({
        queryKey: ['currentStory', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({
        queryKey: ['archivedStories', user?.id, user?.preferred_language],
      });
      queryClient.invalidateQueries({
        queryKey: ['userStories'],
      });

      showNotificationWithClean({
        title: 'Success',
        message: 'Account reset',
        color: 'green',
      });
      await refreshUser();
    } catch (e) {
      showNotificationWithClean({
        title: 'Error',
        message: String(e),
        color: 'red',
      });
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    let errorMsg = '';
    try {
      // Save general settings
      await updateAllSettings({
        username,
        email,
        timezone,
        preferred_language: language,
        current_level: level,
        ai_enabled: aiEnabled,
        ai_provider: aiProvider,
        ai_model: aiModel,
        api_key: apiKey || undefined,
      });
      // Save learning preferences if loaded
      if (learningPrefs) {
        await prefsMutation.mutateAsync({ data: learningPrefs });
        // Immediately update cache so other pages see the new voice without reload
        queryClient.setQueryData(
          getGetV1PreferencesLearningQueryKey(),
          learningPrefs
        );
        // Trigger background refetch to sync from server
        queryClient.invalidateQueries({
          queryKey: getGetV1PreferencesLearningQueryKey(),
        });
      }
      await refreshUser();
      if (aiProvider && refetchApiKeyAvailability) {
        await refetchApiKeyAvailability();
      }
      showNotificationWithClean({
        title: 'Success',
        message: 'Settings saved successfully',
        color: 'green',
        autoClose: 2000,
      });
    } catch (error: unknown) {
      errorMsg =
        (error as { error?: string; message?: string })?.error ||
        (error as { error?: string; message?: string })?.message ||
        'Failed to save settings';
      showNotificationWithClean({
        title: 'Error',
        message: errorMsg,
        color: 'red',
      });
    }
  };

  const getProviderUrl = (providerCode: string) => {
    const selectedProvider = providers?.find(p => p.code === providerCode);
    return selectedProvider?.url || '';
  };

  // Show loading state while data is being fetched
  if (providersLoading) {
    return (
      <Container size='sm' py='md'>
        <Title order={2} mb='md'>
          Settings
        </Title>
        <Center py='xl'>
          <Loader size='lg' />
        </Center>
      </Container>
    );
  }

  // Show error state if API call failed
  if (providersError) {
    return (
      <Container size='sm' py='md'>
        <Title order={2} mb='md'>
          Settings
        </Title>
        <Alert color='error' title='Error' variant='light'>
          Error loading settings: {String(providersError)}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size='sm' py='md' pb='xl'>
      <style>{`\n        @keyframes tts-pulse {\n          0% { transform: scale(1); }\n          50% { transform: scale(1.08); }\n          100% { transform: scale(1); }\n        }\n        .tts-playing-button {\n          animation: tts-pulse 1s infinite ease-in-out;\n        }\n      `}</style>
      <Title order={2} mb='md'>
        Settings
      </Title>
      <form onSubmit={handleSubmit}>
        <Stack gap='md'>
          {/* Using Accordion for collapsible sections on mobile */}
          <Accordion variant='separated' radius='md'>
            {/* Theme Section */}
            <Accordion.Item value='theme'>
              <Accordion.Control icon={<IconPalette size={20} />}>
                <Text fw={500}>Theme</Text>
              </Accordion.Control>
              <Accordion.Panel>
                <Stack gap='md'>
                  <Text size='sm' c='dimmed'>
                    Choose your preferred color theme and mode.
                  </Text>
                  <Switch
                    checked={colorScheme === 'dark'}
                    onChange={e =>
                      setColorScheme(e.currentTarget.checked ? 'dark' : 'light')
                    }
                    label={colorScheme === 'dark' ? 'Dark mode' : 'Light mode'}
                    size='md'
                  />

                  <Box>
                    <Text size='sm' fw={500} mb='xs'>
                      Font Size
                    </Text>
                    <SegmentedControl
                      value={fontSize}
                      onChange={value => setFontSize(value as typeof fontSize)}
                      data={[
                        { label: 'S', value: 'small' },
                        { label: 'M', value: 'medium' },
                        { label: 'L', value: 'large' },
                        { label: 'XL', value: 'extra-large' },
                      ]}
                      fullWidth
                      data-testid='font-size-control'
                    />
                  </Box>

                  <Box>
                    <Text size='sm' fw={500} mb='xs'>
                      Color Theme
                    </Text>
                    <Select
                      value={currentTheme}
                      onChange={value =>
                        value && setTheme(value as keyof typeof themeNames)
                      }
                      data={Object.entries(themeNames).map(
                        ([key, name]) => ({
                          value: key,
                          label: name,
                        })
                      )}
                      data-testid='theme-select'
                    />
                  </Box>
                </Stack>
              </Accordion.Panel>
            </Accordion.Item>

            {/* Account Information */}
            <Accordion.Item value='account'>
              <Accordion.Control icon={<IconUser size={20} />}>
                <Text fw={500}>Account Information</Text>
              </Accordion.Control>
              <Accordion.Panel>
                <Stack gap='md'>
                  <TextInput
                    label='Username'
                    value={username}
                    onChange={e => setUsername(e.target.value)}
                    placeholder='Enter your username'
                    required
                  />

                  <TextInput
                    label='Email'
                    type='email'
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                    placeholder='Enter your email address'
                    required
                  />

                  <TimezoneSelector
                    value={timezone}
                    onChange={setTimezone}
                    placeholder='Select your timezone...'
                  />
                  <Text size='xs' c='dimmed'>
                    Your timezone is used to display times correctly.
                  </Text>
                </Stack>
              </Accordion.Panel>
            </Accordion.Item>

            {/* Learning Settings */}
            <Accordion.Item value='learning'>
              <Accordion.Control icon={<IconTarget size={20} />}>
                <Text fw={500}>Learning Preferences</Text>
              </Accordion.Control>
              <Accordion.Panel>
                <Stack gap='md'>
                  <Select
                    label='Learning Language'
                    value={language}
                    onChange={value => setLanguage(value || '')}
                    data={
                      languages?.map(lang => ({
                        value: lang.name || lang,
                        label: lang.name
                          ? lang.name.charAt(0).toUpperCase() +
                            lang.name.slice(1)
                          : (lang.name || lang).charAt(0).toUpperCase() +
                            (lang.name || lang).slice(1),
                      })) || []
                    }
                    placeholder='Select language'
                    data-testid='learning-language-select'
                  />

                  <Select
                    label='Current Level'
                    value={level}
                    onChange={value => setLevel(value || '')}
                    data={
                      levels
                        ?.filter((l: string) => l)
                        .map((l: string) => ({
                          value: l,
                          label: `${l} — ${levelDescriptions[l] || ''}`.trim(),
                        })) || []
                    }
                    placeholder='Select level'
                    data-testid='level-select'
                  />

                  {prefsLoading ? (
                    <Center>
                      <Loader size='sm' />
                    </Center>
                  ) : prefsError ? (
                    <Alert color='red'>
                      Failed to load learning preferences.
                    </Alert>
                  ) : (
                    learningPrefs && (
                      <>
                        <Group align='center'>
                          <Select
                            label='TTS Voice'
                            placeholder={
                              availableVoices.length
                                ? 'Select voice'
                                : 'No voices available'
                            }
                            data={availableVoices.map(v => ({
                              value: v,
                              label: v,
                            }))}
                            value={
                              (
                                learningPrefs as unknown as {
                                  tts_voice?: string;
                                }
                              )?.tts_voice || ''
                            }
                            onChange={v =>
                              handlePrefsChange(
                                'tts_voice' as keyof UserLearningPreferences,
                                v || ''
                              )
                            }
                            searchable
                            nothingFoundMessage='No voices'
                            disabled={!availableVoices.length}
                            data-testid='tts-voice-select'
                            style={{ flex: 1 }}
                          />
                          <Tooltip
                            label='Play sample'
                            position='top'
                          >
                            <Button
                              variant='subtle'
                              size='xs'
                              className={
                                ttsPlayingLocal
                                  ? 'tts-playing-button'
                                  : undefined
                              }
                              onClick={async () => {
                                try {
                                  const sample =
                                    sampleTextForLanguage(currentLanguageObj);
                                  if (!sample) {
                                    showNotificationWithClean({
                                      title: 'No sample',
                                      message:
                                        'No sample text available for this language',
                                      color: 'yellow',
                                    });
                                    return;
                                  }
                                  const chosenVoice =
                                    (
                                      learningPrefs as unknown as {
                                        tts_voice?: string;
                                      }
                                    )?.tts_voice ||
                                    defaultVoiceForLanguage(
                                      currentLanguageObj
                                    ) ||
                                    'echo';

                                  // Set local buffering indicator
                                  setTtsBufferingLocal(true);
                                  setTtsBufferProgress(0);

                                  const { playTTSOnce, stopTTSOnce } =
                                    await import('../../hooks/useTTS');

                                  // If already playing, stop instead of starting another
                                  if (ttsPlayingLocal) {
                                    stopTTSOnce();
                                    setTtsPlayingLocal(false);
                                    setTtsBufferingLocal(false);
                                    setTtsBufferProgress(0);
                                    return;
                                  }

                                  await playTTSOnce(sample, chosenVoice, {
                                    onBuffering: (p: number) => {
                                      setTtsBufferProgress(p);
                                    },
                                    onPlayStart: () => {
                                      setTtsBufferingLocal(false);
                                      setTtsPlayingLocal(true);
                                    },
                                    onPlayEnd: () => {
                                      setTtsPlayingLocal(false);
                                    },
                                  });
                                } catch (e) {
                                  showNotificationWithClean({
                                    title: 'Playback failed',
                                    message: String(e),
                                    color: 'red',
                                  });
                                } finally {
                                  setTtsBufferingLocal(false);
                                  setTtsBufferProgress(0);
                                }
                              }}
                              data-testid='tts-sample-button'
                              aria-label='Play voice sample'
                              mt='xl'
                            >
                              {ttsBufferingLocal ? (
                                <Loader size='xs' />
                              ) : ttsPlayingLocal ? (
                                '⏸'
                              ) : (
                                '🔊'
                              )}
                            </Button>
                          </Tooltip>
                        </Group>

                        <Group>
                          <Tooltip label="See more questions from topics you struggle with">
                            <Lightbulb size={16} />
                          </Tooltip>
                          <Switch
                            label='Focus on weak areas'
                            checked={learningPrefs.focus_on_weak_areas}
                            onChange={e =>
                              handlePrefsChange(
                                'focus_on_weak_areas',
                                e.currentTarget.checked
                              )
                            }
                            data-testid='focus-weak-areas-switch'
                          />
                        </Group>

                        <Stack gap='sm'>
                          <Group gap={4} align='center'>
                            <Tooltip label='What percent of your questions should be brand new?'>
                              <Lightbulb size={16} />
                            </Tooltip>
                            <Text size='sm' fw={500}>
                              Fresh question ratio
                            </Text>
                          </Group>
                          <Slider
                            min={0}
                            max={1}
                            step={0.05}
                            value={learningPrefs.fresh_question_ratio}
                            onChange={v =>
                              handlePrefsChange('fresh_question_ratio', v)
                            }
                            marks={[
                              { value: 0, label: '0%' },
                              { value: 0.5, label: '50%' },
                              { value: 1, label: '100%' },
                            ]}
                            data-testid='fresh-question-ratio-slider'
                          />
                        </Stack>

                        <Stack gap='sm'>
                          <Group gap={4} align='center'>
                            <Tooltip label="Deprioritize questions you've marked as known">
                              <Lightbulb size={16} />
                            </Tooltip>
                            <Text size='sm' fw={500}>
                              Known question penalty
                            </Text>
                          </Group>
                          <Slider
                            min={0}
                            max={1}
                            step={0.01}
                            value={learningPrefs.known_question_penalty}
                            onChange={v =>
                              handlePrefsChange('known_question_penalty', v)
                            }
                            marks={[
                              { value: 0, label: '0' },
                              { value: 0.5, label: '0.5' },
                              { value: 1, label: '1' },
                            ]}
                            data-testid='known-question-penalty-slider'
                          />
                        </Stack>

                        <Stack gap='sm'>
                          <Group gap={4} align='center'>
                            <Tooltip label='Boost priority for weak area questions'>
                              <Lightbulb size={16} />
                            </Tooltip>
                            <Text size='sm' fw={500}>
                              Weak area boost
                            </Text>
                          </Group>
                          <Slider
                            min={1}
                            max={5}
                            step={0.1}
                            value={learningPrefs.weak_area_boost}
                            onChange={v =>
                              handlePrefsChange('weak_area_boost', v)
                            }
                            marks={[
                              { value: 1, label: '1x' },
                              { value: 3, label: '3x' },
                              { value: 5, label: '5x' },
                            ]}
                            data-testid='weak-area-boost-slider'
                          />
                        </Stack>

                        <Group grow>
                          <Box>
                            <Text size='sm' fw={500} mb='xs'>
                              Review interval (days)
                            </Text>
                            <NumberInput
                              min={1}
                              max={60}
                              value={learningPrefs.review_interval_days}
                              onChange={v =>
                                handlePrefsChange('review_interval_days', v)
                              }
                              data-testid='review-interval-days-input'
                            />
                          </Box>

                          <Box>
                            <Text size='sm' fw={500} mb='xs'>
                              Daily goal
                            </Text>
                            <NumberInput
                              min={1}
                              max={50}
                              value={learningPrefs.daily_goal || 10}
                              onChange={v =>
                                handlePrefsChange('daily_goal', v || 10)
                              }
                              data-testid='daily-goal-input'
                            />
                          </Box>
                        </Group>
                      </>
                    )
                  )}
                </Stack>
              </Accordion.Panel>
            </Accordion.Item>

            {/* Notifications */}
            <Accordion.Item value='notifications'>
              <Accordion.Control icon={<IconBell size={20} />}>
                <Text fw={500}>Notifications</Text>
              </Accordion.Control>
              <Accordion.Panel>
                <Stack gap='md'>
                  <Group justify='space-between' align='flex-start'>
                    <Box style={{ flex: 1 }}>
                      <Text fw={500} mb='xs'>
                        Daily Email Reminders
                      </Text>
                      <Text size='sm' c='dimmed'>
                        Stay on track with your learning goals.
                      </Text>
                    </Box>
                    <Switch
                      checked={learningPrefs?.daily_reminder_enabled || false}
                      onChange={e =>
                        handlePrefsChange(
                          'daily_reminder_enabled',
                          e.currentTarget.checked
                        )
                      }
                      size='lg'
                      data-testid='daily-reminder-switch'
                    />
                  </Group>

                  {learningPrefs?.daily_reminder_enabled && (
                    <Button
                      variant='outline'
                      onClick={handleTestEmail}
                      loading={testEmailMutation.isPending}
                      data-testid='test-email-button'
                      fullWidth
                    >
                      Test Email
                    </Button>
                  )}
                </Stack>
              </Accordion.Panel>
            </Accordion.Item>

            {/* AI Settings */}
            <Accordion.Item value='ai'>
              <Accordion.Control icon={<IconBrain size={20} />}>
                <Text fw={500}>AI Settings</Text>
              </Accordion.Control>
              <Accordion.Panel>
                <Stack gap='md'>
                  <Group justify='space-between' align='flex-start'>
                    <Box style={{ flex: 1 }}>
                      <Text fw={500} mb='xs'>
                        Enable AI Features
                      </Text>
                      <Text size='sm' c='dimmed'>
                        AI-powered question generation and features.
                      </Text>
                    </Box>
                    <Switch
                      checked={aiEnabled}
                      onChange={event =>
                        setAiEnabled(event.currentTarget.checked)
                      }
                      size='lg'
                      data-testid='ai-enabled-switch'
                    />
                  </Group>

                  {aiEnabled && (
                    <>
                      <Select
                        label='AI Provider'
                        value={aiProvider}
                        onChange={value => {
                          if (value && value !== aiProvider) {
                            setAiProvider(value);
                            setApiKey('');
                            if (refetchApiKeyAvailability) {
                              refetchApiKeyAvailability();
                            }
                          }
                        }}
                        data={
                          providers
                            ?.filter(p => p.name && p.code)
                            .map(p => ({ value: p.code!, label: p.name! })) ||
                          []
                        }
                        placeholder='Select a provider'
                        data-testid='ai-provider-select'
                      />

                      <Select
                        label='Model'
                        value={aiModel}
                        onChange={value => setAiModel(value || '')}
                        data={
                          providers
                            ?.find(p => p.code === aiProvider)
                            ?.models?.filter(m => m.name && m.code)
                            .map(m => ({ value: m.code!, label: m.name! })) ||
                          []
                        }
                        placeholder='Select a model'
                        disabled={!aiProvider}
                        data-testid='ai-model-select'
                      />

                      {aiProvider && (
                        <TextInput
                          label='Endpoint URL'
                          value={getProviderUrl(aiProvider)}
                          readOnly
                          styles={{
                            input: {
                              backgroundColor: 'var(--mantine-color-body)',
                            },
                          }}
                        />
                      )}

                      {aiProvider && (
                        <>
                          <TextInput
                            label={
                              <Group gap='xs'>
                                <Text>API Key</Text>
                                {apiKeyAvailable?.has_api_key && (
                                  <Text size='xs' c='success' fw={400}>
                                    (Saved)
                                  </Text>
                                )}
                              </Group>
                            }
                            type='password'
                            value={apiKey}
                            onChange={e => setApiKey(e.target.value)}
                            placeholder={
                              apiKeyAvailable?.has_api_key
                                ? 'Leave empty to use saved key'
                                : 'Enter your API key'
                            }
                          />
                          <Button
                            variant='outline'
                            onClick={handleTestConnection}
                            fullWidth
                          >
                            Test Connection
                          </Button>
                        </>
                      )}
                    </>
                  )}
                </Stack>
              </Accordion.Panel>
            </Accordion.Item>

            {/* Data Management */}
            <Accordion.Item value='data'>
              <Accordion.Control>
                <Text fw={500} c='red'>
                  Data Management
                </Text>
              </Accordion.Control>
              <Accordion.Panel>
                <Stack gap='md'>
                  <Text size='sm' c='dimmed'>
                    These actions cannot be undone.
                  </Text>
                  <Button
                    color='red'
                    variant='outline'
                    onClick={() => setDeleteAllStoriesModal(true)}
                    fullWidth
                  >
                    Delete All Stories
                  </Button>
                  <Button
                    color='red'
                    variant='outline'
                    onClick={() => setDeleteAllAiChatsModal(true)}
                    fullWidth
                  >
                    Delete All AI Chats
                  </Button>
                  <Button
                    color='red'
                    variant='outline'
                    onClick={() => setDeleteAllSnippetsModal(true)}
                    fullWidth
                  >
                    Delete All Snippets
                  </Button>
                  <Button
                    color='red'
                    onClick={() => setResetAccountModal(true)}
                    fullWidth
                  >
                    Reset Account
                  </Button>
                </Stack>
              </Accordion.Panel>
            </Accordion.Item>
          </Accordion>

          {/* Save Button */}
          <Button
            type='submit'
            leftSection={<Save size={16} />}
            size='md'
            fullWidth
          >
            Save Changes
          </Button>
        </Stack>
      </form>

      {/* Error Modal */}
      <ErrorModal
        isOpen={errorModal.isOpen}
        onClose={() => setErrorModal({ ...errorModal, isOpen: false })}
        title={errorModal.title}
        message={errorModal.message}
      />

      {/* Confirmation Modals for Dangerous Actions */}
      <ConfirmationModal
        isOpen={deleteAllStoriesModal}
        onClose={() => setDeleteAllStoriesModal(false)}
        onConfirm={handleDeleteAllStories}
        title='Delete All Stories'
        message='Are you sure you want to delete ALL your stories? This cannot be undone.'
        confirmText='Delete All Stories'
        cancelText='Cancel'
      />

      <ConfirmationModal
        isOpen={deleteAllAiChatsModal}
        onClose={() => setDeleteAllAiChatsModal(false)}
        onConfirm={handleDeleteAllAiChats}
        title='Delete All AI Chats'
        message='Are you sure you want to delete ALL your AI chats? This cannot be undone.'
        confirmText='Delete All AI Chats'
        cancelText='Cancel'
      />

      <ConfirmationModal
        isOpen={deleteAllSnippetsModal}
        onClose={() => setDeleteAllSnippetsModal(false)}
        onConfirm={handleDeleteAllSnippets}
        title='Delete All Snippets'
        message='Are you sure you want to delete ALL your snippets? This cannot be undone.'
        confirmText='Delete All Snippets'
        cancelText='Cancel'
      />

      <ConfirmationModal
        isOpen={resetAccountModal}
        onClose={() => setResetAccountModal(false)}
        onConfirm={handleResetAccount}
        title='Reset Account'
        message='Reset your account? This will delete your stories, questions, and progress. This cannot be undone.'
        confirmText='Reset Account'
        cancelText='Cancel'
      />
    </Container>
  );
};

export default MobileSettingsPage;
