import Nav from '../components/Nav';
import Footer from '../components/Footer';
import LegalPrivacy from '../components/LegalPrivacy';
import LegalTerms from '../components/LegalTerms';

export default function Legal(){return <main className="page"><Nav/><section className="pageHero compact"><p className="eyebrow">Редакция 12 июня 2026 года</p><h1>Privacy Policy и Terms of Service</h1><p>Правила обработки документов и использования сервиса SmetaCheck.</p><p><a href="#privacy">Privacy Policy</a> · <a href="#terms">Terms of Service</a></p></section><LegalPrivacy/><LegalTerms/><Footer/></main>}
