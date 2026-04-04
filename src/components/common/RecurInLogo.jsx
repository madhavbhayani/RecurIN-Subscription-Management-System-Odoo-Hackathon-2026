function RecurInLogo({ compact = false, className = '', taglineClassName = '' }) {
  return (
    <div className={`flex items-center gap-3 ${className}`.trim()}>
      <svg
        className={`${compact ? 'h-12 w-12' : 'h-12 w-12'} flex-none`}
        viewBox="0 0 52 52"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
      >
        <rect width="52" height="52" rx="14" fill="#FF6B00" />
        <path d="M26 12 C18 12 12 18 12 26 C12 34 18 40 26 40" stroke="#fff" strokeWidth="3.5" strokeLinecap="round" fill="none" />
        <path d="M26 40 C34 40 40 34 40 26 C40 18 34 12 26 12" stroke="#138808" strokeWidth="3.5" strokeLinecap="round" fill="none" />
        <polygon points="26,8 31,15 21,15" fill="#fff" />
        <circle cx="26" cy="26" r="4" fill="#fff" />
        <circle cx="26" cy="26" r="2" fill="#FF6B00" />
      </svg>

      <div className="flex flex-col leading-none">
        <span className={`${compact ? 'text-3xl' : 'text-[2rem]'} [font-family:var(--font-display)] font-extrabold tracking-tight`}>
          <span className="text-[var(--orange)]">Recur</span>
          <span className="text-[var(--green)]">IN</span>
        </span>
        <span
          className={`${compact ? 'mt-1 text-[0.58rem]' : 'mt-1 text-[0.62rem]'} font-semibold uppercase tracking-[0.18em] text-[var(--navy)] ${taglineClassName}`.trim()}
        >
          Subscription &amp; Management
        </span>
      </div>
    </div>
  )
}

export default RecurInLogo