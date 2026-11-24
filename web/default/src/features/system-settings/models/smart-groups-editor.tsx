import { Plus, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

type SmartRoute = { values: string[]; group: string }
type SmartGroup = {
  description: string
  parameter: string
  routes: SmartRoute[]
  default_group: string
  user_selectable: boolean
  authorization_required: boolean
}
type SmartGroupDraft = SmartGroup & {
  id: string
  name: string
  routeText: string
}

type Props = {
  value: string
  groupNames: string[]
  onChange: (value: string) => void
  onInvalidChange: (invalid: boolean) => void
}

function routeText(routes: SmartRoute[]) {
  return routes
    .map((route) => `${route.values.join(',')} => ${route.group}`)
    .join('\n')
}

function parseRouteText(value: string): {
  routes: SmartRoute[]
  invalid: boolean
} {
  const routes: SmartRoute[] = []
  let invalid = false
  for (const line of value.split(/\r?\n/)) {
    const trimmed = line.trim()
    if (!trimmed) continue
    const separator = trimmed.indexOf('=>')
    if (separator < 1) {
      invalid = true
      continue
    }
    const values = trimmed
      .slice(0, separator)
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
    const group = trimmed.slice(separator + 2).trim()
    if (!values.length || !group) {
      invalid = true
      continue
    }
    routes.push({ values, group })
  }
  return { routes, invalid }
}

function parseGroups(value: string): SmartGroupDraft[] {
  try {
    const source = JSON.parse(value) as Record<string, SmartGroup>
    return Object.entries(source).map(([name, group]) => ({
      id: crypto.randomUUID(),
      name,
      description: group.description || '',
      parameter: group.parameter || '',
      routes: group.routes || [],
      routeText: routeText(group.routes || []),
      default_group: group.default_group || '',
      user_selectable: group.user_selectable === true,
      authorization_required: group.authorization_required === true,
    }))
  } catch {
    return []
  }
}

export function SmartGroupsEditor({
  value,
  groupNames,
  onChange,
  onInvalidChange,
}: Props) {
  const { t } = useTranslation()
  const [rows, setRows] = useState<SmartGroupDraft[]>(() => parseGroups(value))
  const [sourceValue, setSourceValue] = useState(value)

  useEffect(() => {
    if (value === sourceValue) return
    setRows(parseGroups(value))
    setSourceValue(value)
  }, [sourceValue, value])

  const duplicateNames = useMemo(() => {
    const names = rows.map((row) => row.name.trim()).filter(Boolean)
    return names.filter((name, index) => names.indexOf(name) !== index)
  }, [rows])

  const invalid = useMemo(
    () =>
      rows.some((row) => {
        const parsed = parseRouteText(row.routeText)
        const values = parsed.routes.flatMap((route) => route.values)
        return (
          !row.name.trim() ||
          row.name.trim() === 'auto' ||
          groupNames.includes(row.name.trim()) ||
          !row.parameter.trim() ||
          !groupNames.includes(row.default_group) ||
          parsed.invalid ||
          parsed.routes.some((route) => !groupNames.includes(route.group)) ||
          new Set(values).size !== values.length
        )
      }) || duplicateNames.length > 0,
    [duplicateNames.length, groupNames, rows]
  )

  useEffect(() => onInvalidChange(invalid), [invalid, onInvalidChange])

  const emit = useCallback(
    (nextRows: SmartGroupDraft[]) => {
      setRows(nextRows)
      const names = nextRows.map((row) => row.name.trim()).filter(Boolean)
      const duplicate = new Set(names).size !== names.length
      let hasInvalid = duplicate
      const config: Record<string, SmartGroup> = {}
      for (const row of nextRows) {
        const name = row.name.trim()
        const parsedRoutes = parseRouteText(row.routeText)
        if (
          !name ||
          name === 'auto' ||
          groupNames.includes(name) ||
          !row.parameter.trim() ||
          !groupNames.includes(row.default_group) ||
          parsedRoutes.invalid ||
          parsedRoutes.routes.some((route) => !groupNames.includes(route.group))
        ) {
          hasInvalid = true
        }
        const values = parsedRoutes.routes.flatMap((route) => route.values)
        if (new Set(values).size !== values.length) hasInvalid = true
        if (name) {
          config[name] = {
            description: row.description,
            parameter: row.parameter.trim(),
            routes: parsedRoutes.routes,
            default_group: row.default_group,
            user_selectable: row.user_selectable,
            authorization_required: row.authorization_required,
          }
        }
      }
      onInvalidChange(hasInvalid)
      if (!hasInvalid) {
        const serialized = JSON.stringify(config, null, 2)
        setSourceValue(serialized)
        onChange(serialized)
      }
    },
    [groupNames, onChange, onInvalidChange]
  )

  const addGroup = () => {
    let index = 1
    let name = `smart_group_${index}`
    while (rows.some((row) => row.name === name) || groupNames.includes(name)) {
      name = `smart_group_${++index}`
    }
    emit([
      ...rows,
      {
        id: crypto.randomUUID(),
        name,
        description: '',
        parameter: 'resolution',
        routes: [],
        routeText: '',
        default_group: groupNames[0] || '',
        user_selectable: true,
        authorization_required: false,
      },
    ])
  }

  const updateRow = (index: number, patch: Partial<SmartGroupDraft>) => {
    emit(
      rows.map((row, rowIndex) =>
        rowIndex === index ? { ...row, ...patch } : row
      )
    )
  }

  return (
    <Card className='rounded-md'>
      <CardHeader className='bg-muted/20 border-b'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <CardTitle>{t('Smart groups')}</CardTitle>
            <CardDescription>
              {t(
                'Route requests to pricing groups using a request parameter. Smart groups do not have a multiplier.'
              )}
            </CardDescription>
          </div>
          <Button
            size='sm'
            onClick={addGroup}
            disabled={groupNames.length === 0}
          >
            <Plus className='mr-2 h-4 w-4' />
            {t('Add smart group')}
          </Button>
        </div>
      </CardHeader>
      <CardContent className='space-y-3 pt-4'>
        {rows.length === 0 && (
          <p className='text-muted-foreground text-sm'>
            {t('No smart groups configured.')}
          </p>
        )}
        {groupNames.length === 0 && (
          <p className='text-destructive text-sm'>
            {t('Create a pricing group before configuring smart groups.')}
          </p>
        )}
        {rows.map((row, index) => {
          const badName =
            !row.name.trim() ||
            row.name.trim() === 'auto' ||
            duplicateNames.includes(row.name.trim()) ||
            groupNames.includes(row.name.trim())
          const badRoutes = parseRouteText(row.routeText).invalid
          return (
            <div key={row.id} className='space-y-3 rounded-md border p-3'>
              <div className='grid gap-3 md:grid-cols-3'>
                <label className='space-y-1 text-sm'>
                  <span>{t('Group name')}</span>
                  <Input
                    value={row.name}
                    aria-invalid={badName}
                    onChange={(event) =>
                      updateRow(index, { name: event.target.value })
                    }
                  />
                </label>
                <label className='space-y-1 text-sm'>
                  <span>{t('Description')}</span>
                  <Input
                    value={row.description}
                    onChange={(event) =>
                      updateRow(index, { description: event.target.value })
                    }
                  />
                </label>
                <label className='space-y-1 text-sm'>
                  <span>{t('Request parameter')}</span>
                  <Input
                    value={row.parameter}
                    placeholder='resolution'
                    onChange={(event) =>
                      updateRow(index, { parameter: event.target.value })
                    }
                  />
                </label>
              </div>
              <div className='grid gap-3 md:grid-cols-[minmax(0,1fr)_16rem_auto] md:items-end'>
                <label className='space-y-1 text-sm'>
                  <span>{t('Parameter values and pricing groups')}</span>
                  <Textarea
                    value={row.routeText}
                    aria-invalid={badRoutes}
                    placeholder={'480p => group_a\n720p,1080p,4k => group_b'}
                    rows={3}
                    onChange={(event) =>
                      updateRow(index, { routeText: event.target.value })
                    }
                  />
                </label>
                <label className='space-y-1 text-sm'>
                  <span>{t('Default pricing group')}</span>
                  <select
                    className='border-input bg-background h-9 w-full rounded-md border px-2 text-sm'
                    value={row.default_group}
                    onChange={(event) =>
                      updateRow(index, { default_group: event.target.value })
                    }
                  >
                    <option value=''>{t('Select a pricing group')}</option>
                    {groupNames.map((name) => (
                      <option key={name} value={name}>
                        {name}
                      </option>
                    ))}
                  </select>
                </label>
                <Button
                  variant='ghost'
                  size='icon'
                  aria-label={t('Delete')}
                  onClick={() =>
                    emit(rows.filter((_, rowIndex) => rowIndex !== index))
                  }
                >
                  <Trash2 className='h-4 w-4' />
                </Button>
              </div>
              <div className='flex flex-wrap gap-x-6 gap-y-2 text-sm'>
                <label className='flex items-center gap-2'>
                  <Checkbox
                    checked={row.user_selectable}
                    onCheckedChange={(checked) =>
                      updateRow(index, { user_selectable: checked === true })
                    }
                  />
                  {t('User selectable')}
                </label>
                <label className='flex items-center gap-2'>
                  <Checkbox
                    checked={row.authorization_required}
                    onCheckedChange={(checked) =>
                      updateRow(index, {
                        authorization_required: checked === true,
                      })
                    }
                  />
                  {t('Admin authorization required')}
                </label>
              </div>
            </div>
          )
        })}
        {(duplicateNames.length > 0 ||
          rows.some((row) => groupNames.includes(row.name.trim()))) && (
          <p className='text-destructive text-sm'>
            {t(
              'Smart group names must be unique and cannot match a pricing group.'
            )}
          </p>
        )}
        {rows.some((row) => parseRouteText(row.routeText).invalid) && (
          <p className='text-destructive text-sm'>
            {t(
              'Each routing rule must use the format: values => pricing group.'
            )}
          </p>
        )}
        {rows.some(
          (row) =>
            parseRouteText(row.routeText).routes.some(
              (route) => !groupNames.includes(route.group)
            ) ||
            (row.default_group && !groupNames.includes(row.default_group))
        ) && (
          <p className='text-destructive text-sm'>
            {t(
              'Every route and default must target an existing pricing group.'
            )}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
