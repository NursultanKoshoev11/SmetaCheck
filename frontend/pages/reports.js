import {useEffect, useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export default function Reports(){
  const [estimates, setEstimates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  async function loadReports(){
    setLoading(true);
    setError('');
    try{
      const response = await fetch(`${API_BASE}/v1/estimates`);
      const data = await response.json();
      if(!response.ok){
        throw new Error(data.error || 'Cannot load reports');
      }
      setEstimates(data.estimates || []);
    }catch(err){
      setError(err.message || 'Cannot load reports');
    }finally{
      setLoading(false);
    }
  }

  useEffect(()=>{ loadReports(); }, []);

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Reports</p>
        <h1>Clear reports for owners and construction teams.</h1>
        <p>Review uploaded estimate summaries, scores, findings, and downloadable report files from the API.</p>
      </section>
      <section className="workspace">
        <div className="buttonRow"><button className="btn secondary" type="button" onClick={loadReports}>Refresh</button><a className="btn" href="/upload">Upload new file</a></div>
        {loading && <div className="card"><p>Loading reports...</p></div>}
        {error && <div className="card"><h2>API error</h2><p>{error}</p><p>Check that the Go API is running on {API_BASE}.</p></div>}
        {!loading && !error && estimates.length === 0 && <div className="card"><h2>No reports yet</h2><p>Upload your first estimate to create a report.</p><a className="btn" href="/upload">Upload estimate</a></div>}
        {!loading && !error && estimates.length > 0 && <div className="reportTable">
          <div className="tableHead"><span>ID</span><span>Name</span><span>Score</span><span>Status</span><span>Report</span></div>
          {estimates.map((estimate) => <div className="tableRow" key={estimate.id}><span>{estimate.id}</span><b>{estimate.file_name}</b><strong>{estimate.score}</strong><em>{estimate.status}</em><a href={`${API_BASE}/v1/estimates/${estimate.id}/report`}>Download</a></div>)}
        </div>}
      </section>
      <Footer/>
    </main>
  )
}
