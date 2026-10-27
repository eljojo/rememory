// ReMemory Native Crypto Module

export { sha256, hashBytes, verifyHash } from './hash';
export { combine, recoverPassphrase, base64ToBytes, bytesToBase64 } from './shamir';
export { decrypt } from './age';
export { unlockWithCandidates } from './unlock';
export { extractTarGz, extractArchive, isTlockContainer, openTlockContainer, type ExtractedFile } from './archive';
export { parseShare, parseCompactShare, encodeCompact, type ParsedShare } from './share';
export {
  decodeShareWords,
  decodeWords,
  lookupWord,
  lookupWordInLang,
  encodeWords,
  getWord,
  getWordInLang,
  detectLanguage,
  ALL_LANGS,
  type Lang,
  type ShareWordsReading,
} from './wordlist';
export { extractBundle, extractPersonalizationFromHTML, type BundleContents } from './zip';
