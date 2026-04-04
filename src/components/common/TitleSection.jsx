function TitleSection({ id, title }) {
  return (
    <section id={id} className="mx-auto w-full max-w-6xl px-4 py-5 sm:px-6 lg:px-8">
      <div className="rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] px-6 py-12 text-center sm:text-left">
        <h2 className="text-3xl font-bold tracking-tight text-[var(--navy)] sm:text-4xl">{title}</h2>
      </div>
    </section>
  )
}

export default TitleSection