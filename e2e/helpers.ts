import { Page, expect } from '@playwright/test';
import { execSync } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import AdmZip from 'adm-zip';

// Get absolute path to rememory binary
export function getRememoryBin(): string {
  const binEnv = process.env.REMEMORY_BIN || './rememory';
  return path.resolve(binEnv);
}

// Generate standalone HTML file for testing
export function generateStandaloneHTML(tmpDir: string, type: 'recover' | 'create'): string {
  const bin = getRememoryBin();
  const htmlPath = path.join(tmpDir, type === 'create' ? 'maker.html' : 'recover.html');

  execSync(`${bin} html ${type} -o ${htmlPath}`, { stdio: 'inherit' });

  return htmlPath;
}

// Create a sealed test project with bundles
export function createTestProject(): string {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-e2e-'));
  const projectDir = path.join(tmpDir, 'test-project');
  const bin = getRememoryBin();

  // Create project with 3 friends, threshold 2
  execSync(`${bin} init ${projectDir} --name "E2E Test" --threshold 2 --friend "Alice,alice@test.com" --friend "Bob,bob@test.com" --friend "Carol,carol@test.com"`, {
    stdio: 'inherit'
  });

  // Add secret content
  const manifestDir = path.join(projectDir, 'manifest');
  fs.writeFileSync(path.join(manifestDir, 'secret.txt'), 'The secret password is: correct-horse-battery-staple');
  fs.writeFileSync(path.join(manifestDir, 'notes.txt'), 'Remember to feed the cat!');

  // Seal and generate bundles
  execSync(`${bin} seal`, { cwd: projectDir, stdio: 'inherit' });
  execSync(`${bin} bundle`, { cwd: projectDir, stdio: 'inherit' });

  return projectDir;
}

// Create a sealed anonymous test project with bundles
export function createAnonymousTestProject(): string {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-e2e-anon-'));
  const projectDir = path.join(tmpDir, 'test-anon-project');
  const bin = getRememoryBin();

  // Create anonymous project with 3 shares, threshold 2
  execSync(`${bin} init ${projectDir} --name "Anonymous E2E Test" --anonymous --shares 3 --threshold 2`, {
    stdio: 'inherit'
  });

  // Add secret content
  const manifestDir = path.join(projectDir, 'manifest');
  fs.writeFileSync(path.join(manifestDir, 'secret.txt'), 'Anonymous secret: correct-horse-battery-staple');
  fs.writeFileSync(path.join(manifestDir, 'notes.txt'), 'Anonymous notes!');

  // Seal and generate bundles
  execSync(`${bin} seal`, { cwd: projectDir, stdio: 'inherit' });
  execSync(`${bin} bundle`, { cwd: projectDir, stdio: 'inherit' });

  return projectDir;
}

// Extract a bundle ZIP and return the extracted directory path
// Note: friendName is case-insensitive, bundle files are lowercase
export function extractBundle(bundlesDir: string, friendName: string): string {
  const lowerName = friendName.toLowerCase();
  const bundleZip = path.join(bundlesDir, `bundle-${lowerName}.zip`);
  const extractDir = path.join(bundlesDir, `bundle-${lowerName}`);

  fs.mkdirSync(extractDir, { recursive: true });

  // Use adm-zip for cross-platform extraction
  const zip = new AdmZip(bundleZip);
  zip.extractAllTo(extractDir, true);

  return extractDir;
}

// Extract multiple bundles
export function extractBundles(bundlesDir: string, friendNames: string[]): string[] {
  return friendNames.map(name => extractBundle(bundlesDir, name));
}

// Extract anonymous bundle by share number
export function extractAnonymousBundle(bundlesDir: string, shareNum: number): string {
  return extractBundle(bundlesDir, `share-${shareNum}`);
}

// Extract multiple anonymous bundles
export function extractAnonymousBundles(bundlesDir: string, shareNums: number[]): string[] {
  return shareNums.map(num => extractAnonymousBundle(bundlesDir, num));
}

// Page helper class for recovery tool interactions
export class RecoveryPage {
  constructor(private page: Page, private bundleDir: string) {}

  // Navigate to recover.html and wait for WASM
  async open(): Promise<void> {
    await this.page.goto(`file://${path.join(this.bundleDir, 'recover.html')}`);
    await this.page.waitForFunction(
      () => (window as any).rememoryAppReady === true,
      { timeout: 30000 }
    );
  }

  // Add shares from README.txt files
  async addShares(...bundleDirs: string[]): Promise<void> {
    const readmePaths = bundleDirs.map(dir => path.join(dir, 'README.txt'));
    await this.page.locator('#share-file-input').setInputFiles(readmePaths);
  }

  // Add manifest file
  async addManifest(bundleDir?: string): Promise<void> {
    const dir = bundleDir || this.bundleDir;
    await this.page.locator('#manifest-file-input').setInputFiles(
      path.join(dir, 'MANIFEST.age')
    );
  }

  // Click recover button
  async recover(): Promise<void> {
    await this.page.locator('#recover-btn').click();
  }

  // Assertions
  async expectShareCount(count: number): Promise<void> {
    await expect(this.page.locator('.share-item')).toHaveCount(count);
  }

  async expectShareHolder(name: string): Promise<void> {
    await expect(this.page.locator('.share-item').filter({ hasText: name })).toBeVisible();
  }

  async expectReadyToRecover(): Promise<void> {
    await expect(this.page.locator('#threshold-info')).toHaveClass(/ready/);
  }

  async expectNeedMoreShares(count: number): Promise<void> {
    await expect(this.page.locator('#threshold-info')).toContainText(`Waiting for ${count} more piece`);
  }

  async expectManifestLoaded(): Promise<void> {
    await expect(this.page.locator('#manifest-status')).toHaveClass(/loaded/);
  }

  async expectManifestDropZoneVisible(): Promise<void> {
    await expect(this.page.locator('#manifest-drop-zone')).toBeVisible();
  }

  async expectRecoverEnabled(): Promise<void> {
    await expect(this.page.locator('#recover-btn')).toBeEnabled();
  }

  async expectRecoverDisabled(): Promise<void> {
    await expect(this.page.locator('#recover-btn')).toBeDisabled();
  }

  async expectRecoveryComplete(): Promise<void> {
    await expect(this.page.locator('#status-message')).toContainText('All done', { timeout: 60000 });
  }

  async expectFileCount(count: number): Promise<void> {
    await expect(this.page.locator('.file-item')).toHaveCount(count);
  }

  async expectDownloadVisible(): Promise<void> {
    await expect(this.page.locator('#download-all-btn')).toBeVisible();
  }

  async expectUIElements(): Promise<void> {
    await expect(this.page.locator('h1')).toContainText('🧠 ReMemory Recovery');
    await expect(this.page.locator('#share-drop-zone')).toBeVisible();
    await expect(this.page.locator('#manifest-drop-zone')).toBeVisible();
  }

  // Dismiss dialogs (for duplicate share tests)
  onDialog(action: 'dismiss' | 'accept' = 'dismiss'): void {
    this.page.on('dialog', dialog => dialog[action]());
  }

  // Paste functionality
  async clickPasteButton(): Promise<void> {
    await this.page.locator('#paste-toggle-btn').click();
  }

  async expectPasteAreaVisible(): Promise<void> {
    await expect(this.page.locator('#paste-area')).toBeVisible();
  }

  async pasteShare(content: string): Promise<void> {
    await this.page.locator('#paste-input').fill(content);
  }

  async submitPaste(): Promise<void> {
    await this.page.locator('#paste-submit-btn').click();
  }

  // Holder share label check
  async expectHolderShareLabel(): Promise<void> {
    await expect(this.page.locator('.share-item').first()).toContainText('Your piece');
  }

  // Contact list assertions
  async expectContactListVisible(): Promise<void> {
    await expect(this.page.locator('#contact-list-section')).toBeVisible();
  }

  async expectContactItem(name: string): Promise<void> {
    await expect(this.page.locator('.contact-item').filter({ hasText: name })).toBeVisible();
  }

  async expectContactCollected(name: string): Promise<void> {
    const contact = this.page.locator('.contact-item').filter({ hasText: name });
    await expect(contact).toHaveClass(/collected/);
  }

  async expectContactNotCollected(name: string): Promise<void> {
    const contact = this.page.locator('.contact-item').filter({ hasText: name });
    await expect(contact).not.toHaveClass(/collected/);
  }

  // Steps collapse assertions
  async expectStepsVisible(): Promise<void> {
    await expect(this.page.locator('.card').first()).toBeVisible();
  }

  async expectStepsCollapsed(): Promise<void> {
    await expect(this.page.locator('.card.collapsed').first()).toBeAttached();
  }
}

// Page helper class for bundle creation tool interactions
export class CreationPage {
  constructor(private page: Page, private htmlPath: string) {}

  // Navigate to maker.html and wait for WASM
  async open(): Promise<void> {
    await this.page.goto(`file://${this.htmlPath}`);
    await this.page.waitForFunction(
      () => (window as any).rememoryReady === true,
      { timeout: 30000 }
    );
  }

  // Friends management
  async addFriend(): Promise<void> {
    await this.page.locator('#add-friend-btn').click();
  }

  async removeFriend(index: number): Promise<void> {
    const removeButtons = this.page.locator('.friend-entry .remove-btn');
    await removeButtons.nth(index).click();
  }

  async setFriend(index: number, name: string, contact?: string): Promise<void> {
    const entry = this.page.locator('.friend-entry').nth(index);
    await entry.locator('.friend-name').fill(name);
    if (contact) {
      await entry.locator('.friend-contact').fill(contact);
    }
  }

  async expectFriendCount(count: number): Promise<void> {
    await expect(this.page.locator('.friend-entry')).toHaveCount(count);
  }

  async expectFriendData(index: number, name: string, contact?: string): Promise<void> {
    const entry = this.page.locator('.friend-entry').nth(index);
    await expect(entry.locator('.friend-name')).toHaveValue(name);
    if (contact !== undefined) {
      await expect(entry.locator('.friend-contact')).toHaveValue(contact);
    }
  }

  // Threshold
  async setThreshold(value: number): Promise<void> {
    await this.page.locator('#threshold-select').selectOption(String(value));
  }

  async expectThresholdOptions(options: string[]): Promise<void> {
    const select = this.page.locator('#threshold-select');
    for (const option of options) {
      await expect(select.locator('option', { hasText: option })).toBeAttached();
    }
  }

  // YAML import
  async importYAML(content: string): Promise<void> {
    // Open the details element
    await this.page.locator('.import-section summary').click();
    await this.page.locator('#yaml-import').fill(content);
    await this.page.locator('#import-btn').click();
  }

  // Files
  createTestFiles(tmpDir: string, prefix: string = 'default'): string[] {
    const filesDir = path.join(tmpDir, `test-files-${prefix}`);
    fs.mkdirSync(filesDir, { recursive: true });

    const file1 = path.join(filesDir, `${prefix}-secret.txt`);
    const file2 = path.join(filesDir, `${prefix}-notes.txt`);

    fs.writeFileSync(file1, `This is a secret password (${prefix}): correct-horse-battery-staple`);
    fs.writeFileSync(file2, `Remember to feed the cat! (${prefix})`);

    return [file1, file2];
  }

  async addFiles(filePaths: string[]): Promise<void> {
    await this.page.locator('#files-input').setInputFiles(filePaths);
  }

  async expectFilesPreviewVisible(): Promise<void> {
    await expect(this.page.locator('#files-preview')).toBeVisible();
  }

  async expectFileCount(count: number): Promise<void> {
    await expect(this.page.locator('#files-preview .file-item')).toHaveCount(count);
  }

  // Generation
  async expectGenerateEnabled(): Promise<void> {
    await expect(this.page.locator('#generate-btn')).toBeEnabled();
  }

  async expectGenerateDisabled(): Promise<void> {
    await expect(this.page.locator('#generate-btn')).toBeDisabled();
  }

  async generate(): Promise<void> {
    await this.page.locator('#generate-btn').click();
  }

  async expectGenerationComplete(): Promise<void> {
    await expect(this.page.locator('#status-message')).toContainText('successfully', { timeout: 120000 });
  }

  async expectBundleCount(count: number): Promise<void> {
    await expect(this.page.locator('.bundle-item')).toHaveCount(count);
  }

  async expectBundleFor(name: string): Promise<void> {
    await expect(this.page.locator('.bundle-item').filter({ hasText: name })).toBeVisible();
  }

  // Download bundle and return data
  async downloadBundle(index: number): Promise<Uint8Array | null> {
    // Get bundle data from the page's state
    const data = await this.page.evaluate((idx) => {
      const state = (window as any).rememoryBundles;
      if (!state || !state[idx]) return null;
      return Array.from(state[idx].data as Uint8Array);
    }, index);

    if (!data) return null;
    return new Uint8Array(data);
  }

  // UI assertions
  async expectUIElements(): Promise<void> {
    await expect(this.page.locator('.logo')).toContainText('ReMemory');
    await expect(this.page.locator('#friends-list')).toBeVisible();
    await expect(this.page.locator('#files-drop-zone')).toBeVisible();
    await expect(this.page.locator('#generate-btn')).toBeVisible();
  }

  async expectPageTitle(title: string): Promise<void> {
    await expect(this.page.locator('h1')).toContainText(title);
  }

  // Language
  async setLanguage(lang: string): Promise<void> {
    await this.page.locator(`.lang-toggle button[data-lang="${lang}"]`).click();
  }

  // Dismiss dialogs (for validation error tests)
  onDialog(action: 'dismiss' | 'accept' = 'dismiss'): void {
    this.page.on('dialog', dialog => dialog[action]());
  }

  // Anonymous mode methods
  async toggleAnonymousMode(): Promise<void> {
    await this.page.locator('#anonymous-mode').click();
  }

  async expectAnonymousModeChecked(): Promise<void> {
    await expect(this.page.locator('#anonymous-mode')).toBeChecked();
  }

  async expectAnonymousModeUnchecked(): Promise<void> {
    await expect(this.page.locator('#anonymous-mode')).not.toBeChecked();
  }

  async expectFriendsListHidden(): Promise<void> {
    await expect(this.page.locator('#friends-section')).toHaveClass(/hidden/);
  }

  async expectFriendsListVisible(): Promise<void> {
    await expect(this.page.locator('#friends-section')).not.toHaveClass(/hidden/);
  }

  async expectSharesInputVisible(): Promise<void> {
    await expect(this.page.locator('#shares-input')).toBeVisible();
  }

  async expectSharesInputHidden(): Promise<void> {
    await expect(this.page.locator('#shares-input')).toHaveClass(/hidden/);
  }

  async setNumShares(count: number): Promise<void> {
    await this.page.locator('#num-shares').fill(String(count));
    // Trigger input event to update state
    await this.page.locator('#num-shares').dispatchEvent('input');
  }

  async expectNumShares(count: number): Promise<void> {
    await expect(this.page.locator('#num-shares')).toHaveValue(String(count));
  }
}
