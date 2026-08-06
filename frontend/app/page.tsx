"use client";

import Link from "next/link";
import EIDLogin from "@/components/EIDLogin";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import {ArrowRight,CheckCircle2,Fingerprint,KeyRound,Layers,Network,ShieldCheck,Waypoints} from "lucide-react";

const features=[
  {icon:Fingerprint,title:"Цахим үнэмлэхээр хормын дотор",body:"Регистрийн дугаараар eID апп руу хүсэлт илгээх, компьютер дээр QR уншуулах, утсан дээр App2App холбоосоор нэвтэрнэ."},
  {icon:Network,title:"Нэг нэвтрэлт — олон систем",body:"ERP-ийн баталгаажсан session нь OAuth2/OIDC provider-тэй нэг trust boundary ашиглана. Холбогдсон апп бүр дахин нэвтрүүлэх шаардлагагүй."},
  {icon:ShieldCheck,title:"Нууц үггүй, сервер талын хамгаалалт",body:"RP secret зөвхөн backend-д хадгалагдана. Browser-д identity credential ил гарахгүй, session token hash хэлбэрээр хадгалагдана."},
  {icon:Waypoints,title:"Апп ба вэбийн нэг урсгал",body:"Desktop cross-device, mobile same-device callback, push болон QR бүгд ижил start/poll contract-аар ажиллана."},
];

export default function LandingPage(){return <div className="gp-landing" id="top">
  <header className="gp-nav"><a href="#top" className="gp-brand"><img src="/brand.webp" alt=""/><span>Gerege ERP</span></a><nav><a href="#features">Боломжууд</a><a href="#trust">Аюулгүй байдал</a><a href="#technology">Технологи</a></nav><div className="gp-actions"><LanguageSwitcher variant="dark"/><Link href="/login" className="gp-gold">Нэвтрэх</Link></div></header>
  <main>
    <section className="gp-hero"><div className="gp-pattern"/><div className="gp-hero__inner"><div className="gp-copy"><span className="gp-eyebrow"><i/> ERP · eID · SSO</span><h1>Нэг нэвтрэлт — <em>таны бүх</em> бизнес апп</h1><p>Gerege ERP нь үндэсний цахим үнэмлэхэд суурилсан нэвтрэлтийг OIDC/SSO чадвартай нэгтгэв. Иргэн нэг удаа баталгаажаад эрхтэй бүх аппдаа найдвартай орно.</p><div className="gp-cta"><a href="#eid-login" className="gp-gold gp-gold--large">eID-ээр нэвтрэх <ArrowRight/></a><a href="#features" className="gp-outline">Боломжийг үзэх</a></div><div className="gp-stats"><span><b>eID</b>Баталгаат identity</span><span><b>OAuth2 · OIDC</b>Нээлттэй стандарт</span><span><b>SSO</b>Нэг session</span></div></div><div id="eid-login" className="gp-login-slot"><EIDLogin compact/></div></div></section>
    <section className="gp-section" id="features"><div className="gp-heading"><span>GEREGE IDENTITY LAYER</span><h2>Нэвтрэлт бол тусдаа дэлгэц биш, платформын суурь</h2><p>Gerege Platform-ийн батлагдсан урсгалыг ERP-ийн tenant, role, audit болон SSO загварт нэгтгэлээ.</p></div><div className="gp-grid">{features.map(({icon:Icon,title,body},i)=><article key={title} className={i===1?"gp-feature gp-feature--dark":"gp-feature"}><Icon/><h3>{title}</h3><p>{body}</p></article>)}</div></section>
    <section className="gp-trust" id="trust"><div><span className="gp-eyebrow gp-eyebrow--blue"><i/> ИДЭВХТЭЙ ХАМГААЛАЛТ</span><h2>Танилтаас эрх хүртэл нэг баталгааны гинж</h2><p>eID identity → серверийн session → tenant membership → RBAC → OIDC client. Алхам бүр сервер талд шалгагдана.</p></div><ul>{["httpOnly, SameSite session cookie","Tenant-аар тусгаарласан role ба permission","RP callback origin allowlist","Login ба access audit event"].map(x=><li key={x}><CheckCircle2/>{x}</li>)}</ul></section>
    <section className="gp-tech" id="technology"><div><Layers/><h3>Gerege ERP</h3><p>Business apps, tenant isolation, RBAC</p></div><ArrowRight/><div><Fingerprint/><h3>eID Mongolia</h3><p>Push, QR, App2App, verified identity</p></div><ArrowRight/><div><KeyRound/><h3>OIDC / SSO</h3><p>Connected applications, one session</p></div></section>
  </main><footer className="gp-footer"><span>© 2026 Gerege Systems · Gerege ERP</span><span>eID-д суурилсан · Нээлттэй стандарт · Secure by design</span></footer>
</div>}
