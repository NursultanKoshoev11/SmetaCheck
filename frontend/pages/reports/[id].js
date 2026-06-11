import {useRouter} from 'next/router';
import {useEffect,useState} from 'react';
import Nav from '../../components/Nav';
import Footer from '../../components/Footer';
import {apiFetch,apiJSON,currentUser,readJSON} from '../../lib/api';

const money=value=>Number(value||0).toLocaleString('ru-RU');

export default function ReportDetail(){
  const {query}=useRouter();
  const [estimate,setEstimate]=useState(null);
  const [ai,setAi]=useState(null);
  const [loading,setLoading]=useState(true);
  const [error,setError]=useState('');

  async function load(){
    if(!query.id)return;
    setLoading(true);setError('');
    try{
      if(!await currentUser())throw new Error('Сессия истекла. Войдите снова.');
      const result=await apiJSON(`/v1/estimates/${query.id}`);
      if(!result.response.ok)throw new Error(result.data.error||'Не удалось загрузить отчёт');
      setEstimate(result.data);
      const aiResult=await apiJSON(`/v1/ai/estimate-summary/${query.id}`);
      if(aiResult.response.ok)setAi(aiResult.data);
    }catch(err){setError(err.message||'Не удалось загрузить отчёт');}
    finally{setLoading(false);}
  }

  async function download(){
    const response=await apiFetch(`/v1/estimates/${estimate.id}/report`);
    if(!response.ok){const data=await readJSON(response);setError(data.error||'Не удалось скачать отчёт');return;}
    const url=URL.createObjectURL(await response.blob());
    const link=document.createElement('a');link.href=url;link.download=`${estimate.id}_report.txt`;link.click();URL.revokeObjectURL(url);
  }

  useEffect(()=>{load();},[query.id]);
  return <main className="page"><Nav/><section className="pageHero compact"><p className="eyebrow">Детальный отчёт</p><h1>{estimate?.file_name||'Отчёт проверки сметы'}</h1></section><section className="workspace">
    {loading&&<div className="card">Загружаем отчёт...</div>}
    {error&&<div className="card"><h2>Ошибка</h2><p>{error}</p><a className="btn" href="/login">Войти</a></div>}
    {!loading&&!error&&estimate&&<><div className="buttonRow"><button className="btn" onClick={()=>window.print()}>Сохранить как PDF</button><button className="btn secondary" onClick={download}>Скачать TXT</button><a className="btn secondary" href="/reports">Все отчёты</a></div><div className="grid statsGrid"><article className="statCard"><strong>{estimate.score}</strong><span>Оценка</span></article><article className="statCard"><strong>{estimate.items_count||0}</strong><span>Строк</span></article><article className="statCard"><strong>{money(estimate.total_amount)}</strong><span>Сумма</span></article><article className="statCard"><strong>{(estimate.findings||[]).length}</strong><span>Замечаний</span></article></div>{ai&&<div className="card"><h2>AI-вывод: {ai.risk_level}</h2><p>{ai.executive_brief}</p><p>{ai.recommendation}</p></div>}<div className="card"><h2>Замечания</h2><div className="reportTable"><table><thead><tr><th>Риск</th><th>Проблема</th><th>Детали</th></tr></thead><tbody>{(estimate.findings||[]).map((item,index)=><tr key={index}><td>{item.severity}</td><td>{item.title}</td><td>{item.detail}</td></tr>)}</tbody></table></div></div><div className="card"><h2>Позиции</h2><div className="reportTable"><table><thead><tr><th>Строка</th><th>Название</th><th>Кол-во</th><th>Цена</th><th>Сумма</th></tr></thead><tbody>{(estimate.items||[]).slice(0,100).map(item=><tr key={item.row}><td>{item.row}</td><td>{item.name}</td><td>{item.quantity}</td><td>{money(item.unit_price)}</td><td>{money(item.total)}</td></tr>)}</tbody></table></div></div></>}
  </section><Footer/></main>;
}
