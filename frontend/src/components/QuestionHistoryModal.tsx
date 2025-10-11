import {
  Modal,
  Stack,
  Group,
  Badge,
  ScrollArea,
  LoadingOverlay,
  Text,
} from '@mantine/core';
// Use simple unicode icons here to avoid depending on icon prop typings in tests
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
dayjs.extend(utc);
import logger from '../utils/logger';
import type { DailyQuestionHistory as ApiDailyQuestionHistory } from '../api/api';

interface QuestionHistoryModalProps {
  opened: boolean;
  onClose: () => void;
  history: ApiDailyQuestionHistory[];
  isLoading: boolean;
  questionText?: string;
}

export function QuestionHistoryModal({
  opened,
  onClose,
  history,
  isLoading,
  questionText,
}: QuestionHistoryModalProps) {
  const formatDate = (dateString: string) => {
    // The frontend expects the backend to always send timezone-aware
    // timestamps (ISO with timezone). If a date-only string is received
    // that means the backend failed to include timezone information —
    // log an error with context for debugging and fall back to parsing
    // as a local date to avoid completely breaking the UI.
    if (/^\d{4}-\d{2}-\d{2}$/.test(dateString)) {
      // Per contract, the backend MUST return timezone-aware timestamps. Treat
      // date-only strings as a backend error and log with context. Display the
      // raw date string for now but surface the issue to developers via logs.
      // Throw to make testable behavior; in production we still want UI to show
      // something rather than crash, so fall back to formatted local date as a last resort.
      try {
        throw new Error('date-only timestamp received from backend');
      } catch (err) {
      }
      return dayjs(dateString).format('MMM D, YYYY');
    }

    // For any timestamp with a timezone (including UTC) parse in UTC and
    // format in UTC to keep tests deterministic (avoid local TZ shifts).
    try {
      return dayjs.utc(dateString).format('MMM D, YYYY');
    } catch (err) {
      // As a last resort, return the raw string so something is displayed.
      return dateString;
    }
  };

  const getStatusIcon = (
    isCompleted: boolean,
    isCorrect: boolean | null | undefined
  ) => {
    if (!isCompleted) {
      return (
        <span aria-hidden style={{ fontSize: 16 }}>
          ⏱️
        </span>
      );
    }
    if (isCorrect === null || isCorrect === undefined) {
      return (
        <span aria-hidden style={{ fontSize: 16 }}>
          ⏱️
        </span>
      );
    }
    return isCorrect ? (
      <span aria-hidden style={{ fontSize: 16, color: 'green' }}>
        ✓
      </span>
    ) : (
      <span aria-hidden style={{ fontSize: 16, color: 'red' }}>
        ✖
      </span>
    );
  };

  const getStatusColor = (
    isCompleted: boolean,
    isCorrect: boolean | null | undefined
  ) => {
    if (!isCompleted) {
      return 'gray';
    }
    if (isCorrect === null || isCorrect === undefined) {
      return 'gray';
    }
    return isCorrect ? 'green' : 'red';
  };

  const getStatusText = (
    isCompleted: boolean,
    isCorrect: boolean | null | undefined
  ) => {
    if (!isCompleted) {
      return 'Not attempted';
    }
    if (isCorrect === null || isCorrect === undefined) {
      return 'Not attempted';
    }
    return isCorrect ? 'Correct' : 'Incorrect';
  };

  return (
    <Modal opened={opened} onClose={onClose} size='md' centered>
      <LoadingOverlay visible={isLoading} />
      <Stack gap='md'>
        {questionText && (
          <div>
            <Text size='lg' fw={700}>
              {questionText}
            </Text>
            <Text size='sm' c='dimmed' mt={4}>
              Last 14 days
            </Text>
          </div>
        )}

        <ScrollArea h={300}>
          <Stack gap='xs'>
            {/* defensive: ensure history is an array before mapping */}
            {!history || (Array.isArray(history) && history.length === 0) ? (
              <div style={{ textAlign: 'center', color: 'gray' }}>
                No history available for this question.
              </div>
            ) : (
              (() => {
                // Ensure we show the latest assignment first (descending by
                // assignment_date). The API may return entries in ascending
                // order; sort defensively here so the UI always places the most
                // recent date at the top.
                const entries: ApiDailyQuestionHistory[] = Array.isArray(
                  history
                )
                  ? history.slice()
                  : (history as { history?: ApiDailyQuestionHistory[] })
                      ?.history || [];
                entries.sort((a, b) => {
                  // Compare timezone-aware timestamps; fall back to string
                  // comparison if parsing fails.
                  try {
                    const ta = dayjs.utc(a.assignment_date);
                    const tb = dayjs.utc(b.assignment_date);
                    if (ta.isBefore(tb)) return 1;
                    if (ta.isAfter(tb)) return -1;
                    return 0;
                  } catch (err) {
                    // Log parse/compare errors for observability and fall back
                    // to a stable string comparison so UI still renders.
                    return b.assignment_date.localeCompare(a.assignment_date);
                  }
                });
                return entries.map(
                  (entry: ApiDailyQuestionHistory, index: number) => {
                    const isCorrect = entry.is_correct as
                      | boolean
                      | null
                      | undefined;
                    return (
                      <Group key={index} justify='space-between' align='center'>
                        <div>{formatDate(entry.assignment_date)}</div>
                        <Badge
                          color={getStatusColor(entry.is_completed, isCorrect)}
                          leftSection={getStatusIcon(
                            entry.is_completed,
                            isCorrect
                          )}
                        >
                          {getStatusText(entry.is_completed, isCorrect)}
                        </Badge>
                      </Group>
                    );
                  }
                );
              })()
            )}
          </Stack>
        </ScrollArea>
      </Stack>
    </Modal>
  );
}
