import {useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

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
      const response = await fetch(`${API_BASE}/v1/estimates/upload`, {method:'POST', body:formData});
      const data = await response.json();
      if(!response.ok){
        throw new Error(data.error || 'Не удалось загрузить файл');
      }
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
        <p>Подходит для владельцев домов, прорабов, сметчиков и строительных компаний. Начните с Excel или PDF файла, который уже есть у вас.</p>
      </section>
      <section className="workspace twoColumns">
        <div className="uploadBox">
          <div className="uploadIcon">+</div>
          <h2>Добавьте файл сметы</h2>
          <p>API для проверки: <b>{API_BASE}</b></p>
          <input type="file" onChange={(event)=>setFile(event.target.files?.[0] || null)} />
          <button className="btn" type="button" onClick={submitUpload} disabled={status==='uploading'}>{status==='uploading' ? 'Проверяем...' : 'Начать проверку'}</button>
          {message && <p className={`statusText ${status}`}>{message}</p>}
          {result && <div className="resultBox"><b>Проверка создана</b><p>ID: {result.id}</p><p>Оценка: {result.score}</p><a className="btn secondary" href={`/reports`}>Открыть отчёты</a></div>}
        </div>
        <div className="card checklistCard">
          <h2>Что получает клиент</h2>
          <ul>
            <li>Файл сохраняется в системе</li>
            <li>Создаётся история проверок</li>
            <li>Формируется первичный отчёт</li>
            <li>Команде проще обсудить спорные расходы</li>
          </ul>
          <a className="btn secondary" href="/dashboard">Открыть кабинет</a>
        </div>
      </section>
      <Footer/>
    </main>
  )
}
