import React from 'react';
import { Modal, Button, Group, Text } from '@mantine/core';
import { IconAlertCircle } from '@tabler/icons-react';

interface ErrorModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  message: string;
}

const ErrorModal: React.FC<ErrorModalProps> = ({
  isOpen,
  onClose,
  title,
  message,
}) => {
  return (
    <Modal
      opened={isOpen}
      onClose={onClose}
      title={
        <Group gap='sm'>
          <IconAlertCircle size={20} color='var(--mantine-color-error)' />
          <Text fw={500} size='lg'>
            {title}
          </Text>
        </Group>
      }
      centered
      size='md'
    >
      <Text
        size='sm'
        c='dimmed'
        style={{
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          overflowWrap: 'break-word',
          maxWidth: '100%',
        }}
      >
        {message}
      </Text>

      <Group justify='flex-end' mt='xl'>
        <Button onClick={onClose}>Close</Button>
      </Group>
    </Modal>
  );
};

export default ErrorModal;
