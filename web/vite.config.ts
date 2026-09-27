import { writeFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const outDir = '../agent/webdist/dist'

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
    // xfwd : l'agent voit que la requête vient de la tablette, pas du PC (voir localOnly).
    proxy: { '/api': { target: 'http://localhost:8420', xfwd: true } },
  },
})
