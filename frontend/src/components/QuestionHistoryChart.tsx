import React from 'react';
import { Group, Text, Tooltip, Box, LoadingOverlay } from '@mantine/core';
import { DailyQuestionHistory } from '../api/api';
import dayjs from 'dayjs';

// Extend with UTC plugin if available (for production)
try {
  const utc = require('dayjs/plugin/utc');
  dayjs.extend(utc);
} catch {
  // UTC plugin not available in test environment, continue without it
}

interface QuestionHistoryChartProps {
  history: DailyQuestionHistory[];
  isLoading: boolean;
  questionId: number;
}

export function QuestionHistoryChart({
  history,
  isLoading,
  questionId,
}: QuestionHistoryChartProps) {
  // Show loading state
  if (isLoading) {
    return (
      <Box pos='relative' h={60}>
        <LoadingOverlay visible={true} data-testid='loading-overlay' />
      </Box>
    );
  }

  // Handle empty history
  if (!history || history.length === 0) {
    return (
      <Text size='sm' c='dimmed' ta='center' py='sm'>
        No history available for this question.
      </Text>
    );
  }

  // Sort history by date (earliest first)
  const sortedHistory = [...history].sort((a, b) => {
    try {
      const dateA = dayjs.utc(a.assignment_date);
      const dateB = dayjs.utc(b.assignment_date);
      return dateA.isAfter(dateB) ? 1 : -1;
    } catch {
      return 0;
    }
  });

  // Take only last 14 days
  const last14Days = sortedHistory.slice(0, 14);

  const getStatusColor = (isCorrect: boolean | null | undefined) => {
    if (isCorrect === null || isCorrect === undefined) return 'gray';
    return isCorrect ? 'green' : 'red';
  };

  const getStatusSymbol = (isCorrect: boolean | null | undefined) => {
    if (isCorrect === null || isCorrect === undefined) return '○';
    return isCorrect ? '●' : '●';
  };

  const formatFullDate = (dateString: string) => {
    try {
      return dayjs.utc(dateString).format('MMM D, YYYY');
    } catch {
      return dateString;
    }
  };

  return (
    <Box py='md'>
      <Group gap='xs' justify='center' align='center'>
        <Tooltip label='History for the last 14 days' position='top'>
          <Text size='xs' c='gray' variant='subtle'>
            History:
          </Text>
        </Tooltip>
        {last14Days.map((entry, index) => (
          <Tooltip
            key={`${questionId}-${index}`}
            label={`${formatFullDate(entry.assignment_date)}: ${
              entry.is_correct === null || entry.is_correct === undefined
                ? 'Not attempted'
                : entry.is_correct
                  ? 'Correct'
                  : 'Incorrect'
            }`}
            position='top'
          >
            <Box
              style={{
                width: 16,
                height: 16,
                borderRadius: '50%',
                backgroundColor: getStatusColor(entry.is_correct),
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '10px',
                color: 'white',
                cursor: 'default',
              }}
              data-testid={`history-dot-${index}`}
            >
              {getStatusSymbol(entry.is_correct)}
            </Box>
          </Tooltip>
        ))}
      </Group>
    </Box>
  );
}
