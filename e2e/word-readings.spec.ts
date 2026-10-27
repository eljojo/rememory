import { test, expect } from './fixtures';
import * as fs from 'fs';
import { getCryptoTestHtml } from './helpers';

const v2Golden = JSON.parse(fs.readFileSync('internal/core/testdata/v2-golden.json', 'utf8'));

// Recovery words that fit more than one word list with a valid checksum.
test.describe('Ambiguous recovery words @cross-browser', () => {
  let testHtmlPath: string;

  test.beforeAll(async () => {
    testHtmlPath = getCryptoTestHtml();
  });

  // A v0.0.19 Turkish grid whose words all exist in today's Turkish list too,
  // and whose checksum also passes there: two readings, only one right.
  // Found by searching share data; the right reading must be kept.
  test('ambiguous v0.0.19 Turkish words keep every reading', async ({ page }) => {
    await page.goto('file://' + testHtmlPath);
    await page.waitForFunction(() => (window as any).testReady);

    const words = 'taptaze jikle gilaburu tekdir palmitik mucuk ovulma fener ksilofon dine yuha majeste jandarma heykel derlem figan francala buhar icmal laisizm nevton tecelli sarf affolma ruzname'.split(' ');
    const readings = await page.evaluate(async (words: string[]) => {
      const decoded = await (window as any).rememoryCrypto.decodeShareWords(words);
      const hex = (d: Uint8Array) => Array.from(d).map(b => b.toString(16).padStart(2, '0')).join('');
      return [decoded, ...(decoded.alternatives ?? [])].map((r: any) => ({ index: r.index, dataHex: hex(r.data) }));
    }, words);

    expect(readings.length).toBeGreaterThan(1);
    expect(readings).toContainEqual({
      index: 12,
      dataHex: 'dff0e19a71cbdf5aaf2ae194682fead06862e40f82ed614639de4c7b41c3f4682a',
    });
  });

  // Unlocking tries each piece's other readings when the most likely ones
  // don't open the archive.
  test('unlocking tries alternative readings until one opens', async ({ page }) => {
    await page.goto('file://' + testHtmlPath);
    await page.waitForFunction(() => (window as any).testReady);

    const manifest = Array.from(fs.readFileSync('internal/core/testdata/v2-bundle/MANIFEST.age'));
    const shares = v2Golden.shares.slice(0, 3).map((s: any) => s.data_hex);

    const result = await page.evaluate(async ({ manifest, shares, version }: { manifest: number[]; shares: string[]; version: number }) => {
      const crypto = (window as any).rememoryCrypto;
      const bytes = (hex: string) => new Uint8Array(hex.match(/../g)!.map(h => parseInt(h, 16)));
      const real = shares.map(bytes);
      const wrong = real[0].slice();
      wrong[0] ^= 0xff; // a reading that passed its checksum but isn't the piece
      try {
        const archive = await crypto.unlockWithCandidates(new Uint8Array(manifest), version, [[wrong, real[0]], [real[1]], [real[2]]]);
        const files = await crypto.extractArchive(archive);
        return { files: files.map((f: { name: string }) => f.name).sort() };
      } catch (err) {
        return { error: (err as Error).message };
      }
    }, { manifest, shares, version: v2Golden.version });

    expect(result.error).toBeUndefined();
    expect(result.files).toEqual(Object.keys(v2Golden.manifest.files).sort());
  });
});
