import {useRouter} from 'next/router';
import {useEffect, useMemo, useState} from 'react';
import Nav from '../../components/Nav';
import Footer from '../../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function money(value){ return Number(value || 0).toLocaleString('ru-RU'); }

export default function ReportDetail(){
  const router = useRouter();
  const {id} = router.query;
  const [estimate, setEstimate] = useState(null);
  const [ai, setAi] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [authorized, setAuthorized] = useState(false);
  const [downloadLoading, setDownloadLoading] = useState(false);

  function clearSession(){
    window.localStorage.removeItem('smetacheck_token');
    window.localStorage.removeItem('smetacheck_user_email');
    setAuthorized(false);
  }

  async function loadReport(){
    if(!id) return;
    const token = window.localStorage.getItem('smetacheck_token');
    if(!token){
      setAuthorized(false);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError('');
    try{
      const meResponse = await fetch(`${API_BASE}/v1/auth/me`, {headers:{Authorization:`Bearer ${token}`}});
      if(!meResponse.ok){
        clearSession();
        throw new Error('Сессия истекла. Войдите снова.');
      }
      setAuthorized(true);

      const response = await fetch(`${API_BASE}/v1/estimates/${id}`, {headers:{Authorization:`Bearer ${token}`}});
      const data = await response.json();
      if(response.status === 401){
        clearSession();
        throw new Error('Сессия истекла. Войдите снова.');
      }
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить отчёт'); }
      setEstimate(data);

      const aiResponse = await fetch(`${API_BASE}/v1/ai/estimate-summary/${id}`, {headers:{Authorization:`Bearer ${token}`}});
      if(aiResponse.ok){
        const aiData = await aiResponse.json();
        setAi(aiData);
      }
    }catch(err){
      setError(err.message || 'Не удалось загрузить отчёт');
    }finally{
      setLoading(false);
    }
  }

  async function downloadTXT(){
    if(!estimate) return;
    const token = window.localStorage.getItem('smetacheck_token');
    if(!token){
      clearSession();
      setError('Сессия истекла. Войдите снова.');
      return;
    }
    setDownloadLoading(true);
    setError('');
    try{
      const response = await fetch(`${API_BASE}/v1/estimates/${estimate.id}/report`, {headers:{Authorization:`Bearer ${token}`}});
      if(response.status === 401){
        clearSession();
        throw new Error('Сессия истекла. Войдите снова.');
      }
      if(!response.ok){
        let message = 'Не удалось скачать TXT-отчёт';
        try{
          const data = await response.json();
          message = data.error || message;
        }catch{}
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
      setError(err.message || 'Не удалось скачать TXT-отчёт');
    }finally{
      setDownloadLoading(false);
    }
  }

  useEffect(()=>{ loadReport(); }, [id]);

  const riskStats = useMemo(()=>{
    const stats = {High:0, Medium:0, Low:0, Info:0};
    (estimate?.findings || []).forEach((finding)=>{
      stats[finding.severity] = (stats[finding.severity] || 0) + 1;
    });
    return stats;
  }, [estimate]);

  const score = Math.max(0, Math.min(100, estimate?.score || 0));

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact"><p className="eyebrow">Детальный отчёт</p><h1>{estimate?.file_name || 'Отчёт проверки сметы'}</h1><p>Проверка, риски, AI-вывод и список вопросов к подрядчику в одном месте.</p></section>
      <section className="workspace">
        {loading && <div className="card"><p>Проверяем сессию и загружаем отчёт...</p></div>}
        {!loading && !authorized && <div className="emptyState"><h2>Нужно войти в аккаунт</h2><p>{error || 'Этот отчёт доступен только владельцу аккаунта.'}</p><a className="btn" href="/login">Войти или зарегистрироваться</a></div>}
        {!loading && authorized && error && <div className="card"><h2>Ошибка</h2><p>{error}</p><button className="btn" onClick={loadReport}>Повторить</button></div>}
        {!loading && authorized && !error && estimate && <>
          <div className="buttonRow"><button className="btn" type="button" onClick={()=>window.print()}>Сохранить как PDF</button><button className="btn secondary" type="button" onClick={downloadTXT} disabled={downloadLoading}>{downloadLoading ? 'Скачиваем...' : 'Скачать TXT'}</button><a className="btn secondary" href="/reports">Все отчёты</a></div>
          <div className="twoColumns">
            <article className="card"><div className="scoreMeter" style={{'--score':`${score}%`}}><div><strong>{score}</strong><span>из 100</span></div></div><h2>Итоговая оценка</h2><p>Чем выше оценка, тем меньше автоматических замечаний найдено в смете.</p></article>
            <article className="card aiExpertCard"><p className="eyebrow">AI Expert Review</p><h2>Вывод по смете</h2>{ai ? <><p><b>Риск: {ai.risk_level}</b> · Качество данных: <b>{ai.data_quality_score || score}/100</b></p><p>{ai.executive_brief}</p><p>{ai.recommendation}</p></> : <p>AI-вывод пока недоступен.</p>}</article>
          </div>
          <div className="grid statsGrid"><article className="statCard"><strong>{estimate.items_count || 0}</strong><span>Строк</span></article><article className="statCard"><strong>{money(estimate.total_amount)}</strong><span>Сумма</span></article><article className="statCard"><strong>{(estimate.findings || []).length}</strong><span>Замечаний</span></article><article className="statCard"><strong>{riskStats.High}</strong><span>High risk</span></article></div>
          <div className="twoColumns">
            <div className="card"><h2>График рисков</h2>{Object.entries(riskStats).map(([level,count]) => <div className="riskBar" key={level}><span>{level}</span><div><b style={{width:`${Math.min(100, count*18)}%`}} /></div><em>{count}</em></div>)}</div>
            <div className="card"><h2>Приоритетные действия</h2><div className="priorityList">{(ai?.priority_actions || ['Проверьте замечания высокого и среднего риска.', 'Уточните спорные позиции у подрядчика.', 'После исправления повторно загрузите смету.']).map((item)=><div className="priorityItem" key={item}>{item}</div>)}</div></div>
          </div>
          <div className="card"><h2>Финансовые флаги</h2><ul>{(ai?.cost_flags || ['Проверьте крупные позиции и итоговую сумму.']).map((item)=><li key={item}>{item}</li>)}</ul></div>
          <div className="card"><h2>Что обсудить с подрядчиком</h2><ul>{(ai?.questions || ['Проверьте позиции высокого и среднего риска.', 'Уточните строки без количества, цены или суммы.', 'Сравните итоговую сумму с договором.']).map((item)=><li key={item}>{item}</li>)}</ul></div>
          <div className="card"><h2>Замечания</h2><div className="reportTable"><table><thead><tr><th>Риск</th><th>Проблема</th><th>Детали</th></tr></thead><tbody>{(estimate.findings || []).map((finding,index)=><tr key={index}><td>{finding.severity}</td><td>{finding.title}</td><td>{finding.detail}</td></tr>)}</tbody></table></div></div>
          <div className="card"><h2>Позиции сметы</h2><div className="reportTable"><table><thead><tr><th>Строка</th><th>Название</th><th>Ед.</th><th>Кол-во</th><th>Цена</th><th>Сумма</th></tr></thead><tbody>{(estimate.items || []).slice(0,80).map((item)=><tr key={item.row}><td>{item.row}</td><td>{item.name}</td><td>{item.unit}</td><td>{item.quantity}</td><td>{money(item.unit_price)}</td><td>{money(item.total)}</td></tr>)}</tbody></table></div></div>
        </>}
      </section>
      <Footer/>
    </main>
  );
}
