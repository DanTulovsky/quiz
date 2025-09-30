import { test, expect } from '@playwright/test';
import {assertStatus} from './http-assert';
import { resetTestDatabase } from './reset-db';

// Session name constant - should match backend config.SessionName
const SESSION_NAME = 'quiz-session';

test.beforeAll(() => {
  resetTestDatabase();
});

/**
 * API E2E tests for authentication endpoints
 * These tests use Playwright's API testing capabilities to test the backend directly
 */

test.describe('API Authentication', () => {
  const baseURL = process.env.TEST_BASE_URL || 'http://localhost:3001';

  test('should login successfully with valid credentials', async ({ request }) => {
    const loginData = {
      username: 'testuser',
      password: 'password'
    };

    const response = await request.post(`${baseURL}/v1/auth/login`, {
      data: loginData,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    await assertStatus(response, 200, {
      method: 'POST',
      url: `${baseURL}/v1/auth/login`,
      requestHeaders: {'Content-Type': 'application/json'},
      requestBody: loginData
    });

    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('user');
    expect(responseBody.user).toHaveProperty('id');
    expect(responseBody.user).toHaveProperty('username');
    expect(responseBody.user.username).toBe('testuser');

    // Check that session cookie is set
    const cookies = response.headers()['set-cookie'];
    expect(cookies).toBeDefined();
    expect(cookies).toContain('session=');
  });

  test('should return 401 with invalid credentials', async ({ request }) => {
    const loginData = {
      username: 'invalid',
      password: 'wrongpassword'
    };

    const response = await request.post(`${baseURL}/v1/auth/login`, {
      data: loginData,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    await assertStatus(response, 401, {
      method: 'POST',
      url: `${baseURL}/v1/auth/login`,
      requestHeaders: {'Content-Type': 'application/json'},
      requestBody: loginData
    });

    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('error');
    expect(responseBody.error).toContain('Invalid credentials');
  });

  test('should return 401 with missing username', async ({ request }) => {
    const loginData = {
      password: 'password'
    };

    const response = await request.post(`${baseURL}/v1/auth/login`, {
      data: loginData,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    // Align with swagger.yaml: missing required field -> 400 Invalid request format
    await assertStatus(response, 400, {
      method: 'POST',
      url: `${baseURL}/v1/auth/login`,
      requestHeaders: {'Content-Type': 'application/json'},
      requestBody: loginData
    });

    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('error');
  });

  test('should return 401 with missing password', async ({ request }) => {
    const loginData = {
      username: 'testuser'
    };

    const response = await request.post(`${baseURL}/v1/auth/login`, {
      data: loginData,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    // Align with swagger.yaml: missing required field -> 400 Invalid request format
    await assertStatus(response, 400, {
      method: 'POST',
      url: `${baseURL}/v1/auth/login`,
      requestHeaders: {'Content-Type': 'application/json'},
      requestBody: loginData
    });

    const responseBody = await response.json();
    expect(responseBody).toHaveProperty('error');
  });

  test('should check authentication status', async ({ request }) => {
    // First login to get a session
    const loginData = {
      username: 'testuser',
      password: 'password'
    };

    const loginResponse = await request.post(`${baseURL}/v1/auth/login`, {
      data: loginData,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    await assertStatus(loginResponse, 200, {
      method: 'POST',
      url: `${baseURL}/v1/auth/login`,
      requestHeaders: {'Content-Type': 'application/json'},
      requestBody: loginData
    });

    // Get the session cookie
    const cookies = loginResponse.headers()['set-cookie'];
    const sessionCookie = cookies?.split(',').find(cookie => cookie.trim().startsWith(`${SESSION_NAME}=`));

    // Check authentication status with session
    const statusResponse = await request.get(`${baseURL}/v1/auth/status`, {
      headers: {
        'Cookie': sessionCookie || ''
      }
    });

    await assertStatus(statusResponse, 200, {
      method: 'GET',
      url: `${baseURL}/v1/auth/status`,
      requestHeaders: {Cookie: sessionCookie || ''}
    });

    const statusBody = await statusResponse.json();
    expect(statusBody).toHaveProperty('authenticated');
    expect(statusBody.authenticated).toBe(true);
    expect(statusBody).toHaveProperty('user');
    expect(statusBody.user).toHaveProperty('id');
    expect(statusBody.user).toHaveProperty('username');
    expect(statusBody.user.username).toBe('testuser');
  });

  test('should logout successfully', async ({ request }) => {
    // First login to get a session
    const loginData = {
      username: 'testuser',
      password: 'password'
    };

    const loginResponse = await request.post(`${baseURL}/v1/auth/login`, {
      data: loginData,
      headers: {
        'Content-Type': 'application/json'
      }
    });

    await assertStatus(loginResponse, 200, {
      method: 'POST',
      url: `${baseURL}/v1/auth/login`,
      requestHeaders: {'Content-Type': 'application/json'},
      requestBody: loginData
    });

    // Get the session cookie
    const cookies = loginResponse.headers()['set-cookie'];
    const sessionCookie = cookies?.split(',').find(cookie => cookie.trim().startsWith(`${SESSION_NAME}=`));

    // Logout
    const logoutResponse = await request.post(`${baseURL}/v1/auth/logout`, {
      headers: {
        'Cookie': sessionCookie || ''
      }
    });

    await assertStatus(logoutResponse, 200, {
      method: 'POST',
      url: `${baseURL}/v1/auth/logout`,
      requestHeaders: {Cookie: sessionCookie || ''}
    });

    const logoutBody = await logoutResponse.json();
    expect(logoutBody).toHaveProperty('message');
    expect(logoutBody.message).toContain('success');
  });
});
