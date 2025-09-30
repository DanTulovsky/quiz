import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  isMobileDevice,
  forceMobileView,
  forceDesktopView,
  clearDeviceOverride,
  getDeviceView,
  supportsTouch,
} from '../device';

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

// Mock navigator
Object.defineProperty(window, 'navigator', {
  value: {
    userAgent:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15',
  },
  writable: true,
});

// Mock window properties
Object.defineProperty(window, 'innerWidth', {
  value: 375, // iPhone width
  writable: true,
});

describe('Device Detection Utilities', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.getItem.mockReturnValue(null);
    Object.defineProperty(window, 'innerWidth', {
      value: 375,
      writable: true,
    });
    Object.defineProperty(window, 'navigator', {
      value: {
        userAgent:
          'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15',
      },
      writable: true,
    });
  });

  describe('isMobileDevice', () => {
    it('should detect mobile device from user agent', () => {
      expect(isMobileDevice()).toBe(true);
    });

    it('should detect mobile device from screen size', () => {
      Object.defineProperty(window, 'navigator', {
        value: {
          userAgent:
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
        },
        writable: true,
      });
      Object.defineProperty(window, 'innerWidth', {
        value: 600, // Small screen
        writable: true,
      });

      expect(isMobileDevice()).toBe(true);
    });

    it('should return false for desktop user agent and large screen', () => {
      Object.defineProperty(window, 'navigator', {
        value: {
          userAgent:
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
        },
        writable: true,
      });
      Object.defineProperty(window, 'innerWidth', {
        value: 1200, // Large screen
        writable: true,
      });

      expect(isMobileDevice()).toBe(false);
    });

    it('should respect localStorage override for mobile', () => {
      localStorageMock.getItem.mockReturnValue('mobile');
      expect(isMobileDevice()).toBe(true);
    });

    it('should respect localStorage override for desktop', () => {
      localStorageMock.getItem.mockReturnValue('desktop');
      expect(isMobileDevice()).toBe(false);
    });

    it('should ignore invalid localStorage values', () => {
      localStorageMock.getItem.mockReturnValue('invalid');
      expect(isMobileDevice()).toBe(true); // Falls back to detection
    });
  });

  describe('forceMobileView', () => {
    it('should set localStorage to mobile', () => {
      forceMobileView();
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'deviceView',
        'mobile'
      );
    });
  });

  describe('forceDesktopView', () => {
    it('should set localStorage to desktop', () => {
      forceDesktopView();
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'deviceView',
        'desktop'
      );
    });
  });

  describe('clearDeviceOverride', () => {
    it('should remove deviceView from localStorage', () => {
      clearDeviceOverride();
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('deviceView');
    });
  });

  describe('getDeviceView', () => {
    it('should return mobile when localStorage is set to mobile', () => {
      localStorageMock.getItem.mockReturnValue('mobile');
      expect(getDeviceView()).toBe('mobile');
    });

    it('should return desktop when localStorage is set to desktop', () => {
      localStorageMock.getItem.mockReturnValue('desktop');
      expect(getDeviceView()).toBe('desktop');
    });

    it('should return auto when no override is set', () => {
      localStorageMock.getItem.mockReturnValue(null);
      expect(getDeviceView()).toBe('auto');
    });
  });

  describe('supportsTouch', () => {
    it('should return true when ontouchstart is available', () => {
      Object.defineProperty(window, 'ontouchstart', {
        value: () => {},
        writable: true,
      });
      expect(supportsTouch()).toBe(true);
    });

    it('should return true when maxTouchPoints > 0', () => {
      // Remove ontouchstart property for testing
      const windowWithTouchStart = window as typeof window & {
        ontouchstart?: unknown;
      };
      delete windowWithTouchStart.ontouchstart;
      Object.defineProperty(window, 'navigator', {
        value: {
          maxTouchPoints: 5,
        },
        writable: true,
      });
      expect(supportsTouch()).toBe(true);
    });

    it('should return false when no touch support', () => {
      // Remove ontouchstart property for testing
      const windowWithTouchStart = window as typeof window & {
        ontouchstart?: unknown;
      };
      delete windowWithTouchStart.ontouchstart;
      Object.defineProperty(window, 'navigator', {
        value: {
          maxTouchPoints: 0,
        },
        writable: true,
      });
      expect(supportsTouch()).toBe(false);
    });
  });
});
