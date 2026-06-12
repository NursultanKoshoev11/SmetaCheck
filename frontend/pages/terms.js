import Head from 'next/head';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Terms(){
  return <main className="page">
    <Head><title>Условия использования — SmetaCheck KG</title></Head>
    <Nav/>
    <section className="contentShell">
      <p className="eyebrow">Документы</p>
      <h1>Условия использования</h1>
      <p>Дата обновления: 12 июня 2026 года.</p>
      <h2>О сервисе</h2>
      <p>SmetaCheck помогает предварительно проверять строительные сметы. Автоматический результат следует дополнительно проверять ответственным специалистом.</p>
      <h2>Правила пользователя</h2>
      <p>Пользователь загружает только документы, которые он вправе обрабатывать, защищает свой аккаунт и не нарушает работу сервиса.</p>
      <h2>Данные</h2>
      <p>Порядок обработки и удаления данных описан в <a href="/privacy">Политике конфиденциальности</a>.</p>
      <h2>Обновления</h2>
      <p>Актуальная версия условий публикуется на этой странице.</p>
      <h2>Контакты</h2>
      <p><a href="mailto:support@smetacheck.kg">support@smetacheck.kg</a></p>
    </section>
    <Footer/>
  </main>;
}
