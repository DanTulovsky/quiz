import React from 'react';
import {
  Container,
  Stack,
  Title,
  Text,
  Box,
  Card,
  Center,
} from '@mantine/core';
import { IconCheck } from '@tabler/icons-react';
import DailyDatePicker from './DailyDatePicker';

interface DailyCompletionScreenProps {
  selectedDate: string;
  onDateSelect: (date: string | null) => void;
  availableDates?: string[];
  progressData?: Record<string, { completed: number; total: number }>;
}

// Utility function to format date consistently without timezone issues
const formatDate = (dateString: string): string => {
  const [year, month, day] = dateString.split('-').map(Number);
  return `${month}/${day}/${year}`;
};

const DailyCompletionScreen: React.FC<DailyCompletionScreenProps> = ({
  selectedDate,
  onDateSelect,
  availableDates = [],
  progressData = {},
}) => {
  const isToday = selectedDate === new Date().toISOString().split('T')[0];

  return (
    <Container size='lg' py='xl'>
      <Stack gap='xl' align='center'>
        <Card shadow='sm' p='xl' radius='md' withBorder>
          <Stack gap='lg' align='center'>
            {/* Success Icon */}
            <Center>
              <Box
                style={{
                  width: 80,
                  height: 80,
                  borderRadius: '50%',
                  backgroundColor: 'var(--mantine-color-green-1)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <IconCheck size={40} color='var(--mantine-color-green-6)' />
              </Box>
            </Center>

            {/* Title */}
            <Title order={2} ta='center'>
              {isToday
                ? "Today's Questions Completed!"
                : 'Questions Completed!'}
            </Title>

            {/* Message */}
            <Text size='lg' ta='center' c='dimmed'>
              {isToday
                ? "Great job! You've completed all of today's questions. Come back tomorrow for more practice."
                : `Great job! You've completed all questions for ${formatDate(selectedDate)}.`}
            </Text>

            {/* Progress Summary */}
            {progressData[selectedDate] && (
              <Box ta='center'>
                <Text size='sm' c='dimmed'>
                  Progress: {progressData[selectedDate].completed}/
                  {progressData[selectedDate].total} completed
                </Text>
              </Box>
            )}

            {/* Action Buttons */}
            <Stack gap='md' w='100%' maw={400}>
              {isToday && (
                <Text size='lg' c='dimmed' ta='center'>
                  Come back tomorrow for more practice
                </Text>
              )}

              <Text size='lg' c='dimmed' ta='center'>
                Select another date to practice more questions
              </Text>

              {/* Date Picker (hidden but accessible) */}
              <Box style={{ opacity: 0, height: 0, overflow: 'hidden' }}>
                <DailyDatePicker
                  selectedDate={selectedDate}
                  onDateSelect={onDateSelect}
                  availableDates={availableDates}
                  progressData={progressData}
                  maxDate={new Date()}
                  size='sm'
                  clearable
                  hideOutsideDates
                  withCellSpacing={false}
                  firstDayOfWeek={1}
                />
              </Box>
            </Stack>
          </Stack>
        </Card>
      </Stack>
    </Container>
  );
};

export default DailyCompletionScreen;
