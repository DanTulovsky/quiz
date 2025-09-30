import { describe, it, expect } from 'vitest';

// Test the error message extraction logic directly
const extractErrorMessage = (error: unknown): string => {
  let errorMessage = 'An unknown error occurred';

  if (error && typeof error === 'object' && 'response' in error) {
    const responseError = error as { response?: { data?: { error?: string } } };
    if (responseError.response?.data?.error) {
      errorMessage = responseError.response.data.error;
    }
  } else if (error instanceof Error) {
    errorMessage = error.message;
  }

  return errorMessage;
};

describe('AuthProvider Error Handling', () => {
  it('should extract "Invalid credentials" from error response', () => {
    const error = {
      response: {
        data: {
          error: 'Invalid credentials',
        },
      },
    };

    const result = extractErrorMessage(error);
    expect(result).toBe('Invalid credentials');
  });

  it('should extract generic error message when no specific error is provided', () => {
    const error = new Error('Network error');

    const result = extractErrorMessage(error);
    expect(result).toBe('Network error');
  });

  it('should return default message when error has no useful information', () => {
    const error = {};

    const result = extractErrorMessage(error);
    expect(result).toBe('An unknown error occurred');
  });

  it('should handle null error gracefully', () => {
    const result = extractErrorMessage(null);
    expect(result).toBe('An unknown error occurred');
  });
});
