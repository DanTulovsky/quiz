import React, {useState} from 'react';
import {Button} from '@mantine/core';
import {IconBrandGoogle} from '@tabler/icons-react';

interface GoogleOAuthButtonProps {
  onSuccess?: () => void;
  onError?: (error: string) => void;
  variant?: 'login' | 'signup';
  size?: 'sm' | 'md' | 'lg';
  fullWidth?: boolean;
  redirectUrl?: string;
}

const GoogleOAuthButton: React.FC<GoogleOAuthButtonProps> = ({
  onSuccess,
  onError,
  variant = 'login',
  size = 'md',
  fullWidth = true,
  redirectUrl,
}) => {
  const [isLoading, setIsLoading] = useState(false);

  const handleGoogleAuth = async () => {
    setIsLoading(true);
    try {
      // Get the current URL to preserve it through the OAuth flow
      // Use the provided redirectUrl if available, otherwise use current URL
      const currentPath =
        redirectUrl || window.location.pathname + window.location.search;

      // Call the backend to get the Google OAuth URL
      const response = await fetch(
        `/v1/auth/google/login?redirect_uri=${encodeURIComponent(currentPath)}`,
        {
          method: 'GET',
          credentials: 'include',
        }
      );

      if (!response.ok) {
        throw new Error('Failed to get Google OAuth URL');
      }

      const data = await response.json();

      if (!data.auth_url) {
        throw new Error('No auth URL received from server');
      }

      // Redirect to Google OAuth
      window.location.href = data.auth_url;

      // Call success callback if provided
      if (onSuccess) {
        onSuccess();
      }
    } catch (error) {
      if (onError) {
        onError(error instanceof Error ? error.message : 'Unknown error');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Button
      variant='outline'
      color='gray'
      size={size}
      fullWidth={fullWidth}
      type='button'
      onClick={handleGoogleAuth}
      onTouchEnd={handleGoogleAuth}
      disabled={isLoading}
      leftSection={<IconBrandGoogle size={16} data-testid='google-icon' />}
      style={{pointerEvents: 'auto'}}
    >
      {isLoading
        ? 'Loading...'
        : variant === 'signup'
          ? 'Sign up with Google'
          : 'Sign in with Google'}
    </Button>
  );
};

export default GoogleOAuthButton;
