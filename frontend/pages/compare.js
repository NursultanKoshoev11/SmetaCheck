import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Compare(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Compare</p>
        <h1>Compare two estimate versions before approval.</h1>
        <p>See added rows, removed rows, changed totals, and review notes in one clean comparison view.</p>
      </section>
      <section className="workspace twoColumns">
        <div className="compareDrop"><span>01</span><h2>Base estimate</h2><p>Upload the original estimate version.</p><input type="file" /></div>
        <div className="compareDrop"><span>02</span><h2>New estimate</h2><p>Upload the updated estimate version.</p><input type="file" /></div>
      </section>
      <section className="workspace">
        <div className="card"><h2>Comparison output</h2><div className="grid"><p>Added items</p><p>Removed items</p><p>Changed totals</p></div><a className="btn" href="/reports">Prepare report</a></div>
      </section>
      <Footer/>
    </main>
  )
}
