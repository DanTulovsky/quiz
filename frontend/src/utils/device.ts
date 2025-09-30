/**
 * Mobile device detection utilities
 */

/**
 * Detects if the current device should use mobile interface
 */
export const isMobileDevice = (): boolean => {
  // Check localStorage for manual override
  const override = localStorage.getItem('deviceView');
  if (override === 'mobile') return true;
  if (override === 'desktop') return false;

  // User agent detection
  const ua = navigator.userAgent;
  const isMobileUA =
    /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(ua);

  // Screen size detection (fallback)
  const isSmallScreen = window.innerWidth < 768;

  return isMobileUA || isSmallScreen;
};

/**
 * Forces mobile view regardless of device detection
 */
export const forceMobileView = (): void => {
  localStorage.setItem('deviceView', 'mobile');
};

/**
 * Forces desktop view regardless of device detection
 */
export const forceDesktopView = (): void => {
  localStorage.setItem('deviceView', 'desktop');
};

/**
 * Clears any manual device view override
 */
export const clearDeviceOverride = (): void => {
  localStorage.removeItem('deviceView');
};

/**
 * Gets the current device view setting
 */
export const getDeviceView = (): 'mobile' | 'desktop' | 'auto' => {
  const override = localStorage.getItem('deviceView');
  if (override === 'mobile') return 'mobile';
  if (override === 'desktop') return 'desktop';
  return 'auto';
};

/**
 * Checks if the device supports touch
 */
export const supportsTouch = (): boolean => {
  return 'ontouchstart' in window || navigator.maxTouchPoints > 0;
};

/**
 * Checks if the current path is a mobile route
 */
export const isMobilePath = (): boolean => {
  return window.location.pathname.startsWith('/m/');
};
