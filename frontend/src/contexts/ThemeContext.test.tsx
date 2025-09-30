import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ThemeProvider, useTheme } from './ThemeContext';
import { Button } from '@mantine/core';
import { AllProviders } from '../test-utils';

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
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

describe('ThemeContext', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('provides default theme', () => {
    localStorageMock.getItem.mockReturnValue(undefined); // Ensure no theme is set
    render(
      <AllProviders>
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      </AllProviders>
    );

    expect(screen.getByTestId('current-theme')).toHaveTextContent('blue');
    expect(screen.getByTestId('theme-count')).toHaveTextContent('10');
  });

  it('loads theme from localStorage on mount', () => {
    localStorageMock.getItem.mockReturnValue('green');

    render(
      <AllProviders>
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      </AllProviders>
    );

    expect(screen.getByTestId('current-theme')).toHaveTextContent('green');
  });

  it('changes theme and saves to localStorage', () => {
    localStorageMock.getItem.mockReturnValue(undefined); // Ensure no theme is set
    render(
      <AllProviders>
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      </AllProviders>
    );

    // Initially blue
    expect(screen.getByTestId('current-theme')).toHaveTextContent('blue');

    // Change to green
    fireEvent.click(screen.getByText('Set Green Theme'));
    expect(screen.getByTestId('current-theme')).toHaveTextContent('green');
    expect(localStorageMock.setItem).toHaveBeenCalledWith(
      'quiz-theme',
      'green'
    );

    // Change to pink
    fireEvent.click(screen.getByText('Set Pink Theme'));
    expect(screen.getByTestId('current-theme')).toHaveTextContent('pink');
    expect(localStorageMock.setItem).toHaveBeenCalledWith('quiz-theme', 'pink');
  });

  it('ignores invalid theme from localStorage', () => {
    localStorageMock.getItem.mockReturnValue('invalid-theme');

    render(
      <AllProviders>
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      </AllProviders>
    );

    // Should fall back to default blue theme
    expect(screen.getByTestId('current-theme')).toHaveTextContent('blue');
  });
});
