export interface AntigravityQuotaGroupInfo {
  name: string
  models: string[]
  five_hour_limit_remaining?: string
  five_hour_limit_percent: number
  five_hour_reset_after?: string
  weekly_limit_remaining?: string
  weekly_limit_percent: number
  weekly_reset_after?: string
}

export interface CpaAuthFileInfo {
  id: string
  name: string
  provider: string
  type: string
  status: string
  disabled: boolean
  email?: string
  account?: string
  plan_type?: string
  subscription_to?: string
  quota_signals?: Record<string, any>
  model_quotas?: Record<string, any>
  success: number
  failed: number
  last_refresh?: string
  antigravity_groups?: AntigravityQuotaGroupInfo[]
}

export interface CpaNodeChannelInfo {
  id: number
  name: string
  type: number
  status: number
  model_count: number
  models: string[]
}

export interface CpaNodeUsage {
  window: string
  requests: number
  prompt_tokens: number
  completion_tokens: number
  quota: number
  error_count: number
  error_rate: number
}

export interface CpaNodeItem {
  id: number
  name: string
  base_url: string
  normalized_url: string
  has_api_key: boolean
  status: number // 1: enabled, 2: disabled
  weight: number
  description: string
  created_time: number
  updated_time: number
  is_online: boolean
  latency: number
  http_status: number
  version: string
  model_count: number
  models: string
  auth_files_count?: number
  auth_files_summary?: CpaAuthFileInfo[]
  last_error: string
  last_check_at: number
  channel_count: number
  channels?: CpaNodeChannelInfo[]
  usage?: CpaNodeUsage
}

export interface CpaNodesResponse {
  items: CpaNodeItem[]
  summary: {
    total: number
    online: number
    total_requests: number
    total_quota: number
  }
}

export interface CpaSyncDiffResult {
  channel_id: number
  channel_name: string
  original: string[]
  target: string[]
  added: string[]
  removed: string[]
  kept: string[]
  applied: boolean
  error_message?: string
}
