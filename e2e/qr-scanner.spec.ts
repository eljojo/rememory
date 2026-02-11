import { test, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import {
  getRememoryBin,
  createTestProject,
  extractBundle,
  extractBundles,
  findReadmeFile,
  RecoveryPage
} from './helpers';

test.describe('QR Scanner', () => {
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
    if (projectDir && fs.existsSync(projectDir)) {
      fs.rmSync(projectDir, { recursive: true, force: true });
    }
  });

  test('scan button is visible when BarcodeDetector is available', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');

    // Mock BarcodeDetector before page loads
    await page.addInitScript(() => {
      (window as any).BarcodeDetector = class {
        constructor() {}
        async detect() { return []; }
        static async getSupportedFormats() { return ['qr_code']; }
      };
    });

    const recovery = new RecoveryPage(page, bundleDir);
    await recovery.open();

    await expect(page.locator('#scan-qr-btn')).toBeVisible();
  });

  test('scan button is hidden when BarcodeDetector is not available', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');

    // Ensure BarcodeDetector is NOT defined (default for most test environments)
    await page.addInitScript(() => {
      delete (window as any).BarcodeDetector;
    });

    const recovery = new RecoveryPage(page, bundleDir);
    await recovery.open();

    await expect(page.locator('#scan-qr-btn')).not.toBeVisible();
  });

  test('clicking scan opens modal and close button dismisses it', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');

    await page.addInitScript(() => {
      (window as any).BarcodeDetector = class {
        constructor() {}
        async detect() { return []; }
        static async getSupportedFormats() { return ['qr_code']; }
      };

      navigator.mediaDevices.getUserMedia = async () => {
        const canvas = document.createElement('canvas');
        canvas.width = 640;
        canvas.height = 480;
        const ctx = canvas.getContext('2d')!;
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, 640, 480);
        return canvas.captureStream(1);
      };
    });

    const recovery = new RecoveryPage(page, bundleDir);
    await recovery.open();

    // Modal should be hidden initially
    await expect(page.locator('#qr-scanner-modal')).not.toBeVisible();

    // Click scan button
    await page.locator('#scan-qr-btn').click();

    // Modal should be visible
    await expect(page.locator('#qr-scanner-modal')).toBeVisible();

    // Close button should dismiss modal
    await page.locator('#qr-scanner-close').click();
    await expect(page.locator('#qr-scanner-modal')).not.toBeVisible();
  });

  test('scanning a compact share adds it to the shares list', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);

    const recovery = new RecoveryPage(page, aliceDir);

    // Read Bob's PEM share
    const bobReadme = fs.readFileSync(findReadmeFile(bobDir), 'utf8');
    const pemMatch = bobReadme.match(
      /-----BEGIN REMEMORY SHARE-----([\s\S]*?)-----END REMEMORY SHARE-----/
    );
    if (!pemMatch) throw new Error('No PEM share found');
    const bobPemShare = pemMatch[0];

    // Mock BarcodeDetector + getUserMedia with a real canvas-based video stream
    await page.addInitScript(() => {
      let detectCallCount = 0;
      let compactShare = '';

      (window as any).__qrTestSetCompact = (compact: string) => {
        compactShare = compact;
      };

      (window as any).BarcodeDetector = class {
        constructor() {}
        async detect() {
          detectCallCount++;
          if (compactShare && detectCallCount > 3) {
            return [{ rawValue: compactShare, format: 'qr_code', boundingBox: {}, cornerPoints: [] }];
          }
          return [];
        }
        static async getSupportedFormats() { return ['qr_code']; }
      };

      // Use a real canvas capture stream so the video element gets readyState >= 2
      navigator.mediaDevices.getUserMedia = async () => {
        const canvas = document.createElement('canvas');
        canvas.width = 640;
        canvas.height = 480;
        const ctx = canvas.getContext('2d')!;
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, 640, 480);
        return canvas.captureStream(1);
      };
    });

    await recovery.open();
    await recovery.expectShareCount(1);

    // Convert Bob's PEM share to compact format via WASM
    const compactShare = await page.evaluate((pem: string) => {
      const result = (window as any).rememoryParseShare(pem);
      if (result.error || !result.share) return '';
      return result.share.compact;
    }, bobPemShare);

    expect(compactShare).toMatch(/^RM\d+:\d+:\d+:\d+:[A-Za-z0-9_-]+:[0-9a-f]{4}$/);

    // Verify the compact share parses correctly
    const parseResult = await page.evaluate((compact: string) => {
      return (window as any).rememoryParseCompactShare(compact);
    }, compactShare);
    expect(parseResult.error).toBeFalsy();

    // Set the compact share for the mock BarcodeDetector to "find"
    await page.evaluate((compact: string) => {
      (window as any).__qrTestSetCompact(compact);
    }, compactShare);

    // Open scanner
    await page.locator('#scan-qr-btn').click();
    await expect(page.locator('#qr-scanner-modal')).toBeVisible();

    // Wait for the share to be detected and added
    await recovery.expectShareCount(2);

    // Modal should close after successful scan
    await expect(page.locator('#qr-scanner-modal')).not.toBeVisible();
  });

  test('scanning a URL with fragment adds the share', async ({ page }) => {
    const [aliceDir, bobDir] = extractBundles(bundlesDir, ['Alice', 'Bob']);

    const bobReadme = fs.readFileSync(findReadmeFile(bobDir), 'utf8');
    const pemMatch = bobReadme.match(
      /-----BEGIN REMEMORY SHARE-----([\s\S]*?)-----END REMEMORY SHARE-----/
    );
    if (!pemMatch) throw new Error('No PEM share found');

    await page.addInitScript(() => {
      let compactShare = '';

      (window as any).__qrTestSetCompact = (compact: string) => {
        compactShare = compact;
      };

      let detectCallCount = 0;
      (window as any).BarcodeDetector = class {
        constructor() {}
        async detect() {
          detectCallCount++;
          if (compactShare && detectCallCount > 3) {
            // Return as a URL with fragment, like the QR code from a PDF would contain
            const url = `https://eljojo.github.io/rememory/recover.html#share=${encodeURIComponent(compactShare)}`;
            return [{ rawValue: url, format: 'qr_code', boundingBox: {}, cornerPoints: [] }];
          }
          return [];
        }
        static async getSupportedFormats() { return ['qr_code']; }
      };

      // Use a real canvas capture stream so the video element gets readyState >= 2
      navigator.mediaDevices.getUserMedia = async () => {
        const canvas = document.createElement('canvas');
        canvas.width = 640;
        canvas.height = 480;
        const ctx = canvas.getContext('2d')!;
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, 640, 480);
        return canvas.captureStream(1);
      };
    });

    const recovery = new RecoveryPage(page, aliceDir);
    await recovery.open();

    // Convert PEM share to compact format via WASM
    const compactShare = await page.evaluate((pem: string) => {
      const result = (window as any).rememoryParseShare(pem);
      if (result.error || !result.share) return '';
      return result.share.compact;
    }, pemMatch[0]);

    await page.evaluate((compact: string) => {
      (window as any).__qrTestSetCompact(compact);
    }, compactShare);

    await page.locator('#scan-qr-btn').click();

    // Should detect the URL, extract the fragment, and add the share
    await recovery.expectShareCount(2);
    await expect(page.locator('#qr-scanner-modal')).not.toBeVisible();
  });

  test('camera permission denied shows error and closes modal', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');

    await page.addInitScript(() => {
      (window as any).BarcodeDetector = class {
        constructor() {}
        async detect() { return []; }
        static async getSupportedFormats() { return ['qr_code']; }
      };

      // Mock getUserMedia to reject (permission denied)
      navigator.mediaDevices.getUserMedia = async () => {
        throw new DOMException('Permission denied', 'NotAllowedError');
      };
    });

    const recovery = new RecoveryPage(page, bundleDir);
    await recovery.open();

    await page.locator('#scan-qr-btn').click();

    // Modal should close after error
    await expect(page.locator('#qr-scanner-modal')).not.toBeVisible();

    // A toast warning should appear
    await expect(page.locator('.toast')).toBeVisible();
  });

  test('camera tracks are stopped when modal is closed', async ({ page }) => {
    const bundleDir = extractBundle(bundlesDir, 'Alice');

    await page.addInitScript(() => {
      (window as any).__qrTestTrackStopped = false;

      (window as any).BarcodeDetector = class {
        constructor() {}
        async detect() { return []; }
        static async getSupportedFormats() { return ['qr_code']; }
      };

      // Use canvas capture stream but wrap tracks to detect stop()
      const origGetUserMedia = navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);
      navigator.mediaDevices.getUserMedia = async () => {
        const canvas = document.createElement('canvas');
        canvas.width = 640;
        canvas.height = 480;
        const ctx = canvas.getContext('2d')!;
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, 640, 480);
        const stream = canvas.captureStream(1);

        // Wrap track.stop() to detect when it's called
        for (const track of stream.getTracks()) {
          const origStop = track.stop.bind(track);
          track.stop = () => {
            (window as any).__qrTestTrackStopped = true;
            origStop();
          };
        }
        return stream;
      };
    });

    const recovery = new RecoveryPage(page, bundleDir);
    await recovery.open();

    // Open scanner
    await page.locator('#scan-qr-btn').click();
    await expect(page.locator('#qr-scanner-modal')).toBeVisible();

    // Verify track not yet stopped
    let stopped = await page.evaluate(() => (window as any).__qrTestTrackStopped);
    expect(stopped).toBe(false);

    // Close scanner
    await page.locator('#qr-scanner-close').click();

    // Verify track was stopped
    stopped = await page.evaluate(() => (window as any).__qrTestTrackStopped);
    expect(stopped).toBe(true);
  });
});
