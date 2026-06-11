import {useEffect, useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';
import BatchUploadWorkspace from '../components/BatchUploadWorkspace';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export default function Upload(){
  const [authorized,setAuthorized] = useState(false);
  const [checkingSession,setCheckingSession] = useState(true);
  const [message,setMessage] = useState('');

  useEffect(()=>{
    async function verifySession(){
      const token = window.localStorage.getItem('smetacheck_token');
      if(!token){setCheckingSession(false);return;}
      try{
        const response = await fetch(`${API_BASE}/v1/auth/me`,{headers:{Authorization:`Bearer ${token}`}});
        if(!response.ok){
          window.localStorage.removeItem('smetacheck_token');
          window.localStorage.removeItem('smetacheck_user_email');
          setMessage('Сессия истекла. Войдите снова.');
          return;
        }
        setAuthorized(true);
      }catch{
        setMessage('Не удалось связаться с API.');
      }finally{
        setCheckingSession(false);
      }
    }
    verifySession();
  },[]);

  return <main className="page">
    <Nav/>
    <section className="pageHero compact">
      <p className="eyebrow">Пакетная проверка смет</p>
      <h1>Загрузите одну или несколько смет и получите единый отчёт.</h1>
      <p>Backend проверяет расчёты и структуру, AI выполняет независимый анализ, после чего SmetaCheck сопоставляет оба результата.</p>
    </section>
    <section className="workspace">
      {checkingSession && <div className="card"><p>Проверяем сессию...</p></div>}
      {!checkingSession && !authorized && <div className="emptyState"><h2>Нужно войти в аккаунт</h2><p>{message || 'Анализ доступен только владельцу аккаунта.'}</p><a className="btn" href="/login">Войти или зарегистрироваться</a></div>}
      {!checkingSession && authorized && <BatchUploadWorkspace/>}
    </section>
    <Footer/>
  </main>;
}
