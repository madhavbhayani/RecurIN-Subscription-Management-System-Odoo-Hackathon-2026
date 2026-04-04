export const ROLE_PERMISSION_RESOURCES = [
  {
    key: 'subscriptions',
    label: 'Subscription',
    module: 'Module',
  },
  {
    key: 'products',
    label: 'Products',
    module: 'Module',
  },
  {
    key: 'reporting',
    label: 'Reporting',
    module: 'Module',
  },
  {
    key: 'users',
    label: 'Users',
    module: 'Module',
  },
  {
    key: 'roles',
    label: 'Roles',
    module: 'Module',
  },
  {
    key: 'configurations.attribute',
    label: 'Attribute',
    module: 'Configuration',
  },
  {
    key: 'configurations.recurring-plan',
    label: 'Recurring Plan',
    module: 'Configuration',
  },
  {
    key: 'configurations.quotation-template',
    label: 'Quotation Template',
    module: 'Configuration',
  },
  {
    key: 'configurations.payment-term',
    label: 'Payment Term',
    module: 'Configuration',
  },
  {
    key: 'configurations.discount',
    label: 'Discount',
    module: 'Configuration',
  },
  {
    key: 'configurations.taxes',
    label: 'Taxes',
    module: 'Configuration',
  },
]

export const PERMISSION_ACTIONS = [
  {
    key: 'can_create',
    label: 'Create',
  },
  {
    key: 'can_read',
    label: 'Read',
  },
  {
    key: 'can_update',
    label: 'Update',
  },
  {
    key: 'can_delete',
    label: 'Delete',
  },
]

export function buildDefaultPermissionMap() {
  return ROLE_PERMISSION_RESOURCES.reduce((state, resource) => {
    return {
      ...state,
      [resource.key]: {
        resource_key: resource.key,
        can_create: false,
        can_read: false,
        can_update: false,
        can_delete: false,
      },
    }
  }, {})
}

export function mapPermissionsToState(permissionList) {
  const defaultState = buildDefaultPermissionMap()

  if (!Array.isArray(permissionList)) {
    return defaultState
  }

  const nextState = { ...defaultState }
  permissionList.forEach((permissionItem) => {
    const resourceKey = String(permissionItem?.resource_key ?? '').trim().toLowerCase()
    if (!nextState[resourceKey]) {
      return
    }

    nextState[resourceKey] = {
      resource_key: resourceKey,
      can_create: Boolean(permissionItem?.can_create),
      can_read: Boolean(permissionItem?.can_read),
      can_update: Boolean(permissionItem?.can_update),
      can_delete: Boolean(permissionItem?.can_delete),
    }
  })

  return nextState
}

export function buildPermissionPayload(permissionState) {
  return ROLE_PERMISSION_RESOURCES.map((resource) => {
    const currentPermission = permissionState?.[resource.key] ?? {}

    return {
      resource_key: resource.key,
      can_create: Boolean(currentPermission.can_create),
      can_read: Boolean(currentPermission.can_read),
      can_update: Boolean(currentPermission.can_update),
      can_delete: Boolean(currentPermission.can_delete),
    }
  })
}

export function countPermissionToggles(permissionList) {
  if (!Array.isArray(permissionList)) {
    return 0
  }

  let enabledCount = 0
  permissionList.forEach((permissionItem) => {
    if (permissionItem?.can_create) {
      enabledCount += 1
    }
    if (permissionItem?.can_read) {
      enabledCount += 1
    }
    if (permissionItem?.can_update) {
      enabledCount += 1
    }
    if (permissionItem?.can_delete) {
      enabledCount += 1
    }
  })

  return enabledCount
}
