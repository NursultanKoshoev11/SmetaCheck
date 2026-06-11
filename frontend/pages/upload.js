import {useEffect,useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';
import BatchUploadWorkspace from '../components/BatchUploadWorkspace';
import {currentUser} from '../lib/api';

export default function Upload(){
  const [authorized,setAuthorized]=useState(false);
  const [checkingSession,setCheckingSession]=useState(true);
  const [message,setMessage]=useState('');

  useEffect(()=>{
    currentUser().then(user=>{
      setAuthorized(Boolean(user));
      if(!user)setMessage('Сессия отсутствует или истекла. Войдите снова.');
    }).catch(()=>setMessage('Не удалось связаться с API.')).finally(()=>setCheckingSession(false));
  },[]);

  return <main className="page">
    <Nav/>
    <section className="pageHero compact">
      <p className="eyebrow">Пакетная проверка смет</p>
      <h1>Загрузите одну или несколько смет и получите единый отчёт.</h1>
      <p>Backend проверяет расчёты и структуру, AI выполняет независимый анализ, после чего SmetaCheck сопоставляет оба результата.</p>
    </section>
    <section className="workspace">
      {checkingSession&&<div className="card"><p>Проверяем защищённую сессию...</p></div>}
      {!checkingSession&&!authorized&&<div className="emptyState"><h2>Нужно войти в аккаунт</h2><p>{message||'Анализ доступен только владельцу аккаунта.'}</p><a className="btn" href="/login">Войти или зарегистрироваться</a></div>}
      {!checkingSession&&authorized&&<BatchUploadWorkspace/>}
    </section>
    <Footer/>
  </main>;
}
