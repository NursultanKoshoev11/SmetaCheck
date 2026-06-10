import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Login(){
  return (
    <main className="page">
      <Nav/>
      <section className="authShell">
        <div>
          <p className="eyebrow">Account</p>
          <h1>Sign in to your estimate review workspace.</h1>
          <p>Manage uploads, reports, comparison history, and team review activity from one place.</p>
        </div>
        <form className="authCard">
          <label>Email<input placeholder="name@company.com" type="email" /></label>
          <label>Password<input placeholder="Your password" type="password" /></label>
          <button className="btn" type="button">Sign in</button>
          <button className="btn secondary" type="button">Continue with Google</button>
          <p>New here? <a href="/pricing">Choose a plan</a></p>
        </form>
      </section>
      <Footer/>
    </main>
  )
}
