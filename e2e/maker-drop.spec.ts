import { test, expect } from './fixtures';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { getRememoryBin, generateStandaloneHTML, CreationPage } from './helpers';

test.describe('Dropping files into maker.html', () => {
  let tmpDir: string;
  let htmlPath: string;

  test.beforeAll(async () => {
    if (!fs.existsSync(getRememoryBin())) {
      test.skip();
      return;
    }
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rememory-drop-e2e-'));
    htmlPath = generateStandaloneHTML(tmpDir, 'create');
  });

  test.afterAll(async () => {
    if (tmpDir && fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  // A browser empties a drop's item list once the event has been dispatched,
  // so every item has to be taken out before the handler first awaits.
  test('dropping several folders at once keeps all of them', async ({ page }) => {
    const creation = new CreationPage(page, htmlPath);
    await creation.open();

    await page.evaluate(() => {
      const fileEntry = (name: string) => ({
        isFile: true, isDirectory: false, name,
        file: (cb: (f: File) => void) => cb(new File([name], name)),
      });
      const folder = (name: string, files: string[]) => ({
        isFile: false, isDirectory: true, name,
        createReader: () => {
          let done = false;
          return {
            readEntries: (cb: (entries: unknown[]) => void) => {
              const batch = done ? [] : files.map(fileEntry);
              done = true;
              setTimeout(() => cb(batch));
            },
          };
        },
      });
      const items: any = {
        length: 2,
        0: { kind: 'file', webkitGetAsEntry: () => folder('a', ['a1.txt', 'a2.txt', 'a3.txt']) },
        1: { kind: 'file', webkitGetAsEntry: () => folder('b', ['b1.txt', 'b2.txt', 'b3.txt']) },
      };
      const drop = new Event('drop', { bubbles: true, cancelable: true });
      Object.defineProperty(drop, 'dataTransfer', { value: { items } });
      document.getElementById('files-drop-zone')!.dispatchEvent(drop);
      // What the browser does once dispatch returns.
      items.length = 0;
      delete items[0];
      delete items[1];
    });

    await creation.expectFileCount(6);
  });
});
