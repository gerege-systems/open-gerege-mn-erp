# Орчуулгын гарын авлага · Translation guide

Gerege Nexus-ийн хэрэглэгчид харагдах бүх текст `frontend/lib/i18n/`
дотор амьдардаг. Бүтэц нь Odoo-гийн орчуулгын загварыг дагасан: addon бүр
өөрийн нэр томьёог эзэмшиж, нийтлэг нэр томьёо `base` дотор нэг л удаа
тодорхойлогдоно.

---

## 1. Файлын бүтэц

```
frontend/lib/i18n/
  index.tsx          I18nProvider, useI18n, t(), хэлний бүртгэл
  base.ts            Бүх дэлгэцийн хуваалцдаг нэр томьёо  (Odoo "base")
  web.ts             Клиентийн бүрхүүл: цэс, толгой, хайлт (Odoo "web")
  addons/
    access.ts  ai.ts  app_store.ts  appearance.ts  auth.ts
    billing.ts contacts.ts developer.ts documents.ts esign.ts
    gov.ts     integrations.ts inventory.ts products.ts website.ts
```

App бүр өөрийн файлтай. Нэг л дэлгэц харуулдаг текст тухайн addon-д, хоёроос
дээш апп харуулдаг текст `base` эсвэл `web`-д очно.

## 2. Түлхүүрийн бүтэц

```
<module>.<kind>.<term>
```

`kind` нь Odoo-гийн нэр томьёоны ангиллыг дагана:

| kind | Юу вэ | Жишээ |
|---|---|---|
| `field` | Өгөгдлийн талбарын шошго | `base.field.status`, `gov.field.sla_hours` |
| `action` | Товч, үйлдэл | `base.action.save`, `gov.action.delegate` |
| `menu` | Навигацийн бичлэг | `web.menu.app_store`, `gov.menu.appointments` |
| `state` | Сонголтын утга (selection value) | `gov.state.awaiting_verification` |
| `view` | Дэлгэцийн гарчиг, тайлбар, placeholder | `gov.view.title`, `access.view.subtitle` |
| `message` | Хэрэглэгчид хэлэх зүйл: алдаа, мэдэгдэл, хоосон төлөв | `gov.message.no_child_unit` |

Мөн addon-д хамаарах тусгай ангилал байж болно: `gov.stat.*` (dashboard тоолуур),
`documents.category.*`, `integrations.type.*`, `appearance.mode.*`.

Term нь техникийн нэр — **snake_case**, англи хэл дээр, өгүүлбэр биш:
`gov.message.no_child_unit` ✓, `gov.noChildUnitAvailable` ✗.

## 3. Дүрмүүд

**Хоёр хэл заавал.** Dictionary нь typed тул `mn` эсвэл `en` дутвал TypeScript
compile алдаа өгнө. `npx tsc --noEmit` энэ шалгалтыг гүйцэтгэнэ.

**Давхардуулж болохгүй.** Нэг нэр томьёог хоёр модульд бүү тодорхойл. "Төлөв"
бол `base.field.status` — `contacts.status`, `products.status` гэж дахин
үүсгэхгүй. Ижил үг өөр утгатай бол (жишээ нь `base.action.close` = харилцах
цонх хаах, `gov.action.close` = хүсэлтийг хаах) тусад нь байх нь зөв.

**Ашиглагдахгүй түлхүүр байхгүй.** Дэлгэц харуулдаггүй текст dictionary-д
байхгүй. Хэрэгтэй болох үед нь нэмнэ.

**Дэлгэц дээр текст hardcode хийхгүй.** `locale === "en" ? "Save" : "Хадгалах"`
гэж бичихгүй — `t("base.action.save")`. Ингэснээр хэл солиход бүх дэлгэц зэрэг
солигдоно.

**Fallback нь англи.** Түлхүүр олдоогүй бол `t()` англи эх текстийг буцаана
(gettext-ийн адил), түлхүүрийн нэрийг биш.

## 4. Хувьсагч дамжуулах

```tsx
t("base.message.page_of", { page: 2, total: 7 })   // "2 / 7 хуудас"
t("access.message.confirm_delete", { name: role.name })
```

Текст дотор `{name}` хэлбэрээр бичнэ.

## 5. Динамик түлхүүр

API-аас ирсэн утгаар түлхүүр угсрахдаа техникийн хэлбэрт нь буулгана:

```tsx
// API "AWAITING_VERIFICATION" илгээдэг, dictionary-д lower snake_case байдаг
t(`gov.state.${status.toLowerCase()}` as never);
```

Ийм тохиолдолд `as never` шаардлагатай — TypeScript template literal-ыг
түлхүүр гэж таних боломжгүй. Тиймээс энэ хэв маягийг зөвхөн бүх утга нь
dictionary-д баттай байгаа үед хэрэглэнэ (`gov.state.*`, `gov.action.*`,
`gov.menu.*`).

## 6. Шинэ апп нэмэх үед

1. `frontend/lib/i18n/addons/<app>.ts` файл үүсгэнэ.
2. `export const <app> = { ... } as const;` гэж бичнэ.
3. `index.tsx`-д import хийж `dictionary` дотор spread хийнэ.
4. `npx tsc --noEmit` ажиллуулна.

## 7. Сервер талын текст

Дараах зүйлс dictionary-д **байхгүй** — эдгээрийг сервер орчуулна:

- **Цэсний шошго** — `backend/internal/platform/menu/menu.go` дахь blueprint ба
  модулийн `Menus()` дотор `Labels: map[string]string{"mn": ...}` талбараар.
  Хүсэлтийн `Accept-Language` header-ээр сонгогдоно.
- **Апп дэлгүүрийн тайлбар** — app catalog-ийн manifest дотор.
- **Байгууллагын өгөгдөл** — role-ийн нэр, үйлчилгээний нэр, нэгжийн нэр гэх мэт
  тенант өөрөө оруулдаг зүйлс. Эдгээрийн `*_en` талбарыг API буцаана.

Одоогийн цоорхой: **зөвшөөрлийн нэр, тайлбар** (`internal.PermissionDefinition`)
зөвхөн англиар зарлагддаг тул `/settings/access` дэлгэц дээр монгол горимд
англиар харагдана. Засах бол `PermissionDefinition`-д `Labels` талбар нэмж,
апп бүрийн `Permissions()`-д монгол нэр бичих хэрэгтэй — `MenuDefinition` яг
ийм хэв маягтай.
