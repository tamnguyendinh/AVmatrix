# AVmatrix Impact Contract Hardening

Last updated: 2026-04-23
Status: proposed

## Purpose

Plan nay chot huong sua `impact` theo kieu ben vung:

- khong sua tung diem drift roi de lai no ky thuat
- khong chon huong "co ve an toan" nhung de lai contract lech nhau
- khong de user/agent phai doan xem schema, CLI, runtime, va guidance cai nao moi la dung

Muc tieu cua plan:

- bien `impact` thanh mot contract duy nhat, duoc tai su dung o MCP, CLI, runtime, generated guidance, va tests
- loai bo silent fallback o cac tham so quan trong
- dam bao thong tin hien thi cho user/agent la chinh xac voi hanh vi runtime that
- giam kha nang tai phat mismatch sau cac lan chinh sua tiep theo

## Why The Previous Direction Is Not Enough

Huong "fix nhanh" truoc day co 3 diem de lai no ky thuat:

1. bo `target` khoi `required`, roi de runtime tu validate
- day chi lam schema bot sai hon, chu khong bien no thanh contract dung
- MCP client van khong co schema chinh xac de dua vao

2. doi docs/schema cho khop runtime ma khong dong bo nguon su that
- van de se quay lai khi code doi lan nua
- moi noi van co the noi mot kieu

3. sua hard-code guidance bang tay
- giai quyet trieu chung, khong giai quyet nguyen nhan
- lan sau doi tool list, guidance lai co the drift tiep

Neu di theo huong do, ta van co:

- contract phan manh
- validation lap lai o nhieu noi
- generated guidance de stale
- test khoa tung chieu mot, khong khoa duoc tinh dong bo toan bo

Plan moi phai sua tan goc theo huong:

- contract-first
- schema-first
- single source of truth
- generated guidance khong hard-code capability

## Problem Statement

Hien tai `impact` dang co 5 lop co the lech nhau:

1. MCP input schema
2. runtime parameter parsing / validation
3. CLI command surface
4. generated AI guidance
5. tests

Nhung mismatch da thay ro:

- schema noi `target` bat buoc, trong khi runtime ho tro `target_uid`
- `direction` khong duoc validate chat, typo co the bi suy ra thanh nghia khac
- `relationTypes` invalid bi nuot va fallback ve default
- `minConfidence` docs/schema va runtime khong giong nhau
- generated guidance dang noi toi `group_impact` va `avmatrix group impact` du capability do khong ton tai
- code comments/direct CLI examples van co cu phap cu

Van de cot loi:

- `impact` chua co mot contract dung nghia de moi be mat cung doc chung

## Scope

Plan nay chi xu ly:

- `avmatrix/src/mcp/tools.ts`
- `avmatrix/src/mcp/local/local-backend.ts`
- `avmatrix/src/cli/tool.ts`
- `avmatrix/src/cli/index.ts`
- `avmatrix/src/cli/ai-context.ts`
- cac helper/schema/type moi neu can de lam single source of truth cho `impact`
- tests unit/integration/e2e lien quan den `impact` contract

Plan nay khong xu ly:

- doi thuat toan blast radius
- doi risk scoring cua impact
- them feature cross-repo impact moi
- doi MCP transport hay startup path

## Operational Invariants

Plan nay phai giu nguyen nguyen ly hoat dong hien tai cua `impact`.

Nhung dieu KHONG duoc doi trong pham vi plan nay:

1. `impact` van la blast-radius analysis tren graph hien co, khong bien thanh mot loai analysis moi.

2. Traversal semantics cho request hop le van giu nguyen:
- `upstream` van nghia la "cai gi phu thuoc vao symbol nay"
- `downstream` van nghia la "symbol nay phu thuoc vao cai gi"

3. `target_uid` van la duong zero-ambiguity lookup, bo qua name-based resolver.

4. Default traversal relation set cho request hop le van giu nhu runtime hien tai.

5. Default `minConfidence` cho request hop le van giu theo runtime hien tai.

6. Risk calculation, grouping theo `byDepth`, process enrichment, module enrichment, va output shape co ban cua `impact` khong duoc doi.

Plan nay chi duoc sua:

- cach bieu dat contract
- cach validate input
- cach generated guidance tham chieu capability that
- test coverage de khoa cac semantics hien co

## Product Rules To Lock Before Coding

Day la phan can chot truoc khi sua code. Khong chot ro cho nay thi sua ky thuat de lai drift.

### Rule 1. Contract hop le cua `impact`

Moi request hop le phai co:

- `direction`
- va it nhat mot trong hai:
  - `target`
  - `target_uid`

`file_path` va `kind` chi la hint de disambiguate.

### Rule 2. Invalid input phai fail closed

Khong chap nhan:

- `direction` sai chinh ta
- `relationTypes` toan gia tri khong hop le
- `target` blank va khong co `target_uid`

Trong cac truong hop nay:

- phai tra loi co cau truc
- noi ro tham so nao sai
- neu co the, liet ke gia tri hop le

Khong duoc:

- silently fallback sang `downstream`
- silently fallback sang default relation set

### Rule 3. Defaults phai co mot nguon su that duy nhat

`maxDepth`, `includeTests`, `minConfidence`, relation set mac dinh:

- phai duoc khai bao mot lan
- moi be mat deu doc lai tu cung nguon do
- va phai mirror dung hanh vi runtime hien tai, khong duoc tu y doi sang semantics moi trong plan nay

Khong duoc de:

- tools.ts noi 1 dang
- backend noi 1 dang
- docs/guidance noi 1 dang

### Rule 4. Guidance chi duoc noi ve capability ton tai that

Neu tool/command khong ton tai:

- generated guidance khong duoc dat ra ten moi
- khong duoc hard-code "co ve hop ly"

No phai suy ra tu:

- tool registry that
- hoac mot capability matrix ro rang trong code

### Rule 5. Hop le hoa contract, khong tai dinh nghia tool

Plan nay duoc phep:

- lam ro contract hien co
- loai bo input sai
- chuan hoa defaults ve mot nguon

Plan nay khong duoc phep:

- doi default runtime hien tai
- doi relation set mac dinh cho request hop le
- doi y nghia `upstream` / `downstream`
- doi output shape core cua ket qua `impact`

## Guiding Principles

1. Contract truoc, implementation sau.

2. Schema phai du bieu dat nghia cua contract, khong chi du de "tam chap nhan duoc".

3. Runtime validation va CLI validation khong duoc moi noi tu viet mot ban rieng.

4. User/agent phai nhin thay dung hanh vi that, khong phai hanh vi "co ve dung".

5. Guidance generated phai la derived artifact, khong phai noi hard-code de drift.

6. Khi refactor mot file lon hoac mot file dang giu nhieu trach nhiem, uu tien tao file moi song song roi chuyen wiring dan, khong dap truc tiep len file cu theo kieu chắp vá.

## Refactor Safety Rule

Plan nay them mot quy dinh implementation de giam rui ro khi sua cac file trung tam cua `impact`.

Khi can refactor mot file lon:

1. Tao file moi song song de chua implementation moi.
2. Giu file cu on dinh de lam moc doi chieu trong qua trinh chuyen doi.
3. Chuyen wiring, import, va call sites dan sang file moi theo tung buoc nho.
4. Chi xoa file cu khi:
- file moi da dung duoc
- tests lien quan da pass
- runtime surface da duoc verify

Khong nen lam theo kieu:

- sua chắp vá lien tuc ngay tren file cu
- tron code cu va code moi vao cung mot khoi kho doc
- xoa file cu qua som khi chua co moc doi chieu

Ly do:

- neu refactor sai, van con moc ro rang de so sanh va quay lui
- de tách implementation moi khoi no ky thuat cu
- giam kha nang roi vao trang thai "da sua rat nhieu nhung khong ro sai tu dau"

## Proposed Direction

Huong ben vung nen la:

## Phase 1. Tao shared `impact` contract module

Muc tieu:

- tach mot noi dung duy nhat mo ta `impact` contract

Nen co mot module rieng, vi du:

- `src/mcp/contracts/impact.ts`
- hoac `src/shared/impact-contract.ts`

Module nay phai chua:

1. input shape chinh thuc
2. allowed enums:
- `direction`
- `relationTypes`
3. default values chinh thuc, copy tu runtime semantics hien tai
4. parser/validator helper
5. error messages chuan

Quan trong:

- day khong chi la type TypeScript
- no phai duoc dung that o runtime

### Vi sao day la huong ben vung

Neu khong co contract module chung:

- schema MCP se lai drift
- CLI se validate kieu khac
- generated guidance se noi khac
- tests se tiep tuc bat tung manh roi van lot regression

## Phase 2. Nang `ToolDefinition` len muc bieu dat duoc contract that

Muc tieu:

- MCP schema phai bieu dat duoc quy tac `target` hoac `target_uid`

Huong duoc khuyen nghi:

- mo rong `ToolDefinition` de chap nhan JSON Schema combinators toi thieu:
  - `oneOf`
  - `anyOf`
  - `enum`
  - neu can thi `minLength`

Khong khuyen nghi:

- giam schema xuong cho "de pass"
- roi de runtime validate bu

Vi sao:

- client/agent dung MCP dua rat nhieu vao input schema
- neu schema khong bieu dat dung nghia, user/agent se bi dan sai ngay tu dau

Ket qua mong muon:

- `impact` schema noi ro:
  - `direction` chi duoc `upstream` | `downstream`
  - phai co `target` hoac `target_uid`

## Phase 3. Runtime parsing va validation phai doc tu contract chung

Muc tieu:

- backend khong tu viet mot bo luat rieng nua

Can lam:

1. `impact()` backend goi parser/validator tu shared contract module
2. neu invalid:
- tra ve structured error thong nhat
- bao ro field nao sai
- neu la enum thi liet ke allowed values

3. `relationTypes`:
- khong con fallback im lang neu caller truyen danh sach toan invalid
- tra loi loi ro rang

4. `direction`:
- khong con "khac upstream thi xem nhu downstream"

### Quy tac quan trong

Runtime khong duoc co luat ngam khac voi schema.

Neu co luat ngam nao can giu, phai dua no vao contract chung.

## Phase 4. CLI phai tai su dung cung contract

Muc tieu:

- CLI va MCP khong duoc mo ta `impact` khac nhau

Can lam:

1. direct CLI `impactCommand` su dung cung validator/parser
2. help text / usage text phai doc theo semantics moi
3. comment example trong `tool.ts` phai sua theo cu phap that

Huong ben vung hon nua:

- neu hop ly, help fragments cho `impact` cung nen doc tu shared constants
- tranh lap lai allowed values va defaults trong nhieu file

## Phase 5. Chot defaults bang shared constants, khong de docs va runtime tu noi

Muc tieu:

- `minConfidence` va cac defaults khac khong con co kha nang lech nhau

Can co mot object kieu:

- `IMPACT_DEFAULTS`
- `IMPACT_ALLOWED_RELATION_TYPES`
- `IMPACT_ALLOWED_DIRECTIONS`

Moi noi dung:

- tools.ts
- local-backend.ts
- CLI help
- generated guidance
- tests

deu phai doc lai tu day.

### Ve `minConfidence`

Trong plan nay, `minConfidence` khong phai noi de mo product decision moi.

Plan nay phai:

- lay gia tri default dang dung that trong runtime hien tai
- dua no vao shared constants
- buoc schema/docs/CLI/test doc cung gia tri do

Neu sau nay muon doi default:

- do phai la mot de xuat rieng
- co benchmark va tac dong san pham ro rang
- khong tron vao dot contract hardening nay

## Phase 6. Guidance generated phai duoc derive tu capability that

Muc tieu:

- khong con hard-code ten tool/command khong ton tai

Can sua:

1. `ai-context` khong duoc viet text theo kieu tu do cho cross-repo impact neu capability do chua co
2. guidance cross-repo phai dua tren capability matrix that, vi du:
- tool nao ton tai
- command nao ton tai
- task nao duoc support

Huong ben vung:

- tao helper sinh ra guidance tu registry/capability flags
- thay vi ghep chuoi thu cong trong `ai-context.ts`

Ket qua:

- them/bot tool sau nay se it drift hon
- generated docs se phan anh dung mat bang capability hien tai

## Phase 7. Khoa bang contract tests, khong chi unit tests roi rac

Muc tieu:

- test phai bat mismatch toan bo be mat, khong chi bat tung implementation detail

Can co 4 lop test:

### A. Contract tests

Kiem tra:

- schema MCP co bieu dat dung contract khong
- defaults trong schema co dung shared constants khong
- enums trong schema co dung shared constants khong

### B. Runtime validation tests

Kiem tra:

- `target_uid`-only hop le
- thieu ca `target` va `target_uid` thi loi
- `direction` invalid thi loi
- `relationTypes` invalid-only thi loi

### C. Guidance tests

Kiem tra:

- generated AI context khong con `group_impact`
- khong con `avmatrix group impact`
- chi nhac tool/command that ton tai

### D. Surface alignment tests

Kiem tra:

- CLI, MCP, backend deu nhin thay cung defaults
- build `dist` xong thi `tools/list` van phan anh contract moi

## Decision Points

### 1. Co nen mo rong `ToolDefinition` khong

Khuyen nghi:

- Co

Ly do:

- day la phan trung tam de schema bieu dat dung contract
- neu khong mo rong, ta dang chap nhan MCP schema kem hon contract that
- day la su "an toan gia" ma ban dang muon tranh

### 2. Co nen tiep tuc hard-code generated guidance khong

Khuyen nghi:

- Khong

Ly do:

- hard-code la nguyen nhan goc cua drift o `group_impact`
- fix bang tay mot lan khong giai quyet duoc lan sau

### 3. Co nen doi luon default `minConfidence` trong dot nay khong

Khuyen nghi:

- Khong

Ly do:

- van de cua plan nay la drift, khong phai redesign semantics
- doi default trong cung dot nay se lam mo ranh gioi giua hardening va behavioural change

## Revised Implementation Order

Thu tu fix ben vung nen la:

1. Chot product rules cua `impact`
2. Tao shared contract module
3. Nang `ToolDefinition` de bieu dat dung contract
4. Cho MCP + runtime + CLI cung doc chung contract
5. Rut generated guidance ve capability-derived
6. Them contract tests
7. Build va probe `dist`

Khong nen lam theo thu tu:

1. sua docs
2. sua runtime
3. sau do moi nghi den schema

Vi thu tu do de de lai mismatch moi.

## Validation Commands

Toi thieu:

- `cd avmatrix && npx tsc --noEmit`
- `cd avmatrix && npx vitest run test/unit/tools.test.ts test/unit/calltool-dispatch.test.ts test/unit/tool-runtime-alignment.test.ts test/unit/ai-context.test.ts`

Neu co them test contract rieng:

- `cd avmatrix && npx vitest run test/unit/impact-contract.test.ts`

Neu co integration/e2e:

- `cd avmatrix && npx vitest run test/integration/local-backend-calltool.test.ts test/integration/cli-e2e.test.ts --testNamePattern impact`

Build/runtime verification:

- `cd avmatrix && npm run build`
- probe `node dist/cli/index.js mcp` voi `initialize` + `tools/list`

## Success Criteria

Plan duoc coi la thanh cong khi:

1. `impact` co mot contract chung duoc tai su dung that
2. schema MCP bieu dat dung `target` hoac `target_uid`
3. invalid input khong con bi nuot silently
4. defaults khong con lech giua docs/schema/runtime, va gia tri do trung voi runtime semantics hien tai
5. generated guidance khong con noi capability khong ton tai
6. tests khoa duoc surface alignment, khong chi logic rieng le

## Explicit Non-Durable Choices To Avoid

Khong chon cac huong sau:

- bo `target` khoi `required` roi coi nhu da xong
- sua rieng docs cho khop runtime nhung khong tao shared constants
- sua text `group_impact` bang tay nhung van giu hard-code guidance
- giu runtime fallback am tham de "de dung"
- de CLI va MCP moi noi validate mot kieu
- nhan dip nay doi luon semantics cua traversal hay defaults cua `impact`

Do la cac huong "co ve an toan" nhung thuc te de lai no ky thuat va thong tin sai cho user/agent.
