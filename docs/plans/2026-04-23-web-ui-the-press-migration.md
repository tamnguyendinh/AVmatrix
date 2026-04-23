# AVmatrix Web UI The Press Migration

Last updated: 2026-04-23
Status: proposed

## Purpose

Plan nay chot huong doi giao dien `avmatrix-web` sang phong cach trong [docs/UI-The-Press-design-system-editorial.md](/F:/AVmatrix-main/docs/UI-The-Press-design-system-editorial.md), nhung van giu mot "dark island" cho khu hiển thị graph/code chinh.

Muc tieu:

- toan bo shell, onboarding, header, side panels, settings, chat, forms, va cac khu dieu huong phai chuyen sang `The Press`
- khu hien thi chinh lien quan den graph + file/function linking van giu dark tone de bao toan kha nang doc va focus ky thuat
- khong lam mot ban "pha loang" cua `The Press`
- khong de dark mode cu tiep tuc lenh khenh song song voi ngôn ngu thiet ke moi

## Design Authority

Nguon authority duy nhat cho dot nay la:

- [UI-The-Press-design-system-editorial.md](/F:/AVmatrix-main/docs/UI-The-Press-design-system-editorial.md)

Plan nay phai coi spec do la nguon su that cho:

- palette
- typography
- frame weight
- spacing/radius
- one-filled-CTA rule
- topbar / rail / surface grammar

Khong duoc "dien giai mem di" cho hop voi giao dien hien tai.

## Current Reality In Code

Hien tai `avmatrix-web` van dang mang mot visual language dark-neon / devtool:

- token goc nam trong [index.css](/F:/AVmatrix-main/avmatrix-web/src/index.css)
- shell chinh nam trong [App.tsx](/F:/AVmatrix-main/avmatrix-web/src/App.tsx)
- header nam trong [Header.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/Header.tsx)
- landing/onboarding nam trong [RepoLanding.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/RepoLanding.tsx) va `DropZone`, `OnboardingGuide`, `RepoAnalyzer`
- left rail / filters nam trong [FileTreePanel.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/FileTreePanel.tsx)
- right rail / chat / processes nam trong [RightPanel.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/RightPanel.tsx), `ChatPanel`, `ProcessesPanel`
- khu graph va code inspector nam trong [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx) va [CodeReferencesPanel.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/CodeReferencesPanel.tsx)

Gap lon nhat so voi `The Press`:

1. palette hien tai la dark purple neon, trai nguoc warm monochrome
2. typography hien tai la `Outfit` + `JetBrains Mono`, chua co display / reading / label split theo spec
3. frame hierarchy hien tai mong, glow-heavy, gradient-heavy
4. cac CTA dang nhieu va khong theo rule "1 filled CTA / screen"
5. chat, onboarding, va panel shells chua co editorial rhythm
6. scroll area, dropdown, modal, popover, pills, badges dang theo SaaS/devtool grammar

## Hard Scope

Plan nay ap dung cho:

- `avmatrix-web/src/index.css`
- `avmatrix-web/src/App.tsx`
- `avmatrix-web/src/components/Header.tsx`
- `avmatrix-web/src/components/DropZone.tsx`
- `avmatrix-web/src/components/OnboardingGuide.tsx`
- `avmatrix-web/src/components/RepoLanding.tsx`
- `avmatrix-web/src/components/RepoAnalyzer.tsx`
- `avmatrix-web/src/components/FileTreePanel.tsx`
- `avmatrix-web/src/components/RightPanel.tsx`
- `avmatrix-web/src/components/ChatPanel.tsx`
- `avmatrix-web/src/components/ProcessesPanel.tsx`
- `avmatrix-web/src/components/StatusBar.tsx`
- `avmatrix-web/src/components/SettingsPanel*.tsx`
- `avmatrix-web/src/components/settings/*`
- `avmatrix-web/src/components/right-panel/*`
- cac dialog, sheet, popover, dropdown, toolbar, and form surfaces khac trong `avmatrix-web`

## Explicit Exception Scope

Theo yeu cau cua ban, phan sau KHONG bi doi sang light editorial shell:

- khu graph chinh trong [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx)
- code/reference workspace trong [CodeReferencesPanel.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/CodeReferencesPanel.tsx)
- code syntax dark styling ben trong code inspector va code snippets ky thuat

Phan ngoai le nay se duoc goi la:

- `primary graph workspace`

No van duoc phep nhan mot so token / border rhythm moi neu can, nhung:

- phai giu dark tone
- khong duoc bi ep sang light paper surfaces
- khong duoc lam giam contrast ky thuat cua graph/code view

## Guiding Principles

1. `The Press` la authority, khong phai moodboard tham khao.

2. Shell sang, workspace toi.
- Toan bo application chrome chuyen sang `The Press`
- chi `primary graph workspace` giu dark tone

3. Khong co giao dien "nua nay nua kia".
- shell/editorial surfaces phai nhat quan
- dark zone phai doc nhu mot workspace co chu y, khong phai mot diem sot theme cu

4. One filled CTA rule phai duoc ton trong.
- moi man hinh / panel level chi co 1 filled CTA chinh
- action con lai la outline hoac ghost

5. Depth phai den tu border rhythm va tone shift.
- khong dung decorative glow
- khong dua shadow neon vao shell moi

6. Typography phai co vai tro ro rang.
- display cho masthead/title
- mono cho tech strings, IDs, counts, labels ky thuat
- reading serif cho prose/help/chat support copy
- label sans cho badge/micro-label

7. Khi refactor UI lon, tao file/style helper moi song song roi chuyen tung vung.
- khong dap chắp vá len toan bo component hien tai trong mot luot

## Product Interpretation To Lock Before Coding

### Rule 1. `The Press` ap dung cho phan nao

Ap dung day du cho:

- onboarding
- repo selection
- shell layout
- top header
- left rail / filters
- right rail / tabs
- chat container
- process list / process detail shell
- settings panels
- dropdown / sheet / dialog / menu
- status bar

### Rule 2. Phan nao duoc giu dark

Chi giu dark cho:

- graph canvas
- code inspector / linked file-function view
- dark syntax blocks ky thuat

Neu co panel lai vua la shell vua la ky thuat, uu tien tach shell va workspace:

- shell theo `The Press`
- viewport ky thuat giu dark

### Rule 3. Khong reinterpret `The Press` thanh "beige SaaS"

Cam:

- border 1px cho major frame
- glow tim
- gradient accent manh
- sans lam font chinh cho toan bo shell
- CTA tim / cyan / emerald kiểu cu

### Rule 4. Dark workspace cung phai hop ngu phap voi shell moi

Du workspace chinh giu dark, van can:

- border hierarchy ro
- separator/rhythm ro
- khong neon hóa
- co the doi dark token sang warm dark family neu van giu contrast ky thuat

## Proposed Architecture Direction

## Phase 1. Token foundation and theme split

Muc tieu:

- tao mot nen token ro rang cho `The Press`
- dong thoi dinh nghia bo token rieng cho `primary graph workspace`

Can lam:

1. thay bo token trong [index.css](/F:/AVmatrix-main/avmatrix-web/src/index.css) bang:
- `The Press` light shell tokens
- `The Press` dark tokens khi can
- nhom token rieng cho graph workspace dark island

2. dinh nghia ro typography families:
- display
- body mono
- reading serif
- label sans

3. bo dan:
- purple accent
- glow shadows
- gradient shell cosmetics

4. them utility classes / component-level conventions cho:
- editorial header
- panel shell
- inset control
- mono metric line
- reading prose

Expected outcome:

- co mot he token trung tam de moi component khong tu tu che style rieng

## Phase 2. Shell layout and chrome conversion

Muc tieu:

- doi bo khung ung dung sang `The Press`

Can lam:

1. [App.tsx](/F:/AVmatrix-main/avmatrix-web/src/App.tsx)
- doi overall shell background sang editorial warm light family
- tach ro dark workspace trung tam khoi shell xung quanh

2. [Header.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/Header.tsx)
- doi topbar thanh editorial masthead/topbar
- bo gradient logo pill, button tim, glow
- dung 3px frame va typography authority

3. [StatusBar.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/StatusBar.tsx)
- doi thanh ruled / framed status strip
- mono metrics, subdued tone, khong neon status chip

4. dropdowns, repo picker, search shell
- doi thanh framed popover / compact sheet grammar cua `The Press`

Expected outcome:

- application chrome doc ro nhu mot editorial tool, khong con cam giac devtool neon

## Phase 3. Onboarding and landing rewrite in `The Press`

Muc tieu:

- onboarding va repo landing phai la noi the hien ro nhat design system moi

Can lam:

1. `DropZone`
2. `OnboardingGuide`
3. `RepoLanding`
4. `RepoAnalyzer`

Huong implementation:

- masthead + metadata line dung editorial header pattern
- prose huong dan dung reading serif
- stats / IDs / file counts dung mono
- chi giu 1 filled CTA chinh tren moi screen
- card/list/repo rows dung 3px frames
- bo ambient glows va accent blobs

Expected outcome:

- nguoi mo UI lan dau thay ro phong cach `The Press`, khong phai onboarding dark startup cu

## Phase 4. Left rail and navigation surfaces

Muc tieu:

- doi left panel thanh dark-ink editorial rail / paper utility panel

Can lam:

1. [FileTreePanel.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/FileTreePanel.tsx)
- file explorer shell
- filters shell
- search field
- chips / toggle rows / legend

Nguyen tac:

- rail shell theo `The Press`
- selected item doc ro bang border 3px + tone shift
- file tree text / metrics dung mono
- control rows dung 2px outlines
- khong con cyan/amber selection style cu o shell level

Expected outcome:

- left rail hoa thanh mot editorial utility rail thay vi IDE mini-sidebar

## Phase 5. Right rail, chat, and process surfaces

Muc tieu:

- doi right panel sang editorial support rail

Can lam:

1. [RightPanel.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/RightPanel.tsx)
- tab header
- close action
- framing

2. `ChatPanel`, `ChatTranscript`, `ChatComposer`, `ToolCallCard`
- chat shell theo `The Press`
- prose tro thanh reading-first
- technical fragments van mono
- citation/tool cards framed, khong glow

3. `ProcessesPanel`, `ProcessFlowModal`
- process list va process detail shells theo editorial frame rhythm
- process step tables/list dung mono + ruled separators

Rule quan trong:

- chat panel co the chua code block dark, nhung shell ngoai phai la `The Press`
- chi 1 filled CTA trong mot screen / modal / composer action zone

Expected outcome:

- chat va processes trong browser doc nhu mot annex / editorial support rail thong nhat

## Phase 6. Settings, dialogs, popovers, and forms

Muc tieu:

- tat ca surfaces phu tro phai dong bo grammar

Can lam:

1. `SettingsPanel*`
2. `settings/*`
3. `HelpPanel`
4. `ProcessFlowModal`
5. cac dialog / popover / dropdown khac

Can khoa:

- dialog frame 3px
- inputs/selects/textareas 2px
- focus ring theo spec
- badge/status co text, khong chi dung mau
- footer action row khong duoc co nhieu filled CTA

Expected outcome:

- khong con surface nao le theme trong khu form / settings / overlays

## Phase 7. Dark workspace harmonization

Muc tieu:

- giu dark tone cho workspace chinh, nhung bo chat neon cu

Can lam:

1. [GraphCanvas.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/GraphCanvas.tsx)
- giu dark canvas
- bo hoac giam language neon / purple-glow khong can thiet
- neu doi, chi doi sang warm-dark family hop voi shell moi

2. [CodeReferencesPanel.tsx](/F:/AVmatrix-main/avmatrix-web/src/components/CodeReferencesPanel.tsx)
- giu dark code workspace
- shell frame, header, badge rhythm can hai hoa hon voi `The Press`
- syntax highlight dark duoc giu

Expected outcome:

- workspace chinh van la noi tap trung ky thuat
- nhung tong the app khong con cam giac la hai theme va cham nhau

## Phase 8. Cleanup, consistency pass, and regression sweep

Muc tieu:

- bat het cac diem sot theme cu

Can lam:

1. tim toan bo class/token cu:
- `bg-accent`
- `shadow-glow`
- purple gradients
- neon borders
- Outfit / JetBrains-first usage o shell

2. ra soat:
- button hierarchy
- modal/footer action density
- badge semantics
- line weights
- border radius

3. visual pass tren:
- onboarding
- exploring shell
- repo switching
- search dropdown
- settings
- chat tab
- processes tab
- selected graph workflow

## File Strategy

Khong nen refactor UI nay theo kieu dap truc tiep toan bo component trong mot file lon.

Khi can tach phan style / presentation:

1. uu tien tao helper / class map / subcomponent moi song song
2. chuyen tung block UI sang structure moi
3. chi xoa fragment cu khi visual path moi da dung

Neu can, co the tao them:

- `avmatrix-web/src/styles/the-press.css`
- hoac `avmatrix-web/src/lib/ui-theme.ts`
- hoac cac subcomponent presentation moi cho header / rail / cards

nhung phai giu mot nguon authority ro rang cho token va utility semantics.

## Validation Plan

Bat buoc khi implementation:

1. typecheck
- `cd avmatrix-web && npx tsc -b --noEmit`

2. unit / component tests co lien quan
- `cd avmatrix-web && npm test`

3. Playwright phai duoc chay ky, khong duoc dung o muc smoke test
- `cd avmatrix-web && npm run test:e2e`

Playwright phai cover toi thieu:

- tung tab chinh trong UI
- tung button action quan trong
- tung dropdown / popover / sheet / modal co the mo tu shell moi
- luong onboarding
- luong repo landing
- luong repo switch
- luong settings
- luong chat
- luong processes
- luong graph selection + code inspector
- cac form co nhap du lieu
- cac input / textarea / select trigger can paste, type, submit, clear, cancel

Khong chap nhan:

- chi mo trang roi chup screenshot
- chi verify mot tab duy nhat
- chi click mot vai CTA chinh
- bo qua cac luong nhap du lieu va state transitions

4. visual manual pass tren it nhat cac path:
- onboarding
- repo landing
- exploring shell
- graph selection + code inspector
- open/close right panel
- chat send / stream
- settings dialog
- processes tab

5. trong qua trinh Playwright, phai chu dong nhap du lieu va di het flow:
- paste path repo
- submit analyze
- mo/doi tab
- mo/doi panel
- mo settings va doi gia tri
- gui message chat
- dung response
- dong/mo modal
- thu cac nut clear / cancel / close / retry neu co

## Success Criteria

Plan duoc coi la thanh cong khi:

1. nguoi dung nhin vao shell cua `avmatrix-web` thay ro `The Press`, khong con dark-neon shell cu
2. `primary graph workspace` van giu dark tone va contrast ky thuat
3. khong con major frame 1px o shell level
4. typography roles doc ro theo authority
5. moi screen / modal / panel action row ton trong one-filled-CTA rule
6. khong con purple glow / SaaS gradient nhu visual language chinh
7. app van dung duoc tren desktop + mobile widths ma khong vo layout

## Non-Goals

- khong doi logic chat/runtime
- khong doi graph engine
- khong doi MCP flow
- khong doi data model
- khong doi syntax highlighting strategy cua code workspace
- khong bien graph workspace sang light mode

## Explicit Bad Directions To Avoid

Khong duoc di theo cac huong sau:

- chi doi palette be mat ma giu nguyen component grammar cu
- giu gradient/glow cu roi them mau be len tren
- chuyen shell sang `The Press` nhung de 2-3 CTA filled xuat hien khap noi
- ep ca graph/code workspace sang light mode
- giu font sans lam font mac dinh cho tat ca prose
- chi doi mot vai page hero/landing ma bo qua app chrome va panels

## Expected Outcome

Sau khi xong plan nay:

- `avmatrix-web` se co mot visual language ro rang va nhat quan
- `The Press` se la shell/system authority that su
- graph/code workspace van giu dac tinh ky thuat va dark focus
- va tong the ung dung se doc nhu mot san pham co chu dich, khong phai mot tap hop panel dark-neon ghep lai
