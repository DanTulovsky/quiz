/// <reference types="vite/client" />
import Axios from 'axios';
import { AxiosRequestConfig } from 'axios';

interface CancelablePromise<T> extends Promise<T> {
  cancel: () => void;
}

export const AXIOS_INSTANCE = Axios.create({
  baseURL: import.meta.env.VITE_API_URL,
});

export function customInstance<T>(
  config: AxiosRequestConfig,
  options?: AxiosRequestConfig
): CancelablePromise<T> {
  const source = Axios.CancelToken.source();
  const promise = AXIOS_INSTANCE({
    ...config,
    ...options,
    cancelToken: source.token,
  }).then(({ data }) => data) as CancelablePromise<T>;

  promise.cancel = () => {
    source.cancel('Query was cancelled');
  };

  return promise;
}

export default customInstance;
