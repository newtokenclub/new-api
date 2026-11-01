/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { useQuery } from '@tanstack/react-query'
import { ArrowUpRight, ClipboardCheck, RefreshCw } from 'lucide-react'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'

interface PendingApprovalsResponse {
  success: boolean
  data: {
    quota_applications: number
    settlement_applications: number
  }
}

export function PendingApprovals() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const query = useQuery({
    queryKey: ['dashboard-pending-approvals'],
    queryFn: async () =>
      (await api.get<PendingApprovalsResponse>('/api/dashboard/pending-approvals')).data.data,
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
    retry: false,
  })

  const items = [
    {
      label: t('Quota applications'),
      count: query.data?.quota_applications,
      path: '/quota-applications',
    },
    {
      label: t('Affiliate settlement applications'),
      count: query.data?.settlement_applications,
      path: '/affiliate',
    },
  ] as const
  const total = items.reduce((sum, item) => sum + (item.count ?? 0), 0)

  return (
    <div className='bg-muted/20 flex min-h-10 flex-wrap items-center gap-x-3 gap-y-1 rounded-md border px-2.5 py-1.5 sm:gap-x-4 sm:px-3'>
      <div className='text-muted-foreground flex items-center gap-1.5 text-xs font-medium'>
        <ClipboardCheck className='size-3.5' />
        <span>{t('Pending approvals')}</span>
        {query.isPending ? (
          <span className='text-muted-foreground/70'>—</span>
        ) : (
          <span className='text-foreground tabular-nums'>{total}</span>
        )}
      </div>
      <div className='flex min-w-0 flex-1 flex-wrap items-center gap-x-1 gap-y-1'>
        {items.map((item, index) => (
          <span key={item.path} className='inline-flex items-center'>
            {index > 0 && <span className='text-border mx-1.5'>·</span>}
            <Button
              variant='ghost'
              size='xs'
              className='text-muted-foreground h-6 gap-1 px-1.5'
              onClick={() => void navigate({ to: item.path })}
              disabled={query.isPending}
            >
              <span>{item.label}</span>
              <span
                className={cn(
                  'text-foreground tabular-nums',
                  (item.count ?? 0) > 0 && 'text-primary font-semibold'
                )}
              >
                {item.count ?? '—'}
              </span>
              {(item.count ?? 0) > 0 && <ArrowUpRight className='size-3' />}
            </Button>
          </span>
        ))}
        {!query.isPending && total === 0 && (
          <span className='text-muted-foreground hidden text-xs sm:inline'>
            {t('No pending approvals')}
          </span>
        )}
        {query.isError && (
          <span className='text-destructive text-xs'>{t('Refresh failed')}</span>
        )}
      </div>
      <Button
        variant='ghost'
        size='icon-xs'
        className='text-muted-foreground ml-auto'
        aria-label={t('Refresh')}
        title={t('Refresh')}
        onClick={() => void query.refetch()}
        disabled={query.isFetching}
      >
        <RefreshCw className={cn('size-3.5', query.isFetching && 'animate-spin')} />
      </Button>
    </div>
  )
}
