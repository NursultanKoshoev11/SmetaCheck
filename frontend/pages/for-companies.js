import Head from 'next/head';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const benefits=[
  ['Единый стандарт первичной проверки','Команда использует одинаковые правила и одинаковую структуру отчёта.'],
  ['История по объектам','Сметы и версии остаются привязаны к проекту и ответственному пользователю.'],
  ['Быстрее внутреннее согласование','Руководитель видит сводку рисков, суммы и изменения без ручного просмотра каждой строки.'],
];

export default function ForCompanies(){
  return <main className="page">
    <Head><title>SmetaCheck для строительных компаний</title><meta name="description" content="Пилот SmetaCheck для строительной команды: единая проверка смет, история версий, отчёты и прозрачное согласование."/></Head>
    <Nav/>
    <section className="marketingHero">
      <div><p className="eyebrow">Для строительной компании</p><h1>Стандартизируйте проверку смет до внутреннего согласования.</h1><p className="lead">Запустите пилот на одном объекте: загрузите реальные версии сметы, проверьте качество данных и оцените, сколько ручной работы можно убрать из процесса.</p><div className="heroActions"><a className="btn" href="/support">Запросить пилот</a><a className="btn secondary" href="/sample-report">Посмотреть отчёт</a></div></div>
      <div className="productPreview"><div className="previewHead"><span>Портфель проектов</span><b>Пилот</b></div><div className="reportKpis"><article><strong>7</strong><span>смет</span></article><article><strong>3</strong><span>проекта</span></article><article><strong>18</strong><span>замечаний</span></article><article><strong>5</strong><span>версий</span></article></div><article className="previewFinding"><i>1</i><div><h3>Объект: частный дом</h3><p>Новая версия: изменено 12 позиций, общая сумма выросла.</p></div><em>Проверить</em></article><article className="previewFinding"><i>2</i><div><h3>Объект: ремонт офиса</h3><p>Критичные арифметические замечания закрыты.</p></div><em>Готово</em></article></div>
    </section>
    <section className="marketingSection"><div className="marketingSectionHeader"><p className="eyebrow">Что даёт пилот</p><h2>Проверяем не обещания, а реальный процесс вашей команды.</h2></div><div className="valueGrid">{benefits.map(([title,text],index)=><article className="valueCard" key={title}><span>{String(index+1).padStart(2,'0')}</span><h3>{title}</h3><p>{text}</p></article>)}</div></section>
    <section className="marketingSection"><div className="twoColumns"><article className="card"><h2>Что входит в пилот</h2><ul><li>Один реальный объект.</li><li>Несколько файлов или версий сметы.</li><li>Совместная проверка результатов.</li><li>Список требований к дальнейшему внедрению.</li></ul></article><article className="card"><h2>Что измеряем</h2><ul><li>Процент успешно распознанных строк.</li><li>Количество полезных и ложных замечаний.</li><li>Время до первого понятного результата.</li><li>Доля изменений, найденных между версиями.</li></ul></article></div></section>
    <section className="marketingCta"><h2>Начните с одного объекта.</h2><p>После пилота будет понятно, какие роли, интеграции и отчёты действительно нужны вашей компании.</p><div className="ugActions"><a className="btn" href="/support">Запросить пилот</a><a className="btn secondary" href="/security">Проверить обработку данных</a></div></section>
    <Footer/>
  </main>
}
