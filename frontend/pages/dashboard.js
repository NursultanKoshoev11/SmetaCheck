import {useEffect,useMemo,useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';
import {apiJSON,currentUser,logout} from '../lib/api';

function statusLabel(status){if(status==='review_required')return 'нужна проверка';if(status==='ready')return 'готово';return status||'неизвестно';}

export default function Dashboard(){
  const [estimates,setEstimates]=useState([]);
  const [loading,setLoading]=useState(true);
  const [error,setError]=useState('');
  const [authorized,setAuthorized]=useState(false);
  const [email,setEmail]=useState('');

  async function loadDashboard(){
    setLoading(true);setError('');
    try{
      const user=await currentUser();
      if(!user)throw new Error('Сессия истекла. Войдите снова.');
      setAuthorized(true);setEmail(user.email||'');
      const {response,data}=await apiJSON('/v1/estimates');
      if(!response.ok)throw new Error(data.error||'Не удалось загрузить кабинет');
      setEstimates(data.estimates||[]);
    }catch(err){setAuthorized(false);setError(err.message||'Не удалось загрузить кабинет');}
    finally{setLoading(false);}
  }

  async function signOut(){await logout();window.location.replace('/login');}
  useEffect(()=>{loadDashboard();},[]);

  const stats=useMemo(()=>{const total=estimates.length;const review=estimates.filter(item=>item.status==='review_required').length;const avg=total?Math.round(estimates.reduce((sum,item)=>sum+(item.score||0),0)/total):0;const amount=estimates.reduce((sum,item)=>sum+Number(item.total_amount||0),0);return [[String(total),'Проверок'],[String(review),'С рисками'],[String(avg),'Средняя оценка'],[amount.toLocaleString('ru-RU'),'Сумма по сметам']];},[estimates]);

  return <main className="page"><Nav/>
    <section className="pageHero compact"><p className="eyebrow">Кабинет</p><h1>{authorized?'Контролируйте сметы, риски и суммы в одном месте.':'Кабинет доступен только после входа.'}</h1><p>{authorized?`Вы вошли как ${email}. Здесь показываются только ваши сметы и отчёты.`:'Создайте аккаунт или войдите, чтобы данные были доступны только вам.'}</p></section>
    <section className="workspace">
      {loading&&<div className="card"><p>Проверяем сессию и загружаем кабинет...</p></div>}
      {!loading&&!authorized&&<div className="emptyState"><h2>Нужно войти в аккаунт</h2><p>{error||'Без авторизации кабинет, загрузка и отчёты недоступны.'}</p><div className="buttonRow"><a className="btn" href="/login">Войти или зарегистрироваться</a><a className="btn secondary" href="/">На главную</a></div></div>}
      {!loading&&authorized&&<><div className="buttonRow"><button className="btn secondary" type="button" onClick={loadDashboard}>Обновить</button><a className="btn" href="/upload">Загрузить смету</a><button className="btn secondary" type="button" onClick={signOut}>Выйти</button></div>{error&&<div className="card"><h2>Ошибка API</h2><p>{error}</p></div>}{!error&&<div className="grid statsGrid">{stats.map(([value,label])=><article className="statCard" key={label}><strong>{value}</strong><span>{label}</span></article>)}</div>}{!error&&<div className="twoColumns"><div className="card"><h2>Последние проверки</h2><div className="timeline">{estimates.slice(0,5).map((item,i)=><p key={item.id}><b>{String(i+1).padStart(2,'0')}</b><span>{item.file_name} · {statusLabel(item.status)} · {item.items_count||0} строк · оценка {item.score}/100</span></p>)}</div>{estimates.length===0&&<p>Пока нет смет. Загрузите первую Excel или CSV смету.</p>}</div><div className="card"><h2>Что уже проверяется</h2><ul><li>Пустые названия, количества, цены и суммы</li><li>Расхождение количество × цена и итоговой суммы</li><li>Возможные дубли позиций</li><li>Крупные суммы, требующие ручной проверки</li></ul><div className="buttonRow"><a className="btn" href="/upload">Загрузить</a><a className="btn secondary" href="/reports">Открыть отчёты</a></div></div></div>}</>}
    </section><Footer/></main>;
}
