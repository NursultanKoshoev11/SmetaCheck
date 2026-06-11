import Head from 'next/head';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const findings=[
  ['Высокий','Строка 8','Расхождение в итоговой сумме','120 м² × 950 сом = 114 000 сом, но указано 126 000 сом.'],
  ['Высокий','Строки 12 и 19','Возможный дубль','Две позиции содержат одинаковый материал и близкие объёмы.'],
  ['Средний','Строка 23','Нет единицы измерения','Невозможно понять, за что именно указана цена.'],
  ['Средний','Строка 31','Крупная позиция','Стоимость требует отдельного подтверждения расчётом или предложением поставщика.'],
];

export default function SampleReport(){
  return <main className="page">
    <Head><title>Пример отчёта проверки сметы — SmetaCheck KG</title><meta name="description" content="Публичный пример отчёта SmetaCheck: оценка, замечания, вопросы подрядчику и позиции сметы."/></Head>
    <Nav/>
    <section className="sampleReportShell">
      <div className="sampleReportHero"><p className="eyebrow">Публичный пример отчёта</p><h1>Смета на строительство частного дома</h1><p>Демонстрационные данные показывают структуру результата. Это не заключение по реальному объекту и не замена профессиональной экспертизы.</p><div className="demoActions"><a className="btn" href="/demo">Запустить demo</a><a className="btn secondary" href="/upload">Проверить свой файл</a></div></div>
      <div className="reportKpis"><article><strong>82</strong><span>оценка структуры</span></article><article><strong>34</strong><span>строки</span></article><article><strong>4</strong><span>замечания</span></article><article><strong>2</strong><span>высокий риск</span></article></div>
      <div className="twoColumns"><article className="card"><p className="eyebrow">Краткий вывод</p><h2>Смету можно обсуждать, но не стоит утверждать без уточнений.</h2><p>Основные вопросы связаны с одним арифметическим расхождением, возможным дублем материала, отсутствующей единицей измерения и крупной позицией без подтверждения.</p></article><article className="card"><p className="eyebrow">Следующие действия</p><h2>Что сделать до оплаты</h2><ol><li>Запросить исправленный расчёт строки 8.</li><li>Уточнить, являются ли строки 12 и 19 разными поставками.</li><li>Добавить единицу измерения в строку 23.</li><li>Получить подтверждение стоимости крупной позиции.</li></ol></article></div>
      <article className="card"><h2>Замечания</h2><div className="reportTable"><table className="dataTable"><thead><tr><th>Риск</th><th>Строка</th><th>Проблема</th><th>Пояснение</th></tr></thead><tbody>{findings.map(row=><tr key={row[1]+row[2]}>{row.map(value=><td key={value}>{value}</td>)}</tr>)}</tbody></table></div></article>
      <article className="card"><h2>Вопросы подрядчику</h2><ul><li>Почему итог по строке 8 выше математического расчёта на 12 000 сом?</li><li>Строки 12 и 19 относятся к разным этапам, поставкам или это повтор?</li><li>В какой единице измеряется гидроизоляция и как рассчитан объём?</li><li>Каким предложением или расчётом подтверждена крупная позиция?</li></ul></article>
      <article className="card"><h2>Как читать оценку</h2><p>Оценка показывает качество структуры файла и количество автоматических замечаний. Высокая оценка не подтверждает рыночную стоимость, качество работ или соответствие нормативам.</p><a className="btn secondary" href="/methodology">Открыть методику</a></article>
      <section className="marketingCta"><h2>Получите такой отчёт по своей смете.</h2><p>Для приватной загрузки и сохранения истории потребуется аккаунт.</p><div className="ugActions"><a className="btn" href="/upload">Загрузить смету</a><a className="btn secondary" href="/pricing">Посмотреть тарифы</a></div></section>
    </section>
    <Footer/>
  </main>
}
