import {
  render,
  screen,
  fireEvent,
  act,
  cleanup,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ThemeProvider, useTheme } from './ThemeContext';
import { Button, MantineProvider } from '@mantine/core';

// Create a storage object that will be reset per test
let storage: Record<string, string> = {};

// Create localStorage mock
const createLocalStorageMock = () => ({
  getItem: vi.fn((key: string) => storage[key] || null),
  setItem: vi.fn((key: string, value: string) => {
    storage[key] = value;
  }),
  removeItem: vi.fn((key: string) => {
    delete storage[key];
  }),
  clear: vi.fn(() => {
    storage = {};
  }),
  length: Object.keys(storage).length,
  key: vi.fn((index: number) => Object.keys(storage)[index] || null),
});

// Test component that uses the theme context
const TestComponent = () => {
  const { currentTheme, setTheme, themeNames } = useTheme();

  return (
    <div>
      <div data-testid='current-theme'>{currentTheme}</div>
      <div data-testid='theme-count'>{Object.keys(themeNames).length}</div>
      <Button onClick={() => setTheme('green')}>Set Green Theme</Button>
      <Button onClick={() => setTheme('pink')}>Set Pink Theme</Button>
    </div>
  );
};

// Simple test wrapper that doesn't reuse providers
const TestWrapper = ({ children }: { children: React.ReactNode }) => (
  <MantineProvider>
    <ThemeProvider>{children}</ThemeProvider>
  </MantineProvider>
);

describe('ThemeContext', () => {
  beforeEach(() => {
    // Reset storage
    storage = {};
    // Create fresh localStorage mock for each test
    Object.defineProperty(window, 'localStorage', {
      value: createLocalStorageMock(),
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('provides default theme', () => {
    // Ensure no theme is set in storage
    act(() => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );
    });

    expect(screen.getByTestId('current-theme')).toHaveTextContent('blue');
    expect(screen.getByTestId('theme-count')).toHaveTextContent('10');
  });

  it('loads theme from localStorage on mount', () => {
    storage['quiz-theme'] = 'green';

    act(() => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );
    });

    expect(screen.getByTestId('current-theme')).toHaveTextContent('green');
  });

  it('changes theme and saves to localStorage', () => {
    // Ensure no theme is set in storage
    act(() => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );
    });

    // Initially blue
    expect(screen.getByTestId('current-theme')).toHaveTextContent('blue');

    // Change to green
    fireEvent.click(screen.getByText('Set Green Theme'));
    expect(screen.getByTestId('current-theme')).toHaveTextContent('green');
    expect(window.localStorage.setItem).toHaveBeenCalledWith(
      'quiz-theme',
      'green'
    );

    // Change to pink
    fireEvent.click(screen.getByText('Set Pink Theme'));
    expect(screen.getByTestId('current-theme')).toHaveTextContent('pink');
    expect(window.localStorage.setItem).toHaveBeenCalledWith(
      'quiz-theme',
      'pink'
    );
  });

  it('ignores invalid theme from localStorage', () => {
    storage['quiz-theme'] = 'invalid-theme';

    act(() => {
      render(
        <TestWrapper>
          <TestComponent />
        </TestWrapper>
      );
    });

    // Should fall back to default blue theme
    expect(screen.getByTestId('current-theme')).toHaveTextContent('blue');
  });
});
