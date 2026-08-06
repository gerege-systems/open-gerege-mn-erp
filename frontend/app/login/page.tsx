"use client";
import {useEffect,useState} from "react";
import {useRouter} from "next/navigation";
import Link from "next/link";
import EIDLogin from "@/components/EIDLogin";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import {api} from "@/lib/api";
import {ChevronDown,Lock,Mail} from "lucide-react";

export default function LoginPage(){const router=useRouter();const [next,setNext]=useState("/apps"),[admin,setAdmin]=useState(false),[email,setEmail]=useState("admin@example.com"),[password,setPassword]=useState("Password123!"),[error,setError]=useState("");
  useEffect(()=>{const requested=new URLSearchParams(location.search).get("next");if(requested?.startsWith("/")&&!requested.startsWith("//"))setNext(requested)},[]);
  async function passwordLogin(e:React.FormEvent){e.preventDefault();setError("");try{const res=await api.login(email,password);localStorage.setItem("session_token",res.token);router.push(next)}catch(err:any){setError(err.message||"Нэвтрэх боломжгүй байна")}}
  return <main className="eid-page"><div className="eid-page__pattern"/><header><Link href="/" className="gp-brand"><img src="/brand.webp" alt=""/><span>Gerege ERP</span></Link><LanguageSwitcher variant="dark"/></header><section className="eid-page__content"><div className="eid-page__intro"><span className="gp-eyebrow"><i/> ҮНДЭСНИЙ ЦАХИМ ТАНИЛТ</span><h1>Баталгаатай identity.<br/><em>Нэг удаагийн</em> нэвтрэлт.</h1><p>eID Mongolia апп дээр хүсэлтийг зөвшөөрөхөд ERP болон холбогдсон SSO аппууд таны баталгаажсан session-ийг ашиглана.</p><ul><li>Регистрийн дугаараар push хүсэлт</li><li>QR болон mobile App2App</li><li>Tenant RBAC ба audit хамгаалалт</li></ul></div><div><EIDLogin next={next}/><button className="admin-disclosure" onClick={()=>setAdmin(v=>!v)}><Lock/> Системийн админ нэвтрэлт <ChevronDown className={admin?"rotate-180":""}/></button>{admin&&<form className="admin-login" onSubmit={passwordLogin}>{error&&<p>{error}</p>}<label><Mail/> <input type="email" value={email} onChange={e=>setEmail(e.target.value)} required/></label><label><Lock/> <input type="password" value={password} onChange={e=>setPassword(e.target.value)} required/></label><button>Админаар нэвтрэх</button></form>}</div></section></main>}
