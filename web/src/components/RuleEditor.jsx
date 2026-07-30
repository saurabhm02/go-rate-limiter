import { useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { Button, Field, Modal, inputCls, spring } from './ui'


export const PRESETS = [
  {
    icon: '🔑',
    label: 'Login',
    blurb: 'Slow brute force without locking out a shared office IP.',
    rule: { route_pattern: '/login/*', algorithm: 'sliding_window', limit_count: 10, window_seconds: 900 },
    subject: 'account id',
  },
  {
    icon: '📝',
    label: 'Public form',
    blurb: 'Stop scripted spam on anything a stranger can reach.',
    rule: { route_pattern: '/intake/*', algorithm: 'sliding_window', limit_count: 5, window_seconds: 600 },
    subject: 'client IP',
  },
  {
    icon: '💸',
    label: 'Expensive call',
    blurb: 'Cap anything that costs you money per request.',
    rule: { route_pattern: '/ai/*', algorithm: 'token_bucket', bucket_capacity: 10, refill_rate: 0.5 },
    subject: 'account id',
  },
]

const blank = {
  route_pattern: '',
  algorithm: 'sliding_window',
  limit_count: 5,
  window_seconds: 600,
  bucket_capacity: 10,
  refill_rate: 2,
  enabled: true,
}

export function RuleEditor({ open, initial, onClose, onSave }) {
  const [draft, setDraft] = useState(initial ?? blank)
  const set = (k) => (e) =>
    setDraft((d) => ({ ...d, [k]: e.target.type === 'number' ? Number(e.target.value) : e.target.value }))

  const key = open ? (initial?.id ?? 'new') : 'closed'
  const isBucket = draft.algorithm === 'token_bucket'

  return (
    <Modal
      key={key}
      open={open}
      onClose={onClose}
      title={initial ? 'Edit rule' : 'Add rule'}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={() => {
              if (!draft.route_pattern.trim()) return
              onSave({ ...draft, route_pattern: draft.route_pattern.trim() })
            }}
          >
            Save rule
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <Field
          label="Route pattern"
          hint="Trailing * matches by prefix. A bare * is the catch-all."
        >
          <input
            className={`${inputCls} font-mono`}
            placeholder="/myapp/signup/*"
            value={draft.route_pattern}
            onChange={set('route_pattern')}
            autoFocus
          />
        </Field>

        <Field label="Algorithm">
          <select className={inputCls} value={draft.algorithm} onChange={set('algorithm')}>
            <option value="sliding_window">Sliding window — steady rate, no burst</option>
            <option value="token_bucket">Token bucket — allows a burst, then refills</option>
          </select>
        </Field>

        <AnimatePresence mode="wait">
          <motion.div
            key={draft.algorithm}
            initial={{ opacity: 0, y: -6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 6 }}
            transition={spring}
            className="grid grid-cols-2 gap-3"
          >
            {isBucket ? (
              <>
                <Field label="Burst capacity">
                  <input type="number" min="1" className={inputCls} value={draft.bucket_capacity} onChange={set('bucket_capacity')} />
                </Field>
                <Field label="Refill / second">
                  <input type="number" min="0.01" step="0.01" className={inputCls} value={draft.refill_rate} onChange={set('refill_rate')} />
                </Field>
              </>
            ) : (
              <>
                <Field label="Limit">
                  <input type="number" min="1" className={inputCls} value={draft.limit_count} onChange={set('limit_count')} />
                </Field>
                <Field label="Window (seconds)">
                  <input type="number" min="1" className={inputCls} value={draft.window_seconds} onChange={set('window_seconds')} />
                </Field>
              </>
            )}
          </motion.div>
        </AnimatePresence>

        <Field
          label="Subject note (documentation only)"
          hint="A note for humans. The caller decides what to send as subject; the rule cannot enforce it."
        >
          <input className={inputCls} placeholder="client IP" value={draft.subject ?? ''} onChange={set('subject')} />
        </Field>
      </div>
    </Modal>
  )
}
