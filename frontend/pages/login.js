import {useEffect,useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';
import {API_BASE,apiJSON,currentUser} from '../lib/api';

const defaultProviders={email_login:true,email_registration:false,password_reset:false,google:false,telegram:false};
const providerNames={google:'Google',telegram:'Telegram'};
const oauthErrorMessages={
  cancelled:'Вход был отменён. Попробуйте ещё раз.',
  invalid_callback:'Провайдер вернул неполный ответ. Начните вход заново.',
  invalid_state:'Сессия входа истекла или открыта в другом браузере. Начните вход заново.',
  provider_unavailable:'Провайдер входа временно недоступен. Попробуйте позже.',
  service_unavailable:'Сервис авторизации временно недоступен. Попробуйте позже.',
  exchange_failed:'Не удалось завершить безопасный обмен с провайдером. Попробуйте снова.',
  invalid_token:'Провайдер вернул недействительные данные авторизации.',
  email_unverified:'Google-аккаунт должен иметь подтверждённый email.',
  account_link_failed:'Не удалось создать или связать аккаунт. Обратитесь в поддержку.',
  session_failed:'Аккаунт подтверждён, но сессию создать не удалось. Попробуйте снова.',
  internal_error:'Внутренняя ошибка авторизации. Попробуйте позже.'
};

function safeLocalReturnTo(value){
  if(!value||value.length>512||!value.startsWith('/')||value.startsWith('//')||value.includes('\\'))return '/dashboard';
  return value;
}

export default function Login(){
  const [mode,setMode]=useState('login');
  const [email,setEmail]=useState('');
  const [password,setPassword]=useState('');
  const [fullName,setFullName]=useState('');
  const [status,setStatus]=useState('idle');
  const [message,setMessage]=useState('');
  const [verificationRequired,setVerificationRequired]=useState(false);
  const [providers,setProviders]=useState(defaultProviders);
  const [providersReady,setProvidersReady]=useState(false);
  const [providerLoading,setProviderLoading]=useState('');

  useEffect(()=>{
    const params=new URLSearchParams(window.location.search);
    const oauthError=params.get('oauth_error');
    const oauthProvider=params.get('provider');
    if(oauthError){
      const providerLabel=providerNames[oauthProvider]||'Провайдер';
      setStatus('error');
      setMessage(`${providerLabel}: ${oauthErrorMessages[oauthError]||'Не удалось выполнить вход. Попробуйте ещё раз.'}`);
    }else if(params.get('verified')==='1'){
      setStatus('done');
      setMessage('Email подтверждён. Теперь войдите в аккаунт.');
    }
    if(oauthError||params.has('provider')||params.has('verified')){
      params.delete('oauth_error');
      params.delete('provider');
      params.delete('verified');
      const cleanQuery=params.toString();
      window.history.replaceState({},'',`${window.location.pathname}${cleanQuery?`?${cleanQuery}`:''}`);
    }

    currentUser().then(user=>{if(user)window.location.replace('/dashboard');}).catch(()=>{});
    apiJSON('/v1/auth/providers').then(({response,data})=>{
      if(response.ok&&data.providers)setProviders({...defaultProviders,...data.providers});
    }).catch(()=>{}).finally(()=>setProvidersReady(true));
  },[]);

  function changeMode(next){
    if(next==='register'&&!providers.email_registration){
      setStatus('error');
      setMessage('Регистрация по email временно недоступна: почтовый сервис не настроен. Используйте Google или Telegram.');
      return;
    }
    if(next==='forgot'&&!providers.password_reset){
      setStatus('error');
      setMessage('Восстановление пароля временно недоступно: почтовый сервис не настроен.');
      return;
    }
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
    if(mode==='register'&&!providers.email_registration){setStatus('error');setMessage('Регистрация по email сейчас недоступна.');return;}
    if(mode==='forgot'&&!providers.password_reset){setStatus('error');setMessage('Восстановление пароля сейчас недоступно.');return;}

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
    if(!providers.email_registration){setStatus('error');setMessage('Почтовый сервис сейчас недоступен.');return;}
    setStatus('loading');
    const {response}=await apiJSON('/v1/auth/email/resend',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:cleanEmail})});
    setStatus(response.ok?'done':'error');
    setMessage(response.ok?'Если аккаунт ожидает подтверждения, новое письмо отправлено.':'Не удалось повторно отправить письмо.');
  }

  function startProvider(provider){
    if(providerLoading)return;
    if(!providers[provider]){
      setStatus('error');
      setMessage(`${providerNames[provider]||'Провайдер'} сейчас не настроен.`);
      return;
    }
    const params=new URLSearchParams(window.location.search);
    const returnTo=safeLocalReturnTo(params.get('return_to'));
    setProviderLoading(provider);
    setStatus('loading');
    setMessage(`Переходим в ${providerNames[provider]} для безопасного входа...`);
    window.location.assign(`${API_BASE}/v1/auth/${provider}?return_to=${encodeURIComponent(returnTo)}`);
  }

  const hasSocial=providers.google||providers.telegram;
  const busy=status==='loading'||Boolean(providerLoading);

  return <main className="page">
    <Nav/>
    <section className="authShell">
      <div>
        <p className="eyebrow">Безопасный аккаунт</p>
        <h1>{mode==='register'?'Создайте аккаунт SmetaCheck.':mode==='forgot'?'Восстановите доступ.':'Войдите в кабинет.'}</h1>
        <p>Вход по email, Google или Telegram. При первом входе через Google или Telegram аккаунт создаётся автоматически.</p>
      </div>
      <form className="authCard" aria-busy={busy} onSubmit={(event)=>{event.preventDefault();submitAuth();}}>
        {!providersReady&&<p className="statusText loading">Проверяем доступные способы входа...</p>}
        {providersReady&&hasSocial&&mode!=='forgot'&&<><div className="socialAuthGrid">
          {providers.google&&<button className="btn secondary" type="button" disabled={busy} onClick={()=>startProvider('google')}>{providerLoading==='google'?'Открываем Google...':'Продолжить с Google'}</button>}
          {providers.telegram&&<button className="btn secondary" type="button" disabled={busy} onClick={()=>startProvider('telegram')}>{providerLoading==='telegram'?'Открываем Telegram...':'Продолжить с Telegram'}</button>}
        </div><div className="authDivider"><span>или по email</span></div></>}
        {mode!=='forgot'&&<div className="buttonRow"><button className={mode==='login'?'btn':'btn secondary'} type="button" disabled={busy} onClick={()=>changeMode('login')}>Вход</button>{providers.email_registration&&<button className={mode==='register'?'btn':'btn secondary'} type="button" disabled={busy} onClick={()=>changeMode('register')}>Регистрация</button>}</div>}
        {mode==='register'&&<label>Имя<input value={fullName} onChange={(event)=>setFullName(event.target.value)} placeholder="Ваше имя" autoComplete="name" required/></label>}
        <label>Email<input value={email} onChange={(event)=>setEmail(event.target.value)} type="email" placeholder="name@company.com" autoComplete="email" required/></label>
        {mode!=='forgot'&&<label>Пароль<input value={password} onChange={(event)=>setPassword(event.target.value)} type="password" minLength={12} maxLength={128} placeholder="Минимум 12 символов" autoComplete={mode==='login'?'current-password':'new-password'} required/></label>}
        <button className="btn" type="submit" disabled={busy}>{busy?'Подождите...':mode==='forgot'?'Отправить ссылку':mode==='login'?'Войти':'Создать аккаунт'}</button>
        {message&&<p className={`statusText ${status}`}>{message}</p>}
        {verificationRequired&&providers.email_registration&&<button type="button" className="btn secondary" disabled={busy} onClick={resendVerification}>Отправить письмо ещё раз</button>}
        {mode==='login'&&providers.password_reset&&<button type="button" className="textAction" disabled={busy} onClick={()=>changeMode('forgot')}>Забыли пароль?</button>}
        {mode==='forgot'&&<button type="button" className="btn secondary" disabled={busy} onClick={()=>changeMode('login')}>Вернуться ко входу</button>}
      </form>
    </section>
    <Footer/>
  </main>;
}
