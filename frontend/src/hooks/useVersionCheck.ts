import { useCallback, useEffect, useRef, useState } from 'react';
import { getAppVersion } from '@/utils/version';

const VERSION_JSON_PATH = '/meta/version.json';
const POLL_INTERVAL_MS = 5 * 60 * 1000; // 5 minutes
const DISMISS_DELAY_MS = 15 * 60 * 1000; // 15 minutes for "Later"

interface VersionInfo {
  version?: string;
  commitHash?: string;
  buildTime?: string;
}

export function useVersionCheck() {
  const [isUpdateAvailable, setIsUpdateAvailable] = useState(false);
  const registrationRef = useRef<ServiceWorkerRegistration | null>(null);
  const dismissedUntilRef = useRef<number | null>(null);
  const pollingTimerRef = useRef<number | null>(null);

  const getLocalVersion = useCallback((): VersionInfo => {
    return getAppVersion();
  }, []);

  const shouldShow = useCallback(() => {
    if (!isUpdateAvailable) return false;
    const until = dismissedUntilRef.current;
    if (!until) return true;
    return Date.now() >= until;
  }, [isUpdateAvailable]);

  const dismiss = useCallback((delayMs = DISMISS_DELAY_MS) => {
    dismissedUntilRef.current = Date.now() + delayMs;
    setIsUpdateAvailable(false);
  }, []);

  const applyUpdate = useCallback(async () => {
    // If there's a waiting service worker, tell it to skipWaiting and then reload when controller changes.
    const reg = registrationRef.current;
    if (reg?.waiting) {
      try {
        // Try to message the worker to skip waiting first
        reg.waiting?.postMessage?.({ type: 'SKIP_WAITING' });
      } catch {}
      try {
        // Some TS DOM typings may not expose skipWaiting on the ServiceWorker instance;
        // use an unknown cast and narrow to a shape that optionally implements skipWaiting.
        const anyWaiting = reg.waiting as unknown;
        const waitingWithSkip = anyWaiting as
          | { skipWaiting?: () => Promise<void> }
          | undefined;
        if (
          waitingWithSkip &&
          typeof waitingWithSkip.skipWaiting === 'function'
        ) {
          await waitingWithSkip.skipWaiting();
        }
      } catch {}

      // Wait for controllerchange
      const promise = new Promise<void>(resolve => {
        const onControllerChange = () => {
          navigator.serviceWorker.removeEventListener(
            'controllerchange',
            onControllerChange
          );
          resolve();
        };
        navigator.serviceWorker.addEventListener(
          'controllerchange',
          onControllerChange
        );
        // Fallback timeout: if controllerchange doesn't fire, resolve after short delay
        setTimeout(resolve, 3000);
      });
      await promise;
      window.location.reload();
      return;
    }

    // No SW waiting — just perform a full reload.
    window.location.reload();
  }, []);

  const checkVersionJson = useCallback(async () => {
    try {
      const res = await fetch(VERSION_JSON_PATH, { cache: 'no-cache' });
      if (!res.ok) return;
      const data = (await res.json()) as
        | { frontend?: VersionInfo }
        | VersionInfo;
      const remote = (data as { frontend?: VersionInfo }).frontend
        ? (data as { frontend?: VersionInfo }).frontend
        : (data as VersionInfo);
      const local = getLocalVersion();
      if (!remote?.commitHash) return;
      const remoteHash = (remote?.commitHash ?? '').substring(0, 8);
      const localHash = (local?.commitHash ?? '').substring(0, 8);
      if (remoteHash && localHash && remoteHash !== localHash) {
        dismissedUntilRef.current = null;
        setIsUpdateAvailable(true);
      }
    } catch {
      // ignore
    }
  }, [getLocalVersion]);

  const setupServiceWorkerListeners = useCallback(async () => {
    if (!('serviceWorker' in navigator)) return;
    try {
      const reg = await navigator.serviceWorker.getRegistration();
      if (!reg) return;
      registrationRef.current = reg;

      // If there's already a waiting worker, an update is available
      if (reg.waiting) {
        dismissedUntilRef.current = null;
        setIsUpdateAvailable(true);
      }

      reg.addEventListener('updatefound', () => {
        const newWorker = reg.installing;
        if (!newWorker) return;
        newWorker.addEventListener('statechange', () => {
          if (
            newWorker.state === 'installed' &&
            navigator.serviceWorker.controller
          ) {
            dismissedUntilRef.current = null;
            setIsUpdateAvailable(true);
          }
        });
      });
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    // Start: listen to visibility/focus to trigger checks
    const onVisibility = () => {
      if (document.visibilityState === 'visible') {
        // Prefer SW-driven flow, but also check JSON in case no SW
        checkVersionJson();
      }
    };
    const onFocus = () => checkVersionJson();

    document.addEventListener('visibilitychange', onVisibility);
    window.addEventListener('focus', onFocus);

    // Setup SW listeners if available
    setupServiceWorkerListeners();

    // Start polling for version.json as fallback
    checkVersionJson();
    pollingTimerRef.current = window.setInterval(() => {
      checkVersionJson();
    }, POLL_INTERVAL_MS) as unknown as number;

    return () => {
      if (pollingTimerRef.current) {
        clearInterval(pollingTimerRef.current);
      }
      document.removeEventListener('visibilitychange', onVisibility);
      window.removeEventListener('focus', onFocus);
    };
  }, [checkVersionJson, setupServiceWorkerListeners]);

  return {
    isUpdateAvailable: shouldShow(),
    applyUpdate,
    dismiss,
  } as const;
}

export default useVersionCheck;
