import { Button, Copyable, Modal } from './ui'
import { API_BASE } from '../api'

export function KeyReveal({ open, projectName, rawKey, kind = 'api', onClose, onCopied }) {
  const isProject = kind === 'project'
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isProject ? `Project token for ${projectName}` : `API key for ${projectName}`}
      footer={
        <Button variant="primary" onClick={onClose}>
          I've saved it
        </Button>
      }
    >
      <div className="space-y-4">
        <Copyable value={rawKey} onCopied={onCopied} />

        <div className="rounded-r-lg border-l-2 border-warn bg-warn/10 px-3.5 py-3 text-[12.5px] leading-relaxed text-dim">
          <b className="text-ink">This is the only time it is shown.</b> Only its SHA-256 hash
          reached the server, so it cannot be recovered.
          {isProject
            ? ' Save it somewhere safe. It is how you get back into this project.'
            : ' Put it in your app. Losing it means minting another.'}
        </div>

        {isProject ? (
          <p className="text-[12.5px] leading-relaxed text-faint">
            This token manages the project. It cannot call <code className="text-dim">/v1/check</code> —
            mint a separate API key for your app, so a key leaking from a deployed
            service can never raise its own limits.
          </p>
        ) : (
        <div>
          <div className="mb-1.5 text-[11px] font-semibold uppercase tracking-[0.08em] text-faint">
            Use it like this
          </div>
          <pre className="overflow-x-auto rounded-lg border border-line bg-plane p-3 font-mono text-[11.5px] leading-relaxed text-dim">
{`curl -X POST ${API_BASE}/v1/check \\
  -H "X-API-Key: ${rawKey ? rawKey.slice(0, 14) + '…' : ''}" \\
  -H "Content-Type: application/json" \\
  -d '{"route":"/${projectName}/signup/x","subject":"<client IP>"}'`}
          </pre>
          <p className="mt-2 text-xs text-faint">
            <code className="text-dim">route</code> picks the rule;{' '}
            <code className="text-dim">subject</code> splits the counter so each IP or account
            gets its own budget.
          </p>
        </div>
        )}
      </div>
    </Modal>
  )
}
