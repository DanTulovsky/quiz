import React from 'react';
import { Badge, Group, Box, Tooltip } from '@mantine/core';
import { Question } from '../api/api';

interface VarietyTagsProps {
  question: Question;
  size?: 'xs' | 'sm' | 'md' | 'lg';
  compact?: boolean;
}

const VarietyTags: React.FC<VarietyTagsProps> = ({
  question,
  size = 'xs',
  compact = false,
}) => {
  const varietyElements = [
    { key: 'topic_category', label: 'Topic', value: question.topic_category },
    { key: 'grammar_focus', label: 'Grammar', value: question.grammar_focus },
    {
      key: 'vocabulary_domain',
      label: 'Vocabulary',
      value: question.vocabulary_domain,
    },
    { key: 'scenario', label: 'Scenario', value: question.scenario },
    { key: 'style_modifier', label: 'Style', value: question.style_modifier },
    {
      key: 'difficulty_modifier',
      label: 'Difficulty',
      value: question.difficulty_modifier,
    },
    { key: 'time_context', label: 'Context', value: question.time_context },
  ];

  const validElements = varietyElements.filter(element => element.value);

  if (validElements.length === 0) {
    return null;
  }

  const formatValue = (value: string) => {
    return value
      .split('_')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  };

  const getTagColor = (key: string) => {
    const colors = {
      topic_category: 'blue',
      grammar_focus: 'green',
      vocabulary_domain: 'purple',
      scenario: 'orange',
      style_modifier: 'pink',
      difficulty_modifier: 'red',
      time_context: 'teal',
    };
    return colors[key as keyof typeof colors] || 'gray';
  };

  if (compact) {
    return (
      <Group gap='xs' mb='sm' data-testid='variety-tags'>
        {validElements.map(element => (
          <Badge
            key={element.key}
            size={size}
            variant='light'
            color={getTagColor(element.key)}
            radius='sm'
          >
            {formatValue(element.value!)}
          </Badge>
        ))}
      </Group>
    );
  }

  return (
    <Box mb={8} data-testid='variety-tags'>
      <Group gap={4} wrap='wrap'>
        {validElements.map(element => (
          <Tooltip
            key={element.key}
            label={`${element.label}: ${formatValue(element.value!)}`}
            withArrow
            position='top'
            openDelay={200}
          >
            <Badge
              size={size}
              variant='light'
              color={getTagColor(element.key)}
              radius='sm'
              style={{ cursor: 'pointer' }}
            >
              {formatValue(element.value!)}
            </Badge>
          </Tooltip>
        ))}
      </Group>
    </Box>
  );
};

export default VarietyTags;
