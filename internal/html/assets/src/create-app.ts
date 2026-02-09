// ReMemory Bundle Creator - Browser-based bundle creation using Go WASM

import type {
  CreationState,
  BundleFile,
  GeneratedBundle,
  TranslationFunction
} from './types';

// Translation function (defined in HTML)
declare const t: TranslationFunction;

(function() {
  'use strict';

  // Import shared utilities
  const { escapeHtml, formatSize, toast } = window.rememoryUtils;

  // Sample names for placeholders
  const sampleNames = [
    'Catalina', 'Matthias', 'Sophie', 'Joaquín', 'Emma',
    'Francisca', 'Liam', 'Hannah', 'Sebastián', 'Olivia'
  ];
  let nameIndex = Math.floor(Math.random() * sampleNames.length);

  function getNextSampleName(): string {
    const name = sampleNames[nameIndex];
    nameIndex = (nameIndex + 1) % sampleNames.length;
    return name;
  }

  function generateProjectName(): string {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    return `recovery-${year}-${month}-${day}`;
  }

  // Escape special characters in YAML double-quoted strings
  function escapeYamlString(str: string): string {
    return str
      .replace(/\\/g, '\\\\')  // Backslash must be escaped first
      .replace(/"/g, '\\"')     // Escape double quotes
      .replace(/\n/g, '\\n')    // Escape newlines
      .replace(/\r/g, '\\r')    // Escape carriage returns
      .replace(/\t/g, '\\t');   // Escape tabs
  }

  // State
  const state: CreationState & { anonymous: boolean; numShares: number } = {
    projectName: generateProjectName(),
    friends: [],
    threshold: 2,
    files: [],
    bundles: [],
    wasmReady: false,
    generating: false,
    generationComplete: false,
    anonymous: false,
    numShares: 5
  };

  // DOM elements interface
  interface Elements {
    loadingOverlay: HTMLElement | null;
    anonymousMode: HTMLInputElement | null;
    friendsHint: HTMLElement | null;
    sharesInput: HTMLElement | null;
    numShares: HTMLInputElement | null;
    friendsSection: HTMLElement | null;
    importSection: HTMLElement | null;
    yamlImport: HTMLTextAreaElement | null;
    importBtn: HTMLButtonElement | null;
    friendsList: HTMLElement | null;
    addFriendBtn: HTMLButtonElement | null;
    thresholdSelect: HTMLSelectElement | null;
    friendsValidation: HTMLElement | null;
    filesDropZone: HTMLElement | null;
    filesInput: HTMLInputElement | null;
    folderInput: HTMLInputElement | null;
    filesPreview: HTMLElement | null;
    filesSummary: HTMLElement | null;
    generateBtn: HTMLButtonElement | null;
    progressBar: HTMLElement | null;
    statusMessage: HTMLElement | null;
    bundlesList: HTMLElement | null;
    downloadAllSection: HTMLElement | null;
    downloadAllBtn: HTMLButtonElement | null;
    downloadYamlBtn: HTMLButtonElement | null;
  }

  // DOM elements
  const elements: Elements = {
    loadingOverlay: document.getElementById('loading-overlay'),
    anonymousMode: document.getElementById('anonymous-mode') as HTMLInputElement | null,
    friendsHint: document.getElementById('friends-hint'),
    sharesInput: document.getElementById('shares-input'),
    numShares: document.getElementById('num-shares') as HTMLInputElement | null,
    friendsSection: document.getElementById('friends-section'),
    importSection: document.getElementById('import-section'),
    yamlImport: document.getElementById('yaml-import') as HTMLTextAreaElement | null,
    importBtn: document.getElementById('import-btn') as HTMLButtonElement | null,
    friendsList: document.getElementById('friends-list'),
    addFriendBtn: document.getElementById('add-friend-btn') as HTMLButtonElement | null,
    thresholdSelect: document.getElementById('threshold-select') as HTMLSelectElement | null,
    friendsValidation: document.getElementById('friends-validation'),
    filesDropZone: document.getElementById('files-drop-zone'),
    filesInput: document.getElementById('files-input') as HTMLInputElement | null,
    folderInput: document.getElementById('folder-input') as HTMLInputElement | null,
    filesPreview: document.getElementById('files-preview'),
    filesSummary: document.getElementById('files-summary'),
    generateBtn: document.getElementById('generate-btn') as HTMLButtonElement | null,
    progressBar: document.getElementById('progress-bar'),
    statusMessage: document.getElementById('status-message'),
    bundlesList: document.getElementById('bundles-list'),
    downloadAllSection: document.getElementById('download-all-section'),
    downloadAllBtn: document.getElementById('download-all-btn') as HTMLButtonElement | null,
    downloadYamlBtn: document.getElementById('download-yaml-btn') as HTMLButtonElement | null
  };

  // ============================================
  // Error Display
  // ============================================

  function showError(msg: string, options: { title?: string; guidance?: string } = {}): void {
    const { title, guidance } = options;
    toast.error(title || t('error_title'), msg, guidance);
  }

  // ============================================
  // Initialization
  // ============================================

  async function init(): Promise<void> {
    setupAnonymousMode();
    setupImport();
    setupFriends();
    setupFiles();
    setupGenerate();

    // Add initial 2 friends
    addFriend();
    addFriend();
    updateThresholdOptions();

    await waitForWasm();
  }

  // ============================================
  // Anonymous Mode
  // ============================================

  function setupAnonymousMode(): void {
    elements.anonymousMode?.addEventListener('change', () => {
      state.anonymous = elements.anonymousMode?.checked || false;
      updateAnonymousModeUI();
      updateThresholdOptions();
      checkGenerateReady();
    });

    elements.numShares?.addEventListener('input', () => {
      const value = parseInt(elements.numShares?.value || '5', 10);
      state.numShares = Math.max(2, Math.min(20, value));
      updateThresholdOptions();
      checkGenerateReady();
    });
  }

  function updateAnonymousModeUI(): void {
    if (state.anonymous) {
      // Hide friends list and show shares input
      elements.friendsSection?.classList.add('hidden');
      elements.sharesInput?.classList.remove('hidden');
      elements.importSection?.classList.add('hidden');
      if (elements.friendsHint) {
        elements.friendsHint.textContent = t('anonymous_hint');
      }
    } else {
      // Show friends list and hide shares input
      elements.friendsSection?.classList.remove('hidden');
      elements.sharesInput?.classList.add('hidden');
      elements.importSection?.classList.remove('hidden');
      if (elements.friendsHint) {
        elements.friendsHint.textContent = t('friends_hint');
      }
    }
  }

  async function waitForWasm(): Promise<void> {
    return new Promise((resolve) => {
      const check = (): void => {
        if (window.rememoryReady) {
          state.wasmReady = true;
          elements.loadingOverlay?.classList.add('hidden');
          checkGenerateReady();
          resolve();
        } else {
          setTimeout(check, 50);
        }
      };
      check();
    });
  }

  // ============================================
  // YAML Import
  // ============================================

  function setupImport(): void {
    elements.importBtn?.addEventListener('click', () => {
      const yaml = elements.yamlImport?.value.trim();
      if (!yaml) return;

      if (!state.wasmReady) {
        toast.warning(t('error_not_ready_title'), t('error_not_ready_message'), t('error_not_ready_guidance'));
        return;
      }

      const result = window.rememoryParseProjectYAML(yaml);
      if (result.error || !result.project) {
        showError(
          t('import_error', result.error || 'Unknown error'),
          {
            title: t('error_import_title'),
            guidance: t('error_import_guidance')
          }
        );
        return;
      }

      // Clear existing friends
      state.friends = [];
      if (elements.friendsList) elements.friendsList.innerHTML = '';

      // Import friends
      const project = result.project;
      if (project.name) {
        state.projectName = project.name;
      }

      if (project.friends && project.friends.length > 0) {
        project.friends.forEach(f => {
          addFriend(f.name, f.contact || '');
        });
      }

      if (project.threshold && project.threshold >= 2) {
        state.threshold = project.threshold;
      }

      updateThresholdOptions();
      if (elements.yamlImport) elements.yamlImport.value = '';
      showStatus(t('import_success', project.friends ? project.friends.length : 0), 'success');
      checkGenerateReady();
    });
  }

  // ============================================
  // Friends Management
  // ============================================

  function setupFriends(): void {
    elements.addFriendBtn?.addEventListener('click', () => addFriend());

    elements.thresholdSelect?.addEventListener('change', () => {
      state.threshold = parseInt(elements.thresholdSelect?.value || '2', 10);
    });
  }

  function addFriend(name = '', contact = ''): void {
    const index = state.friends.length;
    state.friends.push({ name, contact });

    const entry = document.createElement('div');
    entry.className = 'friend-entry';
    entry.dataset.index = String(index);

    const sampleName = getNextSampleName();
    const sampleContact = sampleName.toLowerCase() + '@example.com';

    entry.innerHTML = `
      <div class="friend-number">#${index + 1}</div>
      <div class="field">
        <label class="required">${t('name_label')}</label>
        <input type="text" class="friend-name" value="${escapeHtml(name)}" placeholder="${sampleName}" required>
      </div>
      <div class="field">
        <label>${t('contact_label')}</label>
        <input type="text" class="friend-contact" value="${escapeHtml(contact)}" placeholder="${sampleContact}">
      </div>
      <button type="button" class="remove-btn" title="${t('remove')}">&times;</button>
    `;

    // Add event listeners
    const nameInput = entry.querySelector('.friend-name') as HTMLInputElement;
    nameInput?.addEventListener('input', (e) => {
      const target = e.target as HTMLInputElement;
      state.friends[index].name = target.value.trim();
      target.classList.remove('input-error');
      checkGenerateReady();
    });

    const contactInput = entry.querySelector('.friend-contact') as HTMLInputElement;
    contactInput?.addEventListener('input', (e) => {
      const target = e.target as HTMLInputElement;
      state.friends[index].contact = target.value.trim();
    });

    const removeBtn = entry.querySelector('.remove-btn');
    removeBtn?.addEventListener('click', () => {
      removeFriend(index);
    });

    elements.friendsList?.appendChild(entry);
    updateThresholdOptions();
    checkGenerateReady();
  }

  function removeFriend(index: number): void {
    if (state.friends.length <= 2) {
      toast.warning(
        t('error_min_friends_title'),
        t('validation_min_friends'),
        t('error_min_friends_guidance')
      );
      return;
    }

    state.friends.splice(index, 1);
    renderFriendsList();
    updateThresholdOptions();
    checkGenerateReady();
  }

  function renderFriendsList(): void {
    if (elements.friendsList) elements.friendsList.innerHTML = '';
    const friends = [...state.friends];
    state.friends = [];
    friends.forEach(f => addFriend(f.name, f.contact || ''));
  }

  function updateThresholdOptions(): void {
    const n = state.anonymous ? state.numShares : state.friends.length;
    const current = state.threshold;

    if (elements.thresholdSelect) {
      elements.thresholdSelect.innerHTML = '';
      for (let k = 2; k <= n; k++) {
        const option = document.createElement('option');
        option.value = String(k);
        option.textContent = `${k} of ${n}`;
        elements.thresholdSelect.appendChild(option);
      }

      if (current >= 2 && current <= n) {
        elements.thresholdSelect.value = String(current);
        state.threshold = current;
      } else {
        elements.thresholdSelect.value = String(Math.min(2, n));
        state.threshold = Math.min(2, n);
      }
    }
  }

  // ============================================
  // Files Handling
  // ============================================

  function setupFiles(): void {
    elements.filesDropZone?.addEventListener('click', () => {
      if (elements.folderInput && 'webkitdirectory' in elements.folderInput) {
        elements.folderInput.click();
      } else {
        elements.filesInput?.click();
      }
    });

    elements.filesDropZone?.addEventListener('dragover', (e) => {
      e.preventDefault();
      elements.filesDropZone?.classList.add('dragover');
    });

    elements.filesDropZone?.addEventListener('dragleave', () => {
      elements.filesDropZone?.classList.remove('dragover');
    });

    elements.filesDropZone?.addEventListener('drop', async (e) => {
      e.preventDefault();
      elements.filesDropZone?.classList.remove('dragover');

      const items = e.dataTransfer?.items;
      if (items && items.length > 0) {
        const files: { file: File; path: string }[] = [];
        for (let i = 0; i < items.length; i++) {
          const item = items[i];
          if (item.kind === 'file') {
            const entry = item.webkitGetAsEntry?.();
            if (entry) {
              await traverseFileTree(entry, '', files);
            } else {
              const file = item.getAsFile();
              if (file) {
                files.push({ file, path: file.name });
              }
            }
          }
        }
        await loadFiles(files);
      }
    });

    elements.filesInput?.addEventListener('change', async (e) => {
      const target = e.target as HTMLInputElement;
      const fileList = Array.from(target.files || []);
      const files = fileList.map(f => ({ file: f, path: f.name }));
      target.value = '';
      await loadFiles(files);
    });

    elements.folderInput?.addEventListener('change', async (e) => {
      const target = e.target as HTMLInputElement;
      const fileList = Array.from(target.files || []);
      const files = fileList.map(f => ({
        file: f,
        path: (f as File & { webkitRelativePath?: string }).webkitRelativePath || f.name
      }));
      target.value = '';
      await loadFiles(files);
    });
  }

  async function traverseFileTree(
    entry: FileSystemEntry,
    basePath: string,
    files: { file: File; path: string }[]
  ): Promise<void> {
    if (entry.isFile) {
      const fileEntry = entry as FileSystemFileEntry;
      const file = await new Promise<File>((resolve) => fileEntry.file(resolve));
      const path = basePath ? `${basePath}/${entry.name}` : entry.name;
      files.push({ file, path });
    } else if (entry.isDirectory) {
      const dirEntry = entry as FileSystemDirectoryEntry;
      const dirReader = dirEntry.createReader();
      const entries = await new Promise<FileSystemEntry[]>((resolve) => {
        dirReader.readEntries(resolve);
      });
      const newBasePath = basePath ? `${basePath}/${entry.name}` : entry.name;
      for (const childEntry of entries) {
        await traverseFileTree(childEntry, newBasePath, files);
      }
    }
  }

  async function loadFiles(filesWithPaths: { file: File; path: string }[]): Promise<void> {
    // Clear any file-related errors
    elements.filesDropZone?.classList.remove('has-error');
    const existingFilesError = elements.filesDropZone?.parentNode?.querySelector('.inline-error');
    existingFilesError?.remove();

    const existingPaths = new Set(state.files.map(f => f.name));

    for (const { file, path } of filesWithPaths) {
      // Skip hidden files
      if (path.split('/').some(part => part.startsWith('.'))) {
        continue;
      }

      // Skip duplicates
      if (existingPaths.has(path)) {
        continue;
      }

      const buffer = await readFileAsArrayBuffer(file);
      state.files.push({
        name: path,
        data: new Uint8Array(buffer)
      });
      existingPaths.add(path);
    }

    renderFilesPreview();
    checkGenerateReady();
  }

  function renderFilesPreview(): void {
    if (state.files.length === 0) {
      elements.filesPreview?.classList.add('hidden');
      elements.filesSummary?.classList.add('hidden');
      return;
    }

    if (elements.filesPreview) elements.filesPreview.innerHTML = '';
    let totalSize = 0;

    state.files.forEach((file, index) => {
      totalSize += file.data.length;
      const item = document.createElement('div');
      item.className = 'file-item';
      item.innerHTML = `
        <span class="icon">&#128196;</span>
        <span class="name">${escapeHtml(file.name)}</span>
        <span class="size">${formatSize(file.data.length)}</span>
        <button type="button" class="file-remove-btn" data-index="${index}" title="${t('remove')}">&times;</button>
      `;
      const removeBtn = item.querySelector('.file-remove-btn');
      removeBtn?.addEventListener('click', () => {
        removeFile(index);
      });
      elements.filesPreview?.appendChild(item);
    });

    elements.filesPreview?.classList.remove('hidden');
    if (elements.filesSummary) {
      elements.filesSummary.textContent = t('files_summary', state.files.length, formatSize(totalSize));
    }
    elements.filesSummary?.classList.remove('hidden');
  }

  function removeFile(index: number): void {
    state.files.splice(index, 1);
    renderFilesPreview();
    checkGenerateReady();
  }

  // ============================================
  // Bundle Generation
  // ============================================

  function setupGenerate(): void {
    elements.generateBtn?.addEventListener('click', generateBundles);
    elements.downloadAllBtn?.addEventListener('click', downloadAllBundles);
    elements.downloadYamlBtn?.addEventListener('click', downloadProjectYaml);
  }

  function checkGenerateReady(): void {
    if (elements.generateBtn) {
      elements.generateBtn.disabled = !state.wasmReady || state.generating;
    }
  }

  interface ValidationResult {
    valid: boolean;
    errors: string[];
    firstInvalidElement: HTMLElement | null;
  }

  function validateInputs(silent = false): boolean {
    const result: ValidationResult = {
      valid: true,
      errors: [],
      firstInvalidElement: null
    };

    // Clear previous inline errors
    document.querySelectorAll('.friend-entry').forEach(entry => {
      entry.querySelectorAll('input').forEach(input => {
        input.classList.remove('input-error');
      });
      const existingError = entry.querySelector('.field-error');
      existingError?.remove();
    });
    elements.filesDropZone?.classList.remove('has-error');
    const existingFilesError = elements.filesDropZone?.parentNode?.querySelector('.inline-error');
    existingFilesError?.remove();

    // Friends validation (skip for anonymous mode)
    if (state.anonymous) {
      if (state.numShares < 2) {
        result.valid = false;
        if (!silent) result.errors.push(t('validation_min_friends'));
      }
    } else {
      if (state.friends.length < 2) {
        result.valid = false;
        if (!silent) result.errors.push(t('validation_min_friends'));
      } else {
        state.friends.forEach((f, i) => {
          const entry = elements.friendsList?.children[i] as HTMLElement | undefined;
          if (!entry) return;

          if (!f.name) {
            result.valid = false;
            if (!silent) {
              result.errors.push(t('validation_friend_name', i + 1));
              const nameInput = entry.querySelector('.friend-name') as HTMLInputElement;
              nameInput?.classList.add('input-error');
              if (!result.firstInvalidElement) result.firstInvalidElement = nameInput;
            }
          }
        });
      }
    }

    // Files validation
    if (state.files.length === 0) {
      result.valid = false;
      if (!silent) {
        result.errors.push(t('validation_no_files'));
        elements.filesDropZone?.classList.add('has-error');
        if (!result.firstInvalidElement && elements.filesDropZone) {
          result.firstInvalidElement = elements.filesDropZone;
        }
      }
    }

    if (!silent && result.errors.length > 0) {
      if (elements.friendsValidation) {
        elements.friendsValidation.textContent = result.errors.join('. ');
        elements.friendsValidation.classList.remove('hidden');
      }

      // Focus first invalid element
      if (result.firstInvalidElement) {
        result.firstInvalidElement.focus();
        result.firstInvalidElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }

      // Show validation toast
      toast.warning(
        t('validation_title'),
        t('validation_message'),
        t('validation_guidance')
      );
    } else {
      elements.friendsValidation?.classList.add('hidden');
    }

    return result.valid;
  }

  async function generateBundles(): Promise<void> {
    if (!validateInputs(false)) return;
    if (!state.wasmReady) return;
    if (state.generating) return;

    state.generating = true;
    state.generationComplete = false;
    state.bundles = [];

    if (elements.generateBtn) elements.generateBtn.disabled = true;
    elements.progressBar?.classList.remove('hidden');
    elements.bundlesList?.classList.add('hidden');
    elements.downloadAllSection?.classList.add('hidden');
    if (elements.statusMessage) elements.statusMessage.className = 'status-message';

    try {
      setProgress(0);
      setStatus(t('generating'));

      const filesForWasm: BundleFile[] = state.files.map(f => ({
        name: f.name,
        data: f.data
      }));

      // Create friends array - synthetic names for anonymous mode
      let friends;
      if (state.anonymous) {
        friends = [];
        for (let i = 0; i < state.numShares; i++) {
          friends.push({
            name: `Share ${i + 1}`,
            contact: ''
          });
        }
      } else {
        friends = state.friends.map(f => ({
          name: f.name,
          contact: f.contact || ''
        }));
      }

      const config = {
        projectName: state.projectName,
        threshold: state.threshold,
        friends: friends,
        files: filesForWasm,
        version: window.VERSION || 'dev',
        githubURL: window.GITHUB_URL || 'https://github.com/eljojo/rememory',
        anonymous: state.anonymous
      };

      setProgress(10);
      setStatus(t('archiving'));
      await sleep(100);

      setProgress(30);
      setStatus(t('encrypting'));
      await sleep(100);

      setProgress(50);
      setStatus(t('splitting'));
      await sleep(100);

      const result = window.rememoryCreateBundles(config);

      if (result.error || !result.bundles) {
        throw new Error(result.error || 'Failed to create bundles');
      }

      setProgress(80);

      state.bundles = result.bundles;

      // Expose bundles for testing
      (window as unknown as { rememoryBundles?: GeneratedBundle[] }).rememoryBundles = result.bundles;

      renderBundlesList();

      setProgress(100);
      setStatus(t('complete'), 'success');
      state.generationComplete = true;

      elements.bundlesList?.classList.remove('hidden');
      elements.downloadAllSection?.classList.remove('hidden');

    } catch (err) {
      const errorMsg = (err instanceof Error) ? err.message : String(err);
      setStatus(t('error', errorMsg), 'error');

      toast.error(
        t('error_generate_title'),
        errorMsg,
        t('error_generate_guidance'),
        [
          { id: 'retry', label: t('action_try_again'), primary: true, onClick: () => generateBundles() }
        ]
      );
    } finally {
      state.generating = false;
      if (elements.generateBtn) elements.generateBtn.disabled = false;
    }
  }

  function renderBundlesList(): void {
    if (elements.bundlesList) elements.bundlesList.innerHTML = '';

    state.bundles.forEach((bundle, index) => {
      const item = document.createElement('div');
      item.className = 'bundle-item ready';
      item.innerHTML = `
        <span class="icon">&#128230;</span>
        <div class="details">
          <div class="name">${t('bundle_for', escapeHtml(bundle.friendName))}</div>
          <div class="meta">${escapeHtml(bundle.fileName)} (${formatSize(bundle.data.length)})</div>
        </div>
        <button type="button" class="download-btn" data-index="${index}">${t('download')}</button>
      `;

      const downloadBtn = item.querySelector('.download-btn');
      downloadBtn?.addEventListener('click', () => {
        downloadBundle(bundle);
      });

      elements.bundlesList?.appendChild(item);
    });
  }

  function downloadBundle(bundle: GeneratedBundle): void {
    const blob = new Blob([bundle.data as BlobPart], { type: 'application/zip' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = bundle.fileName;
    a.click();
    URL.revokeObjectURL(url);
  }

  function downloadAllBundles(): void {
    state.bundles.forEach((bundle, index) => {
      setTimeout(() => downloadBundle(bundle), index * 500);
    });
  }

  function downloadProjectYaml(): void {
    let yaml = `# ReMemory Project Configuration\n`;
    yaml += `# Generated: ${new Date().toISOString()}\n`;
    yaml += `# Import this file to quickly restore your friend list\n\n`;
    yaml += `name: ${state.projectName}\n`;
    yaml += `threshold: ${state.threshold}\n`;
    if (state.anonymous) {
      yaml += `anonymous: true\n`;
    }
    yaml += `friends:\n`;

    if (state.anonymous) {
      for (let i = 0; i < state.numShares; i++) {
        yaml += `  - name: Share ${i + 1}\n`;
      }
    } else {
      state.friends.forEach(f => {
        yaml += `  - name: "${escapeYamlString(f.name)}"\n`;
        if (f.contact) {
          yaml += `    contact: "${escapeYamlString(f.contact)}"\n`;
        }
      });
    }

    const blob = new Blob([yaml], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'project.yml';
    a.click();
    URL.revokeObjectURL(url);
  }

  // ============================================
  // UI Helpers
  // ============================================

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

  function showStatus(msg: string, type?: string): void {
    setStatus(msg, type);
    if (type === 'success') {
      setTimeout(() => {
        if (elements.statusMessage?.textContent === msg) {
          elements.statusMessage.textContent = '';
        }
      }, 3000);
    }
  }

  // ============================================
  // Utility Functions
  // ============================================

  function readFileAsArrayBuffer(file: File): Promise<ArrayBuffer> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result as ArrayBuffer);
      reader.onerror = reject;
      reader.readAsArrayBuffer(file);
    });
  }

  function sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // ============================================
  // Global Exports
  // ============================================

  window.rememoryUpdateUI = function(): void {
    renderFriendsList();
    renderFilesPreview();
    if (state.generationComplete) {
      renderBundlesList();
    }
  };

  // Start
  document.addEventListener('DOMContentLoaded', init);
})();
