const plans = [
  ['Старт', '0 сом', 'Для первой проверки и демонстрации ценности', ['3 проверки в месяц', 'Базовый отчёт', '1 пользователь']],
  ['Pro', 'по запросу', 'Для прорабов, сметчиков и активных проектов', ['Больше проверок', 'История отчётов', 'Сравнение версий']],
  ['Компания', 'индивидуально', 'Для строительных компаний и команд', ['Командный кабинет', 'Приоритетная настройка', 'Помощь при внедрении']],
]

export default function PricingCards(){
  return (
    <section className="section">
      <div className="sectionHeader">
        <p className="eyebrow">Тарифы</p>
        <h2>Начните с одной сметы и масштабируйте проверку на всю команду.</h2>
      </div>
      <div className="grid plans">
        {plans.map(([name,price,description,features]) => (
          <article className="card plan" key={name}>
            <p className="planName">{name}</p>
            <h3>{price}</h3>
            <p>{description}</p>
            <ul>{features.map(item => <li key={item}>{item}</li>)}</ul>
            <a className="btn secondary" href="/login">Выбрать тариф</a>
          </article>
        ))}
      </div>
    </section>
  )
}
