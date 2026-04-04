function Footer() {
  const currentYear = new Date().getFullYear()

  return (
    <footer className="border-t border-[color:rgba(0,0,128,0.14)] bg-[var(--white)]">
      <div className="flex w-full flex-col gap-1.5 px-2 py-6 sm:px-3 lg:px-4">
        <p className="text-base font-bold text-[var(--navy)]">RecurIN</p>
        <p className="text-xs text-[color:rgba(0,0,128,0.82)]">Subscription Management System</p>
        <p className="text-[11px] text-[color:rgba(0,0,128,0.7)]">© {currentYear} RecurIN. All rights reserved.</p>
      </div>
    </footer>
  )
}

export default Footer