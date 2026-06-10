import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Upload(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Upload</p>
        <h1>Upload an estimate and prepare it for review.</h1>
        <p>Use this page to start the estimate checking workflow. Backend upload processing is the next product integration step.</p>
      </section>
      <section className="workspace twoColumns">
        <div className="uploadBox">
          <div className="uploadIcon">+</div>
          <h2>Drop your file here</h2>
          <p>Supported workflow target: Excel estimate files and report-ready documents.</p>
          <input type="file" />
          <button className="btn">Start review</button>
        </div>
        <div className="card checklistCard">
          <h2>What will be checked</h2>
          <ul>
            <li>Missing item names and units</li>
            <li>Empty quantities and prices</li>
            <li>Duplicate rows</li>
            <li>Total mismatch and review notes</li>
          </ul>
          <a className="btn secondary" href="/dashboard">Open dashboard</a>
        </div>
      </section>
      <Footer/>
    </main>
  )
}
