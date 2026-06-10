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
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить кабинет'); }
      setEstimates(data.estimates || []);
    }catch(err){ setError(err.message || 'Не удалось загрузить кабинет'); }
    finally{ setLoading(false); }
  }

  useEffect(()=>{ loadDashboard(); }, []);

  const stats = useMemo(()=>{
    const total = estimates.length;
    const ready = estimates.filter(item => item.status === 'ready').length;
    const avg = total ? Math.round(estimates.reduce((sum,item)=>sum+(item.score || 0),0)/total) : 0;
    const findings = estimates.reduce((sum,item)=>sum+((item.findings || []).length),0);
    return [[String(total),'Проверок'], [String(ready),'Готовых отчётов'], [String(avg),'Средняя оценка'], [String(findings),'Замечаний']];
  }, [estimates]);

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Кабинет</p>
        <h1>Контролируйте все сметы и отчёты в одном месте.</h1>
        <p>Владелец, прораб или компания видит историю проверок, оценки, замечания и готовые отчёты без хаоса в чатах и файлах.</p>
      </section>
      <section className="workspace">
        <div className="buttonRow"><button className="btn secondary" type="button" onClick={loadDashboard}>Обновить</button><a className="btn" href="/upload">Загрузить смету</a></div>
        {loading && <div className="card"><p>Загружаем кабинет...</p></div>}
        {error && <div className="card"><h2>Ошибка API</h2><p>{error}</p><p>Проверьте, что Go API запущен на {API_BASE}.</p></div>}
        {!loading && !error && <div className="grid statsGrid">
          {stats.map(([value,label]) => <article className="statCard" key={label}><strong>{value}</strong><span>{label}</span></article>)}
        </div>}
        {!loading && !error && <div className="twoColumns">
          <div className="card">
            <h2>Последние проверки</h2>
            <div className="timeline">{estimates.slice(0,4).map((item,i) => <p key={item.id}><b>{String(i+1).padStart(2,'0')}</b><span>{item.file_name} · оценка {item.score}</span></p>)}</div>
            {estimates.length === 0 && <p>Пока нет загруженных смет. Начните с первой проверки.</p>}
          </div>
          <div className="card">
            <h2>Следующий шаг</h2>
            <p>Загрузите смету, получите отчёт и используйте его как аргумент при обсуждении бюджета.</p>
            <div className="buttonRow"><a className="btn" href="/upload">Загрузить</a><a className="btn secondary" href="/reports">Открыть отчёты</a></div>
          </div>
        </div>}
      </section>
      <Footer/>
    </main>
  )
}
