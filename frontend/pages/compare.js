import {useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function money(value){
  return Number(value || 0).toLocaleString('ru-RU');
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
        <div className="compareDrop"><span>01</span><h2>Исходная смета</h2><p>Первая версия документа.</p><input type="file" onChange={(event)=>setBaseFile(event.target.files?.[0] || null)} /></div>
        <div className="compareDrop"><span>02</span><h2>Новая версия</h2><p>Обновлённая смета.</p><input type="file" onChange={(event)=>setNewFile(event.target.files?.[0] || null)} /></div>
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
