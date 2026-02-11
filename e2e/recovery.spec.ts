import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import {
  getRememoryBin,
  createTestProject,
  createAnonymousTestProject,
  extractBundle,
  extractBundles,
  extractAnonymousBundles,
  extractWordsFromReadme,
  findReadmeFile,
  generateStandaloneHTML,
  RecoveryPage
} from './helpers';

test.describe('Browser Recovery Tool', () => {
  let projectDir: string;
  let bundlesDir: string;

  test.beforeAll(async () => {
    // Skip if rememory binary not available
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      console.log(`Skipping tests: rememory binary not found at ${bin}`);
      test.skip();
      return;
    }

    projectDir = createTestProject();
    bundlesDir = path.join(projectDir, 'output', 'bundles');
  });

  test.afterAll(async () => {
    if (projectDir && fs.existsSync(projectDir)) {
      fs.rmSync(projectDir, { recursive: true, force: true });
    }
  });

  test('recover.html loads and shows UI', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();
    await recovery.expectUIElements();
  });

  test('personalized recover.html pre-loads holder share', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();

    // Alice's share should already be loaded (personalization)
    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Alice');
    await recovery.expectHolderShareLabel();

    // Manifest is NOT pre-loaded - user must load it
    await recovery.expectManifestDropZoneVisible();

    // Still need 1 more share (threshold is 2)
    await recovery.expectNeedMoreShares(1);
  });

  test('shows contact list for other friends', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();

    // Contact list should show Bob and Carol (other friends)
    await recovery.expectContactListVisible();
    await recovery.expectContactItem('Bob');
    await recovery.expectContactItem('Carol');
  });

  test('contact list updates when shares are collected', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Bob's contact should not be checked initially
    await recovery.expectContactNotCollected('Bob');

    // Add Bob's share
    await recovery.addShares(bobDir);

    // Bob's contact should now be checked
    await recovery.expectContactCollected('Bob');
  });

  test('paste share functionality', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Click paste button to show textarea
    await recovery.clickPasteButton();
    await recovery.expectPasteAreaVisible();

    // Read Bob's share and paste it
    const bobShare = fs.readFileSync(findReadmeFile(bobDir), 'utf8');
    await recovery.pasteShare(bobShare);
    await recovery.submitPaste();

    // Bob's share should be added
    await recovery.expectShareCount(2);
    await recovery.expectShareHolder('Bob');
  });

  test('auto-recovery when threshold is met', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Alice's share is pre-loaded
    await recovery.expectShareCount(1);

    // Load manifest first
    await recovery.addManifest();
    await recovery.expectManifestLoaded();

    // Add Bob's share - should trigger auto-recovery
    await recovery.addShares(bobDir);

    // Recovery should complete automatically
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3);
    await recovery.expectDownloadVisible();
  });

  test('steps collapse after recovery starts', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Steps 1 and 2 should be visible initially
    await recovery.expectStepsVisible();

    // Load manifest first
    await recovery.addManifest();

    // Add Bob's share - triggers auto-recovery
    await recovery.addShares(bobDir);

    // Steps should collapse
    await recovery.expectStepsCollapsed();
  });

  test('can add shares from README.txt files', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Alice's share is already pre-loaded via personalization
    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Alice');

    // Add Bob's share
    await recovery.addShares(bobDir);
    await recovery.expectShareCount(2);
    await recovery.expectReadyToRecover();
  });

  test('full recovery workflow', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Alice's share is pre-loaded via personalization
    await recovery.expectShareCount(1);

    // Load manifest
    await recovery.addManifest();
    await recovery.expectManifestLoaded();

    // Add Bob's share (triggers auto-recovery)
    await recovery.addShares(bobDir);

    // Recovery should complete automatically
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3); // secret.txt, notes.txt, README.md
    await recovery.expectDownloadVisible();
  });

  test('shows need for more shares with only holder share', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();

    // Only holder's share is loaded (threshold is 2)
    await recovery.expectShareCount(1);
    await recovery.expectNeedMoreShares(1);
  });

  test('recover via typed words in paste area', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Alice's share is pre-loaded via personalization
    await recovery.expectShareCount(1);

    // Extract Bob's 25 recovery words from his README.txt
    const words = extractWordsFromReadme(findReadmeFile(bobDir));
    expect(words.split(' ').length).toBe(25);

    // Type the 25 words into the paste area (includes index as 25th word)
    await recovery.clickPasteButton();
    await recovery.expectPasteAreaVisible();
    await recovery.pasteShare(words);
    await recovery.submitPaste();

    // Bob's share should now be added (index extracted from 25th word)
    await recovery.expectShareCount(2);
  });

  test('paste area accepts numbered word grid directly', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();
    await recovery.expectShareCount(1);

    // Read Bob's README.txt and extract the word grid section as-is
    const bobReadme = fs.readFileSync(findReadmeFile(bobDir), 'utf8');
    const wordsMatch = bobReadme.match(/YOUR 25 RECOVERY WORDS:\n\n([\s\S]*?)\n\nRead these words/);
    expect(wordsMatch).not.toBeNull();
    const wordGrid = wordsMatch![1]; // The numbered two-column grid

    // Paste the word grid into the paste area
    await recovery.clickPasteButton();
    await recovery.expectPasteAreaVisible();
    await recovery.pasteShare(wordGrid);
    await recovery.submitPaste();

    // Share should be added directly (index from 25th word, no manual input needed)
    await recovery.expectShareCount(2);
  });

  test('detects duplicate shares', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();
    recovery.onDialog('dismiss');

    // Alice's share is already pre-loaded, try to add it again
    await recovery.addShares(bundleDir);
    await recovery.expectShareCount(1); // Still 1, duplicate ignored
  });
});

test.describe('Anonymous Bundle Recovery', () => {
  let anonProjectDir: string;
  let anonBundlesDir: string;

  test.beforeAll(async () => {
    // Skip if rememory binary not available
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      console.log(`Skipping tests: rememory binary not found at ${bin}`);
      test.skip();
      return;
    }

    anonProjectDir = createAnonymousTestProject();
    anonBundlesDir = path.join(anonProjectDir, 'output', 'bundles');
  });

  test.afterAll(async () => {
    if (anonProjectDir && fs.existsSync(anonProjectDir)) {
      fs.rmSync(anonProjectDir, { recursive: true, force: true });
    }
  });

  test('anonymous recover.html loads and shows UI without contact list', async ({ page }) => {
    const [share1Dir] = extractAnonymousBundles(anonBundlesDir, [1]);
    const recovery = new RecoveryPage(page, share1Dir);

    await recovery.open();
    await recovery.expectUIElements();

    // Share should be pre-loaded with synthetic name
    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Share 1');

    // Contact list should NOT be visible for anonymous bundles
    await expect(page.locator('#contact-list-section')).not.toBeVisible();
  });

  test('anonymous full recovery workflow', async ({ page }) => {
    const [share1Dir, share2Dir] = extractAnonymousBundles(anonBundlesDir, [1, 2]);
    const recovery = new RecoveryPage(page, share1Dir);

    await recovery.open();

    // Share 1 is pre-loaded
    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Share 1');

    // Load manifest
    await recovery.addManifest();
    await recovery.expectManifestLoaded();

    // Add Share 2 (triggers auto-recovery since threshold is 2)
    await recovery.addShares(share2Dir);

    // Recovery should complete automatically
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3); // secret.txt, notes.txt, README.md
    await recovery.expectDownloadVisible();
  });

  test('anonymous recovery shows generic share labels', async ({ page }) => {
    const [share1Dir, share2Dir] = extractAnonymousBundles(anonBundlesDir, [1, 2]);
    const recovery = new RecoveryPage(page, share1Dir);

    await recovery.open();

    // Add Share 2
    await recovery.addShares(share2Dir);

    // Both shares should be visible with synthetic names
    await recovery.expectShareCount(2);
    await recovery.expectShareHolder('Share 1');
    await recovery.expectShareHolder('Share 2');
  });
});

test.describe('Generic recover.html (no personalization)', () => {
  let projectDir: string;
  let bundlesDir: string;
  let standaloneRecoverHtml: string;
  let tmpDir: string;

  test.beforeAll(async () => {
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      console.log(`Skipping tests: rememory binary not found at ${bin}`);
      test.skip();
      return;
    }

    projectDir = createTestProject();
    bundlesDir = path.join(projectDir, 'output', 'bundles');
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-generic-e2e-'));
    standaloneRecoverHtml = generateStandaloneHTML(tmpDir, 'recover');
  });

  test.afterAll(async () => {
    if (projectDir && fs.existsSync(projectDir)) {
      fs.rmSync(projectDir, { recursive: true, force: true });
    }
    if (tmpDir && fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  test('words-only shares auto-recover when manifest is loaded', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, tmpDir);

    await recovery.openFile(standaloneRecoverHtml);
    await recovery.expectShareCount(0);

    // Paste Alice's words
    const aliceWords = extractWordsFromReadme(findReadmeFile(aliceDir));
    await recovery.clickPasteButton();
    await recovery.pasteShare(aliceWords);
    await recovery.submitPaste();
    await recovery.expectShareCount(1);

    // Paste Bob's words
    const bobWords = extractWordsFromReadme(findReadmeFile(bobDir));
    await recovery.clickPasteButton();
    await recovery.pasteShare(bobWords);
    await recovery.submitPaste();
    await recovery.expectShareCount(2);

    // Load manifest — recovery should auto-trigger (2 shares, threshold unknown)
    await recovery.addManifest(aliceDir);
    await recovery.expectManifestLoaded();

    // Recovery should complete automatically
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3);
    await recovery.expectDownloadVisible();
  });

  test('words-first entry recovers when second share provides threshold', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    // Use a dummy bundleDir — we'll open the standalone HTML directly
    const recovery = new RecoveryPage(page, tmpDir);

    await recovery.openFile(standaloneRecoverHtml);

    // No personalization — no shares pre-loaded
    await recovery.expectShareCount(0);

    // Extract Alice's 25 recovery words from her README.txt
    const aliceWords = extractWordsFromReadme(findReadmeFile(aliceDir));
    expect(aliceWords.split(' ').length).toBe(25);

    // Paste Alice's words as the FIRST share (no threshold/total available)
    await recovery.clickPasteButton();
    await recovery.expectPasteAreaVisible();
    await recovery.pasteShare(aliceWords);
    await recovery.submitPaste();

    // Alice's share should be added (index extracted from 25th word)
    await recovery.expectShareCount(1);

    // Load manifest from Alice's bundle
    await recovery.addManifest(aliceDir);
    await recovery.expectManifestLoaded();

    // Add Bob's share via README.txt file drop — this carries threshold/total
    await recovery.addShares(bobDir);

    // Bob's share should be added and threshold should now be known
    await recovery.expectShareCount(2);

    // Recovery should complete automatically (threshold backfilled from Bob's share)
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3); // secret.txt, notes.txt, README.md
    await recovery.expectDownloadVisible();
  });
});
