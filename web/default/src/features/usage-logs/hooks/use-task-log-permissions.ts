import { useIsRoot } from '@/hooks/use-root'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { useAuthStore } from '@/stores/auth-store'

export function useTaskLogPermissions() {
  const isRoot = useIsRoot()
  const user = useAuthStore((state) => state.auth.user)
  const hasTechnicalSupport = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.TASK_LOG,
    ADMIN_PERMISSION_ACTIONS.VIEW_ALL
  )

  return {
    canViewAllTaskLogs: isRoot || hasTechnicalSupport,
    canViewTaskCostStats: !hasTechnicalSupport || isRoot,
  }
}
