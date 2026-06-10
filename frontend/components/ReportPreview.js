export default function ReportPreview(){
  const rows = [
    ['Missing unit', 'High', '7 rows'],
    ['Duplicate item', 'Medium', '3 rows'],
    ['Total mismatch', 'High', '2 rows'],
  ]

  return (
    <section className="section splitSection">
      <div>
        <p className="eyebrow">Report preview</p>
        <h2>A simple report that a client can understand.</h2>
        <p className="sectionLead">The interface is designed to show what matters first: summary, risk level, issue category, and recommended next step.</p>
        <a className="btn secondary" href="/reports">View reports</a>
      </div>
      <div className="reportCard">
        <div className="reportHeader">
          <span>Estimate review</span>
          <b>PDF</b>
        </div>
        <div className="reportScore">
          <strong>82</strong>
          <div><b>Overall score</b><p>12 items need review before approval.</p></div>
        </div>
        <div className="tableLike">
          {rows.map(([name,level,count]) => <p key={name}><span>{name}</span><b>{level}</b><em>{count}</em></p>)}
        </div>
      </div>
    </section>
  )
}
