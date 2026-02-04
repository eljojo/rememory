import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import {
  getRememoryBin,
  CreationPage,
  generateStandaloneHTML
} from './helpers';

test.describe('Browser Bundle Creation Tool', () => {
  let htmlPath: string;
  let tmpDir: string;

  test.beforeAll(async () => {
    // Skip if rememory binary not available
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      console.log(`Skipping tests: rememory binary not found at ${bin}`);
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

  test('validates required fields', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Generate button should be disabled without required fields
    await creation.expectGenerateDisabled();

    // Fill in first friend
    await creation.setFriend(0, 'Alice', 'alice@test.com');
    await creation.expectGenerateDisabled(); // Still disabled - second friend empty

    // Fill in second friend
    await creation.setFriend(1, 'Bob', 'bob@test.com');
    await creation.expectGenerateDisabled(); // Still disabled - no files
  });

  test('can import contacts from YAML', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    const yamlContent = `
name: imported-project
threshold: 2
friends:
  - name: Charlie
    email: charlie@test.com
    phone: "555-1234"
  - name: Diana
    email: diana@test.com
  - name: Eve
    email: eve@test.com
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

  test('full bundle creation workflow', async ({ page }, testInfo) => {
    // Increase timeout for WASM-heavy operations (especially Firefox)
    testInfo.setTimeout(120000);
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Fill in friends
    await creation.setFriend(0, 'Alice', 'alice@test.com', '555-1111');
    await creation.setFriend(1, 'Bob', 'bob@test.com');

    // Add a third friend
    await creation.addFriend();
    await creation.setFriend(2, 'Carol', 'carol@test.com');

    // Set threshold to 2
    await creation.setThreshold(2);

    // Add test files
    const testFiles = creation.createTestFiles(tmpDir);
    await creation.addFiles(testFiles);

    // Generate button should now be enabled
    await creation.expectGenerateEnabled();

    // Generate bundles
    await creation.generate();

    // Should show completion
    await creation.expectGenerationComplete();

    // Should show bundles for each friend
    await creation.expectBundleCount(3);
    await creation.expectBundleFor('Alice');
    await creation.expectBundleFor('Bob');
    await creation.expectBundleFor('Carol');
  });

  test('generated bundles are valid', async ({ page }, testInfo) => {
    // Increase timeout for WASM-heavy operations (especially Firefox)
    testInfo.setTimeout(120000);
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Quick setup
    await creation.setFriend(0, 'Alice', 'alice@test.com');
    await creation.setFriend(1, 'Bob', 'bob@test.com');

    const testFiles = creation.createTestFiles(tmpDir);
    await creation.addFiles(testFiles);

    // Generate bundles
    await creation.generate();
    await creation.expectGenerationComplete();

    // Download a bundle and verify it can be opened
    const bundleData = await creation.downloadBundle(0);
    expect(bundleData).toBeTruthy();
    expect(bundleData.length).toBeGreaterThan(1000); // Should be substantial
  });

  test('language switching works', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Default should be English
    await creation.expectTitle('ReMemory - Create Bundles');

    // Switch to Spanish
    await creation.setLanguage('es');
    await creation.expectTitle('ReMemory - Guarda lo Importante');

    // Switch to German
    await creation.setLanguage('de');
    await creation.expectTitle('ReMemory — Bewahre das Wichtige');

    // Switch back to English
    await creation.setLanguage('en');
    await creation.expectTitle('ReMemory - Create Bundles');
  });

  test('minimum 2 friends required', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);

    await creation.open();
    creation.onDialog('dismiss');

    // Should start with 2 friends
    await creation.expectFriendCount(2);

    // Try to remove a friend - should fail (can't go below 2)
    await creation.removeFriend(1);

    // Should still have 2 friends
    await creation.expectFriendCount(2);
  });
});
