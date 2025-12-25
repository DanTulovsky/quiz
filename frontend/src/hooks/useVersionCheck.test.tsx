import { render, screen, act } from '@testing-library/react';
import { vi } from 'vitest';
// renderWithProviders not needed in these tests; use plain render
import useVersionCheck from './useVersionCheck';

function TestComponent() {
  const { isUpdateAvailable, applyUpdate, dismiss } = useVersionCheck();
  return (
    <div>
      {isUpdateAvailable && <span data-testid='update-available'>update</span>}
      <button onClick={() => applyUpdate()}>apply</button>
      <button onClick={() => dismiss()}>dismiss</button>
    </div>
  );
}

describe('useVersionCheck', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  beforeAll(() => {
    // Ensure matchMedia exists for ThemeContext + Mantine hooks in JSDOM
    if (
      typeof (window as unknown as { matchMedia?: unknown }).matchMedia !==
      'function'
    ) {
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        configurable: true,
        value: (query: string) => ({
          matches: false,
          media: query,
          addListener: () => {},
          removeListener: () => {},
          addEventListener: () => {},
          removeEventListener: () => {},
          dispatchEvent: () => false,
        }),
      });
    }
  });

  it('shows update when service worker registration has waiting worker', async () => {
    const mockWaiting = {
      postMessage: vi.fn(),
      skipWaiting: vi.fn(),
    } as { postMessage: ReturnType<typeof vi.fn>; skipWaiting: ReturnType<typeof vi.fn> };
    const mockReg = {
      waiting: mockWaiting,
      addEventListener: vi.fn(),
    } as unknown;

    // Mock navigator.serviceWorker.getRegistration
    vi.stubGlobal('navigator', {
      serviceWorker: {
        getRegistration: () => Promise.resolve(mockReg),
        addEventListener: vi.fn(),
      },
    } as unknown as Navigator);

    await act(async () => {
      render(<TestComponent />);
      // allow effects to run
      await Promise.resolve();
    });

    expect(screen.getByTestId('update-available')).toBeInTheDocument();

    // Click apply and ensure we message the waiting worker
    await act(async () => {
      screen.getByText('apply').click();
      await Promise.resolve();
    });

    expect(mockWaiting.postMessage).toHaveBeenCalled();
  });

  it('falls back to version.json polling and shows update when commit differs', async () => {
    // Mock no SW registration
    vi.stubGlobal('navigator', {
      serviceWorker: { getRegistration: () => Promise.resolve(undefined) },
    } as unknown as Navigator);

    // Mock fetch to return different commitHash
    const remote = {
      frontend: {
        commitHash: 'remote1234',
        version: '1.2.3',
        buildTime: new Date().toISOString(),
      },
    };
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve({ ok: true, json: () => Promise.resolve(remote) })
      )
    );

    await act(async () => {
      render(<TestComponent />);
      // allow effect and fetch
      await Promise.resolve();
    });

    expect(screen.getByTestId('update-available')).toBeInTheDocument();
  });
});
