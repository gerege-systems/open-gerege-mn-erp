"use client";

import { Check, Moon, Monitor, Palette, RotateCcw, Sun } from "lucide-react";
import { Accent, ColorMode, Density, DesignTheme, useTheme } from "@/lib/theme";
import { useI18n } from "@/lib/i18n";

// Labels resolve through the shared dictionary so this screen switches with
// the rest of the app instead of carrying its own inline translations.
const modes: { value: ColorMode; label: { mn: string; en: string }; icon: typeof Sun }[] = [
  { value: "light", label: { mn: "Гэгээн", en: "Light" }, icon: Sun },
  { value: "dark", label: { mn: "Харанхуй", en: "Dark" }, icon: Moon },
  { value: "system", label: { mn: "Системийн", en: "System" }, icon: Monitor },
];

const accents: { value: Accent; label: { mn: string; en: string }; color: string }[] = [
  { value: "neutral", label: { mn: "Цагаан саарал", en: "White & grey" }, color: "#64748b" },
  { value: "cobalt", label: { mn: "Gerege cobalt", en: "Gerege cobalt" }, color: "#0064e1" },
  { value: "teal", label: { mn: "Хөх ногоон", en: "Teal" }, color: "#008b99" },
  { value: "violet", label: { mn: "Нил ягаан", en: "Violet" }, color: "#7656d6" },
  { value: "emerald", label: { mn: "Маргад", en: "Emerald" }, color: "#16845b" },
];

export default function AppearanceSettingsPage() {
  const { locale } = useI18n();
  const en = locale === "en";
  const theme = useTheme();

  return (
    <div className="w-full space-y-6">
      <div className="border-b border-slate-200 pb-4 flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Palette className="w-6 h-6 text-[var(--gerege-blue)]" />
            {en ? "Appearance" : "Харагдац"}
          </h1>
          <p className="text-sm text-slate-500 mt-1">{en ? "Choose how Gerege ERP looks on this device." : "Энэ төхөөрөмж дээр Gerege ERP хэрхэн харагдахыг тохируулна."}</p>
        </div>
        <button onClick={theme.resetTheme} className="flex items-center gap-2 px-3 py-2 text-sm border border-slate-200 rounded-lg bg-white hover:bg-slate-50 text-slate-600">
          <RotateCcw className="w-4 h-4" />
          {en ? "Reset to defaults" : "Анхны төлөв"}
        </button>
      </div>

      <section className="bg-white border border-slate-200 rounded-[var(--gerege-radius-card)] p-5 shadow-sm">
        <h2 className="font-semibold text-base">{en ? "Theme style" : "Theme загвар"}</h2>
        <p className="text-sm text-slate-500 mt-1 mb-4">{en ? "Pick the original ERP look or the Gerege design system." : "ERP-ийн анхны харагдац эсвэл Gerege дизайн системийг сонгоно."}</p>
        <div className="grid sm:grid-cols-2 gap-4">
          {([
            { value: "original" as DesignTheme, name: "Original Theme", description: en ? "The original Odoo-style ERP interface" : "Odoo-маягийн анхны ERP интерфэйс", top: "#0f172a", accent: "#6366f1" },
            { value: "gerege" as DesignTheme, name: "Gerege Theme", description: en ? "The Gerege cobalt design system" : "Gerege-ийн cobalt дизайн систем", top: "#ffffff", accent: "#0064e1" },
          ]).map((item) => (
            <button key={item.value} onClick={() => theme.updateTheme({ design: item.value })} className={`overflow-hidden rounded-xl border text-left transition ${theme.design === item.value ? "border-[var(--gerege-blue)] ring-2 ring-[color-mix(in_srgb,var(--gerege-blue)_15%,transparent)]" : "border-slate-200 hover:border-slate-300"}`}>
              <span className="block h-12 border-b border-slate-200" style={{ background: item.top }}>
                <span className="block w-1/3 h-full border-r border-black/10" style={{ background: item.value === "original" ? "#ffffff" : "#f7f9fc" }} />
              </span>
              <span className="flex items-center gap-3 p-4 bg-white">
                <span className="w-3 h-8 rounded-full" style={{ background: item.accent }} />
                <span className="flex-1">
                  <strong className="block text-sm">{item.name}</strong>
                  <span className="block text-xs text-slate-500 mt-0.5">{item.description}</span>
                </span>
                {theme.design === item.value && <Check className="w-4 h-4 text-[var(--gerege-blue)]" />}
              </span>
            </button>
          ))}
        </div>
      </section>

      <section className="bg-white border border-slate-200 rounded-[var(--gerege-radius-card)] p-5 shadow-sm">
        <h2 className="font-semibold text-base">{en ? "Colour mode" : "Өнгөний горим"}</h2>
        <p className="text-sm text-slate-500 mt-1 mb-4">{en ? "Light, dark, or follow the device setting." : "Гэгээн, харанхуй эсвэл төхөөрөмжийн тохиргоог дагана."}</p>
        <div className="grid sm:grid-cols-3 gap-3">
          {modes.map(({ value, label, icon: Icon }) => (
            <button key={value} onClick={() => theme.updateTheme({ mode: value })} className={`relative flex items-center gap-3 p-4 rounded-lg border text-left transition ${theme.mode === value ? "border-[var(--gerege-blue)] bg-[var(--gerege-blue-soft)]" : "border-slate-200 hover:border-slate-300"}`}>
              <Icon className="w-5 h-5 text-[var(--gerege-blue)]" />
              <span className="font-medium text-sm">{label[locale]}</span>
              {theme.mode === value && <Check className="absolute right-3 w-4 h-4 text-[var(--gerege-blue)]" />}
            </button>
          ))}
        </div>
      </section>

      <section className="bg-white border border-slate-200 rounded-[var(--gerege-radius-card)] p-5 shadow-sm">
        <h2 className="font-semibold text-base">{en ? "Accent colour" : "Онцлох өнгө"}</h2>
        <p className="text-sm text-slate-500 mt-1 mb-4">{en ? "Cobalt is the primary Gerege brand colour." : "Cobalt нь Gerege-ийн үндсэн брэнд өнгө."}</p>
        <div className="grid sm:grid-cols-2 gap-3">
          {accents.map((accent) => (
            <button key={accent.value} onClick={() => theme.updateTheme({ accent: accent.value })} className={`flex items-center gap-3 p-3 rounded-lg border text-left ${theme.accent === accent.value ? "border-[var(--gerege-blue)] bg-[var(--gerege-blue-soft)]" : "border-slate-200 hover:border-slate-300"}`}>
              <span className="w-8 h-8 rounded-lg shadow-inner" style={{ background: accent.color }} />
              <span className="font-medium text-sm flex-1">{accent.label[locale]}</span>
              {theme.accent === accent.value && <Check className="w-4 h-4 text-[var(--gerege-blue)]" />}
            </button>
          ))}
        </div>
      </section>

      <section className="bg-white border border-slate-200 rounded-[var(--gerege-radius-card)] p-5 shadow-sm">
        <h2 className="font-semibold text-base">{en ? "Display density" : "Дэлгэцийн нягтрал"}</h2>
        <div className="mt-4 inline-flex rounded-lg border border-slate-200 p-1 bg-slate-50">
          {(["comfortable", "compact"] as Density[]).map((density) => (
            <button key={density} onClick={() => theme.updateTheme({ density })} className={`px-4 py-2 rounded-md text-sm font-medium ${theme.density === density ? "bg-white text-[var(--gerege-blue)] shadow-sm" : "text-slate-500"}`}>
              {density === "comfortable" ? (en ? "Comfortable" : "Тав тухтай") : (en ? "Compact" : "Нягт")}
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}
