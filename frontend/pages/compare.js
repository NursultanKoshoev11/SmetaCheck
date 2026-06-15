import {useEffect,useState} from 'react';
import Nav from '../components/Nav';
import Footer from '../components/Footer';
import {apiFetch,currentUser,readJSON} from '../lib/api';

const money=value=>Number(value||0).toLocaleString('ru-RU');

export default function Compare(){
  const [baseFile,setBaseFile]=useState(null);const [newFile,setNewFile]=useState(null);const [status,setStatus]=useState('idle');const [message,setMessage]=useState('');const [result,setResult]=useState(null);const [authorized,setAuthorized]=useState(false);const [checking,setChecking]=useState(true);

  useEffect(()=>{let active=true;currentUser().then(user=>{if(active)setAuthorized(Boolean(user));}).catch(()=>setMessage('РќРµ СѓРґР°Р»РѕСЃСЊ СЃРІСЏР·Р°С‚СЊСЃСЏ СЃ API.')).finally(()=>{if(active)setChecking(false);});return()=>{active=false;};},[]);

  async function compareFiles(){
    if(!authorized){setMessage('РЎРЅР°С‡Р°Р»Р° РІРѕР№РґРёС‚Рµ РІ Р°РєРєР°СѓРЅС‚.');return;}
    if(!baseFile||!newFile){setMessage('Р—Р°РіСЂСѓР·РёС‚Рµ РёСЃС…РѕРґРЅСѓСЋ Рё РЅРѕРІСѓСЋ РІРµСЂСЃРёРё СЃРјРµС‚С‹.');return;}
    const form=new FormData();form.append('base',baseFile);form.append('new',newFile);setStatus('loading');setMessage('РЎСЂР°РІРЅРёРІР°РµРј РґРІРµ РІРµСЂСЃРёРё...');setResult(null);
    try{const response=await apiFetch('/v1/estimates/compare',{method:'POST',body:form});const data=await readJSON(response);if(response.status===401){setAuthorized(false);throw new Error('РЎРµСЃСЃРёСЏ РёСЃС‚РµРєР»Р°. Р’РѕР№РґРёС‚Рµ СЃРЅРѕРІР°.');}if(!response.ok)throw new Error(data.error||'РќРµ СѓРґР°Р»РѕСЃСЊ СЃСЂР°РІРЅРёС‚СЊ СЃРјРµС‚С‹');setResult(data);setStatus('done');setMessage('РЎСЂР°РІРЅРµРЅРёРµ СЃРѕС…СЂР°РЅРµРЅРѕ.');}catch(err){setStatus('error');setMessage(err.message||'РќРµ СѓРґР°Р»РѕСЃСЊ СЃСЂР°РІРЅРёС‚СЊ СЃРјРµС‚С‹');}
  }

  return <main className="page"><Nav/><section className="pageHero compact"><p className="eyebrow">РЎСЂР°РІРЅРµРЅРёРµ СЃРјРµС‚</p><h1>РЎСЂР°РІРЅРёС‚Рµ РґРІРµ РІРµСЂСЃРёРё СЃРјРµС‚С‹ РїРµСЂРµРґ СЃРѕРіР»Р°СЃРѕРІР°РЅРёРµРј Р±СЋРґР¶РµС‚Р°.</h1></section><section className="workspace">
    {checking&&<div className="card">РџСЂРѕРІРµСЂСЏРµРј СЃРµСЃСЃРёСЋ...</div>}
    {!checking&&!authorized&&<div className="emptyState"><h2>РќСѓР¶РЅРѕ РІРѕР№С‚Рё</h2><p>{message||'РЎСЂР°РІРЅРµРЅРёРµ РґРѕСЃС‚СѓРїРЅРѕ С‚РѕР»СЊРєРѕ Р·Р°СЂРµРіРёСЃС‚СЂРёСЂРѕРІР°РЅРЅС‹Рј РїРѕР»СЊР·РѕРІР°С‚РµР»СЏРј.'}</p><a className="btn" href="/login">Р’РѕР№С‚Рё</a></div>}
    {!checking&&authorized&&<><div className="twoColumns"><label className="modernUploadZone"><input className="modernUploadInput" type="file" accept=".xls,.xlsx,.xlsm,.csv" onChange={e=>setBaseFile(e.target.files?.[0]||null)}/><div className="modernUploadContent"><h2>{baseFile?baseFile.name:'РСЃС…РѕРґРЅР°СЏ СЃРјРµС‚Р°'}</h2></div></label><label className="modernUploadZone"><input className="modernUploadInput" type="file" accept=".xls,.xlsx,.xlsm,.csv" onChange={e=>setNewFile(e.target.files?.[0]||null)}/><div className="modernUploadContent"><h2>{newFile?newFile.name:'РќРѕРІР°СЏ РІРµСЂСЃРёСЏ'}</h2></div></label></div><div className="card"><button className="btn" onClick={compareFiles} disabled={status==='loading'}>{status==='loading'?'РЎСЂР°РІРЅРёРІР°РµРј...':'РЎСЂР°РІРЅРёС‚СЊ СЃРјРµС‚С‹'}</button>{message&&<p className={`statusText ${status}`}>{message}</p>}</div></>}
  </section>{authorized&&result&&<section className="workspace"><div className="compareSummary"><div><strong>{money(result.base_total)}</strong><span>Р‘С‹Р»Рѕ</span></div><div><strong>{money(result.new_total)}</strong><span>РЎС‚Р°Р»Рѕ</span></div><div><strong>{money(result.delta_total)}</strong><span>Р Р°Р·РЅРёС†Р°</span></div></div><div className="twoColumns"><div className="card"><h2>Р”РѕР±Р°РІР»РµРЅРѕ</h2>{(result.added||[]).map(item=><p key={`a-${item.row}`}>{item.name} В· {money(item.total)}</p>)}</div><div className="card"><h2>РЈРґР°Р»РµРЅРѕ</h2>{(result.removed||[]).map(item=><p key={`r-${item.row}`}>{item.name} В· {money(item.total)}</p>)}</div></div></section>}<Footer/></main>;
}
