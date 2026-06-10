import {useRouter} from 'next/router';
import {useEffect, useMemo, useState} from 'react';
import Nav from '../../components/Nav';
import Footer from '../../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function money(value){ return Number(value || 0).toLocaleString('ru-RU'); }
function authHeaders(){
  if(typeof window === 'undefined') return {};
  const token = window.localStorage.getItem('smetacheck_token');
  return token ? {Authorization: `Bearer ${token}`} : {};
}

export default function ReportDetail(){
  const router = useRouter();
  const {id} = router.query;
  const [estimate, setEstimate] = useState(null);
  const [ai, setAi] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  async function loadReport(){
    if(!id) return;
    setLoading(true);
    setError('');
    try{
      const response = await fetch(`${API_BASE}/v1/estimates/${id}`, {headers: authHeaders()});
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить отчёт'); }
      setEstimate(data);
      const aiResponse = await fetch(`${API_BASE}/v1/ai/estimate-summary/${id}`, {headers: authHeaders()});
      const aiData = await aiResponse.json();
      if(aiResponse.ok){ setAi(aiData); }
    }catch(err){ setError(err.message || 'Не удалось загрузить отчёт'); }
    finally{ setLoading(false); }
  }

  useEffect(()=>{ loadReport(); }, [id]);

  const riskStats = useMemo(()=>{
    const stats = {High:0, Medium:0, Low:0, Info:0};
    (estimate?.findings || []).forEach((finding)=>{ stats[finding.severity] = (stats[finding.severity] || 0) + 1; });
    return stats;
  }, [estimate]);

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact"><p className="eyebrow">Детальный отчёт</p><h1>{estimate?.file_name || 'Отчёт проверки сметы'}</h1><p>Проверка, риски, AI-вывод и список вопросов к подрядчику в одном месте.</p></section>
      <section className="workspace">
        {loading && <div className="card"><p>Загружаем отчёт...</p></div>}
        {error && <div className="card"><h2>Ошибка</h2><p>{error}</p><button className="btn" onClick={loadReport}>Повторить</button></div>}
        {!loading && !error && estimate && <>
          <div className="buttonRow"><button className="btn" type="button" onClick={()=>window.print()}>Сохранить как PDF</button><a className="btn secondary" href={`${API_BASE}/v1/estimates/${estimate.id}/report`}>Скачать TXT</a><a className="btn secondary" href="/reports">Все отчёты</a></div>
          <div className="grid statsGrid">
            <article className="statCard"><strong>{estimate.score}/100</strong><span>Оценка</span></article>
            <article className="statCard"><strong>{estimate.items_count || 0}</strong><span>Строк</span></article>
            <article className="statCard"><strong>{money(estimate.total_amount)}</strong><span>Сумма</span></article>
            <article className="statCard"><strong>{(estimate.findings || []).length}</strong><span>Замечаний</span></article>
          </div>
          <div className="twoColumns">
            <div className="card"><h2>График рисков</h2>{Object.entries(riskStats).map(([level,count]) => <div className="riskBar" key={level}><span>{level}</span><div><b style={{width:`${Math.min(100, count*18)}%`}} /></div><em>{count}</em></div>)}</div>
            <div className="card"><h2>AI-вывод</h2>{ai ? <><p><b>Риск: {ai.risk_level}</b></p><p>{ai.executive_brief}</p><p>{ai.recommendation}</p></> : <p>AI-вывод пока недоступен.</p>}</div>
          </div>
          <div className="card"><h2>Что обсудить с подрядчиком</h2><ul>{(ai?.questions || ['Проверьте позиции высокого и среднего риска.', 'Уточните строки без количества, цены или суммы.', 'Сравните итоговую сумму с договором.']).map((item)=><li key={item}>{item}</li>)}</ul></div>
          <div className="card"><h2>Замечания</h2><div className="reportTable"><table><thead><tr><th>Риск</th><th>Проблема</th><th>Детали</th></tr></thead><tbody>{(estimate.findings || []).map((finding,index)=><tr key={index}><td>{finding.severity}</td><td>{finding.title}</td><td>{finding.detail}</td></tr>)}</tbody></table></div></div>
          <div className="card"><h2>Позиции сметы</h2><div className="reportTable"><table><thead><tr><th>Строка</th><th>Название</th><th>Ед.</th><th>Кол-во</th><th>Цена</th><th>Сумма</th></tr></thead><tbody>{(estimate.items || []).slice(0,80).map((item)=><tr key={item.row}><td>{item.row}</td><td>{item.name}</td><td>{item.unit}</td><td>{item.quantity}</td><td>{money(item.unit_price)}</td><td>{money(item.total)}</td></tr>)}</tbody></table></div></div>
        </>}
      </section>
      <Footer/>
    </main>
  )
}
