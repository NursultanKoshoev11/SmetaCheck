import {useEffect,useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';
import {apiFetch,currentUser,readJSON} from '../lib/api';

const money=value=>Number(value||0).toLocaleString('ru-RU');

export default function Compare(){
  const [baseFile,setBaseFile]=useState(null);const [newFile,setNewFile]=useState(null);const [status,setStatus]=useState('idle');const [message,setMessage]=useState('');const [result,setResult]=useState(null);const [authorized,setAuthorized]=useState(false);const [checking,setChecking]=useState(true);

  useEffect(()=>{let active=true;currentUser().then(user=>{if(active)setAuthorized(Boolean(user));}).catch(()=>setMessage('Не удалось связаться с API.')).finally(()=>{if(active)setChecking(false);});return()=>{active=false;};},[]);

  async function compareFiles(){
    if(!authorized){setMessage('Сначала войдите в аккаунт.');return;}
    if(!baseFile||!newFile){setMessage('Загрузите исходную и новую версии сметы.');return;}
    const form=new FormData();form.append('base',baseFile);form.append('new',newFile);setStatus('loading');setMessage('Сравниваем две версии...');setResult(null);
    try{const response=await apiFetch('/v1/estimates/compare',{method:'POST',body:form});const data=await readJSON(response);if(response.status===401){setAuthorized(false);throw new Error('Сессия истекла. Войдите снова.');}if(!response.ok)throw new Error(data.error||'Не удалось сравнить сметы');setResult(data);setStatus('done');setMessage('Сравнение сохранено.');}catch(err){setStatus('error');setMessage(err.message||'Не удалось сравнить сметы');}
  }

  return <main className="page"><Nav/><section className="pageHero compact"><p className="eyebrow">Сравнение смет</p><h1>Сравните две версии сметы перед согласованием бюджета.</h1></section><section className="workspace">
    {checking&&<div className="card">Проверяем сессию...</div>}
    {!checking&&!authorized&&<div className="emptyState"><h2>Нужно войти</h2><p>{message||'Сравнение доступно только зарегистрированным пользователям.'}</p><a className="btn" href="/login">Войти</a></div>}
    {!checking&&authorized&&<><div className="twoColumns"><label className="modernUploadZone"><input className="modernUploadInput" type="file" accept=".xlsx,.xlsm,.csv" onChange={e=>setBaseFile(e.target.files?.[0]||null)}/><div className="modernUploadContent"><h2>{baseFile?baseFile.name:'Исходная смета'}</h2></div></label><label className="modernUploadZone"><input className="modernUploadInput" type="file" accept=".xlsx,.xlsm,.csv" onChange={e=>setNewFile(e.target.files?.[0]||null)}/><div className="modernUploadContent"><h2>{newFile?newFile.name:'Новая версия'}</h2></div></label></div><div className="card"><button className="btn" onClick={compareFiles} disabled={status==='loading'}>{status==='loading'?'Сравниваем...':'Сравнить сметы'}</button>{message&&<p className={`statusText ${status}`}>{message}</p>}</div></>}
  </section>{authorized&&result&&<section className="workspace"><div className="compareSummary"><div><strong>{money(result.base_total)}</strong><span>Было</span></div><div><strong>{money(result.new_total)}</strong><span>Стало</span></div><div><strong>{money(result.delta_total)}</strong><span>Разница</span></div></div><div className="twoColumns"><div className="card"><h2>Добавлено</h2>{(result.added||[]).map(item=><p key={`a-${item.row}`}>{item.name} · {money(item.total)}</p>)}</div><div className="card"><h2>Удалено</h2>{(result.removed||[]).map(item=><p key={`r-${item.row}`}>{item.name} · {money(item.total)}</p>)}</div></div></section>}<Footer/></main>;
}
