import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useMobileDetection } from '../useMobileDetection';

// Mock the device utility
vi.mock('../../utils/device', () => ({
  isMobileDevice: vi.fn(),
  forceMobileView: vi.fn(),
  forceDesktopView: vi.fn(),
  clearDeviceOverride: vi.fn(),
  getDeviceView: vi.fn(),
  supportsTouch: vi.fn(),
}));

import {
  isMobileDevice,
  forceMobileView,
  forceDesktopView,
  clearDeviceOverride,
  getDeviceView,
  supportsTouch,
} from '../../utils/device';

// Type the mocked functions properly
import type { MockedFunction } from 'vitest';
const mockIsMobileDevice = isMobileDevice as MockedFunction<
  typeof isMobileDevice
>;
const mockGetDeviceView = getDeviceView as MockedFunction<
  typeof getDeviceView
>;
const mockSupportsTouch = supportsTouch as MockedFunction<
  typeof supportsTouch
>;

// Mock window events
const mockAddEventListener = vi.fn();
const mockRemoveEventListener = vi.fn();
const mockDispatchEvent = vi.fn();

Object.defineProperty(window, 'addEventListener', {
  value: mockAddEventListener,
  writable: true,
});

Object.defineProperty(window, 'removeEventListener', {
  value: mockRemoveEventListener,
  writable: true,
});

Object.defineProperty(window, 'dispatchEvent', {
  value: mockDispatchEvent,
  writable: true,
});

describe('useMobileDetection', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    // Reset mocks
    mockIsMobileDevice.mockReturnValue(true);
    mockGetDeviceView.mockReturnValue('auto');
    mockSupportsTouch.mockReturnValue(true);

    // Reset event listeners
    mockAddEventListener.mockClear();
    mockRemoveEventListener.mockClear();
    mockDispatchEvent.mockClear();
  });

  it('should respond to resize events', () => {
    mockIsMobileDevice.mockReturnValue(false);

    const { result } = renderHook(() => useMobileDetection());

    // Simulate resize event
    const resizeHandler = mockAddEventListener.mock.calls.find(
      call => call[0] === 'resize'
    )?.[1];

    act(() => {
      mockIsMobileDevice.mockReturnValue(true);
      resizeHandler();
    });

    expect(result.current.isMobile).toBe(true);
  });

  it('should respond to storage events', () => {
    mockGetDeviceView.mockReturnValue('mobile');

    const { result } = renderHook(() => useMobileDetection());

    // Simulate storage event
    const storageHandler = mockAddEventListener.mock.calls.find(
      call => call[0] === 'storage'
    )?.[1];

    act(() => {
      mockGetDeviceView.mockReturnValue('desktop');
      storageHandler();
    });

    expect(result.current.deviceView).toBe('desktop');
  });

  it('should respond to custom deviceViewChanged events', () => {
    mockIsMobileDevice.mockReturnValue(false);

    const { result } = renderHook(() => useMobileDetection());

    // Simulate custom event
    const customHandler = mockAddEventListener.mock.calls.find(
      call => call[0] === 'deviceViewChanged'
    )?.[1];

    act(() => {
      mockIsMobileDevice.mockReturnValue(true);
      customHandler();
    });

    expect(result.current.isMobile).toBe(true);
  });

  it('should set mobile view correctly', () => {
    const { result } = renderHook(() => useMobileDetection());

    act(() => {
      result.current.setMobileView();
    });

    expect(forceMobileView).toHaveBeenCalled();
    expect(mockDispatchEvent).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'deviceViewChanged' })
    );
  });

  it('should set desktop view correctly', () => {
    const { result } = renderHook(() => useMobileDetection());

    act(() => {
      result.current.setDesktopView();
    });

    expect(forceDesktopView).toHaveBeenCalled();
    expect(mockDispatchEvent).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'deviceViewChanged' })
    );
  });

  it('should reset view correctly', () => {
    const { result } = renderHook(() => useMobileDetection());

    act(() => {
      result.current.resetView();
    });

    expect(clearDeviceOverride).toHaveBeenCalled();
    expect(mockDispatchEvent).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'deviceViewChanged' })
    );
  });

  it('should clean up event listeners on unmount', () => {
    const { unmount } = renderHook(() => useMobileDetection());

    unmount();

    expect(mockRemoveEventListener).toHaveBeenCalledWith(
      'resize',
      expect.any(Function)
    );
    expect(mockRemoveEventListener).toHaveBeenCalledWith(
      'storage',
      expect.any(Function)
    );
    expect(mockRemoveEventListener).toHaveBeenCalledWith(
      'deviceViewChanged',
      expect.any(Function)
    );
  });

  it('should initialize with correct default values', () => {
    const { result } = renderHook(() => useMobileDetection());

    expect(result.current.isMobile).toBe(true);
    expect(result.current.deviceView).toBe('auto');
    expect(result.current.isTouchDevice).toBe(true);
  });
});
