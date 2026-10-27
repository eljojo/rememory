import { test, expect } from './fixtures';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { execSync } from 'child_process';

// Time-locked bundles fetch a drand beacon and check its BLS signature
// before using it. These two quicknet beacons are real (rounds 1000000 and
// 1000001, from api.drand.sh); the chain's public key is read from the Go
// source so it can't drift from what bundles embed.
const BEACON = {
  round: 1000000,
  randomness: 'b22aad4794f7451896f7a371aa46106fd84d919f3f569acd5b2fddf1d1440af3',
  signature: '83ad29e4c409f9470fc2ef02f90214df49e02b441a1a241a82d622d9f608ef98fd8b11a029f1bee9d9e83b45088abe72',
};
const NEXT = {
  round: 1000001,
  randomness: '9f45f439afd81e9846b3b4dc5e3e6051922c73c8459d18e9d507b52ddbd884ff',
  signature: 'a5bd91e5e2d8c0bf51bffdfad87eef34348fd9c0b2df2bee39db90bdef7e1399b1a77bb2fe98b24d84c0936a306c4218',
};

test.describe('drand beacon verification @cross-browser', () => {
  let bundlePath: string;
  let publicKey: string;

  test.beforeAll(async () => {
    const goSrc = fs.readFileSync('internal/core/tlock_common.go', 'utf8');
    publicKey = goSrc.match(/QuicknetPublicKey\s*=\s*"([0-9a-f]+)"/)![1];

    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-beacon-'));
    const entry = path.join(dir, 'entry.ts');
    fs.writeFileSync(entry, `export { verifyBeacon } from ${JSON.stringify(path.resolve('internal/html/assets/src/tlock.ts'))};\n`);
    bundlePath = path.join(dir, 'beacon.js');
    execSync(`npx esbuild ${entry} --bundle --format=iife --global-name=BeaconTest --outfile=${bundlePath}`);
  });

  async function verify(page: any, beacon: object): Promise<string> {
    await page.goto('about:blank');
    await page.addScriptTag({ path: bundlePath });
    return page.evaluate(({ beacon, publicKey }: { beacon: object; publicKey: string }) => {
      try {
        (window as any).BeaconTest.verifyBeacon({ publicKey }, beacon);
        return 'ok';
      } catch (err) {
        return (err as Error).message;
      }
    }, { beacon, publicKey });
  }

  test('a genuine beacon is accepted', async ({ page }) => {
    expect(await verify(page, BEACON)).toBe('ok');
  });

  test('a replayed beacon is refused by the signature check', async ({ page }) => {
    // Another round's genuine signature and randomness, relabelled: the
    // randomness still matches its signature, so only BLS can catch it.
    const replayed = { ...NEXT, round: BEACON.round };
    expect(await verify(page, replayed)).toContain('BLS signature verification failed');
  });

  test('a forged signature is refused by the signature check', async ({ page }) => {
    const bytes = BEACON.signature.match(/../g)!.map(h => parseInt(h, 16));
    bytes[bytes.length - 1] ^= 1;
    const signature = bytes.map(b => b.toString(16).padStart(2, '0')).join('');
    const randomness = require('crypto').createHash('sha256').update(Buffer.from(bytes)).digest('hex');
    const result = await verify(page, { round: BEACON.round, signature, randomness });
    expect(result).not.toBe('ok');
  });
});
