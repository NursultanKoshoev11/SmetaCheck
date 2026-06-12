import {useState} from 'react';

export const LEGAL_VERSION='2026-06-12';
export const CONSENT_KEY=`smetacheck-document-consent-${LEGAL_VERSION}`;

export default function DocumentConsentGate({children}){
  const [rights,setRights]=useState(false);
  const [processing,setProcessing]=useState(false);
  const [ai,setAI]=useState(false);
  const [accepted,setAccepted]=useState(false);

  function confirm(){
    if(!rights||!processing||!ai)return;
    window.sessionStorage.setItem(CONSENT_KEY,'accepted');
    setAccepted(true);
  }

  if(accepted)return <><div className="card consentAccepted"><b>Согласие подтверждено</b><p>Файлы будут проверены до обработки. Макросы XLSM не запускаются.</p></div>{children}</>;
  return <div className="card consentCard"><h2>Подтвердите перед загрузкой</h2><label><input type="checkbox" checked={rights} onChange={event=>setRights(event.target.checked)}/> Я имею законное право загружать эти документы.</label><label><input type="checkbox" checked={processing} onChange={event=>setProcessing(event.target.checked)}/> Я принимаю <a href="/legal#terms" target="_blank" rel="noreferrer">Terms of Service</a> и <a href="/legal#privacy" target="_blank" rel="noreferrer">Privacy Policy</a>.</label><label><input type="checkbox" checked={ai} onChange={event=>setAI(event.target.checked)}/> Я согласен на выбранный способ обработки и возможную передачу данных выбранному AI-провайдеру.</label><p>В режиме локальных правил внешний AI не используется. XLSM принимается только как данные; VBA-макросы никогда не исполняются.</p><button className="btn" type="button" disabled={!rights||!processing||!ai} onClick={confirm}>Подтвердить и продолжить</button></div>;
}
