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
        <p>История проверок помогает быстрее объяснить бюджет, спорные строки и следующие действия.</p>
      </section>
      <section className="workspace">
        <div className="buttonRow"><button className="btn secondary" type="button" onClick={loadReports}>Обновить</button><a className="btn" href="/upload">Загрузить новую смету</a></div>
        {loading && <div className="card"><p>Загружаем отчёты...</p></div>}
        {error && <div className="card"><h2>Ошибка API</h2><p>{error}</p><p>Проверьте, что Go API запущен на {API_BASE}.</p></div>}
        {!loading && !error && estimates.length === 0 && <div className="card"><h2>Отчётов пока нет</h2><p>Загрузите первую смету, чтобы получить отчёт.</p><a className="btn" href="/upload">Загрузить смету</a></div>}
        {!loading && !error && estimates.length > 0 && <div className="reportTable">
          <div className="tableHead"><span>ID</span><span>Файл</span><span>Оценка</span><span>Статус</span><span>Отчёт</span></div>
          {estimates.map((estimate) => <div className="tableRow" key={estimate.id}><span>{estimate.id}</span><b>{estimate.file_name}</b><strong>{estimate.score}</strong><em>{estimate.status}</em><a href={`${API_BASE}/v1/estimates/${estimate.id}/report`}>Скачать</a></div>)}
        </div>}
      </section>
      <Footer/>
    </main>
  )
}
