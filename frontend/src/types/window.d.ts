declare global {
  interface Window {
    __ttsStateCheckInterval?: number | NodeJS.Timeout;
    __ttsCleanup?: () => void;
  }
}

export {};


