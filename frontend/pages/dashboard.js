import Nav from '../components/Nav';
import Footer from '../components/Footer';

const stats = [['24','Files reviewed'], ['6','Need attention'], ['82','Avg score'], ['3','Team members']];
const activity = ['Estimate A-104 uploaded', 'Report KG-22 generated', 'Version comparison prepared', 'Manager review requested'];

export default function Dashboard(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Dashboard</p>
        <h1>One clean workspace for every estimate review.</h1>
        <p>Track uploads, reports, risk levels, and team activity from a minimal control center.</p>
      </section>
      <section className="workspace">
        <div className="grid statsGrid">
          {stats.map(([value,label]) => <article className="statCard" key={label}><strong>{value}</strong><span>{label}</span></article>)}
        </div>
        <div className="twoColumns">
          <div className="card">
            <h2>Recent activity</h2>
            <div className="timeline">{activity.map((item,i) => <p key={item}><b>{String(i+1).padStart(2,'0')}</b><span>{item}</span></p>)}</div>
          </div>
          <div className="card">
            <h2>Next action</h2>
            <p>Upload a new estimate or open the reports page to review prepared outputs.</p>
            <div className="buttonRow"><a className="btn" href="/upload">Upload file</a><a className="btn secondary" href="/reports">View reports</a></div>
          </div>
        </div>
      </section>
      <Footer/>
    </main>
  )
}
