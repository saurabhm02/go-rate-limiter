import { motion, AnimatePresence } from 'motion/react'
import { useEffect, useState } from 'react'

export const spring = { type: 'spring', stiffness: 420, damping: 34 }

export function Button({ variant = 'default', className = '', ...props }) {
  const styles = {
    default: 'bg-raise border-line text-ink hover:bg-line',
    primary: 'bg-ink border-ink text-plane font-semibold hover:opacity-90',
    ghost: 'bg-transparent border-transparent text-dim hover:text-ink hover:bg-raise',
    danger: 'bg-transparent border-line text-dim hover:text-bad hover:border-bad',
  }[variant]
  return (
    <motion.button
      whileHover={{ y: -1 }}
      whileTap={{ scale: 0.97 }}
      transition={spring}
      className={`rounded-lg border px-3.5 py-2 text-[13px] disabled:opacity-40 disabled:pointer-events-none ${styles} ${className}`}
      {...props}
    />
  )
}

export function Field({ label, hint, children }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-[11px] font-semibold uppercase tracking-[0.08em] text-faint">
        {label}
      </span>
      {children}
      {hint && <span className="mt-1.5 block text-xs text-faint">{hint}</span>}
    </label>
  )
}

export const inputCls =
  'w-full rounded-lg border border-line bg-plane px-3 py-2.5 text-[13px] text-ink outline-none transition focus:border-brand focus:ring-1 focus:ring-brand/50'

export function Modal({ open, onClose, title, children, footer, wide }) {
  useEffect(() => {
    if (!open) return
    const onKey = (e) => e.key === 'Escape' && onClose?.()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="fixed inset-0 z-50 grid place-items-center overflow-y-auto bg-black/70 p-4 backdrop-blur-sm"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
        >
          <motion.div
            className={`w-full ${wide ? 'max-w-2xl' : 'max-w-lg'} overflow-hidden rounded-2xl border border-line bg-surface shadow-2xl`}
            initial={{ opacity: 0, scale: 0.96, y: 12 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.97, y: 8 }}
            transition={spring}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="border-b border-line px-5 py-4 font-semibold">{title}</div>
            <div className="px-5 py-5">{children}</div>
            {footer && (
              <div className="flex justify-end gap-2 border-t border-line px-5 py-4">{footer}</div>
            )}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

export function Toast({ message }) {
  return (
    <AnimatePresence>
      {message && (
        <motion.div
          className="fixed bottom-6 left-1/2 z-[60] rounded-lg bg-ink px-4 py-2.5 text-[13px] font-semibold text-plane shadow-xl"
          initial={{ opacity: 0, y: 18, x: '-50%' }}
          animate={{ opacity: 1, y: 0, x: '-50%' }}
          exit={{ opacity: 0, y: 10, x: '-50%' }}
          transition={spring}
        >
          {message}
        </motion.div>
      )}
    </AnimatePresence>
  )
}

export function useToast() {
  const [message, setMessage] = useState('')
  useEffect(() => {
    if (!message) return
    const t = setTimeout(() => setMessage(''), 1900)
    return () => clearTimeout(t)
  }, [message])
  return [message, setMessage]
}

export function Copyable({ value, onCopied }) {
  return (
    <button
      type="button"
      onClick={() => {
        navigator.clipboard?.writeText(value)
        onCopied?.()
      }}
      className="group flex w-full items-center gap-3 rounded-lg border border-line bg-plane px-3 py-3 text-left transition hover:border-brand"
    >
      <code className="min-w-0 flex-1 break-all font-mono text-[13px] text-good">{value}</code>
      <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wider text-faint group-hover:text-ink">
        Copy
      </span>
    </button>
  )
}

export function Pill({ tone = 'dim', children }) {
  const tones = {
    dim: 'border-line text-dim',
    good: 'border-good/40 text-good bg-good/10',
    bad: 'border-bad/40 text-bad bg-bad/10',
    brand: 'border-brand/40 text-brand bg-brand/10',
  }[tone]
  return (
    <span className={`rounded-md border px-2 py-0.5 text-[11px] font-medium ${tones}`}>
      {children}
    </span>
  )
}

export function Spinner() {
  return (
    <motion.span
      className="inline-block h-2 w-2 rounded-full bg-brand"
      animate={{ opacity: [0.25, 1, 0.25], scale: [0.85, 1.15, 0.85] }}
      transition={{ duration: 1.1, repeat: Infinity, ease: 'easeInOut' }}
    />
  )
}
