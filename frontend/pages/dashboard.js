import {useEffect, useMemo, useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function statusLabel(status){
  if(status === 'review_required') return 'нужна проверка';
  if(status === 'ready') return 'готово';
  return status || 'неизвестно';
}

export default function Dashboard(){
  const [estimates, setEstimates] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [authorized, setAuthorized] = useState(false);
  const [email, setEmail] = useState('');

  async function loadDashboard(){
    const token = window.localStorage.getItem('smetacheck_token');
    if(!token){
      setAuthorized(false);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError('');
    try{
      const meResponse = await fetch(`${API_BASE}/v1/auth/me`, {headers:{Authorization:`Bearer ${token}`}});
      const meData = await meResponse.json();
      if(!meResponse.ok){
        window.localStorage.removeItem('smetacheck_token');
        window.localStorage.removeItem('smetacheck_user_email');
        throw new Error(meData.error || 'Сессия истекла. Войдите снова.');
      }
      setAuthorized(true);
      setEmail(meData.user?.email || '');

      const response = await fetch(`${API_BASE}/v1/estimates`, {headers:{Authorization:`Bearer ${token}`}});
      const data = await response.json();
      if(!response.ok){ throw new Error(data.error || 'Не удалось загрузить кабинет'); }
      setEstimates(data.estimates || []);
    }catch(err){
      setAuthorized(false);
      setError(err.message || 'Не удалось загрузить кабинет');
    }finally{
      setLoading(false);
    }
  }

  function logout(){
    window.localStorage.removeItem('smetacheck_token');
    window.localStorage.removeItem('smetacheck_user_email');
    window.location.href = '/login';
  }

  useEffect(()=>{ loadDashboard(); }, []);

  const stats = useMemo(()=>{
    const total = estimates.length;
    const review = estimates.filter(item => item.status === 'review_required').length;
    const avg = total ? Math.round(estimates.reduce((sum,item)=>sum+(item.score || 0),0)/total) : 0;
    const amount = estimates.reduce((sum,item)=>sum+Number(item.total_amount || 0),0);
    return [[String(total),'Проверок'], [String(review),'С рисками'], [String(avg),'Средняя оценка'], [amount.toLocaleString('ru-RU'),'Сумма по сметам']];
  }, [estimates]);

  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Кабинет</p>
        <h1>{authorized ? 'Контролируйте сметы, риски и суммы в одном месте.' : 'Кабинет доступен только после входа.'}</h1>
        <p>{authorized ? `Вы вошли как ${email}. Здесь показываются только ваши сметы и отчёты.` : 'Создайте аккаунт или войдите, чтобы данные сохранялись в PostgreSQL и были доступны только вам.'}</p>
      </section>
      <section className="workspace">
        {loading && <div className="card"><p>Проверяем сессию и загружаем кабинет...</p></div>}
        {!loading && !authorized && <div className="emptyState"><h2>Нужно войти в аккаунт</h2><p>{error || 'Без авторизации кабинет, загрузка и отчёты недоступны.'}</p><div className="buttonRow"><a className="btn" href="/login">Войти или зарегистрироваться</a><a className="btn secondary" href="/">На главную</a></div></div>}
        {!loading && authorized && <>
          <div className="buttonRow"><button className="btn secondary" type="button" onClick={loadDashboard}>Обновить</button><a className="btn" href="/upload">Загрузить смету</a><button className="btn secondary" type="button" onClick={logout}>Выйти</button></div>
          {error && <div className="card"><h2>Ошибка API</h2><p>{error}</p></div>}
          {!error && <div className="grid statsGrid">{stats.map(([value,label]) => <article className="statCard" key={label}><strong>{value}</strong><span>{label}</span></article>)}</div>}
          {!error && <div className="twoColumns">
            <div className="card"><h2>Последние проверки</h2><div className="timeline">{estimates.slice(0,5).map((item,i) => <p key={item.id}><b>{String(i+1).padStart(2,'0')}</b><span>{item.file_name} · {statusLabel(item.status)} · {item.items_count || 0} строк · оценка {item.score}/100</span></p>)}</div>{estimates.length === 0 && <p>Пока нет смет. Загрузите первую Excel или CSV смету.</p>}</div>
            <div className="card"><h2>Что уже проверяется</h2><ul><li>Пустые названия, количества, цены и суммы</li><li>Расхождение количество × цена и итоговой суммы</li><li>Возможные дубли позиций</li><li>Крупные суммы, требующие ручной проверки</li></ul><div className="buttonRow"><a className="btn" href="/upload">Загрузить</a><a className="btn secondary" href="/reports">Открыть отчёты</a></div></div>
          </div>}
        </>}
      </section>
      <Footer/>
    </main>
  )
}
