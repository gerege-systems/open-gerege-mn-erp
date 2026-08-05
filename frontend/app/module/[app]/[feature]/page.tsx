"use client";

import { useParams } from "next/navigation";
import { Clock3, Construction, Sparkles } from "lucide-react";

function title(value: string) { return decodeURIComponent(value).split("-").map((word) => word.charAt(0).toUpperCase()+word.slice(1)).join(" "); }

export default function ComingSoonModulePage() {
  const params=useParams<{app:string;feature:string}>();
  return <div className="w-full min-h-[calc(100vh-8rem)] grid place-items-center">
    <section className="w-full max-w-3xl bg-white border border-slate-200 rounded-2xl p-8 sm:p-12 text-center shadow-sm relative overflow-hidden">
      <div className="absolute inset-x-0 top-0 h-1 bg-[var(--gerege-blue)]"/>
      <div className="mx-auto w-16 h-16 rounded-2xl bg-[var(--gerege-blue-soft)] text-[var(--gerege-blue)] grid place-items-center mb-6"><Construction className="w-8 h-8"/></div>
      <p className="text-xs font-bold uppercase tracking-[0.2em] text-[var(--gerege-blue)]">{title(params.app)} · roadmap</p>
      <h1 className="mt-2 text-3xl sm:text-4xl font-bold text-slate-900">{title(params.feature)}</h1>
      <p className="mt-4 text-slate-500 max-w-xl mx-auto">Энэ модуль дараагийн хувилбарын хөгжүүлэлтийн төлөвлөгөөнд орсон. Menu бүтэц бэлэн бөгөөд дэлгэц, permission, workflow хэрэгжих үед энд идэвхжинэ.</p>
      <div className="mt-8 inline-flex items-center gap-2 rounded-full bg-slate-100 px-4 py-2 text-sm font-medium text-slate-600"><Clock3 className="w-4 h-4"/>Coming soon<Sparkles className="w-4 h-4 text-[var(--gerege-blue)]"/></div>
    </section>
  </div>;
}
