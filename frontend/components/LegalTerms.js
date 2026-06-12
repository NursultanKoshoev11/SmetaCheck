import TermsScope from './legal/TermsScope';
import TermsDocuments from './legal/TermsDocuments';
import TermsPayment from './legal/TermsPayment';
import TermsLiability from './legal/TermsLiability';

export default function LegalTerms(){return <section className="workspace legalDocument" id="terms"><h1>Terms of Service</h1><p>Редакция от 12 июня 2026 года. Использование SmetaCheck означает принятие этих правил.</p><TermsScope/><TermsDocuments/><div className="card"><h2>Точность результата</h2><p>Автоматический анализ может содержать ошибки из-за структуры файла, формул, качества PDF, неполных исходных данных, региональных норм или ограничений AI. Пользователь самостоятельно проверяет результат перед его практическим использованием.</p></div><TermsPayment/><TermsLiability/></section>}
