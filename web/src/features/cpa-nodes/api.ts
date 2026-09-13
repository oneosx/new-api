import { api } from '@/lib/api'
import type { CpaNodeItem, CpaNodesResponse, CpaAuthFileInfo } from './types'

export async function fetchCpaNodes(window = 'today', fresh = false): Promise<CpaNodesResponse> {
  const res = await api.get<{ success: boolean; message?: string; data: CpaNodesResponse }>('/api/cpa-node', {
    params: { window, fresh: fresh ? '1' : '0' },
  })
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'Failed to fetch CPA nodes')
  }
  return res.data?.data || { items: [], summary: { total: 0, online: 0, total_requests: 0, total_quota: 0 } }
}

export async function fetchCpaNode(id: number, window = 'today', fresh = false): Promise<CpaNodeItem> {
  const res = await api.get<{ success: boolean; message?: string; data: CpaNodeItem }>(`/api/cpa-node/${id}`, {
    params: { window, fresh: fresh ? '1' : '0' },
  })
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'Failed to fetch CPA node')
  }
  return res.data?.data
}

export async function createCpaNode(data: Partial<CpaNodeItem> & { api_key?: string }): Promise<CpaNodeItem> {
  const res = await api.post<{ success: boolean; message?: string; data: CpaNodeItem }>('/api/cpa-node', data)
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'Failed to create CPA node')
  }
  return res.data?.data
}

export async function updateCpaNode(id: number, data: Partial<CpaNodeItem> & { api_key?: string }): Promise<CpaNodeItem> {
  const res = await api.put<{ success: boolean; message?: string; data: CpaNodeItem }>(`/api/cpa-node/${id}`, data)
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'Failed to update CPA node')
  }
  return res.data?.data
}

export async function deleteCpaNode(id: number): Promise<void> {
  const res = await api.delete<{ success: boolean; message?: string }>(`/api/cpa-node/${id}`)
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'Failed to delete CPA node')
  }
}

export async function probeCpaNode(id: number): Promise<any> {
  const res = await api.post<{ success: boolean; message?: string; data: any }>(`/api/cpa-node/${id}/test`)
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'Probe failed')
  }
  return res.data?.data
}


export async function resetCodexCredentialQuota(nodeId: number, authFileId: string): Promise<any> {
  const res = await api.post<{ success: boolean; message?: string; data: any }>(`/api/cpa-node/${nodeId}/reset-codex-quota`, {
    auth_file_id: authFileId,
    confirm: true,
  })
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'Failed to reset quota')
  }
  return res.data?.data
}

export async function refreshSingleCpaCredential(nodeId: number, authFileId: string): Promise<CpaAuthFileInfo> {
  const res = await api.post<{ success: boolean; message?: string; data: CpaAuthFileInfo }>(
    `/api/cpa-node/${nodeId}/refresh-credential-quota`,
    {},
    { params: { auth_file_id: authFileId } }
  )
  if (!res.data?.success) {
    throw new Error(res.data?.message || 'Failed to refresh credential quota')
  }
  return res.data?.data
}
