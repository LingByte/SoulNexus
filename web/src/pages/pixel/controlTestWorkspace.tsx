import { useNavigate, useSearchParams } from 'react-router-dom'
import 'antd/dist/reset.css'
import ControlTest from '@/pages/pixel/frameronin/ControlTest'
import '@/pages/pixel/frameronin/controlTest.css'

/** Full FrameRonin ControlTest (topdown + arcade) — fullscreen like original. */
export default function ControlTestWorkspace() {
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const variant = params.get('variant') === 'arcade' ? 'arcade' : 'topdown'

  return (
    <>
      <div
        className="control-test-variant-bar"
        style={{
          position: 'fixed',
          top: 12,
          left: 12,
          zIndex: 1600,
          display: 'flex',
          gap: 8,
        }}
      >
        <button
          type="button"
          className={`rounded-md border px-3 py-1 text-sm backdrop-blur border-white/30 bg-black/40 ${
            variant === 'topdown' ? 'is-active' : ''
          }`}
          onClick={() => navigate('/pixel/control-test?variant=topdown', { replace: true })}
        >
          Top-down
        </button>
        <button
          type="button"
          className={`rounded-md border px-3 py-1 text-sm backdrop-blur border-white/30 bg-black/40 ${
            variant === 'arcade' ? 'is-active' : ''
          }`}
          onClick={() => navigate('/pixel/control-test?variant=arcade', { replace: true })}
        >
          街机
        </button>
      </div>
      <ControlTest key={variant} variant={variant} onBack={() => navigate('/pixel/sheet')} />
    </>
  )
}
