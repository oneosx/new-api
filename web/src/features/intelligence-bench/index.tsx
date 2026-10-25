import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BrainCircuit,
  Play,
  RotateCcw,
  CheckCircle2,
  XCircle,
  Clock,
  Sparkles,
  ExternalLink,
  Code2,
  RefreshCw,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import { api, getFreshAuthHeaders } from '@/lib/api'

export type ProbeKind = 'candy' | 'pelican_svg' | 'pelican_3d' | 'cf_ship_3d' | 'qq_speed_3d'

export interface ChannelEvalResult {
  success: boolean
  channel_id: number
  channel_name: string
  channel_type: number
  model: string
  probe_type: ProbeKind
  probe_name: string
  category: string
  prompt: string
  raw_output: string
  reasoning_output?: string
  reasoning_tokens: number
  prompt_tokens: number
  total_tokens: number
  duration_ms: number
  analysis: {
    success: boolean
    score: number
    extracted_data?: string
    summary: string
    html_code?: string
    svg_code?: string
  }
  error_message?: string
}

const PROBE_OPTIONS: { id: ProbeKind; label: string; tag: string; desc: string }[] = [
  {
    id: 'candy',
    label: '🍬 动态糖果最坏情况题',
    tag: '数学 & 深度CoT',
    desc: '极值互斥抽屉原理，参数实时随机防背题，双重门禁检验 Reasoning Tokens',
  },
  {
    id: 'pelican_svg',
    label: '🚲 鹈鹕骑车 (2D SVG)',
    tag: '空间拓扑 & 矢量绘图',
    desc: '经典 Simon Willison 基准，测试空间几何坐标推导与手写 SVG 能力',
  },
  {
    id: 'pelican_3d',
    label: '🚲 鹈鹕骑车 (3D 游戏)',
    tag: '极限 3D 工程代码',
    desc: 'Claude Opus 5.5 测试集：单文件 Three.js 程序化生成完整可玩 3D 页面',
  },
  {
    id: 'cf_ship_3d',
    label: '🔫 穿越火线·运输船 (3D FPS)',
    tag: '极限 3D 工程代码',
    desc: '单文件还原长条甲板、集装箱、弹道与对战，纯代码程序化构建',
  },
  {
    id: 'qq_speed_3d',
    label: '🏎️ QQ 飞车 (3D 漂移)',
    tag: '极限 3D 工程代码',
    desc: '单文件还原赛道、Shift 漂移手感与氮气喷射，支持交互试玩',
  },
]

import { SectionPageLayout } from '@/components/layout'

export function IntelligenceBench() {
  const { t } = useTranslation()
  const [models, setModels] = useState<string[]>([])
  const [selectedModel, setSelectedModel] = useState<string>('')
  const [selectedProbe, setSelectedProbe] = useState<ProbeKind>('candy')
  const [isRunning, setIsRunning] = useState(false)
  const [results, setResults] = useState<ChannelEvalResult[]>([])
  const [activePreview, setActivePreview] = useState<{
    title: string
    type: 'svg' | 'html'
    content: string
  } | null>(null)

  // Fetch available models from channels
  useEffect(() => {
    async function loadModels() {
      try {
        const res = await api.get<{ success: boolean; data: string[] }>('/api/channel/models_enabled')
        if (res.data?.success && Array.isArray(res.data.data) && res.data.data.length > 0) {
          setModels(res.data.data)
          setSelectedModel(res.data.data[0])
        } else {
          // Fallback to channel list models
          const fallbackRes = await api.get<{ success: boolean; data: string[] }>('/api/channel/models')
          if (fallbackRes.data?.success && Array.isArray(fallbackRes.data.data)) {
            setModels(fallbackRes.data.data)
            if (fallbackRes.data.data.length > 0) setSelectedModel(fallbackRes.data.data[0])
          }
        }
      } catch (e) {
        console.error('Failed to load models:', e)
      }
    }
    loadModels()
  }, [])

  const handleRunMatrixEval = async () => {
    if (!selectedModel) {
      toast.error('请选择要评测的目标模型')
      return
    }

    setIsRunning(true)
    setResults([])
    setActivePreview(null)

    try {
      toast.info(`正在并发向所有支持 [${selectedModel}] 的渠道发起降智体检...`)
      
      const authHeaders = await getFreshAuthHeaders()
      const response = await fetch('/api/channel/matrix-eval', {
        method: 'POST',
        credentials: 'include',
        headers: {
          ...authHeaders,
          'Content-Type': 'application/json',
          'Accept': 'text/event-stream',
        },
        body: JSON.stringify({
          model: selectedModel,
          probe_type: selectedProbe,
          randomize: true,
        }),
      })

      if (!response.ok || !response.body) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let currentEvent = 'message'

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed || trimmed.startsWith(':')) continue // ignore ping / empty

          if (trimmed.startsWith('event: ')) {
            currentEvent = trimmed.slice(7).trim()
            continue
          }

          if (trimmed.startsWith('data: ')) {
            const dataStr = trimmed.slice(6).trim()
            try {
              const payload = JSON.parse(dataStr)
              if (currentEvent === 'init') {
                // Initial response
              } else if (currentEvent === 'result') {
                setResults((prev) => [...prev, payload as ChannelEvalResult])
              } else if (currentEvent === 'done') {
                toast.success('体检已全部完成！')
              }
            } catch (err) {
              console.error('Failed to parse SSE payload:', err, dataStr)
            }
          }
        }
      }
    } catch (err: any) {
      toast.error(err?.message || '评测请求异常中断')
    } finally {
      setIsRunning(false)
    }
  }

  const currentProbe = PROBE_OPTIONS.find((p) => p.id === selectedProbe)

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        <span className="flex items-center gap-2">
          <BrainCircuit className="h-5 w-5 text-primary" />
          <span>降智测试</span>
          <Badge variant="outline" className="text-xs font-normal">
            独立测试插件
          </Badge>
        </span>
      </SectionPageLayout.Title>

      <SectionPageLayout.Content>
        <div className="space-y-6 max-w-6xl pb-10">
          <p className="text-xs text-muted-foreground -mt-1">
            完全独立于渠道管理页面。跨所有提供相同模型的渠道进行直连靶向探针测试，秒破假模型偷换、思考截流与静态缓存作弊
          </p>

          {/* Control Card */}
          <div className="p-4 rounded-xl border bg-card/60 shadow-sm space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-12 gap-4 items-end">
              <div className="md:col-span-5 space-y-1.5">
                <label className="text-xs font-semibold text-foreground">目标测试模型</label>
                <Select
                  value={selectedModel}
                  onValueChange={setSelectedModel}
                  disabled={isRunning || models.length === 0}
                >
                  <SelectTrigger className="h-9 text-xs bg-background">
                    <SelectValue placeholder={models.length === 0 ? '正在加载模型列表...' : '选择模型'} />
                  </SelectTrigger>
                  <SelectContent className="max-h-64 text-xs">
                    {models.map((m) => (
                      <SelectItem key={m} value={m}>
                        {m}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="md:col-span-5 space-y-1.5">
                <label className="text-xs font-semibold text-foreground">选择体检探针 (Probe)</label>
                <Select
                  value={selectedProbe}
                  onValueChange={(v) => setSelectedProbe(v as ProbeKind)}
                  disabled={isRunning}
                >
                  <SelectTrigger className="h-9 text-xs bg-background">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="text-xs">
                    {PROBE_OPTIONS.map((p) => (
                      <SelectItem key={p.id} value={p.id}>
                        <span className="font-medium">{p.label}</span>
                        <span className="ml-2 text-muted-foreground text-[10px]">({p.tag})</span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="md:col-span-2">
                <Button
                  onClick={handleRunMatrixEval}
                  disabled={isRunning || !selectedModel}
                  className="h-9 gap-1.5 text-xs font-semibold w-full"
                >
                  {isRunning ? (
                    <>
                      <RotateCcw className="h-3.5 w-3.5 animate-spin" />
                      并发测试中...
                    </>
                  ) : (
                    <>
                      <Play className="h-3.5 w-3.5 fill-current" />
                      一键排查全部渠道
                    </>
                  )}
                </Button>
              </div>
            </div>

            {/* Current Probe Description Banner */}
            {currentProbe && (
              <div className="px-3.5 py-2.5 rounded-lg bg-muted/50 border text-xs text-muted-foreground flex items-center justify-between">
                <div>
                  <span className="font-semibold text-foreground mr-2">探针说明:</span>
                  {currentProbe.desc}
                </div>
                <Badge variant="secondary" className="text-[11px] shrink-0">
                  {currentProbe.tag}
                </Badge>
              </div>
            )}
          </div>

          {/* Results Matrix Table */}
          <div className="space-y-4">
            {results.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-64 text-muted-foreground border border-dashed rounded-xl bg-card/30">
                <BrainCircuit className="h-12 w-12 opacity-25 mb-3 stroke-[1.5]" />
                <p className="text-sm font-medium">尚未发起评测</p>
                <p className="text-xs opacity-75 mt-1">
                  在上方选择要核查的模型和探针，点击「一键排查全部渠道」查看横向对比
                </p>
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-3">
                {results.map((res, idx) => {
                  const isSVG = res.probe_type === 'pelican_svg' && res.analysis?.svg_code
                  const is3D =
                    (res.probe_type === 'pelican_3d' ||
                      res.probe_type === 'cf_ship_3d' ||
                      res.probe_type === 'qq_speed_3d') &&
                    res.analysis?.html_code

                  return (
                    <div
                      key={`${res.channel_id}-${idx}`}
                      className={`p-4 rounded-xl border transition-all ${
                        res.success
                          ? 'border-emerald-500/40 bg-emerald-500/[0.02]'
                          : 'border-destructive/30 bg-destructive/[0.02]'
                      }`}
                    >
                      {/* Channel Header */}
                      <div className="flex items-center justify-between pb-3 border-b border-border/50">
                        <div className="flex items-center gap-2.5">
                          <span className="font-mono text-muted-foreground text-xs">
                            #{res.channel_id}
                          </span>
                          <span className="font-bold text-foreground text-base">
                            {res.channel_name}
                          </span>
                          <Badge variant="outline" className="text-[10px] py-0 h-4.5">
                            渠道类型: {res.channel_type}
                          </Badge>
                        </div>

                        <div className="flex items-center gap-4">
                          <span className="text-muted-foreground text-xs flex items-center gap-1">
                            <Clock className="h-3.5 w-3.5" />
                            {res.duration_ms} ms
                          </span>

                          {res.success ? (
                            <div className="flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400 font-bold text-xs bg-emerald-500/10 px-2.5 py-1 rounded-full">
                              <CheckCircle2 className="h-4 w-4" />
                              通过 ({res.analysis?.score ?? 100}分)
                            </div>
                          ) : (
                            <div className="flex items-center gap-1.5 text-destructive font-bold text-xs bg-destructive/10 px-2.5 py-1 rounded-full">
                              <XCircle className="h-4 w-4" />
                              未通过 ({res.analysis?.score ?? 0}分)
                            </div>
                          )}
                        </div>
                      </div>

                      {/* Metrics Bar */}
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 py-2.5 text-xs">
                        <div>
                          <span className="text-muted-foreground">思维链 (Reasoning): </span>
                          <span
                            className={`font-mono font-bold ${
                              res.reasoning_tokens >= 1500
                                ? 'text-primary'
                                : res.reasoning_tokens > 0
                                ? 'text-amber-500'
                                : 'text-muted-foreground'
                            }`}
                          >
                            {res.reasoning_tokens} tokens
                          </span>
                        </div>
                        <div>
                          <span className="text-muted-foreground">消耗: </span>
                          <span className="font-mono">{res.total_tokens} tokens</span>
                        </div>
                        <div className="col-span-2 flex items-center justify-between">
                          <span className="text-muted-foreground truncate mr-2">
                            评测结论: <span className="text-foreground font-medium">{res.analysis?.summary}</span>
                          </span>

                          {/* Visual Actions */}
                          <div className="flex gap-2 shrink-0">
                            {isSVG && (
                              <Button
                                size="sm"
                                variant="outline"
                                className="h-7 text-xs px-2.5 gap-1 border-primary/40 text-primary"
                                onClick={() =>
                                  setActivePreview({
                                    title: `${res.channel_name} - 鹈鹕矢量图 (SVG)`,
                                    type: 'svg',
                                    content: res.analysis.svg_code!,
                                  })
                                }
                              >
                                <Sparkles className="h-3.5 w-3.5" />
                                查看画作
                              </Button>
                            )}

                            {is3D && (
                              <Button
                                size="sm"
                                variant="outline"
                                className="h-7 text-xs px-2.5 gap-1 border-emerald-500/50 text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/10"
                                onClick={() =>
                                  setActivePreview({
                                    title: `${res.channel_name} - 3D 页面沙箱运行`,
                                    type: 'html',
                                    content: res.analysis.html_code!,
                                  })
                                }
                              >
                                <ExternalLink className="h-3.5 w-3.5" />
                                3D沙箱试玩
                              </Button>
                            )}
                          </div>
                        </div>
                      </div>

                      {/* Thinking Output Details */}
                      {res.reasoning_output && (
                        <div className="mt-2 pt-2 border-t border-dashed border-border/60">
                          <details className="text-xs text-muted-foreground cursor-pointer">
                            <summary className="hover:text-foreground font-medium flex items-center gap-1 select-none">
                              <Code2 className="h-3.5 w-3.5" />
                              展开思维链推导 ({res.reasoning_tokens} tokens)
                            </summary>
                            <div className="mt-1.5 p-3 rounded-md bg-muted/60 font-mono text-[11px] whitespace-pre-wrap max-h-48 overflow-auto leading-relaxed">
                              {res.reasoning_output}
                            </div>
                          </details>
                        </div>
                      )}

                      {/* Error display */}
                      {res.error_message && (
                        <div className="mt-2 p-2 rounded bg-destructive/10 text-destructive text-xs">
                          错误: {res.error_message}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      </SectionPageLayout.Content>

      {/* Visual Sandbox Modal */}
      {activePreview && (
        <Dialog open={true} onOpenChange={() => setActivePreview(null)}>
          <DialogContent className="max-w-4xl max-h-[85vh] flex flex-col p-5">
            <DialogHeader className="pb-3 border-b">
              <DialogTitle className="text-base font-bold">
                {activePreview.title}
              </DialogTitle>
            </DialogHeader>

            <div className="flex-1 w-full min-h-[480px] flex items-center justify-center bg-black/5 dark:bg-black/40 rounded-lg overflow-hidden border">
              {activePreview.type === 'svg' ? (
                <div
                  className="max-w-full max-h-[460px] p-4 flex items-center justify-center overflow-auto"
                  dangerouslySetInnerHTML={{ __html: activePreview.content }}
                />
              ) : (
                <iframe
                  title="3D Sandbox"
                  srcDoc={activePreview.content}
                  className="w-full h-[520px] border-none"
                  sandbox="allow-scripts allow-same-origin"
                />
              )}
            </div>
          </DialogContent>
        </Dialog>
      )}
    </SectionPageLayout>
  )
}
export default IntelligenceBench
