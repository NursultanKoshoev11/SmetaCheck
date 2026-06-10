import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Support(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Support</p>
        <h1>Get help with estimate review workflows.</h1>
        <p>Request product help, report an issue, or ask for onboarding support for your construction team.</p>
      </section>
      <section className="workspace grid features">
        <article className="card feature"><span>01</span><h3>Email support</h3><p>Send a request with project context and screenshots.</p><a href="mailto:support@smetacheck.kg">support@smetacheck.kg</a></article>
        <article className="card feature"><span>02</span><h3>Telegram support</h3><p>Use Telegram for quick operational questions and file workflow help.</p><a href="/dashboard">Open workspace</a></article>
        <article className="card feature"><span>03</span><h3>Bug report</h3><p>Describe what happened, what you expected, and which browser you used.</p><a href="/reports">Attach report context</a></article>
      </section>
      <Footer/>
    </main>
  )
}
