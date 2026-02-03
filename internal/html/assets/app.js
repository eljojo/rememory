// ReMemory Recovery Tool - Browser-based recovery using Go WASM

(function() {
  'use strict';

  // State
  const state = {
    shares: [],        // Array of parsed share objects
    manifest: null,    // Uint8Array of MANIFEST.age content
    threshold: 0,      // Required shares (from first parsed share)
    total: 0,          // Total shares
    wasmReady: false,
    recovering: false
  };

  // DOM elements
  const elements = {
    loadingOverlay: document.getElementById('loading-overlay'),
    shareDropZone: document.getElementById('share-drop-zone'),
    shareFileInput: document.getElementById('share-file-input'),
    sharesList: document.getElementById('shares-list'),
    thresholdInfo: document.getElementById('threshold-info'),
    manifestDropZone: document.getElementById('manifest-drop-zone'),
    manifestFileInput: document.getElementById('manifest-file-input'),
    manifestStatus: document.getElementById('manifest-status'),
    recoverBtn: document.getElementById('recover-btn'),
    recoverSection: document.getElementById('recover-section'),
    progressBar: document.getElementById('progress-bar'),
    statusMessage: document.getElementById('status-message'),
    filesList: document.getElementById('files-list'),
    downloadActions: document.getElementById('download-actions'),
    downloadAllBtn: document.getElementById('download-all-btn')
  };

  // Share regex to extract from README.txt content
  const shareRegex = /-----BEGIN REMEMORY SHARE-----([\s\S]*?)-----END REMEMORY SHARE-----/;

  // Initialize
  async function init() {
    setupDropZones();
    setupButtons();
    await loadWasm();
  }

  // Load WASM module
  async function loadWasm() {
    try {
      const go = new Go();
      const result = await WebAssembly.instantiateStreaming(
        fetch('recover.wasm'),
        go.importObject
      );
      go.run(result.instance);

      // Wait for WASM to signal ready
      await waitForWasm();
      state.wasmReady = true;
      window.rememoryAppReady = true;
      elements.loadingOverlay.classList.add('hidden');
    } catch (err) {
      // Try loading from embedded base64 as fallback
      if (typeof WASM_BINARY !== 'undefined') {
        try {
          const go = new Go();
          const bytes = base64ToArrayBuffer(WASM_BINARY);
          const result = await WebAssembly.instantiate(bytes, go.importObject);
          go.run(result.instance);
          await waitForWasm();
          state.wasmReady = true;
          window.rememoryAppReady = true;
          elements.loadingOverlay.classList.add('hidden');
          return;
        } catch (e) {
          // WASM initialization failed - will show error to user below
        }
      }
      showError(t('error', err.message));
    }
  }

  function waitForWasm() {
    return new Promise((resolve) => {
      const check = () => {
        if (window.rememoryReady) {
          resolve();
        } else {
          setTimeout(check, 50);
        }
      };
      check();
    });
  }

  function base64ToArrayBuffer(base64) {
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes.buffer;
  }

  // Setup drop zones
  function setupDropZones() {
    // Share drop zone
    setupDropZone(elements.shareDropZone, elements.shareFileInput, handleShareFiles);

    // Manifest drop zone
    setupDropZone(elements.manifestDropZone, elements.manifestFileInput, handleManifestFiles);
  }

  function setupDropZone(dropZone, fileInput, handler) {
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
      handler(e.dataTransfer.files);
    });

    fileInput.addEventListener('change', async (e) => {
      const files = Array.from(e.target.files);
      e.target.value = ''; // Reset for re-selection
      await handler(files);
    });
  }

  // Handle share file uploads
  async function handleShareFiles(files) {
    for (const file of files) {
      try {
        // Check if it's a ZIP file (bundle)
        if (file.name.endsWith('.zip') || file.type === 'application/zip') {
          await handleBundleZip(file);
        } else {
          const content = await readFileAsText(file);
          await parseAndAddShare(content, file.name);
        }
      } catch (err) {
        showError(t('error', 'Failed to read file'));
      }
    }
  }

  // Handle bundle ZIP file - extract share and optionally manifest
  async function handleBundleZip(file) {
    if (!state.wasmReady) {
      showError(t('error', 'Recovery module not ready'));
      return;
    }

    const buffer = await readFileAsArrayBuffer(file);
    const zipData = new Uint8Array(buffer);

    const result = window.rememoryExtractBundle(zipData);
    if (result.error) {
      showError(t('error', result.error));
      return;
    }

    // Add the share
    const share = result.share;

    // Check for duplicate index
    if (state.shares.some(s => s.index === share.index)) {
      showError(t('duplicate', share.index));
      return;
    }

    // Set threshold/total from first share
    if (state.shares.length === 0) {
      state.threshold = share.threshold;
      state.total = share.total;
    }

    state.shares.push(share);
    updateSharesUI();

    // If manifest is included and we don't have one yet, use it
    if (result.manifest && !state.manifest) {
      state.manifest = result.manifest;
      elements.manifestStatus.innerHTML = `
        <span class="icon">&#9989;</span>
        <div>
          <strong>MANIFEST.age</strong> ${t('manifest_loaded_bundle')}
          <div style="font-size: 0.875rem; color: #6c757d;">${formatSize(state.manifest.length)}</div>
        </div>
      `;
      elements.manifestStatus.classList.remove('hidden');
      elements.manifestStatus.classList.add('loaded');
    }

    checkRecoverReady();
  }

  async function parseAndAddShare(content, filename) {
    if (!state.wasmReady) {
      showError(t('error', 'Recovery module not ready'));
      return;
    }

    // Check if content contains a share
    if (!shareRegex.test(content)) {
      showError(t('no_share', filename));
      return;
    }

    const result = window.rememoryParseShare(content);
    if (result.error) {
      showError(t('invalid_share', filename, result.error));
      return;
    }

    const share = result.share;

    // Check for duplicate index
    if (state.shares.some(s => s.index === share.index)) {
      showError(t('duplicate', share.index));
      return;
    }

    // Set threshold/total from first share
    if (state.shares.length === 0) {
      state.threshold = share.threshold;
      state.total = share.total;
    }

    state.shares.push(share);
    updateSharesUI();
    checkRecoverReady();
  }

  function updateSharesUI() {
    elements.sharesList.innerHTML = '';

    state.shares.forEach((share, idx) => {
      const item = document.createElement('div');
      item.className = 'share-item valid';
      item.innerHTML = `
        <span class="icon">&#9989;</span>
        <div class="details">
          <div class="name">${escapeHtml(share.holder || 'Share ' + share.index)}</div>
          <div class="meta">${t('share_index', share.index, share.total)}</div>
        </div>
        <button class="remove" data-idx="${idx}" title="${t('remove')}">&times;</button>
      `;
      elements.sharesList.appendChild(item);
    });

    // Add remove handlers
    elements.sharesList.querySelectorAll('.remove').forEach(btn => {
      btn.addEventListener('click', (e) => {
        const idx = parseInt(e.target.dataset.idx, 10);
        state.shares.splice(idx, 1);
        if (state.shares.length === 0) {
          state.threshold = 0;
          state.total = 0;
        }
        updateSharesUI();
        checkRecoverReady();
      });
    });

    // Update threshold info
    if (state.threshold > 0) {
      const needed = Math.max(0, state.threshold - state.shares.length);
      elements.thresholdInfo.innerHTML = needed > 0
        ? `&#128274; ${t('need_more', needed)} (${t('shares_of', state.shares.length, state.threshold)})`
        : `&#9989; ${t('ready')} (${t('shares_of', state.shares.length, state.threshold)})`;
      elements.thresholdInfo.className = 'threshold-info' + (needed === 0 ? ' ready' : '');
      elements.thresholdInfo.classList.remove('hidden');
    } else {
      elements.thresholdInfo.classList.add('hidden');
    }
  }

  // Handle manifest file upload
  async function handleManifestFiles(files) {
    if (files.length === 0) return;

    try {
      const file = files[0];

      // Check if it's a ZIP file (bundle) - extract manifest and share from it
      if (file.name.endsWith('.zip') || file.type === 'application/zip') {
        await handleBundleZip(file);
        return;
      }

      const buffer = await readFileAsArrayBuffer(file);
      state.manifest = new Uint8Array(buffer);

      elements.manifestStatus.innerHTML = `
        <span class="icon">&#9989;</span>
        <div>
          <strong>${escapeHtml(file.name)}</strong> ${t('loaded')}
          <div style="font-size: 0.875rem; color: #6c757d;">${formatSize(state.manifest.length)}</div>
        </div>
      `;
      elements.manifestStatus.classList.remove('hidden');
      elements.manifestStatus.classList.add('loaded');
      checkRecoverReady();
    } catch (err) {
      showError(t('error', err.message));
    }
  }

  // Setup buttons
  function setupButtons() {
    elements.recoverBtn.addEventListener('click', startRecovery);
    elements.downloadAllBtn.addEventListener('click', downloadAll);
  }

  function checkRecoverReady() {
    const ready = state.shares.length >= state.threshold &&
                  state.threshold > 0 &&
                  state.manifest !== null;
    elements.recoverBtn.disabled = !ready;
  }

  // Recovery process
  async function startRecovery() {
    if (state.recovering) return;
    state.recovering = true;

    elements.recoverBtn.disabled = true;
    elements.progressBar.classList.remove('hidden');
    elements.statusMessage.className = 'status-message';
    elements.filesList.innerHTML = '';
    elements.downloadActions.classList.add('hidden');

    try {
      // Step 1: Combine shares
      setProgress(10);
      setStatus(t('combining'));

      const sharesForCombine = state.shares.map(s => ({
        index: s.index,
        dataB64: s.dataB64
      }));

      const combineResult = window.rememoryCombineShares(sharesForCombine);
      if (combineResult.error) {
        throw new Error(combineResult.error);
      }

      const passphrase = combineResult.passphrase;
      setProgress(30);

      // Step 2: Decrypt manifest
      setStatus(t('decrypting'));
      const decryptResult = window.rememoryDecryptManifest(state.manifest, passphrase);
      if (decryptResult.error) {
        throw new Error(decryptResult.error);
      }

      setProgress(60);

      // Store the decrypted tar.gz for download
      state.decryptedArchive = decryptResult.data;

      // Step 3: Extract tar.gz to show file list (preview only)
      setStatus(t('reading'));
      const extractResult = window.rememoryExtractTarGz(decryptResult.data);
      if (extractResult.error) {
        throw new Error(extractResult.error);
      }

      setProgress(90);

      // Step 4: Display files (preview)
      const files = extractResult.files;

      files.forEach(file => {
        const item = document.createElement('div');
        item.className = 'file-item';
        item.innerHTML = `
          <span class="icon">&#128196;</span>
          <span class="name">${escapeHtml(file.name)}</span>
          <span class="size">${formatSize(file.data.length)}</span>
        `;
        elements.filesList.appendChild(item);
      });

      setProgress(100);
      setStatus(t('complete', files.length), 'success');
      elements.downloadActions.classList.remove('hidden');

    } catch (err) {
      setStatus(t('error', err.message), 'error');
    } finally {
      state.recovering = false;
      elements.recoverBtn.disabled = false;
    }
  }

  function setProgress(percent) {
    const fill = elements.progressBar.querySelector('.fill');
    fill.style.width = percent + '%';
  }

  function setStatus(msg, type) {
    elements.statusMessage.textContent = msg;
    elements.statusMessage.className = 'status-message' + (type ? ' ' + type : '');
  }

  // Download the decrypted archive
  function downloadAll() {
    if (!state.decryptedArchive) return;

    const blob = new Blob([state.decryptedArchive], { type: 'application/gzip' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'manifest.tar.gz';
    a.click();
    URL.revokeObjectURL(url);

    // Security: Clear sensitive data from memory after download
    clearSensitiveState();
  }

  // Clear sensitive data from state to minimize memory exposure
  function clearSensitiveState() {
    state.decryptedArchive = null;
    state.manifest = null;
    // Note: shares contain metadata but not the actual secret
  }

  // Utility functions
  function readFileAsText(file) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result);
      reader.onerror = reject;
      reader.readAsText(file);
    });
  }

  function readFileAsArrayBuffer(file) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result);
      reader.onerror = reject;
      reader.readAsArrayBuffer(file);
    });
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  function showError(msg) {
    alert(msg); // Simple for now, could be a toast
    // Note: Removed console.error to avoid logging sensitive information
  }

  // Start
  document.addEventListener('DOMContentLoaded', init);
})();
