import React from 'react';
import {
  Modal,
  Text,
  Stack,
  Title,
  Divider,
  List,
  Group,
  Button,
  Alert,
} from '@mantine/core';
import {
  IconBrain,
  IconFileText,
  IconCalendar,
  IconBook,
  IconAlertCircle,
} from '@tabler/icons-react';

interface HelpModalProps {
  opened: boolean;
  onClose: () => void;
}

const HelpModal: React.FC<HelpModalProps> = ({ opened, onClose }) => {
  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        <Group gap='sm'>
          <IconBrain size={24} />
          <Title order={3}>Quiz App Help</Title>
        </Group>
      }
      size='lg'
      centered
    >
      <Stack gap='md'>
        <Text size='sm' c='dimmed'>
          Welcome to your personalized Italian language learning system! This
          adaptive quiz platform helps you practice and improve your skills
          through AI-generated exercises.
        </Text>

        <Alert
          variant='light'
          color='blue'
          title='Important: Enable AI First'
          icon={<IconAlertCircle size={16} />}
          mt='sm'
        >
          <Text size='sm'>
            <strong>AI must be enabled in Settings</strong> before any sections
            (Quiz, Reading Comprehension, Daily Practice, or Stories) will work.
            The system relies entirely on AI-generated content tailored to your
            skill level.
          </Text>
        </Alert>

        <Divider />

        <Stack gap='sm'>
          <Title order={4}>How to Use the System</Title>
          <List size='sm' spacing='xs'>
            <List.Item icon={<IconBrain size={16} />}>
              <strong>Quiz Section:</strong> Take interactive multiple-choice
              questions adapted to your level
            </List.Item>
            <List.Item icon={<IconFileText size={16} />}>
              <strong>Reading Comprehension:</strong> Practice reading passages
              with comprehension questions
            </List.Item>
            <List.Item icon={<IconCalendar size={16} />}>
              <strong>Daily Practice:</strong> Complete daily exercises to
              maintain consistent progress
            </List.Item>
            <List.Item icon={<IconBook size={16} />}>
              <strong>Story Mode:</strong> Read engaging stories written for
              your current proficiency level
            </List.Item>
          </List>
        </Stack>

        <Divider />

        <Stack gap='sm'>
          <Title order={4}>Navigation Guide</Title>
          <List size='sm' spacing='xs'>
            <List.Item>
              <strong>Settings:</strong> Enable AI and configure your learning
              preferences
            </List.Item>
            <List.Item>
              <strong>Progress:</strong> View your learning statistics and track
              improvement over time
            </List.Item>
            <List.Item>
              <strong>Keyboard Shortcuts:</strong> Use Shift+1-5 to quickly
              navigate between sections
            </List.Item>
            <List.Item>
              <strong>Help:</strong> Click the ? icon anytime for assistance
            </List.Item>
          </List>
        </Stack>

        <Group justify='flex-end' mt='md'>
          <Button onClick={onClose} variant='filled'>
            Got it!
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
};

export default HelpModal;
