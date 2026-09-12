import { api } from '@/lib/api'
import { CpaNodeItem, CpaNodesResponse, CpaSyncDiffResult } from './types'

export async function fetchCpaNodes(window = 'today', fresh = false): Promise<CpaNodesResponse> {
  const res = await api.get<{ data: CpaNodesResponse }>('/api/cpa-node', {
    params: { window, fresh: fresh ? '1' : '0' },
  })
  return res.data?.data || { items: [], summary: { total: 0, online: 0, total_requests: 0, total_quota: 0 } }
}

export async function fetchCpaNode(id: number, window = 'today', fresh = false): Promise<CpaNodeItem> {
  const res = await api.get<{ data: CpaNodeItem }>(`/api/cpa-node/${id}`, {
    params: { window, fresh: fresh ? '1' : '0' },
  })
  return res.data?.data
}

export async function createCpaNode(data: Partial<CpaNodeItem> & { api_key?: string }): Promise<CpaNodeItem> {
  const res = await api.post<{ data: CpaNodeItem }>('/api/cpa-node', data)
  return res.data?.data
}

export async function updateCpaNode(id: number, data: Partial<CpaNodeItem> & { api_key?: string }): Promise<CpaNodeItem> {
  const res = await api.put<{ data: CpaNodeItem }>(`/api/cpa-node/${id}`, data)
  return res.data?.data
}

export async function deleteCpaNode(id: number): Promise<void> {
  await api.delete(`/api/cpa-node/${id}`)
}

export async function probeCpaNode(id: number): Promise<unknown> {
  const res = await api.post(`/api/cpa-node/${id}/test`)
  return res.data
}

export async function syncCpaModels(id: number, req: {
  channel_ids: number[]
  mode: 'merge' | 'replace'
  apply: boolean
  confirm_replace?: boolean
}): Promise<CpaSyncDiffResult[]> {
  const res = await api.post<{ data: CpaSyncDiffResult[] }>(`/api/cpa-node/${id}/sync-channels`, req)
  return res.data?.data || []
}
