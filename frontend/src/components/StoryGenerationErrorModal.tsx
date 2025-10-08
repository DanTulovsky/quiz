import React from 'react';
import { Modal, Button, Group, Text, Stack, Alert } from '@mantine/core';
import { IconAlertCircle } from '@tabler/icons-react';

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
  // For now, just use the error message as-is
  // The error parsing should happen in the useStory hook
  const actualErrorMessage = errorMessage;

  const getErrorDetails = () => {
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
          color='red'
          title={errorDetails.title}
          icon={errorDetails.icon}
        >
          {actualErrorMessage}
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
