import { useEffect, useState } from 'react';
import {
  Container,
  Title,
  Text,
  SimpleGrid,
  Card,
  Group,
  Stack,
  Loader,
  Center,
  Alert,
} from '@mantine/core';
import { useNavigate } from 'react-router-dom';
import { getAllCategories, type CategoryInfo } from '../utils/phrasebook';

const PhrasebookIndexPage = () => {
  const navigate = useNavigate();
  const [categories, setCategories] = useState<CategoryInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadCategories = async () => {
      try {
        setLoading(true);
        setError(null);
        const loadedCategories = await getAllCategories();
        setCategories(loadedCategories);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : 'Failed to load categories'
        );
      } finally {
        setLoading(false);
      }
    };

    loadCategories();
  }, []);

  const handleCategoryClick = (categoryId: string) => {
    navigate(`/phrasebook/${categoryId}`);
  };

  if (loading) {
    return (
      <Center h='50vh'>
        <Loader size='lg' />
      </Center>
    );
  }

  if (error) {
    return (
      <Container size='lg' py='xl'>
        <Alert color='red' title='Error'>
          {error}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size='lg' py='xl'>
      <Stack gap='xl'>
        <div>
          <Title order={1} mb='sm'>
            Phrasebook
          </Title>
          <Text size='lg' c='dimmed'>
            Browse common vocabulary and phrases organized by topics
          </Text>
        </div>

        <SimpleGrid cols={{ base: 1, sm: 2, md: 2 }} spacing='lg'>
          {categories.map(category => (
            <Card
              key={category.id}
              shadow='sm'
              padding='lg'
              radius='md'
              withBorder
              style={{ cursor: 'pointer', transition: 'transform 0.2s' }}
              onClick={() => handleCategoryClick(category.id)}
              onKeyDown={e => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  handleCategoryClick(category.id);
                }
              }}
              tabIndex={0}
              role='button'
              aria-label={`Open ${category.name} category`}
            >
              <Stack gap='md'>
                <Group gap='md'>
                  <Text size='3rem' style={{ lineHeight: 1 }}>
                    {category.emoji}
                  </Text>
                  <div style={{ flex: 1 }}>
                    <Title order={3}>{category.name}</Title>
                  </div>
                </Group>
                <Text c='dimmed'>{category.description}</Text>
              </Stack>
            </Card>
          ))}
        </SimpleGrid>
      </Stack>
    </Container>
  );
};

export default PhrasebookIndexPage;
