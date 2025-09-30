import { Link } from 'react-router-dom';
import { Container, Title, Text, Button, Stack, Center } from '@mantine/core';
import { IconHome } from '@tabler/icons-react';

const NotFoundPage = () => {
  return (
    <Container size='sm' py='xl'>
      <Center style={{ minHeight: '60vh' }}>
        <Stack align='center' gap='xl'>
          <Title order={1} size='6rem' c='gray.3' style={{ lineHeight: 1 }}>
            404
          </Title>
          <Title order={2} size='2rem' ta='center'>
            Page Not Found
          </Title>
          <Text size='lg' c='dimmed' ta='center'>
            The page you're looking for doesn't exist or has been moved.
          </Text>
          <Button
            component={Link}
            to='/'
            size='lg'
            leftSection={<IconHome size={20} />}
          >
            Go Home
          </Button>
        </Stack>
      </Center>
    </Container>
  );
};

export default NotFoundPage;
