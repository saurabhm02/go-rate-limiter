import { useMemo, useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { Button, Pill, inputCls, spring } from './ui'
import { humanWindow, resolveRoute } from '../api'

function RouteTester({ rules, projectName }) {
  const [route, setRoute] = useState(`/${projectName}/signup/example`)
  const { best, next } = useMemo(() => resolveRoute(rules, route.trim()), [rules, route])

  return (
    <div>
      <input
        className={`${inputCls} font-mono`}
        value={route}
        onChange={(e) => setRoute(e.target.value)}
        spellCheck={false}
        aria-label="Route to test"
      />
      <AnimatePresence mode="wait">
        <motion.div
          key={best ? best.rule.route_pattern : 'none'}
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          transition={spring}
          className="mt-3 rounded-lg border border-line bg-plane px-3.5 py-3"
        >
          {!best ? (
            <>
              <div className="font-mono text-[13px] text-faint">✕ no rule matches → allowed</div>
              <div className="mt-1 text-xs text-faint">
                Limiting is opt-in — an unmatched route is never limited.
              </div>
            </>
          ) : (
            <>
              <div className="font-mono text-[13px]">
                <span className="text-good">✓</span> {best.rule.route_pattern}
              </div>
              <div className="mt-1 text-xs text-faint">
                {best.rule.algorithm === 'token_bucket'
                  ? `burst ${best.rule.bucket_capacity}, refills ${best.rule.refill_rate}/sec`
                  : `${best.rule.limit_count} per ${humanWindow(best.rule.window_seconds)}`}
                {' · '}specificity {best.score}
                {next && ` · beat ${next.rule.route_pattern} (${next.score})`}
              </div>
            </>
          )}
        </motion.div>
      </AnimatePresence>
    </div>
  )
}

export function ProjectDetail({ project, onMintKey, onRevokeKey, busy }) {
  const rules = project.rules ?? []
  const sorted = useMemo(
    () =>
      [...rules].sort((a, b) => {
        const s = (p) => (p === '*' ? 1 : p.endsWith('*') ? p.length - 1 : 1000 + p.length)
        return s(b.route_pattern) - s(a.route_pattern)
      }),
    [rules],
  )

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={spring}
      className="space-y-8"
    >
      <section>
        <h3 className="mb-2.5 text-[11px] font-semibold uppercase tracking-[0.08em] text-faint">
          Rules · evaluation order
        </h3>
        <div className="overflow-hidden rounded-xl border border-line bg-surface">
          {sorted.length === 0 ? (
            <p className="px-4 py-8 text-center text-[13px] text-faint">No rules.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-[13.5px]">
                <thead>
                  <tr className="border-b border-line text-[11px] uppercase tracking-[0.06em] text-faint">
                    <th className="px-4 py-3 text-left font-semibold">Pattern</th>
                    <th className="px-4 py-3 text-left font-semibold">Algorithm</th>
                    <th className="px-4 py-3 text-left font-semibold">Limit</th>
                    <th className="px-4 py-3 text-left font-semibold">Window</th>
                    <th className="px-4 py-3 text-left font-semibold">On</th>
                  </tr>
                </thead>
                <tbody>
                  {sorted.map((r) => (
                    <tr key={r.id ?? r.route_pattern} className="border-b border-line/60 last:border-0 hover:bg-raise">
                      <td className="px-4 py-3 font-mono text-[13px]">{r.route_pattern}</td>
                      <td className="px-4 py-3 text-dim">
                        {r.algorithm === 'token_bucket' ? 'token bucket' : 'sliding window'}
                      </td>
                      <td className="px-4 py-3 tabular-nums text-dim">
                        {r.algorithm === 'token_bucket' ? r.bucket_capacity : r.limit_count}
                      </td>
                      <td className="px-4 py-3 text-dim">
                        {r.algorithm === 'token_bucket'
                          ? `${r.refill_rate}/sec refill`
                          : humanWindow(r.window_seconds)}
                      </td>
                      <td className="px-4 py-3">
                        {r.enabled ? <Pill tone="good">on</Pill> : <Pill>off</Pill>}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        <p className="mt-2 text-xs text-faint">
          Exact match beats longest prefix, which beats <code>*</code>. Editing existing rules
          needs a rules write API — not built yet.
        </p>
      </section>

      <section>
        <h3 className="mb-2.5 text-[11px] font-semibold uppercase tracking-[0.08em] text-faint">
          Test a route
        </h3>
        <RouteTester rules={rules} projectName={project.name} />
      </section>

      <section>
        <div className="mb-2.5 flex items-center gap-3">
          <h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-faint">
            API keys
          </h3>
          <div className="flex-1" />
          <Button disabled={busy} onClick={() => onMintKey(project)}>
            + Mint new key
          </Button>
        </div>
        <div className="overflow-hidden rounded-xl border border-line bg-surface">
          <AnimatePresence initial={false}>
            {(project.keys ?? []).map((k) => (
              <motion.div
                key={k.id}
                layout
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0, height: 0 }}
                transition={spring}
                className="flex flex-wrap items-center gap-3 border-b border-line/60 px-4 py-3 last:border-0"
              >
                <code className="font-mono text-[13px] text-dim">{k.prefix}••••••••</code>
                {k.status === 'active' ? <Pill tone="good">active</Pill> : <Pill tone="bad">revoked</Pill>}
                <span className="text-xs text-faint">
                  {new Date(k.created_at).toLocaleDateString()}
                </span>
                <div className="flex-1" />
                {k.status === 'active' && (
                  <Button variant="danger" disabled={busy} onClick={() => onRevokeKey(project, k)}>
                    Revoke
                  </Button>
                )}
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
        <p className="mt-2 text-xs text-faint">
          To rotate without downtime: mint a new key, deploy it, then revoke the old one.
        </p>
      </section>
    </motion.div>
  )
}
