import Nav from '../components/Nav';
import Footer from '../components/Footer';

const steps = [
  ['01', 'Загрузите смету', 'Поддерживаются Excel и CSV файлы. PDF можно сохранить, но для точной проверки лучше использовать таблицу.'],
  ['02', 'Система читает строки', 'SmetaCheck ищет колонки с названием, единицей измерения, количеством, ценой и суммой.'],
  ['03', 'Проверяются ошибки', 'Находим пустые поля, расхождения в суммах, возможные дубли и крупные позиции для ручной проверки.'],
  ['04', 'Получаете отчёт', 'Отчёт можно использовать как список вопросов к прорабу, сметчику или подрядчику.'],
];

const checks = ['Пустое наименование', 'Нет количества', 'Нет цены', 'Нет суммы', 'Количество × цена не равно сумме', 'Повторяющиеся позиции', 'Крупные суммы', 'Изменения между версиями'];

export default function HowItWorks(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Как работает SmetaCheck</p>
        <h1>Проверка сметы превращается из ручного хаоса в понятный процесс.</h1>
        <p>Сервис не заменяет эксперта, а помогает быстрее найти места, которые точно нужно обсудить до оплаты или утверждения бюджета.</p>
      </section>
      <section className="workspace grid features">
        {steps.map(([n,title,text]) => <article className="card feature" key={title}><span>{n}</span><h3>{title}</h3><p>{text}</p></article>)}
      </section>
      <section className="workspace">
        <div className="card"><h2>Что проверяется</h2><div className="grid">{checks.map(item => <p key={item}>{item}</p>)}</div><a className="btn" href="/upload">Проверить смету</a></div>
      </section>
      <Footer/>
    </main>
  )
}
