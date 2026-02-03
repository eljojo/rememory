import { test } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import {
  getRememoryBin,
  createTestProject,
  extractBundle,
  extractBundles,
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
    await recovery.expectRecoverDisabled();
  });

  test('can add shares from README.txt files', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);
    const recovery = new RecoveryPage(page, aliceDir);

    await recovery.open();

    // Add Alice's share
    await recovery.addShares(aliceDir);
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

    // Add both shares at once
    await recovery.addShares(aliceDir, bobDir);
    await recovery.expectShareCount(2);

    // Add manifest
    await recovery.addManifest();
    await recovery.expectManifestLoaded();
    await recovery.expectRecoverEnabled();

    // Recover
    await recovery.recover();
    await recovery.expectRecoveryComplete();
    await recovery.expectFileCount(3); // secret.txt, notes.txt, README.md
    await recovery.expectDownloadVisible();
  });

  test('shows error with insufficient shares', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();

    // Add only one share (threshold is 2)
    await recovery.addShares(bundleDir);
    await recovery.expectNeedMoreShares(1);
    await recovery.expectRecoverDisabled();
  });

  test('detects duplicate shares', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');
    const recovery = new RecoveryPage(page, bundleDir);

    await recovery.open();
    recovery.onDialog('dismiss');

    // Add same share twice
    await recovery.addShares(bundleDir);
    await recovery.expectShareCount(1);

    await recovery.addShares(bundleDir);
    await recovery.expectShareCount(1); // Still 1, duplicate ignored
  });
});
