import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Login(){
  return (
    <main className="page">
      <Nav/>
      <section className="authShell">
        <div>
          <p className="eyebrow">Аккаунт</p>
          <h1>Войдите в кабинет проверки строительных смет.</h1>
          <p>Храните загруженные сметы, отчёты, историю проверок и рабочие материалы в одном месте.</p>
        </div>
        <form className="authCard">
          <label>Email<input placeholder="name@company.com" type="email" /></label>
          <label>Пароль<input placeholder="Ваш пароль" type="password" /></label>
          <button className="btn" type="button">Войти</button>
          <button className="btn secondary" type="button">Продолжить через Google</button>
          <p>Нет аккаунта? <a href="/pricing">Выбрать тариф</a></p>
        </form>
      </section>
      <Footer/>
    </main>
  )
}
