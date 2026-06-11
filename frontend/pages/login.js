import {useEffect,useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';
import {API_BASE,apiJSON,currentUser} from '../lib/api';

export default function Login(){
  const [mode,setMode]=useState('login');
  const [email,setEmail]=useState('');
  const [password,setPassword]=useState('');
  const [fullName,setFullName]=useState('');
  const [status,setStatus]=useState('idle');
  const [message,setMessage]=useState('');
  const [verificationRequired,setVerificationRequired]=useState(false);

  useEffect(()=>{
    currentUser().then(user=>{if(user)window.location.replace('/dashboard');});
    const params=new URLSearchParams(window.location.search);
    if(params.get('verified')==='1'){
      setStatus('done');
      setMessage('Email подтверждён. Теперь войдите в аккаунт.');
    }
    if(params.get('oauth_error')){
      setStatus('error');
      setMessage(params.get('oauth_error'));
    }
  },[]);

  function changeMode(next){
    setMode(next);
    setStatus('idle');
    setMessage('');
    setVerificationRequired(false);
  }

  async function submitAuth(){
    const cleanEmail=email.trim().toLowerCase();
    if(!cleanEmail.includes('@')){setStatus('error');setMessage('Введите корректный email.');return;}
    if(mode!=='forgot'&&password.length<12){setStatus('error');setMessage('Пароль должен содержать минимум 12 символов.');return;}
    if(mode==='register'&&fullName.trim().length<2){setStatus('error');setMessage('Введите ваше имя.');return;}

    setStatus('loading');
    setMessage(mode==='forgot'?'Отправляем ссылку восстановления...':mode==='login'?'Проверяем данные...':'Создаём защищённый аккаунт...');
    try{
      if(mode==='forgot'){
        const {response}=await apiJSON('/v1/auth/password/forgot',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:cleanEmail})});
        if(!response.ok)throw new Error('Не удалось отправить письмо восстановления.');
        setStatus('done');
        setMessage('Если аккаунт существует, ссылка восстановления отправлена на email.');
        return;
      }

      const endpoint=mode==='login'?'/v1/auth/login':'/v1/auth/register';
      const {response,data}=await apiJSON(endpoint,{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({email:cleanEmail,password,full_name:fullName.trim()})
      });

      if(!response.ok){
        if(response.status===409)throw new Error('Аккаунт с таким email уже существует. Используйте вход.');
        if(response.status===401)throw new Error('Неверный email или пароль.');
        if(response.status===403){setVerificationRequired(true);throw new Error('Сначала подтвердите email по ссылке из письма.');}
        if(response.status===429)throw new Error('Слишком много попыток. Попробуйте позже.');
        if(response.status===503)throw new Error('Сервис авторизации временно недоступен.');
        throw new Error(data.error||'Не удалось выполнить авторизацию.');
      }

      if(mode==='register'){
        setVerificationRequired(true);
        setStatus('done');
        setMessage('Аккаунт создан. Мы отправили письмо для подтверждения email.');
        return;
      }

      setStatus('done');
      setMessage('Вход выполнен. Открываем кабинет...');
      window.setTimeout(()=>window.location.replace('/dashboard'),400);
    }catch(error){
      setStatus('error');
      setMessage(error instanceof TypeError?`API недоступен по адресу ${API_BASE}`:(error.message||'Ошибка авторизации'));
    }
  }

  async function resendVerification(){
    const cleanEmail=email.trim().toLowerCase();
    if(!cleanEmail.includes('@')){setStatus('error');setMessage('Введите email аккаунта.');return;}
    setStatus('loading');
    const {response}=await apiJSON('/v1/auth/email/resend',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:cleanEmail})});
    setStatus(response.ok?'done':'error');
    setMessage(response.ok?'Если аккаунт ожидает подтверждения, новое письмо отправлено.':'Не удалось повторно отправить письмо.');
  }

  function startProvider(provider){
    window.location.href=`${API_BASE}/v1/auth/${provider}?return_to=/dashboard`;
  }

  return <main className="page">
    <Nav/>
    <section className="authShell">
      <div>
        <p className="eyebrow">Безопасный аккаунт</p>
        <h1>{mode==='register'?'Создайте аккаунт SmetaCheck.':mode==='forgot'?'Восстановите доступ.':'Войдите в кабинет.'}</h1>
        <p>Вход по email, Google или Telegram. Сессия хранится в защищённых HttpOnly cookies, а не в localStorage.</p>
      </div>
      <form className="authCard" onSubmit={(event)=>{event.preventDefault();submitAuth();}}>
        <div className="socialAuthGrid">
          <button className="btn secondary" type="button" onClick={()=>startProvider('google')}>Продолжить с Google</button>
          <button className="btn secondary" type="button" onClick={()=>startProvider('telegram')}>Продолжить с Telegram</button>
        </div>
        <div className="authDivider"><span>или по email</span></div>
        {mode!=='forgot'&&<div className="buttonRow"><button className={mode==='login'?'btn':'btn secondary'} type="button" onClick={()=>changeMode('login')}>Вход</button><button className={mode==='register'?'btn':'btn secondary'} type="button" onClick={()=>changeMode('register')}>Регистрация</button></div>}
        {mode==='register'&&<label>Имя<input value={fullName} onChange={(event)=>setFullName(event.target.value)} placeholder="Ваше имя" autoComplete="name" required/></label>}
        <label>Email<input value={email} onChange={(event)=>setEmail(event.target.value)} type="email" placeholder="name@company.com" autoComplete="email" required/></label>
        {mode!=='forgot'&&<label>Пароль<input value={password} onChange={(event)=>setPassword(event.target.value)} type="password" minLength={12} maxLength={128} placeholder="Минимум 12 символов" autoComplete={mode==='login'?'current-password':'new-password'} required/></label>}
        <button className="btn" type="submit" disabled={status==='loading'}>{status==='loading'?'Подождите...':mode==='forgot'?'Отправить ссылку':mode==='login'?'Войти':'Создать аккаунт'}</button>
        {message&&<p className={`statusText ${status}`}>{message}</p>}
        {verificationRequired&&<button type="button" className="btn secondary" onClick={resendVerification}>Отправить письмо ещё раз</button>}
        {mode==='login'&&<button type="button" className="textAction" onClick={()=>changeMode('forgot')}>Забыли пароль?</button>}
        {mode==='forgot'&&<button type="button" className="btn secondary" onClick={()=>changeMode('login')}>Вернуться ко входу</button>}
      </form>
    </section>
    <Footer/>
  </main>;
}
