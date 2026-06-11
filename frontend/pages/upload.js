import {useEffect, useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function fileSizeLabel(file){
  if(!file) return '';
  if(file.size > 1024 * 1024) return `${(file.size / 1024 / 1024).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(file.size / 1024))} KB`;
}

function ProcessingPanel(){
  const steps = ['Загружаем файл','Читаем строки сметы','Проверяем суммы','Ищем дубли','Запрашиваем AI-анализ'];
  return <div className="processingPanel"><h3>SmetaCheck анализирует файл</h3><div className="processingSteps">{steps.map((step,index)=><div className="processingStep" key={step}><div className="processingDot">{index+1}</div><span>{step}</span>{index<4 ? <i className="processingPulse"/> : <em>почти готово</em>}</div>)}</div></div>;
}

function SuccessPanel({result, analysis}){
  const source = analysis?.analysis_source === 'openai'
    ? `OpenAI · ${analysis.model || 'модель'}`
    : 'Автоматическая сводка по правилам';
  return <div className="successHero"><h2>Смета проверена</h2><p>Отчёт создан. Источник анализа: <b>{source}</b>.</p>{analysis?.warning && <p>{analysis.warning}</p>}<div className="successMetrics"><div><strong>{result.score}/100</strong><span>Оценка</span></div><div><strong>{result.items_count || 0}</strong><span>Строк</span></div><div><strong>{Number(result.total_amount || 0).toLocaleString('ru-RU')}</strong><span>Сумма</span></div></div><div className="buttonRow"><a className="btn" href={`/reports/${result.id}`}>Открыть отчёт</a><a className="btn secondary" href="/compare">Сравнить версии</a></div></div>;
}

export default function Upload(){
  const [file, setFile] = useState(null);
  const [status, setStatus] = useState('idle');
  const [message, setMessage] = useState('');
  const [result, setResult] = useState(null);
  const [analysis, setAnalysis] = useState(null);
  const [authorized, setAuthorized] = useState(false);
  const [checkingSession, setCheckingSession] = useState(true);

  useEffect(()=>{
    async function verifySession(){
      const token = window.localStorage.getItem('smetacheck_token');
      if(!token){
        setCheckingSession(false);
        return;
      }
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
        setMessage('Не удалось связаться с API. Проверьте, что backend и PostgreSQL запущены.');
      }finally{
        setCheckingSession(false);
      }
    }
    verifySession();
  }, []);

  async function submitUpload(){
    const token = window.localStorage.getItem('smetacheck_token');
    if(!token || !authorized){
      setMessage('Сначала войдите в аккаунт.');
      return;
    }
    if(!file){ setMessage('Сначала выберите файл сметы.'); return; }
    setStatus('uploading');
    setMessage('Загружаем и проверяем смету...');
    setResult(null);
    setAnalysis(null);

    const formData = new FormData();
    formData.append('file', file);

    try{
      const response = await fetch(`${API_BASE}/v1/estimates/upload`, {method:'POST', headers:{Authorization:`Bearer ${token}`}, body:formData});
      const data = await response.json();
      if(response.status === 401){
        window.localStorage.removeItem('smetacheck_token');
        setAuthorized(false);
        throw new Error('Сессия истекла. Войдите снова.');
      }
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить файл'); }
      setResult(data);
      setStatus('analyzing');
      setMessage('Файл сохранён. AI готовит структурированный отчёт...');

      const aiResponse = await fetch(`${API_BASE}/v1/ai/estimate-summary/${data.id}`, {headers:{Authorization:`Bearer ${token}`}});
      const aiData = await aiResponse.json();
      if(!aiResponse.ok){ throw new Error(aiData.error || 'Не удалось получить AI-анализ'); }
      setAnalysis(aiData);
      setStatus('done');
      setMessage(aiData.analysis_source === 'openai'
        ? 'Смета проверена, OpenAI-анализ сохранён в PostgreSQL.'
        : 'Смета проверена. Использована автоматическая сводка по правилам.');
    }catch(error){
      if(result){
        setStatus('done');
        setMessage('Смета сохранена, но AI-анализ временно недоступен. Откройте отчёт и повторите запрос.');
      }else{
        setStatus('error');
        setMessage(error.message || 'Не удалось загрузить файл');
      }
    }
  }

  function loadDemo(){
    const demo = new File(['Наименование,Ед,Количество,Цена,Сумма\nКирпич,шт,1000,12,12000\nЦемент,мешок,20,450,9000\nПесок,м3,5,1200,6000\nАрматура,кг,300,65,19500\n'], 'demo-smeta.csv', {type:'text/csv'});
    setFile(demo);
    setResult(null);
    setAnalysis(null);
    setStatus('idle');
    setMessage('Demo-смета готова. Нажмите «Начать проверку».');
  }

  const processing = status === 'uploading' || status === 'analyzing';

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact"><p className="eyebrow">Проверка сметы</p><h1>Загрузите смету и получите понятный отчёт для обсуждения.</h1><p>Все результаты сохраняются в PostgreSQL и доступны только владельцу аккаунта.</p></section>
      <section className="workspace">
        {checkingSession && <div className="card"><p>Проверяем сессию...</p></div>}
        {!checkingSession && !authorized && <div className="emptyState"><h2>Нужно войти в аккаунт</h2><p>{message || 'Загрузка смет доступна только зарегистрированным пользователям.'}</p><a className="btn" href="/login">Войти или зарегистрироваться</a></div>}
        {!checkingSession && authorized && <div className="twoColumns">
          <div className="uploadBox modernUploadShell">
            <div className="demoStrip"><div><b>Нет файла под рукой?</b><p>Запустите demo-анализ и сразу посмотрите результат.</p></div><button className="btn" type="button" onClick={loadDemo}>Попробовать demo-смету</button></div>
            <label className="modernUploadZone"><input className="modernUploadInput" type="file" accept=".xlsx,.xlsm,.csv,.pdf" onChange={(event)=>setFile(event.target.files?.[0] || null)} /><div className="modernUploadContent"><div className="modernUploadIcon">↑</div><span className="modernUploadHint">XLSX · XLSM · CSV · PDF</span><h2>{file ? 'Файл выбран' : 'Выберите смету для проверки'}</h2><p>{file ? 'Можно запускать анализ. Система прочитает строки и подготовит замечания.' : 'Нажмите на область или перетащите файл сюда.'}</p></div></label>
            {file && <div className="modernFilePill"><div><b>{file.name}</b><br/><span>{fileSizeLabel(file)}</span></div><span>Готов к проверке</span></div>}
            <div className="modernMiniGrid"><span>Проверка сумм</span><span>Поиск дублей</span><span>AI-анализ</span><span>Отчёт и график</span></div>
            <button className="btn" type="button" onClick={submitUpload} disabled={processing}>{processing ? 'Анализируем...' : 'Начать проверку'}</button>
            {processing && <ProcessingPanel/>}
            {message && <p className={`statusText ${status}`}>{message}</p>}
            {result && !processing && <SuccessPanel result={result} analysis={analysis}/>}        
          </div>
          <div className="card checklistCard"><h2>Что получает клиент</h2><ul><li>Смета привязана к аккаунту</li><li>Позиции и замечания сохраняются в PostgreSQL</li><li>AI получает только распознанные строки и findings</li><li>График строится по проверенным данным</li><li>При ошибке AI используется безопасный fallback</li></ul><a className="btn secondary" href="/dashboard">Открыть кабинет</a></div>
        </div>}
      </section>
      <Footer/>
    </main>
  );
}
