// ReMemory Bundle Creator - Browser-based bundle creation using Go WASM

(function() {
  'use strict';

  const sampleNames = [
    'Catalina', 'Matthias', 'Sophie', 'Joaquín', 'Emma',
    'Francisca', 'Liam', 'Hannah', 'Sebastián', 'Olivia'
  ];
  let nameIndex = Math.floor(Math.random() * sampleNames.length);

  function getNextSampleName() {
    const name = sampleNames[nameIndex];
    nameIndex = (nameIndex + 1) % sampleNames.length;
    return name;
  }

  // Generate project name from date
  function generateProjectName() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    return `recovery-${year}-${month}-${day}`;
  }

  // State
  const state = {
    projectName: generateProjectName(),
    friends: [],      // Array of {name, email, phone}
    threshold: 2,
    files: [],        // Array of {name, data: Uint8Array}
    bundles: [],      // Array of {friendName, fileName, data: Uint8Array}
    wasmReady: false,
    generating: false,
    generationComplete: false
  };

  // DOM elements
  const elements = {
    loadingOverlay: document.getElementById('loading-overlay'),
    yamlImport: document.getElementById('yaml-import'),
    importBtn: document.getElementById('import-btn'),
    friendsList: document.getElementById('friends-list'),
    addFriendBtn: document.getElementById('add-friend-btn'),
    thresholdSelect: document.getElementById('threshold-select'),
    friendsValidation: document.getElementById('friends-validation'),
    filesDropZone: document.getElementById('files-drop-zone'),
    filesInput: document.getElementById('files-input'),
    folderInput: document.getElementById('folder-input'),
    filesPreview: document.getElementById('files-preview'),
    filesSummary: document.getElementById('files-summary'),
    generateBtn: document.getElementById('generate-btn'),
    progressBar: document.getElementById('progress-bar'),
    statusMessage: document.getElementById('status-message'),
    bundlesList: document.getElementById('bundles-list'),
    downloadAllSection: document.getElementById('download-all-section'),
    downloadAllBtn: document.getElementById('download-all-btn'),
    downloadYamlBtn: document.getElementById('download-yaml-btn')
  };

  // Initialize
  async function init() {
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

  // Wait for WASM to be ready
  async function waitForWasm() {
    return new Promise((resolve) => {
      const check = () => {
        if (window.rememoryReady) {
          state.wasmReady = true;
          elements.loadingOverlay.classList.add('hidden');
          checkGenerateReady();
          resolve();
        } else {
          setTimeout(check, 50);
        }
      };
      check();
    });
  }

  // YAML import handling
  function setupImport() {
    elements.importBtn.addEventListener('click', () => {
      const yaml = elements.yamlImport.value.trim();
      if (!yaml) return;

      if (!state.wasmReady) {
        showError(t('error', 'WASM not ready'));
        return;
      }

      const result = window.rememoryParseProjectYAML(yaml);
      if (result.error) {
        showError(t('import_error', result.error));
        return;
      }

      // Clear existing friends
      state.friends = [];
      elements.friendsList.innerHTML = '';

      // Import friends
      const project = result.project;
      if (project.name) {
        state.projectName = project.name;
      }

      if (project.friends && project.friends.length > 0) {
        project.friends.forEach(f => {
          addFriend(f.name, f.email, f.phone);
        });
      }

      if (project.threshold && project.threshold >= 2) {
        state.threshold = project.threshold;
      }

      updateThresholdOptions();
      elements.yamlImport.value = '';
      showStatus(t('import_success', project.friends ? project.friends.length : 0), 'success');
      checkGenerateReady();
    });
  }

  // Friends management
  function setupFriends() {
    elements.addFriendBtn.addEventListener('click', () => addFriend());

    elements.thresholdSelect.addEventListener('change', () => {
      state.threshold = parseInt(elements.thresholdSelect.value, 10);
    });
  }

  function addFriend(name = '', email = '', phone = '') {
    const index = state.friends.length;
    state.friends.push({ name, email, phone });

    const entry = document.createElement('div');
    entry.className = 'friend-entry';
    entry.dataset.index = index;

    const sampleName = getNextSampleName();
    const sampleEmail = sampleName.toLowerCase() + '@example.com';

    entry.innerHTML = `
      <div class="friend-number">#${index + 1}</div>
      <div class="field">
        <label>${t('name_label')}</label>
        <input type="text" class="friend-name" value="${escapeHtml(name)}" placeholder="${sampleName}">
      </div>
      <div class="field">
        <label>${t('email_label')}</label>
        <input type="email" class="friend-email" value="${escapeHtml(email)}" placeholder="${sampleEmail}">
      </div>
      <div class="field">
        <label>${t('phone_label')}</label>
        <input type="tel" class="friend-phone" value="${escapeHtml(phone)}" placeholder="+1-555-1234">
      </div>
      <button type="button" class="remove-btn" title="${t('remove')}">&times;</button>
    `;

    // Add event listeners
    entry.querySelector('.friend-name').addEventListener('input', (e) => {
      state.friends[index].name = e.target.value.trim();
      checkGenerateReady();
    });

    entry.querySelector('.friend-email').addEventListener('input', (e) => {
      state.friends[index].email = e.target.value.trim();
      checkGenerateReady();
    });

    entry.querySelector('.friend-phone').addEventListener('input', (e) => {
      state.friends[index].phone = e.target.value.trim();
    });

    entry.querySelector('.remove-btn').addEventListener('click', () => {
      removeFriend(index);
    });

    elements.friendsList.appendChild(entry);
    updateThresholdOptions();
    checkGenerateReady();
  }

  function removeFriend(index) {
    if (state.friends.length <= 2) {
      showError(t('validation_min_friends'));
      return;
    }

    state.friends.splice(index, 1);
    renderFriendsList();
    updateThresholdOptions();
    checkGenerateReady();
  }

  function renderFriendsList() {
    elements.friendsList.innerHTML = '';
    const friends = [...state.friends];
    state.friends = [];
    friends.forEach(f => addFriend(f.name, f.email, f.phone));
  }

  function updateThresholdOptions() {
    const n = state.friends.length;
    const current = state.threshold;

    elements.thresholdSelect.innerHTML = '';
    for (let k = 2; k <= n; k++) {
      const option = document.createElement('option');
      option.value = k;
      option.textContent = `${k} of ${n}`;
      elements.thresholdSelect.appendChild(option);
    }

    // Keep current threshold if valid, otherwise default to 2
    if (current >= 2 && current <= n) {
      elements.thresholdSelect.value = current;
      state.threshold = current;
    } else {
      elements.thresholdSelect.value = Math.min(2, n);
      state.threshold = Math.min(2, n);
    }
  }

  // Files handling
  function setupFiles() {
    // Click to open file dialog
    elements.filesDropZone.addEventListener('click', (e) => {
      // Check if browser supports directory selection
      if ('webkitdirectory' in elements.folderInput) {
        // Ask user preference (simplified: just use folder input)
        elements.folderInput.click();
      } else {
        elements.filesInput.click();
      }
    });

    // Drag and drop
    elements.filesDropZone.addEventListener('dragover', (e) => {
      e.preventDefault();
      elements.filesDropZone.classList.add('dragover');
    });

    elements.filesDropZone.addEventListener('dragleave', () => {
      elements.filesDropZone.classList.remove('dragover');
    });

    elements.filesDropZone.addEventListener('drop', async (e) => {
      e.preventDefault();
      elements.filesDropZone.classList.remove('dragover');

      // Handle dropped items
      const items = e.dataTransfer.items;
      if (items && items.length > 0) {
        const files = [];
        for (let i = 0; i < items.length; i++) {
          const item = items[i];
          if (item.kind === 'file') {
            const entry = item.webkitGetAsEntry ? item.webkitGetAsEntry() : null;
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

    // File input change
    elements.filesInput.addEventListener('change', async (e) => {
      const fileList = Array.from(e.target.files);
      const files = fileList.map(f => ({ file: f, path: f.name }));
      e.target.value = '';
      await loadFiles(files);
    });

    // Folder input change
    elements.folderInput.addEventListener('change', async (e) => {
      const fileList = Array.from(e.target.files);
      const files = fileList.map(f => ({
        file: f,
        path: f.webkitRelativePath || f.name
      }));
      e.target.value = '';
      await loadFiles(files);
    });
  }

  // Traverse file tree for drag-and-drop folders
  async function traverseFileTree(entry, basePath, files) {
    if (entry.isFile) {
      const file = await new Promise((resolve) => entry.file(resolve));
      const path = basePath ? `${basePath}/${entry.name}` : entry.name;
      files.push({ file, path });
    } else if (entry.isDirectory) {
      const dirReader = entry.createReader();
      const entries = await new Promise((resolve) => {
        dirReader.readEntries(resolve);
      });
      const newBasePath = basePath ? `${basePath}/${entry.name}` : entry.name;
      for (const childEntry of entries) {
        await traverseFileTree(childEntry, newBasePath, files);
      }
    }
  }

  // Load files into state (appends to existing files)
  async function loadFiles(filesWithPaths) {
    // Get existing file paths to avoid duplicates
    const existingPaths = new Set(state.files.map(f => f.name));

    for (const { file, path } of filesWithPaths) {
      // Skip hidden files and directories
      if (path.split('/').some(part => part.startsWith('.'))) {
        continue;
      }

      // Skip if file with same path already exists
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

  function renderFilesPreview() {
    if (state.files.length === 0) {
      elements.filesPreview.classList.add('hidden');
      elements.filesSummary.classList.add('hidden');
      return;
    }

    elements.filesPreview.innerHTML = '';
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
      item.querySelector('.file-remove-btn').addEventListener('click', () => {
        removeFile(index);
      });
      elements.filesPreview.appendChild(item);
    });

    elements.filesPreview.classList.remove('hidden');
    elements.filesSummary.textContent = t('files_summary', state.files.length, formatSize(totalSize));
    elements.filesSummary.classList.remove('hidden');
  }

  function removeFile(index) {
    state.files.splice(index, 1);
    renderFilesPreview();
    checkGenerateReady();
  }

  // Generate bundles
  function setupGenerate() {
    elements.generateBtn.addEventListener('click', generateBundles);
    elements.downloadAllBtn.addEventListener('click', downloadAllBundles);
    elements.downloadYamlBtn.addEventListener('click', downloadProjectYaml);
  }

  function checkGenerateReady() {
    const valid = validateInputs(true); // Silent validation
    elements.generateBtn.disabled = !valid || !state.wasmReady || state.generating;
  }

  function validateInputs(silent = false) {
    let valid = true;
    let errors = [];

    // Friends
    if (state.friends.length < 2) {
      valid = false;
      if (!silent) errors.push(t('validation_min_friends'));
    } else {
      state.friends.forEach((f, i) => {
        if (!f.name) {
          valid = false;
          if (!silent) errors.push(t('validation_friend_name', i + 1));
        }
        if (!f.email) {
          valid = false;
          if (!silent) errors.push(t('validation_friend_email', i + 1, f.name || '?'));
        }
      });
    }

    // Files
    if (state.files.length === 0) {
      valid = false;
      if (!silent) errors.push(t('validation_no_files'));
    }

    if (!silent && errors.length > 0) {
      elements.friendsValidation.textContent = errors.join('. ');
      elements.friendsValidation.classList.remove('hidden');
    } else {
      elements.friendsValidation.classList.add('hidden');
    }

    return valid;
  }

  async function generateBundles() {
    if (!validateInputs(false)) return;
    if (!state.wasmReady) return;
    if (state.generating) return;

    state.generating = true;
    state.generationComplete = false;
    state.bundles = [];

    elements.generateBtn.disabled = true;
    elements.progressBar.classList.remove('hidden');
    elements.bundlesList.classList.add('hidden');
    elements.downloadAllSection.classList.add('hidden');
    elements.statusMessage.className = 'status-message';

    try {
      setProgress(0);
      setStatus(t('generating'));

      // Prepare files for WASM
      const filesForWasm = state.files.map(f => ({
        name: f.name,
        data: f.data
      }));

      // Call WASM function
      const config = {
        projectName: state.projectName,
        threshold: state.threshold,
        friends: state.friends.map(f => ({
          name: f.name,
          email: f.email,
          phone: f.phone || ''
        })),
        files: filesForWasm,
        version: VERSION || 'dev',
        githubURL: GITHUB_URL || 'https://github.com/eljojo/rememory'
      };

      setProgress(10);
      setStatus(t('archiving'));
      await sleep(100); // Allow UI to update

      setProgress(30);
      setStatus(t('encrypting'));
      await sleep(100);

      setProgress(50);
      setStatus(t('splitting'));
      await sleep(100);

      // The actual bundle creation happens here
      const result = window.rememoryCreateBundles(config);

      if (result.error) {
        throw new Error(result.error);
      }

      setProgress(80);

      // Store bundles
      state.bundles = result.bundles;

      // Expose bundles for testing
      window.rememoryBundles = result.bundles;

      // Render bundle list
      renderBundlesList();

      setProgress(100);
      setStatus(t('complete'), 'success');
      state.generationComplete = true;

      elements.bundlesList.classList.remove('hidden');
      elements.downloadAllSection.classList.remove('hidden');

    } catch (err) {
      setStatus(t('error', err.message), 'error');
    } finally {
      state.generating = false;
      elements.generateBtn.disabled = false;
    }
  }

  function renderBundlesList() {
    elements.bundlesList.innerHTML = '';

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

      item.querySelector('.download-btn').addEventListener('click', () => {
        downloadBundle(bundle);
      });

      elements.bundlesList.appendChild(item);
    });
  }

  function downloadBundle(bundle) {
    const blob = new Blob([bundle.data], { type: 'application/zip' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = bundle.fileName;
    a.click();
    URL.revokeObjectURL(url);
  }

  function downloadAllBundles() {
    // Download each bundle individually
    state.bundles.forEach((bundle, index) => {
      setTimeout(() => downloadBundle(bundle), index * 500);
    });
  }

  function downloadProjectYaml() {
    // Generate YAML content
    let yaml = `# ReMemory Project Configuration\n`;
    yaml += `# Generated: ${new Date().toISOString()}\n`;
    yaml += `# Import this file to quickly restore your friend list\n\n`;
    yaml += `name: ${state.projectName}\n`;
    yaml += `threshold: ${state.threshold}\n`;
    yaml += `friends:\n`;

    state.friends.forEach(f => {
      yaml += `  - name: ${f.name}\n`;
      yaml += `    email: ${f.email}\n`;
      if (f.phone) {
        yaml += `    phone: "${f.phone}"\n`;
      }
    });

    // Download as file
    const blob = new Blob([yaml], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'project.yml';
    a.click();
    URL.revokeObjectURL(url);
  }

  // UI helpers
  function setProgress(percent) {
    const fill = elements.progressBar.querySelector('.fill');
    fill.style.width = percent + '%';
  }

  function setStatus(msg, type) {
    elements.statusMessage.textContent = msg;
    elements.statusMessage.className = 'status-message' + (type ? ' ' + type : '');
  }

  function showStatus(msg, type) {
    setStatus(msg, type);
    // Auto-clear after 3 seconds for success messages
    if (type === 'success') {
      setTimeout(() => {
        if (elements.statusMessage.textContent === msg) {
          elements.statusMessage.textContent = '';
        }
      }, 3000);
    }
  }

  function showError(msg) {
    alert(msg);
  }

  // Utility functions
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

  function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Expose function to re-render UI when language changes
  window.rememoryUpdateUI = function() {
    renderFriendsList();
    renderFilesPreview();
    if (state.generationComplete) {
      renderBundlesList();
    }
  };

  // Start
  document.addEventListener('DOMContentLoaded', init);
})();
