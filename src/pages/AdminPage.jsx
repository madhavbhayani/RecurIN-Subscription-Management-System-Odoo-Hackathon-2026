function AdminPage() {
  return (
    <section className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-8">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)] sm:text-4xl">
          Admin Panel
        </h1>
        <p className="mt-3 text-sm text-[color:rgba(0,0,128,0.76)]">
          This area is available for Admin and Internal roles.
        </p>
      </div>
    </section>
  )
}

export default AdminPage
