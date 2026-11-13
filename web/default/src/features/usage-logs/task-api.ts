import { api } from '@/lib/api'

export async function forceFailTask(taskId: string, reason: string) {
  const res = await api.post(
    `/api/task/${encodeURIComponent(taskId)}/force-fail`,
    { reason }
  )
  return res.data as {
    success: boolean
    message?: string
    data?: { task_id: string; status: string; refunded_quota: number }
  }
}
