import {useEffect,useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';
import {apiJSON} from '../lib/api';

export default function ResetPassword(){
  const [token,setToken]=useState('');
  const [password,setPassword]=useState('');
  const [confirmPassword,setConfirmPassword]=useState('');
  const [status,setStatus]=useState('idle');
  const [message,setMessage]=useState('');

  useEffect(()=>{
    const params=new URLSearchParams(window.location.search);
    const value=(params.get('token')||'').trim();
    setToken(value);
    if(!value){
      setStatus('error');
      setMessage('Ссылка восстановления неполная. Запросите новую ссылку на странице входа.');
    }
  },[]);

  async function submit(event){
    event.preventDefault();
    if(!token){setStatus('error');setMessage('Токен восстановления отсутствует.');return;}
    if(password.length<12){setStatus('error');setMessage('Пароль должен содержать минимум 12 символов.');return;}
    if(password!==confirmPassword){setStatus('error');setMessage('Пароли не совпадают.');return;}

    setStatus('loading');
    setMessage('Устанавливаем новый пароль...');
    try{
      const {response,data}=await apiJSON('/v1/auth/password/reset',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({token,password})
      });
      if(!response.ok){
        throw new Error(data.error||'Ссылка восстановления недействительна или истекла.');
      }
      setStatus('done');
      setMessage('Пароль изменён. Все прежние сессии отозваны. Перенаправляем ко входу...');
      window.setTimeout(()=>window.location.replace('/login?password_reset=1'),900);
    }catch(error){
      setStatus('error');
      setMessage(error.message||'Не удалось изменить пароль.');
    }
  }

  return <main className="page">
    <Nav/>
    <section className="authShell">
      <div>
        <p className="eyebrow">Восстановление доступа</p>
        <h1>Установите новый пароль.</h1>
        <p>Новый пароль должен содержать минимум 12 символов. После изменения все старые сессии будут отозваны.</p>
      </div>
      <form className="authCard" onSubmit={submit}>
        <label>Новый пароль<input value={password} onChange={(event)=>setPassword(event.target.value)} type="password" minLength={12} maxLength={128} autoComplete="new-password" required disabled={!token}/></label>
        <label>Повторите пароль<input value={confirmPassword} onChange={(event)=>setConfirmPassword(event.target.value)} type="password" minLength={12} maxLength={128} autoComplete="new-password" required disabled={!token}/></label>
        <button className="btn" type="submit" disabled={status==='loading'||!token}>{status==='loading'?'Сохраняем...':'Изменить пароль'}</button>
        {message&&<p className={`statusText ${status}`}>{message}</p>}
        <a className="btn secondary" href="/login">Вернуться ко входу</a>
      </form>
    </section>
    <Footer/>
  </main>;
}
