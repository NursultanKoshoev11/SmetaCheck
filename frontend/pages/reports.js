import Nav from '../components/Nav';
import Footer from '../components/Footer';

const reports = [
  ['KG-2026-001','Residential estimate','82','Ready'],
  ['KG-2026-002','Material purchase review','74','Review'],
  ['KG-2026-003','Foundation budget','91','Ready'],
];

export default function Reports(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Reports</p>
        <h1>Clear reports for owners and construction teams.</h1>
        <p>Review prepared estimate summaries, scores, categories, and export-ready documents.</p>
      </section>
      <section className="workspace">
        <div className="reportTable">
          <div className="tableHead"><span>ID</span><span>Name</span><span>Score</span><span>Status</span></div>
          {reports.map(([id,name,score,status]) => <div className="tableRow" key={id}><span>{id}</span><b>{name}</b><strong>{score}</strong><em>{status}</em></div>)}
        </div>
        <div className="twoColumns">
          <div className="card"><h2>Report structure</h2><p>Summary, issue list, severity, categories, and recommended next steps.</p></div>
          <div className="card"><h2>Export</h2><p>Designed for future PDF and Excel export after backend report generation is connected.</p></div>
        </div>
      </section>
      <Footer/>
    </main>
  )
}
