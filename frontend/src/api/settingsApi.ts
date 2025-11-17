import { AXIOS_INSTANCE } from './axios';

export const clearAllStories = async () => {
  const res = await AXIOS_INSTANCE.post('/v1/settings/clear-stories', {}, {
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
  });
  return res.data;
};

export const resetAccount = async () => {
  const res = await AXIOS_INSTANCE.post('/v1/settings/reset-account', {}, {
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
  });
  return res.data;
};

export const clearAllAIChats = async () => {
  const res = await AXIOS_INSTANCE.post('/v1/settings/clear-ai-chats', {}, {
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
  });
  return res.data;
};

export const clearAllSnippets = async () => {
  const res = await AXIOS_INSTANCE.delete('/v1/snippets', {
    headers: { Accept: 'application/json' },
  });
  return res.data;
};

export const clearAllTranslationPracticeHistory = async () => {
  const res = await AXIOS_INSTANCE.post('/v1/settings/clear-translation-practice-history', {}, {
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
  });
  return res.data;
};

export const updateWordOfDayEmailPreference = async (enabled: boolean) => {
  const res = await AXIOS_INSTANCE.put(
    '/v1/settings/word-of-day-email',
    { enabled },
    { headers: { Accept: 'application/json', 'Content-Type': 'application/json' } }
  );
  return res.data as { success: boolean };
};



