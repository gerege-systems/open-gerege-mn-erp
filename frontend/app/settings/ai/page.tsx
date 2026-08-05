"use client";
import {useEffect,useState} from "react";
import {api} from "@/lib/api";
import {BrainCircuit,BookOpen,Save,Plus} from "lucide-react";

type Prompt={key:string;content:string;active:boolean;global:boolean};
type Knowledge={id:string;title:string;content:string;source_url:string;updated_at:string};
export default function AISettings(){
 const [prompts,setPrompts]=useState<Prompt[]>([]),[knowledge,setKnowledge]=useState<Knowledge[]>([]),[notice,setNotice]=useState("");
 const [draft,setDraft]=useState({title:"",content:"",source_url:""});
 async function load(){try{const [p,k]=await Promise.all([api.getAIPrompts(),api.getAIKnowledge()]);const merged=new Map<string,Prompt>();p.forEach(x=>merged.set(x.key,x));setPrompts([...merged.values()]);setKnowledge(k)}catch(e){setNotice(e instanceof Error?e.message:"Алдаа гарлаа")}}
 useEffect(()=>{void load()},[]);
 async function save(p:Prompt){await api.updateAIPrompt(p.key,p.content,p.active);setNotice("AI prompt хадгалагдлаа")}
 async function add(){if(!draft.title.trim()||!draft.content.trim())return;await api.createAIKnowledge(draft);setDraft({title:"",content:"",source_url:""});setNotice("Мэдлэг нэмэгдлээ");await load()}
 return <div className="max-w-5xl mx-auto space-y-6"><div><p className="text-xs font-semibold uppercase tracking-widest text-indigo-600">Gemini AI</p><h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2"><BrainCircuit/>AI тохиргоо</h1><p className="text-sm text-slate-500">Тухайн байгууллагын AI хариулах хүрээ, заавар болон мэдлэгийн санг удирдана.</p></div>{notice&&<div className="rounded-lg bg-indigo-50 text-indigo-700 p-3 text-sm">{notice}</div>}
 <section className="bg-white border rounded-xl p-5 space-y-4"><h2 className="font-bold">System prompt</h2>{prompts.map((p,i)=><div key={p.key}><div className="flex justify-between mb-1"><label className="text-sm font-semibold">{p.key}</label><span className="text-xs text-slate-400">{p.global?"Үндсэн":"Байгууллагын"}</span></div><textarea rows={5} value={p.content} onChange={e=>setPrompts(v=>v.map((x,j)=>j===i?{...x,content:e.target.value,global:false}:x))} className="w-full border rounded-lg p-3 text-sm"/><button onClick={()=>void save(p)} className="mt-2 bg-indigo-600 text-white rounded-lg px-3 py-2 text-sm flex gap-2"><Save className="w-4 h-4"/>Хадгалах</button></div>)}</section>
 <section className="bg-white border rounded-xl p-5 space-y-4"><h2 className="font-bold flex gap-2"><BookOpen className="w-5 h-5"/>Мэдлэгийн сан</h2><div className="grid md:grid-cols-2 gap-3"><input placeholder="Гарчиг" value={draft.title} onChange={e=>setDraft({...draft,title:e.target.value})} className="border rounded-lg p-2"/><input placeholder="Эх сурвалж URL" value={draft.source_url} onChange={e=>setDraft({...draft,source_url:e.target.value})} className="border rounded-lg p-2"/><textarea placeholder="AI ашиглах баталгаатай мэдээлэл" value={draft.content} onChange={e=>setDraft({...draft,content:e.target.value})} rows={4} className="md:col-span-2 border rounded-lg p-2"/></div><button onClick={()=>void add()} className="bg-indigo-600 text-white rounded-lg px-3 py-2 text-sm flex gap-2"><Plus className="w-4 h-4"/>Мэдлэг нэмэх</button><div className="divide-y">{knowledge.map(k=><article key={k.id} className="py-3"><h3 className="font-semibold text-sm">{k.title}</h3><p className="text-xs text-slate-500 line-clamp-2">{k.content}</p></article>)}</div></section></div>
}
