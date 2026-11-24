import { defineConfig, type Plugin } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

/** Keeps only modern .woff2 font faces in production bundles, dropping ~11 MB of legacy SVG/TTF/WOFF fonts. */
function pruneLegacyFontAssets(): Plugin {
  const legacyFontAssetRe = /(?:^|\/)(?:Phosphor[^/]*\.(?:svg|ttf|woff|eot)|poppins[^/]*\.(?:ttf|woff|eot))$/i;
  const legacySrcSegmentRe = /url\([^)]+\.(?:eot|woff|ttf|svg)(?:[^)]*)\)\s*format\([^)]+\)\s*,?\s*/gi;

  return {
    name: 'velora-prune-legacy-fonts',
    apply: 'build',
    generateBundle(_options, bundle) {
      for (const [fileName, asset] of Object.entries(bundle)) {
        if (asset.type === 'asset' && fileName.endsWith('.css') && typeof asset.source === 'string') {
          asset.source = asset.source
            .replace(/src:\s*url\([^)]+\.eot[^)]*\);\s*/gi, '')
            .replace(legacySrcSegmentRe, (m) => (m.toLowerCase().includes('.woff2') ? m : ''))
            .replace(/,\s*;/g, ';');
        } else if (legacyFontAssetRe.test(fileName)) {
          delete bundle[fileName];
        }
      }
    }
  };
}

export default defineConfig({
  plugins: [tailwindcss(), svelte(), pruneLegacyFontAssets()],
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
});
