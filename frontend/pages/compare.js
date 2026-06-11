import {useEffect, useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function money(value){ return Number(value || 0).toLocaleString('ru-RU'); }
function fileSizeLabel(file){
  if(!file) return '';
  if(file.size > 1024 * 1024) return `${(file.size / 1024 / 1024).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(file.size / 1024))} KB`;
}

function CompareUploadCard({number,title,text,file,onChange}){
  return <div className="compareDrop"><span className="compareNumber">{number}</span><label className="modernUploadZone"><input className="modernUploadInput" type="file" accept=".xlsx,.xlsm,.csv,.pdf" onChange={(event)=>onChange(event.target.files?.[0] || null)} /><div className="modernUploadContent"><div className="modernUploadIcon">↥</div><span className="modernUploadHint">XLSX · CSV · PDF</span><h2>{file ? 'Файл выбран' : title}</h2><p>{file ? 'Версия готова к сравнению.' : text}</p></div></label>{file && <div className="modernFilePill"><div><b>{file.name}</b><br/><span>{fileSizeLabel(file)}</span></div><span>Готово</span></div>}</div>;
}

export default function Compare(){
  const [baseFile, setBaseFile] = useState(null);
  const [newFile, setNewFile] = useState(null);
  const [status, setStatus] = useState('idle');
  const [message, setMessage] = useState('');
  const [result, setResult] = useState(null);
  const [authorized, setAuthorized] = useState(false);
  const [checkingSession, setCheckingSession] = useState(true);

  useEffect(()=>{
    async function verifySession(){
      const token = window.localStorage.getItem('smetacheck_token');
      if(!token){ setCheckingSession(false); return; }
      try{
        const response = await fetch(`${API_BASE}/v1/auth/me`, {headers:{Authorization:`Bearer ${token}`}});
        if(!response.ok){
          window.localStorage.removeItem('smetacheck_token');
          window.localStorage.removeItem('smetacheck_user_email');
          setCheckingSession(false);
          return;
        }
        setAuthorized(true);
      }catch{
        setMessage('Не удалось связаться с API или PostgreSQL.');
      }finally{
        setCheckingSession(false);
      }
    }
    verifySession();
  }, []);

  async function compareFiles(){
    const token = window.localStorage.getItem('smetacheck_token');
    if(!token || !authorized){ setMessage('Сначала войдите в аккаунт.'); return; }
    if(!baseFile || !newFile){ setMessage('Загрузите исходную и новую версию сметы.'); return; }
    setStatus('loading');
    setMessage('Сравниваем две версии сметы...');
    setResult(null);

    const formData = new FormData();
    formData.append('base', baseFile);
    formData.append('new', newFile);

    try{
      const response = await fetch(`${API_BASE}/v1/estimates/compare`, {method:'POST', headers:{Authorization:`Bearer ${token}`}, body:formData});
      const data = await response.json();
      if(response.status === 401){
        window.localStorage.removeItem('smetacheck_token');
        setAuthorized(false);
        throw new Error('Сессия истекла. Войдите снова.');
      }
      if(!response.ok){ throw new Error(data.error || 'Не удалось сравнить сметы'); }
      setResult(data);
      setStatus('done');
      setMessage('Сравнение сохранено в PostgreSQL.');
    }catch(error){
      setStatus('error');
      setMessage(error.message || 'Не удалось сравнить сметы');
    }
  }

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact"><p className="eyebrow">Сравнение смет</p><h1>Сравните две версии сметы перед согласованием бюджета.</h1><p>Результат сохраняется в PostgreSQL и привязывается к вашему аккаунту.</p></section>
      <section className="workspace">
        {checkingSession && <div className="card"><p>Проверяем сессию...</p></div>}
        {!checkingSession && !authorized && <div className="emptyState"><h2>Нужно войти в аккаунт</h2><p>{message || 'Сравнение доступно только зарегистрированным пользователям.'}</p><a className="btn" href="/login">Войти или зарегистрироваться</a></div>}
        {!checkingSession && authorized && <>
          <div className="twoColumns"><CompareUploadCard number="01" title="Исходная смета" text="Добавьте первую версию документа." file={baseFile} onChange={setBaseFile}/><CompareUploadCard number="02" title="Новая версия" text="Добавьте обновлённую смету." file={newFile} onChange={setNewFile}/></div>
          <div className="card"><h2>Запустить сравнение</h2><p>Сервис сравнит позиции по названию и единице измерения, затем сохранит результат в PostgreSQL.</p><button className="btn" type="button" onClick={compareFiles} disabled={status==='loading'}>{status==='loading' ? 'Сравниваем...' : 'Сравнить сметы'}</button>{message && <p className={`statusText ${status}`}>{message}</p>}</div>
        </>}
      </section>
      {authorized && result && <section className="workspace"><div className="compareSummary"><div><strong>{money(result.base_total)}</strong><span>Было</span></div><div><strong>{money(result.new_total)}</strong><span>Стало</span></div><div><strong>{money(result.delta_total)}</strong><span>Разница</span></div></div><div className="twoColumns"><div className="card"><h2>Добавлено</h2>{(result.added || []).slice(0,8).map((item)=><p key={`a-${item.row}`}>{item.name} · {money(item.total)}</p>)}{(result.added || []).length===0 && <p>Новых позиций не найдено.</p>}</div><div className="card"><h2>Удалено</h2>{(result.removed || []).slice(0,8).map((item)=><p key={`r-${item.row}`}>{item.name} · {money(item.total)}</p>)}{(result.removed || []).length===0 && <p>Удалённых позиций не найдено.</p>}</div></div><div className="card"><h2>Изменены суммы</h2>{(result.changed || []).slice(0,10).map((item)=><p key={`${item.name}-${item.new_row}`}>{item.name}: было {money(item.base_total)}, стало {money(item.new_total)}, разница {money(item.delta_total)}</p>)}{(result.changed || []).length===0 && <p>Изменений по суммам не найдено.</p>}</div></section>}
      <Footer/>
    </main>
  );
}
