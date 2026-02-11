// ReMemory Type Definitions
// Types for the WASM interface and shared data structures

// ============================================
// Share Types
// ============================================

export interface ParsedShare {
  version: number;
  index: number;
  threshold: number;
  total: number;
  holder?: string;
  dataB64: string;
  compact?: string;   // Compact-encoded string (e.g. RM1:2:5:3:BASE64:CHECK)
  isHolder?: boolean;  // True if this is the current user's share
}

export interface ShareInput {
  version: number;
  index: number;
  dataB64: string;
}

export interface ShareParseResult {
  error?: string;
  share?: ParsedShare;
}

export interface CombineResult {
  error?: string;
  passphrase?: string;
}

// ============================================
// Bundle Types
// ============================================

export interface BundleExtractResult {
  error?: string;
  share?: ParsedShare;
  manifest?: Uint8Array;
}

export interface BundleFile {
  name: string;
  data: Uint8Array;
}

export interface BundleConfig {
  projectName: string;
  threshold: number;
  friends: FriendInput[];
  files: BundleFile[];
  version: string;
  githubURL: string;
}

export interface GeneratedBundle {
  friendName: string;
  fileName: string;
  data: Uint8Array;
}

export interface BundleCreateResult {
  error?: string;
  bundles?: GeneratedBundle[];
}

// ============================================
// Decryption Types
// ============================================

export interface DecryptResult {
  error?: string;
  data?: Uint8Array;
}

export interface ExtractedFile {
  name: string;
  data: Uint8Array;
}

export interface ExtractResult {
  error?: string;
  files?: ExtractedFile[];
}

// ============================================
// Project Types
// ============================================

export interface FriendInfo {
  name: string;
  contact?: string;
  shareIndex: number;  // 1-based share index for this friend
}

export interface FriendInput {
  name: string;
  contact?: string;
  language?: string;
}

export interface ProjectConfig {
  name?: string;
  threshold?: number;
  friends?: FriendInfo[];
}

export interface ProjectParseResult {
  error?: string;
  project?: ProjectConfig;
}

// ============================================
// Personalization Types (for recover.html)
// ============================================

export interface PersonalizationData {
  holder: string;
  holderShare: string;
  otherFriends: FriendInfo[];
  threshold: number;
  total: number;
  language?: string;
}

// ============================================
// UI State Types
// ============================================

export interface RecoveryState {
  shares: ParsedShare[];
  manifest: Uint8Array | null;
  threshold: number;
  total: number;
  wasmReady: boolean;
  recovering: boolean;
  recoveryComplete: boolean;
  decryptedArchive?: Uint8Array;
}

export interface CreationState {
  projectName: string;
  friends: FriendInput[];
  threshold: number;
  files: BundleFile[];
  bundles: GeneratedBundle[];
  wasmReady: boolean;
  generating: boolean;
  generationComplete: boolean;
}

// ============================================
// Toast Types
// ============================================

export type ToastType = 'error' | 'warning' | 'success' | 'info';

export interface ToastAction {
  id: string;
  label: string;
  primary?: boolean;
  onClick?: () => void;
}

export interface ToastOptions {
  type?: ToastType;
  title?: string;
  message: string;
  guidance?: string;
  actions?: ToastAction[];
  duration?: number;
}

// ============================================
// WASM Global Interface
// ============================================

declare global {
  interface Window {
    // WASM ready flag
    rememoryReady: boolean;
    rememoryAppReady?: boolean;

    // Recovery functions (recover.wasm)
    rememoryParseShare(content: string): ShareParseResult;
    rememoryCombineShares(shares: ShareInput[]): CombineResult;
    rememoryDecryptManifest(manifest: Uint8Array, passphrase: string): DecryptResult;
    rememoryExtractTarGz(data: Uint8Array): ExtractResult;
    rememoryExtractBundle(zipData: Uint8Array): BundleExtractResult;
    rememoryParseCompactShare(compact: string): ShareParseResult;
    rememoryDecodeWords(words: string[]): { data: Uint8Array; index: number; checksum: string; error?: string };

    // Creation functions (create.wasm)
    rememoryCreateBundles(config: BundleConfig): BundleCreateResult;
    rememoryParseProjectYAML(yaml: string): ProjectParseResult;

    // Shared utilities (exposed by shared.ts)
    rememoryUtils: {
      escapeHtml: (str: string | null | undefined) => string;
      formatSize: (bytes: number) => string;
      toast: ToastManager;
      showInlineError: (target: HTMLElement, message: string, guidance?: string) => void;
      clearInlineError: (target: HTMLElement) => void;
    };

    // UI update callback
    rememoryUpdateUI?: () => void;

    // Personalization data (embedded in recover.html)
    PERSONALIZATION?: PersonalizationData | null;

    // Embedded constants
    WASM_BINARY?: string;
    VERSION?: string;
    GITHUB_URL?: string;

    // Go WASM runtime
    Go: new () => GoInstance;
  }

  interface GoInstance {
    importObject: WebAssembly.Imports;
    run(instance: WebAssembly.Instance): Promise<void>;
  }
}

// ============================================
// Toast Manager Interface
// ============================================

export interface ToastManager {
  container: HTMLElement | null;
  backdrop: HTMLElement | null;
  errorCount: number;
  init(): void;
  showBackdrop(): void;
  hideBackdrop(): void;
  dismissAllErrors(): void;
  show(options: ToastOptions): HTMLElement;
  dismiss(toastEl: HTMLElement): void;
  error(title: string, message: string, guidance?: string, actions?: ToastAction[]): HTMLElement;
  warning(title: string, message: string, guidance?: string): HTMLElement;
  success(title: string, message: string): HTMLElement;
  info(title: string, message: string, guidance?: string): HTMLElement;
}

// ============================================
// Translation Function Type
// ============================================

export type TranslationFunction = (key: string, ...args: (string | number)[]) => string;

// ============================================
// BarcodeDetector API (not in standard TS lib)
// ============================================

export interface DetectedBarcode {
  rawValue: string;
  format: string;
  boundingBox: DOMRectReadOnly;
  cornerPoints: Array<{ x: number; y: number }>;
}

declare global {
  class BarcodeDetector {
    constructor(options?: { formats: string[] });
    detect(source: HTMLVideoElement | HTMLCanvasElement | ImageBitmap | ImageData): Promise<DetectedBarcode[]>;
    static getSupportedFormats(): Promise<string[]>;
  }
}
