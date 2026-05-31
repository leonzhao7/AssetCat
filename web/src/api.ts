import type { Asset, AssetSummary, CreateAssetPayload, RiskFinding } from './types'

const API_BASE = import.meta.env.VITE_API_BASE ?? ''

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  })

  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`
    try {
      const body = await response.json()
      message = body.message ?? body.error ?? message
    } catch {
      // Keep the HTTP status text when the response is not JSON.
    }
    throw new Error(message)
  }

  if (response.status === 204) {
    return undefined as T
  }
  return response.json() as Promise<T>
}

export function fetchSummary() {
  return request<AssetSummary>('/summary')
}

export function fetchAssets(filters: { q?: string; severity?: string } = {}) {
  const params = new URLSearchParams()
  if (filters.q) params.set('q', filters.q)
  if (filters.severity) params.set('severity', filters.severity)
  const suffix = params.toString() ? `?${params}` : ''
  return request<Asset[]>(`/assets${suffix}`)
}

export function createAsset(payload: CreateAssetPayload) {
  return request<Asset>('/assets', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export function addRisk(assetID: string, domainName: string, payload: RiskFinding) {
  return request<Asset>(`/assets/${encodeURIComponent(assetID)}/domains/${encodeURIComponent(domainName)}/risks`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}
