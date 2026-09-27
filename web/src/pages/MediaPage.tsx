import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Music, Pause, Play, SkipBack, SkipForward } from 'lucide-react'
import { Loader } from '../components/Loader'
import { useEventStream } from '../hooks/useEventStream'
import { useLatched } from '../hooks/useLatched'

type MediaState = {
  active: boolean
  playing: boolean
  title?: string
  artist?: string
  album?: string
  app?: string
  cover?: string
  unavailable?: string
}

type Action = 'playpause' | 'next' | 'previous'

export function MediaPage() {
  const [state, setState] = useState<MediaState | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const connected = useEventStream<MediaState>('/api/media/stream', setState)
  const [playing, latchPlaying] = useLatched(state?.playing ?? false, 2000)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  async function send(action: Action) {
    if (action === 'playpause') latchPlaying(!playing)
    const r = await fetch(`/api/media/${action}`, { method: 'POST' }).catch(() => null)
    if (!r?.ok) setNotice('Le lecteur ne répond pas')
  }

  if (!state) return <Loader label={connected ? 'Chargement…' : 'Connexion au PC…'} />
  if (state.unavailable) return <p className="coming-soon files-empty">Médias indisponibles : {state.unavailable}</p>
  if (!state.active) {
    return (
      <div className="media-empty">
        <Music size={56} strokeWidth={1.5} aria-hidden />
        <p>Aucun média en cours</p>
        <span>Lancez Spotify, une vidéo YouTube…</span>
      </div>
    )
  }

  return (
    <>
      <div className="media">
        <Cover id={state.cover} />
        <div className="media-side">
          <div className="media-info">
            {state.app && <span className="media-app">{state.app}</span>}
            <p className="media-title">{state.title || 'Titre inconnu'}</p>
            {state.artist && <p className="media-artist">{state.artist}</p>}
            {state.album && <p className="media-album">{state.album}</p>}
          </div>
          <div className="media-controls">
            <button type="button" className="media-btn" aria-label="Précédent" onClick={() => send('previous')}>
              <SkipBack size={34} strokeWidth={2} fill="currentColor" aria-hidden />
            </button>
            <button
              type="button"
              className="media-btn media-play"
              aria-label={playing ? 'Pause' : 'Lecture'}
              onClick={() => send('playpause')}
            >
              {playing ? (
                <Pause size={46} strokeWidth={2} fill="currentColor" aria-hidden />
              ) : (
                <Play size={46} strokeWidth={2} fill="currentColor" aria-hidden />
              )}
            </button>
            <button type="button" className="media-btn" aria-label="Suivant" onClick={() => send('next')}>
              <SkipForward size={34} strokeWidth={2} fill="currentColor" aria-hidden />
            </button>
          </div>
        </div>
      </div>
      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
    </>
  )
}

// Pochette du morceau ; une note de musique s'il n'y en a pas.
function Cover({ id }: { id?: string }) {
  const [failed, setFailed] = useState<string | null>(null)
  const show = id && failed !== id
  return (
    <div className="media-cover">
      {show ? (
        <img src={`/api/media/cover?v=${id}`} alt="" onError={() => setFailed(id)} />
      ) : (
        <Music size={72} strokeWidth={1.5} aria-hidden />
      )}
    </div>
  )
}
