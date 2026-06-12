export const LEGAL_VERSION='2026-06-12';
export const CONSENT_KEY=`smetacheck-document-consent-${LEGAL_VERSION}`;

export function addDocumentConsent(form){
  if(typeof window==='undefined'||!(form instanceof FormData))return form;
  if(window.sessionStorage.getItem(CONSENT_KEY)!=='accepted')return form;
  form.set('document_rights_confirmed','true');
  form.set('processing_consent','true');
  form.set('ai_processing_consent','true');
  form.set('privacy_version',LEGAL_VERSION);
  form.set('terms_version',LEGAL_VERSION);
  return form;
}
