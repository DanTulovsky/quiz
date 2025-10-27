import React, { useEffect, useState, ReactNode } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useNavigate, useLocation } from 'react-router-dom';
import { Button } from '@mantine/core';
import {
  useGetV1AuthStatus,
  usePostV1AuthLogin,
  usePostV1AuthLogout,
  usePutV1Settings,
  User,
  PutV1SettingsMutationBody as UserSettings,
} from '../api/api';
import { AuthContext } from './AuthContext';
import { showNotificationWithClean } from '../notifications';

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({
  children,
}: AuthProviderProps) => {
  const [user, setUser] = useState<User | null>(null);
  const [hasInitialized, setHasInitialized] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [isLoggingIn, setIsLoggingIn] = useState(false);
  const queryClient = useQueryClient();

  const {
    data: authStatusResponse,
    isLoading: isAuthStatusLoading,
    refetch,
  } = useGetV1AuthStatus({
    query: {
      queryKey: ['authStatus'],
      retry: 1, // Allow one retry instead of failing immediately
      // Set a timeout to prevent hanging indefinitely
      staleTime: 5000,
      // Fail fast if the request times out
      refetchOnWindowFocus: false,
      // Add a retry delay
      retryDelay: 1000,
    },
  });

  const loginMutation = usePostV1AuthLogin();
  const logoutMutation = usePostV1AuthLogout();
  const updateSettingsMutation = usePutV1Settings();

  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    // Handle the auth status response properly
    if (authStatusResponse) {
      // The backend returns different formats, so we need to handle both
      const response = authStatusResponse as {
        authenticated?: boolean;
        user?: User;
        username?: string;
      };

      if (response.authenticated === false) {
        // Not authenticated — if we previously had a user, the session likely expired
        // Don't show session expired notification if we're in the process of logging in or out
        if (user && hasInitialized && !isLoggingOut && !isLoggingIn) {
          const currentPath = `${location.pathname}${location.search}`;
          showNotificationWithClean({
            title: 'Session expired',
            message: (
              <div>
                Your session has expired.{' '}
                <Button
                  variant='outline'
                  size='xs'
                  onClick={() =>
                    navigate(
                      `/login?redirect=${encodeURIComponent(currentPath)}`
                    )
                  }
                >
                  Log in
                </Button>
              </div>
            ),
            color: 'yellow',
            autoClose: false,
          });
        }
        setUser(null);
      } else if (response.authenticated === true && response.user) {
        // Authenticated with user data
        setUser(response.user);
      } else if (response.username) {
        // Direct user object (fallback for different response formats)
        setUser(response as User);
      } else {
        // Unknown response format, assume not authenticated
        setUser(null);
      }
      setHasInitialized(true);
    }
  }, [authStatusResponse, user, hasInitialized, isLoggingIn, isLoggingOut]);

  const login = async (
    username: string,
    password: string
  ): Promise<boolean> => {
    try {
      setIsLoggingIn(true);
      const response = await loginMutation.mutateAsync({
        data: { username, password },
      });
      if (response.user) {
        // Clear any stale auth status data and logging out flag
        setIsLoggingOut(false);
        queryClient.removeQueries({ queryKey: ['authStatus'] });
        setUser(response.user);
        setHasInitialized(true); // Ensure we're initialized after login

        // Wait for React to process the state update before allowing navigation
        // Use requestAnimationFrame twice to ensure state is flushed
        await new Promise(resolve => {
          requestAnimationFrame(() => {
            requestAnimationFrame(resolve);
          });
        });

        // Clear the logging in flag after a short delay
        setTimeout(() => setIsLoggingIn(false), 1000);
        return true;
      }
      setIsLoggingIn(false);
      showNotificationWithClean({
        title: 'Error',
        message: response.message || 'Login failed.',
        color: 'error',
      });
      return false;
    } catch (error: unknown) {
      setIsLoggingIn(false);
      // Extract error message from the response
      let errorMessage = 'An unknown error occurred';

      if (error && typeof error === 'object' && 'response' in error) {
        const responseError = error as {
          response?: { data?: { error?: string } };
        };
        if (responseError.response?.data?.error) {
          errorMessage = responseError.response.data.error;
        }
      } else if (error instanceof Error) {
        errorMessage = error.message;
      }

      showNotificationWithClean({
        title: 'Error',
        message: errorMessage,
        color: 'error',
      });
      return false;
    }
  };

  const loginWithUser = async (userData: User): Promise<boolean> => {
    try {
      setIsLoggingIn(true);
      // Clear any stale auth status data and logging out flag
      setIsLoggingOut(false);
      queryClient.removeQueries({ queryKey: ['authStatus'] });
      setUser(userData);
      // Clear the logging in flag after a short delay
      setTimeout(() => setIsLoggingIn(false), 1000);
      return true;
    } catch {
      setIsLoggingIn(false);
      showNotificationWithClean({
        title: 'Error',
        message: 'An unknown error occurred',
        color: 'error',
      });
      return false;
    }
  };

  const logout = async (): Promise<void> => {
    try {
      setIsLoggingOut(true);

      // Navigate to login page first, before clearing user state
      navigate('/login');

      await logoutMutation.mutateAsync();

      // Clear auth status query cache to prevent stale data
      queryClient.removeQueries({ queryKey: ['authStatus'] });

      // Clear user state after navigation
      setUser(null);

      // Reset initialization state after navigation
      setTimeout(() => setHasInitialized(false), 200);

      showNotificationWithClean({
        title: 'Success',
        message: 'Logged out successfully',
        color: 'success',
      });

      // Clear the logging out flag after a short delay
      setTimeout(() => setIsLoggingOut(false), 1000);
    } catch {
      setIsLoggingOut(false);
      showNotificationWithClean({
        title: 'Error',
        message: 'Logout failed',
        color: 'error',
      });
    }
  };

  const updateSettings = async (settings: UserSettings): Promise<boolean> => {
    try {
      await updateSettingsMutation.mutateAsync({ data: settings });

      setUser((prevUser: User | null) => {
        if (!prevUser) return null;
        // Create a new user object with updated settings
        const newUser: User = {
          ...prevUser,
          preferred_language: settings.language,
          current_level: settings.level,
          ai_provider: settings.ai_provider,
          ai_model: settings.ai_model,
          ai_enabled: settings.ai_enabled,
        };
        return newUser;
      });

      // Invalidate API key availability queries to refetch them
      queryClient.invalidateQueries({
        predicate: query => {
          // Invalidate all API key availability queries (they start with '/v1/settings/api-key/')
          return (
            Array.isArray(query.queryKey) &&
            query.queryKey.length > 0 &&
            typeof query.queryKey[0] === 'string' &&
            query.queryKey[0].startsWith('/v1/settings/api-key/')
          );
        },
      });

      refetch(); // Refetch auth status to get the full updated user object from backend
      return true;
    } catch {
      showNotificationWithClean({
        title: 'Error',
        message: 'Failed to update settings',
        color: 'error',
      });
      return false;
    }
  };

  const refreshUser = async (): Promise<void> => {
    const result = await refetch();
    if (result.data) {
      // Handle different response formats
      const response = result.data as {
        authenticated?: boolean;
        user?: User;
        username?: string;
      };

      if (response.authenticated === true && response.user) {
        // Authenticated with user data
        setUser(response.user);
      } else if (response.username) {
        // Direct user object (fallback for different response formats)
        setUser(response as User);
      } else if (response.user) {
        // Direct user object
        setUser(response.user);
      }
    }
  };

  const value = {
    user,
    isAuthenticated: !!user,
    isLoading: isAuthStatusLoading || !hasInitialized,
    login,
    loginWithUser,
    logout,
    updateSettings,
    refreshUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
