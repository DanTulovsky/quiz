import React, { createContext, useContext, useState, ReactNode } from 'react';
import { apiClient } from '../api/axios';

// Types for translation
export interface TranslationResult {
  translatedText: string;
  sourceLanguage: string;
  targetLanguage: string;
}

export interface TranslationContextType {
  translateText: (
    text: string,
    targetLang?: string
  ) => Promise<TranslationResult>;
  isLoading: boolean;
  error: string | null;
}

// Create the context
const TranslationContext = createContext<TranslationContextType | undefined>(
  undefined
);

// Translation Provider Component
export const TranslationProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastErrorTime, setLastErrorTime] = useState<number>(0);
  const [consecutiveFailures, setConsecutiveFailures] = useState<number>(0);
  const [translationCache, setTranslationCache] = useState<
    Map<string, TranslationResult>
  >(new Map());

  // Cooldown period after failures (in milliseconds)
  const ERROR_COOLDOWN = 5000; // 5 seconds
  const MAX_RETRIES = 3;

  const translateText = async (
    text: string,
    targetLang: string = 'en',
    retryCount: number = 0
  ): Promise<TranslationResult> => {
    // Check cache first
    const cacheKey = `${text.trim()}:${targetLang}`;
    const cachedResult = translationCache.get(cacheKey);
    if (cachedResult) {
      return cachedResult;
    }

    // Check if we're in a cooldown period after failures
    const now = Date.now();
    if (
      consecutiveFailures >= MAX_RETRIES &&
      now - lastErrorTime < ERROR_COOLDOWN
    ) {
      throw new Error(
        'Translation service temporarily unavailable. Please try again later.'
      );
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await apiClient.post('/v1/translate', {
        text: text,
        target_language: targetLang,
        source_language: undefined, // Let backend auto-detect or handle as needed
      });

      // Success - reset failure counters and cache result
      setConsecutiveFailures(0);
      setLastErrorTime(0);

      const result = {
        translatedText: response.data.translated_text,
        sourceLanguage: response.data.source_language,
        targetLanguage: response.data.target_language,
      };

      // Cache the result
      setTranslationCache(prev => new Map(prev.set(cacheKey, result)));

      return result;
    } catch (err: unknown) {
      const errorMessage =
        err?.response?.data?.message || err?.message || 'Translation failed';

      // Track consecutive failures
      setConsecutiveFailures(prev => prev + 1);
      setLastErrorTime(now);

      // Handle specific error cases
      if (err?.response?.status === 429) {
        setError('Rate limit exceeded. Please wait a moment and try again.');
        throw new Error('Rate limit exceeded');
      } else if (err?.response?.status === 503) {
        setError('Translation service is temporarily unavailable.');
        throw new Error('Translation service unavailable');
      } else if (err?.response?.status >= 500) {
        setError('Server error. Please try again later.');
        throw new Error('Server error');
      }

      // For other errors, retry with exponential backoff
      if (retryCount < MAX_RETRIES) {
        const backoffDelay = Math.min(1000 * Math.pow(2, retryCount), 5000); // Max 5 seconds
        await new Promise(resolve => setTimeout(resolve, backoffDelay));

        return translateText(text, targetLang, retryCount + 1);
      }

      setError(errorMessage);
      throw new Error(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const value: TranslationContextType = {
    translateText,
    isLoading,
    error,
  };

  return (
    <TranslationContext.Provider value={value}>
      {children}
    </TranslationContext.Provider>
  );
};

// Hook to use translation context
export const useTranslation = (): TranslationContextType => {
  const context = useContext(TranslationContext);
  if (context === undefined) {
    throw new Error('useTranslation must be used within a TranslationProvider');
  }
  return context;
};
