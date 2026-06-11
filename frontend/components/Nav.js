import {useEffect, useState} from 'react';

export default function Nav(){
  const [signedIn, setSignedIn] = useState(false);

  useEffect(()=>{
    setSignedIn(Boolean(window.localStorage.getItem('smetacheck_token')));
  }, []);

  function logout(){
    window.localStorage.removeItem('smetacheck_token');
    window.localStorage.removeItem('smetacheck_user_email');
    window.location.href = '/login';
  }

  return (
    <nav className="nav">
      <a className="brand" href="/">
        <span className="brandMark">S</span>
        <span>SmetaCheck KG</span>
      </a>
      <div className="navLinks">
        <a href="/upload">Проверить</a>
        <a href="/how-it-works">Как работает</a>
        {signedIn && <a href="/dashboard">Кабинет</a>}
        {signedIn && <a href="/reports">Отчёты</a>}
        {signedIn && <a href="/compare">Сравнение</a>}
        <a href="/pricing">Тарифы</a>
        <a href="/faq">FAQ</a>
      </div>
      {signedIn ? <button className="navAction" type="button" onClick={logout}>Выйти</button> : <a className="navAction" href="/login">Войти</a>}
    </nav>
  )
}
