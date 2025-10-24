/**
 * Utility functions for story-related messages
 */

import { StoryWithSections } from '../api/storyApi';

/**
 * Gets the appropriate generating message based on the generation type
 * @param generationType - The type of generation: 'story', 'section', or null
 * @param currentStory - The current story object which may contain a custom message
 * @returns The message to display to the user
 */
export const getGeneratingMessage = (
  generationType: 'story' | 'section' | null,
  currentStory?: StoryWithSections | null
): string => {
  // Check if there's a custom message in the story object
  if (currentStory && 'message' in currentStory && currentStory.message) {
    return currentStory.message as string;
  }

  // Return appropriate message based on generation type
  switch (generationType) {
    case 'story':
      return 'Story created successfully. The first section is being generated. Please check back shortly.';
    case 'section':
      return 'Generating the next section of your story. Please check back shortly.';
    default:
      return 'Generating content. Please check back shortly.';
  }
};
