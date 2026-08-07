# Баримт бичгийн төв — Documentation Hub

Энэ хавтас нь **Gerege Nexus**-ын бүх баримт бичиг болон орчуулгыг
агуулна. Үндсэн хэл нь монгол; орчуулгууд нь файлын нэрийн `_EN`, `_ZH`, `_RU`
дагаварт хадгалагдана.

<p>
  <a href="../README.md"><img src="assets/icons/flag-mn.png" width="18" height="18" alt=""> Монгол</a>
  &nbsp;·&nbsp;
  <a href="README_EN.md"><img src="assets/icons/flag-en.png" width="18" height="18" alt=""> English</a>
  &nbsp;·&nbsp;
  <a href="README_ZH.md"><img src="assets/icons/flag-zh.png" width="18" height="18" alt=""> 中文</a>
  &nbsp;·&nbsp;
  <a href="README_RU.md"><img src="assets/icons/flag-ru.png" width="18" height="18" alt=""> Русский</a>
</p>

---

## Танилцуулга — Overview

| Хэл | Файл |
| --- | --- |
| Монгол | [`../README.md`](../README.md) |
| English | [`README_EN.md`](README_EN.md) |
| 中文 | [`README_ZH.md`](README_ZH.md) |
| Русский | [`README_RU.md`](README_RU.md) |

## Техникийн баримт — Technical documentation

| Баримт | Хэл | Тайлбар |
| --- | --- | --- |
| [`ARCHITECTURE_SPECIFICATION.md`](ARCHITECTURE_SPECIFICATION.md) | MN | Платформын давхаргууд, өгөгдлийн загвар, архитектурын шийдвэрүүд |
| [`ARCHITECTURE_SPECIFICATION_EN.md`](ARCHITECTURE_SPECIFICATION_EN.md) | EN | Architecture specification |
| [`MODULE_AUTHORING_GUIDE.md`](MODULE_AUTHORING_GUIDE.md) | EN | Шинэ апп модуль хөгжүүлэх алхам алхмаар заавар |
| [`GOV_SERVICES_WORKFLOW.md`](GOV_SERVICES_WORKFLOW.md) | EN | Тохируулж болох төрийн үйлчилгээний урсгал, шилжүүлэлт, баталгаажуулалт |

## Төслийн журам — Project governance

| Баримт | Хэл | Тайлбар |
| --- | --- | --- |
| [`../CONTRIBUTING.md`](../CONTRIBUTING.md) | MN | Хувь нэмэр оруулах журам |
| [`CONTRIBUTING_EN.md`](CONTRIBUTING_EN.md) | EN | Contribution guide |
| [`../CODE_OF_CONDUCT.md`](../CODE_OF_CONDUCT.md) | MN | Ёс зүйн дүрэм |
| [`CODE_OF_CONDUCT_EN.md`](CODE_OF_CONDUCT_EN.md) | EN | Code of conduct |
| [`../SECURITY.md`](../SECURITY.md) | MN | Аюулгүй байдлын бодлого |
| [`SECURITY_EN.md`](SECURITY_EN.md) | EN | Security policy |
| [`../CHANGELOG.md`](../CHANGELOG.md) | EN | Өөрчлөлтийн түүх |

---

## Орчуулга нэмэх — Adding a translation

1. Эх баримтыг хуулж, файлын нэрэнд ISO 639-1 хэлний код бүхий дагавар нэмнэ:
   `README_JA.md`, `CONTRIBUTING_JA.md` гэх мэт.
2. Баримтын эхэнд, оршил догол мөрийн дараа, badge-үүдийн өмнө хэлний
   сонголтын мөрийг байрлуулна. Туг бүрийн зураг
   [`assets/icons/`](assets/icons/)-д хадгалагдана.
3. Бүх хэлний хувилбар дээрх сонголтын мөрийг шинэ хэлээр нөхнө — сонголт нь
   хэлүүдийн хооронд тэгш хэмтэй байх ёстой.
4. Энэ индекс файлын хүснэгтэд шинэ мөр нэмнэ.

Хэлний сонголтын мөрийн загвар (`docs/` доторх файлд):

```html
<p>
  <a href="../README.md"><img src="assets/icons/flag-mn.png" width="18" height="18" alt=""> Монгол</a>
  &nbsp;·&nbsp;
  <a href="README_EN.md"><img src="assets/icons/flag-en.png" width="18" height="18" alt=""> English</a>
</p>
```

Идэвхтэй байгаа хэлийг холбоосгүй, `<b>` тэгээр тодруулна.

---

## Дүрсний эх сурвалж — Icon source

Тугны дүрсийг [Flaticon](https://www.flaticon.com/)-оос авч репод хадгалсан.
Дэлгэрэнгүйг [`assets/icons/ATTRIBUTION.md`](assets/icons/ATTRIBUTION.md)-ээс
үзнэ үү. Баримт бичигт emoji дүрс ашиглахгүй — бүх дүрс нь Flaticon-ы дүрсийн
сангаас авсан зураг байна.
