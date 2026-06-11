const plans = [
  {
    name:'Demo',
    price:'0 сом',
    badge:'Без регистрации',
    description:'Для знакомства с результатом на демонстрационной смете.',
    features:['Публичная demo-проверка','Открытый пример отчёта','Без загрузки личного файла'],
    href:'/demo',
    action:'Запустить demo',
  },
  {
    name:'Ранний доступ',
    price:'Бесплатно на этапе пилота',
    badge:'Для реальных смет',
    description:'Для владельцев, прорабов и сметчиков, которые готовы проверить продукт на своём документе.',
    features:['Приватная загрузка Excel или CSV','История проверок','AI-сводка и сравнение версий'],
    href:'/login',
    action:'Создать аккаунт',
    recommended:true,
  },
  {
    name:'Пилот для компании',
    price:'По согласованию',
    badge:'Один реальный объект',
    description:'Для строительной команды, которая хочет проверить процесс до полноценного внедрения.',
    features:['Несколько файлов и версий','Совместный разбор результатов','План ролей, отчётов и интеграций'],
    href:'/support',
    action:'Запросить пилот',
  },
];

export default function PricingCards(){
  return (
    <section className="section">
      <div className="sectionHeader">
        <p className="eyebrow">Тарифы и ранний доступ</p>
        <h2>Сначала проверьте ценность, затем выбирайте формат работы.</h2>
        <p>На текущем этапе доступны demo, ранний доступ и пилот для команды.</p>
      </div>
      <div className="grid plans">
        {plans.map((plan) => (
          <article className={`card plan ${plan.recommended ? 'recommended' : ''}`} key={plan.name}>
            <span className="priceBadge">{plan.badge}</span>
            <p className="planName">{plan.name}</p>
            <h3>{plan.price}</h3>
            <p>{plan.description}</p>
            <ul>{plan.features.map(item => <li key={item}>{item}</li>)}</ul>
            <a className={plan.recommended ? 'btn' : 'btn secondary'} href={plan.href}>{plan.action}</a>
          </article>
        ))}
      </div>
    </section>
  );
}
