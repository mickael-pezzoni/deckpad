import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Music, Pause, Play, SkipBack, SkipForward } from 'lucide-react'
import { useTranslation } from 'react-i18next'
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
  const { t } = useTranslation()
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
    if (!r?.ok) setNotice(t('media.noAnswer'))
  }

  if (!state) return <Loader label={connected ? undefined : t('common.connecting')} />
  if (state.unavailable) return <p className="coming-soon files-empty">{t('media.unavailable', { reason: state.unavailable })}</p>
  if (!state.active) {
    return (
      <div className="media-empty">
        <Music size={56} strokeWidth={1.5} aria-hidden />
        <p>{t('media.nothing')}</p>
        <span>{t('media.hint')}</span>
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
            <p className="media-title">{state.title || t('media.unknownTitle')}</p>
            {state.artist && <p className="media-artist">{state.artist}</p>}
            {state.album && <p className="media-album">{state.album}</p>}
          </div>
          <div className="media-controls">
            <button type="button" className="media-btn" aria-label={t('media.previous')} onClick={() => send('previous')}>
              <SkipBack size={34} strokeWidth={2} fill="currentColor" aria-hidden />
            </button>
            <button
              type="button"
              className="media-btn media-play"
              aria-label={playing ? t('media.pause') : t('media.play')}
              onClick={() => send('playpause')}
            >
              {playing ? (
                <Pause size={46} strokeWidth={2} fill="currentColor" aria-hidden />
              ) : (
                <Play size={46} strokeWidth={2} fill="currentColor" aria-hidden />
              )}
            </button>
            <button type="button" className="media-btn" aria-label={t('media.next')} onClick={() => send('next')}>
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
