import Head from 'next/head';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const checks = [
  ['Расхождения в расчётах', 'Проверяем строки, где количество × цена не совпадает с итоговой суммой.'],
  ['Возможные дубли', 'Выделяем похожие материалы и работы, которые могли попасть в смету несколько раз.'],
  ['Неполные позиции', 'Находим строки без названия, единицы, количества, цены или итоговой суммы.'],
  ['Изменения между версиями', 'Показываем, что добавили, удалили и изменили перед согласованием бюджета.'],
  ['Крупные позиции', 'Собираем дорогие строки в отдельный список для обязательного ручного уточнения.'],
  ['Понятные вопросы', 'Превращаем технические замечания в конкретные вопросы подрядчику или сметчику.'],
];

const audiences = [
  ['Владельцу дома', 'Поймите, какие строки нужно уточнить до предоплаты и утверждения бюджета.', '/for-homeowners', 'Проверить свою смету'],
  ['Прорабу и подрядчику', 'Покажите заказчику прозрачный отчёт и согласуйте обновлённую смету быстрее.', '/for-contractors', 'Подготовить отчёт клиенту'],
  ['Строительной компании', 'Сравнивайте версии, сохраняйте историю и стандартизируйте первичную проверку.', '/for-companies', 'Запустить пилот'],
];

const trust = [
  ['Правила работают без AI', 'Арифметические и структурные проверки выполняются детерминированно.'],
  ['AI объясняет, а не выдумывает', 'AI помогает сформулировать вывод и вопросы, но исходные замечания остаются видимыми.'],
  ['Результат не является экспертизой', 'SmetaCheck помогает найти места для проверки и не заменяет инженера-сметчика.'],
  ['Данные под контролем пользователя', 'В кабинете отображаются только сметы текущего аккаунта; правила обработки описаны отдельно.'],
];

export default function Home(){
  return (
    <main className="page">
      <Head>
        <title>SmetaCheck KG — проверка строительной сметы до оплаты</title>
        <meta name="description" content="Загрузите строительную смету и получите понятный список ошибок, дублей, неполных строк и изменений между версиями до утверждения бюджета." />
        <meta property="og:title" content="Проверьте строительную смету до оплаты подрядчику" />
        <meta property="og:description" content="SmetaCheck находит расчётные ошибки, возможные дубли и спорные позиции и готовит понятный отчёт." />
      </Head>
      <Nav/>

      <section className="marketingHero">
        <div>
          <p className="eyebrow">Независимая проверка строительной сметы</p>
          <h1>Проверьте смету до оплаты подрядчику.</h1>
          <p className="lead">SmetaCheck находит возможные дубли, неполные строки, ошибки в расчётах и изменения между версиями. Вы получаете понятный отчёт с вопросами, которые нужно обсудить до утверждения бюджета.</p>
          <div className="heroActions">
            <a className="btn" href="/demo">Проверить demo-смету</a>
            <a className="btn secondary" href="/sample-report">Посмотреть пример отчёта</a>
          </div>
          <div className="heroTrust">
            <span>Demo без регистрации</span>
            <span>Excel и CSV</span>
            <span>Русский интерфейс</span>
            <span>Для проектов в Кыргызстане</span>
          </div>
        </div>

        <div className="productPreview" aria-label="Пример отчёта SmetaCheck">
          <div className="previewHead"><span>Смета на строительство дома</span><b>Проверка завершена</b></div>
          <div className="previewScore"><strong>82</strong><div><b>Оценка структуры</b><span>Найдено 12 пунктов для проверки до согласования бюджета</span></div></div>
          <article className="previewFinding"><i>!</i><div><h3>Возможный дубль позиции</h3><p>Кладочная сетка указана в двух строках с похожим названием.</p></div><em>Высокий риск</em></article>
          <article className="previewFinding"><i>!</i><div><h3>Не совпадает расчёт</h3><p>Количество × цена отличается от итоговой суммы строки.</p></div><em>Высокий риск</em></article>
          <article className="previewFinding"><i>?</i><div><h3>Нет единицы измерения</h3><p>Без единицы сложно проверить объём и сравнить предложение.</p></div><em>Уточнить</em></article>
        </div>
      </section>

      <section className="marketingSection">
        <div className="marketingSectionHeader"><p className="eyebrow">Что проверяет сервис</p><h2>Не просто оценка, а конкретные строки и причины.</h2><p>Каждое замечание привязано к данным сметы, чтобы вы могли быстро обсудить его с ответственным человеком.</p></div>
        <div className="valueGrid">{checks.map(([title,text], index)=><article className="valueCard" key={title}><span>{String(index+1).padStart(2,'0')}</span><h3>{title}</h3><p>{text}</p></article>)}</div>
      </section>

      <section className="conversionBand">
        <div><h2>Посмотрите реальный результат до регистрации.</h2><p>Сначала проверьте пример, затем решите, стоит ли загружать собственный документ.</p></div>
        <div className="heroActions"><a className="btn" href="/demo">Запустить demo</a><a className="btn secondary" href="/how-it-works">Как это работает</a></div>
      </section>

      <section className="marketingSection">
        <div className="marketingSectionHeader"><p className="eyebrow">Для кого</p><h2>Один отчёт — разные практические задачи.</h2></div>
        <div className="audienceGrid">{audiences.map(([title,text,href,action])=><article className="audienceCard" key={title}><span className="proofPill">SmetaCheck KG</span><h3>{title}</h3><p>{text}</p><a href={href}>{action} →</a></article>)}</div>
      </section>

      <section className="marketingSection">
        <div className="marketingSectionHeader"><p className="eyebrow">Прозрачность</p><h2>Понимайте не только результат, но и ограничения.</h2></div>
        <div className="trustGrid">{trust.map(([title,text])=><article className="trustItem" key={title}><b>{title}</b><p>{text}</p></article>)}</div>
        <div className="demoActions"><a className="btn secondary" href="/security">Как мы обрабатываем данные</a><a className="linkButton" href="/methodology">Методика проверки →</a></div>
      </section>

      <section className="marketingCta">
        <p className="eyebrow">Начните с результата</p>
        <h2>Проверьте пример или загрузите свою смету.</h2>
        <p>Demo не требует регистрации. Для приватной загрузки, сохранения истории и отчётов понадобится аккаунт.</p>
        <div className="ugActions"><a className="btn" href="/demo">Проверить demo-смету</a><a className="btn secondary" href="/upload">Загрузить свой файл</a></div>
      </section>
      <Footer/>
    </main>
  );
}
