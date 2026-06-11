import Head from 'next/head';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

const benefits=[
  ['Понять смету без специальной подготовки','Получите список конкретных строк, которые требуют объяснения.'],
  ['Обсудить спорные расходы до оплаты','Сформулированные вопросы помогают вести предметный разговор с подрядчиком.'],
  ['Увидеть изменения в новой версии','Сравните, что добавили, удалили или пересчитали после обсуждения.'],
];

export default function ForHomeowners(){
  return <main className="page">
    <Head><title>Проверка сметы на дом и ремонт — SmetaCheck KG</title><meta name="description" content="Проверьте смету подрядчика до оплаты: расчётные ошибки, возможные дубли, неполные строки и изменения версий."/></Head>
    <Nav/>
    <section className="marketingHero">
      <div><p className="eyebrow">Для владельца дома или ремонта</p><h1>Поймите, за что вы платите, до предоплаты.</h1><p className="lead">SmetaCheck не обещает заменить инженера. Он помогает быстро найти строки, которые стоит уточнить у прораба, подрядчика или сметчика.</p><div className="heroActions"><a className="btn" href="/demo">Проверить demo-смету</a><a className="btn secondary" href="/sample-report">Посмотреть отчёт</a></div></div>
      <div className="productPreview"><div className="previewHead"><span>Вопросы перед оплатой</span><b>4 пункта</b></div><article className="previewFinding"><i>1</i><div><h3>Почему сумма выше расчёта?</h3><p>Сервис показывает строку и разницу, которую нужно объяснить.</p></div><em>Уточнить</em></article><article className="previewFinding"><i>2</i><div><h3>Это не повторная позиция?</h3><p>Похожие материалы собраны рядом для ручной проверки.</p></div><em>Проверить</em></article><article className="previewFinding"><i>3</i><div><h3>Как рассчитан объём?</h3><p>Неполные единицы и количества выделены отдельно.</p></div><em>Уточнить</em></article></div>
    </section>
    <section className="marketingSection"><div className="marketingSectionHeader"><p className="eyebrow">Что вы получите</p><h2>Понятный список вопросов вместо сотен строк Excel.</h2></div><div className="valueGrid">{benefits.map(([title,text],index)=><article className="valueCard" key={title}><span>{String(index+1).padStart(2,'0')}</span><h3>{title}</h3><p>{text}</p></article>)}</div></section>
    <section className="marketingSection"><div className="twoColumns"><article className="card"><h2>Когда сервис особенно полезен</h2><ul><li>До первой крупной предоплаты.</li><li>После получения обновлённой сметы.</li><li>Когда итоговая сумма выросла без понятного объяснения.</li><li>Когда документ сложно читать без опыта.</li></ul></article><article className="card"><h2>Что всё равно нужно проверить специалисту</h2><ul><li>Правильность объёмов по чертежам.</li><li>Соответствие строительным нормам.</li><li>Качество материалов и работ.</li><li>Рыночную обоснованность цены.</li></ul></article></div></section>
    <section className="marketingCta"><h2>Начните с безопасного demo.</h2><p>Сначала посмотрите результат без регистрации. Затем загрузите собственную смету в приватный кабинет.</p><div className="ugActions"><a className="btn" href="/demo">Запустить demo</a><a className="btn secondary" href="/upload">Загрузить свою смету</a></div></section>
    <Footer/>
  </main>
}
