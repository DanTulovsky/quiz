import React from 'react';
import { Group, Tooltip, Text } from '@mantine/core';
import VarietyTags from './VarietyTags';
import { formatFullTimestamp, formatRelativeTimestamp } from '../utils/time';
import type { Question } from '../api/api';
// Icons are rendered in the question card's bottom stats; no imports needed here.

export interface QuestionHeaderProps {
  question: Question;
  timezone?: string | null;
  /** When true, display the confidence icon if present on the question */
  showConfidence?: boolean;
}

const QuestionHeader: React.FC<QuestionHeaderProps> = ({
  question,
  timezone,
  showConfidence = false,
}) => {
  return (
    <Group justify='space-between' align='center' mb='xs'>
      {question && <VarietyTags question={question} />}
      <Group gap={8} align='center'>
        {/* Render confidence icon when requested and present on the question */}
        {showConfidence && question?.confidence_level && (
          <Tooltip
            label={`Confidence Level: ${question.confidence_level}/5`}
            position='top'
            withArrow
          >
            <Text size='xs' c='dimmed' data-testid='question-header-confidence'>
              {/* Use simple emoji here to avoid extra icon imports; kept minimal */}
              {question.confidence_level === 1 && '😢'}
              {question.confidence_level === 2 && '😕'}
              {question.confidence_level === 3 && '😐'}
              {question.confidence_level === 4 && '🙂'}
              {question.confidence_level === 5 && '😄'}
            </Text>
          </Tooltip>
        )}
        {question?.created_at && (
          <Tooltip
            label={`Created on ${formatFullTimestamp(question.created_at, timezone)}`}
            position='top'
            withArrow
          >
            <Text
              size='xs'
              c='dimmed'
              style={{
                backgroundColor: 'var(--mantine-color-body)',
                padding: '2px 6px',
                borderRadius: '4px',
                border: '1px solid var(--mantine-color-default-border)',
                cursor: 'help',
              }}
            >
              {formatRelativeTimestamp(question.created_at)}
            </Text>
          </Tooltip>
        )}
      </Group>
    </Group>
  );
};

export default QuestionHeader;
