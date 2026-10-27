import { test, expect } from './fixtures';
import * as fs from 'fs';
import * as path from 'path';
import { getRememoryBin, createTestProject, cleanupProject, extractBundle, findReadmeFile, RecoveryPage } from './helpers';

test.describe('Duplicate pieces', () => {
  let current: string;
  let older: string;

  test.beforeAll(async () => {
    if (!fs.existsSync(getRememoryBin())) {
      test.skip();
      return;
    }
    current = createTestProject();
    // The same friends sealed separately: a friend's piece from an older
    // seal has the same index but an unrelated x-coordinate.
    older = createTestProject({
      friends: [
        { name: 'Alice', email: 'alice@test.com' },
        { name: 'Bob', email: 'bob@test.com' },
        { name: 'Carol', email: 'carol@test.com' },
      ],
      threshold: 2,
    });
  });

  test.afterAll(async () => {
    cleanupProject(current);
    cleanupProject(older);
  });

  test("a friend's piece from another seal is refused as a duplicate", async ({ page }) => {
    const aliceDir = extractBundle(path.join(current, 'output', 'bundles'), 'Alice');
    const olderAliceDir = extractBundle(path.join(older, 'output', 'bundles'), 'Alice');

    const recovery = new RecoveryPage(page, aliceDir);
    await recovery.open();
    await recovery.expectShareCount(1);

    await page.locator('#share-file-input').setInputFiles(findReadmeFile(olderAliceDir));

    await expect(page.locator('.toast-warning')).toBeVisible();
    await recovery.expectShareCount(1);
  });
});
