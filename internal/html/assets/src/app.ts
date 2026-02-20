// ReMemory Recovery Tool - Browser-based recovery using native JavaScript crypto

// BarcodeDetector polyfill - provides QR scanning in browsers without native support
import { registerPolyfill } from './barcode-detector';
registerPolyfill();

import type {
  RecoveryState,
  PersonalizationData,
  FriendInfo,
  ToastAction,
  TranslationFunction
} from './types';

// Native crypto imports
import {
  parseShare,
  parseCompactShare,
  encodeCompact,
  combine,
  recoverPassphrase,
  base64ToBytes,
  bytesToBase64,
  decrypt,
  extractTarGz,
  extractBundle,
  extractPersonalizationFromHTML,
  decodeShareWords,
  type ParsedShare,
} from './crypto';

// Translation function (defined in HTML)
declare const t: TranslationFunction;

// Extended share type with UI-specific fields
type UIShare = ParsedShare & { isHolder?: boolean };

(function() {
  'use strict';

  // Import shared utilities
  const { escapeHtml, formatSize, toast, showInlineError, clearInlineError } = window.rememoryUtils;

  // Wrap email addresses in mailto: links. Input must already be HTML-escaped.
  const EMAIL_RE = /[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}/g;
  function linkifyEmail(escaped: string): string {
    return escaped.replace(EMAIL_RE, (match) => {
      const clean = match.replace(/\.+$/, '');
      return `<a href="mailto:${clean}">${clean}</a>`;
    });
  }

  // State
  const state: RecoveryState = {
    shares: [],
    manifest: null,
    threshold: 0,
    total: 0,
    recovering: false,
    recoveryComplete: false
  };

  // DOM elements interface
  interface Elements {
    shareDropZone: HTMLElement | null;
    shareFileInput: HTMLInputElement | null;
    sharesList: HTMLElement | null;
    thresholdInfo: HTMLElement | null;
    manifestDropZone: HTMLElement | null;
    manifestFileInput: HTMLInputElement | null;
    manifestStatus: HTMLElement | null;
    recoverBtn: HTMLButtonElement | null;
    recoverSection: HTMLElement | null;
    progressBar: HTMLElement | null;
    statusMessage: HTMLElement | null;
    filesList: HTMLElement | null;
    downloadActions: HTMLElement | null;
    downloadAllBtn: HTMLButtonElement | null;
    pasteToggleBtn: HTMLButtonElement | null;
    pasteArea: HTMLElement | null;
    pasteInput: HTMLTextAreaElement | null;
    pasteSubmitBtn: HTMLButtonElement | null;
    contactListSection: HTMLElement | null;
    contactList: HTMLElement | null;
    step1Card: HTMLElement | null;
    step2Card: HTMLElement | null;
    scanQrBtn: HTMLButtonElement | null;
    qrScannerModal: HTMLElement | null;
    qrVideo: HTMLVideoElement | null;
    qrScannerClose: HTMLButtonElement | null;
  }

  // DOM elements
  const elements: Elements = {
    shareDropZone: document.getElementById('share-drop-zone'),
    shareFileInput: document.getElementById('share-file-input') as HTMLInputElement | null,
    sharesList: document.getElementById('shares-list'),
    thresholdInfo: document.getElementById('threshold-info'),
    manifestDropZone: document.getElementById('manifest-drop-zone'),
    manifestFileInput: document.getElementById('manifest-file-input') as HTMLInputElement | null,
    manifestStatus: document.getElementById('manifest-status'),
    recoverBtn: document.getElementById('recover-btn') as HTMLButtonElement | null,
    recoverSection: document.getElementById('recover-section'),
    progressBar: document.getElementById('progress-bar'),
    statusMessage: document.getElementById('status-message'),
    filesList: document.getElementById('files-list'),
    downloadActions: document.getElementById('download-actions'),
    downloadAllBtn: document.getElementById('download-all-btn') as HTMLButtonElement | null,
    pasteToggleBtn: document.getElementById('paste-toggle-btn') as HTMLButtonElement | null,
    pasteArea: document.getElementById('paste-area'),
    pasteInput: document.getElementById('paste-input') as HTMLTextAreaElement | null,
    pasteSubmitBtn: document.getElementById('paste-submit-btn') as HTMLButtonElement | null,
    contactListSection: document.getElementById('contact-list-section'),
    contactList: document.getElementById('contact-list'),
    step1Card: null,
    step2Card: null,
    scanQrBtn: document.getElementById('scan-qr-btn') as HTMLButtonElement | null,
    qrScannerModal: document.getElementById('qr-scanner-modal'),
    qrVideo: document.getElementById('qr-video') as HTMLVideoElement | null,
    qrScannerClose: document.getElementById('qr-scanner-close') as HTMLButtonElement | null,
  };

  // Personalization data (embedded in HTML)
  const personalization: PersonalizationData | null =
    (typeof window.PERSONALIZATION !== 'undefined') ? window.PERSONALIZATION : null;

  // Share regex to extract from README.txt content
  const shareRegex = /-----BEGIN REMEMORY SHARE-----([\s\S]*?)-----END REMEMORY SHARE-----/;

  // Compact share format regex: RM{version}:{index}:{total}:{threshold}:{base64url}:{check}
  const compactShareRegex = /^RM\d+:\d+:\d+:\d+:[A-Za-z0-9_-]+:[0-9a-f]{4}$/;

  // ============================================
  // Error Handlers
  // ============================================

  function showError(msg: string, options: {
    title?: string;
    guidance?: string;
    actions?: ToastAction[];
    inline?: boolean;
    targetElement?: HTMLElement;
  } = {}): void {
    const { title, guidance, actions, inline, targetElement } = options;

    if (inline && targetElement) {
      showInlineError(targetElement, msg, guidance);
      return;
    }

    toast.error(title || t('error_title'), msg, guidance, actions);
  }

  const errorHandlers = {
    invalidShare(filename: string, _detail?: string): void {
      if (elements.shareDropZone) {
        showError(
          t('error_invalid_share_message', filename),
          {
            title: t('error_invalid_share_title'),
            guidance: t('error_invalid_share_guidance'),
            inline: true,
            targetElement: elements.shareDropZone
          }
        );
      }
    },

    noShareFound(filename: string): void {
      if (elements.shareDropZone) {
        showError(
          t('error_no_share_message', filename),
          {
            title: t('error_no_share_title'),
            guidance: t('error_no_share_guidance'),
            inline: true,
            targetElement: elements.shareDropZone
          }
        );
      }
    },

    duplicateShare(index: number): void {
      toast.warning(
        t('error_duplicate_title'),
        t('error_duplicate_message', index),
        t('error_duplicate_guidance')
      );
    },

    fileReadFailed(filename: string): void {
      showError(
        t('error_file_read_message', filename),
        {
          title: t('error_file_read_title'),
          guidance: t('error_file_read_guidance')
        }
      );
    },

    decryptionFailed(_err: unknown): void {
      toast.error(
        t('error_decrypt_title'),
        t('error_decrypt_message'),
        t('error_decrypt_guidance'),
        [
          {
            id: 'retry',
            label: t('action_try_different_shares'),
            primary: true,
            onClick: () => {
              const holderName = personalization?.holderName || '';
              const holderShare = state.shares.find((share) => {
                const uiShare = share as UIShare;
                return uiShare.isHolder === true || (holderName && uiShare.name === holderName);
              });

              state.shares = holderShare ? [holderShare] : [];
              state.recoveryComplete = false;
              updateSharesUI();
              elements.step1Card?.classList.remove('collapsed');
            }
          }
        ]
      );
    },

    extractionFailed(_err: unknown): void {
      toast.error(
        t('error_extract_title'),
        t('error_extract_message'),
        t('error_extract_guidance')
      );
    }
  };

  // ============================================
  // Initialization
  // ============================================

  async function init(): Promise<void> {
    // Get step card references
    const cards = document.querySelectorAll('.card');
    elements.step1Card = cards[0] as HTMLElement || null;
    elements.step2Card = cards[1] as HTMLElement || null;

    setupDropZones();
    setupGlobalDrop();
    setupButtons();
    setupPaste();
    setupScanner();

    // Render contact list immediately
    if (personalization?.otherFriends && personalization.otherFriends.length > 0) {
      renderContactList();
      elements.contactListSection?.classList.remove('hidden');
    }

    // Native crypto is always ready
    window.rememoryAppReady = true;

    // Load personalization data
    if (personalization) {
      await loadPersonalizationData();
    }

    // Check URL fragment for compact share (e.g. #share=RM1:2:5:3:BASE64:CHECK)
    await loadShareFromFragment();
  }

  // ============================================
  // Personalization
  // ============================================

  async function loadPersonalizationData(): Promise<void> {
    if (!personalization) return;

    // Load the holder's share automatically
    if (personalization.holderShare) {
      try {
        const share = await parseShare(personalization.holderShare) as UIShare;
        share.isHolder = true;
        share.compact = await encodeCompact(share);
        state.threshold = share.threshold;
        state.total = share.total;
        state.shares.push(share);
        updateSharesUI();
        updateContactList();
      } catch {
        // Silently ignore invalid holder share
      }
    }

    // Load embedded manifest if available (included when MANIFEST.age is small enough)
    if (personalization.manifestB64) {
      const binary = atob(personalization.manifestB64);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
      }
      state.manifest = bytes;
      showManifestLoaded('MANIFEST.age', state.manifest.length, 'embedded');
    }

    checkRecoverReady();
  }

  // ============================================
  // URL Fragment Share Loading
  // ============================================

  async function loadShareFromFragment(): Promise<void> {
    const hash = window.location.hash;
    if (!hash || !hash.startsWith('#share=')) return;

    const compact = decodeURIComponent(hash.slice('#share='.length));
    if (!compactShareRegex.test(compact)) return;

    try {
      const share = await parseCompactShare(compact);

      if (state.shares.some(s => s.index === share.index)) return;

      if (state.shares.length === 0 || (state.threshold === 0 && share.threshold > 0)) {
        state.threshold = share.threshold;
        state.total = share.total;
      }

      state.shares.push(share);
      updateSharesUI();
      checkRecoverReady();

      // Clear the fragment from the URL bar to avoid re-importing on reload
      if (window.history?.replaceState) {
        window.history.replaceState(null, '', window.location.pathname + window.location.search);
      }
    } catch {
      // Silently ignore invalid fragment shares
    }
  }

  function renderContactList(): void {
    if (!personalization?.otherFriends || !elements.contactList) return;

    elements.contactList.innerHTML = '';

    personalization.otherFriends.forEach((friend: FriendInfo) => {
      const item = document.createElement('div');
      item.className = 'contact-item';
      item.dataset.name = friend.name;
      if (friend.shareIndex) {
        item.dataset.shareIndex = String(friend.shareIndex);
      }

      const contactInfo = friend.contact ? linkifyEmail(escapeHtml(friend.contact)) : '';

      item.innerHTML = `
        <div class="checkbox"></div>
        <div class="details">
          <div class="name">${escapeHtml(friend.name)}</div>
          <div class="contact-info">${contactInfo || '—'}</div>
        </div>
      `;

      elements.contactList?.appendChild(item);
    });
  }

  function updateContactList(): void {
    if (!personalization?.otherFriends || !elements.contactList) return;

    const collectedNames = new Set(
      state.shares.map(s => s.holder?.toLowerCase()).filter(Boolean)
    );
    const collectedIndices = new Set(state.shares.map(s => s.index));

    elements.contactList.querySelectorAll('.contact-item').forEach(item => {
      const el = item as HTMLElement;
      const name = el.dataset.name?.toLowerCase();
      const shareIndex = el.dataset.shareIndex ? parseInt(el.dataset.shareIndex, 10) : 0;
      const isCollected = (name ? collectedNames.has(name) : false) || collectedIndices.has(shareIndex);
      el.classList.toggle('collected', isCollected);
      const checkbox = el.querySelector('.checkbox');
      if (checkbox) {
        checkbox.textContent = isCollected ? '✓' : '';
      }
    });
  }

  // ============================================
  // Drop Zone Setup
  // ============================================

  function setupDropZones(): void {
    // Both drop zones use the same unified handler - file type is auto-detected
    if (elements.shareDropZone && elements.shareFileInput) {
      setupDropZone(elements.shareDropZone, elements.shareFileInput, handleFilesUnified);
    }
    if (elements.manifestDropZone && elements.manifestFileInput) {
      setupDropZone(elements.manifestDropZone, elements.manifestFileInput, handleFilesUnified);
    }
  }

  async function handleFilesUnified(files: FileList | File[]): Promise<void> {
    for (const file of Array.from(files)) {
      await handleAnyFile(file);
    }
  }

  function setupDropZone(
    dropZone: HTMLElement,
    fileInput: HTMLInputElement,
    handler: (files: FileList | File[]) => Promise<void>
  ): void {
    dropZone.addEventListener('click', () => fileInput.click());

    dropZone.addEventListener('dragover', (e) => {
      e.preventDefault();
      dropZone.classList.add('dragover');
    });

    dropZone.addEventListener('dragleave', () => {
      dropZone.classList.remove('dragover');
    });

    dropZone.addEventListener('drop', (e) => {
      e.preventDefault();
      e.stopPropagation(); // Prevent global handler from also processing
      dropZone.classList.remove('dragover');
      document.body.classList.remove('drag-active');
      if (e.dataTransfer?.files) {
        handler(e.dataTransfer.files);
      }
    });

    fileInput.addEventListener('change', async (e) => {
      const target = e.target as HTMLInputElement;
      const files = Array.from(target.files || []);
      target.value = '';
      await handler(files);
    });
  }

  // ============================================
  // Global Drop (drop anywhere on page)
  // ============================================

  function setupGlobalDrop(): void {
    // Prevent default drag behavior on the whole document
    document.addEventListener('dragover', (e) => {
      e.preventDefault();
      document.body.classList.add('drag-active');
    });

    document.addEventListener('dragleave', (e) => {
      // Only remove class when leaving the document entirely
      if (e.relatedTarget === null) {
        document.body.classList.remove('drag-active');
      }
    });

    document.addEventListener('drop', async (e) => {
      e.preventDefault();
      document.body.classList.remove('drag-active');

      const files = e.dataTransfer?.files;
      if (!files || files.length === 0) return;

      // Process each file through the unified handler
      for (const file of Array.from(files)) {
        await handleAnyFile(file);
      }
    });
  }

  /**
   * Try to add a parsed share to state. Returns true if added.
   * When quiet is true, silently skips duplicates instead of showing an error.
   */
  async function tryAddShare(shareContent: string, { quiet = false } = {}): Promise<boolean> {
    const share = await parseShare(shareContent);
    share.compact = await encodeCompact(share);

    if (state.shares.some(s => s.index === share.index)) {
      if (!quiet) errorHandlers.duplicateShare(share.index);
      return false;
    }

    if (state.shares.length === 0 || (state.threshold === 0 && share.threshold > 0)) {
      state.threshold = share.threshold;
      state.total = share.total;
    }

    state.shares.push(share);
    updateSharesUI();
    return true;
  }

  function showUnrecognizedFileError(filename: string): void {
    showError(
      t('error_unrecognized_file_message', filename),
      {
        title: t('error_unrecognized_file_title'),
        guidance: t('error_unrecognized_file_guidance'),
      }
    );
  }

  /**
   * Unified file handler - detects file type and processes accordingly.
   */
  async function handleAnyFile(file: File): Promise<void> {
    const name = file.name.toLowerCase();

    try {
      // ZIP bundle - contains share + manifest
      if (name.endsWith('.zip') || file.type === 'application/zip') {
        await handleBundleZipUnified(file);
        return;
      }

      // HTML file - may contain share + manifest
      if (name.endsWith('.html') || name.endsWith('.htm')) {
        await handleHTMLUnified(file);
        return;
      }

      // .age file - manifest only
      if (name.endsWith('.age')) {
        const buffer = await readFileAsArrayBuffer(file);
        state.manifest = new Uint8Array(buffer);
        showManifestLoaded(file.name, state.manifest.length);
        highlightSection('manifest');
        checkRecoverReady();
        return;
      }

      // PDF or TXT - try to parse as share
      if (name.endsWith('.pdf') || name.endsWith('.txt')) {
        const content = await readFileAsText(file);
        await parseAndAddShareUnified(content, file.name);
        return;
      }

      // Unknown file type - try to parse as share (might be README with different name)
      const content = await readFileAsText(file);
      if (shareRegex.test(content)) {
        await parseAndAddShareUnified(content, file.name);
      } else {
        showUnrecognizedFileError(file.name);
      }
    } catch (err) {
      errorHandlers.fileReadFailed(file.name);
    }
  }

  /**
   * Handle ZIP bundle - extract share and manifest, always replace manifest.
   */
  async function handleBundleZipUnified(file: File): Promise<void> {
    const buffer = await readFileAsArrayBuffer(file);
    const zipData = new Uint8Array(buffer);

    try {
      const bundle = extractBundle(zipData);
      let addedShare = false;
      let addedManifest = false;

      if (bundle.share) {
        addedShare = await tryAddShare(bundle.share);
      }

      if (bundle.manifest) {
        state.manifest = bundle.manifest;
        showManifestLoaded('MANIFEST.age', state.manifest.length, 'bundle');
        addedManifest = true;
      }

      highlightAddedSections(addedShare, addedManifest);
      checkRecoverReady();
    } catch {
      showError(
        t('error_bundle_extract_message', file.name),
        {
          title: t('error_bundle_extract_title'),
          guidance: t('error_bundle_extract_guidance'),
        }
      );
    }
  }

  /**
   * Handle HTML file - extract share and manifest, always replace manifest.
   */
  async function handleHTMLUnified(file: File): Promise<void> {
    const text = await readFileAsText(file);

    const personalizationData = extractPersonalizationFromHTML(text);
    if (!personalizationData) {
      showUnrecognizedFileError(file.name);
      return;
    }

    try {
      let addedShare = false;
      let addedManifest = false;
      const hasManifest = !!personalizationData.manifestB64;

      if (personalizationData.manifestB64) {
        const binary = atob(personalizationData.manifestB64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        state.manifest = bytes;
        showManifestLoaded('MANIFEST.age', state.manifest.length, 'html');
        addedManifest = true;
      }

      if (personalizationData.holderShare) {
        try {
          // Suppress duplicate warning if we got a manifest — the file was still useful
          addedShare = await tryAddShare(personalizationData.holderShare, { quiet: hasManifest });
        } catch {
          // Ignore invalid share
        }
      }

      if (!addedShare && !addedManifest) {
        showUnrecognizedFileError(file.name);
      } else {
        highlightAddedSections(addedShare, addedManifest);
      }

      checkRecoverReady();
    } catch {
      showUnrecognizedFileError(file.name);
    }
  }

  /**
   * Parse and add share with section highlighting.
   */
  async function parseAndAddShareUnified(content: string, filename: string): Promise<void> {
    if (!shareRegex.test(content)) {
      errorHandlers.noShareFound(filename);
      return;
    }

    try {
      const added = await tryAddShare(content);
      if (added) {
        highlightSection('share');
        checkRecoverReady();
      }
    } catch (err) {
      errorHandlers.invalidShare(filename, err instanceof Error ? err.message : undefined);
    }
  }

  /**
   * Briefly highlight a section to show where content was added.
   */
  function highlightSection(section: 'share' | 'manifest' | 'both'): void {
    const highlightDuration = 600;

    if (section === 'share' || section === 'both') {
      elements.step1Card?.classList.add('highlight-added');
      setTimeout(() => elements.step1Card?.classList.remove('highlight-added'), highlightDuration);
    }

    if (section === 'manifest' || section === 'both') {
      elements.step2Card?.classList.add('highlight-added');
      setTimeout(() => elements.step2Card?.classList.remove('highlight-added'), highlightDuration);
    }
  }

  function highlightAddedSections(addedShare: boolean, addedManifest: boolean): void {
    if (addedShare && addedManifest) {
      highlightSection('both');
    } else if (addedShare) {
      highlightSection('share');
    } else if (addedManifest) {
      highlightSection('manifest');
    }
  }

  // ============================================
  // Paste Functionality
  // ============================================

  function setupPaste(): void {
    elements.pasteToggleBtn?.addEventListener('click', () => {
      const isHidden = elements.pasteArea?.classList.contains('hidden');
      elements.pasteArea?.classList.toggle('hidden', !isHidden);
      if (isHidden) {
        elements.pasteInput?.focus();
      }
    });

    elements.pasteSubmitBtn?.addEventListener('click', async () => {
      const content = elements.pasteInput?.value.trim();
      if (!content) return;

      await parseAndAddShareFromPaste(content);
      if (elements.pasteInput) elements.pasteInput.value = '';
      elements.pasteArea?.classList.add('hidden');
    });

    elements.pasteInput?.addEventListener('keydown', (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault();
        elements.pasteSubmitBtn?.click();
      }
    });
  }

  async function parseAndAddShareFromPaste(content: string): Promise<void> {
    if (elements.shareDropZone) {
      clearInlineError(elements.shareDropZone);
    }

    let share: UIShare | undefined;

    // Try compact format first
    if (compactShareRegex.test(content.trim())) {
      try {
        share = await parseCompactShare(content.trim());
      } catch (err) {
        showError(
          err instanceof Error ? err.message : t('error_invalid_share_message', t('pasted_content')),
          {
            title: t('error_invalid_share_title'),
            guidance: t('error_invalid_share_guidance')
          }
        );
        return;
      }
    } else if (shareRegex.test(content)) {
      // Try PEM format
      try {
        share = await parseShare(content);
        share.compact = await encodeCompact(share);
      } catch {
        showError(
          t('error_invalid_share_message', t('pasted_content')),
          {
            title: t('error_invalid_share_title'),
            guidance: t('error_invalid_share_guidance')
          }
        );
        return;
      }
    } else {
      // Try to extract BIP39 words from the pasted text
      const extractedWords = extractWordsFromText(content);
      if (extractedWords.length >= 25) {
        try {
          const wordResult = await decodeShareWords(extractedWords);
          share = buildShareFromWords(wordResult);
        } catch (err) {
          // Words were detected but decoding failed — show the specific error
          toast.error(
            t('error_invalid_words_title'),
            err instanceof Error ? err.message : String(err),
            t('error_invalid_words_guidance')
          );
          return;
        }
      }

      if (!share) {
        showError(
          t('error_paste_no_share_message'),
          {
            title: t('error_paste_no_share_title'),
            guidance: t('error_paste_no_share_guidance')
          }
        );
        return;
      }
    }

    if (state.shares.some(s => s.index === share!.index)) {
      errorHandlers.duplicateShare(share!.index);
      return;
    }

    if (state.shares.length === 0 || (state.threshold === 0 && share.threshold > 0)) {
      state.threshold = share.threshold;
      state.total = share.total;
    }

    state.shares.push(share);
    updateSharesUI();
    checkRecoverReady();
  }

  // ============================================
  // Build Share from Decoded Words
  // ============================================

  function buildShareFromWords(wordResult: { data: Uint8Array; index: number }): UIShare {
    // Get version/total/threshold from first loaded share or personalization
    let version = 2;
    let total = 0;
    let threshold = 0;

    if (state.shares.length > 0) {
      version = state.shares[0].version;
      total = state.total;
      threshold = state.threshold;
    } else if (personalization) {
      total = personalization.total;
      threshold = personalization.threshold;
    }

    return {
      version,
      index: wordResult.index,
      threshold,
      total,
      data: wordResult.data,
      dataB64: bytesToBase64(wordResult.data),
    };
  }

  // ============================================
  // QR Code Scanner (BarcodeDetector API)
  // ============================================

  let scannerStream: MediaStream | null = null;
  let scannerAnimFrame: number | null = null;

  function setupScanner(): void {
    // Only show the button if BarcodeDetector is available
    if (!('BarcodeDetector' in window)) return;

    elements.scanQrBtn?.classList.remove('hidden');
    elements.scanQrBtn?.addEventListener('click', () => {
      openScanner();
    });

    elements.qrScannerClose?.addEventListener('click', closeScanner);
  }

  async function openScanner(): Promise<void> {
    elements.qrScannerModal?.classList.remove('hidden');

    try {
      scannerStream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment' }
      });
    } catch (_err) {
      toast.warning(t('scan_camera_error'), t('scan_camera_error'));
      closeScanner();
      return;
    }

    if (elements.qrVideo) {
      elements.qrVideo.srcObject = scannerStream;
    }

    const detector = new BarcodeDetector({ formats: ['qr_code'] });

    function scanLoop(): void {
      if (!scannerStream || !elements.qrVideo) return;

      // Wait until video is playing and has dimensions
      if (elements.qrVideo.readyState < 2 || elements.qrVideo.videoWidth === 0) {
        scannerAnimFrame = requestAnimationFrame(scanLoop);
        return;
      }

      detector.detect(elements.qrVideo).then(barcodes => {
        if (!scannerStream) return; // Scanner was closed

        for (const barcode of barcodes) {
          const value = barcode.rawValue.trim();
          // Check for compact share format directly or URL with fragment
          let compact = '';
          if (compactShareRegex.test(value)) {
            compact = value;
          } else {
            // Check for URL with #share= fragment
            try {
              const url = new URL(value);
              const hash = url.hash;
              if (hash && hash.startsWith('#share=')) {
                const decoded = decodeURIComponent(hash.slice('#share='.length));
                if (compactShareRegex.test(decoded)) {
                  compact = decoded;
                }
              }
            } catch {
              // Not a URL, ignore
            }
          }

          if (compact) {
            handleScannedShare(compact);
            return;
          }
        }

        scannerAnimFrame = requestAnimationFrame(scanLoop);
      }).catch(() => {
        // Detection error, keep trying
        scannerAnimFrame = requestAnimationFrame(scanLoop);
      });
    }

    scannerAnimFrame = requestAnimationFrame(scanLoop);
  }

  async function handleScannedShare(compact: string): Promise<void> {
    closeScanner();
    await parseAndAddShareFromPaste(compact);
  }

  function closeScanner(): void {
    if (scannerAnimFrame !== null) {
      cancelAnimationFrame(scannerAnimFrame);
      scannerAnimFrame = null;
    }

    if (scannerStream) {
      scannerStream.getTracks().forEach(track => track.stop());
      scannerStream = null;
    }

    if (elements.qrVideo) {
      elements.qrVideo.srcObject = null;
    }

    elements.qrScannerModal?.classList.add('hidden');
  }

  // ============================================
  // Shares UI
  // ============================================

  // Resolve a display name for a share: use holder if set, otherwise look up
  // the friend name from personalization data by share index, fall back to generic.
  function resolveShareName(share: UIShare): string {
    if (share.holder) return share.holder;
    if (personalization) {
      if (personalization.holder) {
        // Check if this matches the bundle holder's own share index
        const holderShare = state.shares.find(s => (s as UIShare).isHolder);
        if (holderShare && holderShare.index === share.index) {
          return personalization.holder;
        }
      }
      const friend = personalization.otherFriends.find(f => f.shareIndex === share.index);
      if (friend) return friend.name;
    }
    return 'Share ' + share.index;
  }

  function updateSharesUI(): void {
    if (!elements.sharesList) return;

    elements.sharesList.innerHTML = '';

    state.shares.forEach((share, idx) => {
      const item = document.createElement('div');
      item.className = 'share-item valid';

      const displayName = resolveShareName(share as UIShare);

      const isHolderShare = (share as UIShare).isHolder ||
        (personalization && share.holder &&
         share.holder.toLowerCase() === personalization.holder.toLowerCase());

      const holderLabel = isHolderShare ? ` (${t('your_share')})` : '';
      const showRemove = !isHolderShare;

      item.innerHTML = `
        <span class="icon">&#9989;</span>
        <div class="details">
          <div class="name">${escapeHtml(displayName)}${holderLabel}</div>
        </div>
        ${showRemove ? `<button class="remove" data-idx="${idx}" title="${t('remove')}">&times;</button>` : ''}
      `;
      elements.sharesList?.appendChild(item);
    });

    // Add remove handlers
    elements.sharesList.querySelectorAll('.remove').forEach(btn => {
      btn.addEventListener('click', (e) => {
        const target = e.target as HTMLElement;
        const idx = parseInt(target.dataset.idx || '0', 10);
        state.shares.splice(idx, 1);
        if (state.shares.length === 0) {
          state.threshold = 0;
          state.total = 0;
        }
        updateSharesUI();
        updateContactList();
        checkRecoverReady();
      });
    });

    // Update threshold info
    if (state.threshold > 0 && elements.thresholdInfo) {
      const needed = Math.max(0, state.threshold - state.shares.length);
      const needLabel = needed === 1 ? t('need_more_one') : t('need_more', needed);
      elements.thresholdInfo.innerHTML = needed > 0
        ? `&#128274; ${needLabel} (${t('shares_of', state.shares.length, state.threshold)})`
        : `&#9989; ${t('ready')} (${t('shares_of', state.shares.length, state.threshold)})`;
      elements.thresholdInfo.className = 'threshold-info' + (needed === 0 ? ' ready' : '');
      elements.thresholdInfo.classList.remove('hidden');

      // Collapse step 1 content when threshold is met
      elements.step1Card?.classList.toggle('threshold-met', needed === 0);
    } else {
      elements.thresholdInfo?.classList.add('hidden');
      elements.step1Card?.classList.remove('threshold-met');
    }

    updateContactList();
  }

  // ============================================
  // Manifest UI
  // ============================================

  function showManifestLoaded(filename: string, size: number, source: 'file' | 'bundle' | 'embedded' | 'html' = 'file'): void {
    elements.manifestDropZone?.classList.add('hidden');

    if (elements.manifestStatus) {
      const sourceLabels: Record<string, string> = {
        file: t('loaded'),
        bundle: t('manifest_loaded_bundle'),
        embedded: t('manifest_loaded_embedded'),
        html: t('manifest_loaded_html'),
      };
      const sourceLabel = sourceLabels[source] || t('loaded');
      elements.manifestStatus.innerHTML = `
        <span class="icon">&#9989;</span>
        <div style="flex: 1;">
          <strong>${escapeHtml(filename)}</strong> ${sourceLabel}
          <div style="font-size: 0.875rem; color: #6c757d;">${formatSize(size)}</div>
        </div>
        <button class="clear-manifest" title="${t('remove')}">&times;</button>
      `;
      elements.manifestStatus.classList.remove('hidden');
      elements.manifestStatus.classList.add('loaded');

      const clearBtn = elements.manifestStatus.querySelector('.clear-manifest');
      clearBtn?.addEventListener('click', clearManifest);
    }
  }

  function clearManifest(): void {
    state.manifest = null;
    elements.manifestStatus?.classList.add('hidden');
    elements.manifestStatus?.classList.remove('loaded');
    elements.manifestDropZone?.classList.remove('hidden');
    checkRecoverReady();
  }

  // ============================================
  // Buttons Setup
  // ============================================

  function setupButtons(): void {
    elements.recoverBtn?.addEventListener('click', startRecovery);
    elements.downloadAllBtn?.addEventListener('click', downloadAll);
  }

  function checkRecoverReady(): void {
    const ready = state.manifest !== null && (
      (state.threshold > 0 && state.shares.length >= state.threshold) ||
      (state.threshold === 0 && state.shares.length >= 2)
    );

    if (elements.recoverBtn) {
      elements.recoverBtn.disabled = !ready;
    }

    if (ready && !state.recovering && !state.recoveryComplete) {
      startRecovery();
    }
  }

  function collapseInputSteps(): void {
    elements.step1Card?.classList.add('collapsed');
    elements.step2Card?.classList.add('collapsed');
  }

  // ============================================
  // Recovery Process
  // ============================================

  async function startRecovery(): Promise<void> {
    if (state.recovering) return;
    state.recovering = true;

    collapseInputSteps();

    if (elements.recoverBtn) elements.recoverBtn.disabled = true;
    elements.progressBar?.classList.remove('hidden');
    if (elements.statusMessage) elements.statusMessage.className = 'status-message';
    if (elements.filesList) elements.filesList.innerHTML = '';
    elements.downloadActions?.classList.add('hidden');

    try {
      setProgress(10);
      setStatus(t('combining'));

      // Convert shares to raw bytes and combine
      const shareBytes = state.shares.map(s => base64ToBytes(s.dataB64));
      const recovered = await combine(shareBytes);
      const version = state.shares[0].version;
      const passphrase = recoverPassphrase(recovered, version);

      setProgress(30);

      setStatus(t('decrypting'));
      const decrypted = await decrypt(state.manifest!, passphrase);

      setProgress(60);

      state.decryptedArchive = decrypted;

      setStatus(t('reading'));
      const files = await extractTarGz(decrypted);

      setProgress(90);

      files.forEach(file => {
        const item = document.createElement('div');
        item.className = 'file-item';
        item.innerHTML = `
          <span class="icon">&#128196;</span>
          <span class="name">${escapeHtml(file.name)}</span>
          <span class="size">${formatSize(file.data.length)}</span>
        `;
        elements.filesList?.appendChild(item);
      });

      setProgress(100);
      setStatus(t('complete', files.length), 'success');
      elements.downloadActions?.classList.remove('hidden');
      elements.recoverBtn?.classList.add('hidden');
      state.recoveryComplete = true;

    } catch (err) {
      const errorMsg = (err instanceof Error) ? err.message : String(err);

      if (errorMsg.includes('decrypt') || errorMsg.includes('passphrase') || errorMsg.includes('incorrect')) {
        errorHandlers.decryptionFailed(err);
        setStatus(t('error_decrypt_status'), 'error');
      } else if (errorMsg.includes('extract') || errorMsg.includes('tar') || errorMsg.includes('gzip')) {
        errorHandlers.extractionFailed(err);
        setStatus(t('error_extract_status'), 'error');
      } else {
        toast.error(
          t('error_recovery_title'),
          errorMsg,
          t('error_recovery_guidance'),
          [
            { id: 'retry', label: t('action_try_again'), primary: true, onClick: () => startRecovery() }
          ]
        );
        setStatus(t('error', errorMsg), 'error');
      }

      elements.step1Card?.classList.remove('collapsed');
      elements.step2Card?.classList.remove('collapsed');
    } finally {
      state.recovering = false;
      if (elements.recoverBtn) elements.recoverBtn.disabled = false;
    }
  }

  function setProgress(percent: number): void {
    const fill = elements.progressBar?.querySelector('.fill') as HTMLElement | null;
    if (fill) {
      fill.style.width = percent + '%';
    }
  }

  function setStatus(msg: string, type?: string): void {
    if (elements.statusMessage) {
      elements.statusMessage.textContent = msg;
      elements.statusMessage.className = 'status-message' + (type ? ' ' + type : '');
    }
  }

  // ============================================
  // Download
  // ============================================

  function downloadAll(): void {
    if (!state.decryptedArchive) return;

    const blob = new Blob([state.decryptedArchive as BlobPart], { type: 'application/gzip' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'manifest.tar.gz';
    a.click();
    URL.revokeObjectURL(url);

    clearSensitiveState();
  }

  function clearSensitiveState(): void {
    state.decryptedArchive = undefined;
    state.manifest = null;
  }

  // ============================================
  // Word Extraction
  // ============================================

  // extractWordsFromText extracts BIP39 words from text, handling:
  //   - Numbered two-column grids: " 1. merit   14. beef" (sorted by number)
  //   - Plain word lists: "merit often shuffle wedding"
  // Supports Unicode letters (accented/umlauted characters like ábaco, günther).
  function extractWordsFromText(text: string): string[] {
    // Try to parse numbered format first (e.g. "1. word", "13. ábaco")
    const numbered: { idx: number; word: string }[] = [];
    const re = /(\d+)\.\s+([\p{L}]+)/gu;
    let m;
    while ((m = re.exec(text)) !== null) {
      numbered.push({ idx: parseInt(m[1], 10), word: m[2].toLowerCase() });
    }

    if (numbered.length >= 25) {
      // Sort by number to handle two-column grids correctly
      numbered.sort((a, b) => a.idx - b.idx);
      return numbered.map(e => e.word);
    }

    // Fallback: plain word list (no numbers)
    return text
      .toLowerCase()
      .split(/\s+/)
      .filter(w => w.length > 0 && /^[\p{L}]+$/u.test(w));
  }

  // ============================================
  // Utility Functions
  // ============================================

  function readFileAsText(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as string);
      reader.onerror = reject;
      reader.readAsText(file);
    });
  }

  function readFileAsArrayBuffer(file: File): Promise<ArrayBuffer> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as ArrayBuffer);
      reader.onerror = reject;
      reader.readAsArrayBuffer(file);
    });
  }

  // Global export for HTML templates
  window.rememoryUpdateUI = function(): void {
    updateSharesUI();
    updateContactList();
  };

  // Start
  document.addEventListener('DOMContentLoaded', init);
})();
