import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { Headphones, Mic, MicOff, MonitorSpeaker, Speaker, Volume2, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { VolumeFader } from '../components/VolumeFader'
import { AppIcon } from '../components/AppIcon'
import { SideScroll } from '../components/SideScroll'
import { Loader } from '../components/Loader'
import { useEventStream } from '../hooks/useEventStream'
import { useLatched } from '../hooks/useLatched'

type Level = { volume: number; muted: boolean }
type Device = { id: string; name: string; default: boolean }
type AudioApp = Level & { id: string; name: string }
type AudioState = {
  master: Level | null
  mic: Level | null
  outputs: Device[]
  apps: AudioApp[]
  unavailable?: string
}

export function AudioPage() {
  const { t } = useTranslation()
  const [state, setState] = useState<AudioState | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const connected = useEventStream<AudioState>('/api/audio/stream', setState)
  // Son général coupé : toutes les applis s'affichent coupées, tout de suite.
  const [masterMuted, latchMasterMuted] = useLatched(state?.master?.muted ?? false)

  useEffect(() => {
    if (!notice) return
    const id = setTimeout(() => setNotice(null), 3000)
    return () => clearTimeout(id)
  }, [notice])

  async function post(path: string, body: object, failure: string) {
    const r = await fetch(`/api/audio/${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }).catch(() => null)
    if (!r?.ok) setNotice(failure)
  }

  const setVolume = (target: string, volume: number) => post('volume', { target, volume }, t('audio.volumeFailed'))
  const setMute = (target: string, muted: boolean) =>
    post('mute', { target, muted }, t(muted ? 'audio.muteFailed' : 'audio.unmuteFailed'))

  if (!state) return <Loader label={connected ? undefined : t('common.connecting')} />
  if (state.unavailable) return <p className="coming-soon files-empty">{t('audio.unavailable', { reason: state.unavailable })}</p>

  return (
    <>
      <div className="audio-bar">
        <Outputs
          outputs={state.outputs}
          onSelect={(d) => post('output', { id: d.id }, t('audio.switchFailed', { name: d.name }))}
        />
        <MicButton mic={state.mic} onMute={(m) => setMute('mic', m)} />
      </div>
      <SideScroll className="mixer">
        {state.master ? (
          <VolumeFader
            label={t('audio.master')}
            icon={<Volume2 size={20} strokeWidth={2} aria-hidden />}
            volume={state.master.volume}
            muted={masterMuted}
            onVolume={(v) => setVolume('master', v)}
            onMute={(m) => {
              latchMasterMuted(m)
              setMute('master', m)
            }}
          />
        ) : (
          <p className="mixer-empty">{t('audio.noOutput')}</p>
        )}
        {state.apps.map((a) => (
          <VolumeFader
            key={a.id}
            label={a.name}
            appIcon
            icon={<AppIcon name={a.id} src={`/api/audio/icon?app=${encodeURIComponent(a.id)}`} />}
            volume={a.volume}
            muted={a.muted}
            silenced={masterMuted}
            onVolume={(v) => setVolume(`app:${a.id}`, v)}
            onMute={(m) => setMute(`app:${a.id}`, m)}
          />
        ))}
        {state.master && state.apps.length === 0 && <p className="mixer-empty">{t('audio.noApp')}</p>}
      </SideScroll>
      {notice && createPortal(<div className="toast">{notice}</div>, document.body)}
    </>
  )
}

// Choix de la sortie : la sortie cochée est celle qui joue le son.
function Outputs({ outputs, onSelect }: { outputs: Device[]; onSelect: (d: Device) => void }) {
  const { t } = useTranslation()
  const current = outputs.find((d) => d.default)?.id ?? ''
  const [selected, select] = useLatched(current, 3000)
  if (outputs.length === 0) return <div className="audio-outputs" />
  return (
    <div className="segmented audio-outputs" role="radiogroup" aria-label={t('audio.output')}>
      {outputs.map((d) => {
        const Icon = deviceIcon(d.name)
        return (
          <button
            key={d.id}
            type="button"
            role="radio"
            aria-checked={d.id === selected}
            className={d.id === selected ? 'active' : ''}
            onClick={() => {
              if (d.id === selected) return
              select(d.id)
              onSelect(d)
            }}
          >
            <Icon size={22} strokeWidth={2} aria-hidden />
            <span>{d.name}</span>
          </button>
        )
      })}
    </div>
  )
}

function MicButton({ mic, onMute }: { mic: Level | null; onMute: (muted: boolean) => void }) {
  const { t } = useTranslation()
  const [muted, latch] = useLatched(mic?.muted ?? false)
  if (!mic) {
    return (
      <button type="button" className="audio-mic" disabled>
        <MicOff size={26} strokeWidth={2} aria-hidden />
        {t('audio.noMic')}
      </button>
    )
  }
  const Icon = muted ? MicOff : Mic
  return (
    <button
      type="button"
      className={muted ? 'audio-mic is-muted' : 'audio-mic'}
      aria-pressed={muted}
      onClick={() => {
        latch(!muted)
        onMute(!muted)
      }}
    >
      <Icon size={26} strokeWidth={2} aria-hidden />
      {muted ? t('audio.micMuted') : t('audio.micOn')}
    </button>
  )
}


// Devine le type de sortie d'après son nom (Windows et Linux, en français ou en anglais).
function deviceIcon(name: string): LucideIcon {
  const n = name.toLowerCase()
  if (/casque|headset|headphone|écouteur|earphone|buds|arctis|hyperx/.test(n)) return Headphones
  if (/hdmi|displayport|écran|monitor|display|tv|nvidia|amd|radeon/.test(n)) return MonitorSpeaker
  return Speaker
}
