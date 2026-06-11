import {useRouter} from 'next/router';
import {useEffect,useMemo,useState} from 'react';
import Nav from '../../components/Nav';
import Footer from '../../components/Footer';

const API_BASE=process.env.NEXT_PUBLIC_API_BASE||'http://localhost:8080';
const money=value=>Number(value||0).toLocaleString('ru-RU',{maximumFractionDigits:2});

function RiskBars({title,data=[]}){
  const max=Math.max(1,...data.map(item=>Number(item.value||0)));
  return <div className="card"><h3>{title}</h3>{data.map(item=><div className="riskBar" key={item.label}><span>{item.label}</span><div><b style={{width:`${Math.round(Number(item.value||0)/max*100)}%`}}/></div><em>{item.value||0}</em></div>)}</div>;
}

function FindingTable({items=[]}){
  return <div className="reportTable"><table><thead><tr><th>Риск</th><th>Строка</th><th>Источник</th><th>Статус</th><th>Проблема</th><th>Действие</th></tr></thead><tbody>{items.map((item,index)=><tr key={`${item.file_id}-${item.row}-${index}`}><td>{item.severity}</td><td>{item.row||'—'}</td><td>{(item.sources||[]).join(' + ')}</td><td>{item.reconciliation}</td><td><b>{item.title}</b><br/>{item.detail}</td><td>{item.suggested_action}</td></tr>)}</tbody></table></div>;
}

export default function BatchReport(){
  const router=useRouter();
  const {id}=router.query;
  const [batch,setBatch]=useState(null);
  const [files,setFiles]=useState([]);
  const [error,setError]=useState('');
  const [loading,setLoading]=useState(true);

  useEffect(()=>{
    if(!id)return;
    let cancelled=false;
    async function load(){
      const token=window.localStorage.getItem('smetacheck_token');
      if(!token){setError('Нужно войти в аккаунт');setLoading(false);return;}
      try{
        const response=await fetch(`${API_BASE}/v1/analysis-batches/${id}`,{headers:{Authorization:`Bearer ${token}`}});
        const data=await response.json();
        if(!response.ok)throw new Error(data.error||'Не удалось загрузить пакет');
        if(cancelled)return;
        setBatch(data.batch);setFiles(data.files||[]);setError('');
        if(data.batch.status==='pending'||data.batch.status==='processing')window.setTimeout(load,2000);
      }catch(err){if(!cancelled)setError(err.message||'Ошибка загрузки отчёта');}
      finally{if(!cancelled)setLoading(false);}
    }
    load();
    return()=>{cancelled=true;};
  },[id]);

  const report=batch?.report;
  const completed=report&&batch?.status==='completed';
  const progress=useMemo(()=>batch?.file_count?Math.round((batch.completed_count||0)/batch.file_count*100):0,[batch]);

  return <main className="page"><Nav/>
    <section className="pageHero compact"><p className="eyebrow">Единый отчёт</p><h1>Backend + AI анализ пакета</h1><p>{batch?`${batch.provider} · ${batch.model}`:'Загружаем данные...'}</p></section>
    <section className="workspace">
      {loading&&!batch&&<div className="card"><p>Загружаем отчёт...</p></div>}
      {error&&<div className="card"><h2>Ошибка</h2><p>{error}</p></div>}
      {batch&&!completed&&<div className="card"><h2>Статус: {batch.status}</h2><p>Обработано {batch.completed_count||0} из {batch.file_count} файлов · {progress}%</p>{batch.error_message&&<p>{batch.error_message}</p>}<ul>{files.map(file=><li key={file.id}>{file.file_name}: {file.status}{file.error?` — ${file.error}`:''}</li>)}</ul></div>}
      {completed&&<>
        <div className="buttonRow"><button className="btn" type="button" onClick={()=>window.print()}>Сохранить как PDF</button><a className="btn secondary" href="/upload">Новый анализ</a></div>
        <div className="card"><h2>{report.summary}</h2><p><b>Итоговый риск:</b> {report.overall_risk_level}</p><p><b>Провайдер:</b> {report.provider} · <b>Модель:</b> {report.model}</p></div>
        <div className="grid statsGrid"><article className="statCard"><strong>{report.total_files}</strong><span>Файлов</span></article><article className="statCard"><strong>{report.total_rows}</strong><span>Строк</span></article><article className="statCard"><strong>{money(report.total_amount)}</strong><span>Сумма</span></article><article className="statCard"><strong>{report.overall_risk_level}</strong><span>Риск</span></article></div>
        <RiskBars title="Общий график рисков" data={report.unified_chart||[]}/>
        {(report.files||[]).map(file=><section className="card" key={file.estimate_id}><h2>{file.file_name}</h2><p><b>Backend score:</b> {file.backend_score}/100 · <b>AI риск:</b> {file.ai_risk_level} · <b>Итог:</b> {file.final_risk_level} · <b>Совпадение:</b> {file.agreement_score}%</p><p>{file.summary}</p><div className="grid"><RiskBars title="Backend" data={file.backend_chart||[]}/><RiskBars title="AI" data={file.ai_chart||[]}/><RiskBars title="Общий" data={file.unified_chart||[]}/></div><h3>Объединённые замечания</h3><FindingTable items={file.findings||[]}/><h3>Рекомендации</h3><ul>{(file.recommendations||[]).map(item=><li key={item}>{item}</li>)}</ul></section>)}
        {!!report.cross_file?.length&&<div className="card"><h2>Сравнение файлов</h2><ul>{report.cross_file.map((item,index)=><li key={index}>{item.description} Значение: {money(item.delta)}</li>)}</ul></div>}
        <div className="card"><h2>Общие рекомендации</h2><ul>{(report.recommendations||[]).map(item=><li key={item}>{item}</li>)}</ul></div>
      </>}
    </section><Footer/></main>;
}
