export default function Hero(){
  return (
    <section className="hero">
      <div className="heroText">
        <p className="eyebrow">Construction estimate intelligence</p>
        <h1>Find hidden risks before construction money is lost.</h1>
        <p className="lead">Upload an estimate, detect calculation issues, compare risk categories, and prepare a clear report for owners, builders, and project managers.</p>
        <div className="heroActions">
          <a className="btn" href="/upload">Upload estimate</a>
          <a className="btn secondary" href="/dashboard">Open dashboard</a>
        </div>
      </div>
      <div className="heroPanel">
        <div className="panelTop">
          <span>Live check preview</span>
          <b>Ready</b>
        </div>
        <div className="scoreRing"><strong>82</strong><span>risk score</span></div>
        <div className="miniList">
          <p><b>12</b><span>issues found</span></p>
          <p><b>4</b><span>high priority</span></p>
          <p><b>PDF</b><span>report export</span></p>
        </div>
      </div>
    </section>
  )
}
