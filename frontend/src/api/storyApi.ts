import {
  CreateStoryRequest,
  Story,
  StoryWithSections,
  StorySectionWithQuestions,
  StorySection,
} from './api';

// Story API client functions using the generated types

/**
 * Create a new story
 */
export const createStory = async (data: CreateStoryRequest): Promise<Story> => {
  const { postV1Story } = await import('./api');
  return postV1Story(data);
};

/**
 * Get all user stories, optionally including archived ones
 */
export const getUserStories = async (
  includeArchived?: boolean
): Promise<Story[]> => {
  const { getV1Story } = await import('./api');
  return getV1Story({ include_archived: includeArchived });
};

/**
 * Get the user's current active story
 */
export const getCurrentStory = async (): Promise<StoryWithSections | null> => {
  const { getV1StoryCurrent } = await import('./api');
  try {
    const result = await getV1StoryCurrent();
    // If it's a GeneratingResponse, return null (story is being generated)
    if (result && typeof result === 'object' && 'status' in result && result.status === 'generating') {
      return null;
    }
    // Otherwise, it should be StoryWithSections
    return result as StoryWithSections;
  } catch (error) {
    // If no current story, return null instead of throwing
    if (
      error &&
      typeof error === 'object' &&
      'status' in error &&
      error.status === 404
    ) {
      return null;
    }
    throw error;
  }
};

/**
 * Get a specific story by ID
 */
export const getStory = async (storyId: number): Promise<StoryWithSections> => {
  const { getV1StoryId } = await import('./api');
  return getV1StoryId(storyId);
};

/**
 * Get a specific story section by ID
 */
export const getSection = async (
  sectionId: number
): Promise<StorySectionWithQuestions> => {
  const { getV1StorySectionId } = await import('./api');
  return getV1StorySectionId(sectionId);
};

/**
 * Generate the next section for a story
 */
export const generateNextSection = async (
  storyId: number
): Promise<StorySection> => {
  const { postV1StoryIdGenerate } = await import('./api');
  return postV1StoryIdGenerate(storyId);
};

/**
 * Archive a story
 */
export const archiveStory = async (storyId: number): Promise<void> => {
  const { postV1StoryIdArchive } = await import('./api');
  await postV1StoryIdArchive(storyId);
};

/**
 * Complete a story
 */
export const completeStory = async (storyId: number): Promise<void> => {
  const { postV1StoryIdComplete } = await import('./api');
  await postV1StoryIdComplete(storyId);
};

/**
 * Set a story as the current active story
 */
export const setCurrentStory = async (storyId: number): Promise<void> => {
  const { postV1StoryIdSetCurrent } = await import('./api');
  await postV1StoryIdSetCurrent(storyId);
};

/**
 * Delete a story (only archived or completed stories)
 */
export const deleteStory = async (storyId: number): Promise<void> => {
  const { deleteV1StoryId } = await import('./api');
  await deleteV1StoryId(storyId);
};

/**
 * Export a story as PDF
 */
export const exportStoryPDF = async (storyId: number): Promise<Blob> => {
  const { getV1StoryIdExport } = await import('./api');
  return getV1StoryIdExport(storyId);
};

/**
 * Toggle auto-generation for a story
 */
export const toggleAutoGeneration = async (
  storyId: number,
  paused: boolean
): Promise<void> => {
  const { postV1StoryIdToggleAutoGeneration } = await import('./api');
  await postV1StoryIdToggleAutoGeneration(storyId, { paused });
};

// Re-export types for convenience
export type {
  Story,
  StorySection,
  StorySectionQuestion,
  StoryWithSections,
  StorySectionWithQuestions,
  CreateStoryRequest,
} from './api';
