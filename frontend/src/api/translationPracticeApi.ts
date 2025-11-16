import { useMutation, useQuery } from '@tanstack/react-query';
import { AXIOS_INSTANCE } from './axios';

export type TranslationDirection = 'en_to_learning' | 'learning_to_en' | 'random';

export type GenerateRequest = {
  language: string;
  level: string;
  direction: TranslationDirection;
  topic?: string;
};

export type SentenceResponse = {
  id: number;
  sentence_text: string;
  source_language: string;
  target_language: string;
  language_level: string;
  source_type: string;
  source_id?: number | null;
  topic?: string | null;
  created_at: string;
};

export type SubmitRequest = {
  sentence_id: number;
  original_sentence: string;
  user_translation: string;
  translation_direction: TranslationDirection;
};

export type SessionResponse = {
  id: number;
  sentence_id: number;
  original_sentence: string;
  user_translation: string;
  translation_direction: TranslationDirection;
  ai_feedback: string;
  ai_score?: number | null;
  created_at: string;
};

export type HistoryResponse = {
  sessions: SessionResponse[];
  total: number;
  limit: number;
  offset: number;
};

export type StatsResponse = {
  total_sessions?: number;
  average_score?: number | null;
  min_score?: number | null;
  max_score?: number | null;
  excellent_count?: number;
  good_count?: number;
  needs_improvement_count?: number;
};

export const useGeneratePracticeSentence = () =>
  useMutation({
    mutationFn: async (body: GenerateRequest): Promise<SentenceResponse> => {
      const resp = await AXIOS_INSTANCE.post('/v1/translation-practice/generate', body, {
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      });
      return resp.data as SentenceResponse;
    },
  });

export const useGetPracticeSentence = (params: {
  language?: string;
  level?: string;
  direction?: TranslationDirection;
  enabled?: boolean;
}) =>
  useQuery({
    queryKey: ['tp-sentence', params.language, params.level, params.direction],
    enabled: Boolean(params.language && params.level && params.direction && params.enabled !== false),
    queryFn: async (): Promise<SentenceResponse> => {
      const qs = new URLSearchParams({
        language: params.language!,
        level: params.level!,
        direction: params.direction!,
      });
      const resp = await AXIOS_INSTANCE.get(`/v1/translation-practice/sentence?${qs.toString()}`, {
        headers: { Accept: 'application/json' },
      });
      return resp.data as SentenceResponse;
    },
  });

export const useSubmitTranslation = () =>
  useMutation({
    mutationFn: async (body: SubmitRequest): Promise<SessionResponse> => {
      const resp = await AXIOS_INSTANCE.post('/v1/translation-practice/submit', body, {
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      });
      return resp.data as SessionResponse;
    },
  });

export const usePracticeHistory = (limit: number = 20, offset: number = 0, search?: string) =>
  useQuery({
    queryKey: ['tp-history', limit, offset, search],
    queryFn: async (): Promise<HistoryResponse> => {
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
      });
      if (search && search.trim()) {
        params.append('search', search.trim());
      }
      const resp = await AXIOS_INSTANCE.get(`/v1/translation-practice/history?${params.toString()}`, {
        headers: { Accept: 'application/json' },
      });
      return resp.data as HistoryResponse;
    },
    staleTime: 30000,
  });

export const usePracticeStats = () =>
  useQuery({
    queryKey: ['tp-stats'],
    queryFn: async (): Promise<StatsResponse> => {
      const resp = await AXIOS_INSTANCE.get('/v1/translation-practice/stats', {
        headers: { Accept: 'application/json' },
      });
      return resp.data as StatsResponse;
    },
    staleTime: 30000,
  });
