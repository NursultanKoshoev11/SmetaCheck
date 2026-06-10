import {useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

export default function Login(){
  const [mode, setMode] = useState('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [status, setStatus] = useState('idle');
  const [message, setMessage] = useState('');

  async function submitAuth(){
    setStatus('loading');
    setMessage(mode === 'login' ? 'Входим в аккаунт...' : 'Создаём аккаунт...');
    try{
      const endpoint = mode === 'login' ? '/v1/auth/login' : '/v1/auth/register';
      const response = await fetch(`${API_BASE}${endpoint}`, {
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body: JSON.stringify({email, password, full_name: fullName})
      });
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'Ошибка авторизации'); }
      window.localStorage.setItem('smetacheck_token', data.token);
      window.localStorage.setItem('smetacheck_user_email', data.user.email);
      setStatus('done');
      setMessage('Готово. Сейчас откроется кабинет.');
      setTimeout(()=>{ window.location.href = '/dashboard'; }, 700);
    }catch(error){
      setStatus('error');
      setMessage(error.message || 'Ошибка авторизации');
    }
  }

  return (
    <main className="page">
      <Nav/>
      <section className="authShell">
        <div>
          <p className="eyebrow">Аккаунт</p>
          <h1>{mode === 'login' ? 'Войдите в кабинет проверки строительных смет.' : 'Создайте аккаунт SmetaCheck.'}</h1>
          <p>Аккаунт нужен, чтобы хранить сметы, отчёты, сравнения и историю работы в одном месте.</p>
        </div>
        <form className="authCard" onSubmit={(event)=>{event.preventDefault(); submitAuth();}}>
          <div className="buttonRow"><button className={mode==='login' ? 'btn' : 'btn secondary'} type="button" onClick={()=>setMode('login')}>Вход</button><button className={mode==='register' ? 'btn' : 'btn secondary'} type="button" onClick={()=>setMode('register')}>Регистрация</button></div>
          {mode === 'register' && <label>Имя<input value={fullName} onChange={(event)=>setFullName(event.target.value)} placeholder="Ваше имя" type="text" /></label>}
          <label>Email<input value={email} onChange={(event)=>setEmail(event.target.value)} placeholder="name@company.com" type="email" /></label>
          <label>Пароль<input value={password} onChange={(event)=>setPassword(event.target.value)} placeholder="Минимум 8 символов" type="password" /></label>
          <button className="btn" type="submit" disabled={status==='loading'}>{status==='loading' ? 'Подождите...' : (mode === 'login' ? 'Войти' : 'Создать аккаунт')}</button>
          {message && <p className={`statusText ${status}`}>{message}</p>}
          <button type="button" className="btn secondary" onClick={()=>setMode(mode === 'login' ? 'register' : 'login')}>{mode === 'login' ? 'У меня нет аккаунта' : 'У меня уже есть аккаунт'}</button>
        </form>
      </section>
      <Footer/>
    </main>
  )
}
