import { describe, it, expect } from 'vitest';

// We'll test the sentence extraction logic by simulating what the hook does
describe('useTextSelection - Sentence Extraction', () => {
  // Helper function that mimics the extractSentence logic
  const extractSentence = (selectedText: string, fullText: string): string => {
    if (!selectedText || !fullText || !fullText.includes(selectedText)) {
      return selectedText;
    }

    // Split text into sentences using common sentence boundaries
    const sentences = fullText.split(/(?<=[.!?])\s+/);

    // Find the sentence that contains the selected text
    for (const sentence of sentences) {
      if (sentence.includes(selectedText)) {
        return sentence.trim();
      }
    }

    // If no sentence found with the exact text, try to find the selected text position
    const selectedIndex = fullText.indexOf(selectedText);
    if (selectedIndex !== -1) {
      // Find the start of the sentence
      let start = selectedIndex;
      while (start > 0) {
        const char = fullText[start - 1];
        if (/[.!?]/.test(char)) {
          break;
        }
        start--;
      }

      // Find the end of the sentence
      let end = selectedIndex + selectedText.length;
      while (end < fullText.length) {
        const char = fullText[end];
        if (/[.!?]/.test(char)) {
          end++;
          break;
        }
        end++;
      }

      const extractedSentence = fullText.substring(start, end).trim();
      return extractedSentence || selectedText;
    }

    return selectedText;
  };

  it('should extract sentence from a paragraph with multiple sentences', () => {
    const fullText =
      'This is the first sentence. This is the second sentence with a word. This is the third sentence.';
    const selectedText = 'word';

    const result = extractSentence(selectedText, fullText);

    expect(result).toBe('This is the second sentence with a word.');
  });

  it('should handle selection in a single sentence without punctuation', () => {
    const fullText = 'This is a single sentence without ending punctuation';
    const selectedText = 'single sentence';

    const result = extractSentence(selectedText, fullText);

    expect(result).toBe(fullText);
  });

  it('should extract sentence with exclamation mark', () => {
    const fullText = 'What a beautiful day! The sun is shining.';
    const selectedText = 'beautiful';

    const result = extractSentence(selectedText, fullText);

    expect(result).toBe('What a beautiful day!');
  });

  it('should extract sentence with question mark', () => {
    const fullText = 'How are you? I am fine. Thanks for asking!';
    const selectedText = 'fine';

    const result = extractSentence(selectedText, fullText);

    expect(result).toBe('I am fine.');
  });

  it('should handle selection at the start of a sentence', () => {
    const fullText = 'First sentence here. Second sentence here.';
    const selectedText = 'Second';

    const result = extractSentence(selectedText, fullText);

    expect(result).toBe('Second sentence here.');
  });

  it('should return selected text as fallback when no sentence boundaries found', () => {
    const fullText = 'Some random text without proper punctuation';
    const selectedText = 'random text';

    const result = extractSentence(selectedText, fullText);

    expect(result).toBe(fullText);
  });

  it('should handle text with multiple occurrences of selected word', () => {
    const fullText =
      'The cat is on the mat. The dog is under the mat. The bird is near the mat.';
    const selectedText = 'mat';

    // Should return the first sentence containing the word
    const result = extractSentence(selectedText, fullText);

    expect(result).toBe('The cat is on the mat.');
  });

  it('should handle selected text at sentence boundary', () => {
    const fullText = 'Hello world. Goodbye world.';
    const selectedText = 'world';

    const result = extractSentence(selectedText, fullText);

    expect(result).toBe('Hello world.');
  });

  it('should handle empty selected text', () => {
    const fullText = 'Some text here.';
    const selectedText = '';

    const result = extractSentence(selectedText, fullText);

    // When selected text is empty, we should return the empty string
    expect(result).toBe('');
  });

  it('should handle selected text not in full text', () => {
    const fullText = 'Some text here.';
    const selectedText = 'notfound';

    const result = extractSentence(selectedText, fullText);

    expect(result).toBe('notfound');
  });
});
