# Visual Design Guide — IMAXX Smart Office

สรุป visual language ของเว็บนี้แบบละเอียด สำหรับเอาไป prompt สร้างโปรเจคอื่นให้มี "หน้าตา" แบบเดียวกัน

> **หัวใจของสไตล์นี้ (อ่านก่อน):** เว็บนี้ใช้ Bootstrap 5 เป็นโครง แต่ **ลบกลิ่น Bootstrap default ออกเกือบหมด** แล้วทับด้วยภาษาของ **Ant Design (enterprise SaaS)** + **Stripe (หน้า login)** ผลลัพธ์คือ dashboard สะอาด เงานุ่มเป็นชั้น มุมโค้งพอดี สีฟ้าองค์กรคุมโทน ไม่ใช่ปุ่มฟ้าจัด เงาดำหนา มุมเหลี่ยม แบบ Bootstrap สำเร็จรูป
>
> ถ้าจะให้ "ไม่เหมือนโปรเจคจบเด็กมหาลัย" ให้ยึด 5 ข้อนี้เป็นหลัก — ดูหัวข้อ [Why it doesn't look like default Bootstrap](#why-it-doesnt-look-like-default-bootstrap) ท้ายไฟล์

---

## 1. Design Principles (ปรัชญา)

1. **Enterprise calm, not vibrant.** สีหลักเป็นฟ้าองค์กรเข้ม (`#0061A8`) ไม่ใช่ฟ้าสดของ Bootstrap (`#0d6efd`) พื้นหลังเทาอ่อนมาก (`#F9FAFB`) การ์ดขาว ตัวหนังสือเทาเข้ม ไม่ใช่ดำสนิท
2. **Soft layered shadows.** เงาทุกจุดเป็นแบบ "หลายชั้น นุ่ม โปร่ง" (Ant Design / Stripe style) ไม่ใช่ `box-shadow: 0 0 10px black` — นี่คือสิ่งที่แยก "ดูแพง" ออกจาก "ดู bootstrap"
3. **Restrained radius.** มุมโค้งเล็ก-กลาง สม่ำเสมอ: ปุ่ม 6px, การ์ด 8–16px, badge เป็น pill (999px) ไม่มีมุมเหลี่ยม 0px และไม่มีมุมโค้งมนเกินจนเป็นการ์ตูน
4. **Muted grays, not black.** ตัวหนังสือ/เส้นขอบใช้ palette เทาอมฟ้า (slate) `#374557`, `#5c5c5c`, `#94a3b8`, `#e5e7eb` แทนดำ/เทากลางของ Bootstrap
5. **Micro-interactions.** hover ยกการ์ดขึ้น (`translateY(-2px)`) + เงาเข้มขึ้น, ปุ่มกดจมลง (`translateY(1px)`), transition ทุกอย่างด้วย easing แบบ Ant (`cubic-bezier(0.645, 0.045, 0.355, 1)`)
6. **Accent bars over fills.** state active ใช้แถบสีบางๆ ด้านข้าง/ด้านบน (sidebar active มีแถบ 3px, KPI card มี `border-top: 3px`) แทนการถมสีทั้งก้อน

---

## 2. Color Tokens

กำหนดเป็น CSS variable ที่ `:root` — ใช้ผ่าน `var(--x)` เสมอ

### Brand
```css
--primary:        #0061A8;   /* ฟ้าองค์กร — สีหลักของทั้งระบบ */
--primary-hover:  #00518f;
--primary-dark:   #003a66;
--active-blue:    #4A83FC;   /* ฟ้าสด — ใช้กับ state active / interactive (sidebar, toggle, focus) */

--secondary:      #EE3E23;   /* แดง-ส้ม — accent/brand ตัวที่สอง (โลโก้, ปุ่ม danger) */
--secondary-hover:#d9361f;
--secondary-dark: #b52c18;

--title:          #374557;   /* สีหัวข้อ/heading (slate เข้ม) */
```

มี alpha ramp ครบ 10 ระดับทั้ง primary และ secondary:
```css
--rgba-primary-1 … 9   /* rgba(0,97,168, 0.1 … 0.9) — ใช้ tint พื้นหลัง hover/row */
--rgba-secondary-1 … 9 /* rgba(238,62,35, 0.1 … 0.9) */
```
เช่น table row hover ใช้ `--rgba-primary-1` (ฟ้าจางมาก) ปุ่ม outline hover ก็ใช้ tint 0.1 เดียวกัน

### Neutrals / Grays (สำคัญ — นี่คือกลิ่นของงาน)
```
พื้นหลังหน้า      #F9FAFB   (เทาอมฟ้า อ่อนมาก)
พื้นการ์ด/พื้นขาว  #FFFFFF
เส้นขอบอ่อน       #f0f0f0 / #f0f2f7   (เส้นในตาราง/การ์ด บางมาก)
เส้นขอบกลาง       #e5e7eb / #e9ecef / #dee2e6
ตัวหนังสือหลัก    #212529
ตัวหนังสือรอง     #5c5c5c   (body text, p, table cell)
ตัวหนังสือ muted   #6c757d / #94a3b8   (label, caption, sub)
ตัวหนังสือจางสุด   #adb5bd / #cbd5e1   (placeholder, disabled)
heading/label     #374557 / #374151 / #334155   (slate)
```

### Semantic (badge / status)
ใช้คู่ **พื้นอ่อน + ตัวหนังสือเข้ม** โทนเดียวกัน (Tailwind-ish palette) — **ห้าม** ใช้ badge ทึบสีจัดแบบ Bootstrap
```
success  bg #dcfce7  text #166534
warning  bg #fef9c3  text #854d0e
error    bg #fee2e2  text #991b1b
info     bg #dbeafe  text #1e40af
active   bg #ecf9e5  text #67c23a
inactive bg #fdd4d4  text #f56c6c
noti/red #ef4444 (pill), #E81010 (dot)
```

### KPI / chart accent set
```
primary #0061A8 · success #198754 · warning #ffc107 · danger #dc3545 · purple #a855f7 · teal #14b8a6
```
แต่ละสีมี "tint" พื้นหลังอ่อนๆ ทำด้วย `color-mix(in srgb, <color> 12%, #fff)` ใช้เป็นพื้นไอคอนวงกลม/สี่เหลี่ยมมน

---

## 3. Typography

```css
/* หลัก */
font-family: "Sarabun", sans-serif;      /* body — รองรับไทยดี */
/* โลโก้ / heading บางจุด */
Poppins (300–800)                          /* --font-family-title */
/* fonts ที่ import เผื่อไว้: Prompt, Kanit, IBM Plex Sans Thai */
```
- **Body**: Sarabun 400, `line-height: 1.5`, สี `#212529`
- **Heading/title**: weight 600–800, สี `#374557` (slate) ไม่ใช่ดำ, มัก `letter-spacing: -0.01em ถึง -0.02em` (แน่นนิดๆ ให้ดู modern)
- **ตัวเลขใหญ่ (hero number ใน card)**: `font-size: 40px; font-weight: 800; line-height: 1; letter-spacing: -0.02em`
- **Label เล็ก / section header**: `font-size: 10–11px; font-weight: 700; letter-spacing: 1.5–2px; text-transform: uppercase; color: #94a3b8` — pattern "eyebrow label" สีเทาจาง ตัวห่าง (ใช้ใน sidebar section, logo subtitle)
- **Card title**: `14px / 600 / #374557`
- **Table header**: `0.875rem / 500`

**สเกลที่เจอบ่อย:** 10px (eyebrow) · 11–12px (caption/badge) · 13–14px (body/label) · 15–16px (card title) · 22px (KPI value) · 40px (hero number)

---

## 4. Shape & Elevation (radius + shadow) — *ตัวชี้เป็นชี้ตาย*

### Border radius (สม่ำเสมอ)
```
ปุ่ม             6px  (btn-sm = 4px)
input / select   6–8px
badge / pill     999px (เม็ดยา)
การ์ดทั่วไป       8px
การ์ดเด่น         12–16px  (dashboard card, KPI, login)
ไอคอน chip       10–12px  (สี่เหลี่ยมมุมมน ใส่ไอคอน)
modal            8px
popover/dropdown 12px
```

### Shadows (คัดลอกไปใช้ตรงๆ ได้เลย)
```css
/* การ์ด default — บางมาก เกือบมองไม่เห็น */
box-shadow: 0 1px 2px 0 rgba(0,0,0,0.03);

/* การ์ด hover — เงาหลายชั้นแบบ Ant Design */
box-shadow: 0 1px 2px -2px rgba(0,0,0,0.16),
            0 3px 6px 0 rgba(0,0,0,0.12),
            0 5px 12px 4px rgba(0,0,0,0.09);

/* dashboard card hover (ลิฟต์ขึ้น) */
box-shadow: 0 6px 20px rgba(0,0,0,0.09);   /* + transform: translateY(-2px) */

/* ปุ่ม — เงาจางบางเฉียบ ให้รู้สึกลอยนิดเดียว */
box-shadow: 0 2px 0 rgba(0,0,0,0.015);            /* .btn */
box-shadow: 0 2px 0 rgba(0,97,168,0.15);          /* .btn-primary (เงาสีตัวเอง) */

/* focus ring — ฟ้าจางมาก ไม่ใช่ ring หนา */
box-shadow: 0 0 0 0.25rem rgba(231,241,255,0.4);

/* login card — Stripe style, เงา 2 ชั้นนุ่มลึก */
box-shadow: 0 15px 35px rgba(50,50,93,0.1),
            0 5px 15px rgba(0,0,0,0.07);

/* modal / popover */
box-shadow: 0 6px 16px rgba(0,0,0,0.08),
            0 3px 6px -4px rgba(0,0,0,0.12),
            0 9px 28px 8px rgba(0,0,0,0.05);

/* sidebar (เงาแนวนอนไปขวา) */
box-shadow: 4px 0 24px rgba(0,0,0,0.04);
```
**กฎ:** ถ้าอยากได้ elevation เพิ่ม → เพิ่ม "จำนวนชั้น" และ spread ของเงา ไม่ใช่เพิ่มความเข้ม opacity ชั้นเดียว

### Transition / easing
```css
/* ปุ่ม / interactive แบบ Ant */
transition: all 0.2s cubic-bezier(0.645, 0.045, 0.355, 1);
/* layout (sidebar เปิด-ปิด) */
transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
/* progress bar / modal entrance */
cubic-bezier(0.16, 1, 0.3, 1)   /* ease-out นุ่ม */
```
Micro-interaction: `.btn:active { transform: translateY(1px) }` (กดจม), card hover `translateY(-2px)` (ลอยขึ้น) — ทุกที่ต้อง respect `prefers-reduced-motion`

---

## 5. Component Patterns

### Card
```
พื้นขาว · radius 8px · border 1px #f0f0f0 · shadow 0 1px 2px rgba(0,0,0,.03)
hover → เงาหลายชั้น (ดูข้างบน) · transition box-shadow 0.3s
card-header: พื้นโปร่ง · border-bottom 1px #f0f0f0 · weight 600 · color rgba(0,0,0,.88)
```
การ์ดเด่น (dashboard/KPI) ดันขึ้นเป็น radius 12–16px + มี accent bar (`border-top: 3px <สี>`)

### Button (override Bootstrap เต็มตัว)
```
radius 6px · font-weight 500 · inline-flex + center (ไอคอนกับ text อยู่กลางเสมอ)
:active → translateY(1px)
btn-primary  = พื้น --primary + เงาสีฟ้าจาง
btn-outline  = hover ใช้ tint --rgba-primary-1 (พื้นฟ้าจางมาก) ไม่ถมทึบ
```
มี variant สำเร็จ: `.btn-younger-primary` / `.btn-search` (ฟ้าพาสเทล `#e7f1ff`), `.btn-search-cancel` (เทา outline)

### Badge = pill เสมอ
```
padding 2px 10px · radius 999px · 12px · weight 500
พื้นอ่อน + ตัวเข้มโทนเดียวกัน (ดู semantic palette ข้อ 2)
```
**นี่คือจุดที่คนมักพลาด** — Bootstrap badge default เป็นสี่เหลี่ยมทึบ ต้องเปลี่ยนเป็น pill พื้นพาสเทล

### Table (custom ทั้งหมด ไม่พึ่ง .table ของ Bootstrap)
```
พื้นขาว · cell padding 12px 14px · border-bottom 1px #f0f2f7 (เส้นบางมาก ไม่มีเส้นตั้ง)
th: พื้น #F9FAFB · 0.875rem · weight 500 · color #5c5c5c
row hover: พื้น --rgba-primary-1 (ฟ้าจาง)
```
Pagination: ปุ่มเลข 32×32 กล่องมุมมน 6px, active = border+text สีฟ้า พื้นขาว (ไม่ถมสี), แบบ Ant Design

### Sidebar (SaaS-style — ตัวชูโรง)
```
กว้าง 260px (mini 90px) · พื้นขาว · ไม่มี border · shadow 4px 0 24px rgba(0,0,0,.04)
เมนู: radius 8px, สี #4b5563, hover พื้น #f3f4f6
active: พื้น #EFF6FF + text #4A83FC + แถบ accent 3px ด้านซ้าย (border-radius 0 3px 3px 0)
section header: eyebrow label (10px, uppercase, letter-spacing 1.5px, #94a3b8) + เส้นคั่นด้านขวา
submenu: มีเส้นตั้งด้านซ้าย (border-left 1px #E5E7EB) เหมือน tree
mini mode: submenu เด้งเป็น popout การ์ดลอย radius 12px เงานุ่ม
noti badge: pill แดง #ef4444 เงา
```

### Navbar
```
fixed top · สูง ~58.73px · พื้น rgba(255,255,255,0.8) (โปร่งแสง จะดู glassy) · border-bottom 1px #E5E7EB
toggle: ปุ่มกลม 40px สีฟ้า #4A83FC hover พื้น #EFF6FF
โลโก้: 2 คำ — คำแรก weight 900 สี #155DFC (ฟ้า), คำสอง weight 600 สี #334155 (slate)
     + subtitle eyebrow (10px uppercase letter-spacing 2px #94a3b8)
```

### Modal — Bootstrap 5 native (โครงสร้างบังคับในโปรเจคนี้)
```
radius 8px · header/footer padding 16px 24px · body padding 24px
title 16px/600 · เส้นคั่น 1px #f0f0f0
backdrop: rgba(0,0,0,0.45) + backdrop-filter blur(2px)
entrance: fade + scale(0.95)→1 translateY(-10px)→0, 0.2s
```

### KPI / Stat card
```
radius 16px · padding 20px · border 1px
ไอคอน chip 48px radius 12px พื้น = tint ของสี variant (color-mix 12%)
value 22px/700 · label 16px/600 · sub 12px muted
variant: primary/success/warning/danger/purple/teal (แต่ละอันมีสี + tint คู่กัน)
```

### Dashboard leave card (`.lc`)
```
radius 14px · border 1px #e9ecef + border-top 3px accent (สีต่อ card)
hover: translateY(-2px) + shadow 0 6px 20px rgba(0,0,0,.09)
hero number 40px/800 · progress track 5px radius 999px
```

### Form / input
```
focus-visible: outline 2px #4A83FC + offset 2px  (accessibility)
ลบ spinner ของ number input ออก
login input: radius 8px · border #e0e6ed · focus → ring rgba(74,131,252,.15) 3px + border #4A83FC
```

### react-select (dropdown)
ใช้ `react-select` ตรงๆ + `classNamePrefix="ss"`, style ผ่าน `searchableSelect.css` ให้เข้าธีม `var(--primary)` — ไม่ใช้ native `<select>` สำหรับ dropdown ที่ต้อง search

### Loader
Overlay `rgba(255,255,255,0.65) + backdrop-filter blur(4px)`, วงกลม spinner 60px ขอบฟ้าอ่อน `#e3f2fd` + top `#2196f3`

### Timeline (log history)
เส้นตั้งด้านซ้าย `border-left 2px #2379fc` + จุดกลมสีตามสถานะ (approve เขียว / submit ฟ้า / cancel แดง / draft เทา) จุดล่าสุดมี glow ring `box-shadow 0 0 0 3px rgba(...,.2)`

---

## 6. Layout System

- **Shell**: Navbar fixed บนสุด (z 1000) + Sidebar fixed ซ้าย (260px) + content block เลื่อนตามความกว้าง sidebar
- **Content padding**: `90px 25px 25px 25px` (เผื่อ navbar)
- **Grid การ์ด**: `grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 14–20px` — responsive เอง
- **Section spacing**: การ์ด/section ห่างกัน 24–28px, section title เป็น 14px/600 slate มี header row (title ซ้าย + action ขวา)
- **z-index scale** (ใช้ token เสมอ ห้าม magic number):
  ```
  --z-sticky 100 · --z-dropdown 1000 · --z-modal-backdrop 1040 · --z-modal 1050
  --z-toast 1080 · --z-loader 1090 · --z-dialog 20000
  ```

---

## 7. Copy-paste Design Token Block

เอาไปวางเป็นจุดตั้งต้นของโปรเจคใหม่ได้เลย:

```css
:root {
  /* brand */
  --primary: #0061A8;
  --primary-hover: #00518f;
  --primary-dark: #003a66;
  --active-blue: #4A83FC;
  --secondary: #EE3E23;
  --title: #374557;

  /* surfaces */
  --bg-page: #F9FAFB;
  --bg-card: #FFFFFF;
  --border-faint: #f0f0f0;
  --border: #e5e7eb;

  /* text */
  --text: #212529;
  --text-muted: #5c5c5c;
  --text-subtle: #94a3b8;

  /* radius */
  --r-btn: 6px;
  --r-card: 8px;
  --r-card-lg: 16px;
  --r-pill: 999px;

  /* elevation */
  --shadow-card: 0 1px 2px 0 rgba(0,0,0,0.03);
  --shadow-card-hover: 0 1px 2px -2px rgba(0,0,0,0.16), 0 3px 6px 0 rgba(0,0,0,0.12), 0 5px 12px 4px rgba(0,0,0,0.09);
  --shadow-lift: 0 6px 20px rgba(0,0,0,0.09);
  --shadow-modal: 0 6px 16px rgba(0,0,0,0.08), 0 3px 6px -4px rgba(0,0,0,0.12), 0 9px 28px 8px rgba(0,0,0,0.05);
  --shadow-stripe: 0 15px 35px rgba(50,50,93,0.1), 0 5px 15px rgba(0,0,0,0.07);

  /* motion */
  --ease-ant: cubic-bezier(0.645, 0.045, 0.355, 1);
  --ease-layout: cubic-bezier(0.4, 0, 0.2, 1);
}
body { background: var(--bg-page); color: var(--text); font-family: "Sarabun", sans-serif; line-height: 1.5; }
```

---

## Why it doesn't look like default Bootstrap

เช็กลิสต์กันหน้าตา "เด็กจบใหม่ใช้ Bootstrap" — ทำ 5 ข้อนี้ก่อนเป็นอันดับแรก:

| ปัญหาของ Bootstrap default | ทำแบบนี้แทน |
|---|---|
| ปุ่มฟ้าสด `#0d6efd` มุมโค้ง 0.375rem เงาไม่มี | primary เป็นฟ้าองค์กรเข้ม `#0061A8`, radius 6px, เงาสีจางบางเฉียบ |
| การ์ดเงาดำหนา / ไม่มีเงา ขอบดำ | เงา **หลายชั้นนุ่มโปร่ง** (Ant/Stripe), ขอบ `#f0f0f0` บางมาก, hover ยกขึ้น |
| badge สี่เหลี่ยมทึบสีจัด | **pill พื้นพาสเทล + ตัวเข้มโทนเดียวกัน** (`#dcfce7`/`#166534`) |
| ตัวหนังสือดำสนิท `#000` | slate/gray: heading `#374557`, body `#5c5c5c`, muted `#94a3b8` |
| พื้นหลังขาวโล่ง หรือเทา `#f8f9fa` แข็งๆ | พื้น `#F9FAFB` + การ์ดขาวลอยด้วยเงานุ่ม เป็นชั้นๆ |
| หน้าตานิ่ง ไม่มี feedback | hover ยกการ์ด, ปุ่มกดจม, transition easing แบบ Ant ทุกจุด |
| navbar/sidebar ทึบตัน | navbar โปร่งแสง `rgba(255,255,255,.8)`, sidebar active ใช้แถบ accent 3px + พื้นฟ้าจาง `#EFF6FF` ไม่ถมสี |
| ใช้ font default | Sarabun/Poppins, eyebrow label ตัวเล็ก uppercase letter-spacing กว้าง, heading letter-spacing แน่น |

**สรุปสั้นสุดสำหรับ prompt:** *"Enterprise SaaS dashboard, Ant Design + Stripe influence. Deep corporate blue `#0061A8` primary, off-white `#F9FAFB` background, white cards with soft multi-layer shadows and 8–16px radius, pill-shaped pastel status badges, slate-gray text (`#374557`/`#5c5c5c`), thin `#f0f0f0` borders, subtle hover lift + Ant easing micro-interactions, SaaS sidebar with 3px accent-bar active state. Not vibrant, not heavy — calm, layered, precise."*
