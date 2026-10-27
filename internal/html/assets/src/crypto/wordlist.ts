// BIP39 wordlist handling - imports existing wordlist files at build time
// esbuild bundles these as strings using the text loader
// Supports every language in internal/core/wordlists.go, with auto-detection
// and normalization

// Import wordlists from the existing Go codebase
import englishWords from '../../../../core/wordlists/english.txt';
import spanishWords from '../../../../core/wordlists/spanish.txt';
import frenchWords from '../../../../core/wordlists/french.txt';
import germanWords from '../../../../core/wordlists/german.txt';
import slovenianWords from '../../../../core/wordlists/slovenian.txt';
import portugueseWords from '../../../../core/wordlists/portuguese.txt';
import chineseTraditionalWords from '../../../../core/wordlists/chinese_traditional.txt';
import catalanWords from '../../../../core/wordlists/catalan.txt';
import turkishWords from '../../../../core/wordlists/turkish.txt';
import turkishV0019Words from '../../../../core/wordlists/legacy/turkish-v0.0.19.txt';

import { sha256 } from './hash';

// Supported languages, in the same order as AllLangs in wordlists.go
export type Lang = 'en' | 'es' | 'fr' | 'de' | 'sl' | 'pt' | 'zh-TW' | 'ca' | 'tr';

export const ALL_LANGS: Lang[] = ['en', 'es', 'fr', 'de', 'sl', 'pt', 'zh-TW', 'ca', 'tr'];

// Lists a release printed in READMEs and that later changed in place. They
// are only used to decode: words printed back then still mean their old
// indices. v0.0.19 shipped the first Turkish list; v0.0.20 corrected it.
type LegacyList = 'tr-v0.0.19';
type WordList = Lang | LegacyList;
const DECODE_LISTS: WordList[] = [...ALL_LANGS, 'tr-v0.0.19'];

// Raw wordlist text by list
const WORDLIST_TEXT: Record<WordList, string> = {
  en: englishWords,
  es: spanishWords,
  fr: frenchWords,
  de: germanWords,
  sl: slovenianWords,
  pt: portugueseWords,
  'zh-TW': chineseTraditionalWords,
  ca: catalanWords,
  tr: turkishWords,
  'tr-v0.0.19': turkishV0019Words,
};

// Parse wordlist text into array
function parseWordlist(text: string): string[] {
  return text
    .trim()
    .split('\n')
    .map((w) => w.trim());
}

// --- Normalization ---

/**
 * Normalize a word for tolerant matching:
 * lowercase, trim, NFD decompose, and strip combining marks.
 * Examples: "Ábaco" → "abaco", "GÜNTHER" → "gunther", "čudež" → "cudez"
 */
function normalizeWord(word: string): string {
  word = word.toLowerCase().trim();
  // NFD decomposition separates base characters from combining marks
  const nfd = word.normalize('NFD');
  // Strip combining marks (Unicode category Mn)
  // Combining marks are in range U+0300 to U+036F for most Latin scripts
  return nfd.replace(/[\u0300-\u036f]/g, '');
}

/**
 * Collapse German umlaut digraphs to base vowels.
 * "guenther" → "gunther" (matches NFD-stripped "günther")
 */
function collapseGermanDigraphs(word: string): string {
  return word.replace(/ae/g, 'a').replace(/oe/g, 'o').replace(/ue/g, 'u');
}

// --- Per-language word index ---

interface LangWordIndex {
  words: string[]; // Original words
  exact: Map<string, number>; // lowercase → index
  stripped: Map<string, number>; // NFD-stripped → index (only if different from exact)
  digraph?: Map<string, number>; // German digraph collapsed → index
}

// Lazy-loaded indices
const langIndices = new Map<WordList, LangWordIndex>();

function getLangIndex(lang: WordList): LangWordIndex {
  let idx = langIndices.get(lang);
  if (idx) return idx;

  const words = parseWordlist(WORDLIST_TEXT[lang]);
  if (words.length !== 2048) {
    throw new Error(`Wordlist ${lang} has ${words.length} words, expected 2048`);
  }

  idx = {
    words,
    exact: new Map(),
    stripped: new Map(),
  };

  if (lang === 'de') {
    idx.digraph = new Map();
  }

  for (let i = 0; i < words.length; i++) {
    const w = words[i];
    const lower = w.toLowerCase();
    idx.exact.set(lower, i);

    const normalized = normalizeWord(w);
    // Only store stripped form if it differs from exact
    if (normalized !== lower) {
      idx.stripped.set(normalized, i);
    }

    // German digraph fallback
    if (lang === 'de' && idx.digraph) {
      const collapsed = collapseGermanDigraphs(normalized);
      if (collapsed !== normalized) {
        idx.digraph.set(collapsed, i);
      }
    }
  }

  langIndices.set(lang, idx);
  return idx;
}

// --- Public API ---

/**
 * Look up a word's BIP39 index in a specific language.
 * Tries exact match first, then NFD-stripped, then German digraph expansion.
 * Returns -1 if not found.
 */
export function lookupWordInLang(lang: WordList, word: string): number {
  const idx = getLangIndex(lang);

  const lower = word.toLowerCase().trim();

  // 1. Exact lowercase match
  const exactIdx = idx.exact.get(lower);
  if (exactIdx !== undefined) return exactIdx;

  // 2. NFD-stripped match (ábaco → abaco)
  const normalized = normalizeWord(word);
  const strippedIdx = idx.stripped.get(normalized);
  if (strippedIdx !== undefined) return strippedIdx;

  // 3. German digraph collapse (guenther → gunther, matching günther)
  if (lang === 'de' && idx.digraph) {
    const collapsed = collapseGermanDigraphs(normalized);
    const digraphIdx = idx.digraph.get(collapsed);
    if (digraphIdx !== undefined) return digraphIdx;
  }

  return -1;
}

/**
 * Detect which language a set of words belongs to.
 * Returns the language where the most words match.
 * Requires >50% match. Returns null if no language matches.
 */
export function detectLanguage(words: string[]): Lang | null {
  if (words.length === 0) return null;

  let bestLang: Lang | null = null;
  let bestCount = 0;

  for (const lang of ALL_LANGS) {
    let count = 0;
    for (const w of words) {
      if (lookupWordInLang(lang, w) >= 0) {
        count++;
      }
    }
    if (count > bestCount) {
      bestCount = count;
      bestLang = lang;
    }
  }

  // Require >50% match
  if (bestCount <= words.length / 2) {
    return null;
  }

  return bestLang;
}

/**
 * Look up a word's BIP39 index (0-2047).
 * Tries English first, then auto-detects language if not found.
 * Returns -1 if not found in any language.
 */
export function lookupWord(word: string): number {
  // Try English first (most common)
  const enIdx = lookupWordInLang('en', word);
  if (enIdx >= 0) return enIdx;

  // Try other languages
  for (const lang of ALL_LANGS) {
    if (lang === 'en') continue;
    const idx = lookupWordInLang(lang, word);
    if (idx >= 0) return idx;
  }

  return -1;
}

/**
 * Get word at BIP39 index (English).
 */
export function getWord(index: number): string {
  const idx = getLangIndex('en');
  if (index < 0 || index >= idx.words.length) {
    throw new Error(`Invalid word index: ${index}`);
  }
  return idx.words[index];
}

/**
 * Get word at BIP39 index for a specific language.
 */
export function getWordInLang(lang: Lang, index: number): string {
  const idx = getLangIndex(lang);
  if (index < 0 || index >= idx.words.length) {
    throw new Error(`Invalid word index: ${index}`);
  }
  return idx.words[index];
}

/**
 * Extract an 11-bit value starting at the given bit offset.
 * Out-of-range bits are treated as zero (for padding the final chunk).
 */
function extract11Bits(data: Uint8Array, bitOffset: number): number {
  let val = 0;
  for (let b = 0; b < 11; b++) {
    const byteIdx = Math.floor((bitOffset + b) / 8);
    const bitIdx = 7 - ((bitOffset + b) % 8);
    if (byteIdx < data.length) {
      val = (val << 1) | ((data[byteIdx] >> bitIdx) & 1);
    } else {
      val <<= 1; // pad with zero
    }
  }
  return val;
}

/**
 * Set an 11-bit value at the given bit offset in data.
 * Precondition: target bits must be zero-initialized.
 */
function set11Bits(data: Uint8Array, bitOffset: number, val: number): void {
  for (let b = 0; b < 11; b++) {
    const byteIdx = Math.floor((bitOffset + b) / 8);
    const bitIdx = 7 - ((bitOffset + b) % 8);
    if (byteIdx < data.length) {
      if (((val >> (10 - b)) & 1) === 1) {
        data[byteIdx] |= 1 << bitIdx;
      }
    }
  }
}

/**
 * Encode bytes to BIP39 words (11 bits per word).
 * 33 bytes (264 bits) produces exactly 24 words.
 */
export function encodeWords(data: Uint8Array): string[] {
  const totalBits = data.length * 8;
  const numWords = Math.ceil(totalBits / 11);
  const words: string[] = [];

  for (let i = 0; i < numWords; i++) {
    const idx = extract11Bits(data, i * 11);
    words.push(getWord(idx));
  }

  return words;
}

/**
 * Decode BIP39 words back to bytes.
 * Auto-detects language from the words.
 */
export function decodeWords(words: string[]): Uint8Array {
  if (words.length === 0) {
    throw new Error('no words provided');
  }

  // Detect language
  const lang = detectLanguage(words);
  if (!lang) {
    // Fall back to trying each word individually
    const indices: number[] = [];
    for (let i = 0; i < words.length; i++) {
      const idx = lookupWord(words[i]);
      if (idx < 0) {
        throw new Error(`word ${i + 1} "${words[i]}" not recognized`);
      }
      indices.push(idx);
    }
    return indicesToBytes(indices, words.length);
  }

  // Look up all words in detected language
  const indices: number[] = [];
  for (let i = 0; i < words.length; i++) {
    const idx = lookupWordInLang(lang, words[i]);
    if (idx < 0) {
      throw new Error(`word ${i + 1} "${words[i]}" not recognized`);
    }
    indices.push(idx);
  }

  return indicesToBytes(indices, words.length);
}

/**
 * Convert word indices to bytes.
 */
function indicesToBytes(indices: number[], numWords: number): Uint8Array {
  const totalBits = numWords * 11;
  const numBytes = Math.floor(totalBits / 8);
  const result = new Uint8Array(numBytes);

  for (let i = 0; i < indices.length; i++) {
    set11Bits(result, i * 11, indices[i]);
  }

  return result;
}

/** One way of reading 25 recovery words: share data and share index. */
export interface ShareWordsReading {
  data: Uint8Array;
  index: number;
}

// Word 25 layout (11 bits total):
// - Upper 4 bits (bits 10-7): share index (1-15, 0 = unknown/16+)
// - Lower 7 bits (bits 6-0): checksum (lower 7 bits of SHA-256(data)[0])

const WORD25_INDEX_BITS = 4;
const WORD25_CHECK_BITS = 7;
const WORD25_MAX_INDEX = (1 << WORD25_INDEX_BITS) - 1; // 15
const WORD25_CHECK_MASK = (1 << WORD25_CHECK_BITS) - 1; // 0x7F

/**
 * Compute the 7-bit checksum for the 25th word.
 */
export async function word25Checksum(data: Uint8Array): Promise<number> {
  const hash = await sha256(data);
  return hash[0] & WORD25_CHECK_MASK;
}

/**
 * Pack share index and data checksum into an 11-bit BIP39 word index.
 */
export async function word25Encode(
  shareIndex: number,
  data: Uint8Array
): Promise<number> {
  let idx = shareIndex;
  if (idx > WORD25_MAX_INDEX) {
    idx = 0; // sentinel: index not representable in 4 bits
  }
  const check = await word25Checksum(data);
  return (idx << WORD25_CHECK_BITS) | check;
}

/**
 * Unpack the 25th word's 11-bit value into index and checksum.
 */
export function word25Decode(val: number): { index: number; checksum: number } {
  return {
    index: val >> WORD25_CHECK_BITS,
    checksum: val & WORD25_CHECK_MASK,
  };
}

/**
 * Decode 25 BIP39 words into share data and index.
 * The first 24 words are decoded to bytes; the 25th word carries index + checksum.
 * Returns index=0 if the share index was > 15 (the sentinel value).
 * Throws if the checksum doesn't match.
 * Auto-detects language from the words.
 *
 * Words can fit more than one list with a valid checksum (the v0.0.19
 * Turkish list overlaps today's). The 7-bit checksum can't tell which is
 * right, so the most likely reading is returned and the others are kept in
 * alternatives; unlocking decides (see unlockWithCandidates).
 */
export async function decodeShareWords(words: string[]): Promise<ShareWordsReading & {
  alternatives: ShareWordsReading[];
}> {
  if (words.length !== 25) {
    throw new Error(`expected 25 words, got ${words.length}`);
  }

  // More than one list can hold all 25 words: lists share some words, and
  // the v0.0.19 Turkish list overlaps today's. Keep every reading whose
  // checksum agrees, the list matching the most words first.
  const matches = (list: WordList) => words.filter(w => lookupWordInLang(list, w) >= 0).length;
  const ranked = DECODE_LISTS
    .map(list => ({ list, count: matches(list) }))
    .sort((a, b) => b.count - a.count);

  const readings: ShareWordsReading[] = [];
  let checksumFailed = false;
  for (const { list, count } of ranked) {
    if (count < 25) break;
    const indices = words.map(w => lookupWordInLang(list, w));
    const data = indicesToBytes(indices.slice(0, 24), 24);
    const { index, checksum: expectedCheck } = word25Decode(indices[24]);
    if (await word25Checksum(data) !== expectedCheck) {
      checksumFailed = true;
      continue;
    }
    const seen = readings.some(r => r.data.length === data.length && r.data.every((b, i) => b === data[i]));
    if (!seen) readings.push({ data, index });
  }

  if (readings.length > 0) {
    return { ...readings[0], alternatives: readings.slice(1) };
  }
  if (checksumFailed) {
    throw new Error('word checksum failed — check word order and spelling');
  }
  // Name the first word the most likely list doesn't know.
  const best = ranked[0].list;
  const missing = words.findIndex(w => lookupWordInLang(best, w) < 0);
  throw new Error(`word ${missing + 1} "${words[missing]}" not recognized`);
}
