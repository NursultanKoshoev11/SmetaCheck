import Head from 'next/head';
import {useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const demoFindings = [
  {severity:'Высокий риск', title:'Итог строки не совпадает с расчётом', detail:'Строка 8: 120 м² × 950 сом = 114 000 сом, но в смете указано 126 000 сом.', question:'Почему итоговая сумма выше расчётной на 12 000 сом?'},
  {severity:'Высокий риск', title:'Возможный дубль материала', detail:'Строки 12 и 19 содержат похожие позиции «Арматура A500C, 12 мм».', question:'Это две разные поставки или одна позиция учтена повторно?'},
  {severity:'Уточнить', title:'Не указана единица измерения', detail:'Строка 23 «Гидроизоляция фундамента» содержит количество и цену, но не содержит единицу.', question:'Цена указана за м², погонный метр или за всю работу?'},
  {severity:'Уточнить', title:'Крупная позиция требует подтверждения', detail:'Строка 31 составляет заметную часть общей суммы и выделена для ручной проверки.', question:'Есть ли коммерческое предложение или расчёт, подтверждающий эту стоимость?'},
];

export default function Demo(){
  const [started,setStarted]=useState(false);
  return <main className="page">
    <Head><title>Demo-проверка сметы — SmetaCheck KG</title><meta name="description" content="Посмотрите пример автоматической проверки строительной сметы без регистрации."/></Head>
    <Nav/>
    <section className="demoShell">
      <div className="demoTop">
        <p className="eyebrow">Публичная demo-проверка</p>
        <h1>Посмотрите, что получает клиент после загрузки сметы.</h1>
        <p>Это демонстрационный пример с заранее подготовленными данными. Он не загружает ваш файл и не требует аккаунта.</p>
        {!started ? <div className="demoActions"><button className="btn" type="button" onClick={()=>setStarted(true)}>Запустить demo-анализ</button><a className="btn secondary" href="/sample-report">Сразу открыть полный отчёт</a></div> : <div className="reportNotice">Demo завершено: найдено 4 пункта, которые нужно обсудить до согласования бюджета.</div>}
      </div>
      {started && <>
        <div className="reportKpis"><article><strong>82</strong><span>оценка структуры</span></article><article><strong>34</strong><span>строки сметы</span></article><article><strong>4</strong><span>замечания</span></article><article><strong>2</strong><span>высокий риск</span></article></div>
        <div className="demoResults">{demoFindings.map((item,index)=><article className="demoResultCard" key={item.title}><span>{index+1}</span><div><h3>{item.title}</h3><p>{item.detail}</p><p><b>Вопрос подрядчику:</b> {item.question}</p></div><b>{item.severity}</b></article>)}</div>
        <section className="marketingCta"><h2>Хотите проверить собственный файл?</h2><p>Создайте аккаунт, загрузите Excel или CSV и сохраните приватный отчёт в кабинете.</p><div className="ugActions"><a className="btn" href="/upload">Загрузить свою смету</a><a className="btn secondary" href="/sample-report">Посмотреть полный отчёт</a></div></section>
      </>}
    </section>
    <Footer/>
  </main>
}
