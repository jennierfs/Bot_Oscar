import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Configuración de Vite para Bot Oscar
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    // Proxy para desarrollo local: redirige /api al backend Go
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
