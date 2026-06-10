const plans = [
  ['Free', '$0', 'For testing the first estimate review', ['3 checks per month', 'Basic report preview', 'Single user']],
  ['Pro', '$29', 'For active builders and estimators', ['100 checks per month', 'PDF export', 'Version comparison']],
  ['Company', 'Custom', 'For teams and construction companies', ['Team dashboard', 'Priority workflow', 'Custom onboarding']],
]

export default function PricingCards(){
  return (
    <section className="section">
      <div className="sectionHeader">
        <p className="eyebrow">Pricing</p>
        <h2>Start small. Upgrade when the workflow is proven.</h2>
      </div>
      <div className="grid plans">
        {plans.map(([name,price,description,features]) => (
          <article className="card plan" key={name}>
            <p className="planName">{name}</p>
            <h3>{price}</h3>
            <p>{description}</p>
            <ul>{features.map(item => <li key={item}>{item}</li>)}</ul>
            <a className="btn secondary" href="/login">Choose plan</a>
          </article>
        ))}
      </div>
    </section>
  )
}
