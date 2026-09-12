import { createFileRoute } from '@tanstack/react-router'
import { CpaNodes } from '@/features/cpa-nodes'

export const Route = createFileRoute('/_authenticated/cpa-nodes/')({
  component: CpaNodes,
})
