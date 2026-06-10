import Nav from '../components/Nav';
import Footer from '../components/Footer';

const findings = [
  ['1', 'Quantity and total mismatch', 'Several rows need recalculation before approval.', 'High'],
  ['2', 'Repeated material positions', 'Similar items appear more than once in the estimate.', 'Medium'],
  ['3', 'Missing units and prices', 'Some rows are not ready for a clean budget review.', 'High'],
];

const steps = [
  ['Step 1', 'Upload your estimate', 'Add the file you received from a builder, contractor, or internal team.'],
  ['Step 2', 'Structure the numbers', 'Prepare rows, totals, categories, and items for review.'],
  ['Step 3', 'Get a clear report', 'See what needs review and what can be exported later.'],
];

export default function Home(){
  return (
    <main className="page ugPage">
      <Nav/>
      <section className="ugHero">
        <p className="ugProof">Built for Kyrgyzstan construction estimates · Fast review · Report-ready workflow</p>
        <h1>Know where estimate money leaks before construction starts.</h1>
        <p className="ugLead">Documented issues. Clear numbers. Simple reports. Upload an estimate and turn a messy file into a decision-ready review.</p>
        <div className="ugActions"><a className="btn" href="/upload">Check estimate</a><a className="btn secondary" href="/reports">See report example</a></div>
        <p className="ugFine">No card required for first test · Designed for owners, estimators, and project managers</p>
      </section>

      <section className="ugReport">
        <div className="ugReportTop"><span>Estimate review · Today</span><b>Ready</b></div>
        <h2>Residential estimate: 12 review points found</h2>
        <p className="ugReportSummary">Summary: the file is readable, but several totals, repeated rows, and missing units should be checked before the budget is approved.</p>
        <div className="ugFindingList">
          {findings.map(([n,title,text,tag]) => <article key={title}><span>{n}</span><div><h3>{title}</h3><p>{text}</p></div><b>{tag}</b></article>)}
        </div>
      </section>

      <section className="ugSection">
        <p className="eyebrow">How it works</p>
        <h2>From file to review in one clean flow.</h2>
        <div className="ugSteps">{steps.map(([step,title,text]) => <article key={title}><span>{step}</span><h3>{title}</h3><p>{text}</p></article>)}</div>
      </section>

      <section className="ugCompare">
        <div><p className="eyebrow">Before</p><h2>Manual review</h2><ul><li>Review takes too long.</li><li>Important rows are easy to miss.</li><li>The owner receives unclear explanations.</li></ul></div>
        <div><p className="eyebrow">After</p><h2>SmetaCheck review</h2><ul><li>Issues are grouped and explained.</li><li>Totals and repeated rows are easier to review.</li><li>The result is ready for a clear report.</li></ul></div>
      </section>

      <section className="ugCta"><h2>Upload one estimate. See what needs review.</h2><p>Start with the file you already have. The first goal is clarity before approval.</p><a className="btn" href="/upload">Start checking</a></section>
      <Footer/>
    </main>
  )
}
