import {useRouter} from 'next/router';
import {useEffect,useMemo,useState} from 'react';
import Nav from '../../components/Nav';
import Footer from '../../components/Footer';
import {apiFetch,readJSON} from '../../lib/api';

const money=value=>Number(value||0).toLocaleString('ru-RU',{maximumFractionDigits:2});

export default function BatchReport(){
  const {query}=useRouter();
  const [batch,setBatch]=useState(null);const [files,setFiles]=useState([]);const [error,setError]=useState('');const [loading,setLoading]=useState(true);

  useEffect(()=>{
    if(!query.id)return;
    let cancelled=false;let timer;
    async function load(){
      try{const response=await apiFetch(`/v1/analysis-batches/${query.id}`);const data=await readJSON(response);if(response.status===401)throw new Error('Нужно войти в аккаунт');if(!response.ok)throw new Error(data.error||'Не удалось загрузить пакет');if(cancelled)return;setBatch(data.batch);setFiles(data.files||[]);setError('');if(['pending','processing'].includes(data.batch.status))timer=setTimeout(load,2000);}catch(err){if(!cancelled)setError(err.message||'Ошибка загрузки отчёта');}finally{if(!cancelled)setLoading(false);}
    }
    load();return()=>{cancelled=true;if(timer)clearTimeout(timer);};
  },[query.id]);

  const report=batch?.report;const completed=report&&batch?.status==='completed';const progress=useMemo(()=>batch?.file_count?Math.round((batch.completed_count||0)/batch.file_count*100):0,[batch]);
  return <main className="page"><Nav/><section className="pageHero compact"><p className="eyebrow">Единый отчёт</p><h1>Backend + AI анализ пакета</h1></section><section className="workspace">
    {loading&&!batch&&<div className="card">Загружаем отчёт...</div>}
    {error&&<div className="card"><h2>Ошибка</h2><p>{error}</p><a className="btn" href="/login">Войти</a></div>}
    {batch&&!completed&&<div className="card"><h2>Статус: {batch.status}</h2><p>Обработано {batch.completed_count||0} из {batch.file_count} · {progress}%</p><ul>{files.map(file=><li key={file.id}>{file.file_name}: {file.status}</li>)}</ul></div>}
    {completed&&<><div className="buttonRow"><button className="btn" onClick={()=>window.print()}>Сохранить как PDF</button><a className="btn secondary" href="/upload">Новый анализ</a></div><div className="card"><h2>{report.summary}</h2><p><b>Итоговый риск:</b> {report.overall_risk_level}</p></div><div className="grid statsGrid"><article className="statCard"><strong>{report.total_files}</strong><span>Файлов</span></article><article className="statCard"><strong>{report.total_rows}</strong><span>Строк</span></article><article className="statCard"><strong>{money(report.total_amount)}</strong><span>Сумма</span></article></div>{(report.files||[]).map(file=><section className="card" key={file.estimate_id}><h2>{file.file_name}</h2><p>{file.summary}</p><ul>{(file.recommendations||[]).map(item=><li key={item}>{item}</li>)}</ul></section>)}</>}
  </section><Footer/></main>;
}
