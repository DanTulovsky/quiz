import { IconCalendar } from '@tabler/icons-react';

export interface PhrasebookWord {
  term: string; // English term (default)
  icon?: string;
  note: string;
  en?: string; // English
  it?: string; // Italian
  fr?: string; // French
  de?: string; // German
  ru?: string; // Russian
  ja?: string; // Japanese
  zh?: string; // Chinese
}

export interface PhrasebookSection {
  title: string;
  words: PhrasebookWord[];
}

export interface PhrasebookData {
  category: string;
  sections: PhrasebookSection[];
}

export type CategoryId = string; // Now dynamic - any string can be a category ID

export interface CategoryInfo {
  id: CategoryId;
  name: string;
  icon: typeof IconCalendar;
  emoji: string;
  description: string;
}

// Cache for loaded categories
let categoriesCache: CategoryInfo[] | null = null;

/**
 * Dynamically load all available categories from info.json files
 */
export async function getAllCategories(): Promise<CategoryInfo[]> {
  if (categoriesCache) {
    return categoriesCache;
  }

  try {
    // Load the master index to get all available categories
    const indexData = await import('../data/phrasebook/index.json');
    const index = indexData.default || indexData;
    const categoryIds = index.categories;

    const categories: CategoryInfo[] = [];

    for (const categoryId of categoryIds) {
      const infoData = await import(
        `../data/phrasebook/${categoryId}/info.json`
      );
      const info = infoData.default || infoData;

      categories.push({
        id: info.id,
        name: info.name,
        emoji: info.emoji,
        description: info.description,
        icon: IconCalendar, // All categories use the same icon type for now
      });
    }

    categoriesCache = categories;
    return categories;
  } catch (error) {
    console.error('Failed to load categories:', error);
    throw new Error(
      'Failed to load phrasebook categories. Ensure all categories have proper info.json files.'
    );
  }
}

/**
 * Get category info by ID
 */
export async function getCategoryInfo(
  categoryId: string
): Promise<CategoryInfo | undefined> {
  const categories = await getAllCategories();
  return categories.find(cat => cat.id === categoryId);
}

/**
 * Get the display name for a category
 */
export async function getCategoryDisplayName(
  categoryId: string
): Promise<string> {
  const category = await getCategoryInfo(categoryId);
  return category?.name || categoryId;
}

/**
 * Get the icon for a category
 */
export async function getCategoryIcon(categoryId: string) {
  const category = await getCategoryInfo(categoryId);
  return category?.icon || IconCalendar;
}

/**
 * Get the emoji for a category
 */
export async function getCategoryEmoji(categoryId: string): Promise<string> {
  const category = await getCategoryInfo(categoryId);
  return category?.emoji || '📚';
}

/**
 * Load category data for a specific language
 */
export async function loadCategoryData(
  categoryId: string
): Promise<PhrasebookData | null> {
  try {
    // Dynamically import the consolidated JSON file
    const data = await import(
      `../data/phrasebook/${categoryId}/${categoryId}.json`
    );
    return data.default || data;
  } catch (error) {
    console.error(`Failed to load phrasebook data for ${categoryId}:`, error);
    return null;
  }
}

/**
 * Get the next category in the list
 */
export async function getNextCategory(
  currentCategoryId: string
): Promise<CategoryInfo | null> {
  const categories = await getAllCategories();
  const currentIndex = categories.findIndex(
    cat => cat.id === currentCategoryId
  );
  if (currentIndex === -1 || currentIndex === categories.length - 1) {
    return null;
  }
  return categories[currentIndex + 1];
}

/**
 * Get the previous category in the list
 */
export async function getPreviousCategory(
  currentCategoryId: string
): Promise<CategoryInfo | null> {
  const categories = await getAllCategories();
  const currentIndex = categories.findIndex(
    cat => cat.id === currentCategoryId
  );
  if (currentIndex <= 0) {
    return null;
  }
  return categories[currentIndex - 1];
}
