import React, { useState, useEffect } from 'react';
import {
  Modal,
  Textarea,
  Select,
  Button,
  Stack,
  Text,
  Checkbox,
  Group,
  Image,
  Alert,
  LoadingOverlay,
} from '@mantine/core';
import { IconBug, IconCheck, IconAlertCircle } from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';
import { usePostV1Feedback } from '../api/api';
import html2canvas from 'html2canvas';
import ScreenshotAnnotation from './ScreenshotAnnotation';

interface FeedbackModalProps {
  opened: boolean;
  onClose: () => void;
}

type FeedbackType = 'bug' | 'feature_request' | 'general' | 'improvement';

interface ContextData {
  page_url: string;
  page_title: string;
  language?: string;
  level?: string;
  question_id?: number;
  story_id?: number;
  viewport_width: number;
  viewport_height: number;
  user_agent: string;
  timestamp: string;
  [key: string]: unknown;
}

const FeedbackModal: React.FC<FeedbackModalProps> = ({ opened, onClose }) => {
  const { user } = useAuth();
  const [feedbackText, setFeedbackText] = useState('');
  const [feedbackType, setFeedbackType] = useState<FeedbackType>('general');
  const [includeScreenshot, setIncludeScreenshot] = useState(false);
  const [screenshotData, setScreenshotData] = useState<string | null>(null);
  const [capturingScreenshot, setCapturingScreenshot] = useState(false);
  const [isAnnotating, setIsAnnotating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const { mutate: submitFeedback, isPending, reset } = usePostV1Feedback();

  // Reset form when modal closes
  useEffect(() => {
    if (!opened) {
      setFeedbackText('');
      setFeedbackType('general');
      setIncludeScreenshot(false);
      setScreenshotData(null);
      setIsAnnotating(false);
      setError(null);
      setSuccess(false);
      reset();
    }
  }, [opened, reset]);

  // Auto-capture screenshot when checkbox is checked
  useEffect(() => {
    if (includeScreenshot && opened && !screenshotData) {
      captureScreenshot();
    }
  }, [includeScreenshot, opened]);

  const captureContext = (): ContextData => {
    const context: ContextData = {
      page_url: window.location.pathname + window.location.search,
      page_title: document.title,
      viewport_width: window.innerWidth,
      viewport_height: window.innerHeight,
      user_agent: navigator.userAgent,
      timestamp: new Date().toISOString(),
    };

    // Add user-specific context if available
    if (user?.preferred_language) {
      context.language = user.preferred_language;
    }
    if (user?.current_level) {
      context.level = user.current_level;
    }

    // Try to extract question_id or story_id from URL
    const pathMatch = window.location.pathname.match(/\/(\d+)/);
    if (pathMatch) {
      const id = parseInt(pathMatch[1], 10);
      if (window.location.pathname.includes('/question/')) {
        context.question_id = id;
      } else if (window.location.pathname.includes('/story/')) {
        context.story_id = id;
      }
    }

    return context;
  };

  const captureScreenshot = async () => {
    setCapturingScreenshot(true);

    try {
      // Find all elements with role="dialog" (modals) and temporarily hide them
      const modals = document.querySelectorAll('[role="dialog"]');
      const modalStyles: { element: HTMLElement; originalDisplay: string }[] =
        [];

      modals.forEach(modal => {
        if (modal instanceof HTMLElement) {
          modalStyles.push({
            element: modal,
            originalDisplay: modal.style.display || '',
          });
          modal.style.display = 'none';
        }
      });

      // Wait for the DOM to update
      await new Promise(resolve => setTimeout(resolve, 100));

      // Capture the screenshot
      const canvas = await html2canvas(document.body, {
        logging: false,
        useCORS: true,
        allowTaint: true,
        scale: 0.5,
        backgroundColor: '#ffffff',
      });
      const dataUrl = canvas.toDataURL('image/jpeg', 0.7);
      setScreenshotData(dataUrl);

      // Restore modal visibility
      modalStyles.forEach(({ element, originalDisplay }) => {
        element.style.display = originalDisplay;
      });
    } catch (err) {
      console.error('Screenshot capture failed:', err);
      setError(
        'Failed to capture screenshot. You can still submit feedback without it.'
      );
    } finally {
      setCapturingScreenshot(false);
    }
  };

  const handleSubmit = () => {
    setError(null);

    if (!feedbackText.trim()) {
      setError('Please enter your feedback');
      return;
    }

    if (feedbackText.length > 5000) {
      setError('Feedback text must be 5000 characters or less');
      return;
    }

    const context = captureContext();

    submitFeedback(
      {
        data: {
          feedback_text: feedbackText,
          feedback_type: feedbackType,
          context_data: context,
          screenshot_data: screenshotData || undefined,
        },
      },
      {
        onSuccess: () => {
          setSuccess(true);
          setTimeout(() => {
            onClose();
          }, 1500);
        },
        onError: (err: unknown) => {
          const errorMessage =
            err && typeof err === 'object' && 'response' in err
              ? (err as { response?: { data?: { message?: string } } }).response
                  ?.data?.message
              : undefined;
          setError(
            errorMessage || 'Failed to submit feedback. Please try again.'
          );
        },
      }
    );
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      handleSubmit();
    }
  };

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={
        <Group gap='xs'>
          <IconBug size={24} />
          <Text size='lg' fw={600}>
            Report Issue or Feedback
          </Text>
        </Group>
      }
      size='lg'
      closeOnClickOutside={!isPending}
      closeOnEscape={!isPending}
      data-html2canvas-ignore
    >
      <LoadingOverlay visible={isPending} />
      <Stack gap='md'>
        {success && (
          <Alert
            icon={<IconCheck size={16} />}
            title='Thank you!'
            color='green'
            onClose={() => setSuccess(false)}
            withCloseButton
          >
            Your feedback has been submitted successfully. We appreciate your
            help improving the app!
          </Alert>
        )}

        {error && (
          <Alert
            icon={<IconAlertCircle size={16} />}
            title='Error'
            color='red'
            onClose={() => setError(null)}
            withCloseButton
          >
            {error}
          </Alert>
        )}

        <Textarea
          label='Feedback'
          description='Please describe the issue, feature request, or provide general feedback'
          placeholder='Enter your feedback here...'
          value={feedbackText}
          onChange={e => setFeedbackText(e.target.value)}
          minRows={6}
          maxLength={5000}
          required
          onKeyDown={handleKeyDown}
          disabled={isPending}
        />

        <Select
          label='Type'
          description='Select the type of feedback'
          data={[
            { value: 'bug', label: 'Bug Report' },
            { value: 'feature_request', label: 'Feature Request' },
            { value: 'general', label: 'General Feedback' },
            { value: 'improvement', label: 'Improvement Suggestion' },
          ]}
          value={feedbackType}
          onChange={value => setFeedbackType(value as FeedbackType)}
          disabled={isPending}
        />

        <Stack gap='xs'>
          <Checkbox
            label='Include screenshot'
            description='Captures a screenshot of the current page to help us understand the issue'
            checked={includeScreenshot}
            onChange={e => setIncludeScreenshot(e.target.checked)}
            disabled={isPending || capturingScreenshot}
          />

          {capturingScreenshot && (
            <Text size='sm' c='dimmed'>
              Capturing screenshot...
            </Text>
          )}

          {screenshotData && (
            <Stack gap='xs'>
              <Text size='sm' fw={500}>
                Screenshot Preview
              </Text>
              <Image
                src={screenshotData}
                alt='Screenshot preview'
                maw={400}
                mx='auto'
                radius='md'
              />
              <Group justify='center'>
                <Button
                  size='xs'
                  variant='light'
                  onClick={captureScreenshot}
                  loading={capturingScreenshot}
                  disabled={isPending}
                >
                  Recapture
                </Button>
                <Button
                  size='xs'
                  variant='light'
                  onClick={() => setIsAnnotating(true)}
                  disabled={isPending}
                >
                  Annotate
                </Button>
                <Button
                  size='xs'
                  variant='light'
                  color='red'
                  onClick={() => {
                    setScreenshotData(null);
                    setIncludeScreenshot(false);
                  }}
                  disabled={isPending}
                >
                  Remove
                </Button>
              </Group>
            </Stack>
          )}
        </Stack>

        <Text size='xs' c='dimmed'>
          Tip: Press Ctrl+Enter to submit
        </Text>

        <Group justify='flex-end' mt='md'>
          <Button variant='subtle' onClick={onClose} disabled={isPending}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            loading={isPending}
            leftSection={<IconBug />}
          >
            Submit Feedback
          </Button>
        </Group>
      </Stack>

      {/* Screenshot Annotation Modal */}
      {isAnnotating && screenshotData && (
        <ScreenshotAnnotation
          screenshotData={screenshotData}
          onSave={annotatedData => {
            setScreenshotData(annotatedData);
            setIsAnnotating(false);
          }}
          onCancel={() => setIsAnnotating(false)}
        />
      )}
    </Modal>
  );
};

export default FeedbackModal;
