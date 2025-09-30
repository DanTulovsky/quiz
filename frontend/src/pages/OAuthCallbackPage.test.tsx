import { render, screen, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import OAuthCallbackPage from './OAuthCallbackPage';
import { MantineProvider } from '@mantine/core';

// Mock fetch globally
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Mock useAuth hook
const mockLoginWithUser = vi.fn();
vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({
    loginWithUser: mockLoginWithUser,
  }),
}));

describe('OAuthCallbackPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const renderOAuthCallback = (searchParams = '') => {
    return render(
      <MantineProvider>
        <MemoryRouter initialEntries={[`/oauth-callback${searchParams}`]}>
          <Routes>
            <Route path='/oauth-callback' element={<OAuthCallbackPage />} />
          </Routes>
        </MemoryRouter>
      </MantineProvider>
    );
  };

  it('shows loading state initially', () => {
    renderOAuthCallback('?code=test-code&state=test-state');

    expect(
      screen.getByText('Processing authentication...')
    ).toBeInTheDocument();
    expect(screen.getByTestId('loader')).toBeInTheDocument();
  });

  it('handles successful authentication with redirect_uri', async () => {
    const mockUser = {
      id: 1,
      username: 'testuser',
      email: 'test@example.com',
    };

    const mockResponse = {
      success: true,
      message: 'Google authentication successful',
      user: mockUser,
      redirect_uri: '/daily',
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderOAuthCallback('?code=test-code&state=test-state');

    await waitFor(() => {
      expect(
        screen.getByText('Welcome, testuser! Logging you in...')
      ).toBeInTheDocument();
    });

    expect(mockLoginWithUser).toHaveBeenCalledWith(mockUser);
  });

  it('handles successful authentication without redirect_uri', async () => {
    const mockUser = {
      id: 1,
      username: 'testuser',
      email: 'test@example.com',
    };

    const mockResponse = {
      success: true,
      message: 'Google authentication successful',
      user: mockUser,
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderOAuthCallback('?code=test-code&state=test-state');

    await waitFor(() => {
      expect(
        screen.getByText('Welcome, testuser! Logging you in...')
      ).toBeInTheDocument();
    });

    expect(mockLoginWithUser).toHaveBeenCalledWith(mockUser);
  });

  it('handles authentication error', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: async () => ({ error: 'Authentication failed' }),
    });

    renderOAuthCallback('?code=test-code&state=test-state');

    await waitFor(() => {
      expect(
        screen.getByText('Authentication failed. Please try again.')
      ).toBeInTheDocument();
    });
  });

  it('handles missing code parameter', async () => {
    renderOAuthCallback('?state=test-state');

    await waitFor(() => {
      expect(
        screen.getByText('Invalid callback parameters.')
      ).toBeInTheDocument();
    });
  });

  it('handles missing state parameter', async () => {
    renderOAuthCallback('?code=test-code');

    await waitFor(() => {
      expect(
        screen.getByText('Invalid callback parameters.')
      ).toBeInTheDocument();
    });
  });

  it('handles OAuth error parameter', async () => {
    renderOAuthCallback('?error=access_denied');

    await waitFor(() => {
      expect(
        screen.getByText('Authentication was cancelled or failed.')
      ).toBeInTheDocument();
    });
  });

  it('handles invalid_grant error specifically', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: async () => ({ error: 'invalid_grant' }),
    });

    renderOAuthCallback('?code=test-code&state=test-state');

    await waitFor(() => {
      expect(
        screen.getByText(
          'This authentication link has already been used. Please try signing in again.'
        )
      ).toBeInTheDocument();
    });
  });

  it('handles network errors', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    renderOAuthCallback('?code=test-code&state=test-state');

    await waitFor(() => {
      expect(
        screen.getByText('Authentication failed. Please try again.')
      ).toBeInTheDocument();
    });
  });

  it('prevents multiple requests', async () => {
    const mockUser = {
      id: 1,
      username: 'testuser',
      email: 'test@example.com',
    };

    const mockResponse = {
      success: true,
      message: 'Google authentication successful',
      user: mockUser,
      redirect_uri: '/daily',
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderOAuthCallback('?code=test-code&state=test-state');

    // Wait for the first request to complete
    await waitFor(() => {
      expect(
        screen.getByText('Welcome, testuser! Logging you in...')
      ).toBeInTheDocument();
    });

    // Verify fetch was only called once
    expect(mockFetch).toHaveBeenCalledTimes(1);
  });
});
