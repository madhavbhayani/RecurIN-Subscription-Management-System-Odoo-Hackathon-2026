import { useEffect, useState } from 'react'
import { FiEdit2, FiTrash2 } from 'react-icons/fi'
import { Link } from 'react-router-dom'
import Pagination from '../../../components/common/Pagination'
import ToastMessage from '../../../components/common/ToastMessage'
import { deleteSubscription, listSubscriptions } from '../../../services/subscriptionApi'
import { buildPaginationState, createInitialPagination } from '../../../utils/pagination'

function getStatusMeta(status) {
	const normalizedStatus = String(status ?? '').trim().toLowerCase()

	if (normalizedStatus === 'draft') {
		return {
			label: 'Draft',
			className: 'border-slate-300 bg-slate-100 text-slate-700',
		}
	}

	if (normalizedStatus === 'quotation sent') {
		return {
			label: 'Quotation Sent',
			className: 'border-amber-200 bg-amber-50 text-amber-700',
		}
	}

	if (normalizedStatus === 'active') {
		return {
			label: 'Active',
			className: 'border-emerald-200 bg-emerald-50 text-emerald-700',
		}
	}

	if (normalizedStatus === 'confirmed') {
		return {
			label: 'Confirmed',
			className: 'border-sky-200 bg-sky-50 text-sky-700',
		}
	}

	return {
		label: String(status ?? '-'),
		className: 'border-[color:rgba(0,0,128,0.2)] bg-[rgba(0,0,128,0.04)] text-[var(--navy)]',
	}
}

function SubscriptionPage() {
	const [searchInput, setSearchInput] = useState('')
	const [searchTerm, setSearchTerm] = useState('')
	const [subscriptions, setSubscriptions] = useState([])
	const [currentPage, setCurrentPage] = useState(1)
	const [pagination, setPagination] = useState(() => createInitialPagination())
	const [refreshKey, setRefreshKey] = useState(0)
	const [isLoading, setIsLoading] = useState(false)
	const [toastMessage, setToastMessage] = useState('')
	const [toastVariant, setToastVariant] = useState('error')
	const [subscriptionPendingDelete, setSubscriptionPendingDelete] = useState(null)
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

		const fetchSubscriptions = async () => {
			setIsLoading(true)
			try {
				const response = await listSubscriptions(searchTerm, currentPage)
				if (!isMounted) {
					return
				}

				setSubscriptions(Array.isArray(response?.subscriptions) ? response.subscriptions : [])
				const paginationState = buildPaginationState(response?.pagination, currentPage)
				setPagination(paginationState)

				if (paginationState.total_pages > 0 && currentPage > paginationState.total_pages) {
					setCurrentPage(paginationState.total_pages)
				}
			} catch (error) {
				if (!isMounted) {
					return
				}

				setSubscriptions([])
				setPagination(createInitialPagination(currentPage))
				setToastVariant('error')
				setToastMessage(error.message)
			} finally {
				if (isMounted) {
					setIsLoading(false)
				}
			}
		}

		fetchSubscriptions()

		return () => {
			isMounted = false
		}
	}, [searchTerm, currentPage, refreshKey])

	const handleOpenDeleteDialog = (subscription) => {
		setSubscriptionPendingDelete(subscription)
	}

	const handleCloseDeleteDialog = () => {
		if (isDeleting) {
			return
		}

		setSubscriptionPendingDelete(null)
	}

	const handleConfirmDelete = async () => {
		const subscriptionID = String(subscriptionPendingDelete?.subscription_id ?? '').trim()
		if (!subscriptionID) {
			setSubscriptionPendingDelete(null)
			return
		}

		setIsDeleting(true)
		try {
			const response = await deleteSubscription(subscriptionID)
			setToastVariant('success')
			setToastMessage(response?.message ?? 'Subscription deleted successfully.')
			setSubscriptionPendingDelete(null)
			setRefreshKey((previousKey) => previousKey + 1)
		} catch (error) {
			setToastVariant('error')
			setToastMessage(error.message)
		} finally {
			setIsDeleting(false)
		}
	}

	return (
		<div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-8">
			<ToastMessage message={toastMessage} variant={toastVariant} onClose={() => setToastMessage('')} />

			<div className="flex items-center justify-between gap-4">
				<h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Subscriptions</h1>
				<Link
					to="/admin/subscriptions/new"
					className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000]"
				>
					New
				</Link>
			</div>

			<p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
				Confirmed subscriptions include payment reference, payment date, and expiry date based on billing period.
			</p>

			<div className="mt-5">
				<input
					type="search"
					value={searchInput}
					onChange={(event) => setSearchInput(event.target.value)}
					placeholder="Search by customer name, recurring, status or payment id"
					className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 py-2.5 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
				/>
			</div>

			<div className="mt-6 overflow-x-auto rounded-xl border border-[color:rgba(0,0,128,0.12)]">
				<div className="min-w-[1120px]">
					<div className="grid grid-cols-[1.2fr_1fr_1fr_1.4fr_1fr_136px] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
						<span>Customer Name</span>
						<span>Expiry Date</span>
						<span>Recurring</span>
						<span>Payment</span>
						<span>Status</span>
						<span className="text-right">Action</span>
					</div>

					{isLoading ? (
						<div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading subscriptions...</div>
					) : subscriptions.length === 0 ? (
						<div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
							{searchTerm
								? 'No subscriptions match your search.'
								: 'No subscriptions found yet. Click New to create one.'}
						</div>
					) : (
						<div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
							{subscriptions.map((subscription) => {
								const statusMeta = getStatusMeta(subscription.status)
								const payment = subscription.payment ?? null

								return (
									<div
										key={subscription.subscription_id}
										className="grid grid-cols-[1.2fr_1fr_1fr_1.4fr_1fr_136px] gap-4 px-4 py-4 text-sm text-[var(--navy)]"
									>
										<div>{subscription.customer_name}</div>
										<div>{subscription.next_invoice_date}</div>
										<div>{subscription.recurring}</div>
										<div>
											{payment ? (
												<div className="space-y-1 text-xs">
													<p className="font-semibold text-[var(--navy)]">{payment.invoice_number || '-'}</p>
													<p className="text-[color:rgba(0,0,128,0.72)]">Date: {String(payment.payment_date || '-').slice(0, 10)}</p>
													<p className="text-[color:rgba(0,0,128,0.72)]">${Number(payment.payment_amount || 0).toFixed(2)}</p>
												</div>
											) : (
												<span className="text-xs text-[color:rgba(0,0,128,0.62)]">No payment data</span>
											)}
										</div>
										<div>
											<span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${statusMeta.className}`}>
												{statusMeta.label}
											</span>
										</div>
										<div className="flex items-center justify-end gap-2">
											<Link
												to={`/admin/subscriptions/${subscription.subscription_id}`}
												className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)]"
												title="Edit"
												aria-label="Edit subscription"
											>
												<FiEdit2 className="h-4 w-4" />
											</Link>

											<button
												type="button"
												onClick={() => handleOpenDeleteDialog(subscription)}
												className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-300 text-red-600 transition-colors duration-200 hover:bg-red-50"
												title="Delete"
												aria-label="Delete subscription"
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

			{subscriptionPendingDelete && (
				<div className="fixed inset-0 z-40 flex items-center justify-center bg-black/35 p-4">
					<div className="w-full max-w-md rounded-xl border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] p-6 shadow-[0_16px_40px_rgba(0,0,0,0.2)]">
						<h2 className="[font-family:var(--font-display)] text-xl font-bold text-[var(--navy)]">Delete Subscription</h2>
						<p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">
							Are you sure you want to delete subscription{' '}
							<span className="font-semibold text-[var(--navy)]">{subscriptionPendingDelete.subscription_number}</span>?
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

export default SubscriptionPage
