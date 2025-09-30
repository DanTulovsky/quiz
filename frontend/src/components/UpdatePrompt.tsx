import React from 'react';
import { Button, Group, Text, Box } from '@mantine/core';
import { IconRefresh } from '@tabler/icons-react';
import useVersionCheck from '@/hooks/useVersionCheck';

const UpdatePrompt: React.FC = () => {
  const { isUpdateAvailable, applyUpdate, dismiss } = useVersionCheck();

  if (!isUpdateAvailable) return null;

  return (
    <Box
      role='status'
      aria-live='polite'
      style={{
        position: 'fixed',
        bottom: 16,
        left: 16,
        zIndex: 9999,
        background: 'white',
        border: '1px solid rgba(0,0,0,0.08)',
        padding: 12,
        borderRadius: 6,
        boxShadow: '0 6px 18px rgba(0,0,0,0.08)',
      }}
      data-testid='update-prompt'
    >
      <Group spacing={12} align='center'>
        <IconRefresh />
        <Text>A new version of the app is available.</Text>
        <Button size='xs' onClick={() => applyUpdate()} aria-label='Reload now'>
          Reload now
        </Button>
        <Button
          size='xs'
          variant='default'
          onClick={() => dismiss()}
          aria-label='Later'
        >
          Later
        </Button>
      </Group>
    </Box>
  );
};

export default UpdatePrompt;
