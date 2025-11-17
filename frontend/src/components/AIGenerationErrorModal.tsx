import React from 'react';
import { Modal, Button, Group, Text, Stack, Alert, Paper } from '@mantine/core';
import { IconAlertCircle } from '@tabler/icons-react';
import { ErrorResponse } from '../api/api';

interface AIGenerationErrorModalProps {
  isOpen: boolean;
  onClose: () => void;
  errorMessage: string;
  errorCause?: string;
  errorDetails?: ErrorResponse;
}

const AIGenerationErrorModal: React.FC<AIGenerationErrorModalProps> = ({
  isOpen,
  onClose,
  errorMessage,
  errorCause,
  errorDetails,
}) => {
  // Use the provided error details or fall back to parsing the error message
  const errorResponse = errorDetails;

  // Use structured error response if available, otherwise fall back to the original message
  const displayErrorMessage =
    errorResponse?.message || errorResponse?.error || errorMessage;

  // Get the cause - prioritize the explicit cause prop, then check errorDetails, then fall back to details
  const displayCause =
    errorCause || errorResponse?.cause || errorResponse?.details;

  return (
    <Modal
      opened={isOpen}
      onClose={onClose}
      title={
        <Group gap='sm'>
          <IconAlertCircle size={20} color='var(--mantine-color-error)' />
          <Text fw={500} size='lg'>
            AI Generation Error
          </Text>
        </Group>
      }
      centered
      size='md'
    >
      <Stack gap='md'>
        <Alert
          variant='filled'
          color='red'
          title='Failed to generate'
          icon={<IconAlertCircle size={20} />}
        >
          <Text
            style={{
              wordBreak: 'break-word',
              whiteSpace: 'pre-wrap',
              overflowWrap: 'break-word',
              maxWidth: '100%',
            }}
          >
            {displayErrorMessage}
          </Text>
        </Alert>

        {displayCause && (
          <Paper p='md' withBorder style={{ backgroundColor: '#f8f9fa' }}>
            <Text size='sm' fw={500} mb='xs'>
              Error Details:
            </Text>
            <Text
              size='sm'
              style={{
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                fontFamily: 'monospace',
                color: 'var(--mantine-color-gray-7)',
              }}
            >
              {displayCause}
            </Text>
          </Paper>
        )}

        <Text size='sm' c='dimmed'>
          Please try again later. If the problem persists, check your AI
          provider settings.
        </Text>

        {errorResponse?.retryable === false && (
          <Alert
            color='orange'
            variant='light'
            icon={<IconAlertCircle size={16} />}
          >
            <Text size='sm'>
              This error cannot be automatically retried. Please wait a moment
              and try again manually.
            </Text>
          </Alert>
        )}

        <Group justify='flex-end' mt='md'>
          <Button onClick={onClose}>Close</Button>
        </Group>
      </Stack>
    </Modal>
  );
};

export default AIGenerationErrorModal;
