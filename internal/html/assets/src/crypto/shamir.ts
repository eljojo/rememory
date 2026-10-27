// Shamir's Secret Sharing - using privy-io's peer-reviewed implementation
// https://github.com/privy-io/shamir-secret-sharing
//
// This library has been independently audited by:
// - Cure53: https://cure53.de/audit-report_privy-sss-library.pdf
// - Zellic: https://github.com/Zellic/publications/blob/master/Privy_Shamir_Secret_Sharing_-_Zellic_Audit_Report.pdf
//
// Uses GF(2^8) with the same share format as HashiCorp Vault:
//   [y-values (secret.length bytes)][x-coordinate (1 byte)]

import { combine as privyCombine } from 'shamir-secret-sharing';

/**
 * Combine shares using the privy-io library to recover the secret.
 * Share format is compatible with HashiCorp Vault.
 */
export async function combine(shares: Uint8Array[]): Promise<Uint8Array> {
  if (shares.length < 2) {
    throw new Error(`need at least 2 shares, got ${shares.length}`);
  }

  // Verify all shares have the same length
  const shareLen = shares[0].length;
  for (const share of shares) {
    if (share.length !== shareLen) {
      throw new Error('all shares must have the same length');
    }
  }

  // Split never issues x=0 (that point is the secret itself). The library
  // doesn't reject it in combine, and interpolating at 0 returns such a
  // share's y-values verbatim — so a crafted x=0 share would let its supplier
  // choose the recovered secret. Mirrors core.Combine.
  shares.forEach((share, i) => {
    if (share.length > 0 && share[share.length - 1] === 0) {
      throw new Error(`share ${i + 1} is invalid (x-coordinate 0)`);
    }
  });

  return privyCombine(shares);
}

/**
 * Convert recovered secret bytes to passphrase string.
 * - V1: secret bytes are the raw passphrase string (ASCII)
 * - V2: secret bytes are base64url-encoded to get the passphrase
 */
export function recoverPassphrase(secret: Uint8Array, version: number): string {
  if (version === 1) {
    // V1: raw string bytes
    return new TextDecoder().decode(secret);
  } else {
    // V2: base64url encode the raw bytes
    let base64 = btoa(String.fromCharCode.apply(null, Array.from(secret)));
    // Convert to base64url: replace + with -, / with _, remove =
    base64 = base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
    return base64;
  }
}

/**
 * Decode base64 (standard or URL-safe) to bytes.
 */
export function base64ToBytes(b64: string): Uint8Array {
  // Convert base64url to standard base64
  let standard = b64.replace(/-/g, '+').replace(/_/g, '/');
  // Add padding if needed
  while (standard.length % 4 !== 0) {
    standard += '=';
  }
  const binary = atob(standard);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

/**
 * Encode bytes to base64url (no padding).
 */
export function bytesToBase64(bytes: Uint8Array): string {
  let base64 = btoa(String.fromCharCode.apply(null, Array.from(bytes)));
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}
