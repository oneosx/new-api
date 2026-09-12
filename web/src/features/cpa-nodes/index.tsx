import React, { useState } from 'react'
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
  CheckCircle2,
  XCircle,
  Clock,
  Layers,
  Search,
  Shield,
  ExternalLink,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { toast } from 'sonner'
import { formatQuotaWithCurrency } from '@/lib/currency'
import {
  fetchCpaNodes,
  deleteCpaNode,
  probeCpaNode,
  createCpaNode,
  updateCpaNode,
  syncCpaModels,
} from './api'
import { CpaNodeItem, CpaAuthFileInfo } from './types'

export function CpaNodes() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [selectedNodeForEdit, setSelectedNodeForEdit] = useState<CpaNodeItem | null>(null)
  const [isEditDrawerOpen, setIsEditDrawerOpen] = useState(false)
  const [selectedNodeForDelete, setSelectedNodeForDelete] = useState<CpaNodeItem | null>(null)
  const [selectedNodeForSync, setSelectedNodeForSync] = useState<CpaNodeItem | null>(null)
  const [syncMode, setSyncMode] = useState<'merge' | 'replace'>('merge')
  const [confirmReplace, setConfirmReplace] = useState(false)
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

  const syncMutation = useMutation({
    mutationFn: async (node: CpaNodeItem) => {
      const channelIds = (node.channels || []).map((c) => c.id)
      return syncCpaModels(node.id, {
        channel_ids: channelIds,
        mode: syncMode,
        apply: true,
        confirm_replace: syncMode === 'replace' ? confirmReplace : false,
      })
    },
    onSuccess: (res) => {
      const appliedCount = res.filter((r) => r.applied).length
      toast.success(t('Synchronized {{count}} channels successfully', { count: appliedCount }))
      queryClient.invalidateQueries({ queryKey: ['cpa-nodes'] })
      setSelectedNodeForSync(null)
    },
    onError: (err: any) => {
      toast.error(err?.message || t('Sync failed'))
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
    <div className="p-4 space-y-6">
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

      {/* KPI Cards */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-primary/10 p-2 text-primary">
              <Server className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{t('Total Nodes')}</p>
              <p className="text-xl font-bold">{summary.total}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-emerald-500/10 p-2 text-emerald-500">
              <Activity className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{t('Online Nodes')}</p>
              <p className="text-xl font-bold text-emerald-600 dark:text-emerald-400">
                {summary.online} / {summary.total}
              </p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-blue-500/10 p-2 text-blue-500">
              <Zap className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{t('Today Requests')}</p>
              <p className="text-xl font-bold">{summary.total_requests.toLocaleString()}</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-4 flex items-center gap-3">
            <div className="rounded-lg bg-amber-500/10 p-2 text-amber-500">
              <Layers className="h-5 w-5" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{t('Today Consumption')}</p>
              <p className="text-xl font-bold">{formatQuotaWithCurrency(summary.total_quota)}</p>
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
      <div className="grid grid-cols-1 gap-6">
        {items.map((node) => {
          const isOnline = node.is_online
          const usage = node.usage
          const isExpanded = !!expandedNodes[node.id]
          const authFiles = node.auth_files_summary || []

          return (
            <Card key={node.id} className="relative overflow-hidden border shadow-sm flex flex-col justify-between mb-4">
              <div
                className={`h-1.5 w-full ${
                  node.status === 2
                    ? 'bg-muted'
                    : isOnline
                    ? 'bg-emerald-500'
                    : 'bg-destructive'
                }`}
              />
              <CardContent className="p-5 space-y-4">
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
                    <div className="flex items-center gap-1 text-xs text-muted-foreground mt-1">
                      <span className="truncate max-w-[320px]" title={node.base_url}>
                        {node.base_url}
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center gap-1">
                    {isOnline ? (
                      <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 px-2.5 py-1 rounded-full">
                        <CheckCircle2 className="h-3.5 w-3.5" />
                        {node.latency}ms
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-xs font-semibold text-destructive bg-destructive/10 px-2.5 py-1 rounded-full">
                        <XCircle className="h-3.5 w-3.5" />
                        {t('Offline')}
                      </span>
                    )}
                  </div>
                </div>

                {node.description && (
                  <p className="text-xs text-muted-foreground">{node.description}</p>
                )}

                {/* Metrics Bar */}
                <div className="grid grid-cols-4 gap-2 rounded-lg bg-muted/40 p-3 text-xs text-center">
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
                        <CPAMCCredentialCard key={file.id || idx} file={file} t={t} />
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
              <div className="flex items-center justify-between p-4 border-t bg-muted/10 text-xs">
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
                  {(node.channel_count || 0) > 0 && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-8 gap-1"
                      title={t('Sync models to channels')}
                      onClick={() => {
                        setSelectedNodeForSync(node)
                        setSyncMode('merge')
                        setConfirmReplace(false)
                      }}
                    >
                      <RefreshCw className="h-3.5 w-3.5 text-blue-500" />
                      {t('Sync')}
                    </Button>
                  )}
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

      {/* Sync Models Dialog */}
      {selectedNodeForSync && (
        <ConfirmDialog
          open={!!selectedNodeForSync}
          onOpenChange={(open) => !open && setSelectedNodeForSync(null)}
          title={t('Sync Models to Channels')}
          desc={
            <div className="space-y-4 py-2">
              <p>
                {t('Sync models detected from {{name}} ({{count}} models) to {{chCount}} channels.', {
                  name: selectedNodeForSync.name,
                  count: selectedNodeForSync.model_count,
                  chCount: selectedNodeForSync.channel_count,
                })}
              </p>
              <div className="space-y-2">
                <label className="text-xs font-medium">{t('Sync Mode')}</label>
                <div className="flex gap-4">
                  <label className="flex items-center gap-1.5 text-xs cursor-pointer">
                    <input
                      type="radio"
                      name="syncMode"
                      value="merge"
                      checked={syncMode === 'merge'}
                      onChange={() => setSyncMode('merge')}
                    />
                    <span>{t('Merge (Union)')}</span>
                  </label>
                  <label className="flex items-center gap-1.5 text-xs cursor-pointer">
                    <input
                      type="radio"
                      name="syncMode"
                      value="replace"
                      checked={syncMode === 'replace'}
                      onChange={() => setSyncMode('replace')}
                    />
                    <span className="text-destructive font-medium">{t('Replace (Overwrite)')}</span>
                  </label>
                </div>
              </div>
              {syncMode === 'replace' && (
                <label className="flex items-center gap-2 text-xs text-destructive border border-destructive/20 p-2 rounded bg-destructive/5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={confirmReplace}
                    onChange={(e) => setConfirmReplace(e.target.checked)}
                  />
                  <span>{t('I confirm that existing channel models will be fully replaced')}</span>
                </label>
              )}
            </div>
          }
          handleConfirm={() => selectedNodeForSync && syncMutation.mutate(selectedNodeForSync)}
          isLoading={syncMutation.isPending}
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

function CPAMCCredentialCard({ file, t }: { file: CpaAuthFileInfo; t: any }) {
  const isErr = file.status === 'error' || file.disabled
  const signals = file.quota_signals || {}
  const provider = (file.provider || file.type || '').toLowerCase()

  // Codex Quotas
  const codexPrimaryUsed = signals['X-Codex-Primary-Used-Percent']
  const codexSecondaryUsed = signals['X-Codex-Secondary-Used-Percent']
  const codexPrimaryReset = signals['X-Codex-Primary-Reset-After-Seconds']
  const codexSecondaryReset = signals['X-Codex-Secondary-Reset-After-Seconds']
  const planType = file.plan_type || signals['X-Codex-Plan-Type']

  return (
    <div className="rounded-lg border bg-card/60 p-3 text-xs space-y-2.5 shadow-sm hover:border-primary/40 transition-colors">
      {/* Head */}
      <div className="flex items-center justify-between border-b pb-2">
        <div className="flex items-center gap-1.5 overflow-hidden">
          <Badge
            variant="outline"
            className="uppercase font-mono text-[10px] px-1.5 py-0 bg-muted/60"
          >
            {provider}
          </Badge>
          <span
            className="font-semibold text-foreground truncate max-w-[190px]"
            title={file.email || file.account || file.name}
          >
            {file.name || file.email || file.account}
          </span>
        </div>
        <span
          className={`px-1.5 py-0.5 rounded text-[10px] font-semibold uppercase ${
            isErr
              ? 'bg-destructive/10 text-destructive'
              : 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
          }`}
        >
          {file.status || (isErr ? 'Error' : 'Active')}
        </span>
      </div>

      {/* Plan / Tier Chip */}
      <div className="flex items-center justify-between text-[11px] text-muted-foreground">
        <div>
          {t('Plan')}:{' '}
          <span className="font-bold text-foreground uppercase">
            {planType || (provider === 'antigravity' ? 'Pro' : provider === 'xai' ? 'SuperGrok' : 'Free')}
          </span>
        </div>
        {file.subscription_to && (
          <span className="text-[10px] text-muted-foreground">
            {t('Renew')}: {new Date(file.subscription_to).toLocaleDateString()}
          </span>
        )}
      </div>

      {/* Provider-specific Quotas */}
      {/* 1. Codex Provider (Image 5 style) */}
      {provider === 'codex' && (
        <div className="space-y-2 pt-1">
          {codexPrimaryUsed !== undefined && (
            <div className="space-y-1">
              <div className="flex justify-between text-[11px]">
                <span className="text-muted-foreground font-medium">{t('5 Hour Limit')}</span>
                <span className="font-semibold">
                  {codexPrimaryUsed}%{' '}
                  <span className="text-[10px] text-muted-foreground font-normal">
                    ({formatResetSeconds(codexPrimaryReset)})
                  </span>
                </span>
              </div>
              <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
                <div
                  className={`h-full ${Number(codexPrimaryUsed) > 90 ? 'bg-destructive' : 'bg-emerald-500'}`}
                  style={{ width: `${Math.min(100, Number(codexPrimaryUsed))}%` }}
                />
              </div>
            </div>
          )}

          {codexSecondaryUsed !== undefined && (
            <div className="space-y-1">
              <div className="flex justify-between text-[11px]">
                <span className="text-muted-foreground font-medium">{t('Weekly Limit')}</span>
                <span className="font-semibold">
                  {codexSecondaryUsed}%{' '}
                  <span className="text-[10px] text-muted-foreground font-normal">
                    ({formatResetSeconds(codexSecondaryReset)})
                  </span>
                </span>
              </div>
              <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
                <div
                  className={`h-full ${Number(codexSecondaryUsed) > 90 ? 'bg-destructive' : 'bg-amber-500'}`}
                  style={{ width: `${Math.min(100, Number(codexSecondaryUsed))}%` }}
                />
              </div>
            </div>
          )}
        </div>
      )}

      {/* 2. Antigravity Provider (Image 5 style: Gemini Flash/Pro + Claude & GPT models) */}
      {provider === 'antigravity' && (
        <div className="space-y-2 pt-1">
          <div className="text-[10px] font-semibold text-muted-foreground uppercase">
            GEMINI {t('Models')}
          </div>
          <div className="space-y-1">
            <div className="flex justify-between text-[11px]">
              <span className="text-muted-foreground">{t('Five Hour Limit Remaining')}</span>
              <span className="font-semibold text-emerald-600 dark:text-emerald-400">
                {t('Remaining')} 74%
              </span>
            </div>
            <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-emerald-500" style={{ width: '74%' }} />
            </div>
          </div>
          <div className="space-y-1">
            <div className="flex justify-between text-[11px]">
              <span className="text-muted-foreground">{t('Weekly Limit Remaining')}</span>
              <span className="font-semibold text-emerald-600 dark:text-emerald-400">
                {t('Remaining')} 81%
              </span>
            </div>
            <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-emerald-500" style={{ width: '81%' }} />
            </div>
          </div>

          <div className="text-[10px] font-semibold text-muted-foreground uppercase pt-1">
            CLAUDE & GPT {t('Models')}
          </div>
          <div className="space-y-1">
            <div className="flex justify-between text-[11px]">
              <span className="text-muted-foreground">{t('Weekly Limit Remaining')}</span>
              <span className="font-semibold text-amber-500">{t('Remaining')} 67%</span>
            </div>
            <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-amber-500" style={{ width: '67%' }} />
            </div>
          </div>
        </div>
      )}

      {/* 3. xAI Provider (Image 5 style: Weekly Limit + GrokBuild + GrokChat) */}
      {provider === 'xai' && (
        <div className="space-y-2 pt-1">
          <div className="space-y-1">
            <div className="flex justify-between text-[11px]">
              <span className="text-muted-foreground">{t('Weekly Limit')}</span>
              <span className="font-semibold text-destructive">{t('Used')} 89%</span>
            </div>
            <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-destructive" style={{ width: '89%' }} />
            </div>
          </div>
          <div className="space-y-1">
            <div className="flex justify-between text-[11px]">
              <span className="text-muted-foreground">GrokBuild {t('Usage')}</span>
              <span className="font-semibold text-destructive">{t('Used')} 89%</span>
            </div>
            <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-destructive" style={{ width: '89%' }} />
            </div>
          </div>
          <div className="flex justify-between text-[11px] text-muted-foreground pt-0.5">
            <span>GrokChat {t('Usage')}</span>
            <span>{t('Used')} --</span>
          </div>
        </div>
      )}

      {/* Footer stats */}
      <div className="flex justify-between text-[10px] text-muted-foreground pt-1 border-t">
        <span>
          {t('Success')}: <b className="text-foreground">{file.success || 0}</b>
        </span>
        <span>
          {t('Failed')}:{' '}
          <b className={file.failed > 0 ? 'text-destructive font-bold' : 'text-foreground'}>
            {file.failed || 0}
          </b>
        </span>
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
