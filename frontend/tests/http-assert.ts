import {expect, APIResponse} from '@playwright/test';

type Expected = number | number[];

export async function assertStatus(
  response: APIResponse,
  expected: Expected,
  meta?: {
    method?: string;
    url?: string;
    requestHeaders?: Record<string, any>;
    requestBody?: any;
  }
) {
  const expectedArr = Array.isArray(expected) ? expected : [expected];
  const status = response.status();
  if (!expectedArr.includes(status)) {
    const bodyText = await response.text();
    const respHeaders = response.headers();
    const sanitizedHeaders = {...(meta?.requestHeaders || {})};
    if (sanitizedHeaders['Cookie']) sanitizedHeaders['Cookie'] = '[REDACTED]';
    if (sanitizedHeaders['cookie']) sanitizedHeaders['cookie'] = '[REDACTED]';
    if (sanitizedHeaders['Authorization']) sanitizedHeaders['Authorization'] = '[REDACTED]';
    if (sanitizedHeaders['authorization']) sanitizedHeaders['authorization'] = '[REDACTED]';
    // Emit diagnostics to stderr so they show up on failures
    console.error('\n❌ HTTP status assertion failed', {
      expected: expectedArr,
      actual: status,
      method: meta?.method,
      url: meta?.url,
      requestHeaders: sanitizedHeaders,
      requestBody: meta?.requestBody,
      responseHeaders: respHeaders,
      responseBody: bodyText
    });
  }
  expect(expectedArr.includes(status)).toBe(true);
}


