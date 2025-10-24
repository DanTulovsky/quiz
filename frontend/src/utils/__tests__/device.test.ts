import { describe, it, expect, beforeEach } from 'vitest';
import {
  isMobileDevice,
  forceMobileView,
  forceDesktopView,
  clearDeviceOverride,
  getDeviceView,
  supportsTouch,
} from '../device';

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
    // Clear localStorage before each test
    localStorage.clear();

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
      localStorage.setItem('deviceView', 'mobile');
      expect(isMobileDevice()).toBe(true);
    });

    it('should respect localStorage override for desktop', () => {
      localStorage.setItem('deviceView', 'desktop');
      expect(isMobileDevice()).toBe(false);
    });

    it('should ignore invalid localStorage values', () => {
      localStorage.setItem('deviceView', 'invalid');
      expect(isMobileDevice()).toBe(true); // Falls back to detection
    });
  });

  describe('forceMobileView', () => {
    it('should set localStorage to mobile', () => {
      forceMobileView();
      expect(localStorage.getItem('deviceView')).toBe('mobile');
    });
  });

  describe('forceDesktopView', () => {
    it('should set localStorage to desktop', () => {
      forceDesktopView();
      expect(localStorage.getItem('deviceView')).toBe('desktop');
    });
  });

  describe('clearDeviceOverride', () => {
    it('should remove deviceView from localStorage', () => {
      localStorage.setItem('deviceView', 'mobile');
      clearDeviceOverride();
      expect(localStorage.getItem('deviceView')).toBeNull();
    });
  });

  describe('getDeviceView', () => {
    it('should return mobile when localStorage is set to mobile', () => {
      localStorage.setItem('deviceView', 'mobile');
      expect(getDeviceView()).toBe('mobile');
    });

    it('should return desktop when localStorage is set to desktop', () => {
      localStorage.setItem('deviceView', 'desktop');
      expect(getDeviceView()).toBe('desktop');
    });

    it('should return auto when no override is set', () => {
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
