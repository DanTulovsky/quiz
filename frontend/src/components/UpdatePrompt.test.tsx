import { screen, act } from '@testing-library/react';
import { vi } from 'vitest';
import { renderWithProviders } from '@/test-utils';
import UpdatePrompt from './UpdatePrompt';

vi.mock('@/hooks/useVersionCheck', () => ({
  __esModule: true,
  default: () => ({
    isUpdateAvailable: true,
    applyUpdate: vi.fn(),
    dismiss: vi.fn(),
  }),
}));

describe('UpdatePrompt', () => {
  it('renders when update is available', async () => {
    await act(async () => {
      renderWithProviders(<UpdatePrompt />);
    });

    expect(screen.getByTestId('update-prompt')).toBeInTheDocument();
    expect(screen.getByText(/new version of the app/i)).toBeInTheDocument();
  });
});
