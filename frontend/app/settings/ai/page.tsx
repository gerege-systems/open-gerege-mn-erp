"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { BrainCircuit, BookOpen, Save, Plus } from "lucide-react";

type Prompt = { key: string; content: string; active: boolean; global: boolean };
type Knowledge = { id: string; title: string; content: string; source_url: string; updated_at: string };

export default function AISettings() {
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [knowledge, setKnowledge] = useState<Knowledge[]>([]);
  const [notice, setNotice] = useState("");
  const [draft, setDraft] = useState({ title: "", content: "", source_url: "" });

  async function load() {
    try {
      const [p, k] = await Promise.all([api.getAIPrompts(), api.getAIKnowledge()]);
      const merged = new Map<string, Prompt>();
      p.forEach((item) => merged.set(item.key, item));
      setPrompts([...merged.values()]);
      setKnowledge(k);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Алдаа гарлаа");
    }
  }

  useEffect(() => { void load(); }, []);

  async function save(prompt: Prompt) {
    await api.updateAIPrompt(prompt.key, prompt.content, prompt.active);
    setNotice("AI prompt хадгалагдлаа");
  }

  async function add() {
    if (!draft.title.trim() || !draft.content.trim()) return;
    await api.createAIKnowledge(draft);
    setDraft({ title: "", content: "", source_url: "" });
    setNotice("Мэдлэг нэмэгдлээ");
    await load();
  }

  return (
    <div className="w-full space-y-6">
      <header>
        <p className="text-xs font-semibold uppercase tracking-widest text-[var(--gerege-blue)]">Gemini AI</p>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2"><BrainCircuit />AI тохиргоо</h1>
        <p className="text-sm text-slate-500">Тухайн байгууллагын AI хариулах хүрээ, заавар болон мэдлэгийн санг удирдана.</p>
      </header>

      {notice && <div className="rounded-lg bg-[var(--gerege-blue-soft)] text-[var(--gerege-blue)] p-3 text-sm">{notice}</div>}

      <div className="grid grid-cols-1 xl:grid-cols-[minmax(0,1.35fr)_minmax(360px,.65fr)] gap-6 items-start">
        <section className="bg-white border border-slate-200 rounded-xl p-5 lg:p-6 space-y-5 shadow-sm min-w-0">
          <h2 className="font-bold">System prompt</h2>
          {prompts.map((prompt, index) => (
            <div key={prompt.key}>
              <div className="flex justify-between mb-1.5">
                <label className="text-sm font-semibold">{prompt.key}</label>
                <span className="text-xs text-slate-400">{prompt.global ? "Үндсэн" : "Байгууллагын"}</span>
              </div>
              <textarea rows={7} value={prompt.content} onChange={(event) => setPrompts((items) => items.map((item, itemIndex) => itemIndex === index ? { ...item, content: event.target.value, global: false } : item))} className="w-full border border-slate-200 rounded-lg p-3 text-sm bg-white" />
              <button onClick={() => void save(prompt)} className="mt-2 bg-[var(--gerege-blue)] hover:bg-[var(--gerege-blue-hover)] text-white rounded-lg px-3 py-2 text-sm flex gap-2"><Save className="w-4 h-4" />Хадгалах</button>
            </div>
          ))}
        </section>

        <section className="bg-white border border-slate-200 rounded-xl p-5 lg:p-6 space-y-4 shadow-sm min-w-0 xl:sticky xl:top-24">
          <h2 className="font-bold flex gap-2"><BookOpen className="w-5 h-5" />Мэдлэгийн сан</h2>
          <div className="grid gap-3">
            <input placeholder="Гарчиг" value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} className="border border-slate-200 rounded-lg p-2.5" />
            <input placeholder="Эх сурвалж URL" value={draft.source_url} onChange={(event) => setDraft({ ...draft, source_url: event.target.value })} className="border border-slate-200 rounded-lg p-2.5" />
            <textarea placeholder="AI ашиглах баталгаатай мэдээлэл" value={draft.content} onChange={(event) => setDraft({ ...draft, content: event.target.value })} rows={8} className="border border-slate-200 rounded-lg p-2.5" />
          </div>
          <button onClick={() => void add()} className="bg-[var(--gerege-blue)] hover:bg-[var(--gerege-blue-hover)] text-white rounded-lg px-3 py-2 text-sm flex gap-2"><Plus className="w-4 h-4" />Мэдлэг нэмэх</button>
          <div className="divide-y max-h-72 overflow-y-auto">
            {knowledge.map((item) => <article key={item.id} className="py-3"><h3 className="font-semibold text-sm">{item.title}</h3><p className="text-xs text-slate-500 line-clamp-2">{item.content}</p></article>)}
          </div>
        </section>
      </div>
    </div>
  );
}
