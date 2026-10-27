import { test, expect } from './fixtures';
import * as fs from 'fs';
import * as path from 'path';
import { getRememoryBin, createTestProject, cleanupProject, extractBundle, getCryptoTestHtml } from './helpers';

// Load golden fixtures
const v1Golden = JSON.parse(
  fs.readFileSync('internal/core/testdata/v1-golden.json', 'utf8')
);
const v2Golden = JSON.parse(
  fs.readFileSync('internal/core/testdata/v2-golden.json', 'utf8')
);

const goldenFixtures = [
  { name: 'v1', data: v1Golden, manifestPath: 'internal/core/testdata/v1-bundle/MANIFEST.age' },
  { name: 'v2', data: v2Golden, manifestPath: 'internal/core/testdata/v2-bundle/MANIFEST.age' },
];

test.describe('Golden Crypto Compatibility @cross-browser', () => {
  let testHtmlPath: string;

  test.beforeAll(async () => {
    testHtmlPath = getCryptoTestHtml();
  });

  // Full decrypt flow for both versions
  for (const { name, data, manifestPath } of goldenFixtures) {
    test(`${name}: full decrypt flow matches expected output`, async ({ page }) => {
      await page.goto('file://' + testHtmlPath);
      await page.waitForFunction(() => (window as any).testReady);

      const manifestBytes = Array.from(fs.readFileSync(manifestPath));
      const shares = data.shares.slice(0, 3).map((s: any) => ({
        dataHex: s.data_hex,
      }));

      const result = await page.evaluate(
        async ({ shares, manifestBytes, version }: { shares: any[]; manifestBytes: number[]; version: number }) => {
          const crypto = (window as any).rememoryCrypto;

          // Convert hex shares to Uint8Array
          const shareBytes = shares.map(s => {
            const hex = s.dataHex;
            const bytes = new Uint8Array(hex.length / 2);
            for (let i = 0; i < hex.length; i += 2) {
              bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
            }
            return bytes;
          });

          // Combine shares
          const recovered = await crypto.combine(shareBytes);
          const passphrase = crypto.recoverPassphrase(recovered, version);

          // Decrypt manifest
          const manifest = new Uint8Array(manifestBytes);
          const decrypted = await crypto.decrypt(manifest, passphrase);

          // Extract archive (auto-detects ZIP or tar.gz)
          const files = await crypto.extractArchive(decrypted);

          const result: Record<string, string> = {};
          for (const file of files) {
            result[file.name] = new TextDecoder().decode(file.data);
          }
          return { files: result };
        },
        { shares, manifestBytes, version: data.version }
      );

      expect(result.files).toBeDefined();

      for (const [name, content] of Object.entries(data.manifest.files)) {
        expect(result.files[name]).toBe(content);
      }
    });
  }

  // PEM parsing for both versions
  for (const { name, data } of goldenFixtures) {
    test(`${name}: parse PEM share correctly`, async ({ page }) => {
      await page.goto('file://' + testHtmlPath);
      await page.waitForFunction(() => (window as any).testReady);

      const share = data.shares[0];

      const result = await page.evaluate(async (pem: string) => {
        const crypto = (window as any).rememoryCrypto;
        try {
          const parsed = await crypto.parseShare(pem);
          return {
            version: parsed.version,
            index: parsed.index,
            threshold: parsed.threshold,
            total: parsed.total,
          };
        } catch (err) {
          return { error: (err as Error).message };
        }
      }, share.pem);

      expect(result.error).toBeUndefined();
      expect(result.version).toBe(data.version);
      expect(result.index).toBe(share.index);
      expect(result.threshold).toBe(data.threshold);
      expect(result.total).toBe(data.total);
    });
  }

  // v2-specific: BIP39 word decoding
  test('v2: decode 25 BIP39 words correctly', async ({ page }) => {
    await page.goto('file://' + testHtmlPath);
    await page.waitForFunction(() => (window as any).testReady);

    for (const share of v2Golden.shares) {
      const words = share.words.split(' ');
      expect(words.length).toBe(25);

      const result = await page.evaluate(async (words: string[]) => {
        const crypto = (window as any).rememoryCrypto;
        try {
          const decoded = await crypto.decodeShareWords(words);
          const hex = Array.from(decoded.data as Uint8Array)
            .map((b: number) => b.toString(16).padStart(2, '0'))
            .join('');
          return { index: decoded.index, dataHex: hex };
        } catch (err) {
          return { error: (err as Error).message };
        }
      }, words);

      expect(result.error).toBeUndefined();
      expect(result.index).toBe(share.index);
      expect(result.dataHex).toBe(share.data_hex);
    }
  });

  // v2-specific: compact share parsing
  test('v2: parse compact share correctly', async ({ page }) => {
    await page.goto('file://' + testHtmlPath);
    await page.waitForFunction(() => (window as any).testReady);

    const share = v2Golden.shares[0];

    const result = await page.evaluate(async (compact: string) => {
      const crypto = (window as any).rememoryCrypto;
      try {
        const parsed = await crypto.parseCompactShare(compact);
        return {
          version: parsed.version,
          index: parsed.index,
          threshold: parsed.threshold,
          total: parsed.total,
        };
      } catch (err) {
        return { error: (err as Error).message };
      }
    }, share.compact);

    expect(result.error).toBeUndefined();
    expect(result.version).toBe(v2Golden.version);
    expect(result.index).toBe(share.index);
    expect(result.threshold).toBe(v2Golden.threshold);
    expect(result.total).toBe(v2Golden.total);
  });

  // Security: below-threshold shares should not recover correct passphrase
  test('below-threshold shares do not recover correct passphrase', async ({ page }) => {
    await page.goto('file://' + testHtmlPath);
    await page.waitForFunction(() => (window as any).testReady);

    // v2 has threshold=3, try with just 2 shares
    const shares = v2Golden.shares.slice(0, 2).map((s: any) => s.data_hex);

    const result = await page.evaluate(async ({ shares, version, expectedPassphrase }: { shares: string[]; version: number; expectedPassphrase: string }) => {
      const crypto = (window as any).rememoryCrypto;

      // Convert hex shares to Uint8Array
      const shareBytes = shares.map(hex => {
        const bytes = new Uint8Array(hex.length / 2);
        for (let i = 0; i < hex.length; i += 2) {
          bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
        }
        return bytes;
      });

      try {
        const recovered = await crypto.combine(shareBytes);
        const passphrase = crypto.recoverPassphrase(recovered, version);
        return { passphrase, matchesExpected: passphrase === expectedPassphrase };
      } catch (err) {
        return { error: (err as Error).message };
      }
    }, { shares, version: v2Golden.version, expectedPassphrase: v2Golden.passphrase });

    // Either errors or produces wrong passphrase
    if (!result.error) {
      expect(result.matchesExpected).toBe(false);
    }
  });

  // Security: a share with x-coordinate 0 would make interpolation return its
  // y-values verbatim, letting whoever crafted it choose the passphrase.
  test('share with x-coordinate 0 is rejected', async ({ page }) => {
    await page.goto('file://' + testHtmlPath);
    await page.waitForFunction(() => (window as any).testReady);

    const genuine = v2Golden.shares.slice(0, 2).map((s: any) => s.data_hex);

    const result = await page.evaluate(async ({ genuine }: { genuine: string[] }) => {
      const crypto = (window as any).rememoryCrypto;

      const shareBytes = genuine.map(hex => {
        const bytes = new Uint8Array(hex.length / 2);
        for (let i = 0; i < hex.length; i += 2) {
          bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
        }
        return bytes;
      });
      const crafted = new Uint8Array(shareBytes[0].length).fill(0x41);
      crafted[crafted.length - 1] = 0;

      try {
        const recovered = await crypto.combine([...shareBytes, crafted]);
        return { recovered: Array.from(recovered as Uint8Array) };
      } catch (err) {
        return { error: (err as Error).message };
      }
    }, { genuine });

    expect(result.recovered).toBeUndefined();
    expect(result.error).toBeTruthy();
  });
});

test.describe('extractBundle from recover.html', () => {
  let testHtmlPath: string;
  let projectDir: string;
  let bundlesDir: string;

  test.beforeAll(async () => {
    const bin = getRememoryBin();
    if (!fs.existsSync(bin)) {
      test.skip();
      return;
    }

    testHtmlPath = getCryptoTestHtml();
    projectDir = createTestProject();
    bundlesDir = path.join(projectDir, 'output', 'bundles');
  });

  test.afterAll(async () => {
    cleanupProject(projectDir);
  });

  test('extractBundle extracts holderShare from personalized recover.html', async ({ page }) => {
    const aliceDir = extractBundle(bundlesDir, 'Alice');
    await page.goto('file://' + testHtmlPath);
    await page.waitForFunction(() => (window as any).testReady);

    const recoverHtmlContent = fs.readFileSync(
      path.join(aliceDir, 'recover.html')
    );

    const result = await page.evaluate(async (htmlBytes: number[]) => {
      const crypto = (window as any).rememoryCrypto;
      try {
        const data = new Uint8Array(htmlBytes);
        const bundle = await crypto.extractBundle(data);

        return {
          hasShare: !!bundle.share || !!bundle.holderShare,
          hasManifest: !!bundle.manifest,
          shareHolder: bundle.holder,
          shareIndex: bundle.index,
        };
      } catch (err) {
        return { error: (err as Error).message };
      }
    }, Array.from(recoverHtmlContent));

    expect(result.error).toBeUndefined();
    expect(result.hasShare).toBe(true);
    expect(result.shareHolder).toBe('Alice');
    expect(result.shareIndex).toBe(1);
  });
});
