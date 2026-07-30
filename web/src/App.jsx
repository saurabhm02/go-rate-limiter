import { useCallback, useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import {
  API_BASE, ApiError, createProject, generateKey, getToken,
  loadProject, mintKey, revokeKey, setToken,
} from './api'
import { Button, Field, Pill, Spinner, Toast, inputCls, spring, useToast } from './components/ui'
import { NewProject } from './components/NewProject'
import { KeyReveal } from './components/KeyReveal'
import { ProjectDetail } from './components/ProjectDetail'

function friendly(err) {
  if (!(err instanceof ApiError)) return 'Something went wrong.'
  switch (err.code) {
    case 'unreachable':
      return 'Could not reach the API. It can take 30–60s to wake up. Try again.'
    case 'project_exists':
      return 'That name is taken. Pick another.'
    default:
      if (err.status === 401) return 'That token is not valid.'
      if (err.status === 429) return 'Too many projects created from here. Try again later.'
      return err.code || `Failed (${err.status}).`
  }
}

function Landing({ onSignIn, onCreateClick, error, busy }) {
  const [mode, setMode] = useState(null) // null | 'token'
  const [value, setValue] = useState('')

  return (
    <div className="grid min-h-screen place-items-center px-4">
      <motion.div
        initial={{ opacity: 0, y: 14 }}
        animate={{ opacity: 1, y: 0 }}
        transition={spring}
        className="w-full max-w-md"
      >
        <div className="mb-8 text-center">
          <div className="text-3xl font-bold tracking-tight">
            go<span className="text-brand">rate</span>
          </div>
          <p className="mt-2 text-[13px] text-faint">
            Rate limiting for your app. One project, one token.
          </p>
        </div>

        <AnimatePresence mode="wait">
          {mode === null ? (
            <motion.div
              key="choose"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={spring}
              className="space-y-3"
            >
              <Button variant="primary" className="w-full py-3" onClick={onCreateClick}>
                Create a project
              </Button>
              <Button className="w-full py-3" onClick={() => setMode('token')}>
                I already have a token
              </Button>
              {error && <p className="pt-1 text-center text-[12.5px] text-bad">{error}</p>}
            </motion.div>
          ) : (
            <motion.form
              key="token"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={spring}
              onSubmit={(e) => {
                e.preventDefault()
                onSignIn(value.trim())
              }}
              className="space-y-4 rounded-2xl border border-line bg-surface p-6"
            >
              <Field label="Project token" hint="The token you were shown when the project was created.">
                <input
                  type="password"
                  className={inputCls}
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  placeholder="rl_yourproject_…"
                  autoFocus
                  autoComplete="off"
                />
              </Field>
              {error && <p className="text-[12.5px] text-bad">{error}</p>}
              <Button variant="primary" className="w-full" disabled={!value.trim() || busy}>
                {busy ? 'Checking…' : 'Open project'}
              </Button>
              <button
                type="button"
                onClick={() => setMode(null)}
                className="mx-auto block text-xs text-faint hover:text-dim"
              >
                Back
              </button>
            </motion.form>
          )}
        </AnimatePresence>
      </motion.div>
    </div>
  )
}

export default function App() {
  const [token, setTok] = useState(getToken())
  const [project, setProject] = useState(null)
  const [landingError, setLandingError] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [newOpen, setNewOpen] = useState(false)
  const [reveal, setReveal] = useState(null)
  const [toast, showToast] = useToast()

  const refresh = useCallback(async () => {
    setError('')
    try {
      setProject(await loadProject())
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setToken('')
        setTok('')
        setProject(null)
        setLandingError('That token is no longer valid.')
        return
      }
      setError(friendly(err))
    }
  }, [])

  useEffect(() => {
    if (token) refresh()
  }, [token, refresh])

  const signIn = async (value) => {
    setBusy(true)
    setLandingError('')
    setToken(value)
    try {
      const p = await loadProject()
      setProject(p)
      setTok(value)
    } catch (err) {
      setToken('')
      setLandingError(friendly(err))
    } finally {
      setBusy(false)
    }
  }

  // Create the project, then sign in with the token it was given.
  const handleCreate = async (name, rules) => {
    setBusy(true)
    setError('')
    setLandingError('')
    try {
      const key = await generateKey(name)
      await createProject({
        name,
        key_hash: key.hash,
        key_prefix: key.prefix,
        rules: rules.map((r) => ({
          route_pattern: r.route_pattern,
          algorithm: r.algorithm,
          enabled: r.enabled !== false,
          limit_count: r.algorithm === 'token_bucket' ? 0 : Number(r.limit_count) || 0,
          window_seconds: r.algorithm === 'token_bucket' ? 0 : Number(r.window_seconds) || 0,
          bucket_capacity: r.algorithm === 'token_bucket' ? Number(r.bucket_capacity) || 0 : 0,
          refill_rate: r.algorithm === 'token_bucket' ? Number(r.refill_rate) || 0 : 0,
        })),
      })
      setNewOpen(false)
      setReveal({ projectName: name, rawKey: key.raw, kind: 'project' })
      setToken(key.raw)
      setTok(key.raw)
      return true
    } catch (err) {
      const message = friendly(err)
      setError(message)
      setLandingError(message)
      return false
    } finally {
      setBusy(false)
    }
  }

  const handleMintKey = async () => {
    setBusy(true)
    setError('')
    try {
      const key = await generateKey(project.name)
      await mintKey({ key_hash: key.hash, key_prefix: key.prefix, role: 'check' })
      setReveal({ projectName: project.name, rawKey: key.raw, kind: 'api' })
      await refresh()
    } catch (err) {
      setError(friendly(err))
    } finally {
      setBusy(false)
    }
  }

  const handleRevoke = async (_project, key) => {
    if (!confirm(`Revoke ${key.prefix}…? Anything using it stops working immediately.`)) return
    setBusy(true)
    try {
      await revokeKey(key.id)
      showToast('Key revoked')
      await refresh()
    } catch (err) {
      setError(friendly(err))
    } finally {
      setBusy(false)
    }
  }

  const signOut = () => {
    setToken('')
    setTok('')
    setProject(null)
    setError('')
  }

  if (!token) {
    return (
      <>
        <Landing
          onSignIn={signIn}
          onCreateClick={() => setNewOpen(true)}
          error={landingError}
          busy={busy}
        />
        <NewProject open={newOpen} busy={busy} onClose={() => setNewOpen(false)} onCreate={handleCreate} />
        <KeyReveal
          open={!!reveal}
          projectName={reveal?.projectName ?? ''}
          rawKey={reveal?.rawKey ?? ''}
          kind={reveal?.kind}
          onClose={() => setReveal(null)}
          onCopied={() => showToast('Token copied')}
        />
        <Toast message={toast} />
      </>
    )
  }

  return (
    <div className="min-h-screen">
      <header className="aurora relative border-b border-line bg-surface/80 backdrop-blur">
        <div className="mx-auto flex max-w-5xl items-center gap-4 px-5 py-3.5">
          <span className="text-[15px] font-bold tracking-tight">
            go<span className="text-brand">rate</span>
          </span>
          {project && (
            <motion.span
              initial={{ opacity: 0, x: -8 }}
              animate={{ opacity: 1, x: 0 }}
              transition={spring}
              className="flex items-center gap-2 text-[13px] text-faint"
            >
              <span>/</span>
              <span className="text-ink">{project.name}</span>
            </motion.span>
          )}
          <div className="flex-1" />
          {busy && <Spinner />}
          <Button variant="ghost" onClick={signOut}>
            Sign out
          </Button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-5 py-8">
        <AnimatePresence mode="wait">
          {error && (
            <motion.div
              key={error}
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              className="mb-6 rounded-r-lg border-l-2 border-bad bg-bad/10 px-4 py-3 text-[13px] text-dim"
            >
              {error}
            </motion.div>
          )}
        </AnimatePresence>

        {!project ? (
          <div className="flex items-center gap-3 py-20 text-[13px] text-faint">
            <Spinner /> Loading project…
          </div>
        ) : (
          <ProjectDetail
            project={project}
            busy={busy}
            onMintKey={handleMintKey}
            onRevokeKey={handleRevoke}
          />
        )}
      </main>

      <KeyReveal
        open={!!reveal}
        projectName={reveal?.projectName ?? ''}
        rawKey={reveal?.rawKey ?? ''}
        kind={reveal?.kind}
        onClose={() => setReveal(null)}
        onCopied={() => showToast('Copied')}
      />
      <Toast message={toast} />
      <p className="pb-8 text-center text-[11px] text-faint">{API_BASE.replace(/^https?:\/\//, '')}</p>
    </div>
  )
}
