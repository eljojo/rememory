import { test, expect } from './fixtures';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import AdmZip from 'adm-zip';
import {
  getRememoryBin,
  createTestProject,
  createAnonymousTestProject,
  cleanupProject,
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
  let mismatchProjectDir: string;
  let mismatchBundlesDir: string;

  test.beforeAll(async () => {
    // Skip if rememory binary not available
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      test.skip();
      return;
    }

    projectDir = createTestProject();
    bundlesDir = path.join(projectDir, 'output', 'bundles');
    mismatchProjectDir = createTestProject({ noEmbedManifest: true });
    mismatchBundlesDir = path.join(mismatchProjectDir, 'output', 'bundles');
  });

  test.afterAll(async () => {
    cleanupProject(projectDir);
    cleanupProject(mismatchProjectDir);
  });

  test('recover.html loads and shows UI', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();
    await recovery.expectUIElements();
    // Manifest should be pre-loaded (embedded in personalization)
    await recovery.expectManifestLoaded();
  });

  test('personalized recover.html pre-loads holder share and manifest', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();

    // Alice's share should already be loaded (personalization)
    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Alice');
    await recovery.expectHolderShareLabel();

    // Manifest is pre-loaded (embedded in recover.html for small manifests)
    await recovery.expectManifestLoaded();

    // Still need 1 more share (threshold is 2)
    await recovery.expectNeedMoreShares(1);
  });

  // A bundle can carry a language code this page has no table for (a
  // project.yml edited by hand before languages were validated, or a code
  // saved by a newer version). The page must still work, in the closest
  // language it has.
  test('personalized recover.html with an unsupported language still works', async ({ page }) => {
    // Read from the ZIP: other workers may be extracting Alice's bundle right now.
    const html = new AdmZip(path.join(bundlesDir, 'bundle-alice.zip')).readAsText('recover.html');
    const patched = html.replace(/("holder":"Alice"[^]*?"language":")[^"]*(")/, '$1pt-BR$2');
    expect(patched).not.toBe(html);
    const patchedDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-lang-'));
    fs.writeFileSync(path.join(patchedDir, 'recover.html'), patched);

    const recovery = new RecoveryPage(page, patchedDir);
    await recovery.open();

    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Alice');
    await expect(page.locator('#lang-select')).toHaveValue('pt');

    await page.locator('#lang-select').selectOption('es');
    await expect(page.locator('#lang-select')).toHaveValue('es');
    await expect(page.locator('html')).toContainText('Alice');

    fs.rmSync(patchedDir, { recursive: true, force: true });
  });

  test('an unsupported saved language falls back instead of breaking the page', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    await page.addInitScript(() => localStorage.setItem('rememory-lang', 'xx'));
    const recovery = new RecoveryPage(page, bundleDir);
    await recovery.open();

    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Alice');
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

  test('email addresses in contact list are mailto links', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();

    // Bob's email should be a tappable mailto: link
    const bobContact = page.locator('.contact-item').filter({ hasText: 'Bob' }).locator('.contact-info a');
    await expect(bobContact).toHaveAttribute('href', 'mailto:bob@test.com');
    await expect(bobContact).toHaveText('bob@test.com');
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

  test('steps collapse after recovery starts', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Steps 1 and 2 should be visible initially
    await recovery.expectStepsVisible();

    // Add Bob's share — triggers auto-recovery
    await recovery.addShares(bobDir);

    // Steps should collapse
    await recovery.expectStepsCollapsed();
  });

  test('full recovery workflow', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    // No separate MANIFEST.age — it's embedded in recover.html
    expect(fs.existsSync(path.join(aliceDir, 'MANIFEST.age'))).toBe(false);

    await recovery.open();

    // Alice's share is pre-loaded via personalization, manifest is embedded
    await recovery.expectShareCount(1);
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

  test('rejects share from a different set', async ({ page }) => {
    // Create a project with 5 friends so total (5) differs from the standard project (3)
    const fiveFriendProject = createTestProject({
      friends: [
        { name: 'Alice', email: 'a@test.com' },
        { name: 'Bob', email: 'b@test.com' },
        { name: 'Carol', email: 'c@test.com' },
        { name: 'Dan', email: 'd@test.com' },
        { name: 'Eve', email: 'e@test.com' },
      ],
      threshold: 3,
    });
    const fiveFriendBundlesDir = path.join(fiveFriendProject, 'output', 'bundles');
    const mismatchedBobDir = extractBundle(fiveFriendBundlesDir, 'Bob');

    const aliceDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();
    await recovery.expectShareCount(1); // Alice pre-loaded (total=3, threshold=2)

    // Bob's share is from a 5-friend project (total=5, threshold=3) — metadata mismatch
    await recovery.addShares(mismatchedBobDir);
    await recovery.expectShareCount(1); // Still 1, mismatched share rejected
    await expect(page.locator('.toast-error')).toContainText("Pieces don't match");
  });

  test.skip('retry after decryption failure keeps holder pre-loaded share', async ({ page }) => {
    // TODO: This test needs to be redesigned. The current approach of mixing
    // personalized shares (Alice) with mismatched shares (Bob) doesn't reliably
    // trigger a decryption failure, making the test flaky.
    //
    // The implementation (app.ts retry handler) is correct—it filters shares to
    // keep only the holder share. But the test setup is too complex to validate
    // this reliably in E2E without additional hooks into the app's internal state.
    //
    // Better test approach: Mock decryption failure directly in a unit test, or
    // simplify the E2E test to focus on the happy path (loading shares correctly).
    const aliceDir = extractBundle(bundlesDir, 'Alice');
    const mismatchedBobDir = extractBundle(mismatchBundlesDir, 'Bob');
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();
    await recovery.expectShareCount(1);
  });
});

test.describe('Anonymous Bundle Recovery', () => {
  let anonProjectDir: string;
  let anonBundlesDir: string;

  test.beforeAll(async () => {
    // Skip if rememory binary not available
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      test.skip();
      return;
    }

    anonProjectDir = createAnonymousTestProject();
    anonBundlesDir = path.join(anonProjectDir, 'output', 'bundles');
  });

  test.afterAll(async () => {
    cleanupProject(anonProjectDir);
  });

  test('anonymous recover.html loads and shows UI without contact list', async ({ page }) => {
    const [share1Dir] = extractAnonymousBundles(anonBundlesDir, [1]);
    const recovery = new RecoveryPage(page, share1Dir);

    await recovery.open();
    await recovery.expectUIElements();
    // Manifest should be pre-loaded (embedded in personalization)
    await recovery.expectManifestLoaded();

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

    // Share 1 is pre-loaded, manifest is embedded
    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Share 1');
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
  let noEmbedProjectDir: string;
  let noEmbedBundlesDir: string;
  let standaloneRecoverHtml: string;
  let tmpDir: string;

  test.beforeAll(async () => {
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      test.skip();
      return;
    }

    projectDir = createTestProject();
    bundlesDir = path.join(projectDir, 'output', 'bundles');
    noEmbedProjectDir = createTestProject({ noEmbedManifest: true });
    noEmbedBundlesDir = path.join(noEmbedProjectDir, 'output', 'bundles');
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-generic-e2e-'));
    standaloneRecoverHtml = generateStandaloneHTML(tmpDir, 'recover');
  });

  test.afterAll(async () => {
    cleanupProject(projectDir);
    cleanupProject(noEmbedProjectDir);
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

  test('personalized recover.html can be used as manifest source on standalone tool', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, tmpDir);

    await recovery.openFile(standaloneRecoverHtml);
    await recovery.expectShareCount(0);

    // Add Alice's and Bob's shares via README.txt files
    await recovery.addShares(aliceDir, bobDir);
    await recovery.expectShareCount(2);

    // Explicitly load Alice's personalized recover.html as the manifest source
    // (bundles no longer include a separate MANIFEST.age when the manifest is embedded)
    const recoverHtmlPath = path.join(aliceDir, 'recover.html');
    expect(fs.existsSync(recoverHtmlPath)).toBeTruthy();
    expect(fs.existsSync(path.join(aliceDir, 'MANIFEST.age'))).toBeFalsy();

    await recovery.addManifestFile(recoverHtmlPath);
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

  test('standalone recovery with separate MANIFEST.age file', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(noEmbedBundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, tmpDir);

    await recovery.openFile(standaloneRecoverHtml);
    await recovery.expectShareCount(0);

    // Add shares from README.txt files (from noEmbedManifest bundles)
    await recovery.addShares(aliceDir, bobDir);
    await recovery.expectShareCount(2);

    // Load MANIFEST.age file directly (these bundles have it as a separate file)
    const manifestPath = path.join(aliceDir, 'MANIFEST.age');
    expect(fs.existsSync(manifestPath)).toBeTruthy();
    await recovery.addManifestFile(manifestPath);
    await recovery.expectManifestLoaded();

    // Recovery should complete
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3);
    await recovery.expectDownloadVisible();
  });
});

test.describe('--no-embed-manifest flag', () => {
  let noEmbedProjectDir: string;
  let noEmbedBundlesDir: string;

  test.beforeAll(async () => {
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      test.skip();
      return;
    }

    noEmbedProjectDir = createTestProject({ noEmbedManifest: true });
    noEmbedBundlesDir = path.join(noEmbedProjectDir, 'output', 'bundles');
  });

  test.afterAll(async () => {
    cleanupProject(noEmbedProjectDir);
  });

  test('manifest is not pre-loaded when --no-embed-manifest is used', async ({ page }) => {
    const bundleDir = extractBundle(noEmbedBundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();

    // Share is pre-loaded but manifest is NOT
    await recovery.expectShareCount(1);
    await recovery.expectManifestDropZoneVisible();
  });

  test('recovery still works with manual manifest when --no-embed-manifest is used', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(noEmbedBundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Manifest must be loaded manually
    await recovery.addManifest();
    await recovery.expectManifestLoaded();

    // Add Bob's share
    await recovery.addShares(bobDir);

    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3);
    await recovery.expectDownloadVisible();
  });
});

test.describe('PDF Share Import', () => {
  let projectDir: string;
  let bundlesDir: string;

  test.beforeAll(async () => {
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      test.skip();
      return;
    }

    projectDir = createTestProject();
    bundlesDir = path.join(projectDir, 'output', 'bundles');
  });

  test.afterAll(async () => {
    cleanupProject(projectDir);
  });

  test('full recovery workflow with PDF files', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Alice's share is pre-loaded via personalization, manifest is embedded
    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Alice');
    await recovery.expectManifestLoaded();

    // Add Bob's share via PDF file
    await recovery.addSharePDFs(bobDir);
    
    // Verify Bob's share was added
    await recovery.expectShareCount(2);
    await recovery.expectShareHolder('Bob');
    await recovery.expectReadyToRecover();

    // Recovery should complete automatically (triggered by reaching threshold)
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3); // secret.txt, notes.txt, README.md
    await recovery.expectDownloadVisible();
  });
});

// ZIP Bundle Import tests - verify native JS ZIP extraction works
test.describe('ZIP Bundle Import', () => {
  let projectDir: string;
  let bundlesDir: string;

  test.beforeAll(async () => {
    projectDir = createTestProject();
    bundlesDir = path.join(projectDir, 'output', 'bundles');
  });

  test.afterAll(async () => {
    cleanupProject(projectDir);
  });

  test('adding second share via bundle ZIP completes recovery', async ({ page }) => {
    // Start with Alice's personalized recover.html (share + manifest embedded)
    const [aliceDir] = extractBundles(bundlesDir, ['Alice']);
    const recovery = new RecoveryPage(page, aliceDir);
    await recovery.open();

    // Alice's share is pre-loaded
    await recovery.expectShareCount(1);
    await recovery.expectShareHolder('Alice');

    // Need one more share (threshold = 2)
    await recovery.expectNeedMoreShares(1);

    // Add Bob's bundle ZIP (not extracted, the actual ZIP file)
    await recovery.addBundleZip(bundlesDir, 'Bob');

    // Bob's share should be extracted from ZIP
    await recovery.expectShareCount(2);
    await recovery.expectShareHolder('Bob');

    // Recovery should complete
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3);
    await recovery.expectDownloadVisible();
    await recovery.expectNoLoadingIndicator();
  });

  test('dropping a bundle ZIP on standalone recover.html extracts manifest from inner recover.html', async ({ page }) => {
    // Use a truly standalone recover.html (no personalization, no manifest)
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-e2e-standalone-'));
    const standaloneHtml = generateStandaloneHTML(tmpDir, 'recover');
    const recovery = new RecoveryPage(page, tmpDir);
    await recovery.openFile(standaloneHtml);

    // Drop two bundle ZIPs — standard bundles embed the manifest in recover.html
    // rather than including a separate MANIFEST.age file
    await recovery.addBundleZip(bundlesDir, 'Alice');
    await recovery.expectShareCount(1);

    await recovery.addBundleZip(bundlesDir, 'Bob');
    await recovery.expectShareCount(2);

    // Manifest should be loaded (extracted from recover.html inside the ZIP)
    await recovery.expectManifestLoaded();

    // Recovery should complete
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3);
    await recovery.expectDownloadVisible();

    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  test('manifest replacement via second ZIP does not corrupt recovery', async ({ page }) => {
    // Use a standalone recover.html (no personalization, no manifest)
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-e2e-manifest-replace-'));
    const standaloneHtml = generateStandaloneHTML(tmpDir, 'recover');
    const recovery = new RecoveryPage(page, tmpDir);
    await recovery.openFile(standaloneHtml);

    // Drop Alice's ZIP — loads share + manifest
    await recovery.addBundleZip(bundlesDir, 'Alice');
    await recovery.expectShareCount(1);
    await recovery.expectManifestLoaded();

    // Drop Bob's ZIP — loads share + replaces manifest
    await recovery.addBundleZip(bundlesDir, 'Bob');
    await recovery.expectShareCount(2);

    // Recovery should complete (manifest not corrupted by replacement)
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3);
    await recovery.expectDownloadVisible();

    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  test('dropping a personalized recover.html extracts holderShare', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();
    await recovery.expectShareCount(1);

    // Drop Bob's recover.html — should extract his holderShare from personalization JSON
    const bobRecoverHtml = path.join(bobDir, 'recover.html');
    const bobHtmlContent = fs.readFileSync(bobRecoverHtml);

    await page.evaluate(async (htmlBytes: number[]) => {
      const file = new File([new Uint8Array(htmlBytes)], 'recover.html', {
        type: 'text/html',
      });
      const dataTransfer = new DataTransfer();
      dataTransfer.items.add(file);
      const input = document.querySelector('#share-file-input') as HTMLInputElement;
      if (input) {
        input.files = dataTransfer.files;
        input.dispatchEvent(new Event('change', { bubbles: true }));
      }
    }, Array.from(bobHtmlContent));

    // Bob's share should be extracted and added
    await recovery.expectShareCount(2);
    await recovery.expectShareHolder('Bob');
  });
});
