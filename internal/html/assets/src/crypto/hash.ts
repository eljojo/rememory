// SHA-256 hashing utilities using @noble/hashes (pure JS, works in any context)

import { sha256 as nobleSha256 } from '@noble/hashes/sha2.js';

/**
 * Compute SHA-256 hash of data.
 */
export async function sha256(data: Uint8Array): Promise<Uint8Array> {
  return nobleSha256(data);
}

/**
 * Compute SHA-256 hash and return as hex string with "sha256:" prefix.
 */
export async function hashBytes(data: Uint8Array): Promise<string> {
  const hash = await sha256(data);
  const hex = Array.from(hash)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
  return `sha256:${hex}`;
}

/**
 * Verify a hash matches expected value.
 *
 * Note: We use standard equality here, not constant-time comparison.
 * Timing-safe comparison prevents attackers from learning secrets through
 * timing differences. But here both values are derived from public data
 * (the share string the user provided) — there's no secret to leak.
 * This is data integrity verification, not authentication.
 */
export function verifyHash(got: string, expected: string): boolean {
  return got === expected;
}
