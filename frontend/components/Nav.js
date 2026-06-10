export default function Nav(){
  return (
    <nav className="nav">
      <a className="brand" href="/">
        <span className="brandMark">S</span>
        <span>SmetaCheck KG</span>
      </a>
      <div className="navLinks">
        <a href="/upload">Upload</a>
        <a href="/dashboard">Dashboard</a>
        <a href="/reports">Reports</a>
        <a href="/compare">Compare</a>
        <a href="/pricing">Pricing</a>
        <a href="/support">Support</a>
      </div>
      <a className="navAction" href="/login">Sign in</a>
    </nav>
  )
}
