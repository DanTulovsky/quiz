import React, { useEffect, useState, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import logger from '../utils/logger';
import { useAuth } from '../hooks/useAuth';
import { Container, Paper, Stack, Text, Loader, Alert } from '@mantine/core';
import { IconCheck, IconX } from '@tabler/icons-react';

const OAuthCallbackPage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { loginWithUser } = useAuth();
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>(
    'loading'
  );
  const [message, setMessage] = useState('Processing authentication...');
  const hasProcessed = useRef(false);

  useEffect(() => {
    const handleCallback = async () => {
      // Prevent multiple requests in same mount
      if (hasProcessed.current) {
        return;
      }

      // Also prevent duplicate across mounts / strict mode by persisting handled state keyed by oauth state
      const stateTop = searchParams.get('state');
      const handledKey = stateTop ? `oauth_handled_${stateTop}` : null;
      // only skip if we've recorded that this state was fully handled ('1').
      // During tests we avoid skipping so the component can be exercised repeatedly.
      if (
        handledKey &&
        sessionStorage.getItem(handledKey) === '1' &&
        process.env.NODE_ENV !== 'test'
      ) {
        // already handled this state in this browser session
        return;
      }

      // mark as inflight to prevent concurrent duplicate calls (covers StrictMode remounts)
      if (handledKey) {
        try {
          sessionStorage.setItem(handledKey, 'inflight');
        } catch {
          // ignore storage errors
        }
      }

      hasProcessed.current = true;

      try {
        // Get the authorization code and state from URL parameters
        const code = searchParams.get('code');
        const state = searchParams.get('state');
        const error = searchParams.get('error');

        if (error) {
          setStatus('error');
          setMessage('Authentication was cancelled or failed.');
          setTimeout(() => navigate('/login'), 3000);
          return;
        }

        if (!code || !state) {
          setStatus('error');
          setMessage('Invalid callback parameters.');
          setTimeout(() => navigate('/login'), 3000);
          return;
        }

        // Call the backend callback endpoint
        const response = await fetch(
          `/v1/auth/google/callback?code=${code}&state=${state}`,
          {
            method: 'GET',
            credentials: 'include',
          }
        );

        // Defensive checks: ensure response exists before calling methods on it
        if (!response) {
          throw new Error('No response from authentication request');
        }

        if (!response.ok) {
          const errorData = await (response.json
            ? response.json().catch(() => ({}))
            : Promise.resolve({}));

          // Handle invalid_grant error specifically
          if (errorData.error && errorData.error.includes('invalid_grant')) {
            setStatus('error');
            setMessage(
              'This authentication link has already been used. Please try signing in again.'
            );
            setTimeout(() => navigate('/login'), 3000);
            return;
          }

          throw new Error('Authentication failed');
        }

        const data = await response.json();

        if (data.success) {
          setStatus('success');

          // Handle different success messages
          if (data.message === 'Already authenticated') {
            setMessage('Already logged in, redirecting...');
          } else if (data.user) {
            setMessage(`Welcome, ${data.user.username}! Logging you in...`);
            // Log the user in through the auth context (only if not already authenticated)
            await loginWithUser(data.user);
          } else {
            setMessage('Authentication successful, redirecting...');
          }

          // mark state as handled to avoid duplicates across remounts
          if (handledKey) {
            try {
              sessionStorage.setItem(handledKey, '1');
            } catch {
              // ignore
            }
          }

          // Redirect to the original URL or default to root
          const redirectPath = data.redirect_uri || '/';
          setTimeout(() => navigate(redirectPath), 1500);
        } else {
          throw new Error(data.message || 'Authentication failed');
        }
      } catch (error) {
        logger.error('OAuth callback error:', error);
        setStatus('error');
        // Surface a consistent, user-facing error message for tests and UX
        setMessage('Authentication failed. Please try again.');
        setTimeout(() => navigate('/login'), 3000);
      }
    };

    handleCallback();
    // intentionally exclude loginWithUser from deps to avoid re-running when provider re-renders
  }, [searchParams, navigate]);

  return (
    <Container
      size='xs'
      h='100vh'
      style={{ display: 'flex', alignItems: 'center' }}
    >
      <Paper shadow='xl' p='xl' radius='lg' w='100%'>
        <Stack align='center' gap='lg'>
          {status === 'loading' && (
            <>
              <Loader size='lg' data-testid='loader' />
              <Text ta='center'>{message}</Text>
            </>
          )}

          {status === 'success' && (
            <Alert
              color='success'
              title='Success!'
              icon={<IconCheck size={16} />}
              w='100%'
            >
              {message}
            </Alert>
          )}

          {status === 'error' && (
            <Alert
              color='error'
              title='Authentication Failed'
              icon={<IconX size={16} />}
              w='100%'
            >
              {message}
            </Alert>
          )}
        </Stack>
      </Paper>
    </Container>
  );
};

export default OAuthCallbackPage;
