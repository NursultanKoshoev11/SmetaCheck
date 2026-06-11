import {useEffect, useMemo, useState} from 'react';
import {useRouter} from 'next/router';

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://localhost:8080';

function sizeLabel(bytes){
  if(bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

export default function BatchUploadWorkspace(){
  const router = useRouter();
  const [files,setFiles] = useState([]);
  const [batch,setBatch] = useState(null);
  const [status,setStatus] = useState('idle');
  const [message,setMessage] = useState('');
  const totalSize = useMemo(()=>files.reduce((sum,file)=>sum+file.size,0),[files]);

  useEffect(()=>{
    if(!batch?.id || status!=='processing') return;
    const token = window.localStorage.getItem('smetacheck_token');
    let cancelled = false;
    const poll = async()=>{
      try{
        const response = await fetch(`${API_BASE}/v1/analysis-batches/${batch.id}`,{headers:{Authorization:`Bearer ${token}`}});
        const data = await response.json();
        if(!response.ok) throw new Error(data.error || 'Не удалось получить статус пакета');
        if(cancelled) return;
        setBatch(data.batch);
        if(data.batch.status==='completed'){
          setStatus('done');
          router.push(`/batches/${data.batch.id}`);
        }else if(data.batch.status==='failed'){
          setStatus('error');
          setMessage(data.batch.error_message || 'Анализ завершился с ошибкой');
        }else{
          setMessage(`Обработано ${data.batch.completed_count || 0} из ${data.batch.file_count} файлов`);
        }
      }catch(error){ if(!cancelled) setMessage(error.message); }
    };
    poll();
    const timer = window.setInterval(poll,2000);
    return ()=>{cancelled=true;window.clearInterval(timer);};
  },[batch?.id,status,router]);

  async function submit(){
    const token = window.localStorage.getItem('smetacheck_token');
    if(!token){setMessage('Сначала войдите в аккаунт');return;}
    if(!files.length){setMessage('Выберите хотя бы один файл');return;}
    const form = new FormData();
    files.forEach(file=>form.append('files',file));
    setStatus('uploading');
    setMessage('Загружаем файлы одним запросом...');
    try{
      const response = await fetch(`${API_BASE}/v1/analysis-batches`,{method:'POST',headers:{Authorization:`Bearer ${token}`},body:form});
      const data = await response.json();
      if(!response.ok) throw new Error(data.error || 'Не удалось создать пакет');
      setBatch(data.batch);
      setStatus('processing');
      setMessage(`Пакет создан: ${data.batch.file_count} файлов`);
    }catch(error){setStatus('error');setMessage(error.message);}
  }

  function useDemo(){
    setFiles([
      new File(['Наименование,Ед,Количество,Цена,Сумма\nКирпич,шт,1000,12,12000\nЦемент,мешок,20,450,9000\n'],'demo-base.csv',{type:'text/csv'}),
      new File(['Наименование,Ед,Количество,Цена,Сумма\nКирпич,шт,1100,12,13200\nЦемент,мешок,20,480,9600\n'],'demo-new.csv',{type:'text/csv'})
    ]);
    setBatch(null);setStatus('idle');setMessage('Две demo-сметы готовы');
  }

  const busy = status==='uploading' || status==='processing';
  return <div className="twoColumns">
    <div className="uploadBox modernUploadShell">
      <div className="demoStrip"><div><b>Нет файлов?</b><p>Проверьте две demo-сметы.</p></div><button className="btn" type="button" onClick={useDemo}>Demo</button></div>
      <label className="modernUploadZone"><input className="modernUploadInput" type="file" multiple accept=".xlsx,.xlsm,.csv,.pdf" onChange={event=>{setFiles(Array.from(event.target.files || []));setBatch(null);setStatus('idle');}}/><div className="modernUploadContent"><div className="modernUploadIcon">↑</div><span className="modernUploadHint">XLSX · XLSM · CSV · PDF</span><h2>{files.length ? `Выбрано: ${files.length}` : 'Выберите одну или несколько смет'}</h2><p>Все файлы отправляются одним запросом и получают общий отчёт.</p></div></label>
      {!!files.length && <div className="card"><ul>{files.map((file,index)=><li key={`${file.name}-${index}`}>{file.name} — {sizeLabel(file.size)}</li>)}</ul><p>Общий размер: {sizeLabel(totalSize)}</p></div>}
      <button className="btn" type="button" disabled={busy} onClick={submit}>{busy?'Анализируем...':'Запустить общий анализ'}</button>
      {message && <p className={`statusText ${status}`}>{message}</p>}
      {batch && <p><b>Batch ID:</b> {batch.id}</p>}
    </div>
    <div className="card checklistCard"><h2>Единый анализ</h2><ul><li>Backend проверяет все распознанные строки</li><li>AI независимо анализирует данные по chunks</li><li>Совпадения и расхождения сопоставляются</li><li>Строятся backend, AI и общий графики</li><li>Несколько файлов сравниваются между собой</li></ul></div>
  </div>;
}
