import { defineConfig } from 'vite';
export default defineConfig({
  base: '/webclient/',
  publicDir: 'static',
  build: {outDir: '../resources/webclient', emptyOutDir: true, rollupOptions: {output: {entryFileNames: 'session.js', chunkFileNames: '[name].js', assetFileNames: '[name].[ext]'}}},
});
