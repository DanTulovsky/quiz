import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import {
  Container,
  Paper,
  TextInput,
  PasswordInput,
  Button,
  Title,
  Text,
  Stack,
  Alert,
  Anchor,
  ThemeIcon,
  Divider,
} from '@mantine/core';
import { IconBrain, IconCheck } from '@tabler/icons-react';
import GoogleOAuthButton from '../components/GoogleOAuthButton';
import { useGetV1AuthSignupStatus } from '../api/api';

const LoginPage: React.FC = () => {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [formData, setFormData] = useState({
    username: '',
    password: '',
  });
  const [isLoading, setIsLoading] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');
  const usernameInputRef = useRef<HTMLInputElement>(null);

  // Get the redirect URL from query parameters
  const redirectUrl = searchParams.get('redirect') || '/';

  // Check signup status
  const signupStatusQuery = useGetV1AuthSignupStatus();

  // Check for success message from signup
  useEffect(() => {
    const message = searchParams.get('message');
    if (message === 'account_created') {
      setSuccessMessage(
        'Account created successfully! Please log in with your credentials.'
      );
      // Clear the URL parameter after showing the message
      const newURL = new URL(window.location.href);
      newURL.searchParams.delete('message');
      window.history.replaceState({}, '', newURL.toString());
    }
  }, [searchParams]);

  // Focus the username input on component mount
  useEffect(() => {
    if (usernameInputRef.current) {
      usernameInputRef.current.focus();
    }
  }, []);

  const handleInputChange = (field: string, value: string) => {
    setFormData({
      ...formData,
      [field]: value,
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    try {
      const success = await login(formData.username, formData.password);
      if (success) {
        navigate('/');
      }
    } finally {
      setIsLoading(false);
    }
  };

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
            <ThemeIcon size='xl' radius='xl' variant='light'>
              <IconBrain size={32} />
            </ThemeIcon>
            <Title order={2} ta='center' fw={700}>
              AI Language Quiz
            </Title>
            <Text size='sm' c='dimmed' ta='center'>
              Sign in to start your personalized learning journey
            </Text>
          </Stack>

          {/* Success Message */}
          {successMessage && (
            <Alert
              color='success'
              title='Success!'
              icon={<IconCheck size={16} />}
              withCloseButton
              onClose={() => setSuccessMessage('')}
              w='100%'
            >
              {successMessage}
            </Alert>
          )}

          {/* Login Form */}
          <form onSubmit={handleSubmit} style={{ width: '100%' }}>
            <Stack gap='md'>
              <TextInput
                label='Username'
                placeholder='admin'
                required
                value={formData.username}
                onChange={e => handleInputChange('username', e.target.value)}
                autoFocus
              />

              <PasswordInput
                label='Password'
                placeholder='••••••••'
                required
                value={formData.password}
                onChange={e => handleInputChange('password', e.target.value)}
              />

              <Button
                type='submit'
                fullWidth
                size='md'
                loading={isLoading}
                disabled={!formData.username || !formData.password}
              >
                Sign In
              </Button>
            </Stack>
          </form>

          {/* Signup Link - only show if signups are enabled */}
          {!signupStatusQuery.isLoading &&
            !signupStatusQuery.data?.signups_disabled && (
              <Text size='sm' c='dimmed'>
                Don't have an account?{' '}
                <Anchor
                  component='button'
                  onClick={() => navigate('/signup')}
                  fw={500}
                >
                  Sign up here
                </Anchor>
              </Text>
            )}

          {/* Google OAuth Button */}
          <Divider
            my='md'
            label='or'
            labelPosition='center'
            data-testid='oauth-divider'
          />
          <GoogleOAuthButton redirectUrl={redirectUrl} />
        </Stack>
      </Paper>
    </Container>
  );
};

export default LoginPage;
