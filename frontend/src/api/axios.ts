/// <reference types="vite/client" />
import Axios from 'axios';
import { AxiosRequestConfig } from 'axios';

interface CancelablePromise<T> extends Promise<T> {
  cancel: () => void;
}

export const AXIOS_INSTANCE = Axios.create({
  baseURL: import.meta.env.VITE_API_URL,
});

// Response interceptor removed since we're handling errors in customInstance

export function customInstance<T>(
  config: AxiosRequestConfig,
  options?: AxiosRequestConfig
): CancelablePromise<T> {
  const source = Axios.CancelToken.source();
  const promise = AXIOS_INSTANCE({
    ...config,
    ...options,
    cancelToken: source.token,
  })
    .then((response) => {
      // For successful responses, return the data
      return response.data;
    })
    .catch((error) => {
      // For all errors, return a rejected promise with the error
      // Axios errors already have isAxiosError property set by axios
      return Promise.reject(error);
    }) as CancelablePromise<T>;

  promise.cancel = () => {
    source.cancel('Query was cancelled');
  };

  return promise;
}

export default customInstance;
