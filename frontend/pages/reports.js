import {useEffect, useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function statusLabel(status){
  if(status === 'review_required') return 'Нужна проверка';
  if(status === 'ready') return 'Готово';
  return status || 'Неизвестно';
}

export default function Reports(){
  const [estimates, setEstimates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [authorized, setAuthorized] = useState(false);
  const [ai, setAi] = useState({});
  const [aiLoading, setAiLoading] = useState('');
  const [downloadLoading, setDownloadLoading] = useState('');

  function token(){
    return window.localStorage.getItem('smetacheck_token');
  }

  function logoutExpired(){
    window.localStorage.removeItem('smetacheck_token');
    window.localStorage.removeItem('smetacheck_user_email');
    setAuthorized(false);
  }

  async function loadReports(){
    const jwt = token();
    if(!jwt){
      setAuthorized(false);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError('');
    try{
      const meResponse = await fetch(`${API_BASE}/v1/auth/me`, {headers:{Authorization:`Bearer ${jwt}`}});
      if(!meResponse.ok){
        logoutExpired();
        throw new Error('Сессия истекла. Войдите снова.');
      }
      setAuthorized(true);

      const response = await fetch(`${API_BASE}/v1/estimates`, {headers:{Authorization:`Bearer ${jwt}`}});
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить отчёты'); }
      setEstimates(data.estimates || []);
    }catch(err){
      setError(err.message || 'Не удалось загрузить отчёты');
    }finally{
      setLoading(false);
    }
  }

  async function loadAI(estimateId){
    const jwt = token();
    if(!jwt){ logoutExpired(); return; }
    setAiLoading(estimateId);
    try{
      const response = await fetch(`${API_BASE}/v1/ai/estimate-summary/${estimateId}`, {headers:{Authorization:`Bearer ${jwt}`}});
      const data = await response.json();
      if(response.status === 401){ logoutExpired(); throw new Error('Сессия истекла. Войдите снова.'); }
      if(!response.ok){ throw new Error(data.error || 'AI-анализ недоступен'); }
      setAi((prev)=>({...prev, [estimateId]: data}));
    }catch(error){
      setAi((prev)=>({...prev, [estimateId]: {error: error.message || 'AI-анализ недоступен'}}));
    }finally{
      setAiLoading('');
    }
  }

  async function downloadTXT(estimate){
    const jwt = token();
    if(!jwt){ logoutExpired(); return; }
    setDownloadLoading(estimate.id);
    try{
      const response = await fetch(`${API_BASE}/v1/estimates/${estimate.id}/report`, {headers:{Authorization:`Bearer ${jwt}`}});
      if(response.status === 401){ logoutExpired(); throw new Error('Сессия истекла. Войдите снова.'); }
      if(!response.ok){
        let message = 'Не удалось скачать отчёт';
        try{ const data = await response.json(); message = data.error || message; }catch{}
        throw new Error(message);
      }
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${estimate.id}_report.txt`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    }catch(err){
      setError(err.message || 'Не удалось скачать отчёт');
    }finally{
      setDownloadLoading('');
    }
  }

  useEffect(()=>{ loadReports(); }, []);

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact"><p className="eyebrow">Отчёты</p><h1>Понятные отчёты по каждой проверенной смете.</h1><p>Здесь показываются только сметы текущего пользователя из PostgreSQL.</p></section>
      <section className="workspace">
        {loading && <div className="card"><p>Проверяем сессию и загружаем отчёты...</p></div>}
        {!loading && !authorized && <div className="emptyState"><h2>Нужно войти в аккаунт</h2><p>{error || 'Отчёты доступны только зарегистрированным пользователям.'}</p><a className="btn" href="/login">Войти или зарегистрироваться</a></div>}
        {!loading && authorized && <>
          <div className="buttonRow"><button className="btn secondary" type="button" onClick={loadReports}>Обновить</button><a className="btn" href="/upload">Загрузить новую смету</a></div>
          {error && <div className="card"><h2>Ошибка</h2><p>{error}</p></div>}
          {!error && estimates.length === 0 && <div className="emptyState"><h2>Отчётов пока нет</h2><p>Загрузите первую Excel или CSV смету, чтобы получить реальную проверку.</p><a className="btn" href="/upload">Загрузить смету</a></div>}
          {!error && estimates.length > 0 && <div className="grid">
            {estimates.map((estimate) => <article className="card" key={estimate.id}>
              <p className="eyebrow">{statusLabel(estimate.status)}</p>
              <h2>{estimate.file_name}</h2>
              <p>Оценка: <b>{estimate.score}/100</b> · строк: <b>{estimate.items_count || 0}</b> · сумма: <b>{Number(estimate.total_amount || 0).toLocaleString('ru-RU')}</b></p>
              <ul>{(estimate.findings || []).slice(0,4).map((finding, index) => <li key={`${estimate.id}-${index}`}>{finding.title}: {finding.detail}</li>)}</ul>
              <div className="buttonRow"><a className="btn" href={`/reports/${estimate.id}`}>Открыть отчёт</a><button className="btn secondary" type="button" onClick={()=>downloadTXT(estimate)} disabled={downloadLoading===estimate.id}>{downloadLoading===estimate.id ? 'Скачиваем...' : 'Скачать TXT'}</button><button className="btn secondary" type="button" onClick={()=>loadAI(estimate.id)} disabled={aiLoading===estimate.id}>{aiLoading===estimate.id ? 'AI анализирует...' : 'AI-анализ'}</button></div>
              {ai[estimate.id] && <div className="resultBox">{ai[estimate.id].error ? <p>{ai[estimate.id].error}</p> : <><b>AI-вывод: риск {ai[estimate.id].risk_level}</b><p>{ai[estimate.id].executive_brief}</p><p>{ai[estimate.id].recommendation}</p></>}</div>}
            </article>)}
          </div>}
        </>}
      </section>
      <Footer/>
    </main>
  );
}
