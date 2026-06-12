import Head from 'next/head';
import Nav from '../components/Nav';
import Footer from '../components/Footer';

export default function Privacy(){
  return <main className="page"><Head><title>Политика конфиденциальности — SmetaCheck KG</title></Head><Nav/><section className="contentShell"><p className="eyebrow">Версия 2026-06-12</p><h1>Политика конфиденциальности</h1><p>SmetaCheck KG обрабатывает данные аккаунта, загруженные строительные сметы, извлечённые строки, результаты проверок, технические журналы и сведения об использовании сервиса для предоставления и защиты продукта.</p><h2>Файлы и AI</h2><p>Для AI-анализа распознанные строки и замечания могут передаваться настроенному AI-провайдеру. При raw PDF-анализе провайдеру может передаваться весь PDF.</p><h2>Хранение и удаление</h2><p>Пользователь может удалить отдельную смету или весь аккаунт. Удалённые данные могут сохраняться в зашифрованных резервных копиях до окончания срока хранения backup.</p><h2>Контакт</h2><p><a href="mailto:support@smetacheck.kg">support@smetacheck.kg</a></p></section><Footer/></main>;
}
