import { useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { Button, Field, Modal, inputCls, spring } from './ui'
import { PRESETS, RuleEditor } from './RuleEditor'
import { humanWindow } from '../api'

const NAME_RE = /^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$/

export function NewProject({ open, onClose, onCreate, busy }) {
  const [name, setName] = useState('')
  const [rules, setRules] = useState([])
  const [editing, setEditing] = useState(null)
  const [editorOpen, setEditorOpen] = useState(false)

  const nameOk = NAME_RE.test(name.trim().toLowerCase())
  const canCreate = nameOk && rules.length > 0 && !busy

  const addPreset = (preset) => {

    const scoped = preset.rule.route_pattern.replace(/^\//, `/${name.trim().toLowerCase() || 'app'}/`)
    setRules((rs) =>
      rs.some((r) => r.route_pattern === scoped)
        ? rs
        : [...rs, { ...preset.rule, route_pattern: scoped, enabled: true, subject: preset.subject }],
    )
  }

  const reset = () => {
    setName('')
    setRules([])
  }

  return (
    <>
      <Modal
        open={open}
        onClose={onClose}
        wide
        title="New project"
        footer={
          <>
            <Button variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button
              variant="primary"
              disabled={!canCreate}
              onClick={async () => {
                const ok = await onCreate(name.trim().toLowerCase(), rules)
                if (ok) reset()
              }}
            >
              {busy ? 'Creating…' : 'Create project & mint key'}
            </Button>
          </>
        }
      >
        <div className="space-y-6">
          <Field
            label="Project name"
            hint={
              name && !nameOk
                ? '3–40 characters: lowercase letters, digits and hyphens, not starting or ending with a hyphen.'
                : 'Used in the key prefix and the rule patterns below.'
            }
          >
            <input
              className={`${inputCls} ${name && !nameOk ? 'border-bad focus:border-bad focus:ring-bad/50' : ''}`}
              placeholder="admitdesk"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </Field>

          <div>
            <div className="mb-2.5 text-[11px] font-semibold uppercase tracking-[0.08em] text-faint">
              Start from a preset
            </div>
            <div className="grid gap-3 sm:grid-cols-3">
              {PRESETS.map((p) => (
                <motion.button
                  key={p.label}
                  type="button"
                  onClick={() => addPreset(p)}
                  whileHover={{ y: -3 }}
                  whileTap={{ scale: 0.98 }}
                  transition={spring}
                  className="rounded-xl border border-line bg-raise p-3.5 text-left transition hover:border-brand"
                >
                  <div className="text-lg">{p.icon}</div>
                  <div className="mt-1.5 text-sm font-semibold">{p.label}</div>
                  <p className="mt-1 text-xs leading-snug text-faint">{p.blurb}</p>
                  <div className="mt-2 font-mono text-[11px] text-dim">
                    {p.rule.limit_count
                      ? `${p.rule.limit_count} / ${humanWindow(p.rule.window_seconds)}`
                      : `burst ${p.rule.bucket_capacity}`}{' '}
                    · {p.subject}
                  </div>
                </motion.button>
              ))}
            </div>
          </div>

          <div>
            <div className="mb-2 flex items-center gap-3">
              <span className="text-[11px] font-semibold uppercase tracking-[0.08em] text-faint">
                Rules ({rules.length})
              </span>
              <div className="flex-1" />
              <Button
                onClick={() => {
                  setEditing(null)
                  setEditorOpen(true)
                }}
              >
                + Custom rule
              </Button>
            </div>

            <div className="overflow-hidden rounded-xl border border-line">
              {rules.length === 0 ? (
                <p className="px-4 py-6 text-center text-[13px] text-faint">
                  No rules yet. With none, every route is allowed — limiting is opt-in.
                </p>
              ) : (
                <AnimatePresence initial={false}>
                  {rules.map((r, i) => (
                    <motion.div
                      key={r.route_pattern}
                      layout
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: 'auto' }}
                      exit={{ opacity: 0, height: 0 }}
                      transition={spring}
                      className="flex items-center gap-3 border-b border-line px-4 py-3 last:border-b-0"
                    >
                      <code className="min-w-0 flex-1 truncate font-mono text-[13px]">
                        {r.route_pattern}
                      </code>
                      <span className="hidden text-xs text-dim sm:block">
                        {r.algorithm === 'token_bucket'
                          ? `burst ${r.bucket_capacity} · ${r.refill_rate}/s`
                          : `${r.limit_count} / ${humanWindow(r.window_seconds)}`}
                      </span>
                      <Button
                        variant="danger"
                        onClick={() => setRules((rs) => rs.filter((_, j) => j !== i))}
                      >
                        Remove
                      </Button>
                    </motion.div>
                  ))}
                </AnimatePresence>
              )}
            </div>
          </div>
        </div>
      </Modal>

      <RuleEditor
        open={editorOpen}
        initial={editing}
        onClose={() => setEditorOpen(false)}
        onSave={(rule) => {
          setRules((rs) =>
            rs.some((r) => r.route_pattern === rule.route_pattern) ? rs : [...rs, rule],
          )
          setEditorOpen(false)
        }}
      />
    </>
  )
}
