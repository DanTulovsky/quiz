import { useEffect, useState } from 'react';
import {
  Container,
  Title,
  Text,
  Card,
  Stack,
  Loader,
  Center,
  Alert,
} from '@mantine/core';
import { useNavigate } from 'react-router-dom';
import { getAllCategories, type CategoryInfo } from '../../utils/phrasebook';

const MobilePhrasebookIndexPage = () => {
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
    navigate(`/m/phrasebook/${categoryId}`);
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
      <Container size='lg' py='md'>
        <Alert color='red' title='Error'>
          {error}
        </Alert>
      </Container>
    );
  }

  return (
    <Container size='lg' py='md' px='xs'>
      <Stack gap='md'>
        <Stack gap='xs'>
          <Title order={2}>Phrasebook</Title>
          <Text size='sm' c='dimmed'>
            Browse common vocabulary and phrases organized by topics
          </Text>
        </Stack>

        <Stack gap='sm'>
          {categories.map(category => (
            <Card
              key={category.id}
              shadow='sm'
              padding='md'
              radius='md'
              withBorder
              style={{ cursor: 'pointer' }}
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
              <Stack gap='xs'>
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '12px',
                  }}
                >
                  <Text size='2.5rem' style={{ lineHeight: 1 }}>
                    {category.emoji}
                  </Text>
                  <div style={{ flex: 1 }}>
                    <Title order={4}>{category.name}</Title>
                    <Text size='sm' c='dimmed' mt={4}>
                      {category.description}
                    </Text>
                  </div>
                </div>
              </Stack>
            </Card>
          ))}
        </Stack>
      </Stack>
    </Container>
  );
};

export default MobilePhrasebookIndexPage;
