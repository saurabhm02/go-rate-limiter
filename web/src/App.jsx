import { useCallback, useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import {
  API_BASE, ApiError, addKey, createProject, generateKey,
  getToken, listProjects, revokeKey, setToken,
} from './api'
import { Button, Field, Pill, Spinner, Toast, inputCls, spring, useToast } from './components/ui'
import { NewProject } from './components/NewProject'
import { KeyReveal } from './components/KeyReveal'
import { ProjectDetail } from './components/ProjectDetail'

function Gate({ onUnlock, error, busy }) {
  const [value, setValue] = useState('')
  return (
    <div className="grid min-h-screen place-items-center px-4">
      <motion.div
        initial={{ opacity: 0, y: 14 }}
        animate={{ opacity: 1, y: 0 }}
        transition={spring}
        className="w-full max-w-sm"
      >
        <div className="mb-7 text-center">
          <div className="text-2xl font-bold tracking-tight">
            go<span className="text-brand">rate</span>
          </div>
          <p className="mt-1.5 text-[13px] text-faint">Rate limit console</p>
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            onUnlock(value.trim())
          }}
          className="space-y-4 rounded-2xl border border-line bg-surface p-6"
        >
          <Field label="Admin token" hint="Held in this tab only — cleared when you close it.">
            <input
              type="password"
              className={inputCls}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder="ADMIN_TOKEN"
              autoFocus
              autoComplete="off"
            />
          </Field>
          {error && <p className="text-[12.5px] text-bad">{error}</p>}
          <Button variant="primary" className="w-full" disabled={!value.trim() || busy}>
            {busy ? 'Checking…' : 'Continue'}
          </Button>
        </form>
        <p className="mt-4 text-center text-xs text-faint">{API_BASE}</p>
      </motion.div>
    </div>
  )
}

function friendly(err) {
  if (!(err instanceof ApiError)) return 'Something went wrong.'
  switch (err.code) {
    case 'unreachable':
      return `Could not reach ${API_BASE}. The free instance can take 30–60s to wake — try again.`
    case 'admin_api_disabled':
      return 'The write API is off — ADMIN_TOKEN is not set on the service.'
    case 'project_exists':
      return 'A project with that name already exists.'
    default:
      if (err.status === 401) return 'Admin token rejected.'
      if (err.status === 404) return 'Not found — the API may not have this route deployed yet.'
      return err.code || `Failed (${err.status}).`
  }
}

export default function App() {
  const [token, setTok] = useState(getToken())
  const [gateError, setGateError] = useState('')
  const [projects, setProjects] = useState(null)
  const [selected, setSelected] = useState(null)
  const [busy, setBusy] = useState(false)
  const [newOpen, setNewOpen] = useState(false)
  const [reveal, setReveal] = useState(null)
  const [toast, showToast] = useToast()
  const [error, setError] = useState('')

  const refresh = useCallback(async () => {
    setError('')
    try {
      const list = await listProjects()
      setProjects(list)
      return list
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setToken('')
        setTok('')
        setGateError('Admin token rejected.')
        return null
      }
      setError(friendly(err))
      setProjects([])
      return null
    }
  }, [])

  useEffect(() => {
    if (token) refresh()
  }, [token, refresh])

  const unlock = async (value) => {
    setBusy(true)
    setGateError('')
    setToken(value)
    try {
      await listProjects()
      setTok(value)
    } catch (err) {
      setToken('')
      setGateError(friendly(err))
    } finally {
      setBusy(false)
    }
  }

  const handleCreate = async (name, rules) => {
    setBusy(true)
    setError('')
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
      setReveal({ projectName: name, rawKey: key.raw })
      const list = await refresh()
      setSelected(list?.find((p) => p.name === name)?.id ?? null)
      return true
    } catch (err) {
      setError(friendly(err))
      return false
    } finally {
      setBusy(false)
    }
  }

  const handleMintKey = async (project) => {
    setBusy(true)
    setError('')
    try {
      const key = await generateKey(project.name)
      await addKey(project.id, { key_hash: key.hash, key_prefix: key.prefix })
      setReveal({ projectName: project.name, rawKey: key.raw })
      await refresh()
    } catch (err) {
      setError(friendly(err))
    } finally {
      setBusy(false)
    }
  }

  const handleRevoke = async (project, key) => {
    if (!confirm(`Revoke ${key.prefix}…? Anything using it stops working immediately.`)) return
    setBusy(true)
    try {
      await revokeKey(project.id, key.id)
      showToast('Key revoked')
      await refresh()
    } catch (err) {
      setError(friendly(err))
    } finally {
      setBusy(false)
    }
  }

  if (!token) return <Gate onUnlock={unlock} error={gateError} busy={busy} />

  const current = projects?.find((p) => p.id === selected) ?? null

  return (
    <div className="min-h-screen">
      <header className="aurora relative border-b border-line bg-surface/80 backdrop-blur">
        <div className="mx-auto flex max-w-5xl items-center gap-4 px-5 py-3.5">
          <button
            onClick={() => setSelected(null)}
            className="text-[15px] font-bold tracking-tight transition hover:opacity-80"
          >
            go<span className="text-brand">rate</span>
          </button>

          <AnimatePresence>
            {current && (
              <motion.div
                initial={{ opacity: 0, x: -8 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: -8 }}
                transition={spring}
                className="flex items-center gap-2 text-[13px] text-faint"
              >
                <span>/</span>
                <span className="text-ink">{current.name}</span>
              </motion.div>
            )}
          </AnimatePresence>

          <div className="flex-1" />
          {busy && <Spinner />}
          <Button variant="primary" onClick={() => setNewOpen(true)}>
            + New project
          </Button>
          <Button
            variant="ghost"
            onClick={() => {
              setToken('')
              setTok('')
              setProjects(null)
              setSelected(null)
            }}
          >
            Lock
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

        {projects === null ? (
          <div className="flex items-center gap-3 py-20 text-[13px] text-faint">
            <Spinner /> Loading projects…
          </div>
        ) : current ? (
          <ProjectDetail
            project={current}
            busy={busy}
            onMintKey={handleMintKey}
            onRevokeKey={handleRevoke}
          />
        ) : (
          <>
            <h2 className="mb-4 text-[11px] font-semibold uppercase tracking-[0.08em] text-faint">
              Projects ({projects.length})
            </h2>
            <motion.div layout className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <AnimatePresence initial={false}>
                {projects.map((p, i) => (
                  <motion.button
                    key={p.id}
                    layout
                    initial={{ opacity: 0, y: 14 }}
                    animate={{ opacity: 1, y: 0, transition: { ...spring, delay: i * 0.04 } }}
                    exit={{ opacity: 0, scale: 0.96 }}
                    whileHover={{ y: -4 }}
                    whileTap={{ scale: 0.99 }}
                    transition={spring}
                    onClick={() => setSelected(p.id)}
                    className="rounded-2xl border border-line bg-surface p-5 text-left transition hover:border-brand"
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-semibold">{p.name}</span>
                      {p.status === 'active' ? (
                        <Pill tone="good">active</Pill>
                      ) : (
                        <Pill tone="bad">{p.status}</Pill>
                      )}
                    </div>
                    <div className="mt-3 space-y-1 text-[12.5px] text-faint">
                      <div>
                        {p.rule_count} {p.rule_count === 1 ? 'rule' : 'rules'} ·{' '}
                        {(p.keys ?? []).filter((k) => k.status === 'active').length} active{' '}
                        {(p.keys ?? []).filter((k) => k.status === 'active').length === 1
                          ? 'key'
                          : 'keys'}
                      </div>
                      <div>created {new Date(p.created_at).toLocaleDateString()}</div>
                    </div>
                  </motion.button>
                ))}
              </AnimatePresence>

              <motion.button
                layout
                whileHover={{ y: -4 }}
                whileTap={{ scale: 0.99 }}
                transition={spring}
                onClick={() => setNewOpen(true)}
                className="grid min-h-[132px] place-items-center rounded-2xl border border-dashed border-line bg-transparent p-5 text-center text-faint transition hover:border-brand hover:text-dim"
              >
                <div>
                  <div className="text-xl">+</div>
                  <div className="mt-1 text-[13px]">New project</div>
                  <div className="mt-0.5 text-xs">rules + key in one step</div>
                </div>
              </motion.button>
            </motion.div>
          </>
        )}
      </main>

      <NewProject
        open={newOpen}
        busy={busy}
        onClose={() => setNewOpen(false)}
        onCreate={handleCreate}
      />
      <KeyReveal
        open={!!reveal}
        projectName={reveal?.projectName ?? ''}
        rawKey={reveal?.rawKey ?? ''}
        onClose={() => setReveal(null)}
        onCopied={() => showToast('Key copied')}
      />
      <Toast message={toast} />
    </div>
  )
}
