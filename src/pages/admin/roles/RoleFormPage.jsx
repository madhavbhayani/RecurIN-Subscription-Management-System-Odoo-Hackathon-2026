import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import ToastMessage from '../../../components/common/ToastMessage'
import { createRole, getRoleById, updateRole } from '../../../services/roleApi'
import { listUsers } from '../../../services/userApi'
import {
  PERMISSION_ACTIONS,
  ROLE_PERMISSION_RESOURCES,
  buildDefaultPermissionMap,
  buildPermissionPayload,
  mapPermissionsToState,
} from './permissions'

function normalizeUserRole(role) {
  return String(role ?? '').trim().toLowerCase()
}

const ROLE_NAME_OPTIONS = ['User', 'Admin', 'Internal']

function normalizeRoleNameOption(roleName) {
  const normalizedRoleName = String(roleName ?? '').trim().toLowerCase()
  switch (normalizedRoleName) {
    case 'admin':
      return 'Admin'
    case 'internal':
    case 'internal-user':
    case 'internal_user':
      return 'Internal'
    default:
      return 'User'
  }
}

function buildUserOptionLabel(user) {
  const userName = String(user?.name ?? '').trim()
  const userEmail = String(user?.email ?? '').trim()

  if (userName && userEmail) {
    return `${userName} (${userEmail})`
  }

  return userEmail || userName || ''
}

function RoleFormPage({ mode = 'create' }) {
  const { roleId = '' } = useParams()
  const isEditMode = mode === 'edit'

  const [roleName, setRoleName] = useState('User')
  const [selectedUser, setSelectedUser] = useState(null)
  const [userSearchInput, setUserSearchInput] = useState('')
  const [userSearchTerm, setUserSearchTerm] = useState('')
  const [userSearchResults, setUserSearchResults] = useState([])
  const [isUserSearchLoading, setIsUserSearchLoading] = useState(false)
  const [isUserResultsOpen, setIsUserResultsOpen] = useState(false)
  const [permissionsState, setPermissionsState] = useState(() => buildDefaultPermissionMap())
  const [isRoleLoading, setIsRoleLoading] = useState(isEditMode)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isSystemRole, setIsSystemRole] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const userSearchCloseTimerRef = useRef(null)

  const clearUserSearchCloseTimer = () => {
    if (userSearchCloseTimerRef.current !== null) {
      window.clearTimeout(userSearchCloseTimerRef.current)
      userSearchCloseTimerRef.current = null
    }
  }

  useEffect(() => {
    return () => {
      clearUserSearchCloseTimer()
    }
  }, [])

  useEffect(() => {
    const debounceTimer = window.setTimeout(() => {
      setUserSearchTerm(userSearchInput.trim())
    }, 300)

    return () => {
      window.clearTimeout(debounceTimer)
    }
  }, [userSearchInput])

  useEffect(() => {
    if (!isEditMode) {
      setIsRoleLoading(false)
      setIsSystemRole(false)
      return
    }

    let isMounted = true

    const loadRole = async () => {
      setIsRoleLoading(true)
      try {
        const response = await getRoleById(roleId)
        if (!isMounted) {
          return
        }

        const role = response?.role ?? {}
        const normalizedRoleName = normalizeRoleNameOption(String(role?.role_name ?? '').trim())
        setRoleName(normalizedRoleName)

        const incomingUserID = String(role?.user_id ?? '').trim()
        const incomingUserName = String(role?.user_name ?? '').trim()
        const incomingUserEmail = String(role?.user_email ?? '').trim()

        if (incomingUserID) {
          const incomingSelectedUser = {
            id: incomingUserID,
            name: incomingUserName,
            email: incomingUserEmail,
            role: 'internal',
          }

          setSelectedUser(incomingSelectedUser)
          setUserSearchInput(buildUserOptionLabel(incomingSelectedUser))
        } else {
          setSelectedUser(null)
          setUserSearchInput('')
        }

        setPermissionsState(mapPermissionsToState(role?.permissions))
        setIsSystemRole(Boolean(role?.is_system))
      } catch (error) {
        if (!isMounted) {
          return
        }

        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsRoleLoading(false)
        }
      }
    }

    loadRole()

    return () => {
      isMounted = false
    }
  }, [isEditMode, roleId])

  useEffect(() => {
    let isMounted = true

    const searchUsers = async () => {
      setIsUserSearchLoading(true)
      try {
        const response = await listUsers(userSearchTerm)
        if (!isMounted) {
          return
        }

        const incomingUsers = Array.isArray(response?.users) ? response.users : []
        const assignableUsers = incomingUsers
          .filter((user) => normalizeUserRole(user?.role) !== 'admin')
          .map((user) => ({
            id: String(user?.id ?? '').trim(),
            name: String(user?.name ?? '').trim(),
            email: String(user?.email ?? '').trim(),
            role: String(user?.role ?? '').trim(),
          }))
          .filter((user) => user.id && user.email)

        if (selectedUser?.id && !assignableUsers.some((user) => user.id === selectedUser.id)) {
          assignableUsers.unshift(selectedUser)
        }

        setUserSearchResults(assignableUsers.slice(0, 15))
      } catch (error) {
        if (!isMounted) {
          return
        }

        setUserSearchResults(selectedUser ? [selectedUser] : [])
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsUserSearchLoading(false)
        }
      }
    }

    searchUsers()

    return () => {
      isMounted = false
    }
  }, [userSearchTerm])

  const selectedUserLabel = useMemo(() => {
    if (!selectedUser) {
      return 'No user selected'
    }

    return buildUserOptionLabel(selectedUser)
  }, [selectedUser])

  const handlePermissionToggle = (resourceKey, actionKey, checked) => {
    setPermissionsState((previousState) => {
      const currentResourcePermission = previousState[resourceKey] ?? {
        resource_key: resourceKey,
        can_create: false,
        can_read: false,
        can_update: false,
        can_delete: false,
      }

      return {
        ...previousState,
        [resourceKey]: {
          ...currentResourcePermission,
          [actionKey]: checked,
        },
      }
    })
  }

  const handleSelectUser = (user) => {
    setSelectedUser(user)
    setUserSearchInput(buildUserOptionLabel(user))
    setIsUserResultsOpen(false)
    setToastMessage('')
  }

  const handleUserSearchFocus = () => {
    clearUserSearchCloseTimer()
    setIsUserResultsOpen(true)
  }

  const handleUserSearchBlur = () => {
    clearUserSearchCloseTimer()
    userSearchCloseTimerRef.current = window.setTimeout(() => {
      setIsUserResultsOpen(false)
    }, 150)
  }

  const handleUserSearchInputChange = (event) => {
    const nextValue = event.target.value
    setUserSearchInput(nextValue)
    setIsUserResultsOpen(true)

    if (selectedUser && nextValue.trim() !== buildUserOptionLabel(selectedUser)) {
      setSelectedUser(null)
    }
  }

  const handleSubmit = async (event) => {
    event.preventDefault()

    if (isSystemRole) {
      setToastVariant('error')
      setToastMessage('System role cannot be modified.')
      return
    }

    const normalizedRoleName = roleName.trim()
    const selectedUserID = String(selectedUser?.id ?? '').trim()

    if (!normalizedRoleName) {
      setToastVariant('error')
      setToastMessage('Role name is required.')
      return
    }

    if (!selectedUserID) {
      setToastVariant('error')
      setToastMessage('Please select a user for this role.')
      return
    }

    const payload = {
      role_name: normalizedRoleName,
      user_id: selectedUserID,
      permissions: buildPermissionPayload(permissionsState),
    }

    setIsSubmitting(true)
    try {
      if (isEditMode) {
        const response = await updateRole(roleId, payload)
        const updatedRole = response?.role ?? {}

        setToastVariant('success')
        setToastMessage(response?.message ?? 'Role updated successfully.')
        setRoleName(normalizeRoleNameOption(String(updatedRole?.role_name ?? normalizedRoleName)))
        setPermissionsState(mapPermissionsToState(updatedRole?.permissions))
      } else {
        const response = await createRole(payload)
        const createdRole = response?.role ?? {}

        setToastVariant('success')
        setToastMessage(response?.message ?? 'Role created successfully.')
        setRoleName('User')
        setSelectedUser(null)
        setUserSearchInput('')
        setUserSearchTerm('')
        setIsUserResultsOpen(false)
        setPermissionsState(buildDefaultPermissionMap())

        const createdUserID = String(createdRole?.user_id ?? '').trim()
        const createdUserName = String(createdRole?.user_name ?? '').trim()
        const createdUserEmail = String(createdRole?.user_email ?? '').trim()

        if (createdUserID && createdUserEmail) {
          setUserSearchResults((previousUsers) => {
            return [
              {
                id: createdUserID,
                name: createdUserName,
                email: createdUserEmail,
                role: 'internal',
              },
              ...previousUsers.filter((user) => user.id !== createdUserID),
            ]
          })
        }
      }
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  const pageTitle = isEditMode ? 'Edit Role' : 'New Role'
  const helperText = isEditMode
    ? 'Update role assignment and CRUD permission grants.'
    : 'Assign an existing user and configure CRUD grants per module.'

  return (
    <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-8 sm:p-10">
      <ToastMessage message={toastMessage} variant={toastVariant} onClose={() => setToastMessage('')} />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">{pageTitle}</h1>
        <Link
          to="/admin/roles"
          className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
        >
          Back to Roles
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">{helperText}</p>

      {isRoleLoading ? (
        <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
          Loading role details...
        </div>
      ) : (
        <form className="mt-6 space-y-7" onSubmit={handleSubmit} noValidate>
          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <label htmlFor="role-name" className="block text-sm font-semibold text-[var(--navy)]">
                Role Name
              </label>
              <select
                id="role-name"
                name="role-name"
                value={roleName}
                onChange={(event) => setRoleName(event.target.value)}
                disabled={isSystemRole}
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                {ROLE_NAME_OPTIONS.map((roleOption) => (
                  <option key={roleOption} value={roleOption}>{roleOption}</option>
                ))}
              </select>
            </div>

            <div className="space-y-2 sm:col-span-1">
              <label htmlFor="user-search" className="block text-sm font-semibold text-[var(--navy)]">
                Search User by Name or Email
              </label>
              <div className="relative">
                <input
                  id="user-search"
                  name="user-search"
                  type="search"
                  value={userSearchInput}
                  onChange={handleUserSearchInputChange}
                  onFocus={handleUserSearchFocus}
                  onBlur={handleUserSearchBlur}
                  disabled={isSystemRole}
                  placeholder="Type user name or email"
                  className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)] disabled:cursor-not-allowed disabled:opacity-60"
                />

                {isUserResultsOpen && !isSystemRole && (
                  <div className="absolute z-20 mt-2 w-full overflow-hidden rounded-lg border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] shadow-[0_10px_28px_rgba(0,0,0,0.12)]">
                    {isUserSearchLoading ? (
                      <div className="px-4 py-3 text-sm text-[color:rgba(0,0,128,0.66)]">Searching users...</div>
                    ) : userSearchResults.length === 0 ? (
                      <div className="px-4 py-3 text-sm text-[color:rgba(0,0,128,0.66)]">No users found for this search.</div>
                    ) : (
                      <div className="max-h-64 overflow-y-auto divide-y divide-[color:rgba(0,0,128,0.08)]">
                        {userSearchResults.map((user) => {
                          const isSelected = selectedUser?.id === user.id

                          return (
                            <button
                              key={user.id}
                              type="button"
                              onMouseDown={(event) => event.preventDefault()}
                              onClick={() => handleSelectUser(user)}
                              className={`flex w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors duration-200 ${
                                isSelected
                                  ? 'bg-[rgba(255,128,0,0.12)]'
                                  : 'hover:bg-[rgba(0,0,128,0.03)]'
                              }`}
                            >
                              <div>
                                <p className="text-sm font-semibold text-[var(--navy)]">{user.name || user.email}</p>
                                <p className="text-xs text-[color:rgba(0,0,128,0.66)]">{user.email}</p>
                              </div>
                              <span className="text-xs font-semibold uppercase tracking-[0.08em] text-[color:rgba(0,0,128,0.66)]">
                                {user.role || 'User'}
                              </span>
                            </button>
                          )
                        })}
                      </div>
                    )}
                  </div>
                )}
              </div>

              <p className="text-xs text-[color:rgba(0,0,128,0.66)]">
                Selected User: <span className="font-semibold text-[var(--navy)]">{selectedUserLabel}</span>
              </p>
              <p className="text-xs text-[color:rgba(0,0,128,0.6)]">
                Selected user will be moved to role Internal after save.
              </p>
            </div>
          </div>

          <div className="overflow-x-auto rounded-xl border border-[color:rgba(0,0,128,0.14)]">
            <div className="min-w-[860px]">
              <div className="grid grid-cols-[0.8fr_1.7fr_repeat(4,0.75fr)] gap-3 border-b border-[color:rgba(0,0,128,0.1)] bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
                <span>Section</span>
                <span>Item</span>
                {PERMISSION_ACTIONS.map((action) => (
                  <span key={action.key} className="text-center">{action.label}</span>
                ))}
              </div>

              <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
                {ROLE_PERMISSION_RESOURCES.map((resource) => {
                  const permission = permissionsState[resource.key] ?? {
                    can_create: false,
                    can_read: false,
                    can_update: false,
                    can_delete: false,
                  }

                  return (
                    <div key={resource.key} className="grid grid-cols-[0.8fr_1.7fr_repeat(4,0.75fr)] items-center gap-3 px-4 py-3 text-sm text-[var(--navy)]">
                      <span className="text-xs font-semibold uppercase tracking-[0.08em] text-[color:rgba(0,0,128,0.64)]">{resource.module}</span>
                      <span className="font-semibold">{resource.label}</span>
                      {PERMISSION_ACTIONS.map((action) => (
                        <div key={action.key} className="flex justify-center">
                          <input
                            type="checkbox"
                            checked={Boolean(permission[action.key])}
                            onChange={(event) => handlePermissionToggle(resource.key, action.key, event.target.checked)}
                            disabled={isSystemRole}
                            className="h-4 w-4 rounded border-[color:rgba(0,0,128,0.35)] text-[var(--orange)] focus:ring-[var(--orange)] disabled:cursor-not-allowed disabled:opacity-60"
                          />
                        </div>
                      ))}
                    </div>
                  )
                })}
              </div>
            </div>
          </div>

          {isSystemRole && (
            <div className="rounded-lg border border-[color:rgba(0,0,128,0.18)] bg-[rgba(0,0,128,0.03)] px-4 py-3 text-sm text-[color:rgba(0,0,128,0.76)]">
              This is a system role and cannot be modified.
            </div>
          )}

          <div className="flex flex-wrap items-center justify-end gap-3">
            <Link
              to="/admin/roles"
              className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isSubmitting || isSystemRole}
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
            >
              {isSubmitting
                ? (isEditMode ? 'Updating...' : 'Creating...')
                : (isEditMode ? 'Update Role' : 'Create Role')}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}

export default RoleFormPage
