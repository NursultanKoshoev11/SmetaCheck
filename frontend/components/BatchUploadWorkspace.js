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
        if(!response.ok)throw new Error(data.error||'Не удалось загрузить AI-провайдеры');
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
        if(!cancelled)setMessage(error.message||'Не удалось загрузить AI-провайдеры');
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
        if(!response.ok)throw new Error(data.error||'Не удалось получить статус пакета');
        if(cancelled)return;
        setBatch(data.batch);
        if(data.batch.status==='completed'){
          setStatus('done');
          router.push(`/batches/${data.batch.id}`);
        }else if(data.batch.status==='failed'){
          setStatus('error');
          setMessage(data.batch.error_message||'Анализ завершился с ошибкой');
        }else{
          setMessage(`Обработано ${data.batch.completed_count||0} из ${data.batch.file_count} файлов`);
        }
      }catch(error){
        if(!cancelled)setMessage(error.message||'Не удалось получить статус пакета');
      }
    };
    poll();
    const timer=window.setInterval(poll,2000);
    return()=>{cancelled=true;window.clearInterval(timer);};
  },[batch?.id,status,router]);

  async function submit(){
    if(!files.length){setMessage('Выберите хотя бы один файл');return;}
    const form=new FormData();
    files.forEach(file=>form.append('files',file));
    if(overrideAllowed&&provider){
      form.append('provider',provider);
      if(model)form.append('model',model);
    }
    setStatus('uploading');
    setMessage('Загружаем файлы одним запросом...');
    try{
      const {response,data}=await apiJSON('/v1/analysis-batches',{method:'POST',body:form});
      if(!response.ok)throw new Error(data.error||'Не удалось создать пакет');
      setBatch(data.batch);
      setStatus('processing');
      setMessage(`Пакет создан: ${data.batch.file_count} файлов · ${providerTitle(data.batch.provider)} · ${data.batch.model}`);
    }catch(error){
      setStatus('error');
      setMessage(error.message||'Не удалось создать пакет');
    }
  }

  function useDemo(){
    setFiles([
      new File(['Наименование,Ед,Количество,Цена,Сумма\nКирпич,шт,1000,12,12000\nЦемент,мешок,20,450,9000\n'],'demo-base.csv',{type:'text/csv'}),
      new File(['Наименование,Ед,Количество,Цена,Сумма\nКирпич,шт,1100,12,13200\nЦемент,мешок,20,480,9600\n'],'demo-new.csv',{type:'text/csv'})
    ]);
    setBatch(null);setStatus('idle');setMessage('Две demo-сметы готовы');
  }

  const busy=status==='uploading'||status==='processing';
  return <div className="twoColumns">
    <div className="uploadBox modernUploadShell">
      <div className="demoStrip"><div><b>Нет файлов?</b><p>Проверьте две demo-сметы.</p></div><button className="btn" type="button" onClick={useDemo}>Demo</button></div>
      <label className="modernUploadZone"><input className="modernUploadInput" type="file" multiple accept=".xlsx,.xlsm,.csv,.pdf" onChange={event=>{setFiles(Array.from(event.target.files||[]));setBatch(null);setStatus('idle');}}/><div className="modernUploadContent"><div className="modernUploadIcon">↑</div><span className="modernUploadHint">XLSX · XLSM · CSV · PDF</span><h2>{files.length?`Выбрано: ${files.length}`:'Выберите одну или несколько смет'}</h2><p>Все файлы отправляются одним запросом и получают общий отчёт.</p></div></label>
      {overrideAllowed&&providers.length>0&&<div className="card"><h3>AI-провайдер</h3><label>Провайдер<select value={provider} disabled={busy} onChange={event=>setProvider(event.target.value)}>{providers.map(item=><option key={item.name} value={item.name}>{providerTitle(item.name)}{item.raw_pdf?' · raw PDF':''}</option>)}</select></label><label>Модель<select value={model} disabled={busy||!selectedProvider?.models?.length} onChange={event=>setModel(event.target.value)}>{(selectedProvider?.models||[]).map(item=><option key={item} value={item}>{item}</option>)}</select></label></div>}
      {!!files.length&&<div className="card"><ul>{files.map((file,index)=><li key={`${file.name}-${index}`}>{file.name} — {sizeLabel(file.size)}</li>)}</ul><p>Общий размер: {sizeLabel(totalSize)}</p></div>}
      <button className="btn" type="button" disabled={busy} onClick={submit}>{busy?'Анализируем...':'Запустить общий анализ'}</button>
      {message&&<p className={`statusText ${status}`}>{message}</p>}
      {batch&&<p><b>Batch ID:</b> {batch.id}</p>}
    </div>
    <div className="card checklistCard"><h2>Единый анализ</h2><ul><li>Backend независимо проверяет все распознанные строки</li><li>AI независимо анализирует исходный PDF или все строки по chunks</li><li>Совпадения и расхождения сопоставляются после анализа</li><li>Строятся backend, AI и общий графики</li><li>Ошибка одного AI-файла не уничтожает весь пакет</li></ul></div>
  </div>;
}
