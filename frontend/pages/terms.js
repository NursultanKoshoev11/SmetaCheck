import Head from 'next/head';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Terms() {
  return <main className="page">
    <Head><title>Условия использования — SmetaCheck KG</title></Head>
    <Nav/>
    <section className="contentShell">
      <p className="eyebrow">Версия 2026-06-12</p>
      <h1>Условия использования</h1>
      <p>SmetaCheck выполняет предварительную автоматизированную проверку строительной сметы.</p>
      <h2>Ограничения</h2>
      <p>Результат не является официальной экспертизой и не заменяет квалифицированного инженера или сметчика. Автоматические правила и AI могут ошибаться.</p>
      <h2>Обязанности пользователя</h2>
      <p>Пользователь подтверждает право загружать документ и обязан проверить результат до принятия финансового решения.</p>
      <h2>Оплата</h2>
      <p>Платные функции предоставляются по выбранному тарифу. Условия периода, отмены и возврата показываются перед оплатой.</p>
      <p><a href="mailto:support@smetacheck.kg">support@smetacheck.kg</a></p>
    </section>
    <Footer/>
  </main>;
}
