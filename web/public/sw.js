// Service worker de deckpad : garde l'appli (HTML, JS, CSS, icônes) en cache pour
// qu'elle s'ouvre vite et affiche son écran même si le PC ne répond pas encore.
// Les données (/api) ne passent jamais par le cache : elles viennent toujours du PC.
const CACHE = 'deckpad-app'

self.addEventListener('install', () => self.skipWaiting())
self.addEventListener('activate', (e) => e.waitUntil(self.clients.claim()))

self.addEventListener('fetch', (e) => {
  const req = e.request
  const url = new URL(req.url)
  if (req.method !== 'GET' || url.origin !== location.origin || url.pathname.startsWith('/api/')) return

  // Fichiers Vite nommés par leur contenu : une version en cache est toujours la bonne.
  if (url.pathname.startsWith('/assets/')) {
    e.respondWith(
      caches.match(req).then((hit) => hit || fetch(req).then((res) => keep(req, res))),
    )
    return
  }

  // Le reste (page, manifeste, icônes) : le PC d'abord pour avoir la dernière version,
  // le cache si le PC ne répond pas.
  e.respondWith(
    fetch(req)
      .then((res) => keep(req, res))
      .catch(() => caches.match(req).then((hit) => hit || caches.match('/'))),
  )
})

function keep(req, res) {
  if (res.ok) {
    const copy = res.clone()
    caches.open(CACHE).then((c) => c.put(req, copy))
  }
  return res
}
