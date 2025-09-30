// Simple environment-aware logger for the frontend
// - No-ops in production unless VITE_DEBUG=true
// - In development, proxies to console

type LogMethod = (...args: unknown[]) => void;

const isDebugEnabled = (): boolean => {
  const dev = import.meta.env?.DEV === true;
  const debugFlag = import.meta.env?.VITE_DEBUG;
  return dev || String(debugFlag).toLowerCase() === 'true';
};

const makeMethod = (
  method: 'log' | 'info' | 'warn' | 'error' | 'debug'
): LogMethod => {
  return (...args: unknown[]) => {
    if (!isDebugEnabled()) return;
    (console[method] as (...args: unknown[]) => void)(...args);
  };
};

export const logger = {
  log: makeMethod('log'),
  info: makeMethod('info'),
  warn: makeMethod('warn'),
  error: makeMethod('error'),
  debug: makeMethod('debug'),
};

export default logger;
