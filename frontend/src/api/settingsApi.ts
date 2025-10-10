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



