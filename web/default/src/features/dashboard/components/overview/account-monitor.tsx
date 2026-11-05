/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeftRight,
  Check,
  CircleUserRound,
  Eye,
  EyeOff,
  Link2,
  Plus,
  RefreshCw,
  ShieldCheck,
  Trash2,
  WalletCards,
} from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { saveUserId } from '@/features/auth/lib/storage'
import { api, getSelf } from '@/lib/api'
import { formatQuota } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

const TOKEN_KEY = 'dashboard_account_monitor_token'

interface MonitorAccount {
  id: number
  username: string
  display_name?: string
  quota: number
}

async function fetchMonitorAccount(): Promise<MonitorAccount | null> {
  const response = await api.get('/api/user/self/account-monitor/balance')
  return response.data?.data ?? null
}

export function AccountMonitor() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const currentUser = useAuthStore((state) => state.auth.user)
  const setUser = useAuthStore((state) => state.auth.setUser)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [showPassword, setShowPassword] = useState(false)

  const linkedAccount = useQuery({
    queryKey: ['dashboard', 'account-monitor', 'balance', currentUser?.id],
    queryFn: fetchMonitorAccount,
    enabled: Boolean(currentUser?.id),
    refetchInterval: 60_000,
    staleTime: 0,
  })

  const addAccount = useMutation({
    mutationFn: async () => {
      const response = await api.post(
        '/api/user/self/account-monitor',
        { username, password, code },
        { skipBusinessError: true, skipErrorHandler: true }
      )
      if (!response.data?.success) {
        if (response.data?.data?.require_2fa) {
          throw new Error(t('Enter the verification code for this account'))
        }
        throw new Error(
          t(response.data?.message || 'Account verification failed')
        )
      }
      return response.data.data as { token: string; account: MonitorAccount }
    },
    onSuccess: (data) => {
      window.localStorage.setItem(TOKEN_KEY, data.token)
      setPassword('')
      setCode('')
      setDialogOpen(false)
      void queryClient.invalidateQueries({
        queryKey: ['dashboard', 'account-monitor'],
      })
      toast.success(t('Account linked'))
    },
    onError: (error: Error) => toast.error(t(error.message)),
  })

  const switchAccount = useMutation({
    mutationFn: async () => {
      const token = window.localStorage.getItem(TOKEN_KEY)
      if (!token) {
        throw new Error(
          t('Switch authorization is missing. Relink the account.')
        )
      }
      const response = await api.post(
        '/api/user/self/account-monitor/switch',
        { token },
        { skipBusinessError: true, skipErrorHandler: true }
      )
      if (!response.data?.success) {
        throw new Error(t(response.data?.message || 'Account switch failed'))
      }
      return response.data.data as MonitorAccount
    },
    onSuccess: async (account) => {
      saveUserId(account.id)
      const response = await getSelf()
      if (response?.success && response.data) setUser(response.data)
      await queryClient.clear()
      window.location.reload()
    },
    onError: (error: Error) => toast.error(t(error.message)),
  })

  const removeAccount = useMutation({
    mutationFn: async () => {
      await api.delete('/api/user/self/account-monitor')
      window.localStorage.removeItem(TOKEN_KEY)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['dashboard', 'account-monitor'],
      })
      toast.success(t('Linked account removed'))
    },
  })

  const isLinked = Boolean(linkedAccount.data)
  const primary = currentUser
  const secondary = linkedAccount.data
  const handleAdd = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    addAccount.mutate()
  }

  return (
    <section
      className='bg-card overflow-hidden rounded-xl border shadow-xs'
      aria-labelledby='account-monitor-title'
    >
      <div className='flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3 sm:px-5'>
        <div className='flex min-w-0 items-center gap-3'>
          <span className='bg-primary/10 text-primary flex size-9 shrink-0 items-center justify-center rounded-lg'>
            <WalletCards className='size-4' aria-hidden='true' />
          </span>
          <div className='min-w-0'>
            <h2 id='account-monitor-title' className='text-sm font-semibold'>
              {t('Account balance monitor')}
            </h2>
            <p className='text-muted-foreground text-xs'>
              {t('Check both balances and switch the active account')}
            </p>
          </div>
        </div>
        <div className='flex items-center gap-2'>
          {isLinked && (
            <Button
              variant='ghost'
              size='icon-sm'
              aria-label={t('Refresh balances')}
              title={t('Refresh balances')}
              onClick={() => void linkedAccount.refetch()}
              disabled={linkedAccount.isFetching}
            >
              <RefreshCw
                className={linkedAccount.isFetching ? 'animate-spin' : ''}
              />
            </Button>
          )}
          {isLinked ? (
            <Button
              variant='outline'
              size='sm'
              onClick={() => removeAccount.mutate()}
              disabled={removeAccount.isPending}
            >
              <Trash2 data-icon='inline-start' />
              {t('Unlink')}
            </Button>
          ) : (
            <Button
              size='sm'
              onClick={() => setDialogOpen(true)}
              disabled={linkedAccount.isLoading || linkedAccount.isError}
            >
              <Plus data-icon='inline-start' />
              {t('Add account')}
            </Button>
          )}
        </div>
      </div>

      {isLinked && secondary ? (
        <div className='grid divide-y md:grid-cols-2 md:divide-x md:divide-y-0'>
          {[primary, secondary].map((account) => {
            if (!account) return null
            const active = account.id === currentUser?.id
            return (
              <div
                key={account.id}
                className='flex min-w-0 flex-col gap-4 p-4 sm:p-5'
              >
                <div className='flex items-start justify-between gap-3'>
                  <div className='flex min-w-0 items-center gap-3'>
                    <span
                      className={`flex size-10 shrink-0 items-center justify-center rounded-full ${active ? 'bg-success/10 text-success' : 'bg-muted text-muted-foreground'}`}
                    >
                      <CircleUserRound className='size-5' aria-hidden='true' />
                    </span>
                    <div className='min-w-0'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <span className='truncate text-sm font-semibold'>
                          {account.display_name || account.username}
                        </span>
                        <span
                          className={`inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] font-medium ${active ? 'bg-success/10 text-success' : 'bg-muted text-muted-foreground'}`}
                        >
                          {active ? (
                            <Check className='size-3' />
                          ) : (
                            <Eye className='size-3' />
                          )}
                          {active
                            ? t('Current account')
                            : t('Read-only monitor')}
                        </span>
                      </div>
                      <div className='text-muted-foreground truncate text-xs'>
                        @{account.username}
                      </div>
                    </div>
                  </div>
                </div>
                <div className='flex flex-wrap items-end justify-between gap-3'>
                  <div>
                    <div className='text-muted-foreground mb-1 text-xs'>
                      {t('Balance')}
                    </div>
                    <div className='font-mono text-2xl font-semibold tabular-nums'>
                      {formatQuota(Number(account.quota) || 0)}
                    </div>
                  </div>
                  {!active && (
                    <Button
                      size='sm'
                      onClick={() => switchAccount.mutate()}
                      disabled={switchAccount.isPending}
                    >
                      <ArrowLeftRight data-icon='inline-start' />
                      {switchAccount.isPending
                        ? t('Switching...')
                        : t('Switch to this account')}
                    </Button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      ) : (
        <div className='flex flex-col items-center px-5 py-7 text-center'>
          <span className='bg-muted text-muted-foreground mb-3 flex size-10 items-center justify-center rounded-full'>
            <Link2 className='size-5' />
          </span>
          <p className='text-sm font-medium'>
            {linkedAccount.isLoading
              ? t('Loading account status...')
              : t('Monitor another account')}
          </p>
          <p className='text-muted-foreground mt-1 max-w-md text-xs leading-relaxed'>
            {t(
              'Link a second account to view its live balance here. Only the active account can use other pages.'
            )}
          </p>
          {linkedAccount.isError && (
            <Button
              variant='outline'
              size='sm'
              className='mt-3'
              onClick={() => void linkedAccount.refetch()}
            >
              {t('Retry')}
            </Button>
          )}
        </div>
      )}

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Add account to monitor')}</DialogTitle>
            <DialogDescription>
              {t(
                'The password is used once for verification and is not retained by this app.'
              )}
            </DialogDescription>
          </DialogHeader>
          <form className='grid gap-4' onSubmit={handleAdd}>
            <div className='grid gap-2'>
              <Label htmlFor='monitor-username'>{t('Username')}</Label>
              <Input
                id='monitor-username'
                autoComplete='username'
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                required
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='monitor-password'>{t('Password')}</Label>
              <div className='relative'>
                <Input
                  id='monitor-password'
                  type={showPassword ? 'text' : 'password'}
                  autoComplete='current-password'
                  className='pr-10'
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  required
                />
                <button
                  type='button'
                  className='text-muted-foreground hover:text-foreground absolute inset-y-0 right-2 flex items-center'
                  onClick={() => setShowPassword((visible) => !visible)}
                  aria-label={
                    showPassword ? t('Hide password') : t('Show password')
                  }
                >
                  {showPassword ? (
                    <EyeOff className='size-4' />
                  ) : (
                    <Eye className='size-4' />
                  )}
                </button>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='monitor-code'>
                {t('Two-factor code (if enabled)')}
              </Label>
              <Input
                id='monitor-code'
                inputMode='numeric'
                autoComplete='one-time-code'
                value={code}
                onChange={(event) => setCode(event.target.value)}
              />
            </div>
            <div className='bg-muted/60 text-muted-foreground flex items-start gap-2 rounded-lg p-3 text-xs leading-relaxed'>
              <ShieldCheck className='text-success mt-0.5 size-4 shrink-0' />
              {t(
                'Only the account name, display name, and balance are available in monitor mode.'
              )}
            </div>
            <div className='flex justify-end gap-2'>
              <Button
                type='button'
                variant='outline'
                onClick={() => setDialogOpen(false)}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={addAccount.isPending}>
                {addAccount.isPending
                  ? t('Verifying...')
                  : t('Verify and link')}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </section>
  )
}
