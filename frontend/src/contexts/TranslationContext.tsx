import React, {
  createContext,
  useContext,
  useState,
  ReactNode,
  useCallback,
} from 'react';
import { apiClient } from '../api/axios';
import { useAuth } from '../hooks/useAuth';

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
  translation: TranslationResult | null;
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
  const [translation, setTranslation] = useState<TranslationResult | null>(
    null
  );
  const [lastErrorTime, setLastErrorTime] = useState<number>(0);
  const [consecutiveFailures, setConsecutiveFailures] = useState<number>(0);
  const [translationCache, setTranslationCache] = useState<
    Map<string, TranslationResult>
  >(new Map());

  // Get user data from AuthContext instead of localStorage
  const { user } = useAuth();

  // Cooldown period after failures (in milliseconds)
  const ERROR_COOLDOWN = 5000; // 5 seconds
  const MAX_RETRIES = 3;

  const translateText = useCallback(
    async (
      text: string,
      targetLang: string = 'en',
      retryCount: number = 0
    ): Promise<TranslationResult> => {
      // Check cache first - include source language in cache key for accuracy
      const sourceLang = user?.preferred_language;
      const cacheKey = sourceLang
        ? `${text.trim()}:${sourceLang}:${targetLang}`
        : `${text.trim()}:${targetLang}`;
      const cachedResult = translationCache.get(cacheKey);
      if (cachedResult) {
        // Update the current translation state for UI even with cached result
        setTranslation(cachedResult);
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
        // Get user's preferred language (source language) from AuthContext
        const sourceLang = user?.preferred_language;

        if (!sourceLang) {
          throw new Error(
            'User preferred language not set. Please configure your language settings.'
          );
        }

        // Don't make API call if source and target languages are the same
        if (sourceLang === targetLang) {
          const result = {
            translatedText: text, // Return original text when source = target
            sourceLanguage: sourceLang,
            targetLanguage: targetLang,
          };

          // Update the current translation state
          setTranslation(result);

          return result;
        }

        const response = await apiClient.post('/v1/translate', {
          text: text,
          target_language: targetLang,
          source_language: sourceLang,
        });

        // Success - reset failure counters and cache result
        setConsecutiveFailures(0);
        setLastErrorTime(0);

        const result = {
          translatedText: response.data.translated_text,
          sourceLanguage: response.data.source_language,
          targetLanguage: response.data.target_language,
        };

        // Cache the result with updated cache key format
        const finalCacheKey = sourceLang
          ? `${text.trim()}:${sourceLang}:${targetLang}`
          : `${text.trim()}:${targetLang}`;
        setTranslationCache(prev => new Map(prev.set(finalCacheKey, result)));

        // Update the current translation state
        setTranslation(result);

        return result;
      } catch (err: unknown) {
        const error = err as {
          response?: { data?: { message?: string }; status?: number };
          message?: string;
        };
        const errorMessage =
          error?.response?.data?.message ||
          error?.message ||
          'Translation failed';

        // Don't retry on 400 errors (bad request) as they won't be fixed by retrying
        const shouldRetry =
          !error?.response?.status || error?.response?.status >= 500;

        if (!shouldRetry || retryCount >= MAX_RETRIES) {
          // Track consecutive failures only for retryable errors
          if (shouldRetry) {
            setConsecutiveFailures(prev => prev + 1);
            setLastErrorTime(now);
          }

          // Handle specific error cases
          if (error?.response?.status === 429) {
            setError(
              'Rate limit exceeded. Please wait a moment and try again.'
            );
            throw new Error('Rate limit exceeded');
          }

          setError(errorMessage);
          throw new Error(errorMessage);
        }

        // Retry after a delay for retryable errors
        await new Promise(resolve =>
          setTimeout(resolve, 1000 * (retryCount + 1))
        );
        return translateText(text, targetLang, retryCount + 1);

        setError(errorMessage);
        throw new Error(errorMessage);
      } finally {
        setIsLoading(false);
      }
    },
    [
      translationCache,
      user?.preferred_language,
      consecutiveFailures,
      lastErrorTime,
    ]
  );

  const value: TranslationContextType = {
    translateText,
    translation,
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
