import {useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export default function Login(){
  const [mode,setMode]=useState('login');
  const [email,setEmail]=useState('');
  const [password,setPassword]=useState('');
  const [fullName,setFullName]=useState('');
  const [status,setStatus]=useState('idle');
  const [message,setMessage]=useState('');

  function changeMode(next){setMode(next);setStatus('idle');setMessage('');}

  async function submitAuth(){
    const cleanEmail=email.trim().toLowerCase();
    if(!cleanEmail.includes('@')){setStatus('error');setMessage('Введите корректный email.');return;}
    if(password.length<8){setStatus('error');setMessage('Пароль должен содержать минимум 8 символов.');return;}
    if(mode==='register'&&fullName.trim().length<2){setStatus('error');setMessage('Введите ваше имя.');return;}

    setStatus('loading');
    setMessage(mode==='login'?'Проверяем аккаунт...':'Создаём аккаунт в PostgreSQL...');
    try{
      const endpoint=mode==='login'?'/v1/auth/login':'/v1/auth/register';
      const response=await fetch(`${API_BASE}${endpoint}`,{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({email:cleanEmail,password,full_name:fullName.trim()})
      });
      const text=await response.text();
      let data={};
      try{data=text?JSON.parse(text):{};}catch{data={error:text};}

      if(!response.ok){
        if(response.status===409)throw new Error('Аккаунт с таким email уже существует. Используйте вход.');
        if(response.status===401)throw new Error('Неверный email или пароль.');
        if(response.status===503)throw new Error('PostgreSQL недоступен. Проверьте базу данных и API.');
        throw new Error(data.error||'Не удалось выполнить авторизацию.');
      }
      if(!data.token||!data.user?.email)throw new Error('API вернул неполный ответ авторизации.');

      const check=await fetch(`${API_BASE}/v1/auth/me`,{headers:{Authorization:`Bearer ${data.token}`}});
      if(!check.ok)throw new Error('Аккаунт создан, но JWT-сессия не прошла проверку.');

      window.localStorage.setItem('smetacheck_token',data.token);
      window.localStorage.setItem('smetacheck_user_email',data.user.email);
      setStatus('done');
      setMessage(mode==='register'?'Аккаунт создан в PostgreSQL.':'Вход выполнен.');
      window.setTimeout(()=>window.location.replace('/dashboard'),500);
    }catch(error){
      setStatus('error');
      setMessage(error instanceof TypeError?`API недоступен по адресу ${API_BASE}`:(error.message||'Ошибка авторизации'));
    }
  }

  return <main className="page"><Nav/><section className="authShell"><div><p className="eyebrow">Аккаунт</p><h1>{mode==='login'?'Войдите в кабинет.':'Создайте аккаунт SmetaCheck.'}</h1><p>Пользователь и его данные сохраняются в PostgreSQL. Без входа кабинет и отчёты недоступны.</p></div><form className="authCard" onSubmit={(event)=>{event.preventDefault();submitAuth();}}><div className="buttonRow"><button className={mode==='login'?'btn':'btn secondary'} type="button" onClick={()=>changeMode('login')}>Вход</button><button className={mode==='register'?'btn':'btn secondary'} type="button" onClick={()=>changeMode('register')}>Регистрация</button></div>{mode==='register'&&<label>Имя<input value={fullName} onChange={(event)=>setFullName(event.target.value)} placeholder="Ваше имя" required/></label>}<label>Email<input value={email} onChange={(event)=>setEmail(event.target.value)} type="email" placeholder="name@company.com" required/></label><label>Пароль<input value={password} onChange={(event)=>setPassword(event.target.value)} type="password" minLength={8} placeholder="Минимум 8 символов" required/></label><button className="btn" type="submit" disabled={status==='loading'}>{status==='loading'?'Подождите...':(mode==='login'?'Войти':'Создать аккаунт')}</button>{message&&<p className={`statusText ${status}`}>{message}</p>}<button type="button" className="btn secondary" onClick={()=>changeMode(mode==='login'?'register':'login')}>{mode==='login'?'У меня нет аккаунта':'У меня уже есть аккаунт'}</button></form></section><Footer/></main>;
}
