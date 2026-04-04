import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import ToastMessage from '../../../components/common/ToastMessage'
import { getUserById, updateUser } from '../../../services/userApi'

function UserEditPage() {
  const { userId = '' } = useParams()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [phoneNumber, setPhoneNumber] = useState('')
  const [address, setAddress] = useState('')
  const [role, setRole] = useState('User')
  const [activeSubscriptions, setActiveSubscriptions] = useState([])
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    let isMounted = true

    const loadUser = async () => {
      setIsLoading(true)
      try {
        const response = await getUserById(userId)
        if (!isMounted) {
          return
        }

        const user = response?.user
        setName(String(user?.name ?? ''))
        setEmail(String(user?.email ?? ''))
        setPhoneNumber(String(user?.phone_number ?? ''))
        setAddress(String(user?.address ?? ''))

        const incomingRole = String(user?.role ?? 'User').trim()
        setRole(incomingRole || 'User')

        setActiveSubscriptions(Array.isArray(response?.active_subscriptions) ? response.active_subscriptions : [])
      } catch (error) {
        if (!isMounted) {
          return
        }

        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    loadUser()

    return () => {
      isMounted = false
    }
  }, [userId])

  const handleSubmit = async (event) => {
    event.preventDefault()

    setToastMessage('')

    const normalizedName = name.trim()
    const normalizedEmail = email.trim()
    const normalizedPhoneNumber = phoneNumber.trim()

    if (!normalizedName || !normalizedEmail || !normalizedPhoneNumber) {
      setToastVariant('error')
      setToastMessage('Name, email and phone number are required.')
      return
    }

    setIsSubmitting(true)
    try {
      const response = await updateUser(userId, {
        name: normalizedName,
        email: normalizedEmail,
        phone_number: normalizedPhoneNumber,
        address: address.trim(),
      })

      const updatedUser = response?.user
      setToastVariant('success')
      setToastMessage(response?.message ?? 'User updated successfully.')

      if (updatedUser) {
        setName(String(updatedUser?.name ?? normalizedName))
        setEmail(String(updatedUser?.email ?? normalizedEmail))
        setPhoneNumber(String(updatedUser?.phone_number ?? normalizedPhoneNumber))
        setAddress(String(updatedUser?.address ?? address.trim()))

        const updatedRole = String(updatedUser?.role ?? role).trim()
        setRole(updatedRole || role)
      }
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-8 sm:p-10">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Edit User</h1>
        <Link
          to="/admin/users"
          className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
        >
          Back to Users
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Update user information and review active subscriptions.
      </p>

      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      {isLoading ? (
        <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
          Loading user details...
        </div>
      ) : (
        <form className="mt-6 space-y-7" onSubmit={handleSubmit} noValidate>
          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <label htmlFor="user-name" className="block text-sm font-semibold text-[var(--navy)]">
                Name
              </label>
              <input
                id="user-name"
                name="user-name"
                type="text"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="Enter full name"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="user-role" className="block text-sm font-semibold text-[var(--navy)]">
                Role
              </label>
              <input
                id="user-role"
                name="user-role"
                type="text"
                value={role}
                readOnly
                disabled
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.16)] bg-[rgba(0,0,128,0.03)] px-4 py-3 text-sm font-semibold text-[color:rgba(0,0,128,0.75)] outline-none"
              />
              <p className="text-xs text-[color:rgba(0,0,128,0.62)]">
                Role is managed from the Roles module and cannot be changed here.
              </p>
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <label htmlFor="user-email" className="block text-sm font-semibold text-[var(--navy)]">
                Email
              </label>
              <input
                id="user-email"
                name="user-email"
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="name@example.com"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="user-phone" className="block text-sm font-semibold text-[var(--navy)]">
                Phone Number
              </label>
              <input
                id="user-phone"
                name="user-phone"
                type="text"
                value={phoneNumber}
                onChange={(event) => setPhoneNumber(event.target.value)}
                placeholder="+919876543210"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label htmlFor="user-address" className="block text-sm font-semibold text-[var(--navy)]">
              Address
            </label>
            <textarea
              id="user-address"
              name="user-address"
              rows={3}
              value={address}
              onChange={(event) => setAddress(event.target.value)}
              placeholder="Enter address"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="rounded-xl border border-[color:rgba(0,0,128,0.14)]">
            <div className="border-b border-[color:rgba(0,0,128,0.1)] bg-[rgba(0,0,128,0.04)] px-4 py-3">
              <h2 className="text-sm font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">Active Subscriptions</h2>
            </div>

            {activeSubscriptions.length === 0 ? (
              <div className="px-4 py-6 text-sm text-[color:rgba(0,0,128,0.66)]">
                No active subscriptions found for this user.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <div className="min-w-[760px]">
                  <div className="grid grid-cols-[1fr_1fr_1fr_1fr] gap-4 px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
                    <span>Subscription</span>
                    <span>Recurring</span>
                    <span>Next Invoice</span>
                    <span>Status</span>
                  </div>

                  <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
                    {activeSubscriptions.map((subscription) => (
                      <div key={subscription.subscription_id} className="grid grid-cols-[1fr_1fr_1fr_1fr] gap-4 px-4 py-3 text-sm text-[var(--navy)]">
                        <span className="font-semibold">{subscription.subscription_number}</span>
                        <span>{subscription.recurring}</span>
                        <span>{subscription.next_invoice_date}</span>
                        <span>{subscription.status}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>

          <div className="flex flex-wrap items-center justify-end gap-3">
            <Link
              to="/admin/users"
              className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
            >
              {isSubmitting ? 'Updating...' : 'Update User'}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}

export default UserEditPage
