import {useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function fileSizeLabel(file){
  if(!file) return '';
  if(file.size > 1024 * 1024) return `${(file.size / 1024 / 1024).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(file.size / 1024))} KB`;
}

function authHeaders(){
  if(typeof window === 'undefined') return {};
  const token = window.localStorage.getItem('smetacheck_token');
  return token ? {Authorization: `Bearer ${token}`} : {};
}

export default function Upload(){
  const [file, setFile] = useState(null);
  const [status, setStatus] = useState('idle');
  const [message, setMessage] = useState('');
  const [result, setResult] = useState(null);

  async function submitUpload(){
    if(!file){
      setMessage('Сначала выберите файл сметы.');
      return;
    }
    setStatus('uploading');
    setMessage('Загружаем файл и готовим проверку...');
    setResult(null);

    const formData = new FormData();
    formData.append('file', file);

    try{
      const response = await fetch(`${API_BASE}/v1/estimates/upload`, {method:'POST', headers:authHeaders(), body:formData});
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить файл'); }
      setResult(data);
      setStatus('done');
      setMessage('Смета загружена. Отчёт и данные проверки созданы.');
    }catch(error){
      setStatus('error');
      setMessage(error.message || 'Не удалось загрузить файл');
    }
  }

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Проверка сметы</p>
        <h1>Загрузите смету и получите понятный отчёт для обсуждения.</h1>
        <p>Подходит для владельцев домов, прорабов, сметчиков и строительных компаний. Лучше всего работают XLSX, XLSM и CSV файлы.</p>
      </section>
      <section className="workspace twoColumns">
        <div className="uploadBox modernUploadShell">
          <label className="modernUploadZone">
            <input className="modernUploadInput" type="file" accept=".xlsx,.xlsm,.csv,.pdf" onChange={(event)=>setFile(event.target.files?.[0] || null)} />
            <div className="modernUploadContent">
              <div className="modernUploadIcon">↑</div>
              <span className="modernUploadHint">XLSX · XLSM · CSV · PDF</span>
              <h2>{file ? 'Файл выбран' : 'Выберите смету для проверки'}</h2>
              <p>{file ? 'Можно запускать анализ. Система прочитает строки и подготовит замечания.' : 'Нажмите на область или перетащите файл сюда. Нативная кнопка скрыта, всё в стиле SmetaCheck.'}</p>
            </div>
          </label>
          {file && <div className="modernFilePill"><div><b>{file.name}</b><br/><span>{fileSizeLabel(file)}</span></div><span>Готов к проверке</span></div>}
          <div className="modernMiniGrid"><span>Проверка сумм</span><span>Поиск дублей</span><span>Отчёт для обсуждения</span></div>
          <button className="btn" type="button" onClick={submitUpload} disabled={status==='uploading'}>{status==='uploading' ? 'Проверяем...' : 'Начать проверку'}</button>
          {message && <p className={`statusText ${status}`}>{message}</p>}
          {result && <div className="resultBox"><b>Проверка создана</b><p>ID: {result.id}</p><p>Оценка: {result.score}/100 · строк: {result.items_count || 0}</p><a className="btn secondary" href={`/reports/${result.id}`}>Открыть отчёт</a></div>}
        </div>
        <div className="card checklistCard">
          <h2>Что получает клиент</h2>
          <ul><li>Файл сохраняется в системе</li><li>Создаётся история проверок</li><li>Формируется первичный отчёт</li><li>Команде проще обсудить спорные расходы</li></ul>
          <a className="btn secondary" href="/dashboard">Открыть кабинет</a>
        </div>
      </section>
      <Footer/>
    </main>
  )
}
