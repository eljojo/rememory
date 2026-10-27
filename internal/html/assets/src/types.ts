// ReMemory Type Definitions
// Shared types for recovery (native JS) and creation (WASM)

// ============================================
// Bundle Types (used by maker.html WASM)
// ============================================

export interface BundleFile {
  name: string;
  data: Uint8Array;
}

export interface GeneratedBundle {
  friendName: string;
  fileName: string;
  data: Uint8Array;
}

export interface BundleCreateResult {
  error?: string;
  bundles?: GeneratedBundle[];
  manifest?: Uint8Array;
}

export interface ArchiveCreateResult {
  error?: string;
  data?: Uint8Array;
}

export interface BundleFromArchiveConfig {
  projectName: string;
  threshold: number;
  friends: FriendInput[];
  archiveData: Uint8Array;
  version: string;
  anonymous?: boolean;
  defaultLanguage?: string;
  tlockRound?: number;
  tlockUnlock?: string;
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
  otherFriends: FriendInfo[] | null; // older generators emitted null for anonymous bundles
  threshold: number;
  total: number;
  language?: string;
  manifestB64?: string; // Base64-encoded MANIFEST.age (when small enough to embed)
  tlockEnabled?: boolean; // Signals tlock-js is included for time-lock decryption
}

// ============================================
// Tlock Types (for time-lock encryption)
// ============================================

export interface TlockContainerMeta {
  v: number;
  method: string;
  round: number;
  unlock: string;
  chain: string;
}

// ============================================
// UI State Types
// ============================================

// Import ParsedShare from crypto module for recovery state
import type { ParsedShare } from './crypto/share';

export interface RecoveryState {
  shares: (ParsedShare & { isHolder?: boolean })[];
  manifest: Uint8Array | null;
  threshold: number;
  total: number;
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
  tlockEnabled: boolean;
  tlockValue: number;
  tlockUnit: string;
}

// ============================================
// Selfhosted Config (injected by server at render time)
// ============================================

export interface SelfhostedConfig {
  maxManifestSize: number;
  hasManifest: boolean;
  manifestURL?: string;  // URL to fetch manifest from (set by server or static pages)
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
// WASM Global Interface (for maker.html)
// ============================================

declare global {
  interface Window {
    // WASM ready flag (used by maker.html)
    rememoryReady: boolean;
    rememoryAppReady?: boolean;

    // Creation functions (create.wasm, used by maker.html)
    rememoryCreateArchive(files: BundleFile[]): ArchiveCreateResult;
    rememoryCreateBundlesFromArchive(config: BundleFromArchiveConfig): BundleCreateResult;
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
    BUILD_DATE?: string;
    MAX_TOTAL_FILE_SIZE?: number;

    // Localized README filenames (embedded in recover.html)
    README_NAMES?: string[];

    // Selfhosted mode (only present in selfhosted builds, eliminated in static builds)
    rememoryLoadManifest?: (data: Uint8Array, name?: string) => void;
    SELFHOSTED_CONFIG?: SelfhostedConfig | null;

    // Go WASM runtime (used by maker.html)
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
