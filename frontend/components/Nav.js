export default function Nav(){
  return (
    <nav className="nav">
      <a className="brand" href="/">
        <span className="brandMark">S</span>
        <span>SmetaCheck KG</span>
      </a>
      <div className="navLinks">
        <a href="/upload">Проверить</a>
        <a href="/how-it-works">Как работает</a>
        <a href="/dashboard">Кабинет</a>
        <a href="/reports">Отчёты</a>
        <a href="/compare">Сравнение</a>
        <a href="/pricing">Тарифы</a>
        <a href="/faq">FAQ</a>
      </div>
      <a className="navAction" href="/login">Войти</a>
    </nav>
  )
}
