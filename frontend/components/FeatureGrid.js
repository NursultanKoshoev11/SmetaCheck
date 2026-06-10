const items = [
  ['01', 'Upload estimate files', 'Start from Excel or PDF-style estimate documents and keep every check in one workspace.'],
  ['02', 'Review key numbers', 'Highlight missing units, duplicate rows, empty quantities, and totals that need attention.'],
  ['03', 'Generate clean reports', 'Prepare owner-friendly summaries with categories, severity, and next actions.'],
  ['04', 'Compare versions', 'Review what changed between two estimate versions before approving a budget.'],
  ['05', 'Team dashboard', 'Give owners, engineers, and project managers one shared view of construction cost review.'],
  ['06', 'Telegram-ready workflow', 'Keep teams informed when a file is uploaded, processed, or ready for review.'],
]

export default function FeatureGrid(){
  return (
    <section className="section">
      <div className="sectionHeader">
        <p className="eyebrow">Core workflow</p>
        <h2>Everything needed for a clear estimate review.</h2>
      </div>
      <div className="grid features">
        {items.map(([number,title,text]) => (
          <article className="card feature" key={title}>
            <span>{number}</span>
            <h3>{title}</h3>
            <p>{text}</p>
          </article>
        ))}
      </div>
    </section>
  )
}
