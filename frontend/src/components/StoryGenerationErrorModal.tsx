import React from 'react';
import {
  Modal,
  Button,
  Group,
  Text,
  Stack,
  Alert,
  Tooltip,
  Box,
} from '@mantine/core';
import { IconAlertCircle, IconInfoCircle } from '@tabler/icons-react';
import { ErrorResponse } from '../api/api';

interface StoryGenerationErrorModalProps {
  isOpen: boolean;
  onClose: () => void;
  errorMessage: string;
  errorDetails?: ErrorResponse;
}

const StoryGenerationErrorModal: React.FC<StoryGenerationErrorModalProps> = ({
  isOpen,
  onClose,
  errorMessage,
  errorDetails,
}) => {
  // Use the provided error details or fall back to parsing the error message
  const errorResponse = errorDetails;

  // Use structured error response if available, otherwise fall back to the original message
  const displayErrorMessage =
    errorResponse?.message || errorResponse?.error || errorMessage;

  // Format detailed error information for the tooltip
  const getDetailedErrorInfo = () => {
    if (!errorResponse || !errorResponse.details) {
      return null;
    }

    return errorResponse.details;
  };

  const detailedErrorInfo = getDetailedErrorInfo();

  const getErrorConfig = () => {
    return {
      title: 'Cannot Generate Section',
      icon: <IconAlertCircle size={20} color='var(--mantine-color-error)' />,
      variant: 'filled' as const,
      description:
        'There was an issue generating a new section for your story.',
      suggestion: 'Please check your story status and try again.',
    };
  };

  const errorConfig = getErrorConfig();

  return (
    <Modal
      opened={isOpen}
      onClose={onClose}
      title={
        <Group gap='sm'>
          {errorConfig.icon}
          <Text fw={500} size='lg'>
            {errorConfig.title}
          </Text>
        </Group>
      }
      centered
      size='md'
    >
      <Stack gap='md'>
        <Alert
          variant={errorConfig.variant}
          color='red'
          title={errorConfig.title}
          icon={errorConfig.icon}
        >
          <Group gap='xs' align='flex-start'>
            <Text 
              style={{ 
                wordBreak: 'break-word', 
                whiteSpace: 'pre-wrap',
                overflowWrap: 'break-word',
                maxWidth: '100%'
              }}
            >
              {displayErrorMessage}
            </Text>
            {detailedErrorInfo && (
              <Tooltip label={detailedErrorInfo} multiline withArrow w={400}>
                <Box>
                  <IconInfoCircle
                    size={16}
                    color='var(--mantine-color-blue-6)'
                    style={{ cursor: 'help' }}
                  />
                </Box>
              </Tooltip>
            )}
          </Group>
        </Alert>

        <Text size='sm' c='dimmed'>
          {errorConfig.suggestion}
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

export default StoryGenerationErrorModal;
