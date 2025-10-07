import React from 'react';
import { render, screen } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { vi, describe, it, expect } from 'vitest';
import StoryReadingView from './StoryReadingView';
import { StoryWithSections } from '../api/storyApi';

// Mock the notifications module
vi.mock('../notifications', () => ({
  showNotificationWithClean: vi.fn(),
}));

describe('StoryReadingView', () => {
  const mockStory: StoryWithSections = {
    id: 1,
    title: 'Test Mystery Story',
    language: 'en',
    status: 'active',
    is_current: true,
    sections: [
      {
        id: 1,
        story_id: 1,
        section_number: 1,
        content:
          'In the quiet town of Willow Creek, Detective Sarah Johnson received a mysterious letter that would change everything.',
        language_level: 'intermediate',
        word_count: 18,
        generated_at: '2025-01-15T10:30:00Z',
        generation_date: '2025-01-15',
      },
      {
        id: 2,
        story_id: 1,
        section_number: 2,
        content:
          'The letter contained a cryptic message about a hidden treasure in the old manor house. Sarah knew she had to investigate.',
        language_level: 'intermediate',
        word_count: 20,
        generated_at: '2025-01-15T11:30:00Z',
        generation_date: '2025-01-15',
      },
    ],
  };

  const defaultProps = {
    story: mockStory,
    onViewModeChange: vi.fn(),
    viewMode: 'reading' as const,
  };

  const renderComponent = (props = {}) => {
    const allProps = { ...defaultProps, ...props };

    return render(
      <MantineProvider>
        <StoryReadingView {...allProps} />
      </MantineProvider>
    );
  };

  describe('Story Display', () => {
    it('displays story title and metadata correctly', () => {
      renderComponent();

      expect(screen.getByText('Test Mystery Story')).toBeInTheDocument();
      expect(screen.getByText('EN')).toBeInTheDocument();
      expect(screen.getByText('2 sections')).toBeInTheDocument();
      expect(screen.getByText('active')).toBeInTheDocument();
    });

    it('displays all sections in order', () => {
      renderComponent();

      // StoryReadingView shows continuous text without section headers
      expect(screen.getByText('Test Mystery Story')).toBeInTheDocument();
      expect(
        screen.getByText(/In the quiet town of Willow Creek/)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/The letter contained a cryptic message/)
      ).toBeInTheDocument();
    });

    it('displays story status and continuous reading experience', () => {
      renderComponent();

      // StoryReadingView shows continuous text without individual section metadata
      expect(screen.getByText('Test Mystery Story')).toBeInTheDocument();
      expect(
        screen.getByText(
          'This story is ongoing. New sections will be added daily!'
        )
      ).toBeInTheDocument();
    });

    it('shows story details when available', () => {
      const storyWithDetails = {
        ...mockStory,
        subject: 'A detective mystery',
        author_style: 'Agatha Christie',
        genre: 'mystery',
      };

      renderComponent({ story: storyWithDetails });

      expect(screen.getByText('Story Details')).toBeInTheDocument();
      expect(screen.getByText('A detective mystery')).toBeInTheDocument();
      expect(screen.getByText('Agatha Christie')).toBeInTheDocument();
      expect(screen.getByText('mystery')).toBeInTheDocument();
    });
  });

  describe('Empty States', () => {
    it('displays empty state when no story provided', () => {
      renderComponent({ story: null });

      expect(screen.getByText('No story to display')).toBeInTheDocument();
      expect(
        screen.getByText('Create a new story to start reading.')
      ).toBeInTheDocument();
    });

    it('displays in-progress state when story has no sections', () => {
      const emptyStory = { ...mockStory, sections: [] };
      renderComponent({ story: emptyStory });

      expect(screen.getByText('Story in Progress')).toBeInTheDocument();
      expect(
        screen.getByText(
          'Your story is being prepared. Check back soon for the first section!'
        )
      ).toBeInTheDocument();
    });
  });

  describe('Story Status Display', () => {
    it('shows active story status', () => {
      renderComponent();

      expect(
        screen.getByText(
          'This story is ongoing. New sections will be added daily!'
        )
      ).toBeInTheDocument();
    });

    it('shows completed story status', () => {
      const completedStory = { ...mockStory, status: 'completed' as const };
      renderComponent({ story: completedStory });

      expect(
        screen.getByText('This story has been completed.')
      ).toBeInTheDocument();
    });

    it('shows archived story status', () => {
      const archivedStory = { ...mockStory, status: 'archived' as const };
      renderComponent({ story: archivedStory });

      expect(
        screen.getByText('This story has been archived.')
      ).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('provides proper semantic structure', () => {
      const storyWithDetails = {
        ...mockStory,
        subject: 'A detective mystery',
        author_style: 'Agatha Christie',
        genre: 'mystery',
      };

      renderComponent({ story: storyWithDetails });

      // Should have proper heading structure
      expect(screen.getByRole('heading', { level: 3 })).toBeInTheDocument();
      expect(screen.getByRole('heading', { level: 5 })).toBeInTheDocument();
    });
  });

  describe('Scrolling and Layout', () => {
    it('renders sections in a scrollable area', () => {
      renderComponent();

      // Should have ScrollArea component (check for the viewport element)
      const scrollViewport = document.querySelector(
        '.mantine-ScrollArea-viewport'
      );
      expect(scrollViewport).toBeInTheDocument();
    });

    it('displays story content in scrollable area with proper spacing', () => {
      renderComponent();

      // StoryReadingView shows continuous text in a scrollable area
      const scrollViewport = document.querySelector(
        '.mantine-ScrollArea-viewport'
      );
      expect(scrollViewport).toBeInTheDocument();

      // Should contain all story content
      expect(
        screen.getByText(/In the quiet town of Willow Creek/)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/The letter contained a cryptic message/)
      ).toBeInTheDocument();
    });
  });
});
