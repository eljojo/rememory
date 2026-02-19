// ReMemory Type Definitions
// Shared types for recovery (native JS) and creation (WASM)

// ============================================
// Bundle Types (used by maker.html WASM)
// ============================================

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
  manifestB64?: string; // Base64-encoded MANIFEST.age (when small enough to embed)
  projectName?: string; // Project name for display in guide
  readmeText?: string;  // Pre-generated README for the in-page guide
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

    // Localized README filenames (embedded in recover.html)
    README_NAMES?: string[];

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
