import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  usePostV1AuthSignup,
  useGetV1SettingsLanguages,
  useGetV1SettingsLevels,
  useGetV1AuthSignupStatus,
} from '../api/api';
import { UserCreateRequest } from '../api/api';
import {
  Container,
  Paper,
  TextInput,
  PasswordInput,
  Button,
  Title,
  Text,
  Stack,
  Anchor,
  Progress,
  Select,
  Grid,
  Divider,
  Alert,
} from '@mantine/core';
import { showNotificationWithClean } from '../notifications';
import { IconX, IconLock } from '@tabler/icons-react';
import GoogleOAuthButton from '../components/GoogleOAuthButton';

interface ErrorWithResponse {
  response?: {
    data?: {
      error?: string;
    };
  };
  message?: string;
}

function isErrorWithResponse(error: unknown): error is ErrorWithResponse {
  return (
    typeof error === 'object' &&
    error !== null &&
    ('response' in error || 'message' in error)
  );
}

const SignupPage: React.FC = () => {
  const navigate = useNavigate();
  const signupMutation = usePostV1AuthSignup();
  const languagesQuery = useGetV1SettingsLanguages();
  const signupStatusQuery = useGetV1AuthSignupStatus();

  const [formData, setFormData] = useState<UserCreateRequest>({
    username: '',
    password: '',
    email: '',
    preferred_language: '',
    current_level: '',
  });

  const levelsQuery = useGetV1SettingsLevels(
    formData.preferred_language && formData.preferred_language.trim() !== ''
      ? { language: formData.preferred_language }
      : undefined
  );
  const [confirmPassword, setConfirmPassword] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const usernameInputRef = useRef<HTMLInputElement>(null);

  // The levels query will automatically refetch when the language parameter changes
  // No need for manual refetch since the query key includes the language

  // Ensure the selected level is always valid for the selected language
  useEffect(() => {
    const levels = levelsQuery.data?.levels;
    if (levels && levels.length > 0) {
      if (!formData.current_level || !levels.includes(formData.current_level)) {
        setFormData(prev => ({ ...prev, current_level: levels[0] }));
      }
    } else {
      // If there are no levels, clear the level
      if (formData.current_level) {
        setFormData(prev => ({ ...prev, current_level: '' }));
      }
    }
  }, [
    levelsQuery.data?.levels,
    formData.current_level,
    formData.preferred_language,
  ]);

  // Focus the username input on component mount
  useEffect(() => {
    if (usernameInputRef.current) {
      usernameInputRef.current.focus();
    }
  }, []);

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    // Username validation
    if (!formData.username) {
      newErrors.username = 'Username is required';
    } else if (formData.username.length < 3) {
      newErrors.username = 'Username must be at least 3 characters';
    } else if (formData.username.length > 50) {
      newErrors.username = 'Username must be less than 50 characters';
    } else if (!/^[a-zA-Z0-9_]+$/.test(formData.username)) {
      newErrors.username =
        'Username can only contain letters, numbers, and underscores';
    }

    // Email validation
    if (!formData.email) {
      newErrors.email = 'Email is required';
    } else if (
      !/^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(formData.email)
    ) {
      newErrors.email = 'Please enter a valid email address';
    }

    // Password validation
    if (!formData.password) {
      newErrors.password = 'Password is required';
    } else if (formData.password.length < 8) {
      newErrors.password = 'Password must be at least 8 characters';
    }

    // Confirm password validation
    if (!confirmPassword) {
      newErrors.confirmPassword = 'Please confirm your password';
    } else if (confirmPassword !== formData.password) {
      newErrors.confirmPassword = 'Passwords do not match';
    }

    // Language validation
    if (!formData.preferred_language) {
      newErrors.preferred_language = 'Please select a learning language';
    }

    // Level validation
    if (!formData.current_level) {
      newErrors.current_level = 'Please select your current level';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const getPasswordStrength = (
    password: string
  ): { strength: number; label: string; color: string } => {
    if (password.length === 0) return { strength: 0, label: '', color: 'gray' };
    if (password.length < 8)
      return { strength: 25, label: 'Weak', color: 'error' };

    let score = 0;
    if (password.length >= 8) score++;
    if (/[A-Z]/.test(password)) score++;
    if (/[a-z]/.test(password)) score++;
    if (/[0-9]/.test(password)) score++;
    if (/[^A-Za-z0-9]/.test(password)) score++;

    if (score <= 2) return { strength: 40, label: 'Weak', color: 'error' };
    if (score === 3) return { strength: 70, label: 'Medium', color: 'yellow' };
    return { strength: 100, label: 'Strong', color: 'success' };
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    try {
      await signupMutation.mutateAsync({ data: formData });
      // Success - redirect to login page with success message
      navigate('/login?message=account_created');
    } catch (error: unknown) {
      let errorMsg = 'An error occurred during signup';

      if (isErrorWithResponse(error) && error.response?.data?.error) {
        errorMsg = error.response.data.error;
      } else if (isErrorWithResponse(error) && error.message) {
        errorMsg = error.message;
      }

      showNotificationWithClean({
        title: 'Signup Error',
        message: errorMsg,
        color: 'error',
        icon: <IconX size={16} />,
      });
    }
  };

  const handleInputChange = (field: keyof UserCreateRequest, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    // Clear field error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  const passwordStrength = getPasswordStrength(formData.password);

  // Show disabled message if signups are disabled
  if (
    !signupStatusQuery.isLoading &&
    signupStatusQuery.data?.signups_disabled
  ) {
    return (
      <Container
        size='xs'
        h='100vh'
        style={{ display: 'flex', alignItems: 'center' }}
      >
        <Paper shadow='xl' p='xl' radius='lg' w='100%'>
          <Stack align='center' gap='lg'>
            <Alert
              color='red'
              title='Signups Disabled'
              icon={<IconLock size={16} />}
              w='100%'
            >
              User registration is currently disabled. Please contact an
              administrator if you need access.
            </Alert>
            <Text size='sm' c='dimmed' ta='center'>
              <Anchor
                component='button'
                onClick={() => navigate('/login')}
                fw={500}
              >
                Return to login
              </Anchor>
            </Text>
          </Stack>
        </Paper>
      </Container>
    );
  }

  return (
    <Container
      size='xs'
      h='100vh'
      style={{ display: 'flex', alignItems: 'center' }}
    >
      <Paper shadow='xl' p='xl' radius='lg' w='100%'>
        <Stack align='center' gap='lg'>
          {/* Header */}
          <Stack align='center' gap='sm'>
            <Title order={2} ta='center' fw={700}>
              Create your account
            </Title>
            <Text size='sm' c='dimmed' ta='center'>
              Or{' '}
              <Anchor
                component='button'
                onClick={() => navigate('/login')}
                fw={500}
              >
                sign in to your existing account
              </Anchor>
            </Text>
          </Stack>

          {/* Signup Form */}
          <form onSubmit={handleSubmit} style={{ width: '100%' }}>
            <Stack gap='md'>
              <TextInput
                label='Username'
                placeholder='Enter username'
                required
                value={formData.username}
                onChange={e => handleInputChange('username', e.target.value)}
                error={errors.username}
                autoFocus
              />

              <TextInput
                label='Email address'
                placeholder='Enter email address'
                type='email'
                required
                value={formData.email}
                onChange={e => handleInputChange('email', e.target.value)}
                error={errors.email}
              />

              <Grid gutter='md'>
                <Grid.Col span={{ base: 12, md: 6 }}>
                  <Select
                    label='Learning Language'
                    value={formData.preferred_language}
                    onChange={value =>
                      handleInputChange('preferred_language', value || '')
                    }
                    data={
                      Array.isArray(languagesQuery.data)
                        ? languagesQuery.data.map(lang => ({
                            value: lang,
                            label: lang.charAt(0).toUpperCase() + lang.slice(1),
                          }))
                        : []
                    }
                    placeholder='Select language'
                    required
                    error={errors.preferred_language}
                  />
                </Grid.Col>

                <Grid.Col span={{ base: 12, md: 6 }}>
                  <Select
                    label='Current Level'
                    value={formData.current_level}
                    onChange={value =>
                      handleInputChange('current_level', value || '')
                    }
                    data={
                      Array.isArray(levelsQuery.data?.levels) &&
                      levelsQuery.data?.level_descriptions
                        ? levelsQuery.data.levels.map(level => ({
                            value: level,
                            label:
                              levelsQuery.data.level_descriptions?.[level] ??
                              level,
                          }))
                        : []
                    }
                    placeholder='Select level'
                    required
                    error={errors.current_level}
                    data-testid='level-select'
                  />
                </Grid.Col>
              </Grid>

              <Stack gap='xs'>
                <PasswordInput
                  label='Password'
                  placeholder='Enter password'
                  required
                  value={formData.password}
                  onChange={e => handleInputChange('password', e.target.value)}
                  error={errors.password}
                />
                {formData.password && (
                  <Stack gap='xs'>
                    <Progress
                      value={passwordStrength.strength}
                      color={passwordStrength.color}
                      size='xs'
                    />
                    <Text size='xs' c={passwordStrength.color}>
                      Password strength: {passwordStrength.label}
                    </Text>
                  </Stack>
                )}
              </Stack>

              <PasswordInput
                label='Confirm Password'
                placeholder='Confirm password'
                required
                value={confirmPassword}
                onChange={e => {
                  setConfirmPassword(e.target.value);
                  if (errors.confirmPassword) {
                    setErrors(prev => ({ ...prev, confirmPassword: '' }));
                  }
                }}
                error={errors.confirmPassword}
              />

              <Button
                type='submit'
                fullWidth
                size='md'
                loading={signupMutation.isPending}
              >
                Create Account
              </Button>
            </Stack>
          </form>

          <Divider
            my='md'
            label='or'
            labelPosition='center'
            data-testid='oauth-divider'
          />

          <GoogleOAuthButton variant='signup' />
        </Stack>
      </Paper>
    </Container>
  );
};

export default SignupPage;
