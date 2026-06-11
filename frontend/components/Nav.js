import {useEffect,useState} from 'react';
import {currentUser,logout} from '../lib/api';

export default function Nav(){
  const [user,setUser]=useState(null);
  const [ready,setReady]=useState(false);

  useEffect(()=>{
    currentUser().then(value=>{setUser(value);setReady(true);}).catch(()=>setReady(true));
  },[]);

  async function signOut(){
    await logout();
    setUser(null);
    window.location.replace('/login');
  }

  return <nav className="nav" aria-label="Основная навигация">
    <a className="brand" href="/"><span className="brandMark">S</span><span>SmetaCheck KG</span></a>
    <div className="navLinks">
      {!user&&<a href="/demo">Demo</a>}
      {!user&&<a href="/sample-report">Пример отчёта</a>}
      {!user&&<a href="/how-it-works">Как работает</a>}
      {user&&<a href="/dashboard">Кабинет</a>}
      {user&&<a href="/upload">Новая проверка</a>}
      {user&&<a href="/reports">Отчёты</a>}
      {user&&<a href="/compare">Сравнение</a>}
      <a href="/pricing">Тарифы</a>
      <a href="/support">Связаться</a>
    </div>
    {!ready?<span className="navAction">...</span>:user?<button className="navAction" type="button" onClick={signOut}>Выйти</button>:<a className="navAction" href="/login">Войти</a>}
  </nav>;
}
