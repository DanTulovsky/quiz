import { screen, fireEvent, act } from '@testing-library/react';
import { vi } from 'vitest';
import { renderWithProviders } from '../test-utils';
import VersionDisplay from './VersionDisplay';

// Mock the version utility
vi.mock('../utils/version', () => ({
  getAppVersion: () => ({
    version: '1.0.0',
    buildTime: '2024-01-01T00:00:00.000Z',
    commitHash: 'abc12345',
  }),
  formatVersion: () => 'v1.0.0 (1/1/2024)',
}));

// Mock notifications
vi.mock('@mantine/notifications', () => ({
  notifications: {
    show: vi.fn(),
  },
  Notifications: () => null,
}));

// Mock clipboard API
Object.assign(navigator, {
  clipboard: {
    writeText: vi.fn(),
  },
});

describe('VersionDisplay', () => {
  it('renders version information', async () => {
    await act(async () => {
      renderWithProviders(<VersionDisplay />);
    });

    expect(screen.getByTestId('app-version')).toBeInTheDocument();
    expect(screen.getByText('v1.0.0 (1/1/2024)')).toBeInTheDocument();
  });

  it('has correct positioning styles', async () => {
    await act(async () => {
      renderWithProviders(<VersionDisplay />);
    });

    const versionElement = screen.getByTestId('app-version');
    // Check that the element has the correct positioning
    expect(versionElement).toBeInTheDocument();
    // The styles are applied via inline style, so we check the style attribute
    expect(versionElement).toHaveAttribute('style');
    const style = versionElement.getAttribute('style');
    expect(style).toContain('position: fixed');
    expect(style).toContain('bottom: 8px');
    expect(style).toContain('left: 8px');
    expect(style).toContain('cursor: pointer');
  });

  it('copies version information to clipboard when clicked', async () => {
    const mockWriteText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText,
      },
    });

    const { notifications } = await import('@mantine/notifications');
    const mockShow = vi.mocked(notifications.show);

    await act(async () => {
      renderWithProviders(<VersionDisplay />);
    });

    const versionElement = screen.getByTestId('app-version');

    // First click opens popover
    fireEvent.click(versionElement);

    // Now click inside the popover to copy
    const copyHint = await screen.findByText(/copy all version info/i);
    fireEvent.click(copyHint);

    // Wait for async clipboard copy
    await new Promise(r => setTimeout(r, 0));

    expect(mockWriteText).toHaveBeenCalledWith(
      expect.stringContaining('"frontend"')
    );
    expect(mockShow).toHaveBeenCalledWith({
      title: 'Version info copied!',
      message: 'All version information has been copied to clipboard',
      color: 'green',
      autoClose: 2000,
    });
  });
});
