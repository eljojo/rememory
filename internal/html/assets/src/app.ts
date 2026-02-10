// ReMemory Recovery Tool - Browser-based recovery using Go WASM

import type {
  RecoveryState,
  PersonalizationData,
  FriendInfo,
  ShareInput,
  ToastAction,
  TranslationFunction
} from './types';

// Translation function (defined in HTML)
declare const t: TranslationFunction;

(function() {
  'use strict';

  // Import shared utilities
  const { escapeHtml, formatSize, toast, showInlineError, clearInlineError } = window.rememoryUtils;

  // State
  const state: RecoveryState = {
    shares: [],
    manifest: null,
    threshold: 0,
    total: 0,
    wasmReady: false,
    recovering: false,
    recoveryComplete: false
  };

  // DOM elements interface
  interface Elements {
    loadingOverlay: HTMLElement | null;
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
    loadingOverlay: document.getElementById('loading-overlay'),
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
    wasmLoadFailed(_err: unknown): void {
      toast.error(
        t('error_wasm_title'),
        t('error_wasm_message'),
        t('error_wasm_guidance'),
        [
          { id: 'reload', label: t('action_reload'), primary: true, onClick: () => window.location.reload() },
          { id: 'cli', label: t('action_use_cli'), onClick: () => window.open('https://github.com/eljojo/rememory', '_blank') }
        ]
      );
    },

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
              state.shares = [];
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
    setupButtons();
    setupPaste();
    setupScanner();

    // Render contact list immediately (doesn't need WASM)
    if (personalization?.otherFriends && personalization.otherFriends.length > 0) {
      renderContactList();
      elements.contactListSection?.classList.remove('hidden');
    }

    await loadWasm();

    // Load personalization data after WASM is ready
    if (personalization) {
      loadPersonalizationData();
    }

    // Check URL fragment for compact share (e.g. #share=RM1:2:5:3:BASE64:CHECK)
    loadShareFromFragment();
  }

  // ============================================
  // Personalization
  // ============================================

  function loadPersonalizationData(): void {
    if (!personalization) return;

    // Load the holder's share automatically
    if (personalization.holderShare) {
      const result = window.rememoryParseShare(personalization.holderShare);
      if (!result.error && result.share) {
        const share = result.share;
        share.isHolder = true;
        state.threshold = share.threshold;
        state.total = share.total;
        state.shares.push(share);

        updateSharesUI();
        updateContactList();
      }
    }

    checkRecoverReady();
  }

  // ============================================
  // URL Fragment Share Loading
  // ============================================

  function loadShareFromFragment(): void {
    if (!state.wasmReady) return;

    const hash = window.location.hash;
    if (!hash || !hash.startsWith('#share=')) return;

    const compact = decodeURIComponent(hash.slice('#share='.length));
    if (!compactShareRegex.test(compact)) return;

    const result = window.rememoryParseCompactShare(compact);
    if (result.error || !result.share) return;

    const share = result.share;

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

      const contactInfo = friend.contact ? escapeHtml(friend.contact) : '';

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
  // WASM Loading
  // ============================================

  async function loadWasm(): Promise<void> {
    try {
      const go = new window.Go();
      const result = await WebAssembly.instantiateStreaming(
        fetch('recover.wasm'),
        go.importObject
      );
      go.run(result.instance);

      await waitForWasm();
      state.wasmReady = true;
      window.rememoryAppReady = true;
      elements.loadingOverlay?.classList.add('hidden');
    } catch (err) {
      // Try loading from embedded gzip-compressed base64 as fallback
      if (typeof window.WASM_BINARY !== 'undefined') {
        try {
          const go = new window.Go();
          const bytes = await decodeAndDecompressWasm(window.WASM_BINARY);
          const result = await WebAssembly.instantiate(bytes, go.importObject);
          go.run(result.instance);
          await waitForWasm();
          state.wasmReady = true;
          window.rememoryAppReady = true;
          elements.loadingOverlay?.classList.add('hidden');
          return;
        } catch (e) {
          errorHandlers.wasmLoadFailed(e);
          return;
        }
      }
      errorHandlers.wasmLoadFailed(err);
    }
  }

  function waitForWasm(): Promise<void> {
    return new Promise((resolve) => {
      const check = (): void => {
        if (window.rememoryReady) {
          resolve();
        } else {
          setTimeout(check, 50);
        }
      };
      check();
    });
  }

  async function decodeAndDecompressWasm(base64: string): Promise<ArrayBuffer> {
    const compressed = Uint8Array.from(atob(base64), c => c.charCodeAt(0));

    if (typeof DecompressionStream !== 'undefined') {
      const ds = new DecompressionStream('gzip');
      const writer = ds.writable.getWriter();
      writer.write(compressed);
      writer.close();
      const reader = ds.readable.getReader();
      const chunks: Uint8Array[] = [];
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        if (value) chunks.push(value);
      }
      const totalLength = chunks.reduce((acc, chunk) => acc + chunk.length, 0);
      const bytes = new Uint8Array(totalLength);
      let offset = 0;
      for (const chunk of chunks) {
        bytes.set(chunk, offset);
        offset += chunk.length;
      }
      return bytes.buffer;
    } else if (typeof (window as unknown as { pako?: { inflate: (data: Uint8Array) => Uint8Array } }).pako !== 'undefined') {
      const pako = (window as unknown as { pako: { inflate: (data: Uint8Array) => Uint8Array } }).pako;
      return pako.inflate(compressed).buffer as ArrayBuffer;
    } else {
      throw new Error('Browser does not support DecompressionStream');
    }
  }

  // ============================================
  // Drop Zone Setup
  // ============================================

  function setupDropZones(): void {
    if (elements.shareDropZone && elements.shareFileInput) {
      setupDropZone(elements.shareDropZone, elements.shareFileInput, handleShareFiles);
    }
    if (elements.manifestDropZone && elements.manifestFileInput) {
      setupDropZone(elements.manifestDropZone, elements.manifestFileInput, handleManifestFiles);
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
      dropZone.classList.remove('dragover');
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
    if (!state.wasmReady) {
      toast.warning(t('error_not_ready_title'), t('error_not_ready_message'), t('error_not_ready_guidance'));
      return;
    }

    if (elements.shareDropZone) {
      clearInlineError(elements.shareDropZone);
    }

    // Try compact format first, then PEM format
    let share: import('./types').ParsedShare | undefined;

    if (compactShareRegex.test(content.trim())) {
      const result = window.rememoryParseCompactShare(content.trim());
      if (result.error || !result.share) {
        showError(
          result.error || t('error_invalid_share_message', t('pasted_content')),
          {
            title: t('error_invalid_share_title'),
            guidance: t('error_invalid_share_guidance')
          }
        );
        return;
      }
      share = result.share;
    } else if (shareRegex.test(content)) {
      const result = window.rememoryParseShare(content);
      if (result.error || !result.share) {
        showError(
          t('error_invalid_share_message', t('pasted_content')),
          {
            title: t('error_invalid_share_title'),
            guidance: t('error_invalid_share_guidance')
          }
        );
        return;
      }
      share = result.share;
    } else {
      // Try to extract BIP39 words from the pasted text
      const extractedWords = extractWordsFromText(content);
      if (extractedWords.length >= 25) {
        const wordResult = window.rememoryDecodeWords(extractedWords);
        if (!wordResult.error && wordResult.index > 0) {
          // Valid words found — add share directly (25th word provides the index)
          share = buildShareFromWords(wordResult);
          if (!share) return; // error already shown
        } else if (wordResult.error) {
          // Words were detected but decoding failed — show the specific error
          toast.error(
            t('error_invalid_words_title'),
            wordResult.error,
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

    if (state.shares.some(s => s.index === share.index)) {
      errorHandlers.duplicateShare(share.index);
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

  function buildShareFromWords(wordResult: { data: Uint8Array; index: number; checksum: string }): import('./types').ParsedShare | null {
    const shareIndex = wordResult.index;

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

    // Construct ParsedShare from decoded data
    const dataB64 = btoa(String.fromCharCode(...wordResult.data));
    return {
      version,
      index: shareIndex,
      threshold,
      total,
      dataB64
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
      if (!state.wasmReady) {
        toast.warning(t('error_not_ready_title'), t('error_not_ready_message'), t('error_not_ready_guidance'));
        return;
      }
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
  // Share File Handling
  // ============================================

  async function handleShareFiles(files: FileList | File[]): Promise<void> {
    if (elements.shareDropZone) {
      clearInlineError(elements.shareDropZone);
    }

    for (const file of Array.from(files)) {
      try {
        if (file.name.endsWith('.zip') || file.type === 'application/zip') {
          await handleBundleZip(file);
        } else {
          const content = await readFileAsText(file);
          await parseAndAddShare(content, file.name);
        }
      } catch (_err) {
        errorHandlers.fileReadFailed(file.name);
      }
    }
  }

  async function handleBundleZip(file: File): Promise<void> {
    if (!state.wasmReady) {
      toast.warning(t('error_not_ready_title'), t('error_not_ready_message'), t('error_not_ready_guidance'));
      return;
    }

    const buffer = await readFileAsArrayBuffer(file);
    const zipData = new Uint8Array(buffer);

    const result = window.rememoryExtractBundle(zipData);
    if (result.error || !result.share) {
      if (elements.shareDropZone) {
        showError(
          t('error_bundle_extract_message', file.name),
          {
            title: t('error_bundle_extract_title'),
            guidance: t('error_bundle_extract_guidance'),
            inline: true,
            targetElement: elements.shareDropZone
          }
        );
      }
      return;
    }

    const share = result.share;

    if (state.shares.some(s => s.index === share.index)) {
      errorHandlers.duplicateShare(share.index);
      return;
    }

    if (state.shares.length === 0 || (state.threshold === 0 && share.threshold > 0)) {
      state.threshold = share.threshold;
      state.total = share.total;
    }

    state.shares.push(share);
    updateSharesUI();

    if (result.manifest && !state.manifest) {
      state.manifest = result.manifest;
      showManifestLoaded('MANIFEST.age', state.manifest.length, true);
    }

    checkRecoverReady();
  }

  async function parseAndAddShare(content: string, filename: string): Promise<void> {
    if (!state.wasmReady) {
      toast.warning(t('error_not_ready_title'), t('error_not_ready_message'), t('error_not_ready_guidance'));
      return;
    }

    if (!shareRegex.test(content)) {
      errorHandlers.noShareFound(filename);
      return;
    }

    const result = window.rememoryParseShare(content);
    if (result.error || !result.share) {
      errorHandlers.invalidShare(filename, result.error);
      return;
    }

    const share = result.share;

    if (state.shares.some(s => s.index === share.index)) {
      errorHandlers.duplicateShare(share.index);
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
  // Shares UI
  // ============================================

  function updateSharesUI(): void {
    if (!elements.sharesList) return;

    elements.sharesList.innerHTML = '';

    state.shares.forEach((share, idx) => {
      const item = document.createElement('div');
      item.className = 'share-item valid';

      const isHolderShare = share.isHolder ||
        (personalization && share.holder &&
         share.holder.toLowerCase() === personalization.holder.toLowerCase());

      const holderLabel = isHolderShare ? ` (${t('your_share')})` : '';
      const showRemove = !isHolderShare;

      item.innerHTML = `
        <span class="icon">&#9989;</span>
        <div class="details">
          <div class="name">${escapeHtml(share.holder || 'Share ' + share.index)}${holderLabel}</div>
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
  // Manifest Handling
  // ============================================

  async function handleManifestFiles(files: FileList | File[]): Promise<void> {
    const fileArray = Array.from(files);
    if (fileArray.length === 0) return;

    if (elements.manifestDropZone) {
      clearInlineError(elements.manifestDropZone);
    }

    try {
      const file = fileArray[0];

      if (file.name.endsWith('.zip') || file.type === 'application/zip') {
        await handleBundleZip(file);
        return;
      }

      if (!file.name.endsWith('.age')) {
        if (elements.manifestDropZone) {
          showError(
            t('error_wrong_manifest_message', file.name),
            {
              title: t('error_wrong_manifest_title'),
              guidance: t('error_wrong_manifest_guidance'),
              inline: true,
              targetElement: elements.manifestDropZone
            }
          );
        }
        return;
      }

      const buffer = await readFileAsArrayBuffer(file);
      state.manifest = new Uint8Array(buffer);

      showManifestLoaded(file.name, state.manifest.length);
      checkRecoverReady();
    } catch (_err) {
      errorHandlers.fileReadFailed(fileArray[0]?.name || 'file');
    }
  }

  function showManifestLoaded(filename: string, size: number, fromBundle = false): void {
    elements.manifestDropZone?.classList.add('hidden');

    if (elements.manifestStatus) {
      elements.manifestStatus.innerHTML = `
        <span class="icon">&#9989;</span>
        <div style="flex: 1;">
          <strong>${escapeHtml(filename)}</strong> ${fromBundle ? t('manifest_loaded_bundle') : t('loaded')}
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

      const sharesForCombine: ShareInput[] = state.shares.map(s => ({
        version: s.version,
        index: s.index,
        dataB64: s.dataB64
      }));

      const combineResult = window.rememoryCombineShares(sharesForCombine);
      if (combineResult.error || !combineResult.passphrase) {
        throw new Error(combineResult.error || 'Failed to combine shares');
      }

      const passphrase = combineResult.passphrase;
      setProgress(30);

      setStatus(t('decrypting'));
      const decryptResult = window.rememoryDecryptManifest(state.manifest!, passphrase);
      if (decryptResult.error || !decryptResult.data) {
        throw new Error(decryptResult.error || 'Failed to decrypt');
      }

      setProgress(60);

      state.decryptedArchive = decryptResult.data;

      setStatus(t('reading'));
      const extractResult = window.rememoryExtractTarGz(decryptResult.data);
      if (extractResult.error || !extractResult.files) {
        throw new Error(extractResult.error || 'Failed to extract');
      }

      setProgress(90);

      const files = extractResult.files;

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

  // ============================================
  // Global Exports
  // ============================================

  window.rememoryUpdateUI = function(): void {
    updateSharesUI();
    updateContactList();
  };

  // Start
  document.addEventListener('DOMContentLoaded', init);
})();
