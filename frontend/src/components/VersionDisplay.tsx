import React, { useEffect, useState } from 'react';
import {
  Text,
  Popover,
  Loader,
  Stack,
  Group,
  Badge,
  useMantineTheme,
  useMantineColorScheme,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { getAppVersion, formatVersion } from '../utils/version';

interface ServiceVersion {
  service: string;
  version: string;
  commit: string;
  buildTime: string;
  error?: string;
}

interface AggregatedVersion {
  backend: ServiceVersion;
  worker: ServiceVersion;
}

const fetchAggregatedVersion = async (): Promise<AggregatedVersion | null> => {
  try {
    const res = await fetch('/v1/version');
    if (!res.ok) return null;
    const data = await res.json();
    return data;
  } catch {
    return null;
  }
};

// Helper to convert ServiceVersion to AppVersion for formatting
function toAppVersion(sv: ServiceVersion): {
  version: string;
  buildTime: string;
  commitHash: string;
} {
  return {
    version: sv.version,
    buildTime: sv.buildTime,
    commitHash: sv.commit || '',
  };
}

interface VersionDisplayProps {
  /**
   * When true (default), clicking copies the version JSON to clipboard.
   * For mobile hamburger menu, we disable this.
   */
  copyOnClick?: boolean;
  /**
   * Whether the component should be `fixed` (bottom-left of viewport) or rely on normal flow (`static`).
   * Defaults to `fixed`.
   */
  position?: 'fixed' | 'static';
}

const VersionDisplay: React.FC<VersionDisplayProps> = ({
  copyOnClick = true,
  position = 'fixed',
}) => {
  const frontend = getAppVersion();
  const theme = useMantineTheme();
  const { colorScheme } = useMantineColorScheme();
  const [backend, setBackend] = useState<ServiceVersion | null | undefined>();
  const [worker, setWorker] = useState<ServiceVersion | null | undefined>();

  // Control tooltip manually for mobile where we disable copy
  const [opened, { close, open }] = useDisclosure(false);

  useEffect(() => {
    let cancelled = false;
    setBackend(undefined);
    setWorker(undefined);
    fetchAggregatedVersion().then(data => {
      if (cancelled) return;
      if (!data) {
        setBackend(null);
        setWorker(null);
      } else {
        setBackend(data.backend as ServiceVersion);
        setWorker(data.worker as ServiceVersion);
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const handleVersionClick = async () => {
    // Prepare version data for JSON export
    const versionData = {
      frontend: {
        service: 'frontend',
        version: frontend.version,
        commit: frontend.commitHash,
        buildTime: frontend.buildTime,
        formatted: formatVersion(frontend),
      },
      backend:
        backend === null
          ? { service: 'backend', status: 'unavailable' }
          : backend?.error
            ? { service: 'backend', status: 'error', error: backend.error }
            : backend
              ? {
                  service: 'backend',
                  version: backend.version,
                  commit: backend.commit,
                  buildTime: backend.buildTime,
                  formatted: formatVersion(toAppVersion(backend)),
                }
              : { service: 'backend', status: 'loading' },
      worker:
        worker === null
          ? { service: 'worker', status: 'unavailable' }
          : worker?.error
            ? { service: 'worker', status: 'error', error: worker.error }
            : worker
              ? {
                  service: 'worker',
                  version: worker.version,
                  commit: worker.commit,
                  buildTime: worker.buildTime,
                  formatted: formatVersion(toAppVersion(worker)),
                }
              : { service: 'worker', status: 'loading' },
      timestamp: new Date().toISOString(),
    };

    try {
      await navigator.clipboard.writeText(JSON.stringify(versionData, null, 2));
      notifications.show({
        title: 'Version info copied!',
        message: 'All version information has been copied to clipboard',
        color: 'green',
        autoClose: 2000,
      });
    } catch {
      notifications.show({
        title: 'Copy failed',
        message: 'Failed to copy version information to clipboard',
        color: 'red',
        autoClose: 3000,
      });
    }
  };

  const infoContent = (
    <Stack gap={4} style={{ minWidth: 260 }}>
      <Group gap={8}>
        <Badge size='xs'>frontend</Badge>
        <b>{formatVersion(frontend)}</b>
      </Group>
      <Text size='xs' c='dimmed'>
        Build: {frontend.buildTime}
      </Text>
      <Text size='xs' c='dimmed'>
        Commit: {frontend.commitHash}
      </Text>
      <Group gap={8} mt={8}>
        <Badge size='xs'>backend</Badge>
        {backend === undefined ? (
          <Loader size='xs' />
        ) : backend === null ? (
          <Text size='xs' c='red'>
            unavailable
          </Text>
        ) : backend.error ? (
          <Text size='xs' c='red'>
            {backend.error}
          </Text>
        ) : (
          <b>{formatVersion(toAppVersion(backend))}</b>
        )}
      </Group>
      {backend && !backend.error && (
        <>
          <Text size='xs' c='dimmed'>
            Build: {backend.buildTime}
          </Text>
          <Text size='xs' c='dimmed'>
            Commit: {backend.commit}
          </Text>
        </>
      )}
      <Group gap={8} mt={8}>
        <Badge size='xs'>worker</Badge>
        {worker === undefined ? (
          <Loader size='xs' />
        ) : worker === null ? (
          <Text size='xs' c='red'>
            unavailable
          </Text>
        ) : worker.error ? (
          <Text size='xs' c='red'>
            {worker.error}
          </Text>
        ) : (
          <b>{formatVersion(toAppVersion(worker))}</b>
        )}
      </Group>
      {worker && !worker.error && (
        <>
          <Text size='xs' c='dimmed'>
            Build: {worker.buildTime}
          </Text>
          <Text size='xs' c='dimmed'>
            Commit: {worker.commit}
          </Text>
        </>
      )}
    </Stack>
  );

  const onTextClick: React.MouseEventHandler = e => {
    e.stopPropagation();
    e.preventDefault();

    if (!opened) {
      open();
    }
  };

  // Desktop / default behaviour (copy on click + popover for better theming)
  if (copyOnClick) {
    return (
      <Popover opened={opened} withArrow shadow='md' width='auto'>
        <Popover.Target>
          <Text
            size='xs'
            c='dimmed'
            style={{
              position,
              ...(position === 'fixed'
                ? { bottom: '8px', left: '8px', zIndex: 1000 }
                : {}),
              userSelect: 'none',
              pointerEvents: 'auto',
              cursor: 'pointer',
            }}
            data-testid='app-version'
            onClick={onTextClick}
          >
            {formatVersion(frontend)}
          </Text>
        </Popover.Target>
        <Popover.Dropdown
          style={{
            backgroundColor:
              colorScheme === 'dark' ? theme.colors.dark[6] : theme.white,
            cursor: 'pointer',
          }}
          onClick={() => {
            handleVersionClick();
            close();
          }}
        >
          {infoContent}
          <Text size='xs' c='dimmed' mt={8} ta='center'>
            Click to copy all version info
          </Text>
        </Popover.Dropdown>
      </Popover>
    );
  }

  // Mobile / popover behaviour
  return (
    <Popover opened={opened} withArrow shadow='md' width='auto'>
      <Popover.Target>
        <Text
          size='xs'
          c='dimmed'
          style={{ cursor: 'pointer' }}
          onClick={onTextClick}
        >
          {formatVersion(frontend)}
        </Text>
      </Popover.Target>
      <Popover.Dropdown
        onClick={() => {
          handleVersionClick();
          close();
        }}
        style={{ cursor: 'pointer' }}
      >
        {infoContent}
        <Text size='xs' c='dimmed' mt={8} ta='center'>
          Tap to copy all version info
        </Text>
      </Popover.Dropdown>
    </Popover>
  );
};

export default VersionDisplay;
