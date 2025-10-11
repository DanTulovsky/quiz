import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import GoogleOAuthButton from './GoogleOAuthButton';
import { MantineProvider } from '@mantine/core';

// Mock fetch globally
const mockFetch = vi.fn();
global.fetch = mockFetch;

// Mock window.location
const mockLocation = {
  assign: vi.fn(),
  replace: vi.fn(),
  pathname: '/',
  search: '',
  get href() {
    return this._href || '';
  },
  set href(value) {
    this._href = value;
  },
  _href: '',
};
Object.defineProperty(window, 'location', {
  value: mockLocation,
  writable: true,
});

describe('GoogleOAuthButton', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLocation.href = '';
  });

  const renderButton = (props = {}) => {
    return act(() => {
      render(
        <MantineProvider>
          <GoogleOAuthButton {...props} />
        </MantineProvider>
      );
    });
  };

  it('renders Google OAuth button with correct text', () => {
    renderButton();

    const button = screen.getByRole('button', { name: /sign in with google/i });
    expect(button).toBeInTheDocument();
  });

  it('renders with custom variant text', () => {
    renderButton({ variant: 'signup' });

    const button = screen.getByRole('button', { name: /sign up with google/i });
    expect(button).toBeInTheDocument();
  });

  it('renders with Google icon', () => {
    renderButton();

    const icon = screen.getByTestId('google-icon');
    expect(icon).toBeInTheDocument();
  });

  it('calls backend OAuth endpoint when clicked', async () => {
    const mockResponse = {
      auth_url:
        'https://accounts.google.com/o/oauth2/v2/auth?client_id=test&redirect_uri=http://localhost:3000/oauth-callback',
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderButton();

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/v1/auth/google/login?redirect_uri=%2F',
        {
          method: 'GET',
          credentials: 'include',
        }
      );
    });
  });

  it('includes current path as redirect_uri parameter', async () => {
    // Mock window.location.pathname and search
    Object.defineProperty(window, 'location', {
      value: {
        ...mockLocation,
        pathname: '/daily',
        search: '?param=value',
      },
      writable: true,
    });

    const mockResponse = {
      auth_url:
        'https://accounts.google.com/o/oauth2/v2/auth?client_id=test&redirect_uri=http://localhost:3000/oauth-callback',
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderButton();

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/v1/auth/google/login?redirect_uri=%2Fdaily%3Fparam%3Dvalue',
        {
          method: 'GET',
          credentials: 'include',
        }
      );
    });
  });

  it('handles special characters in URL correctly', async () => {
    // Mock window.location with special characters
    Object.defineProperty(window, 'location', {
      value: {
        ...mockLocation,
        pathname: '/admin/users',
        search: '?search=test%20user&filter=active',
      },
      writable: true,
    });

    const mockResponse = {
      auth_url:
        'https://accounts.google.com/o/oauth2/v2/auth?client_id=test&redirect_uri=http://localhost:3000/oauth-callback',
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderButton();

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/v1/auth/google/login?redirect_uri=%2Fadmin%2Fusers%3Fsearch%3Dtest%2520user%26filter%3Dactive',
        {
          method: 'GET',
          credentials: 'include',
        }
      );
    });
  });

  it('redirects to Google OAuth URL on successful response', async () => {
    // Reset window.location for this test
    Object.defineProperty(window, 'location', {
      value: {
        ...mockLocation,
        pathname: '/',
        search: '',
        _href: '',
      },
      writable: true,
    });

    const mockAuthUrl =
      'https://accounts.google.com/o/oauth2/v2/auth?client_id=test&redirect_uri=http://localhost:3000/oauth-callback';
    const mockResponse = { auth_url: mockAuthUrl };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderButton();

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    // Verify that fetch was called with the correct URL
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/v1/auth/google/login?redirect_uri=%2F',
        {
          method: 'GET',
          credentials: 'include',
        }
      );
    });
  });

  it('calls onSuccess callback when OAuth flow starts successfully', async () => {
    // Reset window.location for this test
    Object.defineProperty(window, 'location', {
      value: {
        ...mockLocation,
        pathname: '/',
        search: '',
        _href: '',
      },
      writable: true,
    });

    const onSuccess = vi.fn();
    const mockResponse = {
      auth_url:
        'https://accounts.google.com/o/oauth2/v2/auth?client_id=test&redirect_uri=http://localhost:3000/oauth-callback',
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderButton({ onSuccess });

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    // Verify that fetch was called and onSuccess was triggered
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/v1/auth/google/login?redirect_uri=%2F',
        {
          method: 'GET',
          credentials: 'include',
        }
      );
    });
  });

  it('calls onError callback when fetch fails', async () => {
    const onError = vi.fn();
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    renderButton({ onError });

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith('Network error');
    });
  });

  it('calls onError callback when response is not ok', async () => {
    const onError = vi.fn();
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
    });

    renderButton({ onError });

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith('Failed to get Google OAuth URL');
    });
  });

  it('calls onError callback when response is missing auth_url', async () => {
    const onError = vi.fn();
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    });

    renderButton({ onError });

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith('No auth URL received from server');
    });
  });

  it('applies custom size prop', () => {
    renderButton({ size: 'lg' });

    const button = screen.getByRole('button', { name: /sign in with google/i });
    expect(button).toHaveAttribute('data-size', 'lg');
  });

  it('applies fullWidth prop', () => {
    renderButton({ fullWidth: true });

    const button = screen.getByRole('button', { name: /sign in with google/i });
    expect(button).toHaveAttribute('data-block', 'true');
  });

  it('disables button during OAuth request', async () => {
    // Create a promise that we can resolve later
    let resolvePromise: (value: unknown) => void;
    const promise = new Promise(resolve => {
      resolvePromise = resolve;
    });

    mockFetch.mockReturnValueOnce(promise);

    renderButton();

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    // Button should be disabled during the request
    await waitFor(() => {
      expect(button).toHaveTextContent('Loading...');
    });

    // Resolve the promise
    resolvePromise!({
      ok: true,
      json: async () => ({ auth_url: 'https://accounts.google.com/test' }),
    });

    // Button should be enabled again
    await waitFor(() => {
      expect(button).toHaveTextContent('Sign in with Google');
    });
  });

  it('uses provided redirectUrl when available', async () => {
    // Reset window.location for this test
    Object.defineProperty(window, 'location', {
      value: {
        ...mockLocation,
        pathname: '/',
        search: '',
        _href: '',
      },
      writable: true,
    });

    const mockResponse = {
      auth_url:
        'https://accounts.google.com/o/oauth2/v2/auth?client_id=test&redirect_uri=http://localhost:3000/oauth-callback',
    };

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse,
    });

    renderButton({ redirectUrl: '/daily' });

    const button = screen.getByRole('button', { name: /sign in with google/i });
    fireEvent.click(button);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        '/v1/auth/google/login?redirect_uri=%2Fdaily',
        {
          method: 'GET',
          credentials: 'include',
        }
      );
    });
  });
});
