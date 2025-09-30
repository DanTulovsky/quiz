import { useState, useEffect } from 'react';
import {
  isMobileDevice,
  forceMobileView,
  forceDesktopView,
  clearDeviceOverride,
  getDeviceView,
  supportsTouch,
} from '../utils/device';

/**
 * Hook for mobile device detection
 */
export const useMobileDetection = () => {
  const [isMobile, setIsMobile] = useState<boolean>(() => isMobileDevice());
  const [deviceView, setDeviceView] = useState<'mobile' | 'desktop' | 'auto'>(
    () => getDeviceView()
  );
  const [isTouchDevice] = useState<boolean>(() => supportsTouch());

  useEffect(() => {
    const handleResize = () => {
      const mobile = isMobileDevice();
      setIsMobile(mobile);
      setDeviceView(getDeviceView());
    };

    const handleStorageChange = () => {
      const mobile = isMobileDevice();
      setIsMobile(mobile);
      setDeviceView(getDeviceView());
    };

    // Listen for resize events
    window.addEventListener('resize', handleResize);

    // Listen for storage changes (for manual overrides)
    window.addEventListener('storage', handleStorageChange);

    // Also listen for our custom storage events
    const handleCustomStorageChange = () => handleStorageChange();
    window.addEventListener('deviceViewChanged', handleCustomStorageChange);

    return () => {
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('storage', handleStorageChange);
      window.removeEventListener(
        'deviceViewChanged',
        handleCustomStorageChange
      );
    };
  }, []);

  const setMobileView = () => {
    forceMobileView();
    setIsMobile(true);
    setDeviceView('mobile');
    window.dispatchEvent(new Event('deviceViewChanged'));
  };

  const setDesktopView = () => {
    forceDesktopView();
    setIsMobile(false);
    setDeviceView('desktop');
    window.dispatchEvent(new Event('deviceViewChanged'));
  };

  const resetView = () => {
    clearDeviceOverride();
    const mobile = isMobileDevice();
    setIsMobile(mobile);
    setDeviceView('auto');
    window.dispatchEvent(new Event('deviceViewChanged'));
  };

  return {
    isMobile,
    deviceView,
    isTouchDevice,
    setMobileView,
    setDesktopView,
    resetView,
  };
};
