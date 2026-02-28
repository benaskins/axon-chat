import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import basicSsl from '@vitejs/plugin-basic-ssl';

export default defineConfig({
  plugins: [sveltekit(), basicSsl()],
  server: {
    host: 'dev.chat.studio.internal',
    port: 5173,
    proxy: {
      '/api': {
        target: 'https://chat.studio.internal',
        changeOrigin: true,
        secure: false
      }
    }
  }
});
