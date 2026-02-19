// Tar.gz extraction using tarparser
import { parseTar } from 'tarparser';
// ZIP creation using fflate
import { zipSync } from 'fflate';

export interface ExtractedFile {
  name: string;
  data: Uint8Array;
}

/**
 * Extract files from a tar.gz archive.
 */
export async function extractTarGz(data: Uint8Array): Promise<ExtractedFile[]> {
  const entries = await parseTar(data);

  return entries
    .filter(entry => entry.type === 'file')
    .map(entry => ({
      name: entry.name,
      data: entry.data,
    }));
}

/**
 * Create a ZIP archive from an array of files.
 */
export function createZip(files: ExtractedFile[]): Uint8Array<ArrayBuffer> {
  const input: Record<string, Uint8Array> = {};
  for (const f of files) {
    input[f.name] = f.data;
  }
  // zipSync always returns a Uint8Array backed by a plain ArrayBuffer at runtime
  return zipSync(input) as unknown as Uint8Array<ArrayBuffer>;
}
