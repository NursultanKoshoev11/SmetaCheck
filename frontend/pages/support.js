import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Support(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Помощь и внедрение</p>
        <h1>Поможем показать ценность SmetaCheck вашей команде или клиенту.</h1>
        <p>Подскажем, как загрузить смету, объяснить отчёт, настроить процесс проверки и подготовить демонстрацию для строительного проекта.</p>
      </section>
      <section className="workspace grid features">
        <article className="card feature"><span>01</span><h3>Консультация</h3><p>Разберём ваш сценарий: частный дом, ремонт, объект компании или работа сметчика.</p><a href="mailto:support@smetacheck.kg">support@smetacheck.kg</a></article>
        <article className="card feature"><span>02</span><h3>Демо для клиента</h3><p>Поможем показать, как проверка сметы снижает недоверие и ускоряет согласование бюджета.</p><a href="/upload">Загрузить пример</a></article>
        <article className="card feature"><span>03</span><h3>Внедрение в команду</h3><p>Настроим понятный процесс: кто загружает, кто проверяет, кто получает отчёт.</p><a href="/pricing">Посмотреть тарифы</a></article>
      </section>
      <Footer/>
    </main>
  )
}
