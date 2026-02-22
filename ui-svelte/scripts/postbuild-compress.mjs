import { promises as fs } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { brotliCompressSync, constants, gzipSync } from "node:zlib";

const here = path.dirname(fileURLToPath(import.meta.url));
const uiDir = path.resolve(here, "..");
const distDir = path.resolve(uiDir, "../proxy/ui_dist");

const minBytes = 1024;
const compressibleExt = new Set([
  ".css",
  ".html",
  ".js",
  ".json",
  ".map",
  ".mjs",
  ".svg",
  ".txt",
  ".webmanifest",
  ".xml",
]);

async function walkFiles(dir, out = []) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      await walkFiles(fullPath, out);
      continue;
    }
    out.push(fullPath);
  }
  return out;
}

function isCompressible(filePath) {
  if (filePath.endsWith(".gz") || filePath.endsWith(".br")) {
    return false;
  }
  return compressibleExt.has(path.extname(filePath).toLowerCase());
}

async function writeIfSmaller(filePath, original, compressed) {
  if (compressed.length >= original.length) {
    return;
  }
  await fs.writeFile(filePath, compressed);
}

async function compressFile(filePath) {
  if (!isCompressible(filePath)) {
    return;
  }

  const raw = await fs.readFile(filePath);
  if (raw.length < minBytes) {
    return;
  }

  const gzip = gzipSync(raw, { level: 9 });
  await writeIfSmaller(`${filePath}.gz`, raw, gzip);

  const brotli = brotliCompressSync(raw, {
    params: {
      [constants.BROTLI_PARAM_QUALITY]: 11,
      [constants.BROTLI_PARAM_MODE]: constants.BROTLI_MODE_TEXT,
    },
  });
  await writeIfSmaller(`${filePath}.br`, raw, brotli);
}

async function main() {
  const files = await walkFiles(distDir);
  for (const filePath of files) {
    await compressFile(filePath);
  }
}

main().catch((error) => {
  console.error("postbuild compression failed:", error);
  process.exitCode = 1;
});
