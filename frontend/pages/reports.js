import {useEffect, useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function statusLabel(status){
  if(status === 'review_required') return 'Нужна проверка';
  if(status === 'ready') return 'Готово';
  return status || 'Неизвестно';
}

function authHeaders(){
  if(typeof window === 'undefined') return {};
  const token = window.localStorage.getItem('smetacheck_token');
  return token ? {Authorization: `Bearer ${token}`} : {};
}

export default function Reports(){
  const [estimates, setEstimates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [ai, setAi] = useState({});
  const [aiLoading, setAiLoading] = useState('');

  async function loadReports(){
    setLoading(true);
    setError('');
    try{
      const response = await fetch(`${API_BASE}/v1/estimates`, {headers: authHeaders()});
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить отчёты'); }
      setEstimates(data.estimates || []);
    }catch(err){ setError(err.message || 'Не удалось загрузить отчёты'); }
    finally{ setLoading(false); }
  }

  async function loadAI(estimateId){
    setAiLoading(estimateId);
    try{
      const response = await fetch(`${API_BASE}/v1/ai/estimate-summary/${estimateId}`, {headers: authHeaders()});
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'AI-анализ недоступен'); }
      setAi((prev)=>({...prev, [estimateId]: data}));
    }catch(error){
      setAi((prev)=>({...prev, [estimateId]: {error: error.message || 'AI-анализ недоступен'}}));
    }finally{
      setAiLoading('');
    }
  }

  useEffect(()=>{ loadReports(); }, []);

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact"><p className="eyebrow">Отчёты</p><h1>Понятные отчёты по каждой проверенной смете.</h1><p>Откройте детальный отчёт, посмотрите AI-вывод и сохраните страницу как PDF.</p></section>
      <section className="workspace">
        <div className="buttonRow"><button className="btn secondary" type="button" onClick={loadReports}>Обновить</button><a className="btn" href="/upload">Загрузить новую смету</a></div>
        {loading && <div className="card"><p>Загружаем отчёты...</p></div>}
        {error && <div className="card"><h2>Ошибка API</h2><p>{error}</p><p>Проверьте, что Go API запущен на {API_BASE}.</p></div>}
        {!loading && !error && estimates.length === 0 && <div className="card"><h2>Отчётов пока нет</h2><p>Загрузите первую Excel или CSV смету, чтобы получить реальную проверку.</p><a className="btn" href="/upload">Загрузить смету</a></div>}
        {!loading && !error && estimates.length > 0 && <div className="grid">
          {estimates.map((estimate) => <article className="card" key={estimate.id}>
            <p className="eyebrow">{statusLabel(estimate.status)}</p>
            <h2>{estimate.file_name}</h2>
            <p>Оценка: <b>{estimate.score}/100</b> · строк: <b>{estimate.items_count || 0}</b> · сумма: <b>{Number(estimate.total_amount || 0).toLocaleString('ru-RU')}</b></p>
            <ul>{(estimate.findings || []).slice(0,4).map((finding, index) => <li key={`${estimate.id}-${index}`}>{finding.title}: {finding.detail}</li>)}</ul>
            <div className="buttonRow"><a className="btn" href={`/reports/${estimate.id}`}>Открыть отчёт</a><a className="btn secondary" href={`${API_BASE}/v1/estimates/${estimate.id}/report`}>TXT</a><button className="btn secondary" type="button" onClick={()=>loadAI(estimate.id)} disabled={aiLoading===estimate.id}>{aiLoading===estimate.id ? 'AI анализирует...' : 'AI-анализ'}</button></div>
            {ai[estimate.id] && <div className="resultBox">{ai[estimate.id].error ? <p>{ai[estimate.id].error}</p> : <><b>AI-вывод: риск {ai[estimate.id].risk_level}</b><p>{ai[estimate.id].executive_brief}</p><p>{ai[estimate.id].recommendation}</p></>}</div>}
          </article>)}
        </div>}
      </section>
      <Footer/>
    </main>
  )
}
