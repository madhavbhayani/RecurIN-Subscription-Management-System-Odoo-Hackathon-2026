import { useEffect, useState } from 'react'
import { FiEdit2, FiTrash2 } from 'react-icons/fi'
import { Link, useNavigate } from 'react-router-dom'
import Pagination from '../../../components/common/Pagination'
import ToastMessage from '../../../components/common/ToastMessage'
import { deleteRole, listRoles } from '../../../services/roleApi'
import { buildPaginationState, createInitialPagination } from '../../../utils/pagination'
import { PERMISSION_ACTIONS, ROLE_PERMISSION_RESOURCES, countPermissionToggles } from './permissions'

const TOTAL_PERMISSION_TOGGLES = ROLE_PERMISSION_RESOURCES.length * PERMISSION_ACTIONS.length

function buildAssignedUserLabel(role) {
  const userName = String(role?.user_name ?? '').trim()
  const userEmail = String(role?.user_email ?? '').trim()

  if (userName && userEmail) {
    return `${userName} (${userEmail})`
  }

  if (userName) {
    return userName
  }

  if (userEmail) {
    return userEmail
  }

  return 'Unassigned'
}

function RolesPage() {
  const navigate = useNavigate()

  const [searchInput, setSearchInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [roles, setRoles] = useState([])
  const [currentPage, setCurrentPage] = useState(1)
  const [pagination, setPagination] = useState(() => createInitialPagination())
  const [refreshKey, setRefreshKey] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [rolePendingDelete, setRolePendingDelete] = useState(null)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    const debounceTimer = window.setTimeout(() => {
      setSearchTerm(searchInput.trim())
      setCurrentPage(1)
    }, 300)

    return () => {
      window.clearTimeout(debounceTimer)
    }
  }, [searchInput])

  useEffect(() => {
    let isMounted = true

    const fetchRoles = async () => {
      setIsLoading(true)
      try {
        const response = await listRoles(searchTerm, currentPage)
        if (!isMounted) {
          return
        }

        setRoles(Array.isArray(response?.roles) ? response.roles : [])
        const paginationState = buildPaginationState(response?.pagination, currentPage)
        setPagination(paginationState)

        if (paginationState.total_pages > 0 && currentPage > paginationState.total_pages) {
          setCurrentPage(paginationState.total_pages)
        }
      } catch (error) {
        if (!isMounted) {
          return
        }

        setRoles([])
        setPagination(createInitialPagination(currentPage))
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchRoles()

    return () => {
      isMounted = false
    }
  }, [searchTerm, currentPage, refreshKey])

  const handleOpenDeleteDialog = (role) => {
    if (role?.is_system) {
      setToastVariant('info')
      setToastMessage('System role cannot be deleted.')
      return
    }

    setRolePendingDelete(role)
  }

  const handleCloseDeleteDialog = () => {
    if (isDeleting) {
      return
    }

    setRolePendingDelete(null)
  }

  const handleConfirmDelete = async () => {
    const roleID = String(rolePendingDelete?.role_id ?? '').trim()
    if (!roleID) {
      setRolePendingDelete(null)
      return
    }

    setIsDeleting(true)
    try {
      const response = await deleteRole(roleID)
      setToastVariant('success')
      setToastMessage(response?.message ?? 'Role deleted successfully.')
      setRolePendingDelete(null)
      setRefreshKey((previousKey) => previousKey + 1)
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsDeleting(false)
    }
  }

  const handleEditRole = (role) => {
    const roleID = String(role?.role_id ?? '').trim()
    if (!roleID) {
      return
    }

    if (role?.is_system) {
      setToastVariant('info')
      setToastMessage('System role cannot be edited.')
      return
    }

    navigate(`/admin/roles/${roleID}`)
  }

  return (
    <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-8">
      <ToastMessage message={toastMessage} variant={toastVariant} onClose={() => setToastMessage('')} />

      <div className="flex items-center justify-between gap-4">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Roles</h1>
        <Link
          to="/admin/roles/new"
          className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000]"
        >
          New
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Manage role assignments and CRUD permissions for modules and configuration sections.
      </p>

      <div className="mt-5">
        <input
          type="search"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder="Search by role, assigned user, or email"
          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 py-2.5 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
        />
      </div>

      <div className="mt-6 overflow-x-auto rounded-xl border border-[color:rgba(0,0,128,0.12)]">
        <div className="min-w-[980px]">
          <div className="grid grid-cols-[1fr_1.6fr_1fr_120px_120px] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
            <span>Role Name</span>
            <span>Assigned User</span>
            <span>Permission Coverage</span>
            <span>Role Type</span>
            <span className="text-right">Action</span>
          </div>

          {isLoading ? (
            <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading roles...</div>
          ) : roles.length === 0 ? (
            <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
              {searchTerm
                ? 'No roles match your search.'
                : 'No roles found yet. Click New to create a role.'}
            </div>
          ) : (
            <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
              {roles.map((role) => {
                const enabledPermissions = countPermissionToggles(role?.permissions)
                const isSystemRole = Boolean(role?.is_system)

                return (
                  <div key={role.role_id} className="grid grid-cols-[1fr_1.6fr_1fr_120px_120px] gap-4 px-4 py-4 text-sm text-[var(--navy)]">
                    <div className="font-semibold">{role.role_name}</div>
                    <div>{buildAssignedUserLabel(role)}</div>
                    <div>{enabledPermissions} / {TOTAL_PERMISSION_TOGGLES}</div>
                    <div>
                      {isSystemRole ? (
                        <span className="inline-flex rounded-full bg-[rgba(0,0,128,0.08)] px-2.5 py-1 text-xs font-semibold text-[var(--navy)]">
                          System
                        </span>
                      ) : (
                        <span className="inline-flex rounded-full bg-[rgba(255,128,0,0.14)] px-2.5 py-1 text-xs font-semibold text-[var(--navy)]">
                          Custom
                        </span>
                      )}
                    </div>
                    <div className="flex items-center justify-end gap-2">
                      <button
                        type="button"
                        onClick={() => handleEditRole(role)}
                        disabled={isSystemRole}
                        className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)] disabled:cursor-not-allowed disabled:opacity-40"
                        title={isSystemRole ? 'System role cannot be edited' : 'Edit'}
                        aria-label="Edit role"
                      >
                        <FiEdit2 className="h-4 w-4" />
                      </button>

                      <button
                        type="button"
                        onClick={() => handleOpenDeleteDialog(role)}
                        disabled={isSystemRole}
                        className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-300 text-red-600 transition-colors duration-200 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-40"
                        title={isSystemRole ? 'System role cannot be deleted' : 'Delete'}
                        aria-label="Delete role"
                      >
                        <FiTrash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>

      <Pagination
        currentPage={currentPage}
        totalPages={pagination.total_pages}
        onPageChange={setCurrentPage}
        isDisabled={isLoading}
      />

      {rolePendingDelete && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/35 p-4">
          <div className="w-full max-w-md rounded-xl border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] p-6 shadow-[0_16px_40px_rgba(0,0,0,0.2)]">
            <h2 className="[font-family:var(--font-display)] text-xl font-bold text-[var(--navy)]">Delete Role</h2>
            <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">
              Are you sure you want to delete{' '}
              <span className="font-semibold text-[var(--navy)]">{rolePendingDelete.role_name}</span>
              ?
            </p>
            <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.62)]">
              The assigned user will be moved back to role User.
            </p>

            <div className="mt-6 flex items-center justify-end gap-3">
              <button
                type="button"
                onClick={handleCloseDeleteDialog}
                disabled={isDeleting}
                className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleConfirmDelete}
                disabled={isDeleting}
                className="inline-flex h-10 items-center rounded-lg bg-red-600 px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-70"
              >
                {isDeleting ? 'Deleting...' : 'Yes, Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default RolesPage
