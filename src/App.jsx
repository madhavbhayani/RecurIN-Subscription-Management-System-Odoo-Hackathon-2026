import Header from './components/layout/Header'
import Footer from './components/layout/Footer'
import TitleSection from './components/common/TitleSection'
import { mainSections } from './constants/site'

function App() {
  return (
    <div className="flex min-h-screen flex-col bg-[var(--light-bg)]">
      <Header />

      <main className="flex-1 py-6">
        {mainSections.map((section) => (
          <TitleSection key={section.id} id={section.id} title={section.title} />
        ))}
      </main>

      <Footer />
    </div>
  )
}

export default App
