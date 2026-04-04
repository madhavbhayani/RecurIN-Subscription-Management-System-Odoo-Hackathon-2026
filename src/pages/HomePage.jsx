import TitleSection from '../components/common/TitleSection'
import { mainSections } from '../constants/site'

function HomePage() {
  return (
    <section className="py-6">
      {mainSections.map((section) => (
        <TitleSection key={section.id} id={section.id} title={section.title} />
      ))}
    </section>
  )
}

export default HomePage