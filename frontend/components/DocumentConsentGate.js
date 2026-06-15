import {useEffect,useState} from 'react';

export const LEGAL_VERSION='2026-06-12';
export const CONSENT_KEY=`smetacheck-document-consent-${LEGAL_VERSION}`;

export default function DocumentConsentGate({children}){
  const [rights,setRights]=useState(false);
  const [processing,setProcessing]=useState(false);
  const [ai,setAI]=useState(false);
  const [accepted,setAccepted]=useState(false);

  useEffect(()=>{if(typeof window!=='undefined')setAccepted(window.sessionStorage.getItem(CONSENT_KEY)==='accepted');},[]);
  function confirm(){if(!rights||!processing||!ai)return;window.sessionStorage.setItem(CONSENT_KEY,'accepted');setAccepted(true);}
  if(accepted)return <><div className="consentAcceptedModern"><span className="consentAcceptedIcon">✓</span><div><b>Согласие подтверждено</b><p>Файлы будут проверены перед обработкой. XLS, XLSX и XLSM принимаются только как данные.</p></div></div>{children}</>;
  const ready=rights&&processing&&ai;
  return <section className="consentModern">
    <div className="consentModernHeader"><div className="consentShield">✓</div><div><span className="consentEyebrow">Безопасная загрузка документов</span><h2>Подтвердите перед загрузкой</h2><p>Проверьте условия обработки перед отправкой сметы на анализ.</p></div></div>
    <div className="consentChecklist">
      <label className={rights?'checked':''}><input type="checkbox" checked={rights} onChange={e=>setRights(e.target.checked)}/><span className="consentCheckmark"/><span>Я имею законное право загружать эти документы.</span></label>
      <label className={processing?'checked':''}><input type="checkbox" checked={processing} onChange={e=>setProcessing(e.target.checked)}/><span className="consentCheckmark"/><span>Я принимаю <a href="/legal#terms" target="_blank" rel="noreferrer">Terms of Service</a> и <a href="/legal#privacy" target="_blank" rel="noreferrer">Privacy Policy</a>.</span></label>
      <label className={ai?'checked':''}><input type="checkbox" checked={ai} onChange={e=>setAI(e.target.checked)}/><span className="consentCheckmark"/><span>Я согласен на выбранный способ обработки и возможную передачу данных выбранному AI-провайдеру.</span></label>
    </div>
    <div className="consentInfo"><span>i</span><p>В режиме локальных правил внешний AI не используется. XLS, XLSX и XLSM принимаются только как данные; VBA-макросы никогда не исполняются.</p></div>
    <div className="consentFooter"><button className="consentPrimary" type="button" disabled={!ready} onClick={confirm}>Подтвердить и продолжить <span>→</span></button><small>{ready?'Все условия подтверждены. Можно продолжать.':'Отметьте все пункты, чтобы продолжить загрузку.'}</small></div>
  </section>;
}
