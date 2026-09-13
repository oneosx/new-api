export interface CodexRateLimitWindowInfo {
  used_percent: number
  reset_after: string
}

export interface CodexQuotaDetailedInfo {
  plan_type: string
  primary_window?: CodexRateLimitWindowInfo
  secondary_window?: CodexRateLimitWindowInfo
  available_reset_credits: number
}

export interface XaiQuotaDetailedInfo {
  weekly_used_percent: number
  weekly_reset_after?: string
  grok_build_used: number
  grok_chat_used: string
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
  codex_detail?: CodexQuotaDetailedInfo
  xai_detail?: XaiQuotaDetailedInfo
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
