import Nav from '../components/Nav';
import PricingCards from '../components/PricingCards';
import Footer from '../components/Footer';

export default function Pricing(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Pricing</p>
        <h1>Simple plans for estimate review teams.</h1>
        <p>Begin with a small workflow, then upgrade when reports and team collaboration become part of daily work.</p>
      </section>
      <PricingCards/>
      <section className="workspace">
        <div className="card"><h2>Need a company plan?</h2><p>Use the support page to request a team setup, private onboarding, or custom report workflow.</p><a className="btn" href="/support">Contact support</a></div>
      </section>
      <Footer/>
    </main>
  )
}
