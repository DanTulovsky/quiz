import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { useWordOfTheDay } from '../hooks/useWordOfTheDay';
import {
  Container,
  Stack,
  Text,
  Center,
  Paper,
  Button,
  Group,
  Badge,
  Title,
  Card,
  ThemeIcon,
  Modal,
  CopyButton,
  Tooltip,
  Textarea,
  Code,
  ActionIcon,
  PasswordInput,
} from '@mantine/core';
import {
  ChevronLeft,
  ChevronRight,
  Calendar,
  Copy,
  Check,
  Link as LinkIcon,
  TerminalSquare,
  Info,
} from 'lucide-react';
import LoadingSpinner from '../components/LoadingSpinner';

const WordOfTheDayPage: React.FC = () => {
  const { date: dateParam } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const [showEmbedModal, setShowEmbedModal] = React.useState(false);
  const [showApiModal, setShowApiModal] = React.useState(false);
  const [apiKey, setApiKey] = React.useState<string>('');

  const {
    selectedDate,
    setSelectedDate,
    word,
    isLoading,
    goToPreviousDay,
    goToNextDay,
    goToToday,
    canGoPrevious,
    canGoNext,
  } = useWordOfTheDay(dateParam);

  // Update URL when date changes
  React.useEffect(() => {
    if (dateParam !== selectedDate) {
      navigate(`/word-of-day/${selectedDate}`, { replace: true });
    }
  }, [selectedDate, dateParam, navigate]);

  // Format date for display
  const formatDisplayDate = (dateStr: string): string => {
    const date = new Date(dateStr + 'T00:00:00');
    return date.toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  };

  // Format date for date picker (YYYY-MM-DD)
  const formatDateForInput = (dateStr: string): string => {
    return dateStr;
  };

  // Handle date input change
  const handleDateChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newDate = e.target.value;
    if (newDate) {
      setSelectedDate(newDate);
    }
  };

  if (!user) {
    return (
      <Container size='md' py='xl'>
        <Center h='60vh'>
          <Text>Please log in to view your word of the day.</Text>
        </Center>
      </Container>
    );
  }

  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  const embedUrl = `${origin}/word-of-day/embed`;
  const iframeSnippet = `<iframe src="${embedUrl}" width="100%" height="300" style="border:none"></iframe>`;
  const apiUrlToday = `${origin}/v1/word-of-day`;
  const apiUrlWithDate = `${origin}/v1/word-of-day/${selectedDate}`;

  const authHeader = `-H 'Authorization: ${apiKey ? `Bearer ${apiKey}` : 'Bearer YOUR_API_KEY'}'`;

  const apiCurlApiKeyToday = `curl -i \
  '${apiUrlToday}' \
  -H 'Accept: application/json' \
  ${authHeader}`;

  const apiCurlApiKeyWithDate = `curl -i \
  '${apiUrlWithDate}' \
  -H 'Accept: application/json' \
  ${authHeader}`;

  return (
    <Container size='md' py='xl'>
      <Stack gap='lg'>
        {/* Header with date navigation */}
        <Group justify='space-between' align='center'>
          <Title order={2}>Word of the Day</Title>
          <Group gap='xs'>
            <Button
              variant='subtle'
              size='sm'
              leftSection={<Calendar size={16} />}
              onClick={goToToday}
            >
              Today
            </Button>
            <Tooltip label='Show iframe embed snippet' withArrow>
              <Button
                variant='light'
                size='xs'
                onClick={() => setShowEmbedModal(true)}
              >
                Embed
              </Button>
            </Tooltip>
            <Tooltip label='Show API request examples' withArrow>
              <Button
                variant='light'
                size='xs'
                onClick={() => setShowApiModal(true)}
              >
                API
              </Button>
            </Tooltip>
          </Group>
        </Group>

        {/* Date navigation */}
        <Group justify='center' align='center'>
          <Button
            variant='subtle'
            leftSection={<ChevronLeft size={20} />}
            onClick={goToPreviousDay}
            disabled={!canGoPrevious || isLoading}
          >
            Previous
          </Button>

          <input
            type='date'
            value={formatDateForInput(selectedDate)}
            onChange={handleDateChange}
            max={new Date().toISOString().split('T')[0]}
            style={{
              padding: '8px 12px',
              borderRadius: '4px',
              border: '1px solid #dee2e6',
              fontSize: '14px',
              cursor: 'pointer',
            }}
          />

          <Button
            variant='subtle'
            rightSection={<ChevronRight size={20} />}
            onClick={goToNextDay}
            disabled={!canGoNext || isLoading}
          >
            Next
          </Button>
        </Group>

        {/* Word display */}
        {isLoading ? (
          <Center h='60vh'>
            <LoadingSpinner />
          </Center>
        ) : word ? (
          <Card
            shadow='md'
            padding='xl'
            radius='md'
            style={{
              background: `var(--mantine-primary-color-0)`,
              border: `2px solid var(--mantine-primary-color-4)`,
              position: 'relative',
              overflow: 'visible',
              wordWrap: 'break-word',
            }}
          >
            <Stack gap='md'>
              {/* Date */}
              <Text
                size='sm'
                fw={600}
                c='dimmed'
                style={{ textTransform: 'uppercase', letterSpacing: '1px' }}
              >
                {formatDisplayDate(word.date)}
              </Text>

              {/* Word */}
              <Title
                order={1}
                style={{
                  lineHeight: 1.2,
                  fontSize: 'clamp(2rem, 5vw, 3.5rem)',
                  wordBreak: 'break-word',
                  overflowWrap: 'anywhere',
                  hyphens: 'auto',
                }}
                c='primary'
              >
                {word.word}
              </Title>

              {/* Translation */}
              <Text size='xl' c='primary' style={{ fontStyle: 'italic' }}>
                {word.translation}
              </Text>

              {/* Example sentence */}
              {word.sentence && (
                <Paper
                  p='md'
                  radius='md'
                  style={{
                    background: 'var(--mantine-color-body)',
                    borderLeft: '3px solid var(--mantine-primary-color-4)',
                  }}
                >
                  <Text
                    size='lg'
                    style={{ lineHeight: 1.8, fontStyle: 'italic' }}
                  >
                    {word.sentence}
                  </Text>
                </Paper>
              )}

              {/* Explanation */}
              {word.explanation && (
                <Paper
                  p='md'
                  radius='md'
                  style={{
                    background: 'var(--mantine-color-body)',
                    borderLeft: '3px solid var(--mantine-primary-color-4)',
                  }}
                >
                  <Text size='sm'>{word.explanation}</Text>
                </Paper>
              )}

              {/* Metadata badges */}
              <Group gap='xs' mt='md'>
                {word.language && (
                  <Badge size='lg' variant='light' color='primary'>
                    {word.language}
                  </Badge>
                )}
                {word.level && (
                  <Badge size='lg' variant='light' color='primary'>
                    {word.level}
                  </Badge>
                )}
                {word.source_type && (
                  <Badge size='lg' variant='light' color='primary'>
                    {word.source_type === 'vocabulary_question'
                      ? 'Vocabulary'
                      : 'Snippet'}
                  </Badge>
                )}
                {word.topic_category && (
                  <Badge size='lg' variant='light' color='primary'>
                    {word.topic_category}
                  </Badge>
                )}
              </Group>
            </Stack>
          </Card>
        ) : (
          <Center h='60vh'>
            <Stack align='center' gap='md'>
              <ThemeIcon size={64} radius='xl' color='gray' variant='light'>
                <Text size='xl'>📚</Text>
              </ThemeIcon>
              <Text size='lg' c='dimmed'>
                No word available for this date.
              </Text>
            </Stack>
          </Center>
        )}
        {/* Embed snippet modal */}
        <Modal
          opened={showEmbedModal}
          onClose={() => setShowEmbedModal(false)}
          title='Embed this Word of the Day'
          centered
        >
          <Stack gap='sm'>
            <Text size='sm'>Copy and paste this iframe into your page:</Text>
            <Textarea
              value={iframeSnippet}
              readOnly
              minRows={3}
              autosize
              styles={{ input: { fontFamily: 'monospace' } }}
            />
            <Group justify='space-between' align='center'>
              <Code>{embedUrl}</Code>
              <CopyButton value={iframeSnippet} timeout={1500}>
                {({ copied, copy }) => (
                  <Button
                    onClick={copy}
                    size='xs'
                    color={copied ? 'green' : 'primary'}
                  >
                    {copied ? 'Copied' : 'Copy iframe'}
                  </Button>
                )}
              </CopyButton>
            </Group>
          </Stack>
        </Modal>

        {/* API examples modal */}
        <Modal
          opened={showApiModal}
          onClose={() => setShowApiModal(false)}
          title='API requests for Word of the Day'
          size='lg'
          centered
        >
          <Stack gap='sm'>
            <PasswordInput
              label='API Key'
              placeholder='YOUR_API_KEY'
              value={apiKey}
              onChange={e => setApiKey(e.currentTarget.value)}
              size='sm'
            />
            <Group gap='xs' align='flex-start'>
              <Info size={16} color='var(--mantine-color-dimmed)' />
              <Text c='dimmed' size='sm'>
                Date is optional. If omitted, today’s date is used.
              </Text>
            </Group>

            <Group justify='space-between' align='center' mt='xs'>
              <Group gap='xs'>
                <LinkIcon size={16} />
                <Text fw={600}>API URL (today)</Text>
              </Group>
              <CopyButton value={apiUrlToday} timeout={1500}>
                {({ copied, copy }) => (
                  <ActionIcon
                    onClick={copy}
                    variant='light'
                    color={copied ? 'teal' : 'primary'}
                    aria-label='Copy URL'
                  >
                    {copied ? <Check size={16} /> : <Copy size={16} />}
                  </ActionIcon>
                )}
              </CopyButton>
            </Group>
            <Paper
              withBorder
              radius='md'
              p='xs'
              bg='var(--mantine-color-default-hover)'
            >
              <Text
                style={{ fontFamily: 'monospace', wordBreak: 'break-all' }}
                size='sm'
              >
                {apiUrlToday}
              </Text>
            </Paper>

            <Group justify='space-between' align='center' mt='sm'>
              <Group gap='xs'>
                <LinkIcon size={16} />
                <Text fw={600}>API URL (specific date)</Text>
              </Group>
              <CopyButton value={apiUrlWithDate} timeout={1500}>
                {({ copied, copy }) => (
                  <ActionIcon
                    onClick={copy}
                    variant='light'
                    color={copied ? 'teal' : 'primary'}
                    aria-label='Copy URL'
                  >
                    {copied ? <Check size={16} /> : <Copy size={16} />}
                  </ActionIcon>
                )}
              </CopyButton>
            </Group>
            <Paper
              withBorder
              radius='md'
              p='xs'
              bg='var(--mantine-color-default-hover)'
            >
              <Text
                style={{ fontFamily: 'monospace', wordBreak: 'break-all' }}
                size='sm'
              >
                {apiUrlWithDate}
              </Text>
            </Paper>

            <Group justify='space-between' align='center' mt='xs'>
              <Group gap='xs'>
                <TerminalSquare size={16} />
                <Text fw={600}>curl (today) — JSON with API Key</Text>
              </Group>
              <CopyButton value={apiCurlApiKeyToday} timeout={1500}>
                {({ copied, copy }) => (
                  <ActionIcon
                    onClick={copy}
                    variant='light'
                    color={copied ? 'teal' : 'primary'}
                    aria-label='Copy curl'
                  >
                    {copied ? <Check size={16} /> : <Copy size={16} />}
                  </ActionIcon>
                )}
              </CopyButton>
            </Group>
            <Textarea
              value={apiCurlApiKeyToday}
              readOnly
              autosize
              minRows={3}
              styles={{
                input: {
                  fontFamily: 'monospace',
                  background: 'var(--mantine-color-default-hover)',
                },
              }}
            />

            <Group justify='space-between' align='center' mt='sm'>
              <Group gap='xs'>
                <TerminalSquare size={16} />
                <Text fw={600}>curl (specific date) — JSON with API Key</Text>
              </Group>
              <CopyButton value={apiCurlApiKeyWithDate} timeout={1500}>
                {({ copied, copy }) => (
                  <ActionIcon
                    onClick={copy}
                    variant='light'
                    color={copied ? 'teal' : 'primary'}
                    aria-label='Copy curl'
                  >
                    {copied ? <Check size={16} /> : <Copy size={16} />}
                  </ActionIcon>
                )}
              </CopyButton>
            </Group>
            <Textarea
              value={apiCurlApiKeyWithDate}
              readOnly
              autosize
              minRows={3}
              styles={{
                input: {
                  fontFamily: 'monospace',
                  background: 'var(--mantine-color-default-hover)',
                },
              }}
            />
          </Stack>
        </Modal>
      </Stack>
    </Container>
  );
};

export default WordOfTheDayPage;
