/// <reference types="vite/client" />
import Axios from 'axios';
import { AxiosRequestConfig } from 'axios';

interface CancelablePromise<T> extends Promise<T> {
  cancel: () => void;
}

export const AXIOS_INSTANCE = Axios.create({
  baseURL: import.meta.env.VITE_API_URL,
});

// Add response interceptor to handle errors properly
AXIOS_INSTANCE.interceptors.response.use(
  (response) => response,
  (error) => {
    // If error has response data with error/message fields, return the error object
    // This allows our error handling logic to work properly
    if (error.response?.data) {
      return Promise.reject(error);
    }
    // For other errors, return the original error
    return Promise.reject(error);
  }
);

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
    .then(({ data }) => data)
    .catch((error) => {
      // If the error has response data, we want to preserve the original error
      // so that our error handling logic can parse it properly
      if (error.response?.data) {
        throw error;
      }
      // For other errors, throw the original error
      throw error;
    }) as CancelablePromise<T>;

  promise.cancel = () => {
    source.cancel('Query was cancelled');
  };

  return promise;
}

export default customInstance;
