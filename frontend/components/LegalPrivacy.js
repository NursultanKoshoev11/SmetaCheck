import PrivacyData from './legal/PrivacyData';
import PrivacyAI from './legal/PrivacyAI';
import PrivacyRetention from './legal/PrivacyRetention';
import PrivacyRights from './legal/PrivacyRights';

export default function LegalPrivacy(){return <section className="workspace legalDocument" id="privacy"><h1>Privacy Policy</h1><p>Редакция от 12 июня 2026 года. Политика описывает обработку данных при регистрации, загрузке и анализе строительных смет.</p><PrivacyRights/><PrivacyData/><PrivacyAI/><PrivacyRetention/><div className="card"><h2>Цели и изменения</h2><p>Данные используются для работы аккаунта, проверки смет, создания отчётов, учёта оплаты и квот, поддержки и защиты сервиса. Новая редакция публикуется с датой вступления в силу; существенные изменения могут потребовать повторного согласия перед загрузкой.</p></div></section>}
