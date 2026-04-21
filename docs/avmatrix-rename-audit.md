# AVmatrix Rename Audit

File nay la inventory thuc te cho ke hoach doi ten tu `GitNexus` sang `AVmatrix`.

Muc dich:
- khong rename sot surface
- khong doi nham package/internal scope cua V1
- phan biet ro:
  - surface user-facing
  - setup/MCP/config
  - storage namespace
  - generator/auto-generated surfaces
  - test/docs fixtures

## Scope audit

Audit nay dua tren grep cac token sau trong toan repo:
- `GitNexus`
- `gitnexus`
- `.gitnexus`
- `gitnexus://`
- `GITNEXUS_HOME`

## Nhan dinh tong quan

- So diem lo brand cu rat lon, nhung phan lon roi vao 5 cum:
  1. docs va huong dan
  2. CLI setup / MCP setup
  3. storage namespace / registry / analyze / status
  4. AI-context generator va local skills auto-install
  5. tests / fixtures / e2e copy

- V1 **khong** nen co gang full rename import noi bo hang loat.
- V1 nen uu tien:
  - docs/spec canonical
  - CLI alias / MCP config
  - storage namespace migration
  - active local UI / setup wording

## Nhom 1: Docs user-facing va contributor-facing

### High-priority docs

- `README.md`
- `docs/local-usage.md`
- `ARCHITECTURE.md`
- `RUNBOOK.md`
- `TESTING.md`
- `CONTRIBUTING.md`
- `GUARDRAILS.md`
- `AGENTS.md`
- `CLAUDE.md`

### Audit notes

- `README.md` dang day dac local setup, MCP setup, command examples, va brand cu.
- `docs/local-usage.md` la noi nen doi som nhat vi no phuc vu local usage truc tiep.
- `AGENTS.md` / `CLAUDE.md` khong chi la docs thuong; chung co block duoc generator cap nhat tu dong.

## Nhom 2: CLI surface / MCP / setup

### Files chinh

- `gitnexus/src/cli/index.ts`
- `gitnexus/src/cli/setup.ts`
- `gitnexus/src/cli/mcp.ts`
- `gitnexus/src/cli/serve.ts`
- `gitnexus/src/cli/status.ts`
- `gitnexus/src/cli/list.ts`
- `gitnexus/src/cli/analyze.ts`
- `gitnexus/src/cli/index-repo.ts`
- `gitnexus/src/cli/tool.ts`
- `.mcp.json`

### Audit notes

- `setup.ts` hien tai la diem tap trung cho:
  - ten MCP server `gitnexus`
  - command `gitnexus`
  - config cho Codex / Claude / Cursor / OpenCode
  - install skills vao cac thu muc `.../gitnexus/`
- `index.ts` dang dong vai tro nguon chinh cho:
  - `program.name('gitnexus')`
  - copy help
  - option text

## Nhom 3: Storage namespace / registry / runtime config

### Files chinh

- `gitnexus/src/storage/repo-manager.ts`
- `gitnexus/src/storage/runtime-config.ts`
- `gitnexus/src/core/group/storage.ts`
- `gitnexus/src/core/lbug/*`
- `gitnexus/src/cli/analyze.ts`
- `gitnexus/src/cli/clean.ts`
- `gitnexus/src/cli/status.ts`
- `gitnexus/src/cli/index-repo.ts`
- `gitnexus/src/server/api.ts`

### Audit notes

- Day la cum phai doi theo mo hinh:
  - migration mot lan
  - cutover sach
  - khong fallback runtime lau dai
- Dac biet can audit:
  - registry path
  - runtime config path
  - repo-local storage path
  - group storage path
  - status / clean / index-repo copy

## Nhom 4: Resource scheme / MCP resources

### Files chinh

- `gitnexus/src/mcp/resources.ts`
- `gitnexus/src/mcp/tools.ts`
- `gitnexus/src/mcp/server.ts`
- `gitnexus/src/mcp/local/local-backend.ts`
- `gitnexus/src/server/mcp-http.ts`
- `gitnexus/src/server/session-bridge.ts`

### Audit notes

- Muc tieu V1:
  - output/generator moi chi sinh `avmatrix://`
  - docs va AI context khong con resource scheme cu

## Nhom 5: Auto-generated surfaces

### Files / generators canh bao

- `gitnexus/src/cli/ai-context.ts`
- `AGENTS.md`
- `CLAUDE.md`
- `.claude/skills/gitnexus/*`
- `.claude/skills/generated/*` neu co copy brand cu

### Audit notes

- Day la cum nguy hiem nhat neu bo sot.
- Neu doi brand tren UI/docs ma khong doi generator:
  - `analyze` se ghi lai brand cu
  - `AGENTS.md` / `CLAUDE.md` se bi "tai nhiem" brand cu
  - local skills se tiep tuc tao duoi namespace `gitnexus`

## Nhom 6: Web active path

### Files chinh

- `gitnexus-web/src/App.tsx`
- `gitnexus-web/src/components/Header.tsx`
- `gitnexus-web/src/components/OnboardingGuide.tsx`
- `gitnexus-web/src/components/HelpPanel.tsx`
- `gitnexus-web/src/components/RepoLanding.tsx`
- `gitnexus-web/src/components/AnalyzeOnboarding.tsx`
- `gitnexus-web/src/components/SettingsPanel*`
- `gitnexus-web/src/services/backend-client.ts`
- `gitnexus-web/src/hooks/useAppState.local-runtime.tsx`
- `gitnexus-web/src/components/StatusBar.tsx`

### Audit notes

- Day la cum can doi som sau docs-first vi no tac dong truc tiep den local user.
- Khong duoc doi thiet ke; chi doi brand/copy/command setup wording theo spec.

## Nhom 7: Test / e2e / fixtures

### Files chinh

- `gitnexus/test/unit/setup*.test.ts`
- `gitnexus/test/unit/cli-index-help.test.ts`
- `gitnexus/test/integration/setup-skills.test.ts`
- `gitnexus/test/integration/cli-e2e.test.ts`
- `gitnexus-web/e2e/*.spec.ts`
- `gitnexus-web/test/unit/*local-only*.test.tsx`
- `gitnexus-web/test/unit/ChatPanel.grounding-links.test.tsx`

### Audit notes

- V1 can behavioral tests moi cho:
  - `avmatrix --help`
  - `avmatrix mcp`
  - Codex/Claude config generation voi `avmatrix`
  - migration `.gitnexus -> .avmatrix`
  - active local UI chi hien `AVmatrix`

## Nhom 8: Package / publish / infra surfaces

### Files chinh

- `gitnexus/package.json`
- `gitnexus-web/package.json`
- `gitnexus-shared/package.json`
- `docker-compose.yaml`
- `.env.example`
- deploy / container image docs

### Audit notes

- Day la cum can phan biet ro:
  - cai nao la local user-facing V1
  - cai nao la upstream packaging / publish strategy
- V1 khong nen full rename package/publish surfaces neu chua co chien luoc phat hanh.

## Khuyen nghi batch code dau tien sau docs

Sau khi docs/spec da khoa, batch code dau tien nen gom:
1. CLI dual-bin:
   - them `avmatrix`
   - giu `gitnexus` alias
2. setup/MCP config generation:
   - sinh `avmatrix`
   - server name `avmatrix`
3. user-facing active local surfaces:
   - onboarding
   - local usage copy
   - help/setup copy
4. AI-context generator:
   - brand
   - command
   - resource scheme
   - skill namespace

## Khong lam trong batch dau

- full rename ten folder package
- mass rename import noi bo
- doi publish/package name ra npm
- doi remote GitHub/upstream naming

## Dieu kien audit duoc coi la xong

- co inventory tai lieu hoa cho 8 nhom tren
- da xac dinh ro auto-generated surfaces
- da xac dinh ro command/MCP/storage la 3 cum can doi som nhat sau docs-first
