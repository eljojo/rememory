// Unlock the archive from pieces that may each have more than one reading.
import { decrypt } from './age';
import { combine, recoverPassphrase } from './shamir';

// Each extra reading doubles the combinations, and each try costs an scrypt
// derivation. Real ambiguity is one piece with two readings; this only
// bounds pathological input.
const MAX_COMBINATIONS = 16;

/**
 * Combine the pieces and decrypt the manifest. candidates[i] lists piece i's
 * possible data, most likely first: recovery words can fit two word lists
 * with a valid checksum, and only decryption can tell which is right.
 * Combinations are tried most likely first; if none opens the archive, the
 * error from the most likely one is thrown.
 */
export async function unlockWithCandidates(
  manifest: Uint8Array,
  version: number,
  candidates: Uint8Array[][],
): Promise<Uint8Array> {
  let firstError: unknown = null;
  let tried = 0;
  for (const pieces of combinations(candidates)) {
    if (tried++ >= MAX_COMBINATIONS) break;
    try {
      const secret = await combine(pieces);
      return await decrypt(manifest, recoverPassphrase(secret, version));
    } catch (err) {
      if (firstError === null) firstError = err;
    }
  }
  throw firstError ?? new Error('no pieces to combine');
}

// All picks of one reading per piece, the first readings first.
function* combinations(candidates: Uint8Array[][], i = 0, picked: Uint8Array[] = []): Generator<Uint8Array[]> {
  if (i === candidates.length) {
    yield picked;
    return;
  }
  for (const option of candidates[i]) {
    yield* combinations(candidates, i + 1, [...picked, option]);
  }
}
