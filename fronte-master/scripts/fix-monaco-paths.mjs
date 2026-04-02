import fs from 'node:fs';
import path from 'node:path';

const distDir = path.resolve('dist');
const wrongDir = path.join(distDir, 'web', 'monacoeditorwork');
const rightDir = path.join(distDir, 'monacoeditorwork');

if (fs.existsSync(wrongDir)) {
  fs.mkdirSync(rightDir, { recursive: true });
  for (const entry of fs.readdirSync(wrongDir)) {
    fs.copyFileSync(path.join(wrongDir, entry), path.join(rightDir, entry));
  }
  fs.rmSync(path.join(distDir, 'web'), { recursive: true, force: true });
  console.log('[OK] moved Monaco workers to dist/monacoeditorwork');
} else {
  console.log('[OK] no Monaco worker relocation needed');
}
