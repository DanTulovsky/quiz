// Utilities for splitting reading passages into readable paragraphs
// Keep implementation small and dependency-free so it is easy to test.

/**
 * Split a passage into sentence strings.
 * Uses a reasonably robust regex to capture sentence-ending punctuation.
 */
export function splitSentences(passage: string): string[] {
  if (!passage) return [];
  // Normalize whitespace
  const normalized = passage.replace(/\s+/g, ' ').trim();

  // Match sentences including their terminating punctuation if present.
  // This will capture segments like "Hello world." "Is it ok?" and also
  // trailing fragments without punctuation.
  const matches = normalized.match(/[^.!?]+[.!?]+[\)"']*|[^.!?]+$/g);
  return matches ? matches.map(s => s.trim()) : [normalized];
}

/**
 * Group sentences into paragraphs of up to `sentencesPerParagraph` sentences.
 */
export function splitIntoParagraphs(
  passage: string,
  sentencesPerParagraph: number
): string[] {
  if (!passage) return [];
  if (!sentencesPerParagraph || sentencesPerParagraph <= 0)
    sentencesPerParagraph = 3;

  const sentences = splitSentences(passage);
  const paragraphs: string[] = [];
  for (let i = 0; i < sentences.length; i += sentencesPerParagraph) {
    const group = sentences.slice(i, i + sentencesPerParagraph);
    paragraphs.push(group.join(' '));
  }
  return paragraphs;
}

export default {
  splitSentences,
  splitIntoParagraphs,
};

