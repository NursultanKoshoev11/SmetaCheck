import Nav from '../components/Nav';
import PricingCards from '../components/PricingCards';
import Footer from '../components/Footer';

export default function Pricing(){
  return (
    <main className="page">
      <Nav/>
      <section className="pageHero compact">
        <p className="eyebrow">Тарифы</p>
        <h1>Выберите формат проверки смет под ваш проект.</h1>
        <p>Можно начать с одной сметы, показать результат клиенту или команде, а затем подключить регулярную проверку для всех проектов.</p>
      </section>
      <PricingCards/>
      <section className="workspace">
        <div className="card"><h2>Нужен тариф для компании?</h2><p>Мы подготовим формат для строительной команды: роли, отчёты, история проверок и понятный процесс согласования.</p><a className="btn" href="/support">Связаться</a></div>
      </section>
      <Footer/>
    </main>
  )
}
