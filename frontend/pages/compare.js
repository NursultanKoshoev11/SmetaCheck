import {useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function money(value){
  return Number(value || 0).toLocaleString('ru-RU');
}

function fileSizeLabel(file){
  if(!file) return '';
  if(file.size > 1024 * 1024) return `${(file.size / 1024 / 1024).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(file.size / 1024))} KB`;
}

function CompareUploadCard({number,title,text,file,onChange}){
  return (
    <div className="compareDrop">
      <span className="compareNumber">{number}</span>
      <label className="modernUploadZone">
        <input className="modernUploadInput" type="file" accept=".xlsx,.xlsm,.csv,.pdf" onChange={(event)=>onChange(event.target.files?.[0] || null)} />
        <div className="modernUploadContent">
          <div className="modernUploadIcon">↥</div>
          <span className="modernUploadHint">XLSX · CSV · PDF</span>
          <h2>{file ? 'Файл выбран' : title}</h2>
          <p>{file ? 'Версия готова к сравнению.' : text}</p>
        </div>
      </label>
      {file && <div className="modernFilePill"><div><b>{file.name}</b><br/><span>{fileSizeLabel(file)}</span></div><span>Готово</span></div>}
    </div>
  )
}

export default function Compare(){
  const [baseFile, setBaseFile] = useState(null);
  const [newFile, setNewFile] = useState(null);
  const [status, setStatus] = useState('idle');
  const [message, setMessage] = useState('');
  const [result, setResult] = useState(null);

  async function compareFiles(){
    if(!baseFile || !newFile){
      setMessage('Загрузите исходную и новую версию сметы.');
      return;
    }
    setStatus('loading');
    setMessage('Сравниваем две версии сметы...');
    setResult(null);

    const formData = new FormData();
    formData.append('base', baseFile);
    formData.append('new', newFile);

    try{
      const response = await fetch(`${API_BASE}/v1/estimates/compare`, {method:'POST', body:formData});
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'Не удалось сравнить сметы'); }
      setResult(data);
      setStatus('done');
      setMessage('Сравнение готово. Проверьте изменения ниже.');
    }catch(error){
      setStatus('error');
      setMessage(error.message || 'Не удалось сравнить сметы');
    }
  }

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Сравнение смет</p>
        <h1>Сравните две версии сметы перед согласованием бюджета.</h1>
        <p>Загрузите старую и новую версию. SmetaCheck покажет добавленные, удалённые и изменённые позиции.</p>
      </section>
      <section className="workspace twoColumns">
        <CompareUploadCard number="01" title="Исходная смета" text="Добавьте первую версию документа." file={baseFile} onChange={setBaseFile}/>
        <CompareUploadCard number="02" title="Новая версия" text="Добавьте обновлённую смету." file={newFile} onChange={setNewFile}/>
      </section>
      <section className="workspace">
        <div className="card"><h2>Запустить сравнение</h2><p>Сервис сравнит позиции по названию и единице измерения, затем покажет разницу по суммам.</p><button className="btn" type="button" onClick={compareFiles} disabled={status==='loading'}>{status==='loading' ? 'Сравниваем...' : 'Сравнить сметы'}</button>{message && <p className={`statusText ${status}`}>{message}</p>}</div>
      </section>
      {result && <section className="workspace">
        <div className="grid statsGrid">
          <article className="statCard"><strong>{money(result.base_total)}</strong><span>Исходная сумма</span></article>
          <article className="statCard"><strong>{money(result.new_total)}</strong><span>Новая сумма</span></article>
          <article className="statCard"><strong>{money(result.delta_total)}</strong><span>Разница</span></article>
          <article className="statCard"><strong>{(result.findings || []).length}</strong><span>Замечаний</span></article>
        </div>
        <div className="twoColumns">
          <div className="card"><h2>Добавлено</h2>{(result.added || []).slice(0,8).map((item)=><p key={`a-${item.row}`}>{item.name} · {money(item.total)}</p>)}{(result.added || []).length===0 && <p>Новых позиций не найдено.</p>}</div>
          <div className="card"><h2>Удалено</h2>{(result.removed || []).slice(0,8).map((item)=><p key={`r-${item.row}`}>{item.name} · {money(item.total)}</p>)}{(result.removed || []).length===0 && <p>Удалённых позиций не найдено.</p>}</div>
        </div>
        <div className="card"><h2>Изменены суммы</h2>{(result.changed || []).slice(0,10).map((item)=><p key={`${item.name}-${item.new_row}`}>{item.name}: было {money(item.base_total)}, стало {money(item.new_total)}, разница {money(item.delta_total)}</p>)}{(result.changed || []).length===0 && <p>Изменений по суммам не найдено.</p>}</div>
      </section>}
      <Footer/>
    </main>
  )
}
