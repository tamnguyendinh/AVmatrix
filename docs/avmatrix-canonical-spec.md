# AVmatrix Canonical Spec

Muc dich cua file nay la khoa ten goi canonical cho local fork truoc khi doi code.

File nay la nguon tham chieu cho moi batch rename tiep theo:
- docs phai bam theo file nay
- code phai doi theo file nay
- neu co xung dot giua wording cu va file nay, file nay la nguon dung

## Trang thai hien tai

- Code runtime hien tai van dang chay voi brand/command/storage cu:
  - `GitNexus`
  - `gitnexus`
  - `.gitnexus`
  - `~/.gitnexus`
  - `gitnexus://`
- Batch docs nay **khong** doi code runtime ngay.
- Batch docs nay chi khoa spec de cac phase code sau nay co diem bam ro rang.

## Canonical mapping

| Surface | Canonical moi | Ghi chu |
|---------|----------------|---------|
| Brand hiển thị | `AVmatrix` | Dung tren local UI, CLI help, onboarding, docs local |
| CLI command | `avmatrix` | Command canonical moi cho local usage |
| MCP command | `avmatrix mcp` | Duong canonical de noi Codex/Claude Code |
| MCP server name | `avmatrix` | Dung trong config MCP |
| Repo-local namespace | `.avmatrix` | Thay cho `.gitnexus` |
| User-global namespace | `~/.avmatrix` | Thay cho `~/.gitnexus` |
| Env var | `AVMATRIX_HOME` | Env runtime primary moi |
| Resource scheme | `avmatrix://` | Generator va docs moi chi duoc sinh scheme nay |

## Quyet dinh docs-first

- Bat dau tu phase nay, moi docs moi phai dung ten canonical o tren.
- Code chi duoc doi sau khi docs/spec da chot xong mapping nay.
- Khong duoc doi code theo kieu "lam truoc roi docs theo sau".

## Quyet dinh migration

### 1. Command / MCP

- `avmatrix` la command canonical moi.
- `gitnexus` co the duoc giu lam alias command/MCP trong giai doan dau de tranh gay script cu.
- Docs moi chi huong dan `avmatrix`.
- Neu co nhac den `gitnexus`, phai ghi ro:
  - day la alias cu / migration alias
  - khong phai duong canonical nua

### 2. Storage / env

- Khong chap nhan runtime fallback lau dai cho:
  - `.gitnexus`
  - `~/.gitnexus`
  - `GITNEXUS_HOME`
- Huong dung la:
  - migration mot lan
  - cutover sach
  - runtime sau do chi doc:
    - `.avmatrix`
    - `~/.avmatrix`
    - `AVMATRIX_HOME`

### 3. Resource scheme

- Generator moi chi sinh `avmatrix://...`
- Trong giai doan chuyen tiep, parser co the chap nhan `gitnexus://...` de doc tai lieu/du lieu cu neu can.
- Tuy nhien user-facing surface, docs, va output moi khong duoc tiep tuc sinh `gitnexus://...`

## Canonical examples

### CLI

```bash
avmatrix analyze .
avmatrix mcp
avmatrix query "auth flow"
avmatrix impact "Function:foo"
```

### Codex config

```toml
[mcp_servers.avmatrix]
command = "avmatrix"
args = ["mcp"]
```

### Claude Code config

```bash
claude mcp add avmatrix -- avmatrix mcp
```

### Storage

```text
<repo>/.avmatrix/
~/.avmatrix/registry.json
~/.avmatrix/runtime.json
```

### Resource scheme

```text
avmatrix://repo/<name>/context
avmatrix://repo/<name>/clusters
avmatrix://repo/<name>/processes
```

## Khong nam trong V1

Nhung thu sau **khong** nam trong V1, nen docs khong duoc mac dinh gia dinh da doi xong:
- ten thu muc package `gitnexus/`, `gitnexus-web/`, `gitnexus-shared`
- npm package name cong bo ra ngoai
- mass rename import noi bo
- rename ten repo tren disk hoac GitHub upstream

## File nay rang buoc nhung docs nao

Trong rollout docs-first, cac file sau phai duoc doi wording/copy theo file nay:
- `docs/local-usage.md`
- `README.md` o muc local setup / MCP setup / onboarding local
- docs setup Codex / Claude Code / Cursor
- docs migration va plan rename
- generator text se tao ra `AGENTS.md`, `CLAUDE.md`, `.claude/skills/gitnexus/*` trong rollout code dau tien

## Dieu kien de bat dau phase code

Chi duoc sang phase code rename neu:
- canonical mapping trong file nay da duoc giu nguyen
- audit inventory da co file rieng
- moi batch code co the chi ro no dang doi surface nao theo mapping nao trong file nay
