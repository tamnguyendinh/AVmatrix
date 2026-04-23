# Design System — Editorial Warm Monochrome
**Codename:** `The Press`  
**Version:** `1.4`  
**Status:** Synced to Website UI SPEC authority after black-border sync, heavier-frame sync, and reading-prose split  
**Dùng cho:** Desktop App & Website

---

## 1. Thiết Kế Cốt Lõi

`The Press` không phải editorial theme trung tính sạch sẽ.  
Nó là hệ giao diện **giấy báo ấm, mực nâu-đen, khung viền đậm và rõ**, ưu tiên typography, frame rhythm, và contrast của surface thay vì màu nhấn.

### Hard rules

- Chỉ dùng **1 filled CTA** trên mỗi màn hình.
- Depth phải đến từ **tone shift + border rhythm + inset treatment**.
- Không dùng decorative shadow.
- Không dùng gray lạnh, white tinh, hoặc accent sáng kiểu SaaS.
- Light mode phải đọc như **giấy cũ pha nâu**.
- Major frame không được mỏng hoặc mờ; phải đọc rõ như khung in mực.

---

## 2. Exact Token Authority

### 2.1 Light mode

```css
--color-bg-base:        #EEE7DB; /* giấy nền chính */
--color-bg-surface:     #E4DCCF; /* card, panel, band */
--color-bg-inset:       #DCD2C3; /* input, filter shell, inset field */
--color-bg-overlay:     #1F1B18; /* backdrop, dark header, dark rail */

--color-border-subtle:  #6A5C4F; /* divider, row rule */
--color-border-default: #2F2822; /* major frame chuẩn */
--color-border-strong:  #1B1815; /* outer shell, active frame, strong focus edge */

--color-text-primary:   #1B1815; /* heading, body chính */
--color-text-secondary: #5F5448; /* support text, metadata */
--color-text-muted:     #968A7C; /* placeholder, low emphasis */
--color-text-inverse:   #EEE7DB; /* text trên nền tối hoặc filled CTA */

--color-accent:         #1B1815; /* filled CTA duy nhất */
--color-accent-hover:   #322C26;

--color-success:        #56684A;
--color-warning:        #8C6B2A;
--color-error:          #7C4035;
--color-info:           #465B70;
```

### 2.2 Dark mode

```css
--color-bg-base:        #1F1B18;
--color-bg-surface:     #29231F;
--color-bg-inset:       #312A25;

--color-border-subtle:  #5B4837;
--color-border-default: #7B634A;
--color-border-strong:  #9A7E63;

--color-text-primary:   #EEE7DB;
--color-text-secondary: #938674;
--color-text-muted:     #9A7E63;
--color-text-inverse:   #1F1B18;

--color-accent:         #EEE7DB;
```

### 2.3 Token interpretation

- Trong **light mode**:
  - divider thường dùng `#6A5C4F`
  - frame chuẩn dùng `#2F2822`
  - active shell, selected item, outer frame mạnh, focus edge mạnh dùng `#1B1815`
- Trong **dark mode**:
  - divider thường dùng `#5B4837`
  - frame chuẩn dùng `#7B634A`
  - frame nhấn mạnh dùng `#9A7E63`
- `#938674` chỉ là **dark support text token**, không phải light-mode border token.

---

## 3. Typography Authority

```css
--font-display: 'Playfair Display', 'IM Fell English', Georgia, serif;
--font-body:    'IBM Plex Mono', 'Courier New', monospace;
--font-reading: Georgia, 'Times New Roman', serif;
--font-label:   'DM Sans', 'Helvetica Neue', sans-serif;

--text-xs:      11px;
--text-sm:      13px;
--text-base:    15px;
--text-md:      18px;
--text-lg:      24px;
--text-xl:      32px;
--text-2xl:     48px;

--tracking-wide:   0.08em;
--tracking-wider:  0.15em;
--tracking-normal: 0;
```

### Vai trò font

- `font-display`: page heading, section title, masthead, editorial title.
- `font-body`: nav, route labels, CTA labels, table text, data, IDs, metrics, timestamps, tuple fragments, version strings, checksums, technical strings.
- `font-reading`: paragraph dài, support prose, summary prose, auth help copy, pricing policy prose, explanatory card copy.
- `font-label`: badge, eyebrow, micro-label, caption ngắn.

### Reading prose defaults

```css
--reading-prose-size:       16px;
--reading-prose-line-height: 1.72;
--reading-prose-max-measure: 64ch;
```

Reading prose phải dùng đúng:

```css
font-family: var(--font-reading);
font-size: var(--reading-prose-size);
line-height: var(--reading-prose-line-height);
max-width: var(--reading-prose-max-measure);
color: var(--color-text-primary);
```

### Editorial header pattern

```css
/* metadata line */
font-family: var(--font-body);
font-size: 11px;
letter-spacing: var(--tracking-wider);
text-transform: uppercase;
color: var(--color-text-secondary);

/* title line */
font-family: var(--font-display);
font-size: 20px;  /* hoặc scale lớn hơn theo screen */
font-weight: 700;
letter-spacing: var(--tracking-wide);
color: var(--color-text-primary);
```

Ví dụ metadata line:

```text
VOL. XII · EDITION 03
```

Account rail legibility:

- eyebrow của rail phải dùng `var(--color-text-primary)`, weight mạnh hơn, tracking rộng.
- nav label trong rail phải dùng `var(--color-text-primary)`, `font-weight: 600`, tracking nhẹ.
- active rail item dùng `3px solid var(--color-border-strong)`.

---

## 4. Spacing, Radius, Border Weight

### 4.1 Spacing

```css
--space-1:  4px;
--space-2:  8px;
--space-3:  12px;
--space-4:  16px;
--space-5:  20px;
--space-6:  24px;
--space-8:  32px;
--space-10: 40px;
--space-12: 48px;
--space-16: 64px;
```

### 4.2 Radius

```css
--radius-sm:   4px;
--radius-md:   8px;
--radius-lg:   12px;
--radius-xl:   16px;
--radius-full: 9999px;
```

### 4.3 Border hierarchy

```css
--frame-border-width:   3px;
--control-border-width: 2px;
--divider-width:        1px;
```

### 4.4 Cách dùng line-weight

- `3px`
  - screen shell
  - topbar bottom rule
  - sidebar / admin rail separator
  - card
  - panel
  - pricing block
  - table shell
  - dialog / sheet / popover panel
  - active nav item nếu muốn frame nhấn mạnh rõ
- `2px`
  - outline button
  - filled CTA shell
  - input
  - select trigger
  - pill
  - badge
  - compact control
- `1px`
  - row divider
  - internal separator
  - ruled-paper line

### 4.5 Layout constants

```css
--public-max-width:  1280px;
--account-max-width: 1360px;
--admin-max-width:   1440px;

--shell-padding-default: 24px;
--shell-padding-narrow:  16px;

--topbar-min-height: 72px;
--admin-sidebar-width: 272px;
--admin-compact-rail-width: 240px;
```

---

## 5. Surface Grammar

### 5.1 Khung nền

- Màn hình sáng phải đọc theo thứ tự:
  - nền chính `#EEE7DB`
  - panel/card `#E4DCCF`
  - control inset `#DCD2C3`
- Màn hình tối phải đọc theo thứ tự:
  - dark base `#1F1B18`
  - dark panel `#29231F`
  - dark inset `#312A25`

### 5.2 Khung viền

- Outer shell hoặc major frame nên dùng:
  - light mode: `3px solid #1B1815` hoặc `3px solid #2F2822`
  - dark mode: `3px solid #9A7E63` hoặc `3px solid #7B634A`
- Không dùng major frame màu quá nhạt.
- Không dùng border `1px` cho major shell.

### 5.3 Topbar / rail

- topbar phải có `min-height: 72px`
- topbar border-bottom: `3px solid var(--color-border-default)`
- sidebar / rail separator: `3px solid var(--color-border-default)`
- admin rail luôn dùng dark family, không dùng light surface trong khối rail

---

## 6. Component Contract

## 6.1 Filled CTA

```css
background:    var(--color-accent);        /* #1B1815 light, #EEE7DB dark */
color:         var(--color-text-inverse);
border:        var(--control-border-width) solid var(--color-accent);
border-radius: var(--radius-md);
min-height:    40px;
padding:       8px 20px;
font-family:   var(--font-body);
font-size:     13px;
font-weight:   500;
letter-spacing: var(--tracking-wide);
```

Rule:

- Mỗi screen chỉ có **1 filled CTA**.
- Không đặt filled CTA lặp lại theo từng card, từng row, từng metric.

## 6.2 Outline Button

```css
background:    transparent;
color:         var(--color-text-primary);
border:        var(--control-border-width) solid var(--color-border-default);
border-radius: var(--radius-sm);
min-height:    36px;
padding:       6px 12px;
font-family:   var(--font-body);
font-size:     12px;
```

Hover:

```css
border-color: var(--color-border-strong);
background:   var(--color-bg-surface);
```

## 6.3 Ghost Button

```css
background: transparent;
color:      var(--color-text-secondary);
border:     none;
padding:    6px 8px;
```

Nếu cần giữ geometry ổn định giữa một action row, có thể dùng:

```css
border: var(--control-border-width) solid transparent;
```

## 6.4 Pill

```css
background:    transparent;
color:         var(--color-text-secondary);
border:        var(--control-border-width) solid var(--color-border-default);
border-radius: var(--radius-full);
padding:       4px 10px;
font-family:   var(--font-label);
font-size:     11px;
```

## 6.5 Input / Textarea / Select Trigger

```css
background:    var(--color-bg-inset);
color:         var(--color-text-primary);
border:        var(--control-border-width) solid var(--color-border-default);
border-radius: var(--radius-md);
padding:       12px 16px;
font-family:   var(--font-body);
font-size:     15px;
```

Focus:

```css
background:   var(--color-bg-surface);
border-color: var(--color-border-strong);
box-shadow:   0 0 0 3px var(--color-border-strong);
```

Shared sizing:

- default field height: `40px`
- dense field height: `32px`
- textarea minimum height: `96px`

## 6.6 Card / Panel / Metric / Table Shell

```css
background:    var(--color-bg-surface);
border:        var(--frame-border-width) solid var(--color-border-default);
border-radius: var(--radius-lg);
padding:       24px;
```

Rules:

- Summary card, metric card, table shell, banner, toast boxed state đều mặc định dùng `3px` frame.
- Outer shell mạnh hoặc featured frame có thể nâng sang `var(--color-border-strong)`.
- Footer actions trong card mặc định là outline hoặc ghost.

## 6.7 Table

Table shell:

```css
border: var(--frame-border-width) solid var(--color-border-default);
```

Row divider:

```css
border-bottom: var(--divider-width) solid var(--color-border-subtle);
```

Rules:

- Per-row filled CTA là forbidden.
- Row hover dùng surface contrast, không dùng bright highlight fill.
- ID, timestamp, version, tuple evidence nên dùng mono styling.

## 6.8 Dialog / Sheet / Popover

Dialog / sheet / compact menu panel:

```css
background:    var(--color-bg-surface);
border:        var(--frame-border-width) solid var(--color-border-default);
border-radius: var(--radius-lg);
```

Backdrops:

```css
background: var(--color-bg-overlay);
```

Rules:

- dialog frame mặc định `3px`
- sheet frame mặc định `3px`
- popover / compact menu frame mặc định `3px`
- popover mới được phép dùng functional dropdown shadow:

```css
box-shadow: 0 4px 12px rgba(26,26,26,0.08);
```

- footer action row vẫn chỉ có **1 filled CTA**

## 6.9 Badge / Status Dot

Dot:

```css
width:  8px;
height: 8px;
border-radius: 9999px;
```

Badge:

```css
padding:       4px 8px;
border:        var(--control-border-width) solid var(--color-border-default);
border-radius: var(--radius-sm);
font-family:   var(--font-label);
font-size:     10px;
letter-spacing: var(--tracking-wider);
text-transform: uppercase;
```

Meaning rule:

- state phải luôn có text
- color không được là semantic channel duy nhất

## 6.10 Navigation active item

Light mode active item:

```css
background: #EEE7DB;
border:     3px solid #1B1815;
color:      #1B1815;
```

Dark mode active item:

```css
background: #29231F;
border:     3px solid #7B634A;
color:      #EEE7DB;
```

---

## 7. Texture and Motion

### 7.1 Ruled-paper treatment

```css
background-image:
  linear-gradient(to bottom, rgba(125, 101, 73, 0.14) 1px, transparent 1px);
background-size: 100% 28px;
```

Hoặc dùng separator nhẹ:

```css
background-image: repeating-linear-gradient(
  0deg,
  transparent,
  transparent 24px,
  var(--color-border-subtle) 24px,
  var(--color-border-subtle) 25px
);
```

### 7.2 Motion

```css
--transition-fast: 80ms ease;
--transition-base: 150ms ease;
--transition-slow: 250ms ease-out;
```

Rules:

- hover dùng `80ms`
- panel open/close dùng `150ms`
- page transition dùng `250ms ease-out`
- không dùng bounce, elastic, spring-heavy motion

---

## 8. Don’t List

```text
❌ Dùng lại palette cũ kiểu #F2EFE8 / #C8C3B8 / #A09A90
❌ Làm major frame mỏng 1px
❌ Giảm khung lớn về 2px sau khi authority đã chốt 3px
❌ Dùng border sáng, mờ cho shell chính
❌ Dùng #938674 làm light-mode strong border
❌ Dùng #5F5448 làm light-mode strong border sau khi authority đã chuyển sang ink-black
❌ Dùng white tinh #FFFFFF
❌ Dùng gray lạnh hoặc accent xanh SaaS
❌ Dùng gradient saturation cao
❌ Dùng shadow dày để tạo depth
❌ Dùng nhiều hơn 1 filled CTA trên cùng screen
❌ Dùng Inter, Roboto, system-ui làm font chính
❌ Dùng border radius > 16px ngoài pill
❌ Dùng icon màu như source chính của meaning
❌ Dùng màu state thuần chroma như #FF0000 / #00FF00 / #0000FF
```

---

## 9. Portable Copy-Forward Rules

Khi mang `The Press` sang project khác, phải giữ nguyên các điểm sau:

- Không reinterpret palette theo hướng neutral hơn.
- Không làm border nhẹ đi để “modern” hơn.
- Không đổi `font-body` sang sans.
- Không ép paragraph dài quay lại `font-body`.
- Không biến card thành soft-shadow container.
- Không nhân bản filled CTA theo từng module.
- Không để mỗi page tự chọn một border color khác nhau.

Nếu project mới không có đủ tất cả component, vẫn phải giữ 3 lớp authority:

1. token authority  
2. frame-weight hierarchy  
3. one-filled-CTA rule

---

## 10. Full CSS Template

```css
:root {
  --color-bg-base:        #EEE7DB;
  --color-bg-surface:     #E4DCCF;
  --color-bg-inset:       #DCD2C3;
  --color-bg-overlay:     #1F1B18;

  --color-border-subtle:  #6A5C4F;
  --color-border-default: #2F2822;
  --color-border-strong:  #1B1815;

  --color-text-primary:   #1B1815;
  --color-text-secondary: #5F5448;
  --color-text-muted:     #968A7C;
  --color-text-inverse:   #EEE7DB;

  --color-accent:         #1B1815;
  --color-accent-hover:   #322C26;

  --color-success:        #56684A;
  --color-warning:        #8C6B2A;
  --color-error:          #7C4035;
  --color-info:           #465B70;

  --font-display: 'Playfair Display', 'IM Fell English', Georgia, serif;
  --font-body:    'IBM Plex Mono', 'Courier New', monospace;
  --font-reading: Georgia, 'Times New Roman', serif;
  --font-label:   'DM Sans', 'Helvetica Neue', sans-serif;

  --reading-prose-size: 16px;
  --reading-prose-line-height: 1.72;
  --reading-prose-max-measure: 64ch;

  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;

  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;
  --radius-full: 9999px;

  --frame-border-width:   3px;
  --control-border-width: 2px;
  --divider-width:        1px;

  --shadow-focus:    0 0 0 3px var(--color-border-strong);
  --shadow-dropdown: 0 4px 12px rgba(26,26,26,0.08);

  --transition-fast: 80ms ease;
  --transition-base: 150ms ease;
  --transition-slow: 250ms ease-out;

  --tracking-wide:   0.08em;
  --tracking-wider:  0.15em;
  --tracking-normal: 0;

  --topbar-min-height: 72px;
  --admin-sidebar-width: 272px;
  --admin-compact-rail-width: 240px;
}

[data-theme='dark'] {
  --color-bg-base:        #1F1B18;
  --color-bg-surface:     #29231F;
  --color-bg-inset:       #312A25;

  --color-border-subtle:  #5B4837;
  --color-border-default: #7B634A;
  --color-border-strong:  #9A7E63;

  --color-text-primary:   #EEE7DB;
  --color-text-secondary: #938674;
  --color-text-muted:     #9A7E63;
  --color-text-inverse:   #1F1B18;

  --color-accent:         #EEE7DB;
}
```

---

*Design System `The Press` — editorial warm monochrome, exact token authority synced from SPEC*  
*Portable handoff edition — v1.3*
