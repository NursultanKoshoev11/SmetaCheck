import {useEffect,useMemo,useState} from 'react';
import {useRouter} from 'next/router';
import {apiJSON} from '../lib/api';

function sizeLabel(bytes){
  if(bytes>=1024*1024)return `${(bytes/1024/1024).toFixed(1)} MB`;
  return `${Math.max(1,Math.round(bytes/1024))} KB`;
}

function providerTitle(name){
  if(name==='openai')return 'OpenAI';
  if(name==='gemini')return 'Google Gemini';
  if(name==='anthropic')return 'Anthropic Claude';
  return 'Backend rules';
}

export default function BatchUploadWorkspace(){
  const router=useRouter();
  const [files,setFiles]=useState([]);
  const [batch,setBatch]=useState(null);
  const [status,setStatus]=useState('idle');
  const [message,setMessage]=useState('');
  const [providers,setProviders]=useState([]);
  const [overrideAllowed,setOverrideAllowed]=useState(false);
  const [provider,setProvider]=useState('');
  const [model,setModel]=useState('');
  const totalSize=useMemo(()=>files.reduce((sum,file)=>sum+file.size,0),[files]);
  const selectedProvider=useMemo(()=>providers.find(item=>item.name===provider)||null,[providers,provider]);

  useEffect(()=>{
    let cancelled=false;
    async function loadProviders(){
      try{
        const {response,data}=await apiJSON('/v1/ai/providers');
        if(!response.ok)throw new Error(data.error||'РќРµ СѓРґР°Р»РѕСЃСЊ Р·Р°РіСЂСѓР·РёС‚СЊ AI-РїСЂРѕРІР°Р№РґРµСЂС‹');
        if(cancelled)return;
        const available=(data.providers||[]).filter(item=>item.available);
        setProviders(available);
        setOverrideAllowed(Boolean(data.override_allowed));
        const initial=available.find(item=>item.name===data.default_provider)||available[0];
        if(initial){
          setProvider(initial.name);
          setModel(initial.models?.[0]||'');
        }
      }catch(error){
        if(!cancelled)setMessage(error.message||'РќРµ СѓРґР°Р»РѕСЃСЊ Р·Р°РіСЂСѓР·РёС‚СЊ AI-РїСЂРѕРІР°Р№РґРµСЂС‹');
      }
    }
    loadProviders();
    return()=>{cancelled=true;};
  },[]);

  useEffect(()=>{
    if(!selectedProvider)return;
    if(!selectedProvider.models?.includes(model))setModel(selectedProvider.models?.[0]||'');
  },[selectedProvider,model]);

  useEffect(()=>{
    if(!batch?.id||status!=='processing')return;
    let cancelled=false;
    const poll=async()=>{
      try{
        const {response,data}=await apiJSON(`/v1/analysis-batches/${batch.id}`);
        if(!response.ok)throw new Error(data.error||'РќРµ СѓРґР°Р»РѕСЃСЊ РїРѕР»СѓС‡РёС‚СЊ СЃС‚Р°С‚СѓСЃ РїР°РєРµС‚Р°');
        if(cancelled)return;
        setBatch(data.batch);
        if(data.batch.status==='completed'){
          setStatus('done');
          router.push(`/batches/${data.batch.id}`);
        }else if(data.batch.status==='failed'){
          setStatus('error');
          setMessage(data.batch.error_message||'РђРЅР°Р»РёР· Р·Р°РІРµСЂС€РёР»СЃСЏ СЃ РѕС€РёР±РєРѕР№');
        }else{
          setMessage(`РћР±СЂР°Р±РѕС‚Р°РЅРѕ ${data.batch.completed_count||0} РёР· ${data.batch.file_count} С„Р°Р№Р»РѕРІ`);
        }
      }catch(error){
        if(!cancelled)setMessage(error.message||'РќРµ СѓРґР°Р»РѕСЃСЊ РїРѕР»СѓС‡РёС‚СЊ СЃС‚Р°С‚СѓСЃ РїР°РєРµС‚Р°');
      }
    };
    poll();
    const timer=window.setInterval(poll,2000);
    return()=>{cancelled=true;window.clearInterval(timer);};
  },[batch?.id,status,router]);

  async function submit(){
    if(!files.length){setMessage('Р’С‹Р±РµСЂРёС‚Рµ С…РѕС‚СЏ Р±С‹ РѕРґРёРЅ С„Р°Р№Р»');return;}
    const form=new FormData();
    files.forEach(file=>form.append('files',file));
    if(overrideAllowed&&provider){
      form.append('provider',provider);
      if(model)form.append('model',model);
    }
    setStatus('uploading');
    setMessage('Р—Р°РіСЂСѓР¶Р°РµРј С„Р°Р№Р»С‹ РѕРґРЅРёРј Р·Р°РїСЂРѕСЃРѕРј...');
    try{
      const {response,data}=await apiJSON('/v1/analysis-batches',{method:'POST',body:form});
      if(!response.ok)throw new Error(data.error||'РќРµ СѓРґР°Р»РѕСЃСЊ СЃРѕР·РґР°С‚СЊ РїР°РєРµС‚');
      setBatch(data.batch);
      setStatus('processing');
      setMessage(`РџР°РєРµС‚ СЃРѕР·РґР°РЅ: ${data.batch.file_count} С„Р°Р№Р»РѕРІ В· ${providerTitle(data.batch.provider)} В· ${data.batch.model}`);
    }catch(error){
      setStatus('error');
      setMessage(error.message||'РќРµ СѓРґР°Р»РѕСЃСЊ СЃРѕР·РґР°С‚СЊ РїР°РєРµС‚');
    }
  }

  function useDemo(){
    setFiles([
      new File(['РќР°РёРјРµРЅРѕРІР°РЅРёРµ,Р•Рґ,РљРѕР»РёС‡РµСЃС‚РІРѕ,Р¦РµРЅР°,РЎСѓРјРјР°\nРљРёСЂРїРёС‡,С€С‚,1000,12,12000\nР¦РµРјРµРЅС‚,РјРµС€РѕРє,20,450,9000\n'],'demo-base.csv',{type:'text/csv'}),
      new File(['РќР°РёРјРµРЅРѕРІР°РЅРёРµ,Р•Рґ,РљРѕР»РёС‡РµСЃС‚РІРѕ,Р¦РµРЅР°,РЎСѓРјРјР°\nРљРёСЂРїРёС‡,С€С‚,1100,12,13200\nР¦РµРјРµРЅС‚,РјРµС€РѕРє,20,480,9600\n'],'demo-new.csv',{type:'text/csv'})
    ]);
    setBatch(null);setStatus('idle');setMessage('Р”РІРµ demo-СЃРјРµС‚С‹ РіРѕС‚РѕРІС‹');
  }

  const busy=status==='uploading'||status==='processing';
  return <div className="twoColumns">
    <div className="uploadBox modernUploadShell">
      <div className="demoStrip"><div><b>РќРµС‚ С„Р°Р№Р»РѕРІ?</b><p>РџСЂРѕРІРµСЂСЊС‚Рµ РґРІРµ demo-СЃРјРµС‚С‹.</p></div><button className="btn" type="button" onClick={useDemo}>Demo</button></div>
      <label className="modernUploadZone"><input className="modernUploadInput" type="file" multiple accept=".xls,.xlsx,.xlsm,.csv,.pdf" onChange={event=>{setFiles(Array.from(event.target.files||[]));setBatch(null);setStatus('idle');}}/><div className="modernUploadContent"><div className="modernUploadIcon">в†‘</div><span className="modernUploadHint">XLS / XLSX / XLSM / CSV / PDF</span><h2>{files.length?`Р’С‹Р±СЂР°РЅРѕ: ${files.length}`:'Р’С‹Р±РµСЂРёС‚Рµ РѕРґРЅСѓ РёР»Рё РЅРµСЃРєРѕР»СЊРєРѕ СЃРјРµС‚'}</h2><p>Р’СЃРµ С„Р°Р№Р»С‹ РѕС‚РїСЂР°РІР»СЏСЋС‚СЃСЏ РѕРґРЅРёРј Р·Р°РїСЂРѕСЃРѕРј Рё РїРѕР»СѓС‡Р°СЋС‚ РѕР±С‰РёР№ РѕС‚С‡С‘С‚.</p></div></label>
      {overrideAllowed&&providers.length>0&&<div className="card"><h3>AI-РїСЂРѕРІР°Р№РґРµСЂ</h3><label>РџСЂРѕРІР°Р№РґРµСЂ<select value={provider} disabled={busy} onChange={event=>setProvider(event.target.value)}>{providers.map(item=><option key={item.name} value={item.name}>{providerTitle(item.name)}{item.raw_pdf?' В· raw PDF':''}</option>)}</select></label><label>РњРѕРґРµР»СЊ<select value={model} disabled={busy||!selectedProvider?.models?.length} onChange={event=>setModel(event.target.value)}>{(selectedProvider?.models||[]).map(item=><option key={item} value={item}>{item}</option>)}</select></label></div>}
      {!!files.length&&<div className="card"><ul>{files.map((file,index)=><li key={`${file.name}-${index}`}>{file.name} вЂ” {sizeLabel(file.size)}</li>)}</ul><p>РћР±С‰РёР№ СЂР°Р·РјРµСЂ: {sizeLabel(totalSize)}</p></div>}
      <button className="btn" type="button" disabled={busy} onClick={submit}>{busy?'РђРЅР°Р»РёР·РёСЂСѓРµРј...':'Р—Р°РїСѓСЃС‚РёС‚СЊ РѕР±С‰РёР№ Р°РЅР°Р»РёР·'}</button>
      {message&&<p className={`statusText ${status}`}>{message}</p>}
      {batch&&<p><b>Batch ID:</b> {batch.id}</p>}
    </div>
    <div className="card checklistCard"><h2>Р•РґРёРЅС‹Р№ Р°РЅР°Р»РёР·</h2><ul><li>Backend РЅРµР·Р°РІРёСЃРёРјРѕ РїСЂРѕРІРµСЂСЏРµС‚ РІСЃРµ СЂР°СЃРїРѕР·РЅР°РЅРЅС‹Рµ СЃС‚СЂРѕРєРё</li><li>AI РЅРµР·Р°РІРёСЃРёРјРѕ Р°РЅР°Р»РёР·РёСЂСѓРµС‚ РёСЃС…РѕРґРЅС‹Р№ PDF РёР»Рё РІСЃРµ СЃС‚СЂРѕРєРё РїРѕ chunks</li><li>РЎРѕРІРїР°РґРµРЅРёСЏ Рё СЂР°СЃС…РѕР¶РґРµРЅРёСЏ СЃРѕРїРѕСЃС‚Р°РІР»СЏСЋС‚СЃСЏ РїРѕСЃР»Рµ Р°РЅР°Р»РёР·Р°</li><li>РЎС‚СЂРѕСЏС‚СЃСЏ backend, AI Рё РѕР±С‰РёР№ РіСЂР°С„РёРєРё</li><li>РћС€РёР±РєР° РѕРґРЅРѕРіРѕ AI-С„Р°Р№Р»Р° РЅРµ СѓРЅРёС‡С‚РѕР¶Р°РµС‚ РІРµСЃСЊ РїР°РєРµС‚</li></ul></div>
  </div>;
}
