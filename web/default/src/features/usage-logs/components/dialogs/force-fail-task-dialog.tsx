import { useQueryClient } from '@tanstack/react-query'
import { AlertTriangle } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { formatLogQuota } from '@/lib/format'

import { TASK_STATUS } from '../../constants'
import { forceFailTask } from '../../task-api'
import type { TaskLog } from '../../types'

interface ForceFailTaskDialogProps {
  log: TaskLog
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function ForceFailTaskDialog({
  log,
  open,
  onOpenChange,
  onSuccess,
}: ForceFailTaskDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [reason, setReason] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && !isSubmitting) {
      setReason('')
      onOpenChange(false)
    } else if (nextOpen) {
      onOpenChange(true)
    }
  }

  const handleConfirm = async () => {
    setIsSubmitting(true)
    try {
      const response = await forceFailTask(log.task_id, reason.trim())
      if (!response.success) {
        toast.error(response.message || t('Failed to update task status'))
        return
      }
      toast.success(t('Task marked as failed'))
      onOpenChange(false)
      setReason('')
      await queryClient.invalidateQueries({ queryKey: ['logs'] })
      onSuccess?.()
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : t('Failed to update task status')
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  const isFinal =
    log.status === TASK_STATUS.SUCCESS || log.status === TASK_STATUS.FAILURE

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent size='sm'>
        <AlertDialogHeader>
          <AlertDialogMedia className='bg-destructive/10 text-destructive'>
            <AlertTriangle />
          </AlertDialogMedia>
          <AlertDialogTitle>{t('Mark task as failed')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              'This will end the task and refund its pre-charged quota. This action cannot be undone.'
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className='space-y-3 text-sm'>
          <div className='bg-muted/50 rounded-md border p-2 font-mono text-xs break-all'>
            {log.task_id}
          </div>
          <div className='text-muted-foreground flex items-center justify-between'>
            <span>{t('Current status')}</span>
            <span>{log.status}</span>
          </div>
          <div className='text-muted-foreground flex items-center justify-between'>
            <span>{t('Pre-charged quota')}</span>
            <span className='font-mono'>{formatLogQuota(log.quota || 0)}</span>
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='force-fail-reason'>{t('Failure reason')}</Label>
            <Textarea
              id='force-fail-reason'
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder={t('Enter the reason for manually failing this task')}
              maxLength={1000}
              disabled={isSubmitting || isFinal}
              rows={3}
            />
          </div>
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel disabled={isSubmitting}>
            {t('Cancel')}
          </AlertDialogCancel>
          <AlertDialogAction
            variant='destructive'
            onClick={handleConfirm}
            disabled={isSubmitting || isFinal}
          >
            {isSubmitting ? t('Processing...') : t('Confirm failure')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
