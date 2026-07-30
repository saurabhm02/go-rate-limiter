
export const API_BASE =
  import.meta.env.VITE_API_BASE || 'https://gorate.onrender.com'

const TOKEN_KEY = 'gorate.token'

export const getToken = () => {
  try {
    return sessionStorage.getItem(TOKEN_KEY) || ''
  } catch {
    return ''
  }
}

export const setToken = (t) => {
  try {
    t ? sessionStorage.setItem(TOKEN_KEY, t) : sessionStorage.removeItem(TOKEN_KEY)
  } catch {
  }
}

export class ApiError extends Error {
  constructor(status, code) {
    super(code || `HTTP ${status}`)
    this.status = status
    this.code = code
  }
}

async function request(path, { method = 'GET', body } = {}) {
  let res
  try {
    res = await fetch(API_BASE + path, {
      method,
      headers: {
        Authorization: `Bearer ${getToken()}`,
        ...(body ? { 'Content-Type': 'application/json' } : {}),
      },
      body: body ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError(0, 'unreachable')
  }

  if (res.status === 204) return null

  let data = null
  try {
    data = await res.json()
  } catch {
  }
  if (!res.ok) throw new ApiError(res.status, data?.error)
  return data
}

export const listProjects = () =>
  request('/v1/admin/projects').then((d) => d?.projects ?? [])

export const createProject = (payload) =>
  request('/v1/admin/projects', { method: 'POST', body: payload })

export const addKey = (projectId, payload) =>
  request(`/v1/admin/projects/${projectId}/keys`, { method: 'POST', body: payload })

export const revokeKey = (projectId, keyId) =>
  request(`/v1/admin/projects/${projectId}/keys/${keyId}`, { method: 'DELETE' })


export async function generateKey(projectName) {
  const bytes = new Uint8Array(24)
  crypto.getRandomValues(bytes)
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('')
  const slug = projectName.replace(/[^a-z0-9]+/gi, '').toLowerCase() || 'project'

  const raw = `rl_${slug}_${hex}`
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(raw))
  const hash = [...new Uint8Array(digest)].map((b) => b.toString(16).padStart(2, '0')).join('')

  return { raw, hash, prefix: `rl_${slug}_` }
}


export function specificity(pattern, route) {
  if (pattern === '*') return 1
  if (pattern.endsWith('*')) {
    const prefix = pattern.slice(0, -1)
    return route.startsWith(prefix) ? prefix.length : 0
  }
  return route === pattern ? 1000 + pattern.length : 0
}

export function resolveRoute(rules, route) {
  const scored = rules
    .filter((r) => r.enabled)
    .map((r) => ({ rule: r, score: specificity(r.route_pattern, route) }))
    .filter((x) => x.score > 0)
    .sort((a, b) => b.score - a.score)
  return { best: scored[0] ?? null, next: scored[1] ?? null }
}

export function humanWindow(seconds) {
  const s = Number(seconds)
  if (!s) return '—'
  if (s % 3600 === 0) return `${s / 3600} ${s / 3600 === 1 ? 'hour' : 'hours'}`
  if (s % 60 === 0) return `${s / 60} min`
  return `${s} sec`
}
