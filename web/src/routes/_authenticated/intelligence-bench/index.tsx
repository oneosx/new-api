import { createFileRoute, redirect } from '@tanstack/react-router'
import { IntelligenceBench } from '@/features/intelligence-bench'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/intelligence-bench/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({ to: '/403' })
    }
  },
  component: IntelligenceBench,
})
