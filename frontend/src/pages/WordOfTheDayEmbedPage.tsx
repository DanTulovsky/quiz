import React from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { useGetV1WordOfDayDateEmbed } from '../api/api';
import { Container, Stack, Text, Center, Card, Title } from '@mantine/core';
import LoadingSpinner from '../components/LoadingSpinner';

const WordOfTheDayEmbedPage: React.FC = () => {
  const { date } = useParams<{ date: string }>();
  const [searchParams] = useSearchParams();
  const userId = searchParams.get('user_id');

  const {
    data: htmlContent,
    isLoading,
    error,
  } = useGetV1WordOfDayDateEmbed(
    date || '',
    { user_id: userId ? parseInt(userId) : undefined },
    {
      query: {
        enabled: !!date && !!userId,
        retry: false,
      },
    }
  );

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

  if (!date || !userId) {
    return (
      <Container size='md' py='xl'>
        <Center h='60vh'>
          <Text c='dimmed'>Date and user_id are required.</Text>
        </Center>
      </Container>
    );
  }

  if (isLoading) {
    return (
      <Container size='md' py='xl'>
        <Center h='60vh'>
          <LoadingSpinner />
        </Center>
      </Container>
    );
  }

  if (error) {
    return (
      <Container size='md' py='xl'>
        <Center h='60vh'>
          <Text c='red'>Failed to load word of the day.</Text>
        </Center>
      </Container>
    );
  }

  // If we have HTML content, render it in an iframe
  if (htmlContent) {
    return (
      <div style={{ width: '100%', height: '100vh', border: 'none' }}>
        <iframe
          srcDoc={htmlContent}
          style={{ width: '100%', height: '100%', border: 'none' }}
          title='Word of the Day Embed'
        />
      </div>
    );
  }

  // Fallback: render word data directly (if API returns JSON instead of HTML)
  return (
    <Container size='md' py='xl'>
      <Card shadow='md' padding='xl' radius='md'>
        <Stack gap='md'>
          <Text
            size='sm'
            fw={600}
            c='dimmed'
            style={{ textTransform: 'uppercase' }}
          >
            {date ? formatDisplayDate(date) : 'Word of the Day'}
          </Text>
          <Title order={1}>Content Loading...</Title>
        </Stack>
      </Card>
    </Container>
  );
};

export default WordOfTheDayEmbedPage;
