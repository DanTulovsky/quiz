import React from 'react';
import { Modal, Button, Group, Text, Stack, Alert } from '@mantine/core';
import { IconAlertCircle, IconInfoCircle } from '@tabler/icons-react';

interface StoryGenerationErrorModalProps {
  isOpen: boolean;
  onClose: () => void;
  errorMessage: string;
}

const StoryGenerationErrorModal: React.FC<StoryGenerationErrorModalProps> = ({
  isOpen,
  onClose,
  errorMessage,
}) => {
  // Determine if this is a limit exceeded error or other error
  const isLimitError =
    errorMessage.includes('daily generation limit reached') ||
    errorMessage.includes('already generated') ||
    errorMessage.includes('try again tomorrow');

  const getErrorDetails = () => {
    if (isLimitError) {
      return {
        title: 'Daily Generation Limit Reached',
        icon: <IconInfoCircle size={20} color='var(--mantine-color-blue)' />,
        variant: 'light' as const,
        description:
          'You can generate up to 2 sections per day for each story. The daily limit resets at midnight.',
        suggestion:
          'Try again tomorrow, or work on a different story in the meantime.',
      };
    }
    return {
      title: 'Cannot Generate Section',
      icon: <IconAlertCircle size={20} color='var(--mantine-color-error)' />,
      variant: 'filled' as const,
      description:
        'There was an issue generating a new section for your story.',
      suggestion: 'Please check your story status and try again.',
    };
  };

  const errorDetails = getErrorDetails();

  return (
    <Modal
      opened={isOpen}
      onClose={onClose}
      title={
        <Group gap='sm'>
          {errorDetails.icon}
          <Text fw={500} size='lg'>
            {errorDetails.title}
          </Text>
        </Group>
      }
      centered
      size='md'
    >
      <Stack gap='md'>
        <Alert
          variant={errorDetails.variant}
          color={isLimitError ? 'blue' : 'red'}
          title={errorMessage}
          icon={errorDetails.icon}
        >
          {errorDetails.description}
        </Alert>

        <Text size='sm' c='dimmed'>
          {errorDetails.suggestion}
        </Text>

        <Group justify='flex-end' mt='md'>
          <Button onClick={onClose}>Close</Button>
        </Group>
      </Stack>
    </Modal>
  );
};

export default StoryGenerationErrorModal;
