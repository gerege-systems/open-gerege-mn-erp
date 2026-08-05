"use client";

import React, { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import brandLogo from "@/public/brand.webp";
import { usePathname, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { useTheme } from "@/lib/theme";
import UserMenu from "@/components/UserMenu";
import AICopilot from "@/components/AICopilot";
import { Landmark, LayoutGrid, Settings, Users, Package, Boxes, Share2, CreditCard, FileText, Code2, Menu as MenuIcon, Palette, Building2, BrainCircuit, Search } from "lucide-react";

interface MenuItem { id:string; app_id?:string; app_name?:string; parent_id?:string; label:string; path?:string; icon:string; order:number }
interface AppNav { id:string; name:string; icon:string; path:string; menus:MenuItem[] }

const iconMap: Record<string, React.ReactNode> = {
  users:<Users className="w-5 h-5"/>, package:<Package className="w-5 h-5"/>, boxes:<Boxes className="w-5 h-5"/>,
  "credit-card":<CreditCard className="w-5 h-5"/>, "file-text":<FileText className="w-5 h-5"/>, code:<Code2 className="w-5 h-5"/>, landmark:<Landmark className="w-5 h-5"/>,
};
const PUBLIC_ROUTES=["/","/login"];

export default function Layout({children}:{children:React.ReactNode}){
  const [menus,setMenus]=useState<MenuItem[]>([]),[user,setUser]=useState<any>(null),[loading,setLoading]=useState(true);
  const [mobileOpen,setMobileOpen]=useState(false),[panelOpen,setPanelOpen]=useState(true);
  const [query,setQuery]=useState("");
  const pathname=usePathname(),router=useRouter(),{t,locale}=useI18n(),theme=useTheme();
  const isPublic=PUBLIC_ROUTES.includes(pathname);

  useEffect(()=>setPanelOpen(localStorage.getItem("gerege_sidebar_open")!=="false"),[]);
  useEffect(()=>{if(isPublic){setLoading(false);return}void(async()=>{try{const [u,m]=await Promise.all([api.getMe(),api.getMenus()]);setUser(u);setMenus(m||[])}catch{router.push("/login")}finally{setLoading(false)}})()},[pathname,router,isPublic,locale]);
  useEffect(()=>setMobileOpen(false),[pathname]);

  const apps=useMemo<AppNav[]>(()=>{
    const groups=new Map<string,MenuItem[]>();
    menus.filter(m=>m.app_id&&m.path).forEach(m=>groups.set(m.app_id!,[...(groups.get(m.app_id!)||[]),m]));
    return [...groups.entries()].map(([id,items])=>{const sorted=items.sort((a,b)=>a.order-b.order);return{id,name:sorted[0].label||sorted[0].app_name||id,icon:sorted[0].icon,path:sorted[0].path!,menus:sorted}});
  },[menus]);
  const selected=apps.find(app=>app.menus.some(m=>m.path&&pathname.startsWith(m.path)))||null;
  const platformActive=!selected;
  const searchIndex=useMemo(()=>[
    {label:t("shell.appStore"),app:locale==="en"?"Platform":"Платформ",path:"/apps",icon:"grid"},
    {label:t("shell.appearance"),app:locale==="en"?"Platform":"Платформ",path:"/settings/appearance",icon:"palette"},
    {label:t("shell.installedApps"),app:locale==="en"?"Platform":"Платформ",path:"/settings/apps",icon:"settings"},
    ...apps.flatMap(app=>app.menus.filter(m=>m.path).map(m=>({label:m.label,app:app.name,path:m.path!,icon:m.icon})))
  ],[apps,locale,t]);
  const results=query.trim()?searchIndex.filter(x=>(x.label+" "+x.app).toLocaleLowerCase().includes(query.trim().toLocaleLowerCase())).slice(0,8):[];

  function togglePanel(){if(window.matchMedia("(min-width:1024px)").matches){setPanelOpen(v=>{localStorage.setItem("gerege_sidebar_open",String(!v));return !v})}else setMobileOpen(v=>!v)}
  async function logout(){try{await api.logout()}catch{}localStorage.removeItem("session_token");router.push("/login")}
  const brandTitle=selected?.name||(locale==="en"?"Platform":"Платформ");

  if(pathname==="/login")return <main className="min-h-screen bg-slate-100 flex items-center justify-center">{children}</main>;
  if(pathname==="/")return <>{children}</>;
  if(loading)return <div className="min-h-screen flex items-center justify-center bg-slate-50 text-slate-500 font-medium">{t("shell.loadingPlatform")}</div>;

  const platformMenus=<>
    <NavLink href="/apps" active={pathname==="/apps"} icon={<LayoutGrid className="w-5 h-5"/>} label={t("shell.appStore")}/>
    <NavLink href="/settings/ai" active={pathname==="/settings/ai"} icon={<BrainCircuit className="w-5 h-5"/>} label="AI тохиргоо"/>
    <NavLink href="/settings/appearance" active={pathname==="/settings/appearance"} icon={<Palette className="w-5 h-5"/>} label={t("shell.appearance")}/>
    <NavLink href="/settings/apps" active={pathname==="/settings/apps"} icon={<Settings className="w-5 h-5"/>} label={t("shell.installedApps")}/>
    <NavLink href="/settings/integrations" active={pathname==="/settings/integrations"} icon={<Share2 className="w-5 h-5"/>} label={t("shell.integrations")}/>
  </>;

  return <div className="gerege-shell min-h-screen flex flex-col">
    <header className="gerege-topbar h-16 flex items-center border-b sticky top-0 z-50">
      <Link href="/apps" className="w-16 h-full shrink-0 grid place-items-center border-r border-[var(--gerege-border)]">
        {theme.design==="gerege"?<img src={brandLogo.src} width={36} height={36} alt="Gerege" className="w-9 h-9 rounded-lg shadow-sm"/>:<span className="original-brand-mark w-9 h-9 rounded-lg grid place-items-center"><Building2 className="w-6 h-6"/></span>}
      </Link>
      <div className={`h-full px-4 flex flex-col justify-center border-r border-[var(--gerege-border)] overflow-hidden transition-all duration-200 ${panelOpen?"w-56":"w-0 px-0"}`}>
        <span className="text-[11px] text-slate-500 truncate">{selected?.name||"Gerege ERP"}</span><strong className="text-[15px] text-slate-900 truncate">{brandTitle}</strong>
      </div>
      <div className="w-16 h-full shrink-0 border-r border-[var(--gerege-border)] grid place-items-center"><button onClick={togglePanel} className="grid place-items-center w-10 h-10 rounded-lg text-slate-600 hover:bg-slate-50" aria-label={locale==="en"?"Toggle menu":"Цэс нээх, хаах"}><MenuIcon className="w-5 h-5"/></button></div>
      <div className="hidden md:flex flex-1 items-center justify-center min-w-0 px-5 relative">
        <div className="relative w-full max-w-md"><Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400"/><input value={query} onChange={e=>setQuery(e.target.value)} onKeyDown={e=>{if(e.key==="Enter"&&results[0]){router.push(results[0].path);setQuery("")}}} placeholder={locale==="en"?"Search apps and menus...":"Апп, цэс хайх..."} className="w-full h-10 rounded-full border border-slate-200 bg-slate-100/80 pl-10 pr-4 text-sm outline-none focus:border-[var(--gerege-blue)] focus:ring-2 focus:ring-[color-mix(in_srgb,var(--gerege-blue)_15%,transparent)]"/>
          {results.length>0&&<div className="absolute top-12 inset-x-0 bg-white border border-slate-200 rounded-xl shadow-xl p-1.5 z-[70]">{results.map(item=><button key={item.path} onClick={()=>{router.push(item.path);setQuery("")}} className="w-full flex items-center gap-3 rounded-lg px-3 py-2.5 text-left hover:bg-[var(--gerege-surface-2)]"><span className="text-[var(--gerege-blue)]">{iconMap[item.icon]||<Search className="w-4 h-4"/>}</span><span className="min-w-0"><strong className="block text-sm truncate">{item.label}</strong><small className="text-slate-500 truncate">{item.app}</small></span></button>)}</div>}
        </div>
      </div>
      <div className="hidden sm:flex pr-4 lg:pr-6"><UserMenu user={user} onLogout={logout}/></div>
    </header>

    <div className="flex flex-1 min-h-0">
      {mobileOpen&&<button className="fixed inset-0 top-16 bg-slate-950/25 z-30 lg:hidden" onClick={()=>setMobileOpen(false)}/>}
      <div className={`gerege-sidebar fixed lg:static top-16 bottom-0 left-0 z-40 flex border-r transition-transform lg:translate-x-0 ${mobileOpen?"translate-x-0":"-translate-x-full"}`}>
        <nav className="w-16 shrink-0 py-3 flex flex-col items-center gap-2 border-r border-[var(--gerege-border)]">
          <AppRailLink href="/apps" active={platformActive} title={locale==="en"?"Platform":"Платформ"} icon={<LayoutGrid className="w-5 h-5"/>}/>
          {apps.map(app=><AppRailLink key={app.id} href={app.path} active={selected?.id===app.id} title={app.name} icon={iconMap[app.icon]||<Package className="w-5 h-5"/>}/>) }
        </nav>
        <aside className={`overflow-hidden transition-all duration-200 ${panelOpen||mobileOpen?"w-56":"w-0"}`}>
          <div className="w-56 py-5"><div className="px-4 mb-3 text-xs font-bold uppercase tracking-wider text-slate-700 truncate">{brandTitle}</div><nav className="space-y-1 px-2">
            {selected?selected.menus.map(m=><NavLink key={m.id} href={m.path!} active={pathname.startsWith(m.path!)} icon={iconMap[m.icon]||<Package className="w-5 h-5"/>} label={m.label}/>):platformMenus}
          </nav></div>
        </aside>
      </div>
      <main className="flex-1 p-4 sm:p-6 lg:p-8 overflow-y-auto min-w-0">{children}</main>
    </div><AICopilot/>
  </div>;
}

function AppRailLink({href,active,title,icon}:{href:string;active:boolean;title:string;icon:React.ReactNode}){return <Link href={href} title={title} aria-label={title} className={`w-11 h-11 rounded-xl grid place-items-center transition ${active?"bg-[var(--gerege-blue-soft)] text-[var(--gerege-blue)] shadow-sm":"text-slate-500 hover:bg-[var(--gerege-surface-2)] hover:text-slate-800"}`}>{icon}</Link>}
function NavLink({href,active,icon,label}:{href:string;active:boolean;icon:React.ReactNode;label:string}){return <Link href={href} className={`gerege-nav-link flex items-center gap-3 px-3 py-2.5 text-sm font-medium transition ${active?"gerege-nav-link-active font-semibold":""}`}><span className="gerege-nav-icon">{icon}</span><span>{label}</span></Link>}
