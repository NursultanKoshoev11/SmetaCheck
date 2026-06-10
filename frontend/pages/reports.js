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

  async function loadReports(){
    setLoading(true);
    setError('');
    try{
      const response = await fetch(`${API_BASE}/v1/estimates`);
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить отчёты'); }
      setEstimates(data.estimates || []);
    }catch(err){ setError(err.message || 'Не удалось загрузить отчёты'); }
    finally{ setLoading(false); }
  }

  useEffect(()=>{ loadReports(); }, []);

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Отчёты</p>
        <h1>Понятные отчёты по каждой проверенной смете.</h1>
        <p>Теперь отчёт показывает количество строк, сумму, оценку и реальные замечания по смете.</p>
      </section>
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
            <div className="buttonRow"><a className="btn" href={`${API_BASE}/v1/estimates/${estimate.id}/report`}>Скачать отчёт</a><a className="btn secondary" href="/upload">Новая проверка</a></div>
          </article>)}
        </div>}
      </section>
      <Footer/>
    </main>
  )
}
