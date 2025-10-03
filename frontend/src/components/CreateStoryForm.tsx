import React, { useState } from 'react';
import {
  Button,
  TextInput,
  Textarea,
  Select,
  SegmentedControl,
  Paper,
  Title,
  Text,
  Group,
  Stack,
  Alert,
} from '@mantine/core';
import { IconBook, IconAlertCircle } from '@tabler/icons-react';
import { CreateStoryRequest } from '../api/storyApi';

interface CreateStoryFormProps {
  onSubmit: (data: CreateStoryRequest) => Promise<void>;
  loading?: boolean;
}

const CreateStoryForm: React.FC<CreateStoryFormProps> = ({
  onSubmit,
  loading = false,
}) => {
  const [formData, setFormData] = useState<CreateStoryRequest>({
    title: '',
    subject: null,
    author_style: null,
    time_period: null,
    genre: null,
    tone: null,
    character_names: null,
    custom_instructions: null,
    section_length_override: null,
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.title?.trim()) {
      newErrors.title = 'Title is required';
    } else if (formData.title.length > 200) {
      newErrors.title = 'Title must be 200 characters or less';
    }

    if (formData.subject && formData.subject.length > 500) {
      newErrors.subject = 'Subject must be 500 characters or less';
    }

    if (formData.author_style && formData.author_style.length > 200) {
      newErrors.author_style = 'Author style must be 200 characters or less';
    }

    if (formData.time_period && formData.time_period.length > 200) {
      newErrors.time_period = 'Time period must be 200 characters or less';
    }

    if (formData.genre && formData.genre.length > 100) {
      newErrors.genre = 'Genre must be 100 characters or less';
    }

    if (formData.tone && formData.tone.length > 100) {
      newErrors.tone = 'Tone must be 100 characters or less';
    }

    if (formData.character_names && formData.character_names.length > 1000) {
      newErrors.character_names =
        'Character names must be 1000 characters or less';
    }

    if (
      formData.custom_instructions &&
      formData.custom_instructions.length > 2000
    ) {
      newErrors.custom_instructions =
        'Custom instructions must be 2000 characters or less';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      // Set the first error to trigger display
      if (Object.keys(errors).length > 0) {
        const firstError = Object.keys(errors)[0];
        setErrors(prev => ({ ...prev, [firstError]: errors[firstError] }));
      }
      return;
    }

    try {
      // Filter out null values for optional fields
      const filteredData: CreateStoryRequest = Object.fromEntries(
        Object.entries(formData).filter(([_, value]) => value !== null)
      ) as CreateStoryRequest;
      await onSubmit(filteredData);
    } catch {
      // Error handling is done in the parent component
    }
  };

  const updateField = (
    field: keyof CreateStoryRequest,
    value: string | null
  ) => {
    setFormData(prev => ({ ...prev, [field]: value }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  const genreOptions = [
    { value: 'mystery', label: 'Mystery' },
    { value: 'romance', label: 'Romance' },
    { value: 'sci-fi', label: 'Science Fiction' },
    { value: 'fantasy', label: 'Fantasy' },
    { value: 'adventure', label: 'Adventure' },
    { value: 'drama', label: 'Drama' },
    { value: 'comedy', label: 'Comedy' },
    { value: 'horror', label: 'Horror' },
    { value: 'historical', label: 'Historical' },
    { value: 'literary', label: 'Literary Fiction' },
  ];

  const toneOptions = [
    { value: 'serious', label: 'Serious' },
    { value: 'humorous', label: 'Humorous' },
    { value: 'dramatic', label: 'Dramatic' },
    { value: 'lighthearted', label: 'Lighthearted' },
    { value: 'suspenseful', label: 'Suspenseful' },
    { value: 'romantic', label: 'Romantic' },
    { value: 'dark', label: 'Dark' },
    { value: 'optimistic', label: 'Optimistic' },
  ];

  return (
    <Paper p='xl' radius='md' style={{ maxWidth: 600, margin: '0 auto' }}>
      <Stack spacing='lg'>
        <Group position='center' spacing='xs'>
          <IconBook size={24} />
          <Title order={3}>Create New Story</Title>
        </Group>

        <Text size='sm' color='dimmed' align='center'>
          Create a personalized story in your target language. The AI will
          generate content at your current proficiency level.
        </Text>

        <form onSubmit={handleSubmit}>
          <Stack spacing='md'>
            {/* Title - Required */}
            <TextInput
              label='Story Title'
              placeholder='Enter a title for your story'
              value={formData.title}
              onChange={e => updateField('title', e.target.value)}
              error={errors.title}
              required
              data-testid='story-title-input'
            />

            {/* Subject */}
            <Textarea
              label='Subject (Optional)'
              placeholder="What is your story about? (e.g., 'A detective solving crimes in Victorian England')"
              value={formData.subject ?? ''}
              onChange={e => updateField('subject', e.target.value || null)}
              error={errors.subject}
              minRows={2}
              maxRows={4}
              data-testid='story-subject-input'
            />

            {/* Author Style */}
            <TextInput
              label='Author Style (Optional)'
              placeholder="In the style of a specific author (e.g., 'Hemingway', 'Agatha Christie')"
              value={formData.author_style ?? ''}
              onChange={e =>
                updateField('author_style', e.target.value || null)
              }
              error={errors.author_style}
              data-testid='story-author-style-input'
            />

            {/* Time Period */}
            <TextInput
              label='Time Period (Optional)'
              placeholder="When does your story take place? (e.g., '1920s', 'medieval times')"
              value={formData.time_period ?? ''}
              onChange={e => updateField('time_period', e.target.value || null)}
              error={errors.time_period}
              data-testid='story-time-period-input'
            />

            {/* Genre */}
            <Select
              label='Genre (Optional)'
              placeholder='Select a genre'
              data={genreOptions}
              value={formData.genre ?? ''}
              onChange={value => updateField('genre', value || null)}
              error={errors.genre}
              clearable
              searchable
              data-testid='story-genre-select'
            />

            {/* Tone */}
            <Select
              label='Tone (Optional)'
              placeholder='Select a tone'
              data={toneOptions}
              value={formData.tone ?? ''}
              onChange={value => updateField('tone', value || null)}
              error={errors.tone}
              clearable
              searchable
              data-testid='story-tone-select'
            />

            {/* Character Names */}
            <Textarea
              label='Main Characters (Optional)'
              placeholder="List the main characters (e.g., 'Detective Smith, Lady Blackwood, Mr. Jones')"
              value={formData.character_names ?? ''}
              onChange={e =>
                updateField('character_names', e.target.value || null)
              }
              error={errors.character_names}
              minRows={2}
              maxRows={3}
              data-testid='story-characters-input'
            />

            {/* Custom Instructions */}
            <Textarea
              label='Custom Instructions (Optional)'
              placeholder="Any specific instructions for the AI (e.g., 'Focus on psychological tension and plot twists')"
              value={formData.custom_instructions ?? ''}
              onChange={e =>
                updateField('custom_instructions', e.target.value || null)
              }
              error={errors.custom_instructions}
              minRows={3}
              maxRows={5}
              data-testid='story-instructions-input'
            />

            {/* Section Length Override */}
            <div>
              <Text size='sm' weight={500} mb='xs'>
                Section Length Preference (Optional)
              </Text>
              <Text size='xs' color='dimmed' mb='sm'>
                Override the default length based on your language level
              </Text>
              <SegmentedControl
                value={formData.section_length_override || ''}
                onChange={value =>
                  updateField(
                    'section_length_override',
                    value === '' ? null : (value as 'short' | 'medium' | 'long')
                  )
                }
                data={[
                  { label: 'Short', value: 'short' },
                  { label: 'Medium', value: 'medium' },
                  { label: 'Long', value: 'long' },
                ]}
                data-testid='story-length-control'
              />
            </div>

            {/* Submit Button */}
            <Button
              type='submit'
              loading={loading}
              disabled={loading}
              size='lg'
              data-testid='create-story-submit'
            >
              {loading ? 'Creating Story...' : 'Create Story'}
            </Button>
          </Stack>
        </form>

        {/* Information Alert */}
        <Alert
          icon={<IconAlertCircle size={16} />}
          color='blue'
          variant='light'
        >
          <Text size='xs'>
            Your story will be generated at your current language proficiency
            level. Each new section will be added daily, and you can read
            comprehension questions to test your understanding.
          </Text>
        </Alert>
      </Stack>
    </Paper>
  );
};

export default CreateStoryForm;
