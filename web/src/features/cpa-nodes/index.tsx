import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import {
  Server,
  Activity,
  Zap,
  Plus,
  RefreshCw,
  Edit2,
  Trash2,
  Clock,
  Layers,
  Search,
  Shield,
  ChevronDown,
  ChevronUp,
  RotateCcw,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { toast } from 'sonner'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { StatusBadge } from '@/components/status-badge'
import { CopyButton } from '@/components/copy-button'
import {
  fetchCpaNodes,
  deleteCpaNode,
  probeCpaNode,
  createCpaNode,
  updateCpaNode,
  resetCodexCredentialQuota,
  refreshSingleCpaCredential,
} from './api'
import type { CpaNodeItem, CpaAuthFileInfo } from './types'

export function CpaNodes() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [selectedNodeForEdit, setSelectedNodeForEdit] = useState<CpaNodeItem | null>(null)
  const [isEditDrawerOpen, setIsEditDrawerOpen] = useState(false)
  const [selectedNodeForDelete, setSelectedNodeForDelete] = useState<CpaNodeItem | null>(null)
  const [selectedCodexReset, setSelectedCodexReset] = useState<{ nodeId: number; file: CpaAuthFileInfo } | null>(null)
  const [expandedNodes, setExpandedNodes] = useState<Record<number, boolean>>({})

  // Form State
  const [formName, setFormName] = useState('')
  const [formBaseUrl, setFormBaseUrl] = useState('')
  const [formApiKey, setFormApiKey] = useState('')
  const [formStatus, setFormStatus] = useState(1)
  const [formWeight, setFormWeight] = useState(0)
  const [formDesc, setFormDesc] = useState('')

  const { data, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['cpa-nodes'],
    queryFn: () => fetchCpaNodes('today', true),
  })

  const probeMutation = useMutation({
    mutationFn: (id: number) => probeCpaNode(id),
    onSuccess: () => {
      toast.success(t('Probe completed'))
      queryClient.invalidateQueries({ queryKey: ['cpa-nodes'] })
    },
    onError: (err: any) => {
      toast.error(err?.message || t('Probe failed'))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteCpaNode(id),
    onSuccess: () => {
      toast.success(t('Deleted successfully'))
      queryClient.invalidateQueries({ queryKey: ['cpa-nodes'] })
      setSelectedNodeForDelete(null)
    },
    onError: (err: any) => {
      toast.error(err?.message || t('Delete failed'))
    },
  })

  const resetCodexMutation = useMutation({
    mutationFn: ({ nodeId, authFileId }: { nodeId: number; authFileId: string }) =>
      resetCodexCredentialQuota(nodeId, authFileId),
    onSuccess: () => {
      toast.success(t('Rate limit reset credit consumed successfully'))
      queryClient.invalidateQueries({ queryKey: ['cpa-nodes'] })
      setSelectedCodexReset(null)
    },
    onError: (err: any) => {
      toast.error(err?.message || t('Failed to reset quota'))
    },
  })

  const refreshSingleMutation = useMutation({
    mutationFn: ({ nodeId, authFileId }: { nodeId: number; authFileId: string }) =>
      refreshSingleCpaCredential(nodeId, authFileId),
    onSuccess: () => {
      toast.success(t('Credential quota refreshed'))
      queryClient.invalidateQueries({ queryKey: ['cpa-nodes'] })
    },
    onError: (err: any) => {
      toast.error(err?.message || t('Refresh failed'))
    },
  })

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (selectedNodeForEdit?.id) {
        return updateCpaNode(selectedNodeForEdit.id, {
          name: formName,
          base_url: formBaseUrl,
          api_key: formApiKey || undefined,
          status: formStatus,
          weight: formWeight,
          description: formDesc,
        })
      } else {
        return createCpaNode({
          name: formName,
          base_url: formBaseUrl,
          api_key: formApiKey,
          status: formStatus,
          weight: formWeight,
          description: formDesc,
        })
      }
    },
    onSuccess: () => {
      toast.success(t('Saved successfully'))
      queryClient.invalidateQueries({ queryKey: ['cpa-nodes'] })
      setIsEditDrawerOpen(false)
    },
    onError: (err: any) => {
      toast.error(err?.message || t('Save failed'))
    },
  })


  const toggleExpand = (nodeId: number) => {
    setExpandedNodes((prev) => ({ ...prev, [nodeId]: !prev[nodeId] }))
  }

  const openCreateModal = () => {
    setSelectedNodeForEdit(null)
    setFormName('')
    setFormBaseUrl('')
    setFormApiKey('')
    setFormStatus(1)
    setFormWeight(0)
    setFormDesc('')
    setIsEditDrawerOpen(true)
  }

  const openEditModal = (node: CpaNodeItem) => {
    setSelectedNodeForEdit(node)
    setFormName(node.name)
    setFormBaseUrl(node.base_url)
    setFormApiKey('')
    setFormStatus(node.status)
    setFormWeight(node.weight)
    setFormDesc(node.description || '')
    setIsEditDrawerOpen(true)
  }

  const items = (data?.items || []).filter(
    (item) =>
      item.name.toLowerCase().includes(search.toLowerCase()) ||
      item.base_url.toLowerCase().includes(search.toLowerCase()) ||
      (item.description && item.description.toLowerCase().includes(search.toLowerCase()))
  )

  const summary = data?.summary || { total: 0, online: 0, total_requests: 0, total_quota: 0 }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain p-4 space-y-4">
      {/* Header & KPI */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t('CPA Nodes')}</h1>
          <p className="text-sm text-muted-foreground">
            {t('Manage and monitor CLI Proxy API instances across servers')}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              refetch()
              toast.info(t('Refreshing...'))
            }}
            disabled={isLoading || isFetching}
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
            {t('Refresh')}
          </Button>
          <Button size="sm" onClick={openCreateModal}>
            <Plus className="mr-2 h-4 w-4" />
            {t('Add CPA Node')}
          </Button>
        </div>
      </div>

      {/* KPI Cards (Aligned with Solarized System Overview) */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Card className="relative overflow-hidden border-border/70 bg-card/75 backdrop-blur-sm shadow-sm transition-all hover:border-primary/40">
          <CardContent className="p-4 flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">{t('Total Nodes')}</p>
              <p className="text-2xl font-bold tracking-tight text-foreground font-mono">{summary.total}</p>
            </div>
            <div className="rounded-xl bg-primary/10 p-2.5 text-primary">
              <Server className="h-5 w-5" />
            </div>
          </CardContent>
        </Card>

        <Card className="relative overflow-hidden border-border/70 bg-card/75 backdrop-blur-sm shadow-sm transition-all hover:border-success/40">
          <CardContent className="p-4 flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">{t('Online Nodes')}</p>
              <p className="text-2xl font-bold tracking-tight text-success font-mono">
                {summary.online} <span className="text-xs font-normal text-muted-foreground">/ {summary.total}</span>
              </p>
            </div>
            <div className="rounded-xl bg-success/10 p-2.5 text-success">
              <Activity className="h-5 w-5" />
            </div>
          </CardContent>
        </Card>

        <Card className="relative overflow-hidden border-border/70 bg-card/75 backdrop-blur-sm shadow-sm transition-all hover:border-chart-1/40">
          <CardContent className="p-4 flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">{t('Today Requests')}</p>
              <p className="text-2xl font-bold tracking-tight text-foreground font-mono">
                {summary.total_requests.toLocaleString()}
              </p>
            </div>
            <div className="rounded-xl bg-chart-1/10 p-2.5 text-chart-1">
              <Zap className="h-5 w-5" />
            </div>
          </CardContent>
        </Card>

        <Card className="relative overflow-hidden border-border/70 bg-card/75 backdrop-blur-sm shadow-sm transition-all hover:border-warning/40">
          <CardContent className="p-4 flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">{t('Today Consumption')}</p>
              <p className="text-2xl font-bold tracking-tight text-foreground font-mono">
                {formatQuotaWithCurrency(summary.total_quota)}
              </p>
            </div>
            <div className="rounded-xl bg-warning/10 p-2.5 text-warning">
              <Layers className="h-5 w-5" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search Filter */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('Search nodes...')}
            className="pl-8"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      {/* Node Cards List */}
      <div className="grid grid-cols-1 gap-4">
        {items.map((node) => {
          const isOnline = node.is_online
          const usage = node.usage
          const isExpanded = !!expandedNodes[node.id]
          const authFiles = node.auth_files_summary || []

          return (
            <Card key={node.id} className="relative overflow-hidden border shadow-sm flex flex-col justify-between">
              <div
                className={`h-1 w-full ${
                  node.status === 2
                    ? 'bg-muted'
                    : isOnline
                    ? 'bg-emerald-500'
                    : 'bg-destructive'
                }`}
              />
              <CardContent className="p-4 space-y-4">
                {/* Node Title & Status */}
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-lg">{node.name}</span>
                      {node.status === 2 && (
                        <Badge variant="outline" className="text-muted-foreground">
                          {t('Disabled')}
                        </Badge>
                      )}
                      {node.version && (
                        <Badge variant="secondary" className="text-xs font-mono">
                          {node.version}
                        </Badge>
                      )}
                    </div>
                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground mt-1">
                      <span className="font-mono truncate max-w-[320px]" title={node.base_url}>
                        {node.base_url}
                      </span>
                      <CopyButton value={node.base_url} className="h-5 w-5" />
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    {isOnline ? (
                      <StatusBadge
                        variant="success"
                        label={`${node.latency}ms`}
                        size="sm"
                      />
                    ) : (
                      <StatusBadge
                        variant="danger"
                        label={t('Offline')}
                        size="sm"
                      />
                    )}
                  </div>
                </div>

                {node.description && (
                  <p className="text-xs text-muted-foreground">{node.description}</p>
                )}

                {/* Metrics Bar */}
                <div className="grid grid-cols-4 gap-2 rounded-lg bg-white/30 dark:bg-white/5 p-3 text-xs text-center">
                  <div>
                    <div className="text-muted-foreground">{t('Today Requests')}</div>
                    <div className="font-bold text-sm mt-0.5">{(usage?.requests || 0).toLocaleString()}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">{t('Today Consumption')}</div>
                    <div className="font-bold text-sm mt-0.5">
                      {formatQuotaWithCurrency(usage?.quota || 0)}
                    </div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">{t('Models')}</div>
                    <div className="font-bold text-sm mt-0.5">{node.model_count || 0}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">{t('Channels')}</div>
                    <div className="font-bold text-sm mt-0.5">{node.channel_count || 0}</div>
                  </div>
                </div>

                {/* Credentials & Quotas Section (CPAMC 1:1 Style) */}
                <div className="space-y-3 pt-2 border-t">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold flex items-center gap-1.5">
                      <Shield className="h-3.5 w-3.5 text-primary" />
                      {t('Mounted Credentials')} ({authFiles.length})
                    </span>
                    {authFiles.length > 0 && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-6 text-xs px-2 text-muted-foreground hover:text-foreground"
                        onClick={() => toggleExpand(node.id)}
                      >
                        {isExpanded ? (
                          <>
                            {t('Collapse')} <ChevronUp className="ml-1 h-3 w-3" />
                          </>
                        ) : (
                          <>
                            {t('Expand Quotas')} <ChevronDown className="ml-1 h-3 w-3" />
                          </>
                        )}
                      </Button>
                    )}
                  </div>

                  {authFiles.length === 0 ? (
                    <p className="text-xs text-muted-foreground italic py-1">
                      {t('No auth files detected or management API not configured')}
                    </p>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                      {authFiles.slice(0, isExpanded ? authFiles.length : 3).map((file, idx) => (
                        <CPAMCCredentialCard
                          key={file.id || idx}
                          file={file}
                          t={t}
                          onRefreshSingle={() =>
                            refreshSingleMutation.mutate({
                              nodeId: node.id,
                              authFileId: file.id || file.name,
                            })
                          }
                          isRefreshing={refreshSingleMutation.isPending}
                          onResetCodex={() => setSelectedCodexReset({ nodeId: node.id, file })}
                        />
                      ))}
                    </div>
                  )}

                  {!isExpanded && authFiles.length > 3 && (
                    <p
                      className="text-xs text-primary cursor-pointer hover:underline text-center pt-1"
                      onClick={() => toggleExpand(node.id)}
                    >
                      {t('...and {{count}} more credentials', { count: authFiles.length - 3 })}
                    </p>
                  )}
                </div>

                {node.last_error && !isOnline && (
                  <p className="text-xs text-destructive truncate bg-destructive/5 p-2 rounded" title={node.last_error}>
                    {node.last_error}
                  </p>
                )}
              </CardContent>

              {/* Card Footer Actions */}
              <div className="flex flex-wrap items-center justify-between gap-2 p-4 border-t bg-muted/10 text-xs">
                <div className="flex items-center gap-1 text-muted-foreground">
                  <Clock className="h-3.5 w-3.5" />
                  <span>
                    {node.last_check_at
                      ? new Date(node.last_check_at * 1000).toLocaleTimeString()
                      : t('Never')}
                  </span>
                </div>

                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-8 gap-1"
                    title={t('Test')}
                    onClick={() => probeMutation.mutate(node.id)}
                    disabled={probeMutation.isPending}
                  >
                    <Zap className="h-3.5 w-3.5 text-amber-500" />
                    {t('Test')}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    title={t('Edit')}
                    onClick={() => openEditModal(node)}
                  >
                    <Edit2 className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                    title={t('Delete')}
                    onClick={() => setSelectedNodeForDelete(node)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </Card>
          )
        })}
      </div>

      {items.length === 0 && !isLoading && (
        <div className="text-center py-12 border rounded-lg border-dashed">
          <Server className="mx-auto h-8 w-8 text-muted-foreground opacity-50" />
          <p className="mt-2 text-sm text-muted-foreground">{t('No CPA nodes found')}</p>
          <Button size="sm" className="mt-4" onClick={openCreateModal}>
            <Plus className="mr-2 h-4 w-4" />
            {t('Add your first node')}
          </Button>
        </div>
      )}

      {/* Delete Dialog */}
      <ConfirmDialog
        open={!!selectedNodeForDelete}
        onOpenChange={(open) => !open && setSelectedNodeForDelete(null)}
        title={t('Delete CPA Node')}
        desc={t(
          'Are you sure you want to delete this CPA node? Associated channels will not be deleted.'
        )}
        handleConfirm={() => selectedNodeForDelete && deleteMutation.mutate(selectedNodeForDelete.id)}
        isLoading={deleteMutation.isPending}
        destructive
      />

      {/* Codex Reset Credit Dialog */}
      {selectedCodexReset && (
        <ConfirmDialog
          open={!!selectedCodexReset}
          onOpenChange={(open) => !open && setSelectedCodexReset(null)}
          title={t('Consume Rate Limit Reset Credit')}
          desc={t(
            'Are you sure you want to consume 1 rate limit reset credit for {{name}}? This will reset the 5-hour and weekly rate limits and cannot be undone.',
            { name: selectedCodexReset.file.email || selectedCodexReset.file.name }
          )}
          handleConfirm={() =>
            resetCodexMutation.mutate({
              nodeId: selectedCodexReset.nodeId,
              authFileId: selectedCodexReset.file.id || selectedCodexReset.file.name,
            })
          }
          isLoading={resetCodexMutation.isPending}
          destructive
        />
      )}


      {/* Edit / Create Dialog */}
      {isEditDrawerOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg border space-y-4">
            <h2 className="text-lg font-bold">
              {selectedNodeForEdit ? t('Edit CPA Node') : t('Add CPA Node')}
            </h2>
            <div className="space-y-3">
              <div>
                <label className="text-xs font-medium">{t('Node Name')}</label>
                <Input
                  value={formName}
                  onChange={(e) => setFormName(e.target.value)}
                  placeholder={t('e.g. US-App')}
                  className="mt-1"
                />
              </div>
              <div>
                <label className="text-xs font-medium">{t('Base URL')}</label>
                <Input
                  value={formBaseUrl}
                  onChange={(e) => setFormBaseUrl(e.target.value)}
                  placeholder="https://cpa-us-app.oneosx.com"
                  className="mt-1"
                />
              </div>
              <div>
                <label className="text-xs font-medium">
                  {t('API Key')}
                  {selectedNodeForEdit && (
                    <span className="ml-1 text-muted-foreground font-normal">
                      ({t('Leave empty to keep unchanged')})
                    </span>
                  )}
                </label>
                <Input
                  type="password"
                  value={formApiKey}
                  onChange={(e) => setFormApiKey(e.target.value)}
                  placeholder="Bearer token"
                  className="mt-1"
                />
              </div>
              <div>
                <label className="text-xs font-medium">{t('Description')}</label>
                <Input
                  value={formDesc}
                  onChange={(e) => setFormDesc(e.target.value)}
                  placeholder={t('e.g. Antigravity OAuth')}
                  className="mt-1"
                />
              </div>
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 text-xs cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formStatus === 1}
                    onChange={(e) => setFormStatus(e.target.checked ? 1 : 2)}
                  />
                  <span>{t('Enabled')}</span>
                </label>
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2 border-t">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setIsEditDrawerOpen(false)}
                disabled={saveMutation.isPending}
              >
                {t('Cancel')}
              </Button>
              <Button
                size="sm"
                onClick={() => saveMutation.mutate()}
                disabled={saveMutation.isPending || !formName || !formBaseUrl}
              >
                {t('Save')}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function CPAMCCredentialCard({
  file,
  t,
  onRefreshSingle,
  isRefreshing,
  onResetCodex,
}: {
  file: CpaAuthFileInfo
  t: any
  onRefreshSingle?: () => void
  isRefreshing?: boolean
  onResetCodex?: () => void
}) {
  const isErr = file.status === 'error' || file.disabled
  const signals = file.quota_signals || {}
  const provider = (file.provider || file.type || '').toLowerCase()

  // Real-time Codex data from WHAM API, fallback to signals
  const codexDetail = file.codex_detail
  const codexPrimaryUsed = codexDetail?.primary_window?.used_percent ?? signals['X-Codex-Primary-Used-Percent']
  const codexSecondaryUsed = codexDetail?.secondary_window?.used_percent ?? signals['X-Codex-Secondary-Used-Percent']
  const codexPrimaryReset = codexDetail?.primary_window?.reset_after ?? formatResetSeconds(signals['X-Codex-Primary-Reset-After-Seconds'])
  const codexSecondaryReset = codexDetail?.secondary_window?.reset_after ?? formatResetSeconds(signals['X-Codex-Secondary-Reset-After-Seconds'])
  const antigravityDetail = file.antigravity_detail
  const displayPlanType = antigravityDetail?.plan || codexDetail?.plan_type || file.plan_type || signals['X-Codex-Plan-Type']
  const planType = codexDetail?.plan_type || file.plan_type || signals['X-Codex-Plan-Type']

  // Real-time xAI data from Billing API
  const xaiDetail = file.xai_detail

  return (
    <div className="group relative flex flex-col justify-between rounded-xl border border-border/70 bg-card/75 p-3.5 text-xs shadow-sm backdrop-blur-sm transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/50 hover:shadow-md space-y-3">
      {/* Head */}
      {/* Head: Icon + Name + Status Pill */}
      <div className="flex items-center justify-between gap-2 border-b border-border/50 pb-2.5">
        <div className="flex items-center gap-2 min-w-0">
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-muted/80 text-[10px] font-bold uppercase tracking-wider text-muted-foreground border border-border/50">
            {provider === 'antigravity' ? 'AG' : provider === 'codex' ? 'CX' : provider === 'xai' ? 'X' : provider.slice(0, 2)}
          </span>
          <span
            className="font-mono text-xs font-semibold text-foreground truncate max-w-[180px] sm:max-w-[200px]"
            title={file.name || file.email || file.account}
          >
            {file.name || file.email || file.account}
          </span>
        </div>
        <StatusBadge
          variant={file.disabled ? 'neutral' : file.status === 'error' ? 'danger' : 'success'}
          label={file.disabled ? 'Disabled' : file.status === 'error' ? 'Error' : 'Active'}
          size="sm"
        />
      </div>

      {/* Plan / Tier Chip */}
      <div className="flex items-center justify-between text-[11px] text-muted-foreground">
        <div>
          {t('Plan')}:{' '}
          <span className="font-bold text-foreground uppercase">
            {displayPlanType || '--'}
          </span>
        </div>
        {file.subscription_to && (
          <span className="text-[10px] text-muted-foreground">
            {t('Renew')}: {new Date(file.subscription_to).toLocaleDateString()}
          </span>
        )}
      </div>

      {/* Provider-specific Quotas */}
      {/* 1. Codex Provider (1:1 with CPAMC QuotaCard: 剩余百分比与对应颜色) */}
      {provider === 'codex' && (
        <div className="space-y-2 pt-1">
          {codexPrimaryUsed !== undefined && (
            <QuotaProgress
              label={t('5 Hour Limit')}
              remaining={codexDetail?.primary_window?.remaining_percent ?? Math.max(0, 100 - Number(codexPrimaryUsed))}
              resetAfter={codexPrimaryReset}
            />
          )}

          {codexSecondaryUsed !== undefined && (
            <QuotaProgress
              label={t('Weekly Limit')}
              remaining={codexDetail?.secondary_window?.remaining_percent ?? Math.max(0, 100 - Number(codexSecondaryUsed))}
              resetAfter={codexSecondaryReset}
            />
          )}
        </div>
      )}

      {/* 2. Antigravity Provider (1:1 with CPAMC) */}
      {provider === 'antigravity' && (
        <div className="space-y-2.5 pt-1">
          {antigravityDetail?.groups && antigravityDetail.groups.length > 0 ? (
            antigravityDetail.groups.map((group) => (
              <div key={group.id} className="space-y-1.5 rounded bg-muted/20 p-2 border border-border/40">
                <div className="flex items-center justify-between text-[11px] font-semibold text-foreground">
                  <span>{group.label}</span>
                </div>
                {group.description && (
                  <div className="text-[10px] text-muted-foreground line-clamp-1" title={group.description}>
                    {group.description}
                  </div>
                )}
                <div className="space-y-1.5 pt-1">
                  {group.buckets.map((b) => (
                    <QuotaProgress
                      key={b.id}
                      label={b.label}
                      remaining={b.remaining_percent}
                      resetAfter={b.reset_after}
                    />
                  ))}
                </div>
              </div>
            ))
          ) : (
            <div className="py-2 text-[11px] text-muted-foreground">
              {t('Quota data is unavailable until CPA returns a supported quota summary')}
            </div>
          )}
        </div>
      )}

      {/* 3. xAI Provider (1:1 with CPAMC: 显示已用比例并配合剩余水位条) */}
      {provider === 'xai' && (
        <div className="space-y-2 pt-1">
          {xaiDetail ? (
            <>
              <QuotaProgress
                label={t('Weekly Limit')}
                usedLabel={`已用 ${xaiDetail.weekly_used_percent}%`}
                remaining={xaiDetail.weekly_remaining_percent ?? Math.max(0, 100 - xaiDetail.weekly_used_percent)}
                resetAfter={xaiDetail.weekly_reset_after ? new Date(xaiDetail.weekly_reset_after).toLocaleDateString() : undefined}
              />
              <QuotaProgress
                label={`GrokBuild ${t('Usage')}`}
                usedLabel={`已用 ${xaiDetail.grok_build_used}%`}
                remaining={xaiDetail.grok_build_remaining ?? Math.max(0, 100 - xaiDetail.grok_build_used)}
              />
            </>
          ) : (
            <div className="py-2 text-[11px] text-muted-foreground">
              {t('Quota data is unavailable until CPA returns a supported quota summary')}
            </div>
          )}
        </div>
      )}

      {/* Actions & Footer Stats */}
      <div className="space-y-2 pt-1 border-t border-border/50">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 text-[10px] text-muted-foreground font-mono">
            <span>
              {t('Success')}: <b className="text-foreground font-semibold">{file.success || 0}</b>
            </span>
            <span>•</span>
            <span>
              {t('Failed')}:{' '}
              <b className={file.failed > 0 ? 'text-destructive font-bold' : 'text-foreground font-semibold'}>
                {file.failed || 0}
              </b>
            </span>
          </div>

          <div className="flex items-center gap-1.5">
            {provider === 'codex' && onResetCodex && (
              <Button
                variant="outline"
                size="sm"
                className="h-6 text-[11px] px-2 gap-1 border-primary/40 hover:bg-primary/10 text-primary font-medium shadow-none"
                onClick={onResetCodex}
                title={t('Consume 1 reset credit to reset 5-hour and weekly limits')}
              >
                <RotateCcw className="h-3 w-3" />
                {t('Reset Quota')}
              </Button>
            )}
            {onRefreshSingle && (
              <Button
                variant="outline"
                size="sm"
                className="h-6 text-[11px] px-2 gap-1 text-muted-foreground hover:text-foreground hover:bg-muted/80 shadow-none"
                onClick={onRefreshSingle}
                disabled={isRefreshing}
                title={t('Refresh single credential quota')}
              >
                <RefreshCw className={`h-3 w-3 ${isRefreshing ? 'animate-spin' : ''}`} />
                {t('Refresh Quota')}
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function QuotaProgress({
  label,
  remaining,
  usedLabel,
  resetAfter,
}: {
  label: string
  remaining: number
  usedLabel?: string
  resetAfter?: string
}) {
  const normalizedRemaining = Number.isFinite(remaining) ? Math.max(0, Math.min(100, remaining)) : 0
  // 语义化状态配色：与全局 Solarized Token 对齐 (Success / Warning / Destructive)
  const barColor =
    normalizedRemaining >= 70
      ? 'bg-success'
      : normalizedRemaining >= 30
        ? 'bg-warning'
        : 'bg-destructive'
  const textColor =
    normalizedRemaining < 30
      ? 'text-destructive'
      : normalizedRemaining < 70
        ? 'text-warning'
        : 'text-success'

  return (
    <div className="space-y-1.5">
      <div className="flex justify-between text-[11px]">
        <span className="text-muted-foreground font-medium">{label}</span>
        <span className={`font-mono font-semibold ${textColor}`}>
          {usedLabel || `剩余 ${normalizedRemaining}%`}{' '}
          {resetAfter ? <span className="text-[10px] font-normal text-muted-foreground ml-1">({resetAfter})</span> : null}
        </span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted/60">
        <div
          className={`h-full rounded-full transition-all duration-300 ${barColor}`}
          style={{ width: `${normalizedRemaining}%` }}
        />
      </div>
    </div>
  )
}

function formatResetSeconds(seconds: any): string {
  const sec = Number(seconds)
  if (isNaN(sec) || sec <= 0) return 'OK'
  if (sec < 60) return `${sec}s`
  if (sec < 3600) return `${Math.ceil(sec / 60)}m`
  const hours = Math.floor(sec / 3600)
  const mins = Math.floor((sec % 3600) / 60)
  return `${hours}h ${mins}m`
}
