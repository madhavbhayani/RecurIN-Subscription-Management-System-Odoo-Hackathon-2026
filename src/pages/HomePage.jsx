import { useEffect } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'

const problemPoints = [
  {
    title: 'Scattered Subscription Data',
    description: 'Teams lose time reconciling renewals and billing details across spreadsheets, chats, and email threads.',
  },
  {
    title: 'Manual Renewal Follow-Up',
    description: 'Missed reminders and delayed approvals lead to churn, payment delays, and inconsistent customer experiences.',
  },
  {
    title: 'Limited Revenue Visibility',
    description: 'Without a unified dashboard, decision makers struggle to track recurring trends and forecast growth.',
  },
]

const solutionSteps = [
  {
    title: '1. Centralize Contracts',
    description: 'Track customers, plans, products, and billing terms in one structured workspace.',
  },
  {
    title: '2. Automate Lifecycle',
    description: 'From quote to confirmation, keep renewal and invoice timelines aligned and predictable.',
  },
  {
    title: '3. Optimize Performance',
    description: 'Use reporting signals to improve retention and recurring revenue outcomes.',
  },
]

const aboutHighlights = [
  'Designed for teams managing growing recurring customer bases.',
  'Role-aware admin operations for secure collaboration.',
  'Focused on reducing manual work while improving billing confidence.',
]

const aboutPillars = [
  {
    title: 'Operational Clarity',
    description: 'Unify customers, products, recurring plans, and quotations into one reliable operating view.',
  },
  {
    title: 'Execution Speed',
    description: 'Reduce turnaround time on subscription operations with reusable templates and role-focused workflows.',
  },
  {
    title: 'Growth Readiness',
    description: 'Improve visibility across recurring revenue decisions with cleaner process data and reporting alignment.',
  },
]

const aboutStats = [
  { value: '1 Platform', label: 'for plans, products, users, and roles' },
  { value: 'Role-Aware', label: 'workflows for secure admin operations' },
  { value: 'Lifecycle Focus', label: 'from quote to recurring renewals' },
]

function HomePage() {
  const location = useLocation()
  const navigate = useNavigate()

  useEffect(() => {
    const targetSectionID = location.pathname === '/about'
      ? 'about'
      : String(location.hash || '').replace('#', '').trim()

    if (!targetSectionID) {
      if (location.pathname === '/' || location.pathname === '/home') {
        window.scrollTo({ top: 0, behavior: 'auto' })
      }
      return
    }

    const scrollTimer = window.setTimeout(() => {
      const sectionElement = document.getElementById(targetSectionID)
      if (sectionElement) {
        sectionElement.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }
    }, 30)

    return () => {
      window.clearTimeout(scrollTimer)
    }
  }, [location.hash, location.pathname])

  const handleLearnMore = () => {
    navigate('/about')
  }

  return (
    <div className="w-full">
      <section
        id="home"
        className="relative min-h-[calc(100vh-78px)] overflow-hidden border-b border-[color:rgba(0,0,128,0.14)]"
      >
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_14%_16%,rgba(255,107,0,0.16),transparent_42%),radial-gradient(circle_at_82%_14%,rgba(19,136,8,0.13),transparent_38%),linear-gradient(180deg,#fff_0%,#fff8f3_100%)]" />

        <div className="relative flex min-h-[calc(100vh-78px)] flex-col items-center justify-center px-4 py-12 text-center sm:px-6 lg:px-8">
          <h1 className="text-5xl font-bold tracking-tight text-[var(--navy)] sm:text-6xl">
            RecurIN
          </h1>

          <p className="mt-2 text-xs font-semibold uppercase tracking-[0.22em] text-[color:rgba(0,0,128,0.72)] sm:text-sm">
            Subscription &amp; Management
          </p>

          <p className="mt-8 max-w-3xl text-sm text-[color:rgba(0,0,128,0.82)] sm:text-base">
            RecurIN helps businesses centralize recurring plans, automate renewal workflows, and improve
            decision making with actionable visibility across the subscription lifecycle.
          </p>

          <div className="mt-7 flex flex-wrap items-center justify-center gap-3">
            <Link
              to="/signup"
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
            >
              Get Started
            </Link>

            <button
              type="button"
              onClick={handleLearnMore}
              className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-5 text-sm font-semibold text-[var(--navy)] transition-colors duration-300 hover:border-[var(--orange)] hover:text-[var(--orange)]"
            >
              Learn More
            </button>
          </div>
        </div>
      </section>

      <section className="px-4 py-8 sm:px-6 lg:px-8" aria-label="Problem areas">
        <div className="grid gap-4 md:grid-cols-3">
          {problemPoints.map((point) => (
            <article
              key={point.title}
              className="group rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-4 transition-all duration-300 hover:-translate-y-0.5 hover:border-[color:rgba(255,107,0,0.45)] hover:shadow-[0_10px_24px_rgba(0,0,128,0.08)]"
            >
              <h2 className="text-base font-bold text-[var(--navy)]">{point.title}</h2>
              <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">{point.description}</p>
            </article>
          ))}
        </div>
      </section>

      <section id="about" className="px-4 pb-10 sm:px-6 lg:px-8">
        <div className="rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-5 sm:p-6">
          <h2 className="text-xl font-bold text-[var(--navy)]">About</h2>

          <div className="mt-3 grid gap-4 md:grid-cols-[1.35fr_1fr]">
            <div className="rounded-lg border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.02)] p-4">
              <p className="text-sm text-[color:rgba(0,0,128,0.8)]">
                RecurIN is built to solve real subscription operations problems: fragmented data,
                manual renewals, and weak recurring visibility. The platform combines structure,
                automation-ready flows, and role-aware control into one practical workspace.
              </p>

              <div className="mt-4 grid gap-2">
                {aboutHighlights.map((highlight) => (
                  <p key={highlight} className="text-sm text-[var(--navy)]">• {highlight}</p>
                ))}
              </div>
            </div>

            <div className="grid gap-2.5">
              {aboutStats.map((stat) => (
                <article key={stat.value} className="rounded-lg border border-[color:rgba(0,0,128,0.12)] bg-[var(--white)] p-3.5">
                  <p className="text-sm font-bold text-[var(--navy)]">{stat.value}</p>
                  <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.72)]">{stat.label}</p>
                </article>
              ))}
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-3">
            {aboutPillars.map((pillar) => (
              <article key={pillar.title} className="rounded-lg border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.02)] p-3.5">
                <h3 className="text-sm font-bold text-[var(--navy)]">{pillar.title}</h3>
                <p className="mt-1.5 text-xs text-[color:rgba(0,0,128,0.76)]">{pillar.description}</p>
              </article>
            ))}
          </div>

          <div className="mt-4 rounded-lg border border-[color:rgba(255,107,0,0.25)] bg-[rgba(255,107,0,0.07)] p-4">
            <h3 className="text-sm font-bold text-[var(--navy)]">How RecurIN works</h3>
            <div className="mt-2 grid gap-2 md:grid-cols-3">
              {solutionSteps.map((step) => (
                <div key={step.title} className="rounded-md border border-[color:rgba(0,0,128,0.1)] bg-[var(--white)] p-3">
                  <p className="text-xs font-bold text-[var(--navy)]">{step.title}</p>
                  <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.74)]">{step.description}</p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}

export default HomePage