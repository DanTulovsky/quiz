import { describe, it, expect } from 'vitest';
import { splitIntoParagraphs } from '../passage';

describe('splitIntoParagraphs', () => {
  it('splits text into groups of N sentences', () => {
    const text =
      'Sentence one. Sentence two! Sentence three? Sentence four. Sentence five.';
    const result = splitIntoParagraphs(text, 2);
    expect(result).toEqual([
      'Sentence one. Sentence two!',
      'Sentence three? Sentence four.',
      'Sentence five.',
    ]);
  });

  it('handles empty input', () => {
    expect(splitIntoParagraphs('', 3)).toEqual([]);
  });

  it('handles fewer sentences than perParagraph', () => {
    const text = 'Only one sentence.';
    expect(splitIntoParagraphs(text, 5)).toEqual(['Only one sentence.']);
  });
});
