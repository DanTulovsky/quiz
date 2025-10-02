// Utility helpers for splitting passage text into readable paragraphs.
// Keep implementation conservative to avoid mis-splitting on unusual punctuation.
export function splitIntoParagraphs(
  passage: string | null | undefined,
  sentencesPerParagraph: number
): string[] {
  if (!passage) return [];

  const text = passage.trim();
  if (!text) return [];

  // Split on sentence boundaries: lookbehind for . ! or ? followed by whitespace.
  // This is intentionally simple and works for the vast majority of short passages.
  const rawSentences = text.split(/(?<=[.!?])\s+/);

  // Fallback: if splitting produced no useful pieces, return the whole text as one paragraph
  if (!rawSentences || rawSentences.length === 0) return [text];

  const paragraphs: string[] = [];
  for (let i = 0; i < rawSentences.length; i += sentencesPerParagraph) {
    const chunk = rawSentences.slice(i, i + sentencesPerParagraph).join(' ');
    paragraphs.push(chunk.trim());
  }

  return paragraphs;
}
