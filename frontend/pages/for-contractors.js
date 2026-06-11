import Head from 'next/head';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const benefits=[
  ['Меньше споров при согласовании','Покажите клиенту, какие строки проверены и какие требуют уточнения.'],
  ['Быстрее объяснять изменения','Сравнение версий показывает добавленные, удалённые и изменённые позиции.'],
  ['Профессиональный отчёт','Соберите замечания и вопросы в одном понятном документе.'],
];

export default function ForContractors(){
  return <main className="page">
    <Head><title>SmetaCheck для прорабов и подрядчиков</title><meta name="description" content="Проверяйте строительные сметы, сравнивайте версии и готовьте прозрачный отчёт для заказчика."/></Head>
    <Nav/>
    <section className="marketingHero">
      <div><p className="eyebrow">Для прораба и подрядчика</p><h1>Согласовывайте смету с клиентом на основе фактов.</h1><p className="lead">SmetaCheck помогает до отправки заказчику найти неполные строки, расчётные расхождения и возможные дубли, а затем показать прозрачный отчёт.</p><div className="heroActions"><a className="btn" href="/demo">Посмотреть demo</a><a className="btn secondary" href="/support">Запросить пилот</a></div></div>
      <div className="productPreview"><div className="previewHead"><span>Отчёт для клиента</span><b>Готов к обсуждению</b></div><div className="previewScore"><strong>91</strong><div><b>После исправлений</b><span>Критичные расчётные замечания устранены</span></div></div><article className="previewFinding"><i>✓</i><div><h3>Исправлен итог строки</h3><p>Новая версия содержит согласованный расчёт.</p></div><em>Закрыто</em></article><article className="previewFinding"><i>?</i><div><h3>Нужно пояснение</h3><p>Одна крупная позиция требует приложения коммерческого предложения.</p></div><em>Открыто</em></article></div>
    </section>
    <section className="marketingSection"><div className="marketingSectionHeader"><p className="eyebrow">Ценность для команды</p><h2>Проверка до отправки клиенту и прозрачное согласование после.</h2></div><div className="valueGrid">{benefits.map(([title,text],index)=><article className="valueCard" key={title}><span>{String(index+1).padStart(2,'0')}</span><h3>{title}</h3><p>{text}</p></article>)}</div></section>
    <section className="marketingSection"><div className="twoColumns"><article className="card"><h2>До отправки сметы</h2><ul><li>Проверить обязательные поля и арифметику.</li><li>Просмотреть возможные дубли.</li><li>Подготовить пояснения по крупным позициям.</li></ul></article><article className="card"><h2>После обратной связи</h2><ul><li>Загрузить новую версию.</li><li>Показать, что именно изменилось.</li><li>Сохранить историю согласования в одном проекте.</li></ul></article></div></section>
    <section className="marketingCta"><h2>Покажите сервис на реальной смете.</h2><p>Для пилота достаточно одного проекта и двух версий документа.</p><div className="ugActions"><a className="btn" href="/upload">Проверить файл</a><a className="btn secondary" href="/support">Обсудить пилот</a></div></section>
    <Footer/>
  </main>
}
