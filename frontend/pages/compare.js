import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Compare(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Сравнение смет</p>
        <h1>Сравните две версии сметы перед согласованием бюджета.</h1>
        <p>Покажите клиенту или команде, что добавилось, что изменилось и какие суммы требуют внимания.</p>
      </section>
      <section className="workspace twoColumns">
        <div className="compareDrop"><span>01</span><h2>Исходная смета</h2><p>Загрузите первую версию документа.</p><input type="file" /></div>
        <div className="compareDrop"><span>02</span><h2>Новая версия</h2><p>Загрузите обновлённую смету.</p><input type="file" /></div>
      </section>
      <section className="workspace">
        <div className="card"><h2>Что даст сравнение</h2><div className="grid"><p>Новые позиции</p><p>Удалённые позиции</p><p>Изменения по суммам</p></div><a className="btn" href="/reports">Перейти к отчётам</a></div>
      </section>
      <Footer/>
    </main>
  )
}
