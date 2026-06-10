import {useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export default function Upload(){
  const [file, setFile] = useState(null);
  const [status, setStatus] = useState('idle');
  const [message, setMessage] = useState('');
  const [result, setResult] = useState(null);

  async function submitUpload(){
    if(!file){
      setMessage('Choose a file first.');
      return;
    }
    setStatus('uploading');
    setMessage('Uploading file...');
    setResult(null);

    const formData = new FormData();
    formData.append('file', file);

    try{
      const response = await fetch(`${API_BASE}/v1/estimates/upload`, {method:'POST', body:formData});
      const data = await response.json();
      if(!response.ok){
        throw new Error(data.error || 'Upload failed');
      }
      setResult(data);
      setStatus('done');
      setMessage('Estimate uploaded and report metadata created.');
    }catch(error){
      setStatus('error');
      setMessage(error.message || 'Upload failed');
    }
  }

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Upload</p>
        <h1>Upload an estimate and prepare it for review.</h1>
        <p>Select an Excel or PDF estimate file. The API stores it, creates review metadata, and prepares a downloadable text report.</p>
      </section>
      <section className="workspace twoColumns">
        <div className="uploadBox">
          <div className="uploadIcon">+</div>
          <h2>Drop your file here</h2>
          <p>Production API target: <b>{API_BASE}</b></p>
          <input type="file" onChange={(event)=>setFile(event.target.files?.[0] || null)} />
          <button className="btn" type="button" onClick={submitUpload} disabled={status==='uploading'}>{status==='uploading' ? 'Uploading...' : 'Start review'}</button>
          {message && <p className={`statusText ${status}`}>{message}</p>}
          {result && <div className="resultBox"><b>Created estimate</b><p>ID: {result.id}</p><p>Score: {result.score}</p><a className="btn secondary" href={`/reports`}>Open reports</a></div>}
        </div>
        <div className="card checklistCard">
          <h2>What currently works</h2>
          <ul>
            <li>File upload to Go API</li>
            <li>Persistent file storage in upload volume</li>
            <li>Estimate metadata creation</li>
            <li>Downloadable report file generation</li>
          </ul>
          <a className="btn secondary" href="/dashboard">Open dashboard</a>
        </div>
      </section>
      <Footer/>
    </main>
  )
}
