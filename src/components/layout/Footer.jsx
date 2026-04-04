function Footer() {
  const currentYear = new Date().getFullYear()

  return (
    <footer className="border-t border-[color:rgba(0,0,128,0.14)] bg-[var(--white)]">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-2 px-4 py-8 sm:px-6 lg:px-8">
        <p className="text-lg font-bold text-[var(--navy)]">RecurIN</p>
        <p className="text-sm text-[color:rgba(0,0,128,0.82)]">Subscription Management System</p>
        <p className="text-xs text-[color:rgba(0,0,128,0.7)]">© {currentYear} RecurIN. All rights reserved.</p>
      </div>
    </footer>
  )
}

export default Footer