import { useEffect, useMemo, useState } from 'react'
import {
  FiActivity,
  FiBarChart2,
  FiDownload,
  FiLayers,
  FiRefreshCw,
  FiShoppingBag,
  FiTrendingUp,
  FiUsers,
} from 'react-icons/fi'
import {
  downloadAdminModuleFrequencyReportPdf,
  downloadAdminSalesReportPdf,
  getAdminReportingDashboard,
} from '../../../services/adminReportingApi'

const REPORTING_MONTH_OPTIONS = [
  { label: 'Last 3 Months', value: 3 },
  { label: 'Last 6 Months', value: 6 },
  { label: 'Last 12 Months', value: 12 },
  { label: 'Last 24 Months', value: 24 },
]

function formatCompactNumber(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return '0'
  }

  return new Intl.NumberFormat('en-IN', {
    notation: numericValue >= 1000 ? 'compact' : 'standard',
    maximumFractionDigits: numericValue >= 1000 ? 1 : 0,
  }).format(numericValue)
}

function formatCurrencyINR(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return 'INR 0.00'
  }

  try {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: 'INR',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(numericValue)
  } catch {
    return `INR ${numericValue.toFixed(2)}`
  }
}

function AdminReportingPage() {
  const [selectedMonths, setSelectedMonths] = useState(6)
  const [refreshCounter, setRefreshCounter] = useState(0)
  const [dashboard, setDashboard] = useState(null)
  const [isLoading, setIsLoading] = useState(true)
  const [errorMessage, setErrorMessage] = useState('')
  const [infoMessage, setInfoMessage] = useState('')
  const [isDownloadingSales, setIsDownloadingSales] = useState(false)
  const [isDownloadingModuleFrequency, setIsDownloadingModuleFrequency] = useState(false)

  useEffect(() => {
    let isMounted = true

    const fetchDashboard = async () => {
      setIsLoading(true)
      setErrorMessage('')

      try {
        const response = await getAdminReportingDashboard(selectedMonths)
        if (!isMounted) {
          return
        }

        setDashboard(response?.dashboard ?? null)
      } catch (error) {
        if (!isMounted) {
          return
        }

        setDashboard(null)
        setErrorMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchDashboard()

    return () => {
      isMounted = false
    }
  }, [selectedMonths, refreshCounter])

  const summary = dashboard?.summary ?? null
  const revenueTrend = Array.isArray(dashboard?.revenue_trend) ? dashboard.revenue_trend : []
  const paymentStatuses = Array.isArray(dashboard?.payment_statuses) ? dashboard.payment_statuses : []
  const moduleStatistics = Array.isArray(dashboard?.module_statistics) ? dashboard.module_statistics : []
  const generatedAt = String(dashboard?.generated_at ?? '').trim()

  const revenuePeak = useMemo(() => {
    if (!revenueTrend.length) {
      return 1
    }

    return revenueTrend.reduce((maxValue, point) => {
      const revenueValue = Number(point?.revenue_inr)
      if (!Number.isFinite(revenueValue)) {
        return maxValue
      }

      return Math.max(maxValue, revenueValue)
    }, 1)
  }, [revenueTrend])

  const paymentTotal = useMemo(() => {
    return paymentStatuses.reduce((runningTotal, statusItem) => {
      const countValue = Number(statusItem?.count)
      return runningTotal + (Number.isFinite(countValue) ? countValue : 0)
    }, 0)
  }, [paymentStatuses])

  const summaryCards = useMemo(() => {
    if (!summary) {
      return []
    }

    return [
      {
        title: 'Users',
        value: formatCompactNumber(summary.total_users),
        helper: 'Registered users',
        icon: FiUsers,
      },
      {
        title: 'Products',
        value: formatCompactNumber(summary.total_products),
        helper: 'Products in catalog',
        icon: FiShoppingBag,
      },
      {
        title: 'Subscriptions',
        value: formatCompactNumber(summary.total_subscriptions),
        helper: 'Total subscriptions',
        icon: FiLayers,
      },
      {
        title: 'Payments',
        value: formatCompactNumber(summary.total_payments),
        helper: 'Captured payment records',
        icon: FiActivity,
      },
      {
        title: 'Revenue',
        value: formatCurrencyINR(summary.total_revenue_inr),
        helper: 'Active + confirmed total',
        icon: FiTrendingUp,
      },
    ]
  }, [summary])

  const handleDownloadSalesReport = async () => {
    setInfoMessage('')
    setIsDownloadingSales(true)
    try {
      await downloadAdminSalesReportPdf(selectedMonths)
      setInfoMessage('Sales report PDF downloaded successfully.')
    } catch (error) {
      setInfoMessage(error.message)
    } finally {
      setIsDownloadingSales(false)
    }
  }

  const handleDownloadModuleFrequencyReport = async () => {
    setInfoMessage('')
    setIsDownloadingModuleFrequency(true)
    try {
      await downloadAdminModuleFrequencyReportPdf(selectedMonths)
      setInfoMessage('Module frequency PDF downloaded successfully.')
    } catch (error) {
      setInfoMessage(error.message)
    } finally {
      setIsDownloadingModuleFrequency(false)
    }
  }

  return (
    <div className="space-y-5">
      <section className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_24px_rgba(0,0,128,0.08)] sm:p-8">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">
              Reporting Dashboard
            </h1>
            <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
              View revenue trends, payment status, and operational reporting insights.
            </p>
            {generatedAt && (
              <p className="mt-2 text-xs font-medium text-[color:rgba(0,0,128,0.62)]">
                Generated at: {generatedAt}
              </p>
            )}
          </div>

          <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:justify-end">
            <select
              value={selectedMonths}
              onChange={(event) => setSelectedMonths(Number(event.target.value))}
              className="h-10 rounded-lg border border-[color:rgba(0,0,128,0.2)] px-3 text-sm font-semibold text-[var(--navy)] outline-none focus:border-[var(--orange)]"
              aria-label="Reporting period"
            >
              {REPORTING_MONTH_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>

            <button
              type="button"
              onClick={handleDownloadSalesReport}
              disabled={isDownloadingSales}
              className="inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:border-[var(--orange)] hover:text-[var(--orange)] disabled:cursor-not-allowed disabled:opacity-70"
            >
              <FiDownload className="h-4 w-4" />
              {isDownloadingSales ? 'Preparing...' : 'Sales PDF'}
            </button>

            <button
              type="button"
              onClick={handleDownloadModuleFrequencyReport}
              disabled={isDownloadingModuleFrequency}
              className="inline-flex h-10 items-center justify-center gap-2 rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:border-[var(--orange)] hover:text-[var(--orange)] disabled:cursor-not-allowed disabled:opacity-70"
            >
              <FiDownload className="h-4 w-4" />
              {isDownloadingModuleFrequency ? 'Preparing...' : 'Module PDF'}
            </button>

            <button
              type="button"
              onClick={() => setRefreshCounter((previousCounter) => previousCounter + 1)}
              className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000]"
            >
              <FiRefreshCw className="h-4 w-4" />
              Refresh
            </button>
          </div>
        </div>

        {infoMessage && (
          <div className="mt-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
            {infoMessage}
          </div>
        )}

        {errorMessage && (
          <div className="mt-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {errorMessage}
          </div>
        )}

        {isLoading ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.12)] px-4 py-5 text-sm text-[color:rgba(0,0,128,0.66)]">
            Loading reporting dashboard...
          </div>
        ) : (
          <>
            <div className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
              {summaryCards.map((card) => (
                <article
                  key={card.title}
                  className="rounded-xl border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.03)] px-4 py-4"
                >
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-xs font-semibold uppercase tracking-[0.08em] text-[color:rgba(0,0,128,0.62)]">{card.title}</p>
                    <card.icon className="h-4 w-4 text-[var(--orange)]" />
                  </div>
                  <p className="mt-2 text-lg font-bold text-[var(--navy)]">{card.value}</p>
                  <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.65)]">{card.helper}</p>
                </article>
              ))}
            </div>

            <div className="mt-6 grid gap-4 xl:grid-cols-[1.5fr_1fr]">
              <section className="rounded-xl border border-[color:rgba(0,0,128,0.12)] p-4">
                <div className="flex items-center gap-2">
                  <FiTrendingUp className="h-4 w-4 text-[var(--orange)]" />
                  <h2 className="text-base font-bold text-[var(--navy)]">Revenue Trend</h2>
                </div>

                {revenueTrend.length === 0 ? (
                  <p className="mt-4 text-sm text-[color:rgba(0,0,128,0.66)]">No revenue records found in this window.</p>
                ) : (
                  <>
                    <div className="mt-4 flex h-[210px] items-end gap-2 rounded-lg border border-[color:rgba(0,0,128,0.08)] bg-[rgba(0,0,128,0.02)] px-3 py-3">
                      {revenueTrend.map((trendPoint) => {
                        const revenueValue = Number(trendPoint?.revenue_inr)
                        const normalizedRevenue = Number.isFinite(revenueValue) ? revenueValue : 0
                        const fillHeight = Math.max((normalizedRevenue / revenuePeak) * 100, normalizedRevenue > 0 ? 6 : 2)

                        return (
                          <div key={trendPoint.period_key} className="flex min-w-0 flex-1 flex-col items-center gap-1">
                            <div
                              className="w-full rounded-t bg-gradient-to-t from-[var(--orange)] to-[#ff9c56]"
                              style={{ height: `${fillHeight}%` }}
                              title={`${trendPoint.period_label}: ${formatCurrencyINR(normalizedRevenue)}`}
                            ></div>
                            <span className="truncate text-[10px] font-semibold text-[color:rgba(0,0,128,0.65)]">
                              {trendPoint.period_label}
                            </span>
                          </div>
                        )
                      })}
                    </div>

                    <div className="mt-3 overflow-x-auto rounded-lg border border-[color:rgba(0,0,128,0.08)]">
                      <table className="min-w-full text-sm">
                        <thead className="bg-[rgba(0,0,128,0.04)] text-[var(--navy)]">
                          <tr>
                            <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.08em]">Period</th>
                            <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-[0.08em]">Revenue</th>
                          </tr>
                        </thead>
                        <tbody>
                          {revenueTrend.map((trendPoint) => (
                            <tr key={`row-${trendPoint.period_key}`} className="border-t border-[color:rgba(0,0,128,0.08)] text-[var(--navy)]">
                              <td className="px-3 py-2">{trendPoint.period_label}</td>
                              <td className="px-3 py-2 text-right font-semibold">{formatCurrencyINR(trendPoint.revenue_inr)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </>
                )}
              </section>

              <section className="rounded-xl border border-[color:rgba(0,0,128,0.12)] p-4">
                <div className="flex items-center gap-2">
                  <FiBarChart2 className="h-4 w-4 text-[var(--orange)]" />
                  <h2 className="text-base font-bold text-[var(--navy)]">Payment Status</h2>
                </div>

                {paymentStatuses.length === 0 ? (
                  <p className="mt-4 text-sm text-[color:rgba(0,0,128,0.66)]">No payment statuses found.</p>
                ) : (
                  <div className="mt-4 space-y-2">
                    {paymentStatuses.map((statusItem) => {
                      const countValue = Number(statusItem?.count)
                      const normalizedCount = Number.isFinite(countValue) ? countValue : 0
                      const sharePercentage = paymentTotal > 0 ? Math.round((normalizedCount / paymentTotal) * 100) : 0

                      return (
                        <article
                          key={statusItem.status}
                          className="rounded-lg border border-[color:rgba(0,0,128,0.1)] bg-[rgba(0,0,128,0.02)] px-3 py-2"
                        >
                          <div className="flex items-center justify-between gap-3 text-sm text-[var(--navy)]">
                            <span className="font-semibold">{statusItem.status}</span>
                            <span className="font-semibold">{normalizedCount}</span>
                          </div>
                          <div className="mt-2 h-1.5 rounded-full bg-[rgba(0,0,128,0.1)]">
                            <div className="h-1.5 rounded-full bg-[var(--orange)]" style={{ width: `${sharePercentage}%` }}></div>
                          </div>
                        </article>
                      )
                    })}
                  </div>
                )}
              </section>
            </div>

            <section className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.12)] p-4">
              <h2 className="text-base font-bold text-[var(--navy)]">Module Registration Frequency</h2>
              <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.66)]">
                Frequency of registrations across all modules in the selected reporting window.
              </p>

              <div className="mt-4 overflow-x-auto rounded-lg border border-[color:rgba(0,0,128,0.08)]">
                {moduleStatistics.length === 0 ? (
                  <div className="px-4 py-5 text-sm text-[color:rgba(0,0,128,0.66)]">No module statistics available.</div>
                ) : (
                  <table className="min-w-[860px] text-sm">
                    <thead className="bg-[rgba(0,0,128,0.04)] text-[var(--navy)]">
                      <tr>
                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.08em]">Module</th>
                        <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-[0.08em]">Total</th>
                        {moduleStatistics[0]?.frequency?.map((frequencyPoint) => (
                          <th key={frequencyPoint.period_key} className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-[0.08em]">
                            {frequencyPoint.period_label}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {moduleStatistics.map((moduleStatistic) => (
                        <tr key={moduleStatistic.module_key} className="border-t border-[color:rgba(0,0,128,0.08)] text-[var(--navy)]">
                          <td className="px-3 py-2 font-semibold">{moduleStatistic.module_label}</td>
                          <td className="px-3 py-2 text-right font-semibold">{formatCompactNumber(moduleStatistic.total_registrations)}</td>
                          {moduleStatistic.frequency.map((frequencyPoint) => (
                            <td key={`${moduleStatistic.module_key}-${frequencyPoint.period_key}`} className="px-3 py-2 text-right">
                              {formatCompactNumber(frequencyPoint.registrations)}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </section>
          </>
        )}
      </section>
    </div>
  )
}

export default AdminReportingPage
