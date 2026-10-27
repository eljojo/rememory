import { test, expect } from './fixtures';
import { ChildProcess, spawn } from 'child_process';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import * as net from 'net';
import AdmZip from 'adm-zip';
import { getRememoryBin, extractWordsFromReadme, findReadmeFile } from './helpers';

// Find an available port
async function getAvailablePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const srv = net.createServer();
    srv.listen(0, '127.0.0.1', () => {
      const addr = srv.address() as net.AddressInfo;
      srv.close(() => resolve(addr.port));
    });
    srv.on('error', reject);
  });
}

// Start rememory serve and wait for it to be ready
async function startServer(bin: string, port: number, dataDir: string): Promise<ChildProcess> {
  const proc = spawn(bin, ['serve', '--port', String(port), '--host', '127.0.0.1', '--data', dataDir], {
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  // Wait for server to be ready by polling the status endpoint
  const baseURL = `http://127.0.0.1:${port}`;
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    try {
      const resp = await fetch(`${baseURL}/api/status`);
      if (resp.ok) return proc;
    } catch {
      // not ready yet
    }
    await new Promise(r => setTimeout(r, 200));
  }

  proc.kill();
  throw new Error('Server did not start within 15 seconds');
}

const adminPassword = 'test-password-e2e';

test.describe('Selfhosted Server', () => {
  // The maker page and uploads need the admin password; the browser answers
  // the server's prompt with these, as the owner would.
  test.use({ allowedHosts: ['127.0.0.1'], httpCredentials: { username: 'admin', password: adminPassword } });
  let proc: ChildProcess;
  let port: number;
  let baseURL: string;
  let dataDir: string;
  let tmpDir: string;

  test.beforeAll(async () => {
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      test.skip();
      return;
    }

    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-selfhosted-e2e-'));
    dataDir = path.join(tmpDir, 'data');
    port = await getAvailablePort();
    baseURL = `http://127.0.0.1:${port}`;
    proc = await startServer(bin, port, dataDir);
  });

  test.afterAll(async () => {
    if (proc) {
      proc.kill();
      // Wait briefly for cleanup
      await new Promise(r => setTimeout(r, 500));
    }
    if (tmpDir && fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  // Helper: ensure admin password is set (idempotent — skips if already done).
  // Calls the API directly to avoid UI timing issues.
  async function ensureSetup() {
    try {
      await fetch(`${baseURL}/api/setup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password: adminPassword }),
      });
    } catch {
      // already set up, or server responded with error — either way, continue
    }
  }

  test('happy path: setup, create, home, recover, delete', async ({ page }, testInfo) => {
    testInfo.setTimeout(180000);

    // -----------------------------------------------------------
    // Step 1: Visit root — should show setup page (no password)
    // -----------------------------------------------------------
    await page.goto(baseURL);
    await expect(page.locator('h1')).toContainText('Set up ReMemory');

    // Fill in password
    await page.locator('#password').fill(adminPassword);
    await page.locator('#confirm').fill(adminPassword);
    await page.locator('button[type="submit"]').click();

    // Should land on home page after setup (root /)
    await page.waitForURL(baseURL + '/');

    // Home page should show empty state
    await expect(page.locator('.empty-state')).toBeVisible();
    await expect(page.locator('.empty-state')).toContainText('No recovery bundles here yet');

    // -----------------------------------------------------------
    // Step 2: Navigate to /create and create bundles
    // -----------------------------------------------------------
    await page.goto(`${baseURL}/maker.html`);

    // Wait for WASM to load
    await page.waitForFunction(
      () => (window as any).rememoryReady === true,
      { timeout: 30000 }
    );

    // Fill in 2 friends
    await page.locator('.friend-entry').nth(0).locator('.friend-name').fill('Alice');
    await page.locator('.friend-entry').nth(0).locator('.friend-contact').fill('alice@test.com');
    await page.locator('.friend-entry').nth(1).locator('.friend-name').fill('Bob');
    await page.locator('.friend-entry').nth(1).locator('.friend-contact').fill('bob@test.com');

    // Add a test file
    const testFile = path.join(tmpDir, 'secret.txt');
    fs.writeFileSync(testFile, 'The secret is: correct-horse-battery-staple');
    await page.locator('#files-input').setInputFiles([testFile]);

    // Generate bundles
    await page.locator('#generate-btn').click();
    await expect(page.locator('#status-message')).toContainText('ready', { timeout: 120000 });

    // Bundles should appear for both friends
    await expect(page.locator('.bundle-item')).toHaveCount(2);

    // The upload runs after the bundles are shown; wait for its toast
    await expect(page.locator('.toast-success')).toBeVisible({ timeout: 10000 });

    // Verify manifest was saved to server (check API status)
    const statusResp = await fetch(`${baseURL}/api/status`);
    const status = await statusResp.json();
    expect(status.hasManifest).toBe(true);

    // Verify "Go to home page" link is present and points to /
    const homeLink = page.locator('.next-steps-hint a[href="/"]');
    await expect(homeLink).toBeVisible();
    expect(await homeLink.textContent()).toBe('Go to home page');

    // Verify meta.json does NOT contain friend names
    const bundleDirs = fs.readdirSync(path.join(dataDir, 'bundles'));
    expect(bundleDirs.length).toBeGreaterThan(0);
    const metaPath = path.join(dataDir, 'bundles', bundleDirs[0], 'meta.json');
    const metaContent = JSON.parse(fs.readFileSync(metaPath, 'utf-8'));
    expect(metaContent).not.toHaveProperty('friends');

    // Download both bundles — extract ZIP data from the page
    const bundleData = await page.evaluate(() => {
      const bundles = (window as any).rememoryBundles;
      if (!bundles) return null;
      return bundles.map((b: any) => ({
        fileName: b.fileName,
        data: Array.from(b.data as Uint8Array),
      }));
    });
    expect(bundleData).toBeTruthy();
    expect(bundleData.length).toBe(2);

    // Save bundles to disk and extract
    const bundlesDir = path.join(tmpDir, 'bundles');
    fs.mkdirSync(bundlesDir, { recursive: true });

    const extractedDirs: string[] = [];
    for (const bundle of bundleData) {
      const zipPath = path.join(bundlesDir, bundle.fileName);
      fs.writeFileSync(zipPath, Buffer.from(bundle.data));

      const extractDir = path.join(bundlesDir, path.basename(bundle.fileName, '.zip'));
      fs.mkdirSync(extractDir, { recursive: true });
      const zip = new AdmZip(zipPath);
      zip.extractAllTo(extractDir, true);
      extractedDirs.push(extractDir);
    }

    // -----------------------------------------------------------
    // Step 3: Navigate to home — single bundle card visible
    // -----------------------------------------------------------
    await page.goto(baseURL);
    await expect(page.locator('.bundle-card')).toHaveCount(1);

    // Bundle card should show date and threshold info
    const card = page.locator('.bundle-card').first();
    await expect(card.locator('.bundle-date')).toBeVisible();
    await expect(card.locator('.bundle-meta')).toContainText('2 of 2 pieces');

    // -----------------------------------------------------------
    // Step 4: Click Recover on bundle card — goes to /recover?id=
    // -----------------------------------------------------------
    const recoverLink = card.locator('.bundle-actions a');
    const recoverHref = await recoverLink.getAttribute('href');
    expect(recoverHref).toContain('recover.html?id=');

    await recoverLink.click();
    await page.waitForURL(/\/recover\.html\?id=/);

    await page.waitForFunction(
      () => (window as any).rememoryAppReady === true,
      { timeout: 30000 }
    );

    // Manifest should auto-load from server
    await expect(page.locator('#manifest-status')).toHaveClass(/loaded/, { timeout: 15000 });

    // Add both friends' shares (README.txt files from extracted bundles)
    const readmePaths = extractedDirs.map(dir => findReadmeFile(dir, '.txt'));
    await page.locator('#share-file-input').setInputFiles(readmePaths);

    // Should have 2 shares
    await expect(page.locator('.share-item')).toHaveCount(2);

    // Recovery should complete automatically (threshold is 2, both shares + manifest loaded)
    await expect(page.locator('#status-message')).toContainText('files are ready', { timeout: 60000 });
    await expect(page.locator('#download-all-btn')).toBeVisible();

    // -----------------------------------------------------------
    // Step 5: Navigate home, delete via the delete button
    // -----------------------------------------------------------
    await page.goto(baseURL);
    await expect(page.locator('.bundle-card')).toHaveCount(1);

    // Click Delete toggle on the bundle card
    await page.locator('.delete-toggle').click();
    await expect(page.locator('.delete-form.visible')).toBeVisible();

    // Enter password and confirm deletion
    await page.locator('.delete-password').fill(adminPassword);
    await page.locator('.delete-btn').click();

    // Home should show empty state again
    await expect(page.locator('.empty-state')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.empty-state')).toContainText('No recovery bundles here yet');

    // Verify manifest is gone via API
    const statusAfter = await fetch(`${baseURL}/api/status`);
    const statusDataAfter = await statusAfter.json();
    expect(statusDataAfter.hasManifest).toBe(false);
  });

  test('create page: nav links show translated text, not raw i18n keys', async ({ page }) => {
    await ensureSetup();
    await page.goto(`${baseURL}/maker.html`);

    // Translations are applied on DOMContentLoaded — no need to wait for WASM
    const nav = page.locator('.site-nav');

    // Wait for a translated nav link to confirm translations loaded
    await expect(nav.locator('a', { hasText: 'About' })).toBeVisible();

    // Nav should not contain raw i18n keys
    const navText = await nav.textContent();
    expect(navText).not.toContain('nav_create');
    expect(navText).not.toContain('nav_about');
    expect(navText).not.toContain('nav_guide');
    expect(navText).not.toContain('nav_recover');

    // Nav should contain actual translated text (Create link is hidden on the create page)
    await expect(nav.locator('a', { hasText: 'Guide' })).toBeVisible();
    await expect(nav.locator('a', { hasText: 'Recover' })).toBeVisible();
  });

  test('recover page: single nav, no duplicates, no raw i18n keys', async ({ page }) => {
    await ensureSetup();
    await page.goto(`${baseURL}/recover.html`);

    // Translations are applied on DOMContentLoaded — no need to wait for full app init
    const nav = page.locator('.site-nav');

    // Wait for a translated nav link to confirm translations loaded
    await expect(nav.locator('a', { hasText: 'About' })).toBeVisible();

    // Should have exactly one visible nav-links div (not duplicated)
    const visibleNavLinks = page.locator('.nav-links:visible');
    await expect(visibleNavLinks).toHaveCount(1);

    // Nav should not contain raw i18n keys
    const navText = await nav.textContent();
    expect(navText).not.toContain('nav_create');
    expect(navText).not.toContain('nav_about');
    expect(navText).not.toContain('nav_guide');
    expect(navText).not.toContain('nav_recover');
  });
});
