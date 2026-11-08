import type { AuthUser } from '@/stores/auth-store'

import { ROLE } from './roles'

export type AdminPermissionMatrix = Record<string, Record<string, boolean>>
export type AdminCapabilities = AdminPermissionMatrix

export const ADMIN_PERMISSION_RESOURCES = {
  CHANNEL: 'channel',
  QUOTA_APPLICATION: 'quota_application',
  TASK_LOG: 'task_log',
} as const

export const ADMIN_PERMISSION_ACTIONS = {
  READ: 'read',
  OPERATE: 'operate',
  WRITE: 'write',
  SENSITIVE_WRITE: 'sensitive_write',
  SECRET_VIEW: 'secret_view',
  APPLY: 'apply',
  VIEW_ALL: 'view_all',
} as const

// The role whose baseline grants are used as defaults in the permission editor.
export const ADMIN_ROLE_KEY = 'admin'

// The permission catalog (resources, actions, labels and role baselines) is owned
// by the backend authz package and fetched from GET /api/authz/catalog. It is
// intentionally NOT duplicated here so the schema stays defined in one place.
// These types mirror the backend JSON shape.
export interface PermissionActionDef {
  action: string
  label_key: string
  description_key: string
}

export interface PermissionResourceDef {
  resource: string
  label_key: string
  actions: PermissionActionDef[]
}

export interface PermissionRoleDef {
  key: string
  name: string
  built_in: boolean
  superuser: boolean
  grants: AdminPermissionMatrix
}

export interface PermissionCatalog {
  resources: PermissionResourceDef[]
  roles: PermissionRoleDef[]
}

export const EMPTY_PERMISSION_CATALOG: PermissionCatalog = {
  resources: [],
  roles: [],
}

export function hasPermission(
  user: AuthUser | null | undefined,
  resource: string,
  action: string
): boolean {
  if (!user) return false
  if (user.role === ROLE.SUPER_ADMIN) return true
  return user.permissions?.admin_permissions?.[resource]?.[action] === true
}

/**
 * Returns true when an admin has no administrator-module capability beyond
 * quota applications. Task-log support is a console capability, so granting it
 * must not expose unrelated administrator navigation.
 */
export function isQuotaApplicationOnlyAdmin(
  user: AuthUser | null | undefined
): boolean {
  if (!user || user.role < ROLE.ADMIN || user.role === ROLE.SUPER_ADMIN) {
    return false
  }

  const permissions = user.permissions?.admin_permissions
  if (!permissions || !hasPermission(user, 'quota_application', 'apply')) {
    return false
  }

  return !Object.entries(permissions).some(([resource, actions]) =>
    Object.entries(actions).some(
      ([action, allowed]) =>
        allowed === true &&
        !(resource === 'quota_application' && action === 'apply') &&
        !(resource === 'task_log' && action === 'view_all')
    )
  )
}

// roleGrants returns the baseline grant matrix for the given role key.
export function roleGrants(
  catalog: PermissionCatalog,
  roleKey: string
): AdminPermissionMatrix {
  return catalog.roles.find((role) => role.key === roleKey)?.grants ?? {}
}

export function defaultNewAdminPermissions(
  catalog: PermissionCatalog
): AdminPermissionMatrix {
  const permissions: AdminPermissionMatrix = {}
  for (const resource of catalog.resources) {
    permissions[resource.resource] = Object.fromEntries(
      resource.actions.map((action) => [action.action, false])
    )
  }
  if (permissions[ADMIN_PERMISSION_RESOURCES.QUOTA_APPLICATION]) {
    permissions[ADMIN_PERMISSION_RESOURCES.QUOTA_APPLICATION][
      ADMIN_PERMISSION_ACTIONS.APPLY
    ] = true
  }
  return permissions
}

// normalizeAdminPermissions produces a full matrix for the catalog, filling any
// value missing from `value` with the admin role's baseline grant.
export function normalizeAdminPermissions(
  value: AdminPermissionMatrix | null | undefined,
  catalog: PermissionCatalog
): AdminPermissionMatrix {
  const baseline = roleGrants(catalog, ADMIN_ROLE_KEY)
  const normalized: AdminPermissionMatrix = {}
  for (const resource of catalog.resources) {
    const actions: Record<string, boolean> = {}
    for (const action of resource.actions) {
      actions[action.action] =
        value?.[resource.resource]?.[action.action] ??
        baseline[resource.resource]?.[action.action] ??
        false
    }
    normalized[resource.resource] = actions
  }
  return normalized
}
