// Verb conjugation data structures and utilities

export interface Conjugation {
  pronoun: string;
  form: string;
  exampleSentence: string;
  exampleSentenceEn: string;
}

export interface Tense {
  tenseId: string;
  tenseName: string;
  tenseNameEn: string;
  description: string;
  conjugations: Conjugation[];
}

export interface VerbConjugation {
  infinitive: string;
  infinitiveEn: string;
  category: string;
  tenses: Tense[];
}

export interface VerbConjugationsData {
  language: string;
  languageName: string;
  verbs: VerbConjugation[];
}

export interface VerbConjugationInfo {
  id: string;
  name: string;
  emoji: string;
  description: string;
}

/**
 * Load verb conjugation data for a specific language from the API
 */
export async function loadVerbConjugations(
  languageCode: string
): Promise<VerbConjugationsData | null> {
  try {
    const response = await fetch(`/v1/verb-conjugations/${languageCode}`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    return await response.json();
  } catch (error) {
    console.error(
      `Failed to load verb conjugations for ${languageCode}:`,
      error
    );
    return null;
  }
}

/**
 * Load a specific verb's conjugations from the API
 */
export async function loadVerbConjugation(
  languageCode: string,
  verbInfinitive: string
): Promise<VerbConjugation | null> {
  try {
    const response = await fetch(
      `/v1/verb-conjugations/${languageCode}/${verbInfinitive}`
    );
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    return await response.json();
  } catch (error) {
    console.error(
      `Failed to load verb conjugation for ${verbInfinitive} in ${languageCode}:`,
      error
    );
    return null;
  }
}

/**
 * Get verb conjugation metadata from the API
 */
export async function getVerbConjugationInfo(): Promise<VerbConjugationInfo | null> {
  try {
    const response = await fetch('/v1/verb-conjugations/info');
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    return await response.json();
  } catch (error) {
    console.error('Failed to load verb conjugation info:', error);
    return null;
  }
}

/**
 * Get available languages for verb conjugations from the API
 */
export async function getAvailableLanguages(): Promise<string[]> {
  try {
    const response = await fetch('/v1/verb-conjugations/languages');
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const data = await response.json();
    // Backend returns the array directly, not wrapped in an object
    return Array.isArray(data) ? data : [];
  } catch (error) {
    console.error('Failed to load available languages:', error);
    return [];
  }
}

/**
 * Get the next verb in the list
 */
export async function getNextVerb(
  currentVerb: string,
  languageCode: string
): Promise<VerbConjugation | null> {
  const data = await loadVerbConjugations(languageCode);
  if (!data) return null;

  const currentIndex = data.verbs.findIndex(v => v.infinitive === currentVerb);
  if (currentIndex === -1 || currentIndex === data.verbs.length - 1)
    return null;

  return data.verbs[currentIndex + 1];
}

/**
 * Get the previous verb in the list
 */
export async function getPreviousVerb(
  currentVerb: string,
  languageCode: string
): Promise<VerbConjugation | null> {
  const data = await loadVerbConjugations(languageCode);
  if (!data) return null;

  const currentIndex = data.verbs.findIndex(v => v.infinitive === currentVerb);
  if (currentIndex <= 0) return null;

  return data.verbs[currentIndex - 1];
}

/**
 * Search verbs by infinitive or English translation
 */
export function searchVerbs(
  verbs: VerbConjugation[],
  query: string
): VerbConjugation[] {
  if (!query.trim()) return verbs;

  const lowercaseQuery = query.toLowerCase();
  return verbs.filter(
    verb =>
      verb.infinitive.toLowerCase().includes(lowercaseQuery) ||
      verb.infinitiveEn.toLowerCase().includes(lowercaseQuery)
  );
}

/**
 * Get verbs by category
 */
export function getVerbsByCategory(
  verbs: VerbConjugation[],
  category: string
): VerbConjugation[] {
  if (category === 'all') return verbs;
  return verbs.filter(verb => verb.category === category);
}

/**
 * Get all unique categories from verbs
 */
export function getCategories(verbs: VerbConjugation[]): string[] {
  const categories = new Set(verbs.map(verb => verb.category));
  return Array.from(categories).sort();
}
