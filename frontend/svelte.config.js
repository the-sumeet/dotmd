import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  preprocess: vitePreprocess(),
  kit: {
    // Output to dist/ so Go's `//go:embed all:frontend/dist` keeps working.
    adapter: adapter({ pages: 'dist', assets: 'dist', fallback: 'index.html' })
  }
};
