import {useEffect, useMemo, useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export default function Dashboard(){
  const [estimates, setEstimates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  async function loadDashboard(){
    setLoading(true);
    setError('');
    try{
      const response = await fetch(`${API_BASE}/v1/estimates`);
      const data = await response.json();
      if(!response.ok){
        throw new Error(data.error || 'Cannot load dashboard');
      }
      setEstimates(data.estimates || []);
    }catch(err){
      setError(err.message || 'Cannot load dashboard');
    }finally{
      setLoading(false);
    }
  }

  useEffect(()=>{ loadDashboard(); }, []);

  const stats = useMemo(()=>{
    const total = estimates.length;
    const ready = estimates.filter(item => item.status === 'ready').length;
    const avg = total ? Math.round(estimates.reduce((sum,item)=>sum+(item.score || 0),0)/total) : 0;
    const findings = estimates.reduce((sum,item)=>sum+((item.findings || []).length),0);
    return [[String(total),'Files reviewed'], [String(ready),'Ready reports'], [String(avg),'Avg score'], [String(findings),'Findings']];
  }, [estimates]);

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Dashboard</p>
        <h1>One clean workspace for every estimate review.</h1>
        <p>Track uploaded files, report status, scores, and generated findings directly from the API.</p>
      </section>
      <section className="workspace">
        <div className="buttonRow"><button className="btn secondary" type="button" onClick={loadDashboard}>Refresh</button><a className="btn" href="/upload">Upload file</a></div>
        {loading && <div className="card"><p>Loading dashboard...</p></div>}
        {error && <div className="card"><h2>API error</h2><p>{error}</p><p>Check that the Go API is running on {API_BASE}.</p></div>}
        {!loading && !error && <div className="grid statsGrid">
          {stats.map(([value,label]) => <article className="statCard" key={label}><strong>{value}</strong><span>{label}</span></article>)}
        </div>}
        {!loading && !error && <div className="twoColumns">
          <div className="card">
            <h2>Recent reports</h2>
            <div className="timeline">{estimates.slice(0,4).map((item,i) => <p key={item.id}><b>{String(i+1).padStart(2,'0')}</b><span>{item.file_name} · score {item.score}</span></p>)}</div>
            {estimates.length === 0 && <p>No uploads yet. Start by uploading the first estimate.</p>}
          </div>
          <div className="card">
            <h2>Next action</h2>
            <p>Upload an estimate, then open reports to download the generated review file.</p>
            <div className="buttonRow"><a className="btn" href="/upload">Upload file</a><a className="btn secondary" href="/reports">View reports</a></div>
          </div>
        </div>}
      </section>
      <Footer/>
    </main>
  )
}
