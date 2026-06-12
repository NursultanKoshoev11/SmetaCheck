import '../styles.css'
import '../premium.css'
import '../upload-modern.css'
import '../report-detail.css'
import '../saas-polish.css'
import '../marketing.css'
import {useRouter} from 'next/router'
import DocumentConsentGate from '../components/DocumentConsentGate'

export default function App({Component,pageProps}){
  const router=useRouter()
  const needsDocumentConsent=router.pathname==='/upload'||router.pathname==='/compare'
  if(needsDocumentConsent)return <DocumentConsentGate><Component {...pageProps}/></DocumentConsentGate>
  return <Component {...pageProps}/>
}
