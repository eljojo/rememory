import { test, expect } from './fixtures';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import AdmZip from 'adm-zip';
import {
  getRememoryBin,
  CreationPage,
  RecoveryPage,
  generateStandaloneHTML
} from './helpers';

test.describe('Browser Bundle Creation Tool', () => {
  let htmlPath: string;
  let tmpDir: string;

  test.beforeAll(async () => {
    // Skip if rememory binary not available
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      test.skip();
      return;
    }

    // Generate standalone maker.html for testing
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-create-e2e-'));
    htmlPath = generateStandaloneHTML(tmpDir, 'create');
  });

  test.afterAll(async () => {
    if (tmpDir && fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  test('maker.html loads and shows UI', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();
    await creation.expectUIElements();
  });

  test('can add and remove friends', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Should start with 2 friends
    await creation.expectFriendCount(2);

    // Add a friend
    await creation.addFriend();
    await creation.expectFriendCount(3);

    // Remove a friend
    await creation.removeFriend(2);
    await creation.expectFriendCount(2);
  });

  test('threshold updates with friend count', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // With 2 friends, threshold should be "2 of 2"
    await creation.expectThresholdOptions(['2 of 2']);

    // Add a friend
    await creation.addFriend();

    // With 3 friends, threshold options should include 2 and 3
    await creation.expectThresholdOptions(['2 of 3', '3 of 3']);
  });

  test('threshold is visible and selectable', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Threshold is hidden until 2 friends have names
    await creation.expectThresholdHidden();

    // Fill in both names — threshold appears
    await creation.setFriend(0, 'Alice');
    await creation.expectThresholdHidden();
    await creation.setFriend(1, 'Bob');
    await creation.expectThresholdVisible();
    await creation.expectThresholdOptions(['2 of 2']);

    // Add a third friend and verify the threshold can be changed
    await creation.addFriend();
    await creation.setFriend(2, 'Carol');
    await creation.expectThresholdVisible();
    await creation.expectThresholdOptions(['2 of 3', '3 of 3']);
    await creation.setThreshold(3);

    // Remove a friend — threshold should still be visible with 2 named friends
    await creation.removeFriend(2);
    await creation.expectThresholdVisible();
    await creation.expectThresholdOptions(['2 of 2']);
  });

  test('step numbers and button reflect progress', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Initial state: steps 2 and 3 are pending, button is secondary
    await creation.expectStepPending(2);
    await creation.expectStepPending(3);
    await creation.expectGenerateSecondary();

    // Fill in friend names — step 2 unlocks, step 3 still pending (no files)
    await creation.setFriend(0, 'Alice', 'alice@test.com');
    await creation.setFriend(1, 'Bob', 'bob@test.com');
    await creation.expectStepActive(2);
    await creation.expectStepPending(3);
    await creation.expectGenerateSecondary();

    // Add files — step 3 unlocks, button turns green
    const testFiles = creation.createTestFiles(tmpDir, 'btn-test');
    await creation.addFiles(testFiles);
    await creation.expectStepActive(2);
    await creation.expectStepActive(3);
    await creation.expectGeneratePrimary();

    // Clear a friend name — steps 2 and 3 go pending, button goes secondary
    await creation.setFriend(0, '', '');
    await creation.expectStepPending(2);
    await creation.expectStepPending(3);
    await creation.expectGenerateSecondary();

    // Restore the name — everything unlocks again
    await creation.setFriend(0, 'Alice', 'alice@test.com');
    await creation.expectStepActive(2);
    await creation.expectStepActive(3);
    await creation.expectGeneratePrimary();
  });

  test('validates required fields', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Generate button is enabled but clicking it shows validation errors
    await creation.expectGenerateEnabled();

    // Click generate without filling required fields - should show validation
    await creation.generate();

    // Should show validation toast
    await expect(page.locator('.toast-warning')).toBeVisible();

    // Required fields should be highlighted
    await expect(page.locator('.input-error').first()).toBeVisible();

    // Dismiss the toast by clicking the backdrop
    await page.locator('.toast-close').first().click();

    // Fill in friends
    await creation.setFriend(0, 'Alice', 'alice@test.com');
    await creation.setFriend(1, 'Bob', 'bob@test.com');

    // Click generate again without files - still should show file validation (name is the only required field)
    await creation.generate();
    await expect(page.locator('#files-drop-zone.has-error')).toBeVisible(); // Files drop zone should be highlighted
  });

  // The friends list is rebuilt one friend at a time on several actions; the
  // threshold used to fall back to 2 on the way, silently weakening bundles.
  test('chosen threshold survives removing a friend', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();
    for (const [i, name] of ['Alice', 'Bob', 'Carol', 'Dave', 'Erin'].entries()) {
      if (i >= 2) await creation.addFriend();
      await creation.setFriend(i, name);
    }
    await creation.setThreshold(3);

    await creation.removeFriend(4);

    await creation.expectFriendCount(4);
    await expect(page.locator('#threshold-select')).toHaveValue('3');
  });

  test('chosen threshold survives a language change', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();
    for (const [i, name] of ['Alice', 'Bob', 'Carol', 'Dave', 'Erin'].entries()) {
      if (i >= 2) await creation.addFriend();
      await creation.setFriend(i, name);
    }
    await creation.setThreshold(4);

    await creation.setLanguage('es');

    await expect(page.locator('#threshold-select')).toHaveValue('4');
  });

  test('chosen threshold comes back when the piece count goes back up', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();
    await creation.toggleAnonymousMode();
    await creation.setNumShares(5);
    await creation.setThreshold(4);

    await creation.setNumShares(3);
    await expect(page.locator('#threshold-select')).toHaveValue('3');

    await creation.setNumShares(5);
    await expect(page.locator('#threshold-select')).toHaveValue('4');
  });

  test('YAML import refuses a threshold above the number of friends', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();
    await creation.setFriend(0, 'Alice');
    await creation.setFriend(1, 'Bob');

    await creation.importYAML(`
name: too-strict
threshold: 4
friends:
  - name: Charlie
  - name: Diana
  - name: Eve
`);

    await expect(page.locator('.toast-error')).toBeVisible();
    await creation.expectFriendCount(2);
    await creation.expectFriendData(0, 'Alice');
    await creation.expectFriendData(1, 'Bob');
  });

  test('YAML import refuses a language the bundles cannot use', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();
    await creation.setFriend(0, 'Alice');
    await creation.setFriend(1, 'Bob');

    await creation.importYAML(`
name: regional
friends:
  - name: Charlie
    language: pt-BR
  - name: Diana
`);

    await expect(page.locator('.toast-error')).toBeVisible();
    await creation.expectFriendCount(2);
    await creation.expectFriendData(0, 'Alice');
  });

  test('YAML import keeps a valid threshold from the file', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();

    await creation.importYAML(`
name: imported
threshold: 3
friends:
  - name: Charlie
  - name: Diana
  - name: Eve
  - name: Frank
`);

    await creation.expectFriendCount(4);
    await expect(page.locator('#threshold-select')).toHaveValue('3');
  });

  test('can import contacts from YAML', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    const yamlContent = `
name: imported-project
threshold: 2
friends:
  - name: Charlie
    contact: charlie@test.com
  - name: Diana
    contact: diana@test.com
  - name: Eve
    contact: eve@test.com
`;

    await creation.importYAML(yamlContent);

    // Friends should be imported
    await creation.expectFriendCount(3);
    await creation.expectFriendData(0, 'Charlie', 'charlie@test.com');
    await creation.expectFriendData(1, 'Diana', 'diana@test.com');
    await creation.expectFriendData(2, 'Eve', 'eve@test.com');
  });

  test('file selection shows preview', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Create test files
    const testFiles = creation.createTestFiles(tmpDir);

    // Add files
    await creation.addFiles(testFiles);

    // Should show file preview
    await creation.expectFilesPreviewVisible();
    await creation.expectFileCount(2);
  });

  // Hidden files are often the secrets themselves (.env, .ssh, .password-store);
  // they used to be dropped without a word. The CLI archives them too.
  test('hidden files are kept', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();

    const dir = path.join(tmpDir, 'hidden-files');
    fs.mkdirSync(dir, { recursive: true });
    const dotenv = path.join(dir, '.env');
    fs.writeFileSync(dotenv, 'API_KEY=secret');
    await creation.addFiles([...creation.createTestFiles(tmpDir, 'hidden'), dotenv]);

    await creation.expectFileCount(3);
    await expect(page.locator('#files-preview')).toContainText('.env');
  });

  // Chrome's directory reader returns at most 100 entries per call; a dropped
  // folder must be read until the reader says it's done.
  test('dropping a folder with more than 100 files keeps them all', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();

    await page.evaluate(() => {
      const fileEntry = (name: string) => ({
        isFile: true, isDirectory: false, name,
        file: (cb: (f: File) => void) => cb(new File([name], name)),
      });
      const all = Array.from({ length: 250 }, (_, i) => fileEntry(`f${i}.txt`));
      const folder = {
        isFile: false, isDirectory: true, name: 'big',
        createReader: () => {
          let offset = 0;
          return {
            readEntries: (cb: (entries: unknown[]) => void) => {
              const batch = all.slice(offset, offset + 100);
              offset += batch.length;
              setTimeout(() => cb(batch));
            },
          };
        },
      };
      const drop = new Event('drop', { bubbles: true, cancelable: true });
      Object.defineProperty(drop, 'dataTransfer', {
        value: { items: [{ kind: 'file', webkitGetAsEntry: () => folder }] },
      });
      document.getElementById('files-drop-zone')!.dispatchEvent(drop);
    });

    await creation.expectFileCount(250);
  });

  test('adding more files appends to existing files', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Create first batch of test files
    const firstBatch = creation.createTestFiles(tmpDir, 'batch1');

    // Add first batch
    await creation.addFiles(firstBatch);
    await creation.expectFilesPreviewVisible();
    await creation.expectFileCount(2);

    // Create second batch of test files
    const secondBatch = creation.createTestFiles(tmpDir, 'batch2');

    // Add second batch - should append, not replace
    await creation.addFiles(secondBatch);

    // Should now have all 4 files
    await creation.expectFileCount(4);
  });

  test('language switching works', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Default should be English
    await creation.expectPageTitle('Create Bundles');

    // Switch to Spanish
    await creation.setLanguage('es');
    await creation.expectPageTitle('Crear Kits de Recuperación');

    // Switch to German
    await creation.setLanguage('de');
    await creation.expectPageTitle('Umschläge erstellen');

    // Switch back to English
    await creation.setLanguage('en');
    await creation.expectPageTitle('Create Bundles');

    // Switch to Portuguese
    await creation.setLanguage('pt');
    await creation.expectPageTitle('Criar Pacotes de Recuperação');
  });

  test('nav guide link updates for languages with translated docs', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    const guideLink = page.locator('#nav-links-main a[data-i18n="nav_guide"]');

    // English: docs.html
    await creation.setLanguage('en');
    await expect(guideLink).toHaveAttribute('href', 'docs.html');

    // German has translated docs — link should point to docs.de.html
    await creation.setLanguage('de');
    await expect(guideLink).toHaveAttribute('href', 'docs.de.html');

    // Spanish has translated docs
    await creation.setLanguage('es');
    await expect(guideLink).toHaveAttribute('href', 'docs.es.html');

    // Portuguese does NOT have translated docs — should stay docs.html
    await creation.setLanguage('pt');
    await expect(guideLink).toHaveAttribute('href', 'docs.html');

    // Back to English
    await creation.setLanguage('en');
    await expect(guideLink).toHaveAttribute('href', 'docs.html');
  });

  test('minimum 2 friends required — remove clears fields instead', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Fill in both friends
    await creation.setFriend(0, 'Alice', 'alice@test.com');
    await creation.setFriend(1, 'Bob', 'bob@test.com');

    // Try to remove a friend — should clear fields, not remove the row
    await creation.removeFriend(1);
    await creation.expectFriendCount(2);
    await creation.expectFriendData(1, '', '');

    // First friend should be untouched
    await creation.expectFriendData(0, 'Alice', 'alice@test.com');
  });

  test('anonymous mode toggle hides friends list', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Anonymous mode should be off by default
    await creation.expectAnonymousModeUnchecked();
    await creation.expectFriendsListVisible();
    await creation.expectSharesInputHidden();

    // Enable anonymous mode
    await creation.toggleAnonymousMode();

    // Friends list should be hidden, shares input should be visible
    await creation.expectAnonymousModeChecked();
    await creation.expectFriendsListHidden();
    await creation.expectSharesInputVisible();

    // Disable anonymous mode
    await creation.toggleAnonymousMode();

    // Friends list should be visible again
    await creation.expectAnonymousModeUnchecked();
    await creation.expectFriendsListVisible();
    await creation.expectSharesInputHidden();
  });

  test('anonymous mode threshold updates with share count', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Enable anonymous mode
    await creation.toggleAnonymousMode();

    // Default 5 shares should have threshold options 2-5
    await creation.expectNumShares(5);
    await creation.expectThresholdOptions(['2 of 5', '3 of 5', '4 of 5', '5 of 5']);

    // Change to 3 shares
    await creation.setNumShares(3);
    await creation.expectThresholdOptions(['2 of 3', '3 of 3']);

    // Change to 7 shares
    await creation.setNumShares(7);
    await creation.expectThresholdOptions(['2 of 7', '3 of 7', '4 of 7', '5 of 7', '6 of 7', '7 of 7']);
  });

  test('anonymous mode full bundle creation workflow', async ({ page }, testInfo) => {
    testInfo.setTimeout(120000);
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Enable anonymous mode
    await creation.toggleAnonymousMode();
    await creation.expectSharesInputVisible();

    // Set 4 shares with threshold 3
    await creation.setNumShares(4);
    await creation.setThreshold(3);

    // Add test files
    const testFiles = creation.createTestFiles(tmpDir, 'anon');
    await creation.addFiles(testFiles);

    // Generate bundles
    await creation.generate();

    // Should complete successfully
    await creation.expectGenerationComplete();

    // Should have 4 bundles (Share 1, Share 2, Share 3, Share 4)
    await creation.expectBundleCount(4);
    await creation.expectBundleFor('Share 1');
    await creation.expectBundleFor('Share 2');
    await creation.expectBundleFor('Share 3');
    await creation.expectBundleFor('Share 4');
  });

  test('friends without latin letters in their names get distinct bundle files', async ({ page }, testInfo) => {
    testInfo.setTimeout(120000);
    const creation = new CreationPage(page, htmlPath);

    await creation.open();
    await creation.setFriend(0, '小明');
    await creation.setFriend(1, '小華');
    await creation.addFiles(creation.createTestFiles(tmpDir, 'nonlatin'));

    await creation.generate();
    await creation.expectGenerationComplete();
    await creation.expectBundleCount(2);
    await expect(page.locator('.bundle-item .meta').nth(0)).toContainText('bundle-1.zip');
    await expect(page.locator('.bundle-item .meta').nth(1)).toContainText('bundle-2.zip');
  });

  test('friends whose names give the same file name are refused', async ({ page }, testInfo) => {
    testInfo.setTimeout(120000);
    const creation = new CreationPage(page, htmlPath);

    await creation.open();
    await creation.setFriend(0, 'José');
    await creation.setFriend(1, 'Jose');
    await creation.addFiles(creation.createTestFiles(tmpDir, 'collide'));

    await creation.generate();
    await expect(page.locator('#status-message')).toContainText('SHARE-jose.txt', { timeout: 120000 });
    await creation.expectBundleCount(0);
  });

  test('anonymous mode validates files required', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Enable anonymous mode
    await creation.toggleAnonymousMode();

    // Don't add any files - this should fail validation

    // Try to generate - should show validation error
    await creation.generate();

    // Should show validation toast for missing files
    await expect(page.locator('.toast-warning')).toBeVisible();

    // Files drop zone should be highlighted
    await expect(page.locator('#files-drop-zone.has-error')).toBeVisible();
  });

  test('YAML export escapes special characters in friend names and contact fields', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Set friends with special characters that need escaping
    // Focus on testing quote and backslash escaping which are most critical for YAML validity
    await page.locator('.friend-entry').nth(0).locator('.friend-name').fill('Alice "The Hacker" Smith');
    await page.locator('.friend-entry').nth(0).locator('.friend-contact').fill('Email: alice@test.com');

    await page.locator('.friend-entry').nth(1).locator('.friend-name').fill('Bob\\Johnson');
    await page.locator('.friend-entry').nth(1).locator('.friend-contact').fill('Contact info: bob@example.com');

    // Export YAML
    const yamlContent = await creation.exportYAML();

    // Verify the YAML contains properly escaped characters
    // The escaping function should convert:
    // - Double quotes to \" (backslash-quote)
    // - Backslashes to \\ (backslash-backslash)
    // This prevents YAML injection and ensures syntactic validity
    expect(yamlContent).toContain('\\"The Hacker\\"');  // Quotes should be escaped
    expect(yamlContent).toContain('Bob\\\\Johnson');     // Backslashes should be doubled
    
    // Verify that the entire name and contact fields are properly quoted
    expect(yamlContent).toMatch(/name: "Alice \\"The Hacker\\" Smith"/);
    expect(yamlContent).toMatch(/name: "Bob\\\\Johnson"/);
    expect(yamlContent).toMatch(/contact: "Email: alice@test\.com"/);
    expect(yamlContent).toMatch(/contact: "Contact info: bob@example\.com"/);
    
    // Verify the YAML can be parsed (imported) without errors
    // This tests that the escaping produces valid YAML
    await creation.importYAML(yamlContent);

    // Should have successfully imported 2 friends
    await creation.expectFriendCount(2);
  });

  test('browser-created bundles can be recovered @cross-browser', async ({ page }, testInfo) => {
    testInfo.setTimeout(120000);
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Set up friends
    await creation.setFriend(0, 'Alice', 'alice@test.com');
    await creation.setFriend(1, 'Bob', 'bob@test.com');

    // Add test files
    const testFiles = creation.createTestFiles(tmpDir, 'roundtrip');
    await creation.addFiles(testFiles);

    // Generate bundles via WASM
    await creation.generate();
    await creation.expectGenerationComplete();
    await creation.expectBundleCount(2);

    // Download both bundles
    const aliceData = await creation.downloadBundle(0);
    const bobData = await creation.downloadBundle(1);
    expect(aliceData).toBeTruthy();
    expect(bobData).toBeTruthy();

    // Save and extract bundles
    const bundlesDir = path.join(tmpDir, 'roundtrip-bundles');
    fs.mkdirSync(bundlesDir, { recursive: true });

    const aliceZipPath = path.join(bundlesDir, 'bundle-alice.zip');
    fs.writeFileSync(aliceZipPath, aliceData!);
    const aliceZip = new AdmZip(aliceZipPath);
    const aliceDir = path.join(bundlesDir, 'alice');
    aliceZip.extractAllTo(aliceDir, true);

    const bobZipPath = path.join(bundlesDir, 'bundle-bob.zip');
    fs.writeFileSync(bobZipPath, bobData!);
    const bobZip = new AdmZip(bobZipPath);
    const bobDir = path.join(bundlesDir, 'bob');
    bobZip.extractAllTo(bobDir, true);

    // Open Alice's recover.html (share + manifest pre-loaded)
    const recovery = new RecoveryPage(page, aliceDir);
    await recovery.open();
    await recovery.expectShareCount(1);

    // Add Bob's share — triggers auto-recovery
    await recovery.addShares(bobDir);

    // Recovery should complete
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(2);
    await recovery.expectDownloadVisible();
  });
});
