import React from 'react';
import { Modal, Button, Group, Text, Badge } from '@mantine/core';
import { IconAlertCircle } from '@tabler/icons-react';
import { useHotkeys } from 'react-hotkeys-hook';

interface ConfirmationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
}

const ConfirmationModal: React.FC<ConfirmationModalProps> = ({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = 'OK',
  cancelText = 'Cancel',
}) => {
  // Handle ESC key to cancel
  useHotkeys(
    'escape',
    e => {
      if (isOpen) {
        e.preventDefault();
        onClose();
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  // Handle Enter key to confirm
  useHotkeys(
    'enter',
    e => {
      if (isOpen) {
        e.preventDefault();
        handleConfirm();
      }
    },
    { enableOnFormTags: false, preventDefault: true }
  );

  const handleConfirm = () => {
    onConfirm();
    onClose();
  };

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
        style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}
      >
        {message}
      </Text>

      <Group justify='flex-end' mt='xl'>
        <Button variant='outline' onClick={onClose}>
          {cancelText}{' '}
          <Badge ml={6} size='xs' color='gray' variant='filled' radius='sm'>
            ESC
          </Badge>
        </Button>
        <Button onClick={handleConfirm}>
          {confirmText}{' '}
          <Badge ml={6} size='xs' color='gray' variant='filled' radius='sm'>
            ↵
          </Badge>
        </Button>
      </Group>
    </Modal>
  );
};

export default ConfirmationModal;
