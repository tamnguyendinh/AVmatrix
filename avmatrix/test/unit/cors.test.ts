import { describe, expect, it } from 'vitest';
import { isAllowedOrigin } from '../../src/server/api.js';

describe('isAllowedOrigin local-only policy', () => {
  it('allows requests without an Origin header', () => {
    expect(isAllowedOrigin(undefined)).toBe(true);
  });

  it('allows localhost and loopback origins', () => {
    expect(isAllowedOrigin('http://localhost:3000')).toBe(true);
    expect(isAllowedOrigin('https://localhost:5173')).toBe(true);
    expect(isAllowedOrigin('http://127.0.0.1:4747')).toBe(true);
    expect(isAllowedOrigin('https://127.0.0.1')).toBe(true);
    expect(isAllowedOrigin('http://[::1]:3000')).toBe(true);
    expect(isAllowedOrigin('https://[::1]')).toBe(true);
  });

  it('rejects hosted, LAN, and malformed origins', () => {
    expect(isAllowedOrigin('https://avmatrix.vercel.app')).toBe(false);
    expect(isAllowedOrigin('http://10.0.0.5:3000')).toBe(false);
    expect(isAllowedOrigin('http://172.16.5.1:3000')).toBe(false);
    expect(isAllowedOrigin('http://192.168.1.10:3000')).toBe(false);
    expect(isAllowedOrigin('https://example.com')).toBe(false);
    expect(isAllowedOrigin('not-a-url')).toBe(false);
    expect(isAllowedOrigin('')).toBe(false);
  });
});
