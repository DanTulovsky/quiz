// This file is run before each test file.
import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';

// Comprehensive DOM API mocks for Mantine compatibility
beforeAll(() => {
  // Mock ResizeObserver for Mantine components
  global.ResizeObserver = vi.fn().mockImplementation(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }));

  // Mock IntersectionObserver for Mantine components
  global.IntersectionObserver = vi.fn().mockImplementation(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }));

  // Mock getComputedStyle for Mantine components
  global.getComputedStyle = vi.fn().mockImplementation(() => ({
    getPropertyValue: vi.fn().mockReturnValue(''),
    fontSize: '16px',
    lineHeight: '1.5',
  }));

  // Mock scrollTo
  global.scrollTo = vi.fn();

  // Mock scrollIntoView for Mantine components
  Element.prototype.scrollIntoView = vi.fn();

  // localStorage is now mocked by vitest-localstorage-mock

  // Mock document for ThemeProvider
  Object.defineProperty(global, 'document', {
    value: {
      documentElement: {
        style: {},
      },
      body: {
        style: {},
      },
    },
    writable: true,
  });

  // Mock URL.createObjectURL
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (global.URL as any).createObjectURL = vi.fn();

  // Mock matchMedia for Mantine hooks - this is critical for use-media-query
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => {
      const mediaQuery = {
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(), // deprecated
        removeListener: vi.fn(), // deprecated
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      };

      // Make sure the object is properly formed
      return mediaQuery;
    }),
  });

  // Ensure window has the property
  if (!window.matchMedia) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));
  }
});

beforeEach(() => {
  // Log the currently running test and reset all mocks before each test
  try {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const currentTestName = (expect as any).getState?.().currentTestName || '';
    // Print to stdout so test runners capture it in their logs
    // Use a clear prefix so it's easy to grep in logs
    // eslint-disable-next-line no-console
    // console.log(`>>> RUNNING TEST: ${currentTestName}`);
  } catch (e) {
    // ignore errors when retrieving test name
  }
  vi.clearAllMocks();
});

// Mock fetch globally to prevent real HTTP requests during tests
global.fetch = vi.fn().mockImplementation(() => {
  throw new Error('fetch() called without being mocked in test');
});

// Mock XMLHttpRequest to prevent real HTTP requests
global.XMLHttpRequest = vi.fn().mockImplementation(() => ({
  open: vi.fn(),
  send: vi.fn(),
  setRequestHeader: vi.fn(),
  getResponseHeader: vi.fn(),
  getAllResponseHeaders: vi.fn().mockReturnValue(''),
  readyState: 4,
  status: 200,
  statusText: 'OK',
  responseText: '',
  response: '',
  responseType: '',
  responseURL: '',
  timeout: 0,
  withCredentials: false,
  upload: {},
  onreadystatechange: null,
  onload: null,
  onerror: null,
  onabort: null,
  ontimeout: null,
  onloadstart: null,
  onprogress: null,
  onloadend: null,
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  dispatchEvent: vi.fn(),
}));
