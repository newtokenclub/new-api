/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useQuery } from '@tanstack/react-query'
import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, ChevronLeft, ChevronRight } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TASK_STATUS } from '@/features/usage-logs/constants'
import {
  taskActionMapper,
  taskPlatformMapper,
  taskStatusMapper,
} from '@/features/usage-logs/lib/mappers'
import { api } from '@/lib/api'

const route = getRouteApi('/_authenticated/affiliate/tasks/$userId')
const PAGE_SIZE = 25

type AffiliateTask = {
  id: number
  platform: string
  task_id: string
  action: string
  model_name: string
  status: string
  submit_time: number
  start_time: number
  finish_time: number
  progress: string
  fail_reason: string
}

type AffiliateTasksResponse = {
  success: boolean
  message?: string
  data?: {
    items: AffiliateTask[]
    total: number
    page: number
    page_size: number
    user: { id: number; username: string; display_name: string }
  }
}

type TaskFilters = { task_id: string; model_name: string; status: string }

function formatTaskTime(timestamp: number) {
  return timestamp ? new Date(timestamp * 1000).toLocaleString() : '-'
}

export function AffiliateSubordinateTasks() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { userId: rawUserId } = route.useParams()
  const userId = Number(rawUserId)
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState<TaskFilters>({
    task_id: '',
    model_name: '',
    status: '',
  })
  const [draftFilters, setDraftFilters] = useState<TaskFilters>(filters)

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['affiliate-user-tasks', userId, page, filters],
    enabled: Number.isSafeInteger(userId) && userId > 0,
    queryFn: async () => {
      const res = await api.get<AffiliateTasksResponse>(
        `/api/user/affiliate/users/${userId}/tasks`,
        {
          params: {
            p: page,
            page_size: PAGE_SIZE,
            task_id: filters.task_id || undefined,
            model_name: filters.model_name || undefined,
            status: filters.status || undefined,
          },
        }
      )
      return res.data
    },
  })

  const items = data?.data?.items ?? []
  const total = data?.data?.total ?? 0
  const user = data?.data?.user
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const applyFilters = () => {
    setPage(1)
    setFilters(draftFilters)
  }

  const clearFilters = () => {
    const cleared = { task_id: '', model_name: '', status: '' }
    setDraftFilters(cleared)
    setFilters(cleared)
    setPage(1)
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Breadcrumb>
        <Button
          variant='ghost'
          size='sm'
          onClick={() => void navigate({ to: '/affiliate' })}
        >
          <ArrowLeft />
          {t('Back to direct subordinates')}
        </Button>
      </SectionPageLayout.Breadcrumb>
      <SectionPageLayout.Title>
        {user?.display_name || user?.username || t('Subordinate tasks')}
        {user && (
          <span className='text-muted-foreground ml-2 text-sm font-normal'>
            {user.display_name && `${user.username} · `}
            {t('User ID')}: {user.id}
          </span>
        )}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button variant='outline' size='sm' onClick={() => void refetch()}>
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-0 flex-col gap-3'>
          <form
            className='flex shrink-0 flex-wrap items-end gap-2'
            onSubmit={(event) => {
              event.preventDefault()
              applyFilters()
            }}
          >
            <label className='grid min-w-48 gap-1 text-xs'>
              <span className='text-muted-foreground'>{t('Task ID')}</span>
              <Input
                value={draftFilters.task_id}
                onChange={(event) =>
                  setDraftFilters((current) => ({
                    ...current,
                    task_id: event.target.value,
                  }))
                }
              />
            </label>
            <label className='grid min-w-48 gap-1 text-xs'>
              <span className='text-muted-foreground'>{t('Model')}</span>
              <Input
                value={draftFilters.model_name}
                onChange={(event) =>
                  setDraftFilters((current) => ({
                    ...current,
                    model_name: event.target.value,
                  }))
                }
              />
            </label>
            <label className='grid min-w-44 gap-1 text-xs'>
              <span className='text-muted-foreground'>{t('Status')}</span>
              <Select
                value={draftFilters.status || 'all'}
                onValueChange={(value) =>
                  setDraftFilters((current) => ({
                    ...current,
                    status: value === 'all' ? '' : value,
                  }))
                }
              >
                <SelectTrigger className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='all'>{t('All task statuses')}</SelectItem>
                  <SelectItem value={TASK_STATUS.SUCCESS}>
                    {t('Successful tasks')}
                  </SelectItem>
                  <SelectItem value={TASK_STATUS.FAILURE}>
                    {t('Failed tasks')}
                  </SelectItem>
                  <SelectItem value='RUNNING'>{t('Running tasks')}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <Button type='submit'>{t('Search')}</Button>
            <Button type='button' variant='outline' onClick={clearFilters}>
              {t('Clear filters')}
            </Button>
          </form>

          <div className='hidden min-h-0 flex-1 overflow-auto rounded-md border lg:block'>
            <Table>
              <TableHeader className='bg-muted/60 sticky top-0 z-10'>
                <TableRow>
                  <TableHead>{t('Submit Time')}</TableHead>
                  <TableHead>{t('Platform')}</TableHead>
                  <TableHead>{t('Task ID')}</TableHead>
                  <TableHead>{t('Action')}</TableHead>
                  <TableHead>{t('Model')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Progress')}</TableHead>
                  <TableHead>{t('Fail Reason')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading && (
                  <TableRow>
                    <TableCell colSpan={8} className='text-muted-foreground h-24 text-center'>
                      {t('Loading...')}
                    </TableCell>
                  </TableRow>
                )}
                {isError && !isLoading && (
                  <TableRow>
                    <TableCell colSpan={8} className='text-destructive h-24 text-center'>
                      {data?.message || t('Failed to load task logs')}
                    </TableCell>
                  </TableRow>
                )}
                {!isLoading && !isError && items.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={8} className='text-muted-foreground h-24 text-center'>
                      {t('No task logs')}
                    </TableCell>
                  </TableRow>
                )}
                {items.map((task) => (
                  <TableRow key={task.id}>
                    <TableCell className='min-w-40 font-mono text-xs'>
                      <div>{formatTaskTime(task.submit_time)}</div>
                      {task.finish_time > 0 && (
                        <div className='text-muted-foreground'>
                          {formatTaskTime(task.finish_time)}
                        </div>
                      )}
                    </TableCell>
                    <TableCell>
                      {t(taskPlatformMapper.getLabel(task.platform))}
                    </TableCell>
                    <TableCell className='max-w-56 truncate font-mono text-xs' title={task.task_id}>
                      {task.task_id}
                    </TableCell>
                    <TableCell>{t(taskActionMapper.getLabel(task.action))}</TableCell>
                    <TableCell className='max-w-48 truncate' title={task.model_name}>
                      {task.model_name || '-'}
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        label={t(taskStatusMapper.getLabel(task.status))}
                        variant={taskStatusMapper.getVariant(task.status)}
                        copyable={false}
                      />
                    </TableCell>
                    <TableCell>{task.progress || '-'}</TableCell>
                    <TableCell className='max-w-64 truncate' title={task.fail_reason}>
                      {task.fail_reason || '-'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <div className='min-h-0 flex-1 overflow-auto space-y-2 lg:hidden'>
            {isLoading && (
              <div className='text-muted-foreground rounded-md border p-6 text-center'>
                {t('Loading...')}
              </div>
            )}
            {isError && !isLoading && (
              <div className='text-destructive rounded-md border p-6 text-center'>
                {data?.message || t('Failed to load task logs')}
              </div>
            )}
            {!isLoading && !isError && items.length === 0 && (
              <div className='text-muted-foreground rounded-md border p-6 text-center'>
                {t('No task logs')}
              </div>
            )}
            {items.map((task) => (
              <article key={task.id} className='space-y-2 rounded-md border p-3'>
                <div className='flex items-start justify-between gap-3'>
                  <div className='min-w-0'>
                    <div className='font-mono text-xs break-all'>{task.task_id}</div>
                    <div className='text-muted-foreground mt-1 text-xs'>
                      {formatTaskTime(task.submit_time)}
                    </div>
                  </div>
                  <StatusBadge
                    label={t(taskStatusMapper.getLabel(task.status))}
                    variant={taskStatusMapper.getVariant(task.status)}
                    copyable={false}
                  />
                </div>
                <div className='text-sm break-words'>
                  {task.model_name || '-'}
                </div>
                <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs'>
                  <span>{t(taskPlatformMapper.getLabel(task.platform))}</span>
                  <span>{t(taskActionMapper.getLabel(task.action))}</span>
                  <span>{task.progress || '-'}</span>
                </div>
                {task.fail_reason && (
                  <div className='text-destructive text-xs break-words'>
                    {task.fail_reason}
                  </div>
                )}
              </article>
            ))}
          </div>

          <div className='flex shrink-0 items-center justify-between gap-3'>
            <span className='text-muted-foreground text-sm'>
              {t('Page {{page}} of {{total}}', { page, total: pageCount })}
              <span className='ml-2'>({total})</span>
            </span>
            <div className='flex gap-1'>
              <Button
                variant='outline'
                size='icon'
                aria-label={t('Previous')}
                disabled={page <= 1 || isLoading}
                onClick={() => setPage((current) => Math.max(1, current - 1))}
              >
                <ChevronLeft />
              </Button>
              <Button
                variant='outline'
                size='icon'
                aria-label={t('Next')}
                disabled={page >= pageCount || isLoading}
                onClick={() => setPage((current) => Math.min(pageCount, current + 1))}
              >
                <ChevronRight />
              </Button>
            </div>
          </div>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
