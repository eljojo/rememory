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

  test('full bundle creation workflow', async ({ page }, testInfo) => {
    // Increase timeout for WASM-heavy operations (especially Firefox)
    testInfo.setTimeout(120000);
    const creation = new CreationPage(page, htmlPath);

    await creation.open();

    // Fill in friends
    await creation.setFriend(0, 'Alice', 'alice@test.com');
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

    // Quick setup - name is the only required field, contact is optional
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
});
