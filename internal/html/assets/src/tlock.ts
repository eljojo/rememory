// tlock.ts — Custom tlock decryption for drand quicknet (recovery path only).
// Replaces tlock-js 0.9.0 + drand-client 1.4.2 in the recovery bundle
// (app-tlock.js). The create path (create-app.js) still uses tlock-js directly.
//
// Uses age-encryption's Decrypter for the age layer (armor, header parsing,
// MAC verification, STREAM cipher). Only implements the tlock-specific IBE
// unwrap and drand beacon fetch/verify.
//
// Only supports bls-unchained-g1-rfc9380 (quicknet) — the only scheme
// rememory uses. Signatures on G1, public key on G2.
//
// === Sources ===
//
// tlock-js 0.9.0 (Apache-2.0 OR MIT)
//   commit 17d817ee259e79381111dd75009b0f022c39ace3
//   https://github.com/drand/tlock-js
//
//   src/crypto/utils.ts            → xor, bytesToNumberBE, bytesToHex (from Noble),
//                                    fpToBytes, fp2ToBytes, fp6ToBytes, fp12ToBytes
//   src/crypto/ibe.ts              → decryptOnG2, gtToHash (H2), h3 (H3), h4 (H4)
//   src/drand/timelock-decrypter.ts → parseCiphertext (U || V || W split)
//
// drand-client 1.4.2 (Apache-2.0 OR MIT)
//   commit ef8c9260294f8699b5e8c27a6b764f8f0d768bea
//   https://github.com/drand/drand-client
//
//   lib/beacon-verification.ts     → verifyBeacon: randomnessIsValid, roundBuffer,
//                                    the DST 'BLS_SIG_BLS12381G1_XMD:SHA-256_SSWU_RO_NUL_'
//
// === What changed from the sources ===
//
// 1. Node.js Buffer replaced with Uint8Array throughout (browser compatibility).
//    fpToBytes uses parseInt hex loop instead of Buffer.write(hex,"hex").
//    fp2/fp6/fp12ToBytes use a concatBytes helper instead of Buffer.concat.
//    sha256.create().update() receives TextEncoder output instead of raw strings
//    (our version of @noble/hashes types update() as Uint8Array-only).
//
// 2. BLS signature verification uses bls12_381.shortSignatures.verify() from
//    @noble/curves instead of drand-client's manual pairing in verifySigOnG1().
//    The library function is audited; the manual pairing existed because
//    @noble/curves lacked G1 signature verification when drand-client was written.
//
// 3. parseCiphertext inlined into decryptOnG2.
//
// Specs:
//   age format    — https://github.com/C2SP/C2SP/blob/main/age.md
//   drand beacon  — https://drand.love/docs/specification/

import { Decrypter } from 'age-encryption';
import type { Identity, Stanza } from 'age-encryption';
import { bls12_381 } from '@noble/curves/bls12-381.js';
import { sha256 } from '@noble/hashes/sha2.js';

const G1 = bls12_381.G1;
const G2 = bls12_381.G2;

// @noble/hashes update() requires Uint8Array in our version's types.
// The original tlock-js passes raw strings, which worked with their version.
const enc = new TextEncoder();

// BLS signature verification DST for quicknet (bls-unchained-g1-rfc9380).
// drand-client lib/beacon-verification.ts (isG1Rfc9380 branch, line 46).
// https://github.com/drand/drand-client/blob/ef8c9260294f8699b5e8c27a6b764f8f0d768bea/lib/beacon-verification.ts#L46
const BLS_VERIFY_DST = 'BLS_SIG_BLS12381G1_XMD:SHA-256_SSWU_RO_NUL_';

// ============================================================================
// From tlock-js src/crypto/utils.ts
// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/crypto/utils.ts
// ============================================================================

////// code from tlock-js src/crypto/utils.ts xor (line 6)
////// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/crypto/utils.ts#L6
////// Verbatim. Removed export.

function xor(a: Uint8Array, b: Uint8Array): Uint8Array {
  if (a.length != b.length) {
    throw new Error("Error: incompatible sizes");
  }

  const ret = new Uint8Array(a.length);

  for (let i = 0; i < a.length; i++) {
    ret[i] = a[i] ^ b[i];
  }

  return ret;
}

////// end of code from tlock-js xor.

////// code from Noble (via tlock-js src/crypto/utils.ts lines 22-37)
////// tlock-js marks this section as "code from Noble":
////// https://github.com/paulmillr/noble-bls12-381/blob/6380415f1b7e5078c8883a5d8d687f2dd3bff6c2/index.ts#L132-L145
////// Verbatim. Removed exports.

const hexes = Array.from({length: 256}, (_v, i) => i.toString(16).padStart(2, '0'));

function bytesToHex(uint8a: Uint8Array): string {
  // pre-caching chars could speed this up 6x.
  let hex = '';
  for (let i = 0; i < uint8a.length; i++) {
    hex += hexes[uint8a[i]];
  }
  return hex;
}

function bytesToNumberBE(uint8a: Uint8Array): bigint {
  return BigInt('0x' + bytesToHex(Uint8Array.from(uint8a)));
}

////// end of code from Noble.

// Fp12 structure types (matches @noble/curves bls12-381 internal representation).
// Original tlock-js imports Fp, Fp2, Fp6, Fp12 from its own fp.ts; we use
// structural types to avoid importing internal noble-curves types.
interface Fp2Like { c0: bigint; c1: bigint }
interface Fp6Like { c0: Fp2Like; c1: Fp2Like; c2: Fp2Like }
interface Fp12Like { c0: Fp6Like; c1: Fp6Like }

// Replaces Buffer.concat (not from tlock-js source).
function concatBytes(...arrays: Uint8Array[]): Uint8Array {
  let totalLen = 0;
  for (const a of arrays) totalLen += a.length;
  const out = new Uint8Array(totalLen);
  let offset = 0;
  for (const a of arrays) { out.set(a, offset); offset += a.length; }
  return out;
}

////// code from tlock-js src/crypto/utils.ts fpToBytes, fp2ToBytes, fp6ToBytes, fp12ToBytes (lines 36-57)
////// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/crypto/utils.ts#L36
////// CHANGED: Buffer.alloc + .write(hex,"hex") → Uint8Array + parseInt loop (browser compat).
////// CHANGED: Buffer.concat → concatBytes helper (browser compat).
////// Iteration order and field layout identical to the original.

// Original:
//   const buf = Buffer.alloc(hex.length / 2)
//   buf.write(hex, "hex")
function fpToBytes(fp: bigint): Uint8Array {
  const hex = fp.toString(16).padStart(96, "0");
  const out = new Uint8Array(48);
  for (let i = 0; i < 48; i++) {
    out[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

// Original: return Buffer.concat([fp2.c1, fp2.c0].map(fpToBytes))
function fp2ToBytes(fp2: Fp2Like): Uint8Array {
  return concatBytes(...[fp2.c1, fp2.c0].map(fpToBytes));
}

// Original: return Buffer.concat([fp6.c2, fp6.c1, fp6.c0].map(fp2ToBytes))
function fp6ToBytes(fp6: Fp6Like): Uint8Array {
  return concatBytes(...[fp6.c2, fp6.c1, fp6.c0].map(fp2ToBytes));
}

// Original: return Buffer.concat([fp12.c1, fp12.c0].map(fp6ToBytes))
function fp12ToBytes(fp12: Fp12Like): Uint8Array {
  return concatBytes(...[fp12.c1, fp12.c0].map(fp6ToBytes));
}

////// end of code from tlock-js fp serialization.

// ============================================================================
// From tlock-js src/crypto/ibe.ts
// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/crypto/ibe.ts
// ============================================================================

////// code from tlock-js src/crypto/ibe.ts gtToHash (line 117) — IBE H2
////// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/crypto/ibe.ts#L117
////// CHANGED: .update("IBE-H2") → .update(enc.encode("IBE-H2")) (Uint8Array types).

// Original:
//   return sha256.create().update("IBE-H2").update(fp12ToBytes(gt)).digest().slice(0, len)
function gtToHash(gt: Fp12Like, len: number): Uint8Array {
  return sha256
    .create()
    .update(enc.encode("IBE-H2"))
    .update(fp12ToBytes(gt))
    .digest()
    .slice(0, len);
}

////// end of code from tlock-js gtToHash.

////// code from tlock-js src/crypto/ibe.ts h4 (line 152) — IBE H4
////// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/crypto/ibe.ts#L152
////// CHANGED: .update("IBE-H4") → .update(enc.encode("IBE-H4")) (Uint8Array types).

// Original:
//   sha256.create().update("IBE-H4").update(sigma).digest()
function h4(sigma: Uint8Array, len: number): Uint8Array {
  const h4sigma = sha256
    .create()
    .update(enc.encode("IBE-H4"))
    .update(sigma)
    .digest();
  return h4sigma.slice(0, len);
}

////// end of code from tlock-js h4.

////// code from tlock-js src/crypto/ibe.ts h3 (line 131), create16BitUintBuffer (line 158)
////// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/crypto/ibe.ts#L131
////// CHANGED: .update("IBE-H3") → .update(enc.encode("IBE-H3")) (Uint8Array types).
////// CHANGED: Buffer.alloc(2) + writeUint16LE → Uint8Array + DataView (browser compat).

const BitsToMaskForBLS12381 = 1;

function h3(sigma: Uint8Array, msg: Uint8Array): bigint {
  const h3ret = sha256
    .create()
    .update(enc.encode("IBE-H3"))
    .update(sigma)
    .update(msg)
    .digest();

  // We will hash iteratively: H(i || H("IBE-H3" || sigma || msg)) until we get a
  // value that is suitable as a scalar.
  for (let i = 1; i < 65535; i++) {
    let data = h3ret;
    data = sha256.create()
      .update(create16BitUintBuffer(i))
      .update(data)
      .digest();
    // assuming Big Endianness
    data[0] = data[0] >> BitsToMaskForBLS12381;
    const n = bytesToNumberBE(data);
    if (n < bls12_381.fields.Fr.ORDER) {
      return n;
    }
  }

  throw new Error("invalid proof: rP check failed");
}

// Original: Buffer.alloc(2) + writeUint16LE
function create16BitUintBuffer(input: number): Uint8Array {
  if (input < 0) {
    throw Error("cannot write a negative value as uint!");
  }
  if (input > (2 ** 16)) {
    throw Error("input value too large to fit in a uint16!");
  }

  const buf = new Uint8Array(2);
  new DataView(buf.buffer).setUint16(0, input, true); // true = little-endian
  return buf;
}

////// end of code from tlock-js h3.

////// code from tlock-js src/crypto/ibe.ts decryptOnG2 (line 95)
////// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/crypto/ibe.ts#L95
//////
////// with parseCiphertext inlined from src/drand/timelock-decrypter.ts (line 78)
////// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/drand/timelock-decrypter.ts#L78
//////
////// parseCiphertext inlined; G2 point length computed the same way as the
////// original: base.toRawBytes(true).byteLength.

function decryptOnG2(signatureBytes: Uint8Array, ciphertextBody: Uint8Array): Uint8Array {
  // --- parseCiphertext (inlined from timelock-decrypter.ts) ---
  // Original: const pointLength = base.toRawBytes(true).byteLength
  const pointLength = G2.Point.BASE.toBytes(true).byteLength;
  const U = ciphertextBody.subarray(0, pointLength);
  const rest = ciphertextBody.subarray(pointLength);
  const halfLen = rest.length / 2;
  const V = rest.subarray(0, halfLen);
  const W = rest.subarray(halfLen);

  // --- decryptOnG2 (from ibe.ts) ---
  // 1. Compute sigma = V XOR H2(e(rP, private))
  const Qid = G1.Point.fromBytes(signatureBytes);
  const m = G2.Point.fromBytes(U);
  const gidt = bls12_381.pairing(Qid, m);
  const hgidt = gtToHash(gidt as unknown as Fp12Like, W.length);
  if (hgidt.length != V.length) {
    throw new Error("XorSigma is of invalid length");
  }
  const sigma = xor(hgidt, V);

  // 2. Compute msg = W XOR H4(sigma)
  const hsigma = h4(sigma, W.length);
  const msg = xor(hsigma, W);

  // 3. Check U = rP
  const r = h3(sigma, msg);
  const rP = G2.Point.BASE.multiply(r);
  if (!rP.equals(m)) {
    throw new Error("invalid proof: rP check failed");
  }

  return msg;
}

////// end of code from tlock-js decryptOnG2.

// ============================================================================
// Drand beacon fetching and verification
//
// fetchBeacon: Custom endpoint-fallback HTTP fetch. Not from drand-client
// (which uses a ChainClient abstraction). URL pattern from the drand HTTP
// API spec: https://drand.love/docs/specification/
//
// verifyBeacon: Adapted from drand-client lib/beacon-verification.ts
// https://github.com/drand/drand-client/blob/ef8c9260294f8699b5e8c27a6b764f8f0d768bea/lib/beacon-verification.ts
// ============================================================================

interface DrandBeacon {
  round: number;
  randomness: string;
  signature: string;
}

interface DrandConfig {
  chainHash: string;
  genesis: number;
  period: number;
  publicKey: string;
  endpoints: string[];
  schemeID: string;
}

// Custom — not from drand-client source.
async function fetchBeacon(config: DrandConfig, round: number): Promise<DrandBeacon> {
  let lastError: Error | undefined;
  for (const endpoint of config.endpoints) {
    try {
      const url = `${endpoint}/${config.chainHash}/public/${round}`;
      const resp = await fetch(url);
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const beacon: DrandBeacon = await resp.json();
      verifyBeacon(config, beacon);
      return beacon;
    } catch (e) {
      lastError = e instanceof Error ? e : new Error(String(e));
    }
  }
  throw new Error(`could not fetch beacon: ${lastError?.message ?? 'all endpoints failed'}`);
}

////// adapted from drand-client lib/beacon-verification.ts
////// https://github.com/drand/drand-client/blob/ef8c9260294f8699b5e8c27a6b764f8f0d768bea/lib/beacon-verification.ts
//////
////// randomnessIsValid (line 90): sha256(signature) == randomness
////// roundBuffer (line 86): round as 8-byte big-endian
////// unchainedBeaconMessage: sha256(roundBuffer(round))
//////
////// CHANGED: BLS verification uses bls12_381.shortSignatures.verify() from
////// @noble/curves instead of drand-client's manual pairing in verifySigOnG1().
////// The library function is audited; the manual implementation existed because
////// @noble/curves lacked G1 signature verification when drand-client was written.
////// CHANGED: Buffer → DataView.setBigUint64 + hexToBytes + bytesEqual.
////// CHANGED: throws on failure instead of returning boolean.
////// Only handles bls-unchained-g1-rfc9380 (quicknet), not all 5 schemes.

// Exported for e2e/beacon-verify.spec.ts.
export function verifyBeacon(config: Pick<DrandConfig, 'publicKey'>, beacon: DrandBeacon): void {
  const sigBytes = hexToBytes(beacon.signature);
  const randBytes = hexToBytes(beacon.randomness);

  // randomnessIsValid (line 90): randomness must equal sha256(signature)
  const sigHash = sha256(sigBytes);
  if (!bytesEqual(sigHash, randBytes)) {
    throw new Error('beacon: randomness does not match signature hash');
  }

  // roundBuffer (line 86) + unchainedBeaconMessage: message = sha256(round as 8-byte BE)
  // Original: Buffer.alloc(8) + writeBigUInt64BE
  const roundBuf = new Uint8Array(8);
  new DataView(roundBuf.buffer).setBigUint64(0, BigInt(beacon.round), false);
  const message = sha256(roundBuf);

  // BLS signature verification.
  // Original drand-client verifySigOnG1 (line 51) implemented manual pairing:
  //   e(H(m), P) * e(S, G)^-1 = 1
  // We use the @noble/curves library function instead; it takes the message
  // already hashed to G1 with drand's DST.
  const bls = bls12_381.shortSignatures;
  const valid = bls.verify(sigBytes, bls.hash(message, BLS_VERIFY_DST), hexToBytes(config.publicKey));
  if (!valid) {
    throw new Error('beacon: BLS signature verification failed');
  }
}

////// end of adapted drand-client beacon verification.

// ============================================================================
// Tlock Identity — adapted from tlock-js src/drand/timelock-decrypter.ts
// https://github.com/drand/tlock-js/blob/17d817ee259e79381111dd75009b0f022c39ace3/src/drand/timelock-decrypter.ts
//
// Implements age-encryption's Identity interface. The original returns a
// function matching tlock-js's own age implementation; this class implements
// the age-encryption npm package's Identity interface instead.
//
// Stanza format: age-encryption puts the recipient type in args[0], so
// args = ['tlock', round, chainHash] (3 elements). Original tlock-js has
// a separate `type` field, so args = [round, chainHash] (2 elements).
// ============================================================================

class TlockIdentity implements Identity {
  constructor(private config: DrandConfig) {}

  async unwrapFileKey(stanzas: Stanza[]): Promise<Uint8Array | null> {
    for (const s of stanzas) {
      if (s.args[0] !== 'tlock') continue;
      if (s.args.length !== 3) throw new Error('invalid tlock stanza');

      const round = parseInt(s.args[1], 10);
      const chainHash = s.args[2];
      if (chainHash !== this.config.chainHash) {
        throw new Error(`chain hash mismatch: ${chainHash} vs ${this.config.chainHash}`);
      }

      const beacon = await fetchBeacon(this.config, round);
      return decryptOnG2(hexToBytes(beacon.signature), s.body);
    }
    return null;
  }
}

// ============================================================================
// Main entry point
// ============================================================================

// Decrypt a tlock-encrypted age payload. Fetches the drand beacon over HTTP.
export async function timelockDecrypt(
  ciphertext: Uint8Array,
  config: DrandConfig,
): Promise<Uint8Array> {
  const d = new Decrypter();
  d.addIdentity(new TlockIdentity(config));
  return d.decrypt(ciphertext);
}

// ============================================================================
// Utility functions
// hexToBytes, bytesEqual: standard conversions (not from tlock-js source).
// Replace tlock-js's use of Node.js Buffer for browser compatibility.
// ============================================================================

function hexToBytes(hex: string): Uint8Array {
  const out = new Uint8Array(hex.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  return out;
}

// Constant-time comparison (OR-accumulator).
function bytesEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) diff |= a[i] ^ b[i];
  return diff === 0;
}
