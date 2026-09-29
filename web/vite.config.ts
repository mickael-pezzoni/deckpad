import { writeFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const outDir = '../hub/webdist/dist'

export default defineConfig({
  plugins: [
    react(),
    {
      // Vite vide dist/ à chaque build : on recrée le .gitkeep pour que `go build` compile sans build web.
      name: 'keep-dist-placeholder',
      closeBundle() {
        writeFileSync(`${outDir}/.gitkeep`, '')
      },
    },
  ],
  build: { outDir, emptyOutDir: true },
  server: {
    host: true, // accessible depuis la tablette sur le réseau local
    // Tout passe par le hub, qui relaie vers les PC.
    proxy: {
      '/api': 'http://localhost:8430',
      '/ca': 'http://localhost:8430', // certificat HTTPS à installer
    },
  },
})
